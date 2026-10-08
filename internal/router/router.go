package router

import (
	"net/http"

	"bikko-app/internal/controller"
	infraCtrl "bikko-app/internal/infrastructure/http/controller"
	"bikko-app/internal/infrastructure/http/middleware"

	"github.com/gin-gonic/gin"
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
	r := gin.Default()

	// Initialize and apply IP Rate Limiter (5 requests per second, burst 10)
	limiter := middleware.NewIPRateLimiter(rate.Limit(5), 10)
	limiter.Cleanup()
	r.Use(middleware.RateLimitMiddleware(limiter))

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

		// Provider (Bikkerfy) Endpoints
		provider := v1.Group("/provider")
		{
			provider.GET("/feed", providerCtrl.GetFeed)
			provider.GET("/solicitations/:id", providerCtrl.GetSolicitationDetails)
			provider.POST("/solicitations/:id/proposal", providerCtrl.SubmitProposal)
			provider.POST("/solicitations/:id/counter-offer", providerCtrl.SubmitCounterOffer)
			provider.POST("/solicitations/:id/reject", providerCtrl.RejectSolicitation)
			provider.GET("/portfolio", providerCtrl.GetPortfolio)
			provider.POST("/portfolio", providerCtrl.AddPortfolioItem)
			provider.POST("/evaluations", providerCtrl.SubmitClientEvaluation)
		}

		// Auth Group
		auth := v1.Group("/auth")
		{
			auth.POST("/register", authCtrl.Register)
			auth.POST("/login", authCtrl.Login)
			
		}

		// Storage (Cloudflare R2 Pre-signed URLs)
		v1.POST("/storage/upload-url", storageCtrl.GenerateUploadURL)

		// Protected Routes (Require JWT)
		protected := v1.Group("/")
		if authMiddleware != nil {
			protected.Use(authMiddleware)
		}
		{
			// Profile Routes
			protected.GET("/profile", profileCtrl.GetProfile)
			protected.PUT("/profile", profileCtrl.UpdateProfile)
			protected.DELETE("/profile", profileCtrl.DeleteAccount)

			// User Device Token Routes
			protected.POST("/users/device-token", userCtrl.SaveDeviceToken)
			protected.DELETE("/users/device-token", userCtrl.DeleteDeviceToken)

			// Solicitations Routes (Meus Pedidos & Acompanhamento)
			protected.GET("/solicitations", solicitationCtrl.GetSolicitations)
			protected.GET("/solicitations/:id", solicitationCtrl.GetSolicitationByID)
			protected.POST("/solicitations", solicitationCtrl.CreateSolicitation)
			protected.POST("/solicitations/:id/status", solicitationCtrl.UpdateStatus)
			protected.POST("/solicitations/:id/review", reviewCtrl.CreateSolicitationReview)

			// Orders Routes
			protected.POST("/orders/:id/review", reviewCtrl.CreateOrderReview)

			// Reviews (Transactional rating cascade)
			protected.POST("/reviews", reviewCtrl.CreateReview)
		}
	}

	return r
}

