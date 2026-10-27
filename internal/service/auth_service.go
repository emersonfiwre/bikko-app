package service

import (
	"context"
	"errors"
	"fmt"
	"math"
	"os"
	"strings"

	"bikko-app/internal/domain"
	"bikko-app/internal/infrastructure/security"
	"bikko-app/internal/model"
	firebaseAuth "firebase.google.com/go/v4/auth"
)

var (
	ErrUnconfirmedEmail              = errors.New("unconfirmed_email")
	ErrInvalidCredentials            = errors.New("invalid_credentials")
	ErrUserNotFound                  = errors.New("user_not_found")
	ErrInvalidFirebaseToken          = errors.New("invalid_firebase_token")
	ErrUserAlreadyExists             = errors.New("user_already_exists")
	ErrClientNeedsBikkerRegistration = errors.New("client_needs_bikker_registration")
)

type AuthService interface {
	CheckPhone(context.Context, model.CheckPhoneRequest) (*model.CheckPhoneResponse, error)
	Register(context.Context, model.RegisterRequest) (*model.AuthResponse, error)
	Login(context.Context, model.LoginRequest) (*model.AuthResponse, error)
	VerifyIDToken(context.Context, string) (*firebaseAuth.Token, error)
}

type FirebaseTokenVerifier interface {
	VerifyIDToken(context.Context, string) (*firebaseAuth.Token, error)
}
type authService struct {
	repo              domain.UserRepository
	jwtService        security.JWTService
	firebaseAuth      FirebaseTokenVerifier
	localTestAccounts bool
}
type AuthServiceOption func(*authService)

// WithLocalTestAccounts enables only the two explicit local fixtures in development.
// It never permits mock bearer tokens or affects normal Firebase identities.
func WithLocalTestAccounts(enabled bool) AuthServiceOption {
	return func(s *authService) {
		s.localTestAccounts = enabled && os.Getenv("ENV") == "development" && os.Getenv("GIN_MODE") != "release"
	}
}

func NewAuthService(repo domain.UserRepository, jwtService security.JWTService, fbAuth FirebaseTokenVerifier, opts ...AuthServiceOption) AuthService {
	if client, ok := fbAuth.(*firebaseAuth.Client); ok && client == nil {
		fbAuth = nil
	}
	s := &authService{repo: repo, jwtService: jwtService, firebaseAuth: fbAuth}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

func (s *authService) VerifyIDToken(ctx context.Context, idToken string) (*firebaseAuth.Token, error) {
	if s.localTestAccounts {
		for _, phone := range []string{"11999990000", "11988880000"} {
			if idToken == "local_test_"+phone {
				return &firebaseAuth.Token{UID: "local_test_" + phone, Claims: map[string]interface{}{"phone_number": "+55" + phone}}, nil
			}
		}
	}
	if strings.TrimSpace(idToken) == "" || s.firebaseAuth == nil {
		return nil, ErrInvalidFirebaseToken
	}
	token, err := s.firebaseAuth.VerifyIDToken(ctx, idToken)
	if err != nil || token == nil {
		return nil, ErrInvalidFirebaseToken
	}
	phone, ok := token.Claims["phone_number"].(string)
	if !ok {
		return nil, ErrInvalidFirebaseToken
	}
	if _, err := domain.NormalizePhone(phone); err != nil {
		return nil, ErrInvalidFirebaseToken
	}
	return token, nil
}

func (s *authService) verifiedPhone(ctx context.Context, idToken, requestedPhone string) (string, error) {
	token, err := s.VerifyIDToken(ctx, idToken)
	if err != nil {
		return "", ErrInvalidFirebaseToken
	}
	phone, err := domain.NormalizePhone(token.Claims["phone_number"].(string))
	if err != nil {
		return "", ErrInvalidFirebaseToken
	}
	if requestedPhone != "" {
		requested, err := domain.NormalizePhone(requestedPhone)
		if err != nil || requested != phone {
			return "", ErrInvalidCredentials
		}
	}
	return phone, nil
}

func (s *authService) CheckPhone(ctx context.Context, req model.CheckPhoneRequest) (*model.CheckPhoneResponse, error) {
	phone, err := domain.NormalizePhone(req.Phone)
	if err != nil {
		return nil, ErrInvalidCredentials
	}
	user, err := s.repo.GetByPhone(ctx, phone)
	if errors.Is(err, domain.ErrNotFound) {
		return nil, ErrUserNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("auth: find phone: %w", err)
	}
	if user == nil {
		return nil, ErrUserNotFound
	}
	response := &model.CheckPhoneResponse{Exists: true, IsBikker: user.IsBikker, Message: "Número cadastrado"}
	if strings.EqualFold(req.AppType, "bikker") && !user.IsBikker {
		return response, ErrClientNeedsBikkerRegistration
	}
	return response, nil
}

func registrationBikker(req model.RegisterRequest) (*domain.Bikker, error) {
	if req.Latitude == nil || req.Longitude == nil || math.IsNaN(*req.Latitude) || math.IsNaN(*req.Longitude) || math.IsInf(*req.Latitude, 0) || math.IsInf(*req.Longitude, 0) || *req.Latitude < -90 || *req.Latitude > 90 || *req.Longitude < -180 || *req.Longitude > 180 {
		return nil, fmt.Errorf("localização profissional válida é obrigatória: %w", domain.ErrInvalidInput)
	}
	profession := strings.TrimSpace(req.Profession)
	if profession == "" {
		return nil, fmt.Errorf("a profissão é obrigatória: %w", domain.ErrInvalidInput)
	}
	return &domain.Bikker{Profession: profession, ExperienceYears: strings.TrimSpace(req.ExperienceYears), Description: strings.TrimSpace(req.Description), LocationName: strings.TrimSpace(req.LocationName), Latitude: *req.Latitude, Longitude: *req.Longitude, IsActive: true}, nil
}

func (s *authService) response(user *domain.User, message string) (*model.AuthResponse, error) {
	token, err := s.jwtService.GenerateToken(user.ID)
	if err != nil {
		return nil, fmt.Errorf("auth: generate session: %w", err)
	}
	return &model.AuthResponse{Message: message, Token: token, User: model.User{ID: user.ID, Name: user.FullName, Email: user.Email, Phone: user.Phone, IsBikker: user.IsBikker}}, nil
}

func (s *authService) Register(ctx context.Context, req model.RegisterRequest) (*model.AuthResponse, error) {
	phone, err := s.verifiedPhone(ctx, req.IDToken, req.Phone)
	if err != nil {
		return nil, err
	}
	name := strings.TrimSpace(req.FullName)
	if name == "" {
		name = strings.TrimSpace(req.Name)
	}
	if name == "" {
		return nil, fmt.Errorf("o nome é obrigatório: %w", domain.ErrInvalidInput)
	}
	isBikker := req.IsBikker || strings.TrimSpace(req.Profession) != ""
	var bikker *domain.Bikker
	if isBikker {
		bikker, err = registrationBikker(req)
		if err != nil {
			return nil, err
		}
	}
	existing, err := s.repo.GetByPhone(ctx, phone)
	if err != nil && !errors.Is(err, domain.ErrNotFound) {
		return nil, fmt.Errorf("auth: find phone: %w", err)
	}
	if existing != nil {
		if existing.IsBikker || !isBikker {
			return nil, ErrUserAlreadyExists
		}
		upgraded, err := s.repo.UpgradeToBikker(ctx, existing.ID, bikker)
		if err != nil {
			return nil, fmt.Errorf("auth: upgrade provider: %w", err)
		}
		return s.response(upgraded, "Cadastro de Bikker vinculado com sucesso")
	}
	if req.Email != "" {
		byEmail, err := s.repo.GetByEmail(ctx, strings.TrimSpace(req.Email))
		if err != nil && !errors.Is(err, domain.ErrNotFound) {
			return nil, fmt.Errorf("auth: find email: %w", err)
		}
		if byEmail != nil {
			return nil, ErrUserAlreadyExists
		}
	}
	user, err := s.repo.CreateUser(ctx, &domain.User{FullName: name, Email: strings.TrimSpace(req.Email), Phone: phone, CPF: strings.TrimSpace(req.CPF), IsBikker: isBikker, Bikker: bikker})
	if err != nil {
		if errors.Is(err, domain.ErrConflict) {
			return nil, ErrUserAlreadyExists
		}
		return nil, fmt.Errorf("auth: register: %w", err)
	}
	return s.response(user, "Usuário registrado com sucesso")
}

func (s *authService) Login(ctx context.Context, req model.LoginRequest) (*model.AuthResponse, error) {
	phone, err := s.verifiedPhone(ctx, req.IDToken, req.Phone)
	if err != nil {
		return nil, ErrInvalidCredentials
	}
	user, err := s.repo.GetByPhone(ctx, phone)
	if errors.Is(err, domain.ErrNotFound) {
		return nil, ErrUserNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("auth: login: %w", err)
	}
	if user == nil {
		return nil, ErrUserNotFound
	}
	if strings.EqualFold(req.AppType, "bikker") && !user.IsBikker {
		return nil, ErrClientNeedsBikkerRegistration
	}
	return s.response(user, "Login realizado com sucesso")
}
