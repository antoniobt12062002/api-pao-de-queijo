package job

import (
	"log/slog"
	"strconv"
	"time"

	"github.com/antoniobt12062002/pao-de-queijo/internal/domain"
)

// DailyRoundCreator cria a rodada do dia e agenda os jobs dinâmicos.
type DailyRoundCreator struct {
	roundRepo    domain.RoundRepository
	rotationRepo domain.RotationRepository
	configRepo   domain.ConfigRepository
	absenceRepo  domain.AbsenceRepository
	notifySvc    domain.NotificationService
	closer       *ParticipationWindowCloser
	reminder     *ReminderSender
}

func NewDailyRoundCreator(
	roundRepo domain.RoundRepository,
	rotationRepo domain.RotationRepository,
	configRepo domain.ConfigRepository,
	absenceRepo domain.AbsenceRepository,
	notifySvc domain.NotificationService,
	closer *ParticipationWindowCloser,
	reminder *ReminderSender,
) *DailyRoundCreator {
	return &DailyRoundCreator{
		roundRepo:    roundRepo,
		rotationRepo: rotationRepo,
		configRepo:   configRepo,
		absenceRepo:  absenceRepo,
		notifySvc:    notifySvc,
		closer:       closer,
		reminder:     reminder,
	}
}

// resolvePayerSkippingAbsent returns the first non-absent member starting from
// the current position in the rotation. Falls back to the current payer if all
// members are absent (edge case).
func (j *DailyRoundCreator) resolvePayerSkippingAbsent(rotation *domain.Rotation, date string) string {
	absentIDs, err := j.absenceRepo.GetAbsentUserIDsForDate(date)
	if err != nil || len(absentIDs) == 0 {
		return rotation.CurrentPayerID()
	}
	absentSet := make(map[string]struct{}, len(absentIDs))
	for _, id := range absentIDs {
		absentSet[id] = struct{}{}
	}
	members := rotation.Members
	n := len(members)
	for i := 0; i < n; i++ {
		idx := (rotation.CurrentPos + i) % n
		if _, absent := absentSet[members[idx].UserID]; !absent {
			return members[idx].UserID
		}
	}
	return rotation.CurrentPayerID()
}

// CreateForDate cria uma rodada para uma data específica (uso admin). payerID e notifyAt
// são opcionais; se vazios, usa o pagador atual da rotação e o horário configurado.
// notifyAt aceita "YYYY-MM-DDTHH:MM" (datetime-local do frontend).
func (j *DailyRoundCreator) CreateForDate(date, payerID, notifyAt string) error {
	existing, err := j.roundRepo.GetByDate(date)
	if err != nil {
		return err
	}
	if existing != nil {
		return domain.ErrRoundAlreadyExists
	}

	configs, err := j.configRepo.GetAll()
	if err != nil {
		return err
	}
	configMap := make(map[string]string)
	for _, c := range configs {
		configMap[c.Key] = c.Value
	}
	windowMinutes := 30
	if v, ok := configMap["round_window_minutes"]; ok {
		if n, err2 := strconv.Atoi(v); err2 == nil {
			windowMinutes = n
		}
	}

	if payerID == "" {
		rotation, err := j.rotationRepo.Get()
		if err != nil || rotation == nil || len(rotation.Members) == 0 {
			return domain.ErrRotationEmpty
		}
		payerID = j.resolvePayerSkippingAbsent(rotation, date)
	}

	// Se o frontend enviou um datetime-local ("YYYY-MM-DDTHH:MM"), usa diretamente;
	// caso contrário, combina a data com o horário configurado.
	var resolvedNotifyAt time.Time
	if notifyAt != "" {
		loc := time.Now().Location()
		if t, err2 := time.ParseInLocation("2006-01-02T15:04", notifyAt, loc); err2 == nil {
			resolvedNotifyAt = t
		}
	}
	if resolvedNotifyAt.IsZero() {
		resolvedNotifyAt = resolveNotifyAt(date, configMap["notify_at"])
	}
	if resolvedNotifyAt.Before(time.Now()) {
		resolvedNotifyAt = time.Now()
	}
	closesAt := resolvedNotifyAt.Add(time.Duration(windowMinutes) * time.Minute)

	round := &domain.Round{
		Date:     date,
		PayerID:  payerID,
		Status:   domain.RoundStatusPending,
		NotifyAt: resolvedNotifyAt,
		ClosesAt: closesAt,
	}
	if err := j.roundRepo.Create(round); err != nil {
		return err
	}

	slog.Info("DailyRoundCreator: round created for specific date by admin", "date", date, "id", round.ID)
	_ = j.notifySvc.SendRoundAnnounced(payerID, round.ID)
	return nil
}

// Run executa o job de criação diária de rodada. É idempotente.
func (j *DailyRoundCreator) Run() {
	now := time.Now()
	if wd := now.Weekday(); wd == time.Saturday || wd == time.Sunday {
		slog.Info("DailyRoundCreator: skipping weekend", "weekday", wd)
		return
	}
	today := now.Format("2006-01-02")

	// Idempotência: verifica se já existe rodada para hoje
	existing, err := j.roundRepo.GetByDate(today)
	if err != nil {
		slog.Error("DailyRoundCreator: error checking existing round", "err", err)
		return
	}
	if existing != nil {
		slog.Info("DailyRoundCreator: round already exists for today, skipping", "date", today)
		return
	}

	// Lê configuração
	configs, err := j.configRepo.GetAll()
	if err != nil {
		slog.Error("DailyRoundCreator: error reading config", "err", err)
		return
	}
	configMap := make(map[string]string)
	for _, c := range configs {
		configMap[c.Key] = c.Value
	}

	windowMinutes := 30
	if v, ok := configMap["round_window_minutes"]; ok {
		if n, err := strconv.Atoi(v); err == nil {
			windowMinutes = n
		}
	}

	// Obtém pagador atual do rodízio, pulando ausentes
	rotation, err := j.rotationRepo.Get()
	if err != nil || rotation == nil || len(rotation.Members) == 0 {
		slog.Warn("DailyRoundCreator: no rotation configured, skipping round creation")
		return
	}

	payerID := j.resolvePayerSkippingAbsent(rotation, today)
	closesAt := now.Add(time.Duration(windowMinutes) * time.Minute)
	reminderAt := closesAt.Add(-5 * time.Minute)

	// Cria a rodada
	round := &domain.Round{
		Date:     today,
		PayerID:  payerID,
		Status:   domain.RoundStatusPending,
		NotifyAt: now,
		ClosesAt: closesAt,
	}
	if err := j.roundRepo.Create(round); err != nil {
		slog.Error("DailyRoundCreator: error creating round", "err", err)
		return
	}

	slog.Info("DailyRoundCreator: round created", "id", round.ID, "payer", payerID)

	// Agenda ParticipationWindowCloser dinamicamente (one-shot)
	durationToClose := time.Until(closesAt)
	if durationToClose > 0 {
		time.AfterFunc(durationToClose, j.closer.Run)
		slog.Info("DailyRoundCreator: ParticipationWindowCloser scheduled", "at", closesAt)
	}

	// Agenda ReminderSender dinamicamente (one-shot, 5 min antes do fechamento)
	durationToRemind := time.Until(reminderAt)
	if durationToRemind > 0 {
		time.AfterFunc(durationToRemind, j.reminder.Run)
		slog.Info("DailyRoundCreator: ReminderSender scheduled", "at", reminderAt)
	}

	// Notifica o pagador (noop por enquanto)
	if err := j.notifySvc.SendRoundAnnounced(payerID, round.ID); err != nil {
		slog.Error("DailyRoundCreator: error sending notification", "err", err)
	}
}

// resolveNotifyAt combina a data (YYYY-MM-DD) com o horário de notificação
// configurado (HH:MM) para produzir um timestamp no fuso local.
func resolveNotifyAt(date, notifyAtCfg string) time.Time {
	hour, minute := 8, 0
	if len(notifyAtCfg) == 5 {
		h, _ := strconv.Atoi(notifyAtCfg[:2])
		m, _ := strconv.Atoi(notifyAtCfg[3:])
		hour, minute = h, m
	}
	loc := time.Now().Location()
	t, err := time.ParseInLocation("2006-01-02", date, loc)
	if err != nil {
		return time.Now()
	}
	return time.Date(t.Year(), t.Month(), t.Day(), hour, minute, 0, 0, loc)
}
