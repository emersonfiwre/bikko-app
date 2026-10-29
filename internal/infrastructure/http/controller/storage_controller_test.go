package controller

import (
	"bikko-app/config"
	"bikko-app/internal/infrastructure/storage"
	"bikko-app/internal/usecase"
	"bytes"
	"encoding/json"
	"github.com/gin-gonic/gin"
	"image"
	"image/png"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestStorageCannotSilentlyUseMock(t *testing.T) {
	if _, err := storage.NewStorageService(&config.Config{Env: "production"}); err == nil {
		t.Fatal("missing R2 must not be a successful fake upload")
	}
}
func TestLocalStoragePersistsImageAndRejectsTampering(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc, err := storage.NewStorageService(&config.Config{Env: "development", LocalStorage: true, LocalStorageDir: t.TempDir(), JWTSecret: "explicit-local-secret-longer-than-32-characters"})
	if err != nil {
		t.Fatal(err)
	}
	ctrl := NewStorageController(usecase.NewStorageUseCase(svc))
	r := gin.New()
	r.POST("/api/v1/storage/upload-url", func(c *gin.Context) {
		c.Set("userID", "11111111-1111-1111-1111-111111111111")
		ctrl.GenerateUploadURL(c)
	})
	r.PUT("/api/v1/local-storage/*key", ctrl.LocalObject)
	r.GET("/api/v1/local-storage/*key", ctrl.LocalObject)
	var imageBytes bytes.Buffer
	if err = png.Encode(&imageBytes, image.NewRGBA(image.Rect(0, 0, 1, 1))); err != nil {
		t.Fatal(err)
	}
	payload, _ := json.Marshal(map[string]any{"filename": "photo.png", "content_type": "image/png", "size_bytes": imageBytes.Len()})
	req := httptest.NewRequest(http.MethodPost, "http://localhost/api/v1/storage/upload-url", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("presign %d: %s", w.Code, w.Body.String())
	}
	var signed storage.PreSignedURLResponse
	if err = json.Unmarshal(w.Body.Bytes(), &signed); err != nil {
		t.Fatal(err)
	}
	req = httptest.NewRequest(http.MethodPut, signed.UploadURL, bytes.NewReader(imageBytes.Bytes()))
	req.Header.Set("Content-Type", "image/png")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != 204 {
		t.Fatalf("upload %d: %s", w.Code, w.Body.String())
	}
	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, signed.PublicURL, nil))
	if w.Code != 200 || !bytes.Equal(w.Body.Bytes(), imageBytes.Bytes()) {
		t.Fatalf("image was not persisted, status %d", w.Code)
	}
	u, _ := url.Parse(signed.UploadURL)
	q := u.Query()
	q.Set("signature", "tampered")
	u.RawQuery = q.Encode()
	req = httptest.NewRequest(http.MethodPut, u.String(), bytes.NewReader(imageBytes.Bytes()))
	req.Header.Set("Content-Type", "image/png")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != 403 {
		t.Fatalf("tampered capability accepted: %d", w.Code)
	}
}
