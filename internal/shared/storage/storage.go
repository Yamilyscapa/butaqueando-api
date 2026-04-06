package storage

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/smithy-go"
)

var ErrClientNotConfigured = errors.New("storage client not configured")

type BucketConfig struct {
	Endpoint        string
	Region          string
	AccessKeyID     string
	SecretAccessKey string
	Bucket          string
}

type PresignPutObjectInput struct {
	ObjectKey     string
	ContentType   string
	ContentLength *int64
	ExpiresIn     time.Duration
}

type PresignGetObjectInput struct {
	ObjectKey string
	ExpiresIn time.Duration
}

type HeadObjectInput struct {
	ObjectKey string
}

type HeadObjectOutput struct {
	ContentType   string
	ContentLength int64
}

type Client interface {
	Bucket() string
	PresignPutObject(ctx context.Context, input PresignPutObjectInput) (string, error)
	PresignGetObject(ctx context.Context, input PresignGetObjectInput) (string, error)
	HeadObject(ctx context.Context, input HeadObjectInput) (HeadObjectOutput, error)
}

type S3Client struct {
	bucket    string
	client    *s3.Client
	presigner *s3.PresignClient
}

func NewS3Client(ctx context.Context, cfg BucketConfig) (*S3Client, error) {
	endpoint := strings.TrimSpace(cfg.Endpoint)
	region := strings.TrimSpace(cfg.Region)
	accessKeyID := strings.TrimSpace(cfg.AccessKeyID)
	secretAccessKey := strings.TrimSpace(cfg.SecretAccessKey)
	bucket := strings.TrimSpace(cfg.Bucket)

	if endpoint == "" || region == "" || accessKeyID == "" || secretAccessKey == "" || bucket == "" {
		return nil, fmt.Errorf("invalid bucket config")
	}

	awsCfg, err := awsconfig.LoadDefaultConfig(
		ctx,
		awsconfig.WithRegion(region),
		awsconfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(accessKeyID, secretAccessKey, "")),
		awsconfig.WithEndpointResolverWithOptions(
			aws.EndpointResolverWithOptionsFunc(func(service string, _ string, _ ...interface{}) (aws.Endpoint, error) {
				if service != s3.ServiceID {
					return aws.Endpoint{}, &aws.EndpointNotFoundError{}
				}

				return aws.Endpoint{URL: endpoint}, nil
			}),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("load aws config: %w", err)
	}

	s3Client := s3.NewFromConfig(awsCfg)

	return &S3Client{
		bucket:    bucket,
		client:    s3Client,
		presigner: s3.NewPresignClient(s3Client),
	}, nil
}

func (c *S3Client) Bucket() string {
	return c.bucket
}

func (c *S3Client) PresignPutObject(ctx context.Context, input PresignPutObjectInput) (string, error) {
	if c == nil || c.client == nil || c.presigner == nil {
		return "", ErrClientNotConfigured
	}

	key := strings.TrimSpace(input.ObjectKey)
	if key == "" {
		return "", fmt.Errorf("object key is required")
	}

	contentType := strings.TrimSpace(input.ContentType)
	if contentType == "" {
		return "", fmt.Errorf("content type is required")
	}

	if input.ExpiresIn <= 0 {
		return "", fmt.Errorf("expiresIn must be greater than 0")
	}

	requestInput := &s3.PutObjectInput{
		Bucket:      aws.String(c.bucket),
		Key:         aws.String(key),
		ContentType: aws.String(contentType),
	}

	if input.ContentLength != nil {
		requestInput.ContentLength = input.ContentLength
	}

	request, err := c.presigner.PresignPutObject(
		ctx,
		requestInput,
		s3.WithPresignExpires(input.ExpiresIn),
	)
	if err != nil {
		return "", fmt.Errorf("presign put object: %w", err)
	}

	return request.URL, nil
}

func (c *S3Client) PresignGetObject(ctx context.Context, input PresignGetObjectInput) (string, error) {
	if c == nil || c.client == nil || c.presigner == nil {
		return "", ErrClientNotConfigured
	}

	key := strings.TrimSpace(input.ObjectKey)
	if key == "" {
		return "", fmt.Errorf("object key is required")
	}

	if input.ExpiresIn <= 0 {
		return "", fmt.Errorf("expiresIn must be greater than 0")
	}

	request, err := c.presigner.PresignGetObject(
		ctx,
		&s3.GetObjectInput{
			Bucket: aws.String(c.bucket),
			Key:    aws.String(key),
		},
		s3.WithPresignExpires(input.ExpiresIn),
	)
	if err != nil {
		return "", fmt.Errorf("presign get object: %w", err)
	}

	return request.URL, nil
}

func (c *S3Client) HeadObject(ctx context.Context, input HeadObjectInput) (HeadObjectOutput, error) {
	if c == nil || c.client == nil {
		return HeadObjectOutput{}, ErrClientNotConfigured
	}

	key := strings.TrimSpace(input.ObjectKey)
	if key == "" {
		return HeadObjectOutput{}, fmt.Errorf("object key is required")
	}

	response, err := c.client.HeadObject(ctx, &s3.HeadObjectInput{Bucket: aws.String(c.bucket), Key: aws.String(key)})
	if err != nil {
		return HeadObjectOutput{}, fmt.Errorf("head object: %w", err)
	}

	result := HeadObjectOutput{}
	if response.ContentLength != nil {
		result.ContentLength = *response.ContentLength
	}
	if response.ContentType != nil {
		result.ContentType = *response.ContentType
	}

	return result, nil
}

func IsNotFoundError(err error) bool {
	if err == nil {
		return false
	}

	var apiErr smithy.APIError
	if errors.As(err, &apiErr) {
		code := strings.ToLower(strings.TrimSpace(apiErr.ErrorCode()))
		return code == "notfound" || code == "nosuchkey"
	}

	return false
}

type NoopClient struct{}

func (NoopClient) Bucket() string {
	return ""
}

func (NoopClient) PresignPutObject(_ context.Context, _ PresignPutObjectInput) (string, error) {
	return "", ErrClientNotConfigured
}

func (NoopClient) PresignGetObject(_ context.Context, _ PresignGetObjectInput) (string, error) {
	return "", ErrClientNotConfigured
}

func (NoopClient) HeadObject(_ context.Context, _ HeadObjectInput) (HeadObjectOutput, error) {
	return HeadObjectOutput{}, ErrClientNotConfigured
}
