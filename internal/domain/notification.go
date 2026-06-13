package domain

import "time"

// NotificationType classifies the event.
type NotificationType string

const (
	NotifRoundAnnounced    NotificationType = "round_announced"
	NotifParticipationOpen NotificationType = "participation_open"
	NotifRoundClosed       NotificationType = "round_closed"
	NotifReminder          NotificationType = "reminder"
	NotifManual            NotificationType = "manual"
)

// NotificationChannel is the delivery channel.
type NotificationChannel string

const (
	ChannelPush NotificationChannel = "push"
	ChannelWeb  NotificationChannel = "web"
)

// Notification is an audit log entry for a sent notification.
type Notification struct {
	ID           string
	UserID       string
	RoundID      *string // nullable — nil for manual notifications
	Type         NotificationType
	SentAt       time.Time
	Channel      NotificationChannel
	Success      bool
	ErrorMessage string
}

// NotificationRepository persists notification log entries.
type NotificationRepository interface {
	Create(n *Notification) error
}

// NotificationService sends notifications to users.
type NotificationService interface {
	SendRoundAnnounced(payerID, roundID string) error
	SendParticipationOpen(userIDs []string, roundID string) error
	SendRoundClosed(payerID, roundID string) error
	SendReminder(participantIDs []string, roundID string) error
	SendManual(userIDs []string, title, body string) error
}

// NoopNotificationService is the stub used when Firebase is not configured.
type NoopNotificationService struct{}

func (n *NoopNotificationService) SendRoundAnnounced(payerID, roundID string) error             { return nil }
func (n *NoopNotificationService) SendParticipationOpen(userIDs []string, roundID string) error { return nil }
func (n *NoopNotificationService) SendRoundClosed(payerID, roundID string) error               { return nil }
func (n *NoopNotificationService) SendReminder(participantIDs []string, roundID string) error  { return nil }
func (n *NoopNotificationService) SendManual(userIDs []string, title, body string) error       { return nil }
