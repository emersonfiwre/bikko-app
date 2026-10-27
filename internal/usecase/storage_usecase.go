package usecase

import (
	"bikko-app/internal/domain"
	"bikko-app/internal/infrastructure/storage"
	"context"
	"fmt"
	"github.com/google/uuid"
)

type StorageUseCase struct{ storageService *storage.StorageService }

func NewStorageUseCase(s *storage.StorageService) *StorageUseCase {
	return &StorageUseCase{storageService: s}
}

type UploadURLRequest struct {
	Filename    string `json:"filename"`
	FileName    string `json:"file_name"`
	Folder      string `json:"folder"`
	ContentType string `json:"content_type"`
	SizeBytes   int64  `json:"size_bytes"`
}

func (uc *StorageUseCase) GenerateUploadURL(ctx context.Context, actor, origin string, req UploadURLRequest) (*storage.PreSignedURLResponse, error) {
	if _, err := uuid.Parse(actor); err != nil {
		return nil, domain.ErrForbidden
	}
	folder := req.Folder
	if folder == "" {
		folder = "quotes"
	}
	if folder != "quotes" && folder != "profiles" && folder != "services" {
		return nil, domain.ErrInvalidInput
	}
	ext := map[string]string{"image/jpeg": ".jpg", "image/png": ".png", "image/webp": ".webp"}[req.ContentType]
	if ext == "" || req.SizeBytes <= 0 || req.SizeBytes > storage.MaxUploadBytes {
		return nil, fmt.Errorf("%w: image type or size", domain.ErrInvalidInput)
	}
	return uc.storageService.GenerateUploadURL(ctx, folder+"/"+actor+"/"+uuid.NewString()+ext, req.ContentType, origin, req.SizeBytes)
}
func (uc *StorageUseCase) Storage() *storage.StorageService { return uc.storageService }
