package cloud

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// S3Provider implements Provider interface for S3-compatible storage
type S3Provider struct {
	client   *s3.Client
	uploader *manager.Uploader
	bucket   string
	prefix   string
	endpoint string
}

// NewS3Provider creates a new S3 provider
func NewS3Provider(cfg Config) (*S3Provider, error) {
	ctx := context.Background()

	// Load AWS configuration
	var awsCfg aws.Config
	var err error

	if cfg.Endpoint != "" {
		// Custom endpoint for S3-compatible services
		awsCfg, err = config.LoadDefaultConfig(ctx,
			config.WithRegion(cfg.Region),
		)
		if err != nil {
			return nil, fmt.Errorf("failed to load AWS config: %w", err)
		}
	} else {
		// Standard AWS S3
		awsCfg, err = config.LoadDefaultConfig(ctx,
			config.WithRegion(cfg.Region),
		)
		if err != nil {
			return nil, fmt.Errorf("failed to load AWS config: %w", err)
		}
	}

	// Create S3 client with optional custom endpoint
	var client *s3.Client
	if cfg.Endpoint != "" {
		client = s3.NewFromConfig(awsCfg, func(o *s3.Options) {
			o.BaseEndpoint = aws.String(cfg.Endpoint)
			o.UsePathStyle = true // Required for most S3-compatible services
		})
	} else {
		client = s3.NewFromConfig(awsCfg)
	}

	uploader := manager.NewUploader(client)

	return &S3Provider{
		client:   client,
		uploader: uploader,
		bucket:   cfg.Bucket,
		prefix:   cfg.Prefix,
		endpoint: cfg.Endpoint,
	}, nil
}

// GetName returns the provider name
func (p *S3Provider) GetName() string {
	if p.endpoint != "" {
		return "S3-compatible"
	}
	return "AWS S3"
}

// Upload uploads a local file to S3
func (p *S3Provider) Upload(localPath, remotePath string) error {
	file, err := os.Open(localPath)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	key := p.buildKey(remotePath)

	_, err = p.uploader.Upload(context.Background(), &s3.PutObjectInput{
		Bucket: aws.String(p.bucket),
		Key:    aws.String(key),
		Body:   file,
	})
	if err != nil {
		return fmt.Errorf("failed to upload to S3: %w", err)
	}

	return nil
}

// Download downloads a file from S3 to local path
func (p *S3Provider) Download(remotePath, localPath string) error {
	key := p.buildKey(remotePath)

	// Create the directory if it doesn't exist
	dir := filepath.Dir(localPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Create the local file
	file, err := os.Create(localPath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	// Download the file
	downloader := manager.NewDownloader(p.client)
	_, err = downloader.Download(context.Background(), file, &s3.GetObjectInput{
		Bucket: aws.String(p.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		os.Remove(localPath) // Clean up partial file
		return fmt.Errorf("failed to download from S3: %w", err)
	}

	return nil
}

// List lists all backups in the S3 bucket
func (p *S3Provider) List(prefix string) ([]BackupEntry, error) {
	fullPrefix := p.buildKey(prefix)

	var entries []BackupEntry

	paginator := s3.NewListObjectsV2Paginator(p.client, &s3.ListObjectsV2Input{
		Bucket: aws.String(p.bucket),
		Prefix: aws.String(fullPrefix),
	})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(context.Background())
		if err != nil {
			return nil, fmt.Errorf("failed to list objects: %w", err)
		}

		for _, obj := range page.Contents {
			// Skip directories
			if strings.HasSuffix(*obj.Key, "/") {
				continue
			}

			// Only include backup files
			key := *obj.Key
			if !strings.HasSuffix(key, ".tar.gz") && !strings.HasSuffix(key, ".tar.gz.age") {
				continue
			}

			name := filepath.Base(key)
			entries = append(entries, BackupEntry{
				Name:         name,
				Key:          key,
				Size:         *obj.Size,
				LastModified: *obj.LastModified,
			})
		}
	}

	return entries, nil
}

// Delete deletes a file from S3
func (p *S3Provider) Delete(remotePath string) error {
	key := p.buildKey(remotePath)

	_, err := p.client.DeleteObject(context.Background(), &s3.DeleteObjectInput{
		Bucket: aws.String(p.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return fmt.Errorf("failed to delete from S3: %w", err)
	}

	return nil
}

// Exists checks if a file exists in S3
func (p *S3Provider) Exists(remotePath string) (bool, error) {
	key := p.buildKey(remotePath)

	_, err := p.client.HeadObject(context.Background(), &s3.HeadObjectInput{
		Bucket: aws.String(p.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		// Check if the error is "not found"
		if strings.Contains(err.Error(), "NotFound") || strings.Contains(err.Error(), "404") {
			return false, nil
		}
		return false, fmt.Errorf("failed to check S3 object: %w", err)
	}

	return true, nil
}

// buildKey constructs the full S3 key with optional prefix
func (p *S3Provider) buildKey(path string) string {
	if p.prefix == "" {
		return path
	}
	return strings.TrimSuffix(p.prefix, "/") + "/" + strings.TrimPrefix(path, "/")
}
