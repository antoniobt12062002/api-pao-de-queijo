package postgres

import (
	"errors"
	"time"

	"github.com/antoniobt12062002/pao-de-queijo/internal/domain"
	"gorm.io/gorm"
)

type rotationModel struct {
	ID         string    `gorm:"column:id;primaryKey;type:uuid;default:gen_random_uuid()"`
	CurrentPos int       `gorm:"column:current_pos"`
	UpdatedAt  time.Time `gorm:"column:updated_at;autoUpdateTime"`
}

func (rotationModel) TableName() string { return "rotations" }

type rotationMemberModel struct {
	RotationID string `gorm:"column:rotation_id"`
	UserID     string `gorm:"column:user_id"`
	Position   int    `gorm:"column:position"`
}

func (rotationMemberModel) TableName() string { return "rotation_members" }

type RotationRepository struct {
	db *gorm.DB
}

func NewRotationRepository(db *gorm.DB) *RotationRepository {
	return &RotationRepository{db: db}
}

func (r *RotationRepository) Get() (*domain.Rotation, error) {
	var rm rotationModel
	result := r.db.First(&rm)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if result.Error != nil {
		return nil, result.Error
	}

	var memberModels []rotationMemberModel
	if err := r.db.Where("rotation_id = ?", rm.ID).Order("position ASC").Find(&memberModels).Error; err != nil {
		return nil, err
	}

	members := make([]*domain.RotationMember, len(memberModels))
	for i, m := range memberModels {
		members[i] = &domain.RotationMember{UserID: m.UserID, Position: m.Position}
	}

	return &domain.Rotation{
		ID:         rm.ID,
		CurrentPos: rm.CurrentPos,
		UpdatedAt:  rm.UpdatedAt,
		Members:    members,
	}, nil
}

func (r *RotationRepository) SetOrder(userIDs []string) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		// Upsert rotation singleton
		var rm rotationModel
		result := tx.First(&rm)
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			rm = rotationModel{CurrentPos: 0}
			if err := tx.Create(&rm).Error; err != nil {
				return err
			}
		} else if result.Error != nil {
			return result.Error
		} else {
			if err := tx.Model(&rm).Update("current_pos", 0).Error; err != nil {
				return err
			}
		}

		// Delete all existing members for this rotation
		if err := tx.Where("rotation_id = ?", rm.ID).Delete(&rotationMemberModel{}).Error; err != nil {
			return err
		}

		// Insert new members
		members := make([]rotationMemberModel, len(userIDs))
		for i, uid := range userIDs {
			members[i] = rotationMemberModel{
				RotationID: rm.ID,
				UserID:     uid,
				Position:   i,
			}
		}
		return tx.Create(&members).Error
	})
}

func (r *RotationRepository) AdvancePosition() error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		var rm rotationModel
		if err := tx.First(&rm).Error; err != nil {
			return err
		}

		var count int64
		if err := tx.Model(&rotationMemberModel{}).Where("rotation_id = ?", rm.ID).Count(&count).Error; err != nil {
			return err
		}
		if count == 0 {
			return domain.ErrRotationNotInitialized
		}

		nextPos := (rm.CurrentPos + 1) % int(count)
		return tx.Model(&rm).Update("current_pos", nextPos).Error
	})
}

// Verify interface compliance at compile time
var _ domain.RotationRepository = (*RotationRepository)(nil)
