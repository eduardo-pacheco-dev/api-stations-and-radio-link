// Package r2storage implements document.Storage using Cloudflare R2.
package r2storage

import (
	"context"
	"fmt"
	"io"
	"path"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"

	"github.com/eduardo-pacheco-dev/api-stations-and-radio-link/internal/document"
)

// Config configures the Cloudflare R2 storage.
type Config struct {
	Endpoint        string
	AccessKeyID     string
	SecretAccessKey string
	Bucket          string
	Region          string
	Prefix          string
	MaxBytes        int64
}

// Storage stores documents in Cloudflare R2.
type Storage struct {
	client   *s3.Client
	uploader *manager.Uploader
	bucket   string
	prefix   string
	maxBytes int64
}

// New creates a Cloudflare R2 storage adapter.
func New(cfg Config) (*Storage, error) {
	if strings.TrimSpace(cfg.Endpoint) == "" {
		return nil, fmt.Errorf("R2_ENDPOINT must not be empty")
	}
	if strings.TrimSpace(cfg.AccessKeyID) == "" {
		return nil, fmt.Errorf("R2_ACCESS_KEY_ID must not be empty")
	}
	if strings.TrimSpace(cfg.SecretAccessKey) == "" {
		return nil, fmt.Errorf("R2_SECRET_ACCESS_KEY must not be empty")
	}
	if strings.TrimSpace(cfg.Bucket) == "" {
		return nil, fmt.Errorf("R2_BUCKET must not be empty")
	}
	if cfg.MaxBytes <= 0 {
		return nil, fmt.Errorf("document max bytes must be positive")
	}

	region := strings.TrimSpace(cfg.Region)
	if region == "" {
		region = "auto"
	}

	client := s3.NewFromConfig(aws.Config{
		Region:      region,
		Credentials: credentials.NewStaticCredentialsProvider(cfg.AccessKeyID, cfg.SecretAccessKey, ""),
	}, func(options *s3.Options) {
		options.BaseEndpoint = aws.String(cfg.Endpoint)
		options.UsePathStyle = true
	})

	return &Storage{
		client:   client,
		uploader: manager.NewUploader(client),
		bucket:   cfg.Bucket,
		prefix:   strings.Trim(cfg.Prefix, "/"),
		maxBytes: cfg.MaxBytes,
	}, nil
}

// Save uploads a document to R2 and returns its object key and size.
func (s *Storage) Save(ctx context.Context, id, originalName, contentType string, reader io.Reader) (string, int64, error) {
	key := s.objectKey(id + document.SafeExtension(originalName))
	counter := &countingReader{reader: io.LimitReader(reader, s.maxBytes+1)}

	if strings.TrimSpace(contentType) == "" {
		contentType = "application/octet-stream"
	}

	_, err := s.uploader.Upload(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(s.bucket),
		Key:         aws.String(key),
		Body:        counter,
		ContentType: aws.String(contentType),
	})
	if err != nil {
		_ = s.Remove(ctx, key)
		return "", 0, fmt.Errorf("upload document to R2: %w", err)
	}
	if counter.written > s.maxBytes {
		_ = s.Remove(ctx, key)
		return "", 0, document.ErrFileTooLarge
	}

	return key, counter.written, nil
}

// Open opens a document object from R2.
func (s *Storage) Open(ctx context.Context, key string) (io.ReadCloser, error) {
	output, err := s.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, fmt.Errorf("get document from R2: %w", err)
	}

	return output.Body, nil
}

// Remove deletes a document object from R2.
func (s *Storage) Remove(ctx context.Context, key string) error {
	if key == "" {
		return nil
	}

	if _, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	}); err != nil {
		return fmt.Errorf("delete document from R2: %w", err)
	}

	return nil
}

func (s *Storage) objectKey(name string) string {
	if s.prefix == "" {
		return name
	}

	return path.Join(s.prefix, name)
}

type countingReader struct {
	reader  io.Reader
	written int64
}

func (r *countingReader) Read(buffer []byte) (int, error) {
	read, err := r.reader.Read(buffer)
	r.written += int64(read)

	return read, err
}
