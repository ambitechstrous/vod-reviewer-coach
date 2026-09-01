package storage

import (
	"context"
	"fmt"
	"io"
	"mime"
	"os"
	"path/filepath"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
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

	// AWS_ENDPOINT_URL (read automatically by config.LoadDefaultConfig) points
	// the SDK at a non-AWS endpoint like MinIO for local development. MinIO
	// only understands path-style requests (host/bucket/key), not the
	// virtual-hosted style (bucket.host/key) the SDK defaults to, so that
	// must be opted into separately here.
	usePathStyle := os.Getenv("S3_FORCE_PATH_STYLE") == "true"

	return &S3Client{
		client: s3.NewFromConfig(cfg, func(o *s3.Options) {
			o.UsePathStyle = usePathStyle
		}),
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

// CompletedPart identifies one successfully uploaded part of a multipart
// upload, as reported back by the client after each part PUT.
type CompletedPart struct {
	PartNumber int32
	ETag       string
}

// CreateMultipartUpload starts a multipart upload for key and returns the
// upload ID clients need to presign and complete/abort it.
func (c *S3Client) CreateMultipartUpload(ctx context.Context, key, contentType string) (string, error) {
	out, err := c.client.CreateMultipartUpload(ctx, &s3.CreateMultipartUploadInput{
		Bucket:      aws.String(c.bucket),
		Key:         aws.String(key),
		ContentType: aws.String(contentType),
	})
	if err != nil {
		return "", fmt.Errorf("create multipart upload %q: %w", key, err)
	}
	return aws.ToString(out.UploadId), nil
}

// PresignUploadPart returns a time-limited URL the client can PUT a single
// part's bytes to directly.
func (c *S3Client) PresignUploadPart(ctx context.Context, key, uploadID string, partNumber int32, ttl time.Duration) (string, error) {
	pc := s3.NewPresignClient(c.client)
	req, err := pc.PresignUploadPart(ctx, &s3.UploadPartInput{
		Bucket:     aws.String(c.bucket),
		Key:        aws.String(key),
		UploadId:   aws.String(uploadID),
		PartNumber: aws.Int32(partNumber),
	}, s3.WithPresignExpires(ttl))
	if err != nil {
		return "", fmt.Errorf("presign upload part %q (part %d): %w", key, partNumber, err)
	}
	return req.URL, nil
}

// CompleteMultipartUpload finalizes a multipart upload once every part has
// been uploaded. parts must be sorted by PartNumber.
func (c *S3Client) CompleteMultipartUpload(ctx context.Context, key, uploadID string, parts []CompletedPart) error {
	completed := make([]types.CompletedPart, len(parts))
	for i, p := range parts {
		completed[i] = types.CompletedPart{
			PartNumber: aws.Int32(p.PartNumber),
			ETag:       aws.String(p.ETag),
		}
	}

	_, err := c.client.CompleteMultipartUpload(ctx, &s3.CompleteMultipartUploadInput{
		Bucket:   aws.String(c.bucket),
		Key:      aws.String(key),
		UploadId: aws.String(uploadID),
		MultipartUpload: &types.CompletedMultipartUpload{
			Parts: completed,
		},
	})
	if err != nil {
		return fmt.Errorf("complete multipart upload %q: %w", key, err)
	}
	return nil
}

// AbortMultipartUpload cancels an in-progress multipart upload and releases
// any parts already uploaded, so it should be called whenever a client gives
// up partway through (network failure, user cancels, etc).
func (c *S3Client) AbortMultipartUpload(ctx context.Context, key, uploadID string) error {
	_, err := c.client.AbortMultipartUpload(ctx, &s3.AbortMultipartUploadInput{
		Bucket:   aws.String(c.bucket),
		Key:      aws.String(key),
		UploadId: aws.String(uploadID),
	})
	if err != nil {
		return fmt.Errorf("abort multipart upload %q: %w", key, err)
	}
	return nil
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
