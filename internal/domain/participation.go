package domain

import (
	"errors"
	"time"
)

var (
	ErrParticipationNotFound  = errors.New("participation not found")
	ErrRoundNotOpen           = errors.New("round is not open")
	ErrAlreadyParticipating   = errors.New("already participating in this round")
)

type Participation struct {
	ID          string    `json:"id"`
	RoundID     string    `json:"round_id"`
	UserID      string    `json:"user_id"`
	Quantity    int       `json:"quantity"`
	ConfirmedAt time.Time `json:"confirmed_at"`
}

type ParticipationRepository interface {
	Create(p *Participation) error
	GetByRoundAndUser(roundID, userID string) (*Participation, error)
	GetByRound(roundID string) ([]*Participation, error)
	Delete(roundID, userID string) error
}
