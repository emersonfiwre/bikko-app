package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"bikko-app/internal/domain"
	"bikko-app/internal/model"
)

type fakeUserRepository struct {
	users         map[string]*domain.User
	createErr     error
	getByEmailErr error
	getByIDErr    error
	updateErr     error
	deleteErr     error
	lastUpdatedID string
	lastDeletedID string
}

func (f *fakeUserRepository) CreateUser(ctx context.Context, user *domain.User, password string) (*domain.User, error) {
	if f.createErr != nil {
		return nil, f.createErr
	}
	return user, nil
}

func (f *fakeUserRepository) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	if f.getByEmailErr != nil {
		return nil, f.getByEmailErr
	}
	for _, u := range f.users {
		if u.Email == email {
			return u, nil
		}
	}
	return nil, errors.New("user not found")
}

func (f *fakeUserRepository) GetByPhone(ctx context.Context, phone string) (*domain.User, error) {
	for _, u := range f.users {
		if u.Phone == phone {
			return u, nil
		}
	}
	return nil, errors.New("user not found")
}

func (f *fakeUserRepository) GetByID(ctx context.Context, id string) (*domain.User, error) {
	if f.getByIDErr != nil {
		return nil, f.getByIDErr
	}
	if u, ok := f.users[id]; ok {
		return u, nil
	}
	return nil, errors.New("user not found")
}

func (f *fakeUserRepository) UpdateUser(ctx context.Context, id string, name, email, phone string) error {
	f.lastUpdatedID = id
	if f.updateErr != nil {
		return f.updateErr
	}
	if u, ok := f.users[id]; ok {
		u.FullName = name
		u.Email = email
		u.Phone = phone
		u.UpdatedAt = time.Now()
	}
	return nil
}

func (f *fakeUserRepository) DeleteUser(ctx context.Context, id string) error {
	f.lastDeletedID = id
	if f.deleteErr != nil {
		return f.deleteErr
	}
	delete(f.users, id)
	return nil
}

type fakeProfileRepo struct {
	profileResp *model.ProfileResponse
	profileErr  error
	deleteErr   error
}

func (f *fakeProfileRepo) GetProfileByID(ctx context.Context, id string) (*model.ProfileResponse, error) {
	if f.profileErr != nil {
		return nil, f.profileErr
	}
	return f.profileResp, nil
}

func (f *fakeProfileRepo) DeleteAccount(ctx context.Context, id string) error {
	return f.deleteErr
}

func TestProfileService_GetProfile(t *testing.T) {
	ctx := context.Background()

	t.Run("success from user repository with positive rating", func(t *testing.T) {
		userRepo := &fakeUserRepository{
			users: map[string]*domain.User{
				"usr_1": {
					ID:              "usr_1",
					FullName:        "Carlos Silva",
					Email:           "carlos@example.com",
					Phone:           "(11) 98888-7777",
					ProfilePhotoURL: "https://example.com/carlos.jpg",
					Rating:          4.8,
					TotalRatings:    25,
				},
			},
		}
		mockRepo := &fakeProfileRepo{
			profileResp: &model.ProfileResponse{
				Orders: []model.ProfileOrderItem{
					{ID: "order_1", Name: "Pintura"},
				},
			},
		}

		svc := NewProfileService(userRepo, mockRepo)
		resp, err := svc.GetProfile(ctx, "usr_1")

		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if resp == nil {
			t.Fatal("expected profile response, got nil")
		}
		if resp.ID != "usr_1" {
			t.Errorf("expected ID 'usr_1', got '%s'", resp.ID)
		}
		if resp.Name != "Carlos Silva" {
			t.Errorf("expected name 'Carlos Silva', got '%s'", resp.Name)
		}
		if resp.Email != "carlos@example.com" {
			t.Errorf("expected email 'carlos@example.com', got '%s'", resp.Email)
		}
		if resp.Phone != "(11) 98888-7777" {
			t.Errorf("expected phone '(11) 98888-7777', got '%s'", resp.Phone)
		}
		if resp.ImageProfile != "https://example.com/carlos.jpg" {
			t.Errorf("expected image 'https://example.com/carlos.jpg', got '%s'", resp.ImageProfile)
		}
		if resp.Rating != "4.8" {
			t.Errorf("expected rating '4.8', got '%s'", resp.Rating)
		}
		if resp.TotalRatings != 25 {
			t.Errorf("expected total ratings 25, got %d", resp.TotalRatings)
		}
		if len(resp.Orders) != 1 || resp.Orders[0].ID != "order_1" {
			t.Errorf("expected 1 order with ID order_1, got %+v", resp.Orders)
		}
	})

	t.Run("formats zero rating as N/A", func(t *testing.T) {
		userRepo := &fakeUserRepository{
			users: map[string]*domain.User{
				"usr_new": {
					ID:           "usr_new",
					FullName:     "Novo Usuario",
					Rating:       0.0,
					TotalRatings: 0,
				},
			},
		}
		mockRepo := &fakeProfileRepo{}

		svc := NewProfileService(userRepo, mockRepo)
		resp, err := svc.GetProfile(ctx, "usr_new")

		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if resp.Rating != "N/A" {
			t.Errorf("expected rating 'N/A', got '%s'", resp.Rating)
		}
	})

	t.Run("empty userID defaults to user_123", func(t *testing.T) {
		userRepo := &fakeUserRepository{
			users: map[string]*domain.User{
				"user_123": {
					ID:       "user_123",
					FullName: "Default User",
				},
			},
		}
		mockRepo := &fakeProfileRepo{}

		svc := NewProfileService(userRepo, mockRepo)
		resp, err := svc.GetProfile(ctx, "")

		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if resp.ID != "user_123" {
			t.Errorf("expected ID 'user_123', got '%s'", resp.ID)
		}
	})

	t.Run("fallback to mockRepo when userRepo.GetByID returns error", func(t *testing.T) {
		userRepo := &fakeUserRepository{
			getByIDErr: errors.New("database connection failed"),
		}
		mockRepo := &fakeProfileRepo{
			profileResp: &model.ProfileResponse{
				ID:   "fallback_id",
				Name: "Emerson Torres Mock",
			},
		}

		svc := NewProfileService(userRepo, mockRepo)
		resp, err := svc.GetProfile(ctx, "fallback_id")

		if err != nil {
			t.Fatalf("expected fallback to succeed, got %v", err)
		}
		if resp == nil || resp.Name != "Emerson Torres Mock" {
			t.Errorf("expected fallback profile, got %+v", resp)
		}
	})

	t.Run("returns error when both userRepo and mockRepo fail", func(t *testing.T) {
		userRepo := &fakeUserRepository{
			getByIDErr: errors.New("db error"),
		}
		mockRepo := &fakeProfileRepo{
			profileErr: errors.New("mock not found"),
		}

		svc := NewProfileService(userRepo, mockRepo)
		_, err := svc.GetProfile(ctx, "missing_user")

		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

func TestProfileService_DeleteAccount(t *testing.T) {
	ctx := context.Background()

	t.Run("error when userID is empty", func(t *testing.T) {
		svc := NewProfileService(&fakeUserRepository{}, &fakeProfileRepo{})
		err := svc.DeleteAccount(ctx, "")
		if err == nil || err.Error() != "user id required" {
			t.Errorf("expected 'user id required', got %v", err)
		}
	})

	t.Run("success when valid userID is provided", func(t *testing.T) {
		userRepo := &fakeUserRepository{
			users: map[string]*domain.User{
				"usr_1": {ID: "usr_1"},
			},
		}
		svc := NewProfileService(userRepo, &fakeProfileRepo{})
		err := svc.DeleteAccount(ctx, "usr_1")
		if err != nil {
			t.Fatalf("expected nil error, got %v", err)
		}
		if userRepo.lastDeletedID != "usr_1" {
			t.Errorf("expected lastDeletedID to be 'usr_1', got '%s'", userRepo.lastDeletedID)
		}
	})

	t.Run("returns repository error if deletion fails", func(t *testing.T) {
		userRepo := &fakeUserRepository{
			deleteErr: errors.New("delete failed"),
		}
		svc := NewProfileService(userRepo, &fakeProfileRepo{})
		err := svc.DeleteAccount(ctx, "usr_1")
		if err == nil || err.Error() != "delete failed" {
			t.Errorf("expected 'delete failed', got %v", err)
		}
	})
}

func TestProfileService_UpdateProfile(t *testing.T) {
	ctx := context.Background()

	t.Run("error when userID is empty", func(t *testing.T) {
		svc := NewProfileService(&fakeUserRepository{}, &fakeProfileRepo{})
		err := svc.UpdateProfile(ctx, "", "Name", "email@test.com", "123")
		if err == nil || err.Error() != "user id required" {
			t.Errorf("expected 'user id required', got %v", err)
		}
	})

	t.Run("success updates user fields", func(t *testing.T) {
		userRepo := &fakeUserRepository{
			users: map[string]*domain.User{
				"usr_1": {ID: "usr_1", FullName: "Old Name", Email: "old@test.com"},
			},
		}
		svc := NewProfileService(userRepo, &fakeProfileRepo{})
		err := svc.UpdateProfile(ctx, "usr_1", "New Name", "new@test.com", "(11) 91111-2222")
		if err != nil {
			t.Fatalf("expected nil error, got %v", err)
		}
		if userRepo.lastUpdatedID != "usr_1" {
			t.Errorf("expected lastUpdatedID 'usr_1', got '%s'", userRepo.lastUpdatedID)
		}
		if userRepo.users["usr_1"].FullName != "New Name" {
			t.Errorf("expected FullName 'New Name', got '%s'", userRepo.users["usr_1"].FullName)
		}
	})

	t.Run("returns repository error if update fails", func(t *testing.T) {
		userRepo := &fakeUserRepository{
			updateErr: errors.New("update conflict"),
		}
		svc := NewProfileService(userRepo, &fakeProfileRepo{})
		err := svc.UpdateProfile(ctx, "usr_1", "New Name", "new@test.com", "123")
		if err == nil || err.Error() != "update conflict" {
			t.Errorf("expected 'update conflict', got %v", err)
		}
	})
}
