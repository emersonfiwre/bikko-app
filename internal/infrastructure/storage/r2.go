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
	PublicURL string `json:"public_url"`
	FileKey   string `json:"file_key"`
}

type StorageService struct {
	s3PresignClient *s3.PresignClient
	bucketName      string
	publicBaseURL   string
}

func NewStorageService(cfg *config.Config) (*StorageService, error) {
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
		return nil, fmt.Errorf("falha ao carregar credenciais AWS/R2: %w", err)
	}

	s3Client := s3.NewFromConfig(awsCfg)
	presignClient := s3.NewPresignClient(s3Client)

	return &StorageService{
		s3PresignClient: presignClient,
		bucketName:      cfg.R2BucketName,
		publicBaseURL:   cfg.R2PublicURL,
	}, nil
}

func (s *StorageService) GeneratePreSignedUploadURL(ctx context.Context, fileKey, contentType string) (*PreSignedURLResponse, error) {
	putObjectInput := &s3.PutObjectInput{
		Bucket:      aws.String(s.bucketName),
		Key:         aws.String(fileKey),
		ContentType: aws.String(contentType),
	}

	req, err := s.s3PresignClient.PresignPutObject(ctx, putObjectInput, func(opts *s3.PresignOptions) {
		opts.Expires = 15 * time.Minute
	})
	if err != nil {
		return nil, fmt.Errorf("falha ao gerar Pre-signed URL para R2: %w", err)
	}

	publicURL := fmt.Sprintf("%s/%s", s.publicBaseURL, fileKey)

	return &PreSignedURLResponse{
		UploadURL: req.URL,
		PublicURL: publicURL,
		FileKey:   fileKey,
	}, nil
}
