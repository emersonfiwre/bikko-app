package storage

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"bikko-app/config"
	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

const MaxUploadBytes int64 = 10 << 20

type PreSignedURLResponse struct {
	UploadURL string `json:"upload_url"`
	Key       string `json:"key"`
	PublicURL string `json:"public_url"`
	FileKey   string `json:"file_key,omitempty"`
}

type StorageService struct {
	s3PresignClient *s3.PresignClient
	bucketName      string
	publicBaseURL   string
	localDir        string
	signingKey      []byte
}

func NewStorageService(cfg *config.Config) (*StorageService, error) {
	if cfg.LocalStorage {
		if cfg.Env != "development" || len(cfg.JWTSecret) < 32 {
			return nil, fmt.Errorf("local storage requires explicit development configuration")
		}
		dir, err := filepath.Abs(cfg.LocalStorageDir)
		if err != nil {
			return nil, err
		}
		if err = os.MkdirAll(dir, 0700); err != nil {
			return nil, fmt.Errorf("local storage: %w", err)
		}
		return &StorageService{localDir: dir, signingKey: []byte(cfg.JWTSecret)}, nil
	}
	for _, v := range []string{cfg.R2AccountID, cfg.R2AccessKey, cfg.R2SecretKey} {
		if v == "" || strings.HasPrefix(v, "mock_") {
			return nil, fmt.Errorf("R2 credentials missing; mock upload fallback is disabled")
		}
	}
	resolver := aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...interface{}) (aws.Endpoint, error) {
		return aws.Endpoint{URL: fmt.Sprintf("https://%s.r2.cloudflarestorage.com", cfg.R2AccountID)}, nil
	})
	awsCfg, err := awsconfig.LoadDefaultConfig(context.Background(), awsconfig.WithEndpointResolverWithOptions(resolver), awsconfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(cfg.R2AccessKey, cfg.R2SecretKey, "")), awsconfig.WithRegion("auto"))
	if err != nil {
		return nil, fmt.Errorf("R2 initialization: %w", err)
	}
	return &StorageService{s3PresignClient: s3.NewPresignClient(s3.NewFromConfig(awsCfg)), bucketName: cfg.R2BucketName, publicBaseURL: strings.TrimRight(cfg.R2PublicURL, "/")}, nil
}

func validKey(key string) bool {
	if len(key) > 240 || strings.Contains(key, "\\") || strings.HasPrefix(key, "/") || path.Clean(key) != key {
		return false
	}
	parts := strings.Split(key, "/")
	if len(parts) != 3 || (parts[0] != "quotes" && parts[0] != "profiles" && parts[0] != "services") {
		return false
	}
	for _, part := range parts {
		if part == "" || part == "." || part == ".." {
			return false
		}
	}
	return true
}

func (s *StorageService) Local() bool { return s != nil && s.localDir != "" }
func (s *StorageService) signature(method, key, mime, expires string) string {
	h := hmac.New(sha256.New, s.signingKey)
	fmt.Fprintf(h, "%s\n%s\n%s\n%s", method, key, mime, expires)
	return hex.EncodeToString(h.Sum(nil))
}
func (s *StorageService) localURL(base, method, key, mime string, lifetime time.Duration) string {
	exp := strconv.FormatInt(time.Now().Add(lifetime).Unix(), 10)
	q := url.Values{"expires": {exp}, "content_type": {mime}, "signature": {s.signature(method, key, mime, exp)}}
	return strings.TrimRight(base, "/") + "/api/v1/local-storage/" + key + "?" + q.Encode()
}

func (s *StorageService) GenerateUploadURL(ctx context.Context, key, mime, base string, size int64) (*PreSignedURLResponse, error) {
	if size <= 0 || size > MaxUploadBytes {
		return nil, fmt.Errorf("image size must be between 1 byte and 10 MiB")
	}
	if s == nil || !validKey(key) {
		return nil, fmt.Errorf("invalid storage key or unavailable storage")
	}
	if mime != "image/jpeg" && mime != "image/png" && mime != "image/webp" {
		return nil, fmt.Errorf("unsupported image type")
	}
	if s.Local() {
		u, err := url.Parse(base)
		if err != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") {
			return nil, fmt.Errorf("invalid local API origin")
		}
		return &PreSignedURLResponse{UploadURL: s.localURL(base, http.MethodPut, key, mime, 15*time.Minute), Key: key, FileKey: key, PublicURL: s.localURL(base, http.MethodGet, key, mime, 7*24*time.Hour)}, nil
	}
	if s.s3PresignClient == nil {
		return nil, fmt.Errorf("R2 unavailable")
	}
	req, err := s.s3PresignClient.PresignPutObject(ctx, &s3.PutObjectInput{Bucket: aws.String(s.bucketName), Key: aws.String(key), ContentType: aws.String(mime), ContentLength: aws.Int64(size)}, func(o *s3.PresignOptions) { o.Expires = 15 * time.Minute })
	if err != nil {
		return nil, fmt.Errorf("R2 presign PUT: %w", err)
	}
	publicURL := s.publicBaseURL + "/" + key
	if strings.HasPrefix(key, "quotes/") {
		read, err := s.s3PresignClient.PresignGetObject(ctx, &s3.GetObjectInput{Bucket: aws.String(s.bucketName), Key: aws.String(key)}, func(o *s3.PresignOptions) { o.Expires = 7 * 24 * time.Hour })
		if err != nil {
			return nil, fmt.Errorf("R2 presign GET: %w", err)
		}
		publicURL = read.URL
	}
	return &PreSignedURLResponse{UploadURL: req.URL, Key: key, FileKey: key, PublicURL: publicURL}, nil
}

// LocalPath authorizes both upload and read capabilities and confines files to the configured root.
func (s *StorageService) LocalPath(method, key, mime, expires, signature string) (string, error) {
	if !s.Local() || !validKey(key) {
		return "", fmt.Errorf("local object unavailable")
	}
	exp, err := strconv.ParseInt(expires, 10, 64)
	if err != nil || time.Now().Unix() > exp {
		return "", fmt.Errorf("expired media capability")
	}
	expected := s.signature(method, key, mime, expires)
	if !hmac.Equal([]byte(expected), []byte(signature)) {
		return "", fmt.Errorf("invalid media capability")
	}
	return filepath.Join(s.localDir, filepath.FromSlash(key)), nil
}
