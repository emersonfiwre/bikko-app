// Package application composes the modular monolith shared by both API entrypoints.
package application

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"bikko-app/config"
	"bikko-app/internal/controller"
	"bikko-app/internal/infrastructure/cache"
	infraCtrl "bikko-app/internal/infrastructure/http/controller"
	"bikko-app/internal/infrastructure/http/middleware"
	"bikko-app/internal/infrastructure/persistence/postgres"
	"bikko-app/internal/infrastructure/push"
	"bikko-app/internal/infrastructure/security"
	"bikko-app/internal/infrastructure/storage"
	"bikko-app/internal/repository"
	"bikko-app/internal/router"
	"bikko-app/internal/service"
	"bikko-app/internal/usecase"
	"bikko-app/migrations"
	firebase "firebase.google.com/go/v4"
	firebaseAuth "firebase.google.com/go/v4/auth"
	"github.com/gin-gonic/gin"
	"google.golang.org/api/option"
)

func Run(name string) error {
	middleware.InitStructuredLogger()
	cfg := config.LoadConfig()
	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("configuration: %w", err)
	}
	if cfg.Env == "production" {
		gin.SetMode(gin.ReleaseMode)
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	pool, err := postgres.NewPostgresPool(cfg)
	if err != nil {
		return fmt.Errorf("required PostgreSQL: %w", err)
	}
	defer pool.Close()
	migrationCtx, cancel := context.WithTimeout(ctx, 60*time.Second)
	err = migrations.Apply(migrationCtx, pool)
	cancel()
	if err != nil {
		return err
	}
	if cfg.LocalTestAccounts {
		if err = migrations.SeedLocalAccounts(ctx, pool); err != nil {
			return err
		}
	}
	middleware.RegisterPgxPoolMetrics(pool)
	redisSvc, err := cache.NewRedisService(cfg)
	if err != nil {
		slog.Warn("Redis unavailable; reading PostgreSQL directly")
	}
	storageSvc, err := storage.NewStorageService(cfg)
	if err != nil {
		return err
	}
	users := postgres.NewUserRepository(pool)
	categories := postgres.NewCategoryRepository(pool)
	services := postgres.NewServiceRepository(pool)
	reviews := postgres.NewReviewRepository(pool)
	bikkers := postgres.NewBikkerRepository(pool)
	solicitations := postgres.NewSolicitationPostgresRepository(pool)
	var fbApp *firebase.App
	var fbAuth *firebaseAuth.Client
	if cfg.FirebaseProjectID != "" {
		opts := []option.ClientOption{}
		if cfg.FirebaseCredentialsFile != "" {
			opts = append(opts, option.WithCredentialsFile(cfg.FirebaseCredentialsFile))
		}
		fbApp, err = firebase.NewApp(ctx, &firebase.Config{ProjectID: cfg.FirebaseProjectID}, opts...)
		if err == nil {
			fbAuth, err = fbApp.Auth(ctx)
		}
		if err != nil {
			return fmt.Errorf("Firebase initialization: %w", err)
		}
	}
	if fbAuth == nil && !cfg.LocalTestAccounts {
		return fmt.Errorf("Firebase authentication is required")
	}
	jwt := security.NewJWTService(cfg.JWTSecret)
	var verifier service.FirebaseTokenVerifier
	if fbAuth != nil {
		verifier = fbAuth
	}
	auth := service.NewAuthService(users, jwt, verifier, service.WithLocalTestAccounts(cfg.LocalTestAccounts))
	profile := service.NewProfileService(users, repository.NewProfileRepository(solicitations), service.WithAuthAndJWT(auth, jwt))
	solicitationCtrl := controller.NewSolicitationController(services)
	solicitationCtrl.SetPostgresRepository(solicitations)
	storageCtrl := infraCtrl.NewStorageController(usecase.NewStorageUseCase(storageSvc))
	r := router.SetupRouter(
		controller.NewHealthController(pool.Ping), controller.NewAuthController(auth), controller.NewProfileController(profile), controller.NewUserController(users), controller.NewBootstrapController(redisSvc), solicitationCtrl, controller.NewBikkerController(bikkers),
		infraCtrl.NewHomeController(usecase.NewHomeUseCase(categories, services, bikkers, redisSvc)), infraCtrl.NewReviewController(usecase.NewRatingUseCase(reviews)), storageCtrl, infraCtrl.NewSearchController(usecase.NewSearchUseCase(categories, services)), infraCtrl.NewProviderControllerWithDeps(solicitations, services, bikkers), middleware.AuthMiddleware(jwt, users),
	)
	// Firebase-less local testing preserves outbox events for inspection/replay.
	workerDone := make(chan struct{})
	if fbApp != nil {
		go func() { defer close(workerDone); push.RunOutbox(ctx, pool, push.NewFirebasePushService(fbApp, users)) }()
	} else {
		close(workerDone)
	}
	server := &http.Server{Addr: ":" + cfg.Port, Handler: r, ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 20 * time.Second, WriteTimeout: 30 * time.Second, IdleTimeout: 60 * time.Second, MaxHeaderBytes: 16 << 10}
	failures := make(chan error, 1)
	go func() {
		slog.Info("API listening", slog.String("name", name), slog.String("port", cfg.Port), slog.String("env", cfg.Env))
		failures <- server.ListenAndServe()
	}()
	select {
	case <-ctx.Done():
	case err = <-failures:
		if !errors.Is(err, http.ErrServerClosed) {
			stop()
			<-workerDone
			return err
		}
	}
	stop()
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer shutdownCancel()
	err = server.Shutdown(shutdownCtx)
	<-workerDone
	return err
}
