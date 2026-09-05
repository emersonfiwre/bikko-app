package controller

import (
	"net/http"
	"strings"

	"bikko-app/internal/usecase"

	"github.com/gin-gonic/gin"
)

type SearchController struct {
	searchUC *usecase.SearchUseCase
}

func NewSearchController(searchUC *usecase.SearchUseCase) *SearchController {
	return &SearchController{searchUC: searchUC}
}

func (ctrl *SearchController) Search(c *gin.Context) {
	q := strings.TrimSpace(c.Query("q"))

	res, err := ctrl.searchUC.Search(c.Request.Context(), q)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, res)
}
