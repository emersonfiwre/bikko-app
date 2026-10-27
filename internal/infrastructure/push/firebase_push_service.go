package push

import (
	"context"
	"fmt"
	"log/slog"

	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/messaging"

	"bikko-app/internal/domain"
)

type PushService interface {
	SendNotification(ctx context.Context, userID string, title, body string, data map[string]string) error
}

type messagingSender interface {
	Send(ctx context.Context, message *messaging.Message) (string, error)
}

type FirebasePushService struct {
	app      *firebase.App
	userRepo domain.UserRepository
	sender   messagingSender
}

func NewFirebasePushService(app *firebase.App, userRepo domain.UserRepository) *FirebasePushService {
	return &FirebasePushService{
		app:      app,
		userRepo: userRepo,
	}
}

func (s *FirebasePushService) SendNotification(ctx context.Context, userID string, title, body string, data map[string]string) error {
	if s.userRepo == nil {
		return fmt.Errorf("user repository is required")
	}

	tokens := []string{}
	var err error
	if devices, ok := s.userRepo.(domain.UserDeviceRepository); ok {
		tokens, err = devices.GetDeviceTokens(ctx, userID)
	} else {
		var token string
		token, err = s.userRepo.GetDeviceToken(ctx, userID)
		if token != "" {
			tokens = append(tokens, token)
		}
	}
	if err != nil {
		return fmt.Errorf("failed to retrieve device token for user %s: %w", userID, err)
	}

	if len(tokens) == 0 {
		slog.Info("user has no registered device token, skipping push", slog.String("user_id", userID))
		return nil
	}

	var sender messagingSender = s.sender
	if sender == nil {
		if s.app == nil {
			return fmt.Errorf("firebase messaging unavailable")
		}
		client, err := s.app.Messaging(ctx)
		if err != nil {
			return fmt.Errorf("failed to initialize Firebase messaging client: %w", err)
		}
		sender = client
	}

	for _, token := range tokens {
		msg := &messaging.Message{Token: token, Notification: &messaging.Notification{Title: title, Body: body}, Data: data}
		if _, err := sender.Send(ctx, msg); err != nil {
			if messaging.IsRegistrationTokenNotRegistered(err) {
				if devices, ok := s.userRepo.(domain.UserDeviceRepository); ok {
					for _, app := range []string{"client", "provider"} {
						if cleanupErr := devices.DeleteDevice(ctx, userID, app, token); cleanupErr != nil {
							return fmt.Errorf("remove expired device: %w", cleanupErr)
						}
					}
					continue
				}
			}
			return fmt.Errorf("FCM delivery failed for user %s", userID)
		}
	}
	slog.Info("sent FCM messages", slog.String("user_id", userID), slog.Int("devices", len(tokens)))
	return nil
}
