package controller

import (
	"bikko-app/internal/domain"
	"bikko-app/internal/infrastructure/http/middleware"
	"bikko-app/internal/infrastructure/storage"
	"bikko-app/internal/usecase"
	"errors"
	"github.com/gin-gonic/gin"
	"io"
	"net/http"
	"os"
	"path/filepath"
)

type StorageController struct{ storageUC *usecase.StorageUseCase }

func NewStorageController(uc *usecase.StorageUseCase) *StorageController {
	return &StorageController{storageUC: uc}
}
func (ctrl *StorageController) GenerateUploadURL(c *gin.Context) {
	actor := middleware.GetUserID(c)
	if actor == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "UNAUTHORIZED"})
		return
	}
	var req usecase.UploadURLRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "INVALID_INPUT"})
		return
	}
	scheme := "http"
	if c.Request.TLS != nil {
		scheme = "https"
	}
	res, err := ctrl.storageUC.GenerateUploadURL(c.Request.Context(), actor, scheme+"://"+c.Request.Host, req)
	if err != nil {
		status := http.StatusServiceUnavailable
		if errors.Is(err, domain.ErrInvalidInput) {
			status = http.StatusBadRequest
		}
		c.JSON(status, gin.H{"error": http.StatusText(status)})
		return
	}
	c.JSON(http.StatusOK, res)
}

// LocalObject receives actual bytes, enforcing the signed MIME and a hard size bound.
// This route exists only for explicitly enabled local development storage.
func (ctrl *StorageController) LocalObject(c *gin.Context) {
	s := ctrl.storageUC.Storage()
	key := c.Param("key")
	if len(key) > 0 && key[0] == '/' {
		key = key[1:]
	}
	mime := c.Query("content_type")
	filename, err := s.LocalPath(c.Request.Method, key, mime, c.Query("expires"), c.Query("signature"))
	if err != nil {
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "INVALID_MEDIA_CAPABILITY"})
		return
	}
	if c.Request.Method == http.MethodGet {
		c.Header("Cache-Control", "private, max-age=300")
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("Content-Type", mime)
		c.File(filename)
		return
	}
	if c.GetHeader("Content-Type") != mime {
		c.AbortWithStatus(http.StatusUnsupportedMediaType)
		return
	}
	body := http.MaxBytesReader(c.Writer, c.Request.Body, storage.MaxUploadBytes)
	data, err := io.ReadAll(body)
	if err != nil {
		c.AbortWithStatus(http.StatusRequestEntityTooLarge)
		return
	}
	if len(data) == 0 || http.DetectContentType(data) != mime {
		c.AbortWithStatus(http.StatusUnsupportedMediaType)
		return
	}
	if err = os.MkdirAll(filepath.Dir(filename), 0700); err != nil {
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}
	tmp, err := os.CreateTemp(filepath.Dir(filename), ".upload-")
	if err != nil {
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}
	defer os.Remove(tmp.Name())
	if _, err = tmp.Write(data); err != nil {
		tmp.Close()
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}
	if err = tmp.Close(); err == nil {
		err = os.Rename(tmp.Name(), filename)
	}
	if err != nil {
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}
	c.Status(http.StatusNoContent)
}

func (ctrl *StorageController) LocalEnabled() bool {
	return ctrl != nil && ctrl.storageUC != nil && ctrl.storageUC.Storage().Local()
}
