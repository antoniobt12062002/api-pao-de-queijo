package notification

import (
	"context"
	"log/slog"

	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/messaging"
	"google.golang.org/api/option"

	"github.com/antoniobt12062002/pao-de-queijo/internal/domain"
)

// FCMSender is the subset of messaging.Client used by FCMNotificationService.
// Defined as an interface so tests can inject a mock.
type FCMSender interface {
	SendEachForMulticast(ctx context.Context, message *messaging.MulticastMessage) (*messaging.BatchResponse, error)
}

// FCMNotificationService implements domain.NotificationService using Firebase FCM.
// All methods are fire-and-forget: errors are logged but never returned to callers.
type FCMNotificationService struct {
	fcm        FCMSender
	deviceRepo domain.DeviceTokenRepository
	notifRepo  domain.NotificationRepository
}

// NewFCMNotificationService creates a production instance using Firebase credentials JSON.
func NewFCMNotificationService(credentialsJSON []byte, deviceRepo domain.DeviceTokenRepository, notifRepo domain.NotificationRepository) (*FCMNotificationService, error) {
	app, err := firebase.NewApp(context.Background(), nil, option.WithCredentialsJSON(credentialsJSON))
	if err != nil {
		return nil, err
	}
	client, err := app.Messaging(context.Background())
	if err != nil {
		return nil, err
	}
	return &FCMNotificationService{
		fcm:        client,
		deviceRepo: deviceRepo,
		notifRepo:  notifRepo,
	}, nil
}

// NewFCMNotificationServiceWithSender creates a testable instance with an injected sender.
func NewFCMNotificationServiceWithSender(fcm FCMSender, deviceRepo domain.DeviceTokenRepository, notifRepo domain.NotificationRepository) *FCMNotificationService {
	return &FCMNotificationService{fcm: fcm, deviceRepo: deviceRepo, notifRepo: notifRepo}
}

func (s *FCMNotificationService) SendRoundAnnounced(payerID, roundID string) error {
	s.sendToUsers(context.Background(), []string{payerID}, roundID, domain.NotifRoundAnnounced, "Pão de Queijo hoje!", "Você é o pagador de hoje. Confirme a rodada.")
	return nil
}

func (s *FCMNotificationService) SendParticipationOpen(userIDs []string, roundID string) error {
	s.sendToUsers(context.Background(), userIDs, roundID, domain.NotifParticipationOpen, "Pão de Queijo aberto!", "A rodada está aberta. Registre sua participação.")
	return nil
}

func (s *FCMNotificationService) SendRoundClosed(payerID, roundID string) error {
	s.sendToUsers(context.Background(), []string{payerID}, roundID, domain.NotifRoundClosed, "Rodada encerrada", "A janela de participação foi fechada.")
	return nil
}

func (s *FCMNotificationService) SendReminder(participantIDs []string, roundID string) error {
	s.sendToUsers(context.Background(), participantIDs, roundID, domain.NotifReminder, "Lembrete: Pão de Queijo", "A rodada fecha em breve!")
	return nil
}

// sendToUsers fetches FCM tokens for all userIDs, sends a multicast message,
// cleans up invalid tokens, and logs one Notification entry per user.
func (s *FCMNotificationService) sendToUsers(ctx context.Context, userIDs []string, roundID string, notifType domain.NotificationType, title, body string) {
	tokens, err := s.deviceRepo.GetTokensByUserIDs(userIDs)
	if err != nil {
		slog.Error("FCMNotificationService: error fetching tokens", "type", notifType, "err", err)
		return
	}
	if len(tokens) == 0 {
		slog.Warn("FCMNotificationService: no tokens found for users", "type", notifType, "userIDs", userIDs)
		return
	}
	slog.Info("FCMNotificationService: sending notification", "type", notifType, "tokenCount", len(tokens))

	msg := &messaging.MulticastMessage{
		Tokens: tokens,
		Data: map[string]string{
			"title": title,
			"body":  body,
		},
	}

	br, err := s.fcm.SendEachForMulticast(ctx, msg)
	if err != nil {
		slog.Error("FCMNotificationService: multicast error", "type", notifType, "err", err)
		errMsg := err.Error()
		for _, userID := range userIDs {
			_ = s.notifRepo.Create(&domain.Notification{
				UserID: userID, RoundID: roundID, Type: notifType,
				Channel: domain.ChannelPush, Success: false, ErrorMessage: errMsg,
			})
		}
		return
	}

	slog.Info("FCMNotificationService: multicast sent", "type", notifType, "successCount", br.SuccessCount, "failureCount", br.FailureCount)

	for i, resp := range br.Responses {
		if !resp.Success {
			if messaging.IsUnregistered(resp.Error) || messaging.IsInvalidArgument(resp.Error) {
				if deleteErr := s.deviceRepo.DeleteByToken(tokens[i]); deleteErr != nil {
					slog.Error("FCMNotificationService: error deleting invalid token", "token", tokens[i], "err", deleteErr)
				}
			} else {
				slog.Error("FCMNotificationService: failed to send to token", "token", tokens[i], "err", resp.Error)
			}
		}
	}

	// Log one entry per user: success if at least one of their tokens was delivered.
	overallSuccess := br.SuccessCount > 0
	var overallErrMsg string
	if br.FailureCount > 0 && br.SuccessCount == 0 {
		overallErrMsg = "all tokens failed"
	}
	for _, userID := range userIDs {
		if logErr := s.notifRepo.Create(&domain.Notification{
			UserID:       userID,
			RoundID:      roundID,
			Type:         notifType,
			Channel:      domain.ChannelPush,
			Success:      overallSuccess,
			ErrorMessage: overallErrMsg,
		}); logErr != nil {
			slog.Error("FCMNotificationService: error logging notification", "type", notifType, "userID", userID, "err", logErr)
		}
	}
}

// Verify interface compliance at compile time.
var _ domain.NotificationService = (*FCMNotificationService)(nil)
