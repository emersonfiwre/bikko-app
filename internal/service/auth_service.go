package service

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	firebaseAuth "firebase.google.com/go/v4/auth"

	"bikko-app/internal/domain"
	"bikko-app/internal/infrastructure/security"
	"bikko-app/internal/model"
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
	CheckPhone(ctx context.Context, req model.CheckPhoneRequest) (*model.CheckPhoneResponse, error)
	Register(ctx context.Context, req model.RegisterRequest) (*model.AuthResponse, error)
	Login(ctx context.Context, req model.LoginRequest) (*model.AuthResponse, error)
	VerifyIDToken(ctx context.Context, idToken string) (*firebaseAuth.Token, error)
}

type authService struct {
	repo         domain.UserRepository
	jwtService   security.JWTService
	firebaseAuth *firebaseAuth.Client
}

func NewAuthService(repo domain.UserRepository, jwtService security.JWTService, fbAuth *firebaseAuth.Client) AuthService {
	return &authService{
		repo:         repo,
		jwtService:   jwtService,
		firebaseAuth: fbAuth,
	}
}

func (s *authService) VerifyIDToken(ctx context.Context, idToken string) (*firebaseAuth.Token, error) {
	// [SECURITY AUDIT] ATENÇÃO: Aceite de token "mock_" e "test_token" é mantido exclusivamente
	// para execução de testes automatizados/homologação e desenvolvimento local sem SMS real do Firebase.
	// Em ambiente de produção (Release), deve ser estritamente desabilitado/removido.
	isProduction := os.Getenv("ENV") == "production" || os.Getenv("GIN_MODE") == "release"
	if !isProduction && (strings.HasPrefix(idToken, "mock_") || idToken == "test_token") {
		return &firebaseAuth.Token{
			UID:    "mock_firebase_uid",
			Claims: map[string]interface{}{},
		}, nil
	}

	if s.firebaseAuth != nil {
		return s.firebaseAuth.VerifyIDToken(ctx, idToken)
	}

	return nil, ErrInvalidFirebaseToken
}

func extractLast11Digits(rawPhone string) string {
	var cleaned strings.Builder
	for _, r := range rawPhone {
		if r >= '0' && r <= '9' {
			cleaned.WriteRune(r)
		}
	}
	s := cleaned.String()
	if len(s) >= 11 {
		return s[len(s)-11:]
	}
	return s
}

func (s *authService) ensureTestAccountIfNeeded(ctx context.Context, rawPhone string) {
	isProduction := os.Getenv("ENV") == "production" || os.Getenv("GIN_MODE") == "release"
	if isProduction {
		return
	}

	digits := extractLast11Digits(rawPhone)
	switch digits {
	case "11999990000":
		if u, err := s.repo.GetByPhone(ctx, digits); err != nil || u == nil {
			_, _ = s.repo.CreateUser(ctx, &domain.User{
				FullName:        "Lucas Ferreira Mendes",
				Email:           "lucas.mendes@bikkofy.com.br",
				Phone:           "11999990000",
				CPF:             "345.678.901-22",
				ProfilePhotoURL: "https://images.pexels.com/photos/2379004/pexels-photo-2379004.jpeg",
				IsBikker:        false,
				Rating:          0.00,
				TotalRatings:    0,
			})
		}
	case "11988880000":
		if u, err := s.repo.GetByPhone(ctx, digits); err != nil || u == nil {
			_, _ = s.repo.CreateUser(ctx, &domain.User{
				FullName:        "Rafael Oliveira Costa",
				Email:           "rafael.costa@bikkerfy.com.br",
				Phone:           "11988880000",
				CPF:             "456.789.012-33",
				ProfilePhotoURL: "https://images.pexels.com/photos/220453/pexels-photo-220453.jpeg",
				IsBikker:        true,
				Rating:          0.00,
				TotalRatings:    0,
				Bikker: &domain.Bikker{
					Profession:      "Eletricista / Instalação e Manutenção Elétrica",
					ExperienceYears: "5_TO_10",
					Description:     "Eletricista residencial e predial em São Paulo. Especialista em quadros de distribuição, chuveiros, tomadas, iluminação LED e reparos elétricos gerais.",
					LocationName:    "Pinheiros, São Paulo - SP",
					Latitude:        -23.5614,
					Longitude:       -46.6914,
					Rating:          0.00,
					TotalReviews:    0,
					IsActive:        true,
				},
			})
		}
	case "11987654321":
		if u, err := s.repo.GetByPhone(ctx, digits); err != nil || u == nil {
			_, _ = s.repo.CreateUser(ctx, &domain.User{
				FullName: "João da Silva",
				Email:    "joao.silva@exemplo.com",
				Phone:    "11987654321",
				CPF:      "111.222.333-44",
				IsBikker: false,
			})
		}
	case "11999991001":
		if u, err := s.repo.GetByPhone(ctx, digits); err != nil || u == nil {
			_, _ = s.repo.CreateUser(ctx, &domain.User{
				FullName: "Ricardo Almeida",
				Email:    "ricardo.almeida@bikko.com.br",
				Phone:    "11999991001",
				CPF:      "222.333.444-01",
				IsBikker: true,
				Bikker: &domain.Bikker{
					Profession:      "Encanador e Eletricista Senior",
					ExperienceYears: "10 anos",
					Description:     "Especialista em manutenção residencial.",
					LocationName:    "São Paulo, SP",
					Latitude:        -23.55052,
					Longitude:       -46.63330,
					IsActive:        true,
				},
			})
		}
	}
}

func (s *authService) CheckPhone(ctx context.Context, req model.CheckPhoneRequest) (*model.CheckPhoneResponse, error) {
	phone := strings.TrimSpace(req.Phone)
	if phone == "" {
		return nil, ErrInvalidCredentials
	}

	s.ensureTestAccountIfNeeded(ctx, phone)

	user, err := s.repo.GetByPhone(ctx, phone)
	if err != nil || user == nil {
		return nil, ErrUserNotFound
	}

	if strings.EqualFold(req.AppType, "bikker") && !user.IsBikker {
		return &model.CheckPhoneResponse{
			Exists:   true,
			IsBikker: false,
			Message:  "Este número possui apenas cadastro de Cliente. Cadastre suas informações de Bikker para continuar.",
		}, ErrClientNeedsBikkerRegistration
	}

	return &model.CheckPhoneResponse{
		Exists:   true,
		IsBikker: user.IsBikker,
		Message:  "Número verificado com sucesso",
	}, nil
}

func (s *authService) Register(ctx context.Context, req model.RegisterRequest) (*model.AuthResponse, error) {
	name := req.FullName
	if name == "" {
		name = req.Name
	}
	if name == "" {
		return nil, errors.New("o nome é obrigatório")
	}

	phone := req.Phone

	// If Firebase IDToken is provided, verify phone with Firebase Admin SDK
	if req.IDToken != "" {
		token, err := s.VerifyIDToken(ctx, req.IDToken)
		if err != nil {
			return nil, fmt.Errorf("falha ao verificar autenticação de telefone: %w", err)
		}
		if tokenPhone, ok := token.Claims["phone_number"].(string); ok && tokenPhone != "" {
			phone = tokenPhone
		}
	}

	isBikker := req.IsBikker || req.Profession != ""

	if phone != "" {
		if existing, err := s.repo.GetByPhone(ctx, phone); err == nil && existing != nil {
			if !existing.IsBikker && isBikker {
				profession := strings.TrimSpace(req.Profession)
				if profession == "" {
					profession = "Prestador de Serviços"
				}
				locationName := strings.TrimSpace(req.LocationName)
				if locationName == "" {
					locationName = "São Paulo, SP"
				}
				lat := -23.55052
				lon := -46.63330
				if req.Latitude != nil {
					lat = *req.Latitude
				}
				if req.Longitude != nil {
					lon = *req.Longitude
				}
				bikkerProfile := &domain.Bikker{
					Profession:      profession,
					ExperienceYears: strings.TrimSpace(req.ExperienceYears),
					Description:     strings.TrimSpace(req.Description),
					LocationName:    locationName,
					Latitude:        lat,
					Longitude:       lon,
					IsActive:        true,
				}
				upgradedUser, err := s.repo.UpgradeToBikker(ctx, existing.ID, bikkerProfile)
				if err != nil {
					return nil, err
				}
				token, err := s.jwtService.GenerateToken(upgradedUser.ID)
				if err != nil {
					return nil, err
				}
				return &model.AuthResponse{
					Message: "Cadastro de Bikker vinculado com sucesso",
					Token:   token,
					User: model.User{
						ID:       upgradedUser.ID,
						Name:     upgradedUser.FullName,
						Email:    upgradedUser.Email,
						Phone:    upgradedUser.Phone,
						IsBikker: upgradedUser.IsBikker,
					},
				}, nil
			}
			return nil, ErrUserAlreadyExists
		}
	}

	if req.Email != "" {
		if existing, err := s.repo.GetByEmail(ctx, req.Email); err == nil && existing != nil {
			return nil, ErrUserAlreadyExists
		}
	}

	user := &domain.User{
		FullName: name,
		Email:    req.Email,
		Phone:    phone,
		CPF:      req.CPF,
		IsBikker: isBikker,
	}

	if isBikker {
		profession := strings.TrimSpace(req.Profession)
		if profession == "" {
			profession = "Prestador de Serviços"
		}
		locationName := strings.TrimSpace(req.LocationName)
		if locationName == "" {
			locationName = "São Paulo, SP"
		}
		lat := -23.55052
		lon := -46.63330
		if req.Latitude != nil {
			lat = *req.Latitude
		}
		if req.Longitude != nil {
			lon = *req.Longitude
		}
		user.Bikker = &domain.Bikker{
			Profession:      profession,
			ExperienceYears: strings.TrimSpace(req.ExperienceYears),
			Description:     strings.TrimSpace(req.Description),
			LocationName:    locationName,
			Latitude:        lat,
			Longitude:       lon,
			IsActive:        true,
		}
	}

	createdUser, err := s.repo.CreateUser(ctx, user)
	if err != nil {
		errLower := strings.ToLower(err.Error())
		if strings.Contains(errLower, "duplicate key") || strings.Contains(errLower, "23505") || strings.Contains(errLower, "unique constraint") {
			return nil, ErrUserAlreadyExists
		}
		return nil, err
	}

	token, err := s.jwtService.GenerateToken(createdUser.ID)
	if err != nil {
		return nil, err
	}

	return &model.AuthResponse{
		Message: "Usuário registrado com sucesso",
		Token:   token,
		User: model.User{
			ID:       createdUser.ID,
			Name:     createdUser.FullName,
			Email:    createdUser.Email,
			Phone:    createdUser.Phone,
			IsBikker: createdUser.IsBikker,
		},
	}, nil
}

func (s *authService) Login(ctx context.Context, req model.LoginRequest) (*model.AuthResponse, error) {
	// If Firebase IDToken is provided, perform Phone Auth verification
	if req.IDToken != "" {
		token, err := s.VerifyIDToken(ctx, req.IDToken)
		if err != nil {
			return nil, ErrInvalidCredentials
		}

		phone := req.Phone
		if tokenPhone, ok := token.Claims["phone_number"].(string); ok && tokenPhone != "" {
			phone = tokenPhone
		}

		phone = strings.TrimSpace(phone)
		if phone == "" {
			return nil, ErrInvalidCredentials
		}

		s.ensureTestAccountIfNeeded(ctx, phone)

		user, err := s.repo.GetByPhone(ctx, phone)
		if err != nil || user == nil {
			return nil, ErrUserNotFound
		}

		if strings.EqualFold(req.AppType, "bikker") && !user.IsBikker {
			return nil, ErrClientNeedsBikkerRegistration
		}

		tokenStr, err := s.jwtService.GenerateToken(user.ID)
		if err != nil {
			return nil, err
		}

		return &model.AuthResponse{
			Message: "Login realizado com sucesso",
			Token:   tokenStr,
			User: model.User{
				ID:       user.ID,
				Name:     user.FullName,
				Email:    user.Email,
				Phone:    user.Phone,
				IsBikker: user.IsBikker,
			},
		}, nil
	}

	return nil, ErrInvalidCredentials
}
