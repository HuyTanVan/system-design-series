package workers

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// https://docs.aws.amazon.com/sdk-for-go/v2/developer-guide/sdk-utilities-s3.html

func UploadSnapshot(filePath, bucket, key, region string) error {
	start := time.Now()

	cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion(region))
	if err != nil {
		log.Fatalf("failed to load configuration, %v", err)
	}

	f, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open snapshot file: %w", err)
	}
	defer f.Close()

	client := s3.NewFromConfig(cfg)

	uploader := manager.NewUploader(client)
	_, err = uploader.Upload(context.TODO(), &s3.PutObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
		Body:   f,
	})
	if err != nil {
		return fmt.Errorf("failed to upload snapshot: %w", err)
	}

	log.Printf("snapshot uploaded to S3 | bucket=%s | key=%s | time=%s", bucket, key, time.Since(start))
	return nil
}

func DownloadSnapshot(outputPath, bucket, key, region string) error {
	start := time.Now()

	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		return fmt.Errorf("failed to load configuration, %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
		return fmt.Errorf("failed to create data dir: %w", err)
	}
	tmpPath := outputPath + ".tmp"
	f, err := os.Create(tmpPath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer f.Close()

	client := s3.NewFromConfig(cfg)
	downloader := manager.NewDownloader(client)
	_, err = downloader.Download(context.TODO(), f, &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return fmt.Errorf("failed to download snapshot: %w", err)
	}

	if err := os.Rename(tmpPath, outputPath); err != nil {
		return fmt.Errorf("failed to rename temp file: %w", err)
	}

	log.Printf("snapshot downloaded from S3 | bucket=%s | key=%s | time=%s", bucket, key, time.Since(start))
	return nil
}
