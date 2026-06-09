package domain

import (
	"errors"
	"time"
)

var (
	ErrRoundNotFound      = errors.New("round not found")
	ErrRoundNotPending    = errors.New("round is not in pending status")
	ErrRoundNotPayer      = errors.New("only the current payer can perform this action")
	ErrRotationEmpty      = errors.New("rotation has no members configured")
	ErrRoundAlreadyExists = errors.New("round already exists for this date")
)

type RoundStatus string

const (
	RoundStatusPending   RoundStatus = "pending"
	RoundStatusOpen      RoundStatus = "open"
	RoundStatusClosed    RoundStatus = "closed"
	RoundStatusCancelled RoundStatus = "cancelled"
)

type Round struct {
	ID        string      `json:"id"`
	Date      string      `json:"date"` // "YYYY-MM-DD"
	PayerID   string      `json:"payer_id"`
	Status    RoundStatus `json:"status"`
	NotifyAt  time.Time   `json:"notify_at"`
	ClosesAt  time.Time   `json:"closes_at"`
	CreatedAt time.Time   `json:"created_at"`
}

type RoundRepository interface {
	Create(round *Round) error
	GetByDate(date string) (*Round, error)
	GetByID(id string) (*Round, error)
	GetAll(page, limit int) ([]*Round, int64, error)
	Update(round *Round) error
}
