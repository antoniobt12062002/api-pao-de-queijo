package usecase

import (
	"log/slog"
	"strconv"
	"time"

	"github.com/antoniobt12062002/pao-de-queijo/internal/domain"
)

type RoundUseCase struct {
	roundRepo    domain.RoundRepository
	rotationRepo domain.RotationRepository
	configRepo   domain.ConfigRepository
	notifySvc    domain.NotificationService
	scoreUpdater domain.ScoreUpdater
}

func NewRoundUseCase(
	roundRepo domain.RoundRepository,
	rotationRepo domain.RotationRepository,
	configRepo domain.ConfigRepository,
	notifySvc domain.NotificationService,
	scoreUpdater domain.ScoreUpdater,
) *RoundUseCase {
	return &RoundUseCase{
		roundRepo:    roundRepo,
		rotationRepo: rotationRepo,
		configRepo:   configRepo,
		notifySvc:    notifySvc,
		scoreUpdater: scoreUpdater,
	}
}

// PaginatedRoundsResponse é a resposta paginada do GetAll.
type PaginatedRoundsResponse struct {
	Data  []*domain.Round `json:"data"`
	Total int64           `json:"total"`
	Page  int             `json:"page"`
	Limit int             `json:"limit"`
}

// TodayRoundResponse é a resposta do GetToday, incluindo is_payer.
type TodayRoundResponse struct {
	*domain.Round
	IsPayer bool `json:"is_payer"`
}

func (uc *RoundUseCase) GetAll(page, limit int) (*PaginatedRoundsResponse, error) {
	rounds, total, err := uc.roundRepo.GetAll(page, limit)
	if err != nil {
		return nil, err
	}
	return &PaginatedRoundsResponse{
		Data:  rounds,
		Total: total,
		Page:  page,
		Limit: limit,
	}, nil
}

func (uc *RoundUseCase) GetToday(callerID, date string) (*TodayRoundResponse, error) {
	if date == "" {
		date = time.Now().Format("2006-01-02")
	}
	round, err := uc.roundRepo.GetByDate(date)
	if err != nil {
		return nil, err
	}
	if round == nil {
		return nil, nil
	}
	return &TodayRoundResponse{
		Round:   round,
		IsPayer: round.PayerID == callerID,
	}, nil
}

func (uc *RoundUseCase) Confirm(roundID, callerID string) error {
	round, err := uc.roundRepo.GetByID(roundID)
	if err != nil {
		return err
	}
	if round == nil {
		return domain.ErrRoundNotFound
	}
	if round.PayerID != callerID {
		return domain.ErrRoundNotPayer
	}
	if round.Status != domain.RoundStatusPending {
		return domain.ErrRoundNotPending
	}

	// Se o pagador confirmou após o closes_at, reabre a janela de participação
	if time.Now().After(round.ClosesAt) {
		windowMinutes := uc.getWindowMinutes()
		round.ClosesAt = time.Now().Add(time.Duration(windowMinutes) * time.Minute)
	}

	round.Status = domain.RoundStatusOpen
	if err := uc.roundRepo.Update(round); err != nil {
		return err
	}

	// Notifica todos os membros exceto o próprio pagador (ele já foi notificado no round_announced)
	rotation, err := uc.rotationRepo.Get()
	if err == nil && rotation != nil && len(rotation.Members) > 0 {
		var userIDs []string
		for _, m := range rotation.Members {
			if m.UserID != callerID {
				userIDs = append(userIDs, m.UserID)
			}
		}
		if len(userIDs) > 0 {
			_ = uc.notifySvc.SendParticipationOpen(userIDs, roundID)
		}
	}
	return nil
}

func (uc *RoundUseCase) getWindowMinutes() int {
	configs, err := uc.configRepo.GetAll()
	if err != nil {
		return 30
	}
	for _, c := range configs {
		if c.Key == "round_window_minutes" {
			if v, err2 := strconv.Atoi(c.Value); err2 == nil && v > 0 {
				return v
			}
		}
	}
	return 30
}

func (uc *RoundUseCase) SetActualCost(roundID, callerID string, cost float64) error {
	round, err := uc.roundRepo.GetByID(roundID)
	if err != nil {
		return err
	}
	if round == nil {
		return domain.ErrRoundNotFound
	}
	if round.PayerID != callerID {
		return domain.ErrRoundNotPayer
	}
	round.ActualCost = &cost
	return uc.roundRepo.Update(round)
}

func (uc *RoundUseCase) ChangePayer(roundID, newPayerID string) error {
	round, err := uc.roundRepo.GetByID(roundID)
	if err != nil {
		return err
	}
	if round == nil {
		return domain.ErrRoundNotFound
	}
	if round.Status != domain.RoundStatusPending && round.Status != domain.RoundStatusOpen {
		return domain.ErrRoundNotPending
	}
	round.PayerID = newPayerID
	return uc.roundRepo.Update(round)
}

func (uc *RoundUseCase) Cancel(roundID, callerID string) error {
	round, err := uc.roundRepo.GetByID(roundID)
	if err != nil {
		return err
	}
	if round == nil {
		return domain.ErrRoundNotFound
	}
	if round.PayerID != callerID {
		return domain.ErrRoundNotPayer
	}
	if round.Status != domain.RoundStatusPending {
		return domain.ErrRoundNotPending
	}

	// Avança o rodízio para o próximo pagador
	if err := uc.rotationRepo.AdvancePosition(); err != nil {
		return err
	}

	rotation, err := uc.rotationRepo.Get()
	if err != nil {
		return err
	}
	if rotation == nil || len(rotation.Members) == 0 {
		return domain.ErrRotationEmpty
	}

	// Reatribui o pagador (mantém mesma linha, evita conflito UNIQUE(date))
	round.PayerID = rotation.CurrentPayerID()
	round.Status = domain.RoundStatusPending
	if err := uc.roundRepo.Update(round); err != nil {
		return err
	}

	// Notifica o novo pagador (noop por enquanto)
	_ = uc.notifySvc.SendRoundAnnounced(round.PayerID, round.ID)

	// Registra o skip no score do pagador original
	if err := uc.scoreUpdater.UpdateOnCancel(roundID); err != nil {
		slog.Error("RoundUseCase: error updating score on cancel", "err", err)
	}
	return nil
}
