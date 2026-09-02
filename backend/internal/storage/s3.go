package storage

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

type AnalyzerStatus string

const (
	AnalyzerStatusUploaded  AnalyzerStatus = "uploaded"
	AnalyzerStatusAnalyzing AnalyzerStatus = "analyzing"
	AnalyzerStatusAnalyzed  AnalyzerStatus = "analyzed"
	AnalyzerStatusError     AnalyzerStatus = "error"
)

const (
	AnalyzerFileName = "analysis.json"
	MetadataFileName = "metadata.json"
	VideoFileName    = "video.mp4"
)

type S3Client struct {
	client   *s3.Client
	psClient *s3.PresignClient
	bucket   string
}

// VideoFile holds the streaming body and content metadata for a video object.
// The caller must close Body when done to avoid leaking the HTTP connection.
type VideoFile struct {
	Body          io.ReadCloser
	ContentType   string
	ContentLength int64
}

// ObjectInfo is a lightweight listing entry — just the key and last-modified
// time S3 reports. It carries no domain meaning; callers decide what a key
// represents.
type ObjectInfo struct {
	Key          string
	LastModified time.Time
}

// CompletedPart identifies one successfully uploaded part of a multipart upload, as reported back by the client after each part PUT.
type CompletedPart struct {
	PartNumber int32
	ETag       string
}

func NewS3Client(ctx context.Context, bucket string) (*S3Client, error) {
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("load aws config: %w", err)
	}

	client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		// Override to local S3 instance for local development environments
		env := os.Getenv("ENVIRONMENT")
		if env == "development" || env == "test" {
			o.UsePathStyle = true
			o.BaseEndpoint = aws.String("http://localhost:9000")

			user, password := os.Getenv("MINIO_USER"), os.Getenv("MINIO_PASSWORD")
			o.Credentials = credentials.NewStaticCredentialsProvider(user, password, "")
		}
	})
	return &S3Client{
		client:   client,
		psClient: s3.NewPresignClient(client),
		bucket:   bucket,
	}, nil
}

// ListObjects lists every object under prefix (S3 has no real directories,
// so this recurses implicitly through anything sharing the prefix).
func (c *S3Client) ListObjects(ctx context.Context, prefix string) ([]ObjectInfo, error) {
	if !strings.HasSuffix(prefix, "/") {
		prefix = fmt.Sprintf("%s/", prefix)
	}

	var objects []ObjectInfo

	paginator := s3.NewListObjectsV2Paginator(c.client, &s3.ListObjectsV2Input{
		Bucket: aws.String(c.bucket),
		Prefix: aws.String(prefix),
	})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("list objects: %w", err)
		}

		for _, obj := range page.Contents {
			objects = append(objects, ObjectInfo{
				Key:          aws.ToString(obj.Key),
				LastModified: aws.ToTime(obj.LastModified),
			})
		}
	}

	return objects, nil
}

// GetObject fetches an object's full body. Meant for small objects like JSON metadata
// Use GetVideo for streaming video bytes instead.
func (c *S3Client) GetObject(ctx context.Context, key string) ([]byte, error) {
	out, err := c.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, fmt.Errorf("get s3 object %q: %w", key, err)
	}
	defer out.Body.Close()

	data, err := io.ReadAll(out.Body)
	if err != nil {
		return nil, fmt.Errorf("read s3 object %q: %w", key, err)
	}
	return data, nil
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

// PutJSON marshals v and writes it to key as application/json
func (c *S3Client) PutJSON(ctx context.Context, key string, v any) error {
	data, err := json.Marshal(v)
	if err != nil {
		return fmt.Errorf("marshal json for %q: %w", key, err)
	}
	return c.put(ctx, key, bytes.NewReader(data), "application/json")
}

// UpdateAnalyzerStatus updates the status field in the metadata JSON for a given video.
func (c *S3Client) UpdateAnalyzerStatus(ctx context.Context, key string, newStatus AnalyzerStatus) error {
	// Fetch the existing metadata JSON from S3
	data, err := c.GetObject(ctx, key)
	if err != nil {
		return fmt.Errorf("get metadata for %q: %w", key, err)
	}

	// Unmarshal the JSON into a map to modify the status field
	var meta map[string]any
	if err := json.Unmarshal(data, &meta); err != nil {
		return fmt.Errorf("unmarshal metadata for %q: %w", key, err)
	}

	// Update the status field
	meta["status"] = newStatus

	return c.PutJSON(ctx, key, meta)
}

// GetPresignedURL returns a time-limited URL that grants read access to a single video object. Keep ttl short (5–15 min) and never log the URL.
func (c *S3Client) GetPresignedURL(ctx context.Context, key string, ttl time.Duration) (string, error) {
	req, err := c.psClient.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(key),
	}, s3.WithPresignExpires(ttl))
	if err != nil {
		return "", fmt.Errorf("presign video %q: %w", key, err)
	}
	return req.URL, nil
}

// CreateMultipartUpload starts a multipart upload for key and returns the upload ID clients need to presign and complete/abort it.
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

// PresignUploadPart returns a time-limited URL the client can PUT a single part's bytes to directly.
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

// CompleteMultipartUpload finalizes a multipart upload once every part has been uploaded. parts must be sorted by PartNumber.
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
