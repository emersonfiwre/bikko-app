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
	latStr := c.DefaultQuery("lat", "-23.55052")
	lonStr := c.DefaultQuery("lon", "-46.63330")

	lat, err := strconv.ParseFloat(latStr, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "parâmetro 'lat' inválido"})
		return
	}

	lon, err := strconv.ParseFloat(lonStr, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "parâmetro 'lon' inválido"})
		return
	}

	homeResponse, err := ctrl.homeUC.GetHomeFeed(c.Request.Context(), lat, lon)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, homeResponse)
}
