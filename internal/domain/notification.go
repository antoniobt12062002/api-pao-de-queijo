package domain

// NotificationService envia notificações aos usuários.
// Implementação real será feita em feature/notification.
type NotificationService interface {
	SendRoundAnnounced(payerID string) error
	SendRoundClosed(payerID string) error
	SendReminder(participantIDs []string) error
	SendParticipationOpen(userIDs []string) error
}

// NoopNotificationService é a implementação stub usada até feature/notification.
type NoopNotificationService struct{}

func (n *NoopNotificationService) SendRoundAnnounced(payerID string) error       { return nil }
func (n *NoopNotificationService) SendRoundClosed(payerID string) error          { return nil }
func (n *NoopNotificationService) SendReminder(participantIDs []string) error    { return nil }
func (n *NoopNotificationService) SendParticipationOpen(userIDs []string) error  { return nil }
