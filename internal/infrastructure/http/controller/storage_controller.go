package controller

import (
	"net/http"

	"bikko-app/internal/usecase"

	"github.com/gin-gonic/gin"
)

type StorageController struct {
	storageUC *usecase.StorageUseCase
}

func NewStorageController(storageUC *usecase.StorageUseCase) *StorageController {
	return &StorageController{storageUC: storageUC}
}

func (ctrl *StorageController) GenerateUploadURL(c *gin.Context) {
	var req usecase.UploadURLRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "corpo da requisição inválido"})
		return
	}

	res, err := ctrl.storageUC.GenerateUploadURL(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, res)
}
