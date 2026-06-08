package postgres

import (
	"errors"
	"time"

	"github.com/antoniobt12062002/pao-de-queijo/internal/domain"
	"gorm.io/gorm"
)

type deviceTokenModel struct {
	ID        string    `gorm:"column:id;primaryKey;type:uuid;default:gen_random_uuid()"`
	UserID    string    `gorm:"column:user_id;type:uuid"`
	Token     string    `gorm:"column:token"`
	Platform  string    `gorm:"column:platform"`
	CreatedAt time.Time `gorm:"column:created_at;autoCreateTime"`
}

func (deviceTokenModel) TableName() string { return "device_tokens" }

type DeviceTokenRepository struct {
	db *gorm.DB
}

func NewDeviceTokenRepository(db *gorm.DB) *DeviceTokenRepository {
	return &DeviceTokenRepository{db: db}
}

// Upsert inserts a new token or updates user_id/platform if the token already exists.
// Note: dt.ID is not back-filled because this uses a raw Exec (no model scan).
// Callers do not need the generated ID after an upsert.
func (r *DeviceTokenRepository) Upsert(dt *domain.DeviceToken) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		// Remove tokens anteriores do mesmo usuário e plataforma (exceto o novo token)
		if err := tx.Exec(
			`DELETE FROM device_tokens WHERE user_id = ? AND token != ?`,
			dt.UserID, dt.Token,
		).Error; err != nil {
			return err
		}
		return tx.Exec(
			`INSERT INTO device_tokens (id, user_id, token, platform, created_at)
			 VALUES (gen_random_uuid(), ?, ?, ?, now())
			 ON CONFLICT (token) DO UPDATE SET user_id = EXCLUDED.user_id, platform = EXCLUDED.platform`,
			dt.UserID, dt.Token, string(dt.Platform),
		).Error
	})
}

func (r *DeviceTokenRepository) GetByToken(token string) (*domain.DeviceToken, error) {
	var m deviceTokenModel
	result := r.db.Where("token = ?", token).First(&m)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if result.Error != nil {
		return nil, result.Error
	}
	return toDomainDeviceToken(&m), nil
}

func (r *DeviceTokenRepository) GetTokensByUserIDs(userIDs []string) ([]string, error) {
	if len(userIDs) == 0 {
		return nil, nil
	}
	var tokens []string
	err := r.db.Model(&deviceTokenModel{}).
		Where("user_id IN ?", userIDs).
		Pluck("token", &tokens).Error
	return tokens, err
}

func (r *DeviceTokenRepository) DeleteByToken(token string) error {
	result := r.db.Where("token = ?", token).Delete(&deviceTokenModel{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return domain.ErrDeviceTokenNotFound
	}
	return nil
}

func toDomainDeviceToken(m *deviceTokenModel) *domain.DeviceToken {
	return &domain.DeviceToken{
		ID:        m.ID,
		UserID:    m.UserID,
		Token:     m.Token,
		Platform:  domain.DeviceTokenPlatform(m.Platform),
		CreatedAt: m.CreatedAt,
	}
}

// Verify interface compliance at compile time.
var _ domain.DeviceTokenRepository = (*DeviceTokenRepository)(nil)
