package domain

import (
	"errors"
	"time"
)

var ErrAbsenceAlreadyExists = errors.New("absence already registered for this date")
var ErrAbsenceNotFound = errors.New("absence not found")

type Absence struct {
	ID        string
	UserID    string
	Date      string
	CreatedAt time.Time
}

type AbsenceRepository interface {
	Create(a *Absence) error
	Delete(userID, date string) error
	GetByUser(userID string) ([]*Absence, error)
	GetAbsentUserIDsForDate(date string) ([]string, error)
}
