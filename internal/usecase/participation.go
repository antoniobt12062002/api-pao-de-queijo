package usecase

import "github.com/antoniobt12062002/pao-de-queijo/internal/domain"

type ParticipationUseCase struct {
	partRepo  domain.ParticipationRepository
	roundRepo domain.RoundRepository
}

func NewParticipationUseCase(
	partRepo domain.ParticipationRepository,
	roundRepo domain.RoundRepository,
) *ParticipationUseCase {
	return &ParticipationUseCase{partRepo: partRepo, roundRepo: roundRepo}
}

type ParticipationsResponse struct {
	Data          []*domain.Participation `json:"data"`
	TotalQuantity int                     `json:"total_quantity"`
}

func (uc *ParticipationUseCase) Participate(roundID, userID string, quantity int) error {
	round, err := uc.roundRepo.GetByID(roundID)
	if err != nil {
		return err
	}
	if round == nil {
		return domain.ErrRoundNotFound
	}
	if round.Status != domain.RoundStatusOpen {
		return domain.ErrRoundNotOpen
	}
	return uc.partRepo.Create(&domain.Participation{
		RoundID:  roundID,
		UserID:   userID,
		Quantity: quantity,
	})
}

func (uc *ParticipationUseCase) Withdraw(roundID, userID string) error {
	round, err := uc.roundRepo.GetByID(roundID)
	if err != nil {
		return err
	}
	if round == nil {
		return domain.ErrRoundNotFound
	}
	if round.Status != domain.RoundStatusOpen {
		return domain.ErrRoundNotOpen
	}
	existing, err := uc.partRepo.GetByRoundAndUser(roundID, userID)
	if err != nil {
		return err
	}
	if existing == nil {
		return domain.ErrParticipationNotFound
	}
	return uc.partRepo.Delete(roundID, userID)
}

func (uc *ParticipationUseCase) GetParticipations(roundID string) (*ParticipationsResponse, error) {
	parts, err := uc.partRepo.GetByRound(roundID)
	if err != nil {
		return nil, err
	}
	total := 0
	for _, p := range parts {
		total += p.Quantity
	}
	return &ParticipationsResponse{
		Data:          parts,
		TotalQuantity: total,
	}, nil
}
