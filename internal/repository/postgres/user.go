package postgres

import (
	"errors"
	"time"

	"github.com/antoniobt12062002/pao-de-queijo/internal/domain"
	"gorm.io/gorm"
)

type userModel struct {
	ID           string    `gorm:"type:uuid;column:id;primaryKey;default:gen_random_uuid()"`
	Name         string    `gorm:"column:name"`
	Email        string    `gorm:"column:email"`
	PasswordHash *string   `gorm:"column:password_hash"`
	Role         string    `gorm:"column:role"`
	Phone        *string   `gorm:"column:phone"`
	Provider     string    `gorm:"column:provider"`
	ProviderID   *string   `gorm:"column:provider_id"`
	CreatedAt    time.Time `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt    time.Time `gorm:"column:updated_at;autoUpdateTime"`
}

func (userModel) TableName() string { return "users" }

type UserRepository struct {
	db *gorm.DB
}

func NewUserRepository(db *gorm.DB) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) DB() *gorm.DB { return r.db }

func (r *UserRepository) Create(u *domain.User) error {
	m := toModel(u)
	result := r.db.Create(m)
	if result.Error != nil {
		return result.Error
	}
	u.ID = m.ID
	return nil
}

func (r *UserRepository) FindByEmail(email string) (*domain.User, error) {
	var m userModel
	result := r.db.Where("email = ?", email).First(&m)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if result.Error != nil {
		return nil, result.Error
	}
	return toDomain(&m), nil
}

func (r *UserRepository) FindByProviderID(provider, providerID string) (*domain.User, error) {
	var m userModel
	result := r.db.Where("provider = ? AND provider_id = ?", provider, providerID).First(&m)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if result.Error != nil {
		return nil, result.Error
	}
	return toDomain(&m), nil
}

func (r *UserRepository) FindByID(id string) (*domain.User, error) {
	var m userModel
	result := r.db.Where("id = ?", id).First(&m)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if result.Error != nil {
		return nil, result.Error
	}
	return toDomain(&m), nil
}

func (r *UserRepository) FindAll() ([]*domain.User, error) {
	var models []userModel
	if err := r.db.Find(&models).Error; err != nil {
		return nil, err
	}
	users := make([]*domain.User, len(models))
	for i, m := range models {
		users[i] = toDomain(&m)
	}
	return users, nil
}

func toModel(u *domain.User) *userModel {
	return &userModel{
		ID:           u.ID,
		Name:         u.Name,
		Email:        u.Email,
		PasswordHash: u.PasswordHash,
		Role:         u.Role,
		Phone:        u.Phone,
		Provider:     u.Provider,
		ProviderID:   u.ProviderID,
	}
}

func toDomain(m *userModel) *domain.User {
	return &domain.User{
		ID:           m.ID,
		Name:         m.Name,
		Email:        m.Email,
		PasswordHash: m.PasswordHash,
		Role:         m.Role,
		Phone:        m.Phone,
		Provider:     m.Provider,
		ProviderID:   m.ProviderID,
		CreatedAt:    m.CreatedAt,
		UpdatedAt:    m.UpdatedAt,
	}
}
