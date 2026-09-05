package router

import (
	"net/http"

	"bikko-app/internal/controller"
	infraCtrl "bikko-app/internal/infrastructure/http/controller"

	"github.com/gin-gonic/gin"
)

func SetupRouter(
	healthCtrl *controller.HealthController,
	authCtrl *controller.AuthController,
	profileCtrl *controller.ProfileController,
	homeCtrl *infraCtrl.HomeController,
	reviewCtrl *infraCtrl.ReviewController,
	storageCtrl *infraCtrl.StorageController,
	searchCtrl *infraCtrl.SearchController,
) *gin.Engine {
	r := gin.Default()

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

		// Home Aggregated Route
		v1.GET("/home", homeCtrl.GetHome)

		// Search Route (Returns Services + Categories)
		v1.GET("/search", searchCtrl.Search)

		// Auth Group
		auth := v1.Group("/auth")
		{
			auth.POST("/register", authCtrl.Register)
			auth.POST("/login", authCtrl.Login)
		}

		v1.GET("/profile", profileCtrl.GetProfile)

		// Reviews (Transactional rating cascade)
		v1.POST("/reviews", reviewCtrl.CreateReview)

		// Storage (Cloudflare R2 Pre-signed URLs)
		v1.POST("/storage/upload-url", storageCtrl.GenerateUploadURL)
	}

	return r
}
