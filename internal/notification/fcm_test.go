package notification_test

import (
	"context"
	"fmt"
	"testing"

	"firebase.google.com/go/v4/messaging"
	"github.com/antoniobt12062002/pao-de-queijo/internal/domain"
	"github.com/antoniobt12062002/pao-de-queijo/internal/notification"
)

// --- mocks ---

type mockFCMSender struct {
	calls []*messaging.MulticastMessage
	resp  *messaging.BatchResponse
	err   error
}

func (m *mockFCMSender) SendEachForMulticast(_ context.Context, msg *messaging.MulticastMessage) (*messaging.BatchResponse, error) {
	m.calls = append(m.calls, msg)
	if m.resp != nil {
		return m.resp, m.err
	}
	// Default: all success
	responses := make([]*messaging.SendResponse, len(msg.Tokens))
	for i := range responses {
		responses[i] = &messaging.SendResponse{Success: true}
	}
	return &messaging.BatchResponse{
		SuccessCount: len(msg.Tokens),
		Responses:    responses,
	}, m.err
}

type mockDeviceTokenRepo struct {
	tokensByUser map[string][]string // userID -> tokens
	deleted      []string
}

func newMockDeviceTokenRepo() *mockDeviceTokenRepo {
	return &mockDeviceTokenRepo{tokensByUser: make(map[string][]string)}
}

func (m *mockDeviceTokenRepo) Upsert(dt *domain.DeviceToken) error { return nil }

func (m *mockDeviceTokenRepo) GetByToken(token string) (*domain.DeviceToken, error) { return nil, nil }

func (m *mockDeviceTokenRepo) GetTokensByUserIDs(userIDs []string) ([]string, error) {
	var result []string
	for _, id := range userIDs {
		result = append(result, m.tokensByUser[id]...)
	}
	return result, nil
}

func (m *mockDeviceTokenRepo) DeleteByToken(token string) error {
	m.deleted = append(m.deleted, token)
	return nil
}

type mockNotifRepo struct {
	created []*domain.Notification
}

func (m *mockNotifRepo) Create(n *domain.Notification) error {
	m.created = append(m.created, n)
	return nil
}

// --- tests ---

func TestFCMService_SendRoundAnnounced_NoTokens(t *testing.T) {
	sender := &mockFCMSender{}
	deviceRepo := newMockDeviceTokenRepo()
	notifRepo := &mockNotifRepo{}

	svc := notification.NewFCMNotificationServiceWithSender(sender, deviceRepo, notifRepo)
	if err := svc.SendRoundAnnounced("payer-1", "round-1"); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if len(sender.calls) != 0 {
		t.Errorf("expected 0 FCM calls, got %d", len(sender.calls))
	}
}

func TestFCMService_SendRoundAnnounced_WithToken(t *testing.T) {
	sender := &mockFCMSender{}
	deviceRepo := newMockDeviceTokenRepo()
	deviceRepo.tokensByUser["payer-1"] = []string{"fcm-token-abc"}
	notifRepo := &mockNotifRepo{}

	svc := notification.NewFCMNotificationServiceWithSender(sender, deviceRepo, notifRepo)
	if err := svc.SendRoundAnnounced("payer-1", "round-1"); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if len(sender.calls) != 1 {
		t.Fatalf("expected 1 FCM call, got %d", len(sender.calls))
	}
	if len(sender.calls[0].Tokens) != 1 || sender.calls[0].Tokens[0] != "fcm-token-abc" {
		t.Errorf("unexpected tokens: %v", sender.calls[0].Tokens)
	}
	if len(notifRepo.created) != 1 {
		t.Errorf("expected 1 notification log entry, got %d", len(notifRepo.created))
	}
	if notifRepo.created[0].Type != domain.NotifRoundAnnounced {
		t.Errorf("expected type %s, got %s", domain.NotifRoundAnnounced, notifRepo.created[0].Type)
	}
}

func TestFCMService_SendParticipationOpen_MultipleUsers(t *testing.T) {
	sender := &mockFCMSender{}
	deviceRepo := newMockDeviceTokenRepo()
	deviceRepo.tokensByUser["user-1"] = []string{"token-1"}
	deviceRepo.tokensByUser["user-2"] = []string{"token-2", "token-3"}
	notifRepo := &mockNotifRepo{}

	svc := notification.NewFCMNotificationServiceWithSender(sender, deviceRepo, notifRepo)
	if err := svc.SendParticipationOpen([]string{"user-1", "user-2"}, "round-1"); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if len(sender.calls) != 1 {
		t.Fatalf("expected 1 FCM call, got %d", len(sender.calls))
	}
	if len(sender.calls[0].Tokens) != 3 {
		t.Errorf("expected 3 tokens in multicast, got %d", len(sender.calls[0].Tokens))
	}
	if len(notifRepo.created) != 2 {
		t.Errorf("expected 2 notification log entries (one per user), got %d", len(notifRepo.created))
	}
}

// TestFCMService_FailedResponse_NoCrash verifies fire-and-forget: a failed FCM
// response does not cause SendRoundAnnounced to return an error.
// Note: real Firebase invalid-token cleanup (messaging.IsRegistrationTokenNotRegistered)
// requires an actual Firebase error type and is verified manually/integration only.
func TestFCMService_FailedResponse_NoCrash(t *testing.T) {
	sender := &mockFCMSender{
		resp: &messaging.BatchResponse{
			FailureCount: 1,
			Responses: []*messaging.SendResponse{
				{Success: false, Error: fmt.Errorf("some-fcm-error")},
			},
		},
	}
	deviceRepo := newMockDeviceTokenRepo()
	deviceRepo.tokensByUser["payer-1"] = []string{"some-token"}
	notifRepo := &mockNotifRepo{}

	svc := notification.NewFCMNotificationServiceWithSender(sender, deviceRepo, notifRepo)
	if err := svc.SendRoundAnnounced("payer-1", "round-1"); err != nil {
		t.Errorf("expected nil (fire-and-forget), got: %v", err)
	}
}
