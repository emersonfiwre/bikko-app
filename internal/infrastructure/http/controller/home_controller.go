package controller

import (
	"net/http"
	"strconv"

	"bikko-app/internal/usecase"

	"github.com/gin-gonic/gin"
)

type HomeController struct {
	homeUC *usecase.HomeUseCase
}

func NewHomeController(homeUC *usecase.HomeUseCase) *HomeController {
	return &HomeController{homeUC: homeUC}
}

func (ctrl *HomeController) GetHome(c *gin.Context) {
	latStr := c.Query("lat")
	lonStr := c.Query("lon")

	var lat, lon float64
	var hasLocation bool

	if latStr != "" && lonStr != "" {
		var errLat, errLon error
		lat, errLat = strconv.ParseFloat(latStr, 64)
		lon, errLon = strconv.ParseFloat(lonStr, 64)
		if errLat == nil && errLon == nil {
			hasLocation = true
		}
	}

	homeResponse, err := ctrl.homeUC.GetHomeFeed(c.Request.Context(), lat, lon, hasLocation)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, homeResponse)
}

func (ctrl *HomeController) GetCategories(c *gin.Context) {
	categories, err := ctrl.homeUC.GetCategories(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, categories)
}

func (ctrl *HomeController) GetCategoryByID(c *gin.Context) {
	id := c.Param("id")
	category, err := ctrl.homeUC.GetCategoryByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "categoria não encontrada"})
		return
	}
	c.JSON(http.StatusOK, category)
}

func (ctrl *HomeController) GetServiceByID(c *gin.Context) {
	id := c.Param("id")
	service, err := ctrl.homeUC.GetServiceByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "serviço não encontrado"})
		return
	}
	c.JSON(http.StatusOK, service)
}

func (ctrl *HomeController) GetServices(c *gin.Context) {
	categoryID := c.Query("categoryId")
	if categoryID != "" {
		services, err := ctrl.homeUC.GetServicesByCategoryID(c.Request.Context(), categoryID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, services)
		return
	}

	bikkerID := c.Query("bikkerId")
	if bikkerID != "" {
		services, err := ctrl.homeUC.GetServicesByBikkerID(c.Request.Context(), bikkerID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, services)
		return
	}

	services, err := ctrl.homeUC.GetFeaturedAdviceServices(c.Request.Context(), 20)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, services)
}
