package postgres

import (
	"errors"
	"time"

	"github.com/antoniobt12062002/pao-de-queijo/internal/domain"
	"gorm.io/gorm"
)

type absenceModel struct {
	ID        string    `gorm:"column:id;primaryKey;type:uuid;default:gen_random_uuid()"`
	UserID    string    `gorm:"column:user_id;type:uuid"`
	Date      string    `gorm:"column:date;type:date"`
	CreatedAt time.Time `gorm:"column:created_at;autoCreateTime"`
}

func (absenceModel) TableName() string { return "absences" }

type AbsenceRepository struct {
	db *gorm.DB
}

func NewAbsenceRepository(db *gorm.DB) *AbsenceRepository {
	return &AbsenceRepository{db: db}
}

func (r *AbsenceRepository) Create(a *domain.Absence) error {
	m := &absenceModel{UserID: a.UserID, Date: a.Date}
	if err := r.db.Create(m).Error; err != nil {
		if isUniqueViolation(err) {
			return domain.ErrAbsenceAlreadyExists
		}
		return err
	}
	a.ID = m.ID
	return nil
}

func (r *AbsenceRepository) Delete(userID, date string) error {
	res := r.db.Where("user_id = ? AND date = ?", userID, date).Delete(&absenceModel{})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return domain.ErrAbsenceNotFound
	}
	return nil
}

func (r *AbsenceRepository) GetByUser(userID string) ([]*domain.Absence, error) {
	var models []absenceModel
	if err := r.db.Where("user_id = ? AND date >= CURRENT_DATE", userID).
		Order("date ASC").Find(&models).Error; err != nil {
		return nil, err
	}
	out := make([]*domain.Absence, len(models))
	for i, m := range models {
		out[i] = &domain.Absence{ID: m.ID, UserID: m.UserID, Date: m.Date, CreatedAt: m.CreatedAt}
	}
	return out, nil
}

func (r *AbsenceRepository) GetAbsentUserIDsForDate(date string) ([]string, error) {
	var ids []string
	err := r.db.Model(&absenceModel{}).Where("date = ?", date).Pluck("user_id", &ids).Error
	return ids, err
}

func isUniqueViolation(err error) bool {
	var pgErr interface{ SQLState() string }
	if errors.As(err, &pgErr) {
		return pgErr.SQLState() == "23505"
	}
	return false
}

var _ domain.AbsenceRepository = (*AbsenceRepository)(nil)
