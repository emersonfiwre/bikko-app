package controller

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"bikko-app/config"
	"bikko-app/internal/infrastructure/storage"
	"bikko-app/internal/usecase"

	"github.com/gin-gonic/gin"
)

func TestStorageController_GenerateUploadURL_MockFallback(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cfg := &config.Config{
		R2AccountID:  "mock_r2_account",
		R2AccessKey:  "mock_r2_access_key",
		R2SecretKey:  "mock_r2_secret_key",
		R2BucketName: "bikko-media",
		R2PublicURL:  "https://pub-bikko.r2.dev",
	}

	storageService, err := storage.NewStorageService(cfg)
	if err != nil {
		t.Fatalf("esperado nil error no NewStorageService mock, obteve: %v", err)
	}

	storageUC := usecase.NewStorageUseCase(storageService)
	storageCtrl := NewStorageController(storageUC)

	r := gin.New()
	r.POST("/api/v1/storage/upload-url", storageCtrl.GenerateUploadURL)

	reqBody := []byte(`{"filename": "foto_exemplo.png", "content_type": "image/png"}`)
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/storage/upload-url", bytes.NewBuffer(reqBody))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status code esperado %d, obteve %d: %s", http.StatusOK, w.Code, w.Body.String())
	}

	var resp storage.PreSignedURLResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("falha ao deserializar resposta: %v", err)
	}

	if resp.UploadURL == "" {
		t.Errorf("upload_url não deveria ser vazio")
	}
	if resp.Key == "" {
		t.Errorf("key não deveria ser vazio")
	}
	if !strings.HasSuffix(resp.Key, ".png") {
		t.Errorf("key deveria ter extensão .png, obteve: %s", resp.Key)
	}
	if resp.PublicURL == "" {
		t.Errorf("public_url não deveria ser vazio")
	}
	if !strings.HasPrefix(resp.PublicURL, "https://pub-bikko.r2.dev/") {
		t.Errorf("public_url deveria começar com base URL, obteve: %s", resp.PublicURL)
	}
}
