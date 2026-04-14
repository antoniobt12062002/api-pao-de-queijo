package postgres

import (
	"errors"
	"time"

	"github.com/antoniobt12062002/pao-de-queijo/internal/domain"
	"gorm.io/gorm"
)

type participationModel struct {
	ID          string    `gorm:"column:id;primaryKey;type:uuid;default:gen_random_uuid()"`
	RoundID     string    `gorm:"column:round_id;type:uuid"`
	UserID      string    `gorm:"column:user_id;type:uuid"`
	Quantity    int       `gorm:"column:quantity"`
	ConfirmedAt time.Time `gorm:"column:confirmed_at;autoCreateTime"`
}

func (participationModel) TableName() string { return "participations" }

type ParticipationRepository struct {
	db *gorm.DB
}

func NewParticipationRepository(db *gorm.DB) *ParticipationRepository {
	return &ParticipationRepository{db: db}
}

func (r *ParticipationRepository) Create(p *domain.Participation) error {
	m := &participationModel{
		RoundID:  p.RoundID,
		UserID:   p.UserID,
		Quantity: p.Quantity,
	}
	if err := r.db.Create(m).Error; err != nil {
		return err
	}
	p.ID = m.ID
	return nil
}

func (r *ParticipationRepository) GetByRoundAndUser(roundID, userID string) (*domain.Participation, error) {
	var m participationModel
	result := r.db.Where("round_id = ? AND user_id = ?", roundID, userID).First(&m)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if result.Error != nil {
		return nil, result.Error
	}
	return toDomainParticipation(&m), nil
}

func (r *ParticipationRepository) GetByRound(roundID string) ([]*domain.Participation, error) {
	var models []participationModel
	if err := r.db.Where("round_id = ?", roundID).Find(&models).Error; err != nil {
		return nil, err
	}
	result := make([]*domain.Participation, len(models))
	for i, m := range models {
		result[i] = toDomainParticipation(&m)
	}
	return result, nil
}

func (r *ParticipationRepository) Delete(roundID, userID string) error {
	result := r.db.Where("round_id = ? AND user_id = ?", roundID, userID).Delete(&participationModel{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return domain.ErrParticipationNotFound
	}
	return nil
}

func toDomainParticipation(m *participationModel) *domain.Participation {
	return &domain.Participation{
		ID:          m.ID,
		RoundID:     m.RoundID,
		UserID:      m.UserID,
		Quantity:    m.Quantity,
		ConfirmedAt: m.ConfirmedAt,
	}
}

// Verify interface compliance at compile time
var _ domain.ParticipationRepository = (*ParticipationRepository)(nil)
