package job

import (
	"log/slog"
	"time"

	"github.com/antoniobt12062002/pao-de-queijo/internal/domain"
)

// ParticipationWindowCloser fecha a rodada do dia após o período de participação.
type ParticipationWindowCloser struct {
	roundRepo    domain.RoundRepository
	notifySvc    domain.NotificationService
	scoreUpdater domain.ScoreUpdater
	badgeChecker domain.BadgeChecker
}

func NewParticipationWindowCloser(
	roundRepo domain.RoundRepository,
	notifySvc domain.NotificationService,
	scoreUpdater domain.ScoreUpdater,
	badgeChecker domain.BadgeChecker,
) *ParticipationWindowCloser {
	return &ParticipationWindowCloser{
		roundRepo:    roundRepo,
		notifySvc:    notifySvc,
		scoreUpdater: scoreUpdater,
		badgeChecker: badgeChecker,
	}
}

// Run fecha a rodada de hoje se estiver aberta.
func (j *ParticipationWindowCloser) Run() {
	today := time.Now().Format("2006-01-02")
	round, err := j.roundRepo.GetByDate(today)
	if err != nil {
		slog.Error("ParticipationWindowCloser: error fetching round", "err", err)
		return
	}
	if round == nil {
		slog.Warn("ParticipationWindowCloser: no round found for today")
		return
	}
	if round.Status != domain.RoundStatusOpen {
		slog.Info("ParticipationWindowCloser: round is not open, skipping", "status", round.Status)
		return
	}

	round.Status = domain.RoundStatusClosed
	if err := j.roundRepo.Update(round); err != nil {
		slog.Error("ParticipationWindowCloser: error closing round", "err", err)
		return
	}

	slog.Info("ParticipationWindowCloser: round closed", "id", round.ID)

	// Notifica o pagador (noop por enquanto)
	if err := j.notifySvc.SendRoundClosed(round.PayerID, round.ID); err != nil {
		slog.Error("ParticipationWindowCloser: error sending notification", "err", err)
	}

	// Atualiza score (noop por enquanto)
	if err := j.scoreUpdater.UpdateAfterRound(round.ID); err != nil {
		slog.Error("ParticipationWindowCloser: error updating score", "err", err)
	}

	if err := j.badgeChecker.CheckAfterRound(round.ID); err != nil {
		slog.Error("ParticipationWindowCloser: error checking badges", "err", err)
	}
}

// ReminderSender envia lembretes aos participantes antes do fechamento.
type ReminderSender struct {
	roundRepo domain.RoundRepository
	partRepo  domain.ParticipationRepository
	notifySvc domain.NotificationService
}

func NewReminderSender(
	roundRepo domain.RoundRepository,
	partRepo domain.ParticipationRepository,
	notifySvc domain.NotificationService,
) *ReminderSender {
	return &ReminderSender{
		roundRepo: roundRepo,
		partRepo:  partRepo,
		notifySvc: notifySvc,
	}
}

// Run envia lembrete se a rodada estiver aberta e tiver ao menos 1 participante.
func (j *ReminderSender) Run() {
	today := time.Now().Format("2006-01-02")
	round, err := j.roundRepo.GetByDate(today)
	if err != nil {
		slog.Error("ReminderSender: error fetching round", "err", err)
		return
	}
	if round == nil || round.Status != domain.RoundStatusOpen {
		slog.Info("ReminderSender: round not open, skipping reminder")
		return
	}

	parts, err := j.partRepo.GetByRound(round.ID)
	if err != nil {
		slog.Error("ReminderSender: error fetching participations", "err", err)
		return
	}
	if len(parts) == 0 {
		slog.Info("ReminderSender: no participants yet, skipping reminder", "round_id", round.ID)
		return
	}

	userIDs := make([]string, len(parts))
	for i, p := range parts {
		userIDs[i] = p.UserID
	}

	if err := j.notifySvc.SendReminder(userIDs, round.ID); err != nil {
		slog.Error("ReminderSender: error sending reminder", "err", err)
	}
}
