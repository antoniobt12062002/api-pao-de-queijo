package usecase

import "github.com/antoniobt12062002/pao-de-queijo/internal/domain"

type AbsenceUseCase struct {
	repo domain.AbsenceRepository
}

func NewAbsenceUseCase(repo domain.AbsenceRepository) *AbsenceUseCase {
	return &AbsenceUseCase{repo: repo}
}

func (uc *AbsenceUseCase) MarkAbsent(userID, date string) error {
	return uc.repo.Create(&domain.Absence{UserID: userID, Date: date})
}

func (uc *AbsenceUseCase) RemoveAbsence(userID, date string) error {
	return uc.repo.Delete(userID, date)
}

func (uc *AbsenceUseCase) ListAbsences(userID string) ([]*domain.Absence, error) {
	return uc.repo.GetByUser(userID)
}
