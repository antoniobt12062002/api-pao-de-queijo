package postgres

import (
	"errors"
	"time"

	"github.com/antoniobt12062002/pao-de-queijo/internal/domain"
	"gorm.io/gorm"
)

type roundModel struct {
	ID         string   `gorm:"column:id;primaryKey;type:uuid;default:gen_random_uuid()"`
	Date       string   `gorm:"column:date;type:date"`
	PayerID    string   `gorm:"column:payer_id;type:uuid"`
	Status     string   `gorm:"column:status"`
	NotifyAt   time.Time `gorm:"column:notify_at"`
	ClosesAt   time.Time `gorm:"column:closes_at"`
	CreatedAt  time.Time `gorm:"column:created_at;autoCreateTime"`
	ActualCost *float64  `gorm:"column:actual_cost"`
}

func (roundModel) TableName() string { return "rounds" }

type RoundRepository struct {
	db *gorm.DB
}

func NewRoundRepository(db *gorm.DB) *RoundRepository {
	return &RoundRepository{db: db}
}

func (r *RoundRepository) Create(round *domain.Round) error {
	m := toRoundModel(round)
	if err := r.db.Create(m).Error; err != nil {
		return err
	}
	round.ID = m.ID
	return nil
}

func (r *RoundRepository) GetByDate(date string) (*domain.Round, error) {
	var m roundModel
	result := r.db.Where("date = ?", date).First(&m)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if result.Error != nil {
		return nil, result.Error
	}
	return toDomainRound(&m), nil
}

func (r *RoundRepository) GetByID(id string) (*domain.Round, error) {
	var m roundModel
	result := r.db.Where("id = ?", id).First(&m)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if result.Error != nil {
		return nil, result.Error
	}
	return toDomainRound(&m), nil
}

func (r *RoundRepository) GetAll(page, limit int) ([]*domain.Round, int64, error) {
	var models []roundModel
	var total int64

	if err := r.db.Model(&roundModel{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * limit
	if err := r.db.Order("created_at DESC").Offset(offset).Limit(limit).Find(&models).Error; err != nil {
		return nil, 0, err
	}

	rounds := make([]*domain.Round, len(models))
	for i, m := range models {
		rounds[i] = toDomainRound(&m)
	}
	return rounds, total, nil
}

func (r *RoundRepository) Update(round *domain.Round) error {
	m := toRoundModel(round)
	return r.db.Save(m).Error
}

func toRoundModel(r *domain.Round) *roundModel {
	return &roundModel{
		ID:         r.ID,
		Date:       r.Date,
		PayerID:    r.PayerID,
		Status:     string(r.Status),
		NotifyAt:   r.NotifyAt,
		ClosesAt:   r.ClosesAt,
		ActualCost: r.ActualCost,
	}
}

func toDomainRound(m *roundModel) *domain.Round {
	return &domain.Round{
		ID:         m.ID,
		Date:       m.Date,
		PayerID:    m.PayerID,
		Status:     domain.RoundStatus(m.Status),
		NotifyAt:   m.NotifyAt,
		ClosesAt:   m.ClosesAt,
		CreatedAt:  m.CreatedAt,
		ActualCost: m.ActualCost,
	}
}

// Verify interface compliance at compile time
var _ domain.RoundRepository = (*RoundRepository)(nil)
