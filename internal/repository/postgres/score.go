package postgres

import (
	"errors"
	"time"

	"github.com/antoniobt12062002/pao-de-queijo/internal/domain"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type scoreModel struct {
	UserID            string    `gorm:"column:user_id;primaryKey;type:uuid"`
	TimesPaid         int       `gorm:"column:times_paid"`
	TimesParticipated int       `gorm:"column:times_participated"`
	TotalAmountSpent  float64   `gorm:"column:total_amount_spent"`
	SkipCount         int       `gorm:"column:skip_count"`
	CurrentStreak     int       `gorm:"column:current_streak"`
	Score             float64   `gorm:"column:score"`
	UpdatedAt         time.Time `gorm:"column:updated_at;autoUpdateTime"`
}

func (scoreModel) TableName() string { return "scores" }

type ScoreRepository struct {
	db *gorm.DB
}

func NewScoreRepository(db *gorm.DB) *ScoreRepository {
	return &ScoreRepository{db: db}
}

func (r *ScoreRepository) Upsert(s *domain.Score) error {
	m := &scoreModel{
		UserID:            s.UserID,
		TimesPaid:         s.TimesPaid,
		TimesParticipated: s.TimesParticipated,
		TotalAmountSpent:  s.TotalAmountSpent,
		SkipCount:         s.SkipCount,
		CurrentStreak:     s.CurrentStreak,
		Score:             s.Score,
	}
	return r.db.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "user_id"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"times_paid", "times_participated", "total_amount_spent",
			"skip_count", "current_streak", "score", "updated_at",
		}),
	}).Create(m).Error
}

func (r *ScoreRepository) GetAll() ([]*domain.Score, error) {
	var models []scoreModel
	if err := r.db.Find(&models).Error; err != nil {
		return nil, err
	}
	out := make([]*domain.Score, len(models))
	for i, m := range models {
		out[i] = toDomainScore(&m)
	}
	return out, nil
}

func (r *ScoreRepository) GetByUserID(userID string) (*domain.Score, error) {
	var m scoreModel
	result := r.db.Where("user_id = ?", userID).First(&m)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if result.Error != nil {
		return nil, result.Error
	}
	return toDomainScore(&m), nil
}

func toDomainScore(m *scoreModel) *domain.Score {
	return &domain.Score{
		UserID:            m.UserID,
		TimesPaid:         m.TimesPaid,
		TimesParticipated: m.TimesParticipated,
		TotalAmountSpent:  m.TotalAmountSpent,
		SkipCount:         m.SkipCount,
		CurrentStreak:     m.CurrentStreak,
		Score:             m.Score,
		UpdatedAt:         m.UpdatedAt,
	}
}

// Verify interface compliance at compile time.
var _ domain.ScoreRepository = (*ScoreRepository)(nil)
