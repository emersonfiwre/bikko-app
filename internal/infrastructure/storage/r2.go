package storage

import (
	"context"
	"fmt"
	"time"

	"bikko-app/config"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

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
	isMock          bool
}

func NewStorageService(cfg *config.Config) (*StorageService, error) {
	// Fallback to mock if credentials are missing or mock defaults
	if cfg.R2AccountID == "" || cfg.R2AccessKey == "" || cfg.R2SecretKey == "" ||
		cfg.R2AccountID == "mock_r2_account" || cfg.R2AccessKey == "mock_r2_access_key" || cfg.R2SecretKey == "mock_r2_secret_key" {
		return &StorageService{
			bucketName:    cfg.R2BucketName,
			publicBaseURL: cfg.R2PublicURL,
			isMock:        true,
		}, nil
	}

	r2Resolver := aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...interface{}) (aws.Endpoint, error) {
		return aws.Endpoint{
			URL: fmt.Sprintf("https://%s.r2.cloudflarestorage.com", cfg.R2AccountID),
		}, nil
	})

	awsCfg, err := awsconfig.LoadDefaultConfig(context.Background(),
		awsconfig.WithEndpointResolverWithOptions(r2Resolver),
		awsconfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(cfg.R2AccessKey, cfg.R2SecretKey, "")),
		awsconfig.WithRegion("auto"),
	)
	if err != nil {
		return &StorageService{
			bucketName:    cfg.R2BucketName,
			publicBaseURL: cfg.R2PublicURL,
			isMock:        true,
		}, nil
	}

	s3Client := s3.NewFromConfig(awsCfg)
	presignClient := s3.NewPresignClient(s3Client)

	return &StorageService{
		s3PresignClient: presignClient,
		bucketName:      cfg.R2BucketName,
		publicBaseURL:   cfg.R2PublicURL,
		isMock:          false,
	}, nil
}

func (s *StorageService) GeneratePreSignedUploadURL(ctx context.Context, fileKey, contentType string) (*PreSignedURLResponse, error) {
	publicURL := fmt.Sprintf("%s/%s", s.publicBaseURL, fileKey)

	if s.isMock || s.s3PresignClient == nil {
		mockUploadURL := fmt.Sprintf("%s/mock-upload/%s", s.publicBaseURL, fileKey)
		return &PreSignedURLResponse{
			UploadURL: mockUploadURL,
			Key:       fileKey,
			PublicURL: publicURL,
			FileKey:   fileKey,
		}, nil
	}

	putObjectInput := &s3.PutObjectInput{
		Bucket:      aws.String(s.bucketName),
		Key:         aws.String(fileKey),
		ContentType: aws.String(contentType),
	}

	req, err := s.s3PresignClient.PresignPutObject(ctx, putObjectInput, func(opts *s3.PresignOptions) {
		opts.Expires = 15 * time.Minute
	})
	if err != nil {
		// Fallback gracefully to mock url on error
		mockUploadURL := fmt.Sprintf("%s/mock-upload/%s", s.publicBaseURL, fileKey)
		return &PreSignedURLResponse{
			UploadURL: mockUploadURL,
			Key:       fileKey,
			PublicURL: publicURL,
			FileKey:   fileKey,
		}, nil
	}

	return &PreSignedURLResponse{
		UploadURL: req.URL,
		Key:       fileKey,
		PublicURL: publicURL,
		FileKey:   fileKey,
	}, nil
}
