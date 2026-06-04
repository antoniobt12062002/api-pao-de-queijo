package domain

import "time"

// BadgeType classifies the badge.
type BadgeType string

const (
	BadgeNovoNaFila    BadgeType = "novo_na_fila"
	BadgeNuncaFoge     BadgeType = "nunca_foge"
	BadgeQueijeiroFiel BadgeType = "queijeiro_fiel"
	BadgePapaiNoel     BadgeType = "papai_noel"
	BadgeBigSpender    BadgeType = "big_spender"
)

// Score holds the justice score for a user.
type Score struct {
	UserID            string    `json:"user_id"`
	TimesPaid         int       `json:"times_paid"`
	TimesParticipated int       `json:"times_participated"`
	TotalAmountSpent  float64   `json:"total_amount_spent"`
	SkipCount         int       `json:"skip_count"`
	CurrentStreak     int       `json:"current_streak"`
	Score             float64   `json:"score"`
	UpdatedAt         time.Time `json:"updated_at"`
}

// Badge is an earned achievement.
type Badge struct {
	ID       string    `json:"id"`
	UserID   string    `json:"user_id"`
	Type     BadgeType `json:"type"`
	Period   string    `json:"period,omitempty"` // "YYYY-MM" for monthly; "" for permanent
	EarnedAt time.Time `json:"earned_at"`
}

// ScoreRepository persists Score records.
type ScoreRepository interface {
	Upsert(s *Score) error
	GetAll() ([]*Score, error)
	GetByUserID(userID string) (*Score, error)
}

// BadgeRepository persists Badge records.
type BadgeRepository interface {
	// Insert inserts a badge using INSERT ... ON CONFLICT DO NOTHING.
	// It never updates existing rows — idempotent by design.
	Insert(b *Badge) error
	GetByUserID(userID string) ([]*Badge, error)
	// GetMonthlyTopRoundPayer returns the payer_id of the round with the most
	// participants in the given month ("YYYY-MM"). Returns "" if no closed rounds.
	GetMonthlyTopRoundPayer(month string) (string, error)
	// GetMonthlyBigSpender returns the user_id with the highest total spending
	// as a payer in the given month. pricePerUnit is the current config value.
	// Returns "" if no closed rounds in the month.
	GetMonthlyBigSpender(month string, pricePerUnit float64) (string, error)
}

// BadgeChecker awards badges after a round closes.
type BadgeChecker interface {
	CheckAfterRound(roundID string) error
}

// ScoreUpdater updates scores after a round event.
type ScoreUpdater interface {
	// UpdateAfterRound is called after a round is closed (open → closed).
	// It updates times_paid, times_participated, streaks, total_amount_spent,
	// and recalculates scores for all affected users.
	UpdateAfterRound(roundID string) error
	// UpdateOnCancel is called after a round is cancelled (pending → cancelled).
	// It increments skip_count for the payer.
	UpdateOnCancel(roundID string) error
}

// NoopScoreUpdater is the stub used until feature/score-badges wires the real service.
type NoopScoreUpdater struct{}

func (n *NoopScoreUpdater) UpdateAfterRound(roundID string) error { return nil }
func (n *NoopScoreUpdater) UpdateOnCancel(roundID string) error   { return nil }
