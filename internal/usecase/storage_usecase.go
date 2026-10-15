package usecase

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"bikko-app/internal/infrastructure/storage"

	"github.com/google/uuid"
)

type StorageUseCase struct {
	storageService *storage.StorageService
}

func NewStorageUseCase(storageService *storage.StorageService) *StorageUseCase {
	return &StorageUseCase{storageService: storageService}
}

type UploadURLRequest struct {
	Filename    string `json:"filename"`
	FileName    string `json:"file_name"` // alias
	Folder      string `json:"folder"`    // optional e.g. "quotes", "profiles", "services"
	ContentType string `json:"content_type"`
}

func (uc *StorageUseCase) GenerateUploadURL(ctx context.Context, req UploadURLRequest) (*storage.PreSignedURLResponse, error) {
	filename := req.Filename
	if filename == "" {
		filename = req.FileName
	}
	if filename == "" {
		filename = fmt.Sprintf("upload_%d.jpg", time.Now().Unix())
	}

	contentType := req.ContentType
	if contentType == "" {
		contentType = "image/jpeg"
	}

	folder := req.Folder
	if folder == "" {
		folder = "quotes"
	}

	ext := filepath.Ext(filename)
	cleanName := strings.TrimSuffix(filepath.Base(filename), ext)
	if cleanName == "" {
		cleanName = "photo"
	}

	fileKey := fmt.Sprintf("%s/%s_%s%s", folder, uuid.New().String()[:8], cleanName, ext)
	return uc.storageService.GeneratePreSignedUploadURL(ctx, fileKey, contentType)
}
