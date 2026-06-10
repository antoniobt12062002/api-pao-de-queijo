package postgres

import (
	"time"

	"github.com/antoniobt12062002/pao-de-queijo/internal/domain"
	"gorm.io/gorm"
)

type notificationLogModel struct {
	ID           string    `gorm:"column:id;primaryKey;type:uuid;default:gen_random_uuid()"`
	UserID       string    `gorm:"column:user_id;type:uuid"`
	RoundID      string    `gorm:"column:round_id;type:uuid"`
	Type         string    `gorm:"column:type"`
	SentAt       time.Time `gorm:"column:sent_at;autoCreateTime"`
	Channel      string    `gorm:"column:channel"`
	Success      bool      `gorm:"column:success;default:true"`
	ErrorMessage string    `gorm:"column:error_message"`
}

func (notificationLogModel) TableName() string { return "notifications" }

type NotificationLogRepository struct {
	db *gorm.DB
}

func NewNotificationLogRepository(db *gorm.DB) *NotificationLogRepository {
	return &NotificationLogRepository{db: db}
}

func (r *NotificationLogRepository) Create(n *domain.Notification) error {
	m := &notificationLogModel{
		UserID:       n.UserID,
		RoundID:      n.RoundID,
		Type:         string(n.Type),
		Channel:      string(n.Channel),
		Success:      n.Success,
		ErrorMessage: n.ErrorMessage,
	}
	return r.db.Create(m).Error
}

// Verify interface compliance at compile time.
var _ domain.NotificationRepository = (*NotificationLogRepository)(nil)
