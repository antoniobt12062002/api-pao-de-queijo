package postgres

import (
	"time"

	"github.com/antoniobt12062002/pao-de-queijo/internal/domain"
	"gorm.io/gorm"
)

type badgeModel struct {
	ID       string    `gorm:"column:id;primaryKey;type:uuid;default:gen_random_uuid()"`
	UserID   string    `gorm:"column:user_id;type:uuid"`
	Type     string    `gorm:"column:type"`
	Period   string    `gorm:"column:period"`
	EarnedAt time.Time `gorm:"column:earned_at;autoCreateTime"`
}

func (badgeModel) TableName() string { return "badges" }

type BadgeRepository struct {
	db *gorm.DB
}

func NewBadgeRepository(db *gorm.DB) *BadgeRepository {
	return &BadgeRepository{db: db}
}

// Insert inserts a badge using INSERT ... ON CONFLICT DO NOTHING.
func (r *BadgeRepository) Insert(b *domain.Badge) error {
	return r.db.Exec(
		`INSERT INTO badges (id, user_id, type, period, earned_at)
		 VALUES (gen_random_uuid(), ?, ?, ?, NOW())
		 ON CONFLICT DO NOTHING`,
		b.UserID, string(b.Type), b.Period,
	).Error
}

func (r *BadgeRepository) GetByUserID(userID string) ([]*domain.Badge, error) {
	var models []badgeModel
	if err := r.db.Where("user_id = ?", userID).Find(&models).Error; err != nil {
		return nil, err
	}
	out := make([]*domain.Badge, len(models))
	for i, m := range models {
		out[i] = toDomainBadge(&m)
	}
	return out, nil
}

// GetMonthlyTopRoundPayer returns the payer_id of the round with the most participants
// in the given month ("YYYY-MM"). Returns "" if no closed rounds exist in the month.
func (r *BadgeRepository) GetMonthlyTopRoundPayer(month string) (string, error) {
	var result struct {
		PayerID string `gorm:"column:payer_id"`
	}
	err := r.db.Raw(`
		SELECT r.payer_id
		FROM rounds r
		JOIN (
			SELECT round_id, COUNT(*) AS participant_count
			FROM participations
			GROUP BY round_id
		) pc ON pc.round_id = r.id
		WHERE r.status = 'closed'
		  AND TO_CHAR(r.date, 'YYYY-MM') = ?
		ORDER BY pc.participant_count DESC
		LIMIT 1
	`, month).Scan(&result).Error
	if err != nil {
		return "", err
	}
	return result.PayerID, nil
}

// GetMonthlyBigSpender returns the user_id with the highest total spending
// as payer in the given month. Returns "" if no closed rounds in the month.
func (r *BadgeRepository) GetMonthlyBigSpender(month string, pricePerUnit float64) (string, error) {
	var result struct {
		PayerID string `gorm:"column:payer_id"`
	}
	err := r.db.Raw(`
		SELECT r.payer_id
		FROM rounds r
		JOIN participations p ON p.round_id = r.id
		WHERE r.status = 'closed'
		  AND TO_CHAR(r.date, 'YYYY-MM') = ?
		GROUP BY r.payer_id
		ORDER BY SUM(p.quantity) * ? DESC
		LIMIT 1
	`, month, pricePerUnit).Scan(&result).Error
	if err != nil {
		return "", err
	}
	return result.PayerID, nil
}

func toDomainBadge(m *badgeModel) *domain.Badge {
	return &domain.Badge{
		ID:       m.ID,
		UserID:   m.UserID,
		Type:     domain.BadgeType(m.Type),
		Period:   m.Period,
		EarnedAt: m.EarnedAt,
	}
}

// Verify interface compliance at compile time.
var _ domain.BadgeRepository = (*BadgeRepository)(nil)
