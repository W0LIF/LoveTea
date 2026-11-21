package storage

import (
	"bytes"
	"context"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"net/url"
	"time"

	"github.com/W0LIF/LoveTea/backend/internal/config"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

const (
	UploadURLTTL   = 10 * time.Minute
	DownloadURLTTL = 5 * time.Minute
	MaxUploadBytes = 20 << 20
)

type UploadForm struct {
	URL       string            `json:"url"`
	Fields    map[string]string `json:"fields"`
	ExpiresAt time.Time         `json:"expires_at"`
}

type ObjectInfo struct {
	Size        int64
	ContentType string
	Format      string
}

type Client struct {
	internal *minio.Client
	public   *minio.Client
	bucket   string
}

func NewClient(cfg *config.Config) (*Client, error) {
	internal, err := newMinIOClient(cfg.MinIOEndpoint, cfg)
	if err != nil {
		return nil, fmt.Errorf("configure internal MinIO client: %w", err)
	}
	public, err := newMinIOClient(cfg.MinIOPublicEndpoint, cfg)
	if err != nil {
		return nil, fmt.Errorf("configure public MinIO client: %w", err)
	}
	return &Client{
		internal: internal,
		public:   public,
		bucket:   cfg.MinIOBucket,
	}, nil
}

func newMinIOClient(endpoint string, cfg *config.Config) (*minio.Client, error) {
	return minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.MinIOAccessKey, cfg.MinIOSecretKey, ""),
		Secure: cfg.MinIOUseTLS,
		Region: "us-east-1",
	})
}

func (c *Client) EnsureBucket(ctx context.Context) error {
	exists, err := c.internal.BucketExists(ctx, c.bucket)
	if err != nil {
		return fmt.Errorf("check media bucket: %w", err)
	}
	if exists {
		return nil
	}
	if err := c.internal.MakeBucket(ctx, c.bucket, minio.MakeBucketOptions{Region: "us-east-1"}); err != nil {
		return fmt.Errorf("create media bucket: %w", err)
	}
	return nil
}

func (c *Client) CreateUploadForm(
	ctx context.Context,
	objectKey string,
	contentType string,
) (UploadForm, error) {
	expiresAt := time.Now().UTC().Add(UploadURLTTL)
	policy := minio.NewPostPolicy()
	if err := policy.SetBucket(c.bucket); err != nil {
		return UploadForm{}, fmt.Errorf("set upload policy bucket: %w", err)
	}
	if err := policy.SetKey(objectKey); err != nil {
		return UploadForm{}, fmt.Errorf("set upload policy key: %w", err)
	}
	if err := policy.SetContentType(contentType); err != nil {
		return UploadForm{}, fmt.Errorf("set upload policy content type: %w", err)
	}
	if err := policy.SetContentLengthRange(1, MaxUploadBytes); err != nil {
		return UploadForm{}, fmt.Errorf("set upload policy size: %w", err)
	}
	if err := policy.SetExpires(expiresAt); err != nil {
		return UploadForm{}, fmt.Errorf("set upload policy expiry: %w", err)
	}
	uploadURL, fields, err := c.public.PresignedPostPolicy(ctx, policy)
	if err != nil {
		return UploadForm{}, fmt.Errorf("create signed upload policy: %w", err)
	}
	return UploadForm{
		URL:       uploadURL.String(),
		Fields:    fields,
		ExpiresAt: expiresAt,
	}, nil
}

func (c *Client) CreateDownloadURL(
	ctx context.Context,
	objectKey string,
) (string, time.Time, error) {
	expiresAt := time.Now().UTC().Add(DownloadURLTTL)
	signedURL, err := c.public.PresignedGetObject(
		ctx,
		c.bucket,
		objectKey,
		DownloadURLTTL,
		url.Values{},
	)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("create signed download URL: %w", err)
	}
	return signedURL.String(), expiresAt, nil
}

func (c *Client) Inspect(ctx context.Context, objectKey string) (ObjectInfo, error) {
	info, err := c.internal.StatObject(ctx, c.bucket, objectKey, minio.StatObjectOptions{})
	if err != nil {
		return ObjectInfo{}, fmt.Errorf("inspect uploaded media: %w", err)
	}
	if info.Size < 1 || info.Size > MaxUploadBytes {
		return ObjectInfo{}, fmt.Errorf("uploaded media size is outside the permitted range")
	}
	object, err := c.internal.GetObject(ctx, c.bucket, objectKey, minio.GetObjectOptions{})
	if err != nil {
		return ObjectInfo{}, fmt.Errorf("read uploaded media for validation: %w", err)
	}
	defer object.Close()
	data, err := io.ReadAll(io.LimitReader(object, MaxUploadBytes+1))
	if err != nil {
		return ObjectInfo{}, fmt.Errorf("read uploaded media bytes: %w", err)
	}
	if int64(len(data)) != info.Size {
		return ObjectInfo{}, fmt.Errorf("uploaded media size changed during validation")
	}
	config, format, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return ObjectInfo{}, fmt.Errorf("decode uploaded image header: %w", err)
	}
	if config.Width < 1 || config.Height < 1 ||
		config.Width > 10000 || config.Height > 10000 ||
		int64(config.Width)*int64(config.Height) > 25000000 {
		return ObjectInfo{}, fmt.Errorf("uploaded image dimensions exceed the permitted range")
	}
	if _, _, err := image.Decode(bytes.NewReader(data)); err != nil {
		return ObjectInfo{}, fmt.Errorf("decode uploaded image: %w", err)
	}
	return ObjectInfo{Size: info.Size, ContentType: info.ContentType, Format: format}, nil
}

func (c *Client) Delete(ctx context.Context, objectKey string) error {
	if err := c.internal.RemoveObject(ctx, c.bucket, objectKey, minio.RemoveObjectOptions{}); err != nil {
		return fmt.Errorf("delete media object: %w", err)
	}
	return nil
}
