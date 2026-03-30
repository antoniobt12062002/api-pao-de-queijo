package domain

import (
	"errors"
	"time"
)

var (
	ErrRotationNotInitialized = errors.New("rotation not initialized")
	ErrRotationEmptyOrder     = errors.New("rotation order cannot be empty")
	ErrRotationDuplicateUser  = errors.New("rotation order contains duplicate users")
	ErrRotationUnknownUser    = errors.New("rotation order contains unknown user")
)

type RotationMember struct {
	UserID   string `json:"user_id"`
	Position int    `json:"position"`
}

type Rotation struct {
	ID         string            `json:"id"`
	CurrentPos int               `json:"current_pos"`
	UpdatedAt  time.Time         `json:"updated_at"`
	Members    []*RotationMember `json:"members"`
}

// CurrentPayerID returns the user_id at current_pos.
// Members must be sorted by position (0..n-1) — the repository guarantees this.
func (r *Rotation) CurrentPayerID() string {
	if len(r.Members) == 0 || r.CurrentPos >= len(r.Members) {
		return ""
	}
	return r.Members[r.CurrentPos].UserID
}

type RotationRepository interface {
	Get() (*Rotation, error)         // returns nil if no rotation exists yet
	SetOrder(userIDs []string) error // full replacement + resets current_pos to 0
	AdvancePosition() error          // circular increment of current_pos
}
