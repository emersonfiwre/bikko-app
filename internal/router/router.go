package router

import (
	"net/http"
	"strings"

	"bikko-app/internal/controller"
	infraCtrl "bikko-app/internal/infrastructure/http/controller"
	"bikko-app/internal/infrastructure/http/middleware"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"golang.org/x/time/rate"
)

func SetupRouter(
	healthCtrl *controller.HealthController,
	authCtrl *controller.AuthController,
	profileCtrl *controller.ProfileController,
	userCtrl *controller.UserController,
	bootstrapCtrl *controller.BootstrapController,
	solicitationCtrl *controller.SolicitationController,
	bikkerCtrl *controller.BikkerController,
	homeCtrl *infraCtrl.HomeController,
	reviewCtrl *infraCtrl.ReviewController,
	storageCtrl *infraCtrl.StorageController,
	searchCtrl *infraCtrl.SearchController,
	providerCtrl *infraCtrl.ProviderController,
	authMiddleware gin.HandlerFunc,
) *gin.Engine {
	r := gin.New()
	if err := r.SetTrustedProxies(nil); err != nil {
		panic(err)
	}
	r.Use(gin.Recovery())
	r.Use(func(c *gin.Context) {
		if c.Request.Body != nil && !strings.HasPrefix(c.Request.URL.Path, "/api/v1/local-storage/") {
			c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 1<<20)
		}
		c.Next()
	})
	r.Use(middleware.SecurityHeadersMiddleware())

	// Observability Middlewares (Correlation ID, Structured JSON Logging, Prometheus Metrics)
	r.Use(middleware.CorrelationIDMiddleware())
	r.Use(middleware.SlogMiddleware())
	r.Use(middleware.MetricsMiddleware())

	// Initialize and apply IP Rate Limiter (25 requests per second, burst 50)
	limiter := middleware.NewIPRateLimiter(rate.Limit(25), 50)

	r.Use(middleware.RateLimitMiddleware(limiter))

	// Prometheus Metrics Endpoint
	r.GET("/metrics", gin.WrapH(promhttp.Handler()))

	// Local development receives and persists real files through signed capabilities.
	if storageCtrl != nil && storageCtrl.LocalEnabled() {
		r.PUT("/api/v1/local-storage/*key", storageCtrl.LocalObject)
		r.GET("/api/v1/local-storage/*key", storageCtrl.LocalObject)
	}

	// Root Route
	r.GET("/", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "Hello World! Bikko Backend API (Postgres 16 + PostGIS + Redis) is running.",
			"status":  "success",
		})
	})

	// API v1 Group
	v1 := r.Group("/api/v1")
	{
		v1.GET("/health", healthCtrl.CheckHealth)
		v1.GET("/ready", healthCtrl.CheckReady)
		v1.GET("/bootstrap", bootstrapCtrl.GetBootstrap)

		// Home & Services Routes
		v1.GET("/home", homeCtrl.GetHome)
		v1.GET("/categories", homeCtrl.GetCategories)
		v1.GET("/categories/:id", homeCtrl.GetCategoryByID)
		v1.GET("/services", homeCtrl.GetServices)
		v1.GET("/services/:id", homeCtrl.GetServiceByID)

		// Bikker Profiles
		v1.GET("/bikkers", bikkerCtrl.GetBikkers)
		v1.GET("/bikkers/:id", bikkerCtrl.GetBikkerByID)

		// Search Route (Returns Services + Categories)
		v1.GET("/search", searchCtrl.Search)

		// Auth Group
		auth := v1.Group("/auth")
		{
			auth.POST("/check-phone", authCtrl.CheckPhone)
			auth.POST("/register", authCtrl.Register)
			auth.POST("/login", authCtrl.Login)
		}

		// Provider (Bikkerfy) Endpoints (JWT Protected when authMiddleware != nil)
		provider := v1.Group("/provider")
		provider.Use(requiredAuth(authMiddleware), middleware.RequireProvider())
		{
			provider.GET("/feed", providerCtrl.GetFeed)
			provider.GET("/solicitations/:id", providerCtrl.GetSolicitationDetails)
			provider.POST("/solicitations/:id/proposal", providerCtrl.SubmitProposal)
			provider.POST("/solicitations/:id/counter-offer", providerCtrl.SubmitCounterOffer)
			provider.POST("/solicitations/:id/accept", providerCtrl.AcceptCounterOffer)
			provider.POST("/solicitations/:id/accept-counter-offer", providerCtrl.AcceptCounterOffer)
			provider.POST("/solicitations/:id/reject", providerCtrl.RejectSolicitation)
			provider.GET("/portfolio", providerCtrl.GetPortfolio)
			provider.POST("/portfolio", providerCtrl.AddPortfolioItem)
			provider.PUT("/portfolio/:id", providerCtrl.UpdatePortfolioItem)
			provider.DELETE("/portfolio/:id", providerCtrl.DeletePortfolioItem)
			provider.PUT("/profile", providerCtrl.UpdateProviderProfile)
			provider.POST("/evaluations", providerCtrl.SubmitClientEvaluation)
		}

		// Protected Routes (Require JWT)
		protected := v1.Group("/")
		protected.Use(requiredAuth(authMiddleware))
		{
			// Storage (Cloudflare R2 Pre-signed URLs)
			protected.POST("/storage/upload-url", storageCtrl.GenerateUploadURL)

			// Profile Routes
			protected.GET("/profile", profileCtrl.GetProfile)
			protected.PUT("/profile", profileCtrl.UpdateProfile)
			protected.PUT("/profile/photo", profileCtrl.UpdateProfilePhoto)
			protected.DELETE("/profile", profileCtrl.DeleteAccount)

			// User Device Token Routes
			protected.POST("/users/device-token", userCtrl.SaveDeviceToken)
			protected.DELETE("/users/device-token", userCtrl.DeleteDeviceToken)

			// Solicitations Routes (Meus Pedidos & Acompanhamento)
			protected.GET("/solicitations", solicitationCtrl.GetSolicitations)
			protected.GET("/solicitations/:id", solicitationCtrl.GetSolicitationByID)
			protected.POST("/solicitations", solicitationCtrl.CreateSolicitation)
			protected.POST("/solicitations/:id/status", solicitationCtrl.UpdateStatus)
			protected.PATCH("/solicitations/:id/status", solicitationCtrl.UpdateStatus)
			protected.POST("/solicitations/:id/review", reviewCtrl.CreateSolicitationReview)

			// Orders Routes
			protected.POST("/orders/:id/review", reviewCtrl.CreateOrderReview)

			// Reviews (Transactional rating cascade)
			protected.POST("/reviews", reviewCtrl.CreateReview)
		}
	}

	return r
}

func requiredAuth(auth gin.HandlerFunc) gin.HandlerFunc {
	if auth != nil {
		return auth
	}
	return func(c *gin.Context) { c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "UNAUTHORIZED"}) }
}
