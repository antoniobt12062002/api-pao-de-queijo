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
}

func NewParticipationWindowCloser(
	roundRepo domain.RoundRepository,
	notifySvc domain.NotificationService,
	scoreUpdater domain.ScoreUpdater,
) *ParticipationWindowCloser {
	return &ParticipationWindowCloser{
		roundRepo:    roundRepo,
		notifySvc:    notifySvc,
		scoreUpdater: scoreUpdater,
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
}

// ReminderSender envia lembretes aos participantes antes do fechamento.
type ReminderSender struct {
	roundRepo domain.RoundRepository
	notifySvc domain.NotificationService
}

func NewReminderSender(
	roundRepo domain.RoundRepository,
	notifySvc domain.NotificationService,
) *ReminderSender {
	return &ReminderSender{
		roundRepo: roundRepo,
		notifySvc: notifySvc,
	}
}

// Run envia lembrete se a rodada estiver aberta e tiver ao menos 1 participante.
// Como feature/participation ainda não foi implementada, este job é um stub.
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
	// TODO: buscar participantes quando feature/participation for implementada
	slog.Info("ReminderSender: participation data not available yet, skipping reminder", "round_id", round.ID)
}
