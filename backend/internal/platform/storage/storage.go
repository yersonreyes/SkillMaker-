package storage

import (
	"context"
	"net/url"
	"strings"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"

	"github.com/yersonreyes/SkillMaker-/backend/internal/platform/config"
)

// Client defines the object storage operations required by the application.
// All implementations (MinIO, S3, GCS) must satisfy this interface.
type Client interface {
	// PresignPutURL returns a pre-signed URL for uploading an object.
	PresignPutURL(ctx context.Context, objectName string, ttl time.Duration) (string, error)

	// PresignGetURL returns a pre-signed URL for downloading an object.
	PresignGetURL(ctx context.Context, objectName string, ttl time.Duration) (string, error)

	// Delete removes an object from storage. It is idempotent — deleting a
	// non-existent object returns nil.
	Delete(ctx context.Context, objectName string) error

	// Ping verifies that the storage backend is reachable.
	Ping(ctx context.Context) error
}

type minioClient struct {
	client *minio.Client
	bucket string
}

// New initialises a MinIO client from the provided StorageConfig and returns
// a Client interface. It validates the connection by listing buckets.
func New(cfg config.StorageConfig) (Client, error) {
	// Strip the scheme from the endpoint — the MinIO SDK expects "host:port".
	endpoint := strings.TrimPrefix(cfg.Endpoint, "http://")
	endpoint = strings.TrimPrefix(endpoint, "https://")

	mc, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
		Secure: cfg.UseSSL,
	})
	if err != nil {
		return nil, err
	}

	return &minioClient{client: mc, bucket: cfg.Bucket}, nil
}

func (m *minioClient) PresignPutURL(ctx context.Context, objectName string, ttl time.Duration) (string, error) {
	u, err := m.client.PresignedPutObject(ctx, m.bucket, objectName, ttl)
	if err != nil {
		return "", err
	}
	return u.String(), nil
}

func (m *minioClient) PresignGetURL(ctx context.Context, objectName string, ttl time.Duration) (string, error) {
	u, err := m.client.PresignedGetObject(ctx, m.bucket, objectName, ttl, url.Values{})
	if err != nil {
		return "", err
	}
	return u.String(), nil
}

func (m *minioClient) Delete(ctx context.Context, objectName string) error {
	return m.client.RemoveObject(ctx, m.bucket, objectName, minio.RemoveObjectOptions{})
}

func (m *minioClient) Ping(ctx context.Context) error {
	_, err := m.client.BucketExists(ctx, m.bucket)
	return err
}
