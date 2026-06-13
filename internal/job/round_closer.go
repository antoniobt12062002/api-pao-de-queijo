package job

import (
	"log/slog"
	"strconv"
	"time"

	"github.com/antoniobt12062002/pao-de-queijo/internal/domain"
)

// ParticipationWindowCloser fecha a rodada do dia após o período de participação.
type ParticipationWindowCloser struct {
	roundRepo    domain.RoundRepository
	partRepo     domain.ParticipationRepository
	configRepo   domain.ConfigRepository
	rotationRepo domain.RotationRepository
	notifySvc    domain.NotificationService
	scoreUpdater domain.ScoreUpdater
	badgeChecker domain.BadgeChecker
}

func NewParticipationWindowCloser(
	roundRepo domain.RoundRepository,
	partRepo domain.ParticipationRepository,
	configRepo domain.ConfigRepository,
	rotationRepo domain.RotationRepository,
	notifySvc domain.NotificationService,
	scoreUpdater domain.ScoreUpdater,
	badgeChecker domain.BadgeChecker,
) *ParticipationWindowCloser {
	return &ParticipationWindowCloser{
		roundRepo:    roundRepo,
		partRepo:     partRepo,
		configRepo:   configRepo,
		rotationRepo: rotationRepo,
		notifySvc:    notifySvc,
		scoreUpdater: scoreUpdater,
		badgeChecker: badgeChecker,
	}
}

// Run fecha a rodada de hoje se estiver aberta e o closes_at tiver passado.
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
	j.closeRound(round)
}

// CheckAndClose é chamado periodicamente para fechar rodadas abertas que já
// ultrapassaram closes_at (cobre confirmações tardias pelo pagador).
func (j *ParticipationWindowCloser) CheckAndClose() {
	today := time.Now().Format("2006-01-02")
	round, err := j.roundRepo.GetByDate(today)
	if err != nil || round == nil {
		return
	}
	if round.Status != domain.RoundStatusOpen {
		return
	}
	if time.Now().Before(round.ClosesAt) {
		return
	}
	slog.Info("ParticipationWindowCloser: closing overdue open round", "id", round.ID)
	j.closeRound(round)
}

// CloseRoundByID fecha uma rodada específica pelo ID (uso admin).
func (j *ParticipationWindowCloser) CloseRoundByID(roundID string) error {
	round, err := j.roundRepo.GetByID(roundID)
	if err != nil {
		return err
	}
	if round == nil {
		return domain.ErrRoundNotFound
	}
	if round.Status != domain.RoundStatusOpen {
		return domain.ErrRoundNotOpen
	}
	j.closeRound(round)
	return nil
}

// closeRound contém a lógica comum de fechamento: calcula custo, salva, avança
// rotação, notifica e atualiza score/badges.
func (j *ParticipationWindowCloser) closeRound(round *domain.Round) {
	j.setAutoCost(round)

	round.Status = domain.RoundStatusClosed
	if err := j.roundRepo.Update(round); err != nil {
		slog.Error("ParticipationWindowCloser: error closing round", "err", err)
		return
	}
	slog.Info("ParticipationWindowCloser: round closed", "id", round.ID)

	// Avança rotação para o próximo pagador
	if err := j.rotationRepo.AdvancePosition(); err != nil {
		slog.Error("ParticipationWindowCloser: error advancing rotation", "err", err)
	} else {
		slog.Info("ParticipationWindowCloser: rotation advanced after round close")
	}

	if err := j.notifySvc.SendRoundClosed(round.PayerID, round.ID); err != nil {
		slog.Error("ParticipationWindowCloser: error sending notification", "err", err)
	}

	if err := j.scoreUpdater.UpdateAfterRound(round.ID); err != nil {
		slog.Error("ParticipationWindowCloser: error updating score", "err", err)
	}

	if err := j.badgeChecker.CheckAfterRound(round.ID); err != nil {
		slog.Error("ParticipationWindowCloser: error checking badges", "err", err)
	}
}

// setAutoCost calcula e define o custo da rodada a partir das participações e
// do preço unitário configurado. Não sobrescreve se já definido manualmente.
func (j *ParticipationWindowCloser) setAutoCost(round *domain.Round) {
	if round.ActualCost != nil {
		return // já definido manualmente
	}

	parts, err := j.partRepo.GetByRound(round.ID)
	if err != nil || len(parts) == 0 {
		return
	}
	totalQty := 0
	for _, p := range parts {
		totalQty += p.Quantity
	}

	configs, err := j.configRepo.GetAll()
	if err != nil {
		return
	}
	var pricePerUnit float64
	for _, c := range configs {
		if c.Key == "price_per_unit" {
			if v, err2 := strconv.ParseFloat(c.Value, 64); err2 == nil {
				pricePerUnit = v
			}
		}
	}
	if pricePerUnit <= 0 {
		return
	}

	cost := float64(totalQty) * pricePerUnit
	round.ActualCost = &cost
	slog.Info("ParticipationWindowCloser: auto-cost calculated", "totalQty", totalQty, "pricePerUnit", pricePerUnit, "cost", cost)
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
