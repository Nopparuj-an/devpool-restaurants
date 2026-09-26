// Package storage puts and deletes image objects in S3-compatible storage
// (Garage, ADR-0005). Reads never go through the API: the bucket is served
// publicly by Garage's web endpoint, so the DB stores only object keys.
package storage

import (
	"bytes"
	"context"
	"fmt"
	"sync"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

type Store interface {
	Put(ctx context.Context, key, contentType string, body []byte) error
	Delete(ctx context.Context, keys ...string) error
}

type S3Config struct {
	Endpoint, Region, Bucket, AccessKeyID, SecretAccessKey string
}

type S3 struct {
	client *s3.Client
	bucket string
}

func NewS3(cfg S3Config) *S3 {
	client := s3.New(s3.Options{
		BaseEndpoint: aws.String(cfg.Endpoint),
		Region:       cfg.Region,
		Credentials:  credentials.NewStaticCredentialsProvider(cfg.AccessKeyID, cfg.SecretAccessKey, ""),
		UsePathStyle: true, // Garage's S3 API is addressed as endpoint/bucket/key
		// Only send checksums when the operation requires them; the SDK's
		// newer default (trailing CRC checksums) is not needed for Garage.
		RequestChecksumCalculation: aws.RequestChecksumCalculationWhenRequired,
		ResponseChecksumValidation: aws.ResponseChecksumValidationWhenRequired,
	})
	return &S3{client: client, bucket: cfg.Bucket}
}

func (s *S3) Put(ctx context.Context, key, contentType string, body []byte) error {
	_, err := s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:        &s.bucket,
		Key:           &key,
		Body:          bytes.NewReader(body),
		ContentLength: aws.Int64(int64(len(body))),
		ContentType:   &contentType,
		// Keys contain a random UUID and are never overwritten, so browsers may cache forever.
		CacheControl: aws.String("public, max-age=31536000, immutable"),
	})
	if err != nil {
		return fmt.Errorf("s3 put %s: %w", key, err)
	}
	return nil
}

func (s *S3) Delete(ctx context.Context, keys ...string) error {
	if len(keys) == 0 {
		return nil
	}
	objs := make([]types.ObjectIdentifier, len(keys))
	for i, k := range keys {
		objs[i] = types.ObjectIdentifier{Key: aws.String(k)}
	}
	_, err := s.client.DeleteObjects(ctx, &s3.DeleteObjectsInput{
		Bucket: &s.bucket,
		Delete: &types.Delete{Objects: objs, Quiet: aws.Bool(true)},
	})
	if err != nil {
		return fmt.Errorf("s3 delete %d objects: %w", len(keys), err)
	}
	return nil
}

// Memory is an in-memory Store for tests.
type Memory struct {
	mu      sync.Mutex
	Objects map[string][]byte
}

func NewMemory() *Memory { return &Memory{Objects: map[string][]byte{}} }

func (m *Memory) Put(_ context.Context, key, _ string, body []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Objects[key] = body
	return nil
}

func (m *Memory) Delete(_ context.Context, keys ...string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, k := range keys {
		delete(m.Objects, k)
	}
	return nil
}

func (m *Memory) Len() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.Objects)
}
