package usecase

import (
	"time"

	"github.com/antoniobt12062002/pao-de-queijo/internal/domain"
)

type RoundUseCase struct {
	roundRepo    domain.RoundRepository
	rotationRepo domain.RotationRepository
	notifySvc    domain.NotificationService
}

func NewRoundUseCase(
	roundRepo domain.RoundRepository,
	rotationRepo domain.RotationRepository,
	notifySvc domain.NotificationService,
) *RoundUseCase {
	return &RoundUseCase{
		roundRepo:    roundRepo,
		rotationRepo: rotationRepo,
		notifySvc:    notifySvc,
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

func (uc *RoundUseCase) GetToday(callerID string) (*TodayRoundResponse, error) {
	today := time.Now().Format("2006-01-02")
	round, err := uc.roundRepo.GetByDate(today)
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
	round.Status = domain.RoundStatusOpen
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
	_ = uc.notifySvc.SendRoundAnnounced(round.PayerID)
	return nil
}
