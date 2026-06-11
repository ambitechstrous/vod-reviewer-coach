package storage

import (
	"context"
	"fmt"
	"io"
	"mime"
	"path/filepath"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type S3Client struct {
	client *s3.Client
	bucket string
}

// VideoFile holds the streaming body and content metadata for a video object.
// The caller must close Body when done to avoid leaking the HTTP connection.
type VideoFile struct {
	Body          io.ReadCloser
	ContentType   string
	ContentLength int64
}

func NewS3Client(ctx context.Context, bucket string) (*S3Client, error) {
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("load aws config: %w", err)
	}
	return &S3Client{
		client: s3.NewFromConfig(cfg),
		bucket: bucket,
	}, nil
}

// PresignGetVideo returns a time-limited URL that grants read access to a
// single video object. Keep ttl short (5–15 min) and never log the URL.
func (c *S3Client) PresignGetVideo(ctx context.Context, key string, ttl time.Duration) (string, error) {
	pc := s3.NewPresignClient(c.client)
	req, err := pc.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(key),
	}, s3.WithPresignExpires(ttl))
	if err != nil {
		return "", fmt.Errorf("presign video %q: %w", key, err)
	}
	return req.URL, nil
}

// PutAudio uploads an audio file to S3 under the given key.
// Content-Type is inferred from the key extension (e.g. .aac → audio/aac, .mp3 → audio/mpeg).
func (c *S3Client) PutAudio(ctx context.Context, key string, body io.Reader) error {
	return c.put(ctx, key, body, contentTypeFromKey(key, "audio/octet-stream"))
}

// PutImage uploads an image file to S3 under the given key.
// Content-Type is inferred from the key extension (e.g. .jpg → image/jpeg, .png → image/png).
func (c *S3Client) PutImage(ctx context.Context, key string, body io.Reader) error {
	return c.put(ctx, key, body, contentTypeFromKey(key, "image/octet-stream"))
}

func contentTypeFromKey(key, fallback string) string {
	ct := mime.TypeByExtension(filepath.Ext(key))
	if ct == "" {
		return fallback
	}
	return ct
}

func (c *S3Client) put(ctx context.Context, key string, body io.Reader, contentType string) error {
	_, err := c.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(c.bucket),
		Key:         aws.String(key),
		Body:        body,
		ContentType: aws.String(contentType),
	})
	if err != nil {
		return fmt.Errorf("put s3 object %q: %w", key, err)
	}
	return nil
}

// GetVideo streams a video object from S3 by its key.
func (c *S3Client) GetVideo(ctx context.Context, key string) (*VideoFile, error) {
	out, err := c.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, fmt.Errorf("get s3 object %q: %w", key, err)
	}

	var contentType string
	if out.ContentType != nil {
		contentType = *out.ContentType
	}

	var contentLength int64
	if out.ContentLength != nil {
		contentLength = *out.ContentLength
	}

	return &VideoFile{
		Body:          out.Body,
		ContentType:   contentType,
		ContentLength: contentLength,
	}, nil
}
