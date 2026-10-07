package push

import (
	"context"
	"errors"
	"testing"

	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/messaging"

	"bikko-app/internal/domain"
)

type mockUserRepoForPush struct {
	token string
	err   error
}

func (m *mockUserRepoForPush) CreateUser(ctx context.Context, user *domain.User) (*domain.User, error) {
	return nil, errors.New("not implemented")
}

func (m *mockUserRepoForPush) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	return nil, errors.New("not implemented")
}

func (m *mockUserRepoForPush) GetByPhone(ctx context.Context, phone string) (*domain.User, error) {
	return nil, errors.New("not implemented")
}

func (m *mockUserRepoForPush) GetByID(ctx context.Context, id string) (*domain.User, error) {
	return nil, errors.New("not implemented")
}

func (m *mockUserRepoForPush) UpdateUser(ctx context.Context, id string, name, email, phone string) error {
	return errors.New("not implemented")
}

func (m *mockUserRepoForPush) DeleteUser(ctx context.Context, id string) error {
	return errors.New("not implemented")
}

func (m *mockUserRepoForPush) UpdateDeviceToken(ctx context.Context, id string, token *string) error {
	return errors.New("not implemented")
}

func (m *mockUserRepoForPush) GetDeviceToken(ctx context.Context, id string) (string, error) {
	if m.err != nil {
		return "", m.err
	}
	return m.token, nil
}

type mockMessagingSender struct {
	sendFunc func(ctx context.Context, message *messaging.Message) (string, error)
}

func (m *mockMessagingSender) Send(ctx context.Context, message *messaging.Message) (string, error) {
	if m.sendFunc != nil {
		return m.sendFunc(ctx, message)
	}
	return "projects/test/messages/msg_123", nil
}

func TestFirebasePushService_SendNotification(t *testing.T) {
	ctx := context.Background()

	t.Run("successfully sends notification when user has valid token", func(t *testing.T) {
		repo := &mockUserRepoForPush{token: "fcm_token_valid"}
		var capturedMessage *messaging.Message

		sender := &mockMessagingSender{
			sendFunc: func(ctx context.Context, message *messaging.Message) (string, error) {
				capturedMessage = message
				return "projects/test/messages/msg_001", nil
			},
		}

		svc := &FirebasePushService{
			userRepo: repo,
			sender:   sender,
		}

		err := svc.SendNotification(ctx, "usr_1", "Nova Contraproposta!", "Você recebeu uma nova oferta.", map[string]string{
			"solicitation_id": "sol_123",
			"type":            "counter_offer",
		})

		if err != nil {
			t.Fatalf("expected nil error, got: %v", err)
		}

		if capturedMessage == nil {
			t.Fatal("expected message to be sent, but capturedMessage was nil")
		}
		if capturedMessage.Token != "fcm_token_valid" {
			t.Errorf("expected Token 'fcm_token_valid', got '%s'", capturedMessage.Token)
		}
		if capturedMessage.Notification.Title != "Nova Contraproposta!" {
			t.Errorf("expected Title 'Nova Contraproposta!', got '%s'", capturedMessage.Notification.Title)
		}
		if capturedMessage.Notification.Body != "Você recebeu uma nova oferta." {
			t.Errorf("expected Body 'Você recebeu uma nova oferta.', got '%s'", capturedMessage.Notification.Body)
		}
		if capturedMessage.Data["solicitation_id"] != "sol_123" || capturedMessage.Data["type"] != "counter_offer" {
			t.Errorf("unexpected Data payload: %+v", capturedMessage.Data)
		}
	})

	t.Run("skips sending when user has no device token", func(t *testing.T) {
		repo := &mockUserRepoForPush{token: ""}
		called := false
		sender := &mockMessagingSender{
			sendFunc: func(ctx context.Context, message *messaging.Message) (string, error) {
				called = true
				return "", nil
			},
		}

		svc := &FirebasePushService{
			userRepo: repo,
			sender:   sender,
		}

		err := svc.SendNotification(ctx, "usr_1", "Title", "Body", nil)
		if err != nil {
			t.Fatalf("expected nil error, got: %v", err)
		}
		if called {
			t.Error("expected sender.Send NOT to be called when token is empty")
		}
	})

	t.Run("returns error when user repo fails", func(t *testing.T) {
		repo := &mockUserRepoForPush{err: errors.New("db connection failure")}
		svc := &FirebasePushService{userRepo: repo}

		err := svc.SendNotification(ctx, "usr_1", "Title", "Body", nil)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("returns error when user repo is nil", func(t *testing.T) {
		svc := &FirebasePushService{userRepo: nil}
		err := svc.SendNotification(ctx, "usr_1", "Title", "Body", nil)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("handles nil app gracefully when no custom sender is provided", func(t *testing.T) {
		repo := &mockUserRepoForPush{token: "fcm_token_valid"}
		svc := NewFirebasePushService(nil, repo)

		err := svc.SendNotification(ctx, "usr_1", "Title", "Body", nil)
		if err != nil {
			t.Fatalf("expected nil error when app is nil, got: %v", err)
		}
	})

	t.Run("returns error when FCM sender fails", func(t *testing.T) {
		repo := &mockUserRepoForPush{token: "fcm_token_valid"}
		sender := &mockMessagingSender{
			sendFunc: func(ctx context.Context, message *messaging.Message) (string, error) {
				return "", errors.New("fcm network error")
			},
		}

		svc := &FirebasePushService{
			userRepo: repo,
			sender:   sender,
		}

		err := svc.SendNotification(ctx, "usr_1", "Title", "Body", nil)
		if err == nil {
			t.Fatal("expected error from failed FCM send, got nil")
		}
	})

	t.Run("constructor NewFirebasePushService works with nil or real app", func(t *testing.T) {
		repo := &mockUserRepoForPush{}
		var app *firebase.App
		svc := NewFirebasePushService(app, repo)
		if svc == nil || svc.userRepo != repo {
			t.Fatal("unexpected NewFirebasePushService result")
		}
	})
}
