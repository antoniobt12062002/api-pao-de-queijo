package usecase

import "github.com/antoniobt12062002/pao-de-queijo/internal/domain"

type RotationUseCase struct {
	repo     domain.RotationRepository
	userRepo domain.UserRepository
}

func NewRotationUseCase(repo domain.RotationRepository, userRepo domain.UserRepository) *RotationUseCase {
	return &RotationUseCase{repo: repo, userRepo: userRepo}
}

// RotationResponse is the response type returned by GetCurrent.
type RotationResponse struct {
	CurrentPos     int                      `json:"current_pos"`
	CurrentPayerID string                   `json:"current_payer_id"`
	Members        []*domain.RotationMember `json:"members"`
}

func (uc *RotationUseCase) GetCurrent() (*RotationResponse, error) {
	r, err := uc.repo.Get()
	if err != nil {
		return nil, err
	}
	if r == nil {
		return &RotationResponse{Members: []*domain.RotationMember{}}, nil
	}
	return &RotationResponse{
		CurrentPos:     r.CurrentPos,
		CurrentPayerID: r.CurrentPayerID(),
		Members:        r.Members,
	}, nil
}

func (uc *RotationUseCase) UpdateOrder(userIDs []string) error {
	if len(userIDs) == 0 {
		return domain.ErrRotationEmptyOrder
	}

	// Check for duplicates
	seen := make(map[string]bool, len(userIDs))
	for _, id := range userIDs {
		if seen[id] {
			return domain.ErrRotationDuplicateUser
		}
		seen[id] = true
	}

	// Validate all users exist
	allUsers, err := uc.userRepo.FindAll()
	if err != nil {
		return err
	}
	validIDs := make(map[string]bool, len(allUsers))
	for _, u := range allUsers {
		validIDs[u.ID] = true
	}
	for _, id := range userIDs {
		if !validIDs[id] {
			return domain.ErrRotationUnknownUser
		}
	}

	return uc.repo.SetOrder(userIDs)
}

func (uc *RotationUseCase) Skip() error {
	r, err := uc.repo.Get()
	if err != nil {
		return err
	}
	if r == nil || len(r.Members) == 0 {
		return domain.ErrRotationNotInitialized
	}
	return uc.repo.AdvancePosition()
}
