package postgres

import (
	"github.com/antoniobt12062002/pao-de-queijo/internal/domain"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type configModel struct {
	Key   string `gorm:"column:key;primaryKey"`
	Value string `gorm:"column:value"`
}

func (configModel) TableName() string { return "configs" }

type ConfigRepository struct {
	db *gorm.DB
}

func NewConfigRepository(db *gorm.DB) *ConfigRepository {
	return &ConfigRepository{db: db}
}

func (r *ConfigRepository) GetAll() ([]*domain.Config, error) {
	var models []configModel
	if err := r.db.Find(&models).Error; err != nil {
		return nil, err
	}
	configs := make([]*domain.Config, len(models))
	for i, m := range models {
		configs[i] = &domain.Config{Key: m.Key, Value: m.Value}
	}
	return configs, nil
}

func (r *ConfigRepository) Set(key, value string) error {
	m := configModel{Key: key, Value: value}
	return r.db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "key"}},
		DoUpdates: clause.AssignmentColumns([]string{"value"}),
	}).Create(&m).Error
}

// Verify interface compliance at compile time
var _ domain.ConfigRepository = (*ConfigRepository)(nil)
