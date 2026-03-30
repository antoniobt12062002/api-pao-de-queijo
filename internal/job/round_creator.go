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
	if err := j.notifySvc.SendRoundAnnounced(payerID); err != nil {
		slog.Error("DailyRoundCreator: error sending notification", "err", err)
	}
}
