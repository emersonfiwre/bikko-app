package usecase

import (
	"context"
	"fmt"
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
	Folder      string `json:"folder" binding:"required"` // e.g. "profiles", "services", "quotes"
	ContentType string `json:"content_type" binding:"required"` // e.g. "image/jpeg", "image/png"
}

func (uc *StorageUseCase) GenerateUploadURL(ctx context.Context, folder, contentType string) (*storage.PreSignedURLResponse, error) {
	fileKey := fmt.Sprintf("%s/%s_%d", folder, uuid.New().String(), time.Now().Unix())
	return uc.storageService.GeneratePreSignedUploadURL(ctx, fileKey, contentType)
}
