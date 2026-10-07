package push

import (
	"context"
	"fmt"
	"log"

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

	token, err := s.userRepo.GetDeviceToken(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to retrieve device token for user %s: %w", userID, err)
	}

	if token == "" {
		log.Printf("[PushService] User %s has no registered device token. Skipping push.\n", userID)
		return nil
	}

	var sender messagingSender = s.sender
	if sender == nil {
		if s.app == nil {
			log.Printf("[PushService] Firebase app not initialized. Skipping push to user %s (token: %s)\n", userID, token)
			return nil
		}
		client, err := s.app.Messaging(ctx)
		if err != nil {
			return fmt.Errorf("failed to initialize Firebase messaging client: %w", err)
		}
		sender = client
	}

	msg := &messaging.Message{
		Token: token,
		Notification: &messaging.Notification{
			Title: title,
			Body:  body,
		},
		Data: data,
	}

	resp, err := sender.Send(ctx, msg)
	if err != nil {
		return fmt.Errorf("failed to send FCM notification to user %s: %w", userID, err)
	}

	log.Printf("[PushService] Successfully sent FCM message %s to user %s\n", resp, userID)
	return nil
}
