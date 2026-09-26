package storage

import (
	"context"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
)

// TestS3AgainstGarage round-trips an object through the real Garage from
// docker compose, including the public read path. `make test` sets the env.
func TestS3AgainstGarage(t *testing.T) {
	cfg := S3Config{
		Endpoint:        os.Getenv("TEST_S3_ENDPOINT"),
		Region:          os.Getenv("S3_REGION"),
		Bucket:          os.Getenv("S3_BUCKET"),
		AccessKeyID:     os.Getenv("S3_ACCESS_KEY_ID"),
		SecretAccessKey: os.Getenv("S3_SECRET_ACCESS_KEY"),
	}
	publicURL := os.Getenv("TEST_IMAGE_PUBLIC_URL")
	if cfg.Endpoint == "" || publicURL == "" {
		t.Skip("TEST_S3_ENDPOINT / TEST_IMAGE_PUBLIC_URL not set (run `make test`)")
	}
	ctx := context.Background()
	s := NewS3(cfg)
	key := "test/" + t.Name() + ".txt"

	if err := s.Put(ctx, key, "text/plain", []byte("hello")); err != nil {
		t.Fatal(err)
	}
	res, err := http.Get(publicURL + "/" + key)
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(res.Body)
	res.Body.Close()
	if res.StatusCode != 200 || string(body) != "hello" {
		t.Fatalf("public GET = %d %q", res.StatusCode, body)
	}
	if cc := res.Header.Get("Cache-Control"); !strings.Contains(cc, "immutable") {
		t.Errorf("Cache-Control = %q, want immutable", cc)
	}

	if err := s.Delete(ctx, key); err != nil {
		t.Fatal(err)
	}
	res, err = http.Get(publicURL + "/" + key)
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	if res.StatusCode != 404 {
		t.Fatalf("after delete GET = %d, want 404", res.StatusCode)
	}
}
