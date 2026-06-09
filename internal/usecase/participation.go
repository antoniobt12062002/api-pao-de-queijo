package usecase

import "github.com/antoniobt12062002/pao-de-queijo/internal/domain"

type ParticipationUseCase struct {
	partRepo  domain.ParticipationRepository
	roundRepo domain.RoundRepository
	userRepo  domain.UserRepository
}

func NewParticipationUseCase(
	partRepo domain.ParticipationRepository,
	roundRepo domain.RoundRepository,
	userRepo domain.UserRepository,
) *ParticipationUseCase {
	return &ParticipationUseCase{partRepo: partRepo, roundRepo: roundRepo, userRepo: userRepo}
}

type ParticipationItem struct {
	UserID   string `json:"user_id"`
	Name     string `json:"name"`
	Quantity int    `json:"quantity"`
}

type ParticipationsResponse struct {
	Participations []*ParticipationItem `json:"participations"`
	TotalQuantity  int                  `json:"total_quantity"`
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

	users, err := uc.userRepo.FindAll()
	if err != nil {
		return nil, err
	}
	nameMap := make(map[string]string, len(users))
	for _, u := range users {
		nameMap[u.ID] = u.Name
	}

	total := 0
	items := make([]*ParticipationItem, len(parts))
	for i, p := range parts {
		total += p.Quantity
		items[i] = &ParticipationItem{
			UserID:   p.UserID,
			Name:     nameMap[p.UserID],
			Quantity: p.Quantity,
		}
	}
	return &ParticipationsResponse{
		Participations: items,
		TotalQuantity:  total,
	}, nil
}
