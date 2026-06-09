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
	notifySvc    domain.NotificationService
	closer       *ParticipationWindowCloser
	reminder     *ReminderSender
}

func NewDailyRoundCreator(
	roundRepo domain.RoundRepository,
	rotationRepo domain.RotationRepository,
	configRepo domain.ConfigRepository,
	notifySvc domain.NotificationService,
	closer *ParticipationWindowCloser,
	reminder *ReminderSender,
) *DailyRoundCreator {
	return &DailyRoundCreator{
		roundRepo:    roundRepo,
		rotationRepo: rotationRepo,
		configRepo:   configRepo,
		notifySvc:    notifySvc,
		closer:       closer,
		reminder:     reminder,
	}
}

// CreateForDate cria uma rodada para uma data específica (uso admin). payerID opcional;
// se vazio, usa o pagador atual da rotação.
func (j *DailyRoundCreator) CreateForDate(date, payerID string) error {
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
		payerID = rotation.CurrentPayerID()
	}

	// Calcula notifyAt combinando a data escolhida com o horário configurado.
	// Se o horário já passou (ou é data passada), usa time.Now() para que a
	// rodada fique disponível imediatamente.
	notifyAt := resolveNotifyAt(date, configMap["notify_at"])
	if notifyAt.Before(time.Now()) {
		notifyAt = time.Now()
	}
	closesAt := notifyAt.Add(time.Duration(windowMinutes) * time.Minute)

	round := &domain.Round{
		Date:     date,
		PayerID:  payerID,
		Status:   domain.RoundStatusPending,
		NotifyAt: notifyAt,
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
	today := time.Now().Format("2006-01-02")

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

	// Obtém pagador atual do rodízio
	rotation, err := j.rotationRepo.Get()
	if err != nil || rotation == nil || len(rotation.Members) == 0 {
		slog.Warn("DailyRoundCreator: no rotation configured, skipping round creation")
		return
	}

	payerID := rotation.CurrentPayerID()
	now := time.Now()
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
