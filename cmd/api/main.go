package main

import (
	"log"

	"bikko-app/config"
	"bikko-app/internal/controller"
	"bikko-app/internal/infrastructure/cache"
	infraCtrl "bikko-app/internal/infrastructure/http/controller"
	"bikko-app/internal/infrastructure/http/middleware"
	"bikko-app/internal/infrastructure/persistence/postgres"
	"bikko-app/internal/infrastructure/security"
	"bikko-app/internal/infrastructure/storage"
	"bikko-app/internal/repository"
	"bikko-app/internal/router"
	"bikko-app/internal/service"
	"bikko-app/internal/usecase"
)

func main() {
	cfg := config.LoadConfig()

	// 1. Initialize PostgreSQL connection pool (pgxpool)
	pgPool, err := postgres.NewPostgresPool(cfg)
	if err != nil {
		log.Printf("Aviso: Não foi possível conectar ao PostgreSQL: %v. Operando com fallbacks.\n", err)
	} else {
		defer pgPool.Close()
	}

	// 2. Initialize Redis service (go-redis/v9)
	redisSvc, err := cache.NewRedisService(cfg)
	if err != nil {
		log.Printf("Aviso: Não foi possível conectar ao Redis: %v. Operando sem cache Redis.\n", err)
	}

	// 3. Initialize Cloudflare R2 Storage service (AWS SDK v2)
	storageSvc, err := storage.NewStorageService(cfg)
	if err != nil {
		log.Printf("Aviso: Não foi possível inicializar o Cloudflare R2 Storage: %v\n", err)
	}

	// 4. Repositories
	var categoryRepo = postgres.NewCategoryRepository(pgPool)
	var serviceRepo = postgres.NewServiceRepository(pgPool)
	var reviewRepo = postgres.NewReviewRepository(pgPool)
	var bikkerRepo = postgres.NewBikkerRepository(pgPool)
	var userRepo = postgres.NewUserRepository(pgPool)

	// Early instantiation for Solicitations (tracks active orders for Profile)
	solicitationCtrl := controller.NewSolicitationController(serviceRepo)

	// Legacy Mocks for Auth & Profile orders (mock orders still needed temporarily)
	profileMockRepo := repository.NewMockProfileRepository(solicitationCtrl)

	// Security
	jwtService := security.NewJWTService(cfg.JWTSecret)
	authMiddleware := middleware.AuthMiddleware(jwtService)

	// 5. UseCases & Services
	authSvc := service.NewAuthService(userRepo, jwtService)
	profileSvc := service.NewProfileService(userRepo, profileMockRepo)

	homeUC := usecase.NewHomeUseCase(categoryRepo, serviceRepo, bikkerRepo, redisSvc)
	searchUC := usecase.NewSearchUseCase(categoryRepo, serviceRepo)
	ratingUC := usecase.NewRatingUseCase(reviewRepo)
	storageUC := usecase.NewStorageUseCase(storageSvc)

	// 6. Controllers
	healthCtrl := controller.NewHealthController()
	authCtrl := controller.NewAuthController(authSvc)
	profileCtrl := controller.NewProfileController(profileSvc)
	bootstrapCtrl := controller.NewBootstrapController(redisSvc)
	bikkerCtrl := controller.NewBikkerController(bikkerRepo)

	homeCtrl := infraCtrl.NewHomeController(homeUC)
	searchCtrl := infraCtrl.NewSearchController(searchUC)
	reviewCtrl := infraCtrl.NewReviewController(ratingUC)
	storageCtrl := infraCtrl.NewStorageController(storageUC)

	// 7. Router
	r := router.SetupRouter(
		healthCtrl,
		authCtrl,
		profileCtrl,
		bootstrapCtrl,
		solicitationCtrl,
		bikkerCtrl,
		homeCtrl,
		reviewCtrl,
		storageCtrl,
		searchCtrl,
		authMiddleware,
	)

	log.Printf("Servidor Golang rodando no ambiente [%s] na porta :%s\n", cfg.Env, cfg.Port)
	if err := r.Run(":" + cfg.Port); err != nil {
		log.Fatalf("Erro ao iniciar o servidor: %v", err)
	}
}

