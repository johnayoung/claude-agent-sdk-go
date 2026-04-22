package s3_test

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"os"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	awss3 "github.com/aws/aws-sdk-go-v2/service/s3"

	claude "github.com/johnayoung/claude-agent-sdk-go"
	"github.com/johnayoung/claude-agent-sdk-go/agenttest/sessionstoretest"
	s3store "github.com/johnayoung/claude-agent-sdk-go/examples/session_stores/s3"
)

// TestConformance runs the shared SessionStore conformance suite against a
// live S3 backend. Skipped unless SESSION_STORE_S3_BUCKET is set. Target
// real AWS by leaving SESSION_STORE_S3_ENDPOINT unset, or target MinIO by
// setting both the endpoint and static credentials.
//
// Example (MinIO):
//
//	docker run -d --rm -p 9000:9000 \
//	    -e MINIO_ROOT_USER=minioadmin -e MINIO_ROOT_PASSWORD=minioadmin \
//	    minio/minio server /data
//	docker run --rm --network host minio/mc sh -c \
//	    'mc alias set l http://localhost:9000 minioadmin minioadmin && mc mb l/sstest'
//
//	SESSION_STORE_S3_BUCKET=sstest \
//	SESSION_STORE_S3_ENDPOINT=http://localhost:9000 \
//	SESSION_STORE_S3_ACCESS_KEY=minioadmin \
//	SESSION_STORE_S3_SECRET_KEY=minioadmin \
//	SESSION_STORE_S3_REGION=us-east-1 \
//	    go test -v ./...
//
// Each subtest uses a unique object-key prefix plus per-test cleanup, so
// concurrent runs don't collide.
func TestConformance(t *testing.T) {
	bucket := os.Getenv("SESSION_STORE_S3_BUCKET")
	if bucket == "" {
		t.Skip("SESSION_STORE_S3_BUCKET not set; skipping live S3 conformance")
	}

	region := os.Getenv("SESSION_STORE_S3_REGION")
	if region == "" {
		region = "us-east-1"
	}

	loadCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	opts := []func(*config.LoadOptions) error{config.WithRegion(region)}
	if ak, sk := os.Getenv("SESSION_STORE_S3_ACCESS_KEY"), os.Getenv("SESSION_STORE_S3_SECRET_KEY"); ak != "" && sk != "" {
		opts = append(opts, config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(ak, sk, "")))
	}
	cfg, err := config.LoadDefaultConfig(loadCtx, opts...)
	if err != nil {
		t.Fatalf("config.LoadDefaultConfig: %v", err)
	}

	var s3Opts []func(*awss3.Options)
	if ep := os.Getenv("SESSION_STORE_S3_ENDPOINT"); ep != "" {
		s3Opts = append(s3Opts, func(o *awss3.Options) {
			o.BaseEndpoint = aws.String(ep)
			// MinIO and many S3-compatible services require path-style URLs.
			o.UsePathStyle = true
		})
	}
	client := awss3.NewFromConfig(cfg, s3Opts...)

	// Sanity-check the bucket exists before running 14 subtests against it.
	if _, err := client.HeadBucket(loadCtx, &awss3.HeadBucketInput{Bucket: aws.String(bucket)}); err != nil {
		t.Fatalf("HeadBucket %s: %v", bucket, err)
	}

	sessionstoretest.Run(t, func(t *testing.T) claude.SessionStore {
		prefix := "sstest-" + randomHex(t, 6)
		t.Cleanup(func() {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			deleteUnderPrefix(ctx, t, client, bucket, prefix+"/")
		})
		return s3store.New(client, bucket, prefix)
	})
}

func deleteUnderPrefix(ctx context.Context, t *testing.T, client *awss3.Client, bucket, prefix string) {
	t.Helper()
	var token *string
	for {
		out, err := client.ListObjectsV2(ctx, &awss3.ListObjectsV2Input{
			Bucket:            aws.String(bucket),
			Prefix:            aws.String(prefix),
			ContinuationToken: token,
		})
		if err != nil {
			t.Logf("s3 cleanup list %s: %v", prefix, err)
			return
		}
		for _, obj := range out.Contents {
			if _, err := client.DeleteObject(ctx, &awss3.DeleteObjectInput{
				Bucket: aws.String(bucket),
				Key:    obj.Key,
			}); err != nil {
				t.Logf("s3 cleanup delete %s: %v", aws.ToString(obj.Key), err)
			}
		}
		if out.IsTruncated == nil || !*out.IsTruncated {
			return
		}
		token = out.NextContinuationToken
	}
}

func randomHex(t *testing.T, n int) string {
	t.Helper()
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		t.Fatalf("rand: %v", err)
	}
	return hex.EncodeToString(b)
}
