package storage

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"

	"indieforge/internal/config"
)

// S3 wraps an S3-compatible object store (MinIO locally, S3/Yandex in prod).
type S3 struct {
	client *s3.Client // in-network: PutObject, HeadBucket, bucket policy

	// presignClient is built against PublicEndpoint, not Endpoint. SigV4
	// signs the Host header, so a URL handed to a browser must be presigned
	// for the host the browser will actually hit — signing with the internal
	// endpoint and rewriting the host afterwards would invalidate the
	// signature.
	presignClient *s3.PresignClient

	bucket         string
	publicEndpoint string
}

// New builds an S3 client configured for a path-style endpoint (MinIO/Yandex).
func New(c config.S3Config) *S3 {
	creds := credentials.NewStaticCredentialsProvider(c.AccessKey, c.SecretKey, "")
	client := s3.New(s3.Options{
		Region:       c.Region,
		Credentials:  creds,
		BaseEndpoint: aws.String(c.Endpoint),
		UsePathStyle: true,
	})
	publicClient := s3.New(s3.Options{
		Region:       c.Region,
		Credentials:  creds,
		BaseEndpoint: aws.String(c.PublicEndpoint),
		UsePathStyle: true,
	})
	return &S3{
		client:         client,
		presignClient:  s3.NewPresignClient(publicClient),
		bucket:         c.Bucket,
		publicEndpoint: strings.TrimRight(c.PublicEndpoint, "/"),
	}
}

// EnsureBucket creates the bucket if it does not exist, then makes the
// media/* and web/* prefixes publicly readable via a bucket policy.
//
// Per-object "public-read" ACLs are not reliably honoured by every
// S3-compatible backend (notably MinIO, which favours bucket policies), so
// a policy is the portable way to expose cover art, screenshots and browser
// builds to the frontend without authentication. downloads/* stays private —
// those are only ever reached through a short-lived presigned URL.
func (s *S3) EnsureBucket(ctx context.Context) error {
	_, err := s.client.HeadBucket(ctx, &s3.HeadBucketInput{Bucket: &s.bucket})
	if err != nil {
		if _, err := s.client.CreateBucket(ctx, &s3.CreateBucketInput{Bucket: &s.bucket}); err != nil {
			return fmt.Errorf("create bucket: %w", err)
		}
	}
	if err := s.ensurePublicReadPolicy(ctx); err != nil {
		return fmt.Errorf("set public-read policy: %w", err)
	}
	return nil
}

func (s *S3) ensurePublicReadPolicy(ctx context.Context) error {
	policy := map[string]any{
		"Version": "2012-10-17",
		"Statement": []map[string]any{
			{
				"Effect":    "Allow",
				"Principal": map[string]any{"AWS": []string{"*"}},
				"Action":    []string{"s3:GetObject"},
				"Resource": []string{
					"arn:aws:s3:::" + s.bucket + "/media/*",
					"arn:aws:s3:::" + s.bucket + "/web/*",
				},
			},
		},
	}
	body, err := json.Marshal(policy)
	if err != nil {
		return err
	}
	policyStr := string(body)
	_, err = s.client.PutBucketPolicy(ctx, &s3.PutBucketPolicyInput{
		Bucket: &s.bucket,
		Policy: &policyStr,
	})
	return err
}

// PublicURL returns the browser-reachable URL for a public object key.
func (s *S3) PublicURL(key string) string {
	return s.publicEndpoint + "/" + s.bucket + "/" + key
}

func contentTypeFor(name string) string {
	if ct := mime.TypeByExtension(filepath.Ext(name)); ct != "" {
		return ct
	}
	return "application/octet-stream"
}

// put stores an object. Public readability comes from the bucket policy set
// up in EnsureBucket (media/*, web/*), not from a per-object ACL — ACL grants
// are inconsistently supported across S3-compatible backends and rejected by
// AWS S3 buckets with Block Public ACLs enabled.
func (s *S3) put(ctx context.Context, key, contentType string, data []byte) error {
	_, err := s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      &s.bucket,
		Key:         &key,
		Body:        bytes.NewReader(data),
		ContentType: aws.String(contentType),
	})
	return err
}

// PutPublic stores an object under a publicly-readable prefix and returns its URL.
func (s *S3) PutPublic(ctx context.Context, key, contentType string, data []byte) (string, error) {
	if contentType == "" {
		contentType = contentTypeFor(key)
	}
	if err := s.put(ctx, key, contentType, data); err != nil {
		return "", err
	}
	return s.PublicURL(key), nil
}

// PutPrivate stores a private object (downloadable build).
func (s *S3) PutPrivate(ctx context.Context, key, contentType string, data []byte) error {
	if contentType == "" {
		contentType = contentTypeFor(key)
	}
	return s.put(ctx, key, contentType, data)
}

// PresignGet returns a short-lived URL to download a private object, signed
// for the public endpoint so it's directly fetchable by a browser.
func (s *S3) PresignGet(ctx context.Context, key string, ttl time.Duration) (string, error) {
	out, err := s.presignClient.PresignGetObject(ctx, &s3.GetObjectInput{Bucket: &s.bucket, Key: &key},
		s3.WithPresignExpires(ttl))
	if err != nil {
		return "", err
	}
	return out.URL, nil
}

// ExtractZipToPrefix unzips an HTML5 build into the prefix (public) and returns
// the URL of the build's index.html. Guards against Zip-Slip.
func (s *S3) ExtractZipToPrefix(ctx context.Context, prefix string, zipData []byte) (string, error) {
	zr, err := zip.NewReader(bytes.NewReader(zipData), int64(len(zipData)))
	if err != nil {
		return "", fmt.Errorf("read zip: %w", err)
	}
	prefix = strings.TrimRight(prefix, "/") + "/"

	indexKey := ""
	for _, f := range zr.File {
		if f.FileInfo().IsDir() {
			continue
		}
		clean := path.Clean("/" + strings.ReplaceAll(f.Name, "\\", "/"))
		if strings.Contains(clean, "..") {
			continue // Zip-Slip guard
		}
		rel := strings.TrimPrefix(clean, "/")
		key := prefix + rel

		rc, err := f.Open()
		if err != nil {
			return "", err
		}
		data, err := io.ReadAll(rc)
		_ = rc.Close()
		if err != nil {
			return "", err
		}
		if _, err := s.PutPublic(ctx, key, contentTypeFor(rel), data); err != nil {
			return "", err
		}
		if base := strings.ToLower(path.Base(rel)); base == "index.html" {
			if indexKey == "" || strings.Count(key, "/") < strings.Count(indexKey, "/") {
				indexKey = key
			}
		}
	}
	if indexKey == "" {
		return "", fmt.Errorf("no index.html found in build")
	}
	return s.PublicURL(indexKey), nil
}
