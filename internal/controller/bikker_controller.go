package controller

import (
	"net/http"

	"bikko-app/internal/domain"

	"github.com/gin-gonic/gin"
)

type BikkerController struct {
	bikkerRepo domain.BikkerRepository
}

func NewBikkerController(bikkerRepo domain.BikkerRepository) *BikkerController {
	return &BikkerController{bikkerRepo: bikkerRepo}
}

func (ctrl *BikkerController) GetBikkerByID(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id é obrigatório"})
		return
	}

	bikker, err := ctrl.bikkerRepo.GetByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Bikker não encontrado"})
		return
	}

	c.JSON(http.StatusOK, bikker)
}

func (ctrl *BikkerController) GetBikkers(c *gin.Context) {
	bikkers, err := ctrl.bikkerRepo.GetAll(c.Request.Context(), 20)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, bikkers)
}
