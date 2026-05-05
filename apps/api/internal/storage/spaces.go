package storage

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// Client abstracts binary object storage (DO Spaces / S3).
type Client interface {
	Upload(ctx context.Context, key string, data []byte, contentType string) error
	PresignURL(ctx context.Context, key string, expiry time.Duration) (string, error)
}

// SpacesClient stores objects in DigitalOcean Spaces (S3-compatible).
type SpacesClient struct {
	client *s3.Client
	bucket string
}

// NewFromEnv builds a SpacesClient from environment variables:
//
//	SPACES_KEY, SPACES_SECRET, SPACES_ENDPOINT, SPACES_BUCKET
//
// Returns an error if any variable is missing.
func NewFromEnv() (Client, error) {
	key := os.Getenv("SPACES_KEY")
	secret := os.Getenv("SPACES_SECRET")
	endpoint := os.Getenv("SPACES_ENDPOINT") // e.g. "https://sgp1.digitaloceanspaces.com"
	bucket := os.Getenv("SPACES_BUCKET")

	if key == "" || secret == "" || endpoint == "" || bucket == "" {
		return nil, fmt.Errorf("SPACES_KEY, SPACES_SECRET, SPACES_ENDPOINT, SPACES_BUCKET are required")
	}

	cfg, err := awsconfig.LoadDefaultConfig(context.Background(),
		awsconfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(key, secret, "")),
		awsconfig.WithRegion("us-east-1"), // region is irrelevant for DO Spaces
	)
	if err != nil {
		return nil, fmt.Errorf("spaces config: %w", err)
	}

	c := s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.BaseEndpoint = aws.String(endpoint)
		o.UsePathStyle = true
	})

	return &SpacesClient{client: c, bucket: bucket}, nil
}

func (c *SpacesClient) Upload(ctx context.Context, key string, data []byte, contentType string) error {
	_, err := c.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(c.bucket),
		Key:         aws.String(key),
		Body:        bytes.NewReader(data),
		ContentType: aws.String(contentType),
	})
	if err != nil {
		return fmt.Errorf("spaces upload %q: %w", key, err)
	}
	return nil
}

func (c *SpacesClient) PresignURL(ctx context.Context, key string, expiry time.Duration) (string, error) {
	presigner := s3.NewPresignClient(c.client)
	req, err := presigner.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(key),
	}, s3.WithPresignExpires(expiry))
	if err != nil {
		return "", fmt.Errorf("spaces presign %q: %w", key, err)
	}
	return req.URL, nil
}
