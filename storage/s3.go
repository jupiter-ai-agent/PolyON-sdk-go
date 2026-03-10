// Package storage provides S3-compatible storage access for PolyON modules.
//
// Uses the PRC objectStorage credentials (RustFS/MinIO) from environment variables.
package storage

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	polyon "github.com/jupiter-ai-agent/PolyON-sdk-go"
)

// Client provides S3 operations via HTTP (no external dependency required).
// For advanced S3 operations, use the official AWS SDK with these credentials.
type Client struct {
	Endpoint  string
	Bucket    string
	AccessKey string
	SecretKey string
	http      *http.Client
}

// NewClient creates a storage client from PRC config.
func NewClient(cfg polyon.StorageConfig) *Client {
	return &Client{
		Endpoint:  strings.TrimRight(cfg.Endpoint, "/"),
		Bucket:    cfg.Bucket,
		AccessKey: cfg.AccessKey,
		SecretKey: cfg.SecretKey,
		http:      &http.Client{Timeout: 30 * time.Second},
	}
}

// URL returns the full S3 URL for an object key.
func (c *Client) URL(key string) string {
	return fmt.Sprintf("%s/%s/%s", c.Endpoint, c.Bucket, key)
}

// Put uploads an object to the bucket.
func (c *Client) Put(ctx context.Context, key string, body io.Reader, contentType string) error {
	url := c.URL(key)
	req, err := http.NewRequestWithContext(ctx, "PUT", url, body)
	if err != nil {
		return err
	}
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	// Note: For production, use proper AWS Signature V4.
	// This is a simplified client for internal RustFS access.

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("storage put failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return fmt.Errorf("storage put %s: %d", key, resp.StatusCode)
	}
	return nil
}

// Get downloads an object from the bucket.
func (c *Client) Get(ctx context.Context, key string) (io.ReadCloser, error) {
	url := c.URL(key)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("storage get failed: %w", err)
	}

	if resp.StatusCode == 404 {
		resp.Body.Close()
		return nil, fmt.Errorf("storage: %s not found", key)
	}
	if resp.StatusCode >= 300 {
		resp.Body.Close()
		return nil, fmt.Errorf("storage get %s: %d", key, resp.StatusCode)
	}
	return resp.Body, nil
}

// Delete removes an object from the bucket.
func (c *Client) Delete(ctx context.Context, key string) error {
	url := c.URL(key)
	req, err := http.NewRequestWithContext(ctx, "DELETE", url, nil)
	if err != nil {
		return err
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("storage delete failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 && resp.StatusCode != 404 {
		return fmt.Errorf("storage delete %s: %d", key, resp.StatusCode)
	}
	return nil
}
