package downloader

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"
)

func TestBasicDownload(t *testing.T) {
	content := []byte("Hello, World!")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", fmt.Sprintf("%d", len(content)))
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(content)
	}))
	defer server.Close()

	tempFile := filepath.Join(t.TempDir(), "test.txt")

	config := Config{
		URL:      server.URL + "/file",
		Filepath: tempFile,
	}

	dl, err := New(config)
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	if err := dl.Download(ctx); err != nil {
		t.Fatalf("Download failed: %v", err)
	}

	// Verify content
	downloaded, err := os.ReadFile(tempFile)
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(downloaded, content) {
		t.Errorf("Content mismatch: got %s, want %s", downloaded, content)
	}
}

func TestResumeDownload(t *testing.T) {
	fullContent := make([]byte, 1024*1024) // 1MB
	for i := range fullContent {
		fullContent[i] = byte(i % 256)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rangeHeader := r.Header.Get("Range")

		if r.Method == "HEAD" {
			w.Header().Set("Accept-Ranges", "bytes")
			w.Header().Set("Content-Length", fmt.Sprintf("%d", len(fullContent)))
			w.WriteHeader(http.StatusOK)
			return
		}

		if rangeHeader != "" {
			// Parse range header (simplified)
			var start, end int64
			_, _ = fmt.Sscanf(rangeHeader, "bytes=%d-%d", &start, &end)

			if end == 0 || end >= int64(len(fullContent)) {
				end = int64(len(fullContent)) - 1
			}

			w.Header().Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", start, end, len(fullContent)))
			w.Header().Set("Content-Length", fmt.Sprintf("%d", end-start+1))
			w.WriteHeader(http.StatusPartialContent)
			_, _ = w.Write(fullContent[start : end+1])
		} else {
			w.Header().Set("Content-Length", fmt.Sprintf("%d", len(fullContent)))
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(fullContent)
		}
	}))
	defer server.Close()

	tempFile := filepath.Join(t.TempDir(), "resume-test.bin")

	// First, create a partial file
	partialSize := len(fullContent) / 2
	if err := os.WriteFile(tempFile+".download", fullContent[:partialSize], 0644); err != nil {
		t.Fatal(err)
	}

	config := Config{
		URL:      server.URL + "/file",
		Filepath: tempFile,
		TempDir:  t.TempDir(),
	}

	dl, err := New(config)
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	if err := dl.Download(ctx); err != nil {
		t.Fatalf("Resume download failed: %v", err)
	}

	// Verify complete content
	downloaded, err := os.ReadFile(tempFile)
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(downloaded, fullContent) {
		t.Errorf("Content mismatch after resume: lengths %d vs %d", len(downloaded), len(fullContent))
	}
}

func TestParallelDownload(t *testing.T) {
	size := 10 * 1024 * 1024 // 10MB
	content := make([]byte, size)
	for i := range content {
		content[i] = byte(i % 256)
	}

	var requestCount int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "HEAD" {
			w.Header().Set("Accept-Ranges", "bytes")
			w.Header().Set("Content-Length", fmt.Sprintf("%d", len(content)))
			w.WriteHeader(http.StatusOK)
			return
		}

		atomic.AddInt32(&requestCount, 1)

		rangeHeader := r.Header.Get("Range")
		if rangeHeader != "" {
			var start, end int64
			_, _ = fmt.Sscanf(rangeHeader, "bytes=%d-%d", &start, &end)

			if end >= int64(len(content)) {
				end = int64(len(content)) - 1
			}

			w.Header().Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", start, end, len(content)))
			w.Header().Set("Content-Length", fmt.Sprintf("%d", end-start+1))
			w.WriteHeader(http.StatusPartialContent)
			_, _ = w.Write(content[start : end+1])
		} else {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(content)
		}
	}))
	defer server.Close()

	tempFile := filepath.Join(t.TempDir(), "parallel-test.bin")

	config := Config{
		URL:         server.URL + "/file",
		Filepath:    tempFile,
		Concurrency: 4,
		ChunkSize:   1024 * 1024, // 1MB chunks
	}

	dl, err := New(config)
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	if err := dl.Download(ctx); err != nil {
		t.Fatalf("Parallel download failed: %v", err)
	}

	// Verify multiple requests were made
	count := atomic.LoadInt32(&requestCount)
	if count < 2 {
		t.Errorf("Expected multiple parallel requests, got %d", count)
	}

	// Verify content
	downloaded, err := os.ReadFile(tempFile)
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(downloaded, content) {
		t.Error("Content mismatch in parallel download")
	}
}

func TestRetryMechanism(t *testing.T) {
	var attempts int32
	maxAttempts := int32(3)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		current := atomic.AddInt32(&attempts, 1)

		if r.Method == "HEAD" {
			w.Header().Set("Content-Length", "100")
			w.WriteHeader(http.StatusOK)
			return
		}

		if current < maxAttempts {
			// Simulate failure
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		// Success on last attempt
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("Success after retries"))
	}))
	defer server.Close()

	tempFile := filepath.Join(t.TempDir(), "retry-test.txt")

	var retryCount int
	config := Config{
		URL:              server.URL + "/file",
		Filepath:         tempFile,
		MaxRetries:       3,
		InitialRetryWait: 10 * time.Millisecond,
		OnRetry: func(attempt int, err error) {
			retryCount = attempt
		},
	}

	dl, err := New(config)
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	if err := dl.Download(ctx); err != nil {
		t.Fatalf("Download with retry failed: %v", err)
	}

	if retryCount == 0 {
		t.Error("Expected retries but none occurred")
	}

	content, _ := os.ReadFile(tempFile)
	if string(content) != "Success after retries" {
		t.Errorf("Unexpected content: %s", content)
	}
}

func TestIntegrityVerification(t *testing.T) {
	content := []byte("Test content for SHA256 verification")
	h := sha256.Sum256(content)
	expectedHash := hex.EncodeToString(h[:])

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(content)
	}))
	defer server.Close()

	t.Run("ValidHash", func(t *testing.T) {
		tempFile := filepath.Join(t.TempDir(), "valid-hash.txt")

		config := Config{
			URL:            server.URL + "/file",
			Filepath:       tempFile,
			ExpectedSHA256: expectedHash,
		}

		dl, err := New(config)
		if err != nil {
			t.Fatal(err)
		}

		ctx := context.Background()
		if err := dl.Download(ctx); err != nil {
			t.Fatalf("Download with valid hash failed: %v", err)
		}

		// File should exist
		if _, err := os.Stat(tempFile); os.IsNotExist(err) {
			t.Error("File should exist after successful verification")
		}
	})

	t.Run("InvalidHash", func(t *testing.T) {
		tempFile := filepath.Join(t.TempDir(), "invalid-hash.txt")

		config := Config{
			URL:            server.URL + "/file",
			Filepath:       tempFile,
			ExpectedSHA256: "invalid0000000000000000000000000000000000000000000000000000000",
		}

		dl, err := New(config)
		if err != nil {
			t.Fatal(err)
		}

		ctx := context.Background()
		err = dl.Download(ctx)
		if err == nil {
			t.Fatal("Expected integrity check to fail")
		}

		// File should be cleaned up
		if _, err := os.Stat(tempFile); !os.IsNotExist(err) {
			t.Error("File should be removed after failed verification")
		}
	})
}

func TestContextCancellation(t *testing.T) {
	blocked := make(chan struct{})
	unblock := make(chan struct{})

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "HEAD" {
			w.Header().Set("Content-Length", "1000000")
			w.WriteHeader(http.StatusOK)
			return
		}

		close(blocked)
		<-unblock // Block until test cancels
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()
	defer close(unblock)

	tempFile := filepath.Join(t.TempDir(), "cancel-test.bin")

	config := Config{
		URL:      server.URL + "/file",
		Filepath: tempFile,
	}

	dl, err := New(config)
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	errChan := make(chan error)
	go func() {
		errChan <- dl.Download(ctx)
	}()

	<-blocked // Wait for download to start
	cancel()  // Cancel the context

	err = <-errChan
	if !errors.Is(err, context.Canceled) {
		t.Errorf("Expected context.Canceled, got %v", err)
	}
}

func TestProgressCallback(t *testing.T) {
	content := make([]byte, 1024*100) // 100KB
	for i := range content {
		content[i] = byte(i % 256)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", fmt.Sprintf("%d", len(content)))
		w.WriteHeader(http.StatusOK)

		// Write in chunks to trigger progress updates
		chunkSize := 1024
		for i := 0; i < len(content); i += chunkSize {
			end := i + chunkSize
			if end > len(content) {
				end = len(content)
			}
			_, _ = w.Write(content[i:end])
			_, _ = w.Write(content[i:end])
			time.Sleep(10 * time.Millisecond) // Slow down to ensure progress callbacks
		}
	}))
	defer server.Close()

	tempFile := filepath.Join(t.TempDir(), "progress-test.bin")

	var progressCalls int
	var lastTotal int64

	config := Config{
		URL:      server.URL + "/file",
		Filepath: tempFile,
		OnProgress: func(downloaded, total int64) {
			progressCalls++
			lastTotal = total
		},
	}

	dl, err := New(config)
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	if err := dl.Download(ctx); err != nil {
		t.Fatalf("Download failed: %v", err)
	}

	if progressCalls == 0 {
		t.Error("Progress callback was never called")
	}

	if lastTotal != int64(len(content)) {
		t.Errorf("Final total incorrect: got %d, want %d", lastTotal, len(content))
	}

	stats := dl.GetStats()
	if stats.BytesDownloaded != int64(len(content)) {
		t.Errorf("Stats bytes downloaded incorrect: got %d, want %d",
			stats.BytesDownloaded, len(content))
	}
}

func TestCustomHeaders(t *testing.T) {
	var receivedHeaders http.Header

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedHeaders = r.Header.Clone()
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	}))
	defer server.Close()

	tempFile := filepath.Join(t.TempDir(), "headers-test.txt")

	config := Config{
		URL:      server.URL + "/file",
		Filepath: tempFile,
		Headers: map[string]string{
			"Authorization": "Bearer token123",
			"X-Custom":      "CustomValue",
		},
		UserAgent: "TestAgent/1.0",
	}

	dl, err := New(config)
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	if err := dl.Download(ctx); err != nil {
		t.Fatalf("Download failed: %v", err)
	}

	if receivedHeaders.Get("Authorization") != "Bearer token123" {
		t.Error("Authorization header not sent correctly")
	}

	if receivedHeaders.Get("X-Custom") != "CustomValue" {
		t.Error("Custom header not sent correctly")
	}

	if receivedHeaders.Get("User-Agent") != "TestAgent/1.0" {
		t.Error("User-Agent not sent correctly")
	}
}

func BenchmarkDownload(b *testing.B) {
	size := 10 * 1024 * 1024 // 10MB
	content := make([]byte, size)
	for i := range content {
		content[i] = byte(i % 256)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "HEAD" {
			w.Header().Set("Accept-Ranges", "bytes")
			w.Header().Set("Content-Length", fmt.Sprintf("%d", len(content)))
			w.WriteHeader(http.StatusOK)
			return
		}

		rangeHeader := r.Header.Get("Range")
		if rangeHeader != "" {
			var start, end int64
			_, _ = fmt.Sscanf(rangeHeader, "bytes=%d-%d", &start, &end)

			if end >= int64(len(content)) {
				end = int64(len(content)) - 1
			}

			w.Header().Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", start, end, len(content)))
			w.WriteHeader(http.StatusPartialContent)
			_, _ = w.Write(content[start : end+1])
		} else {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(content)
		}
	}))
	defer server.Close()

	b.Run("Sequential", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			tempFile := filepath.Join(b.TempDir(), fmt.Sprintf("bench-seq-%d.bin", i))

			config := Config{
				URL:         server.URL + "/file",
				Filepath:    tempFile,
				Concurrency: 1,
			}

			dl, _ := New(config)
			ctx := context.Background()
			_ = dl.Download(ctx)
		}
	})

	b.Run("Parallel4", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			tempFile := filepath.Join(b.TempDir(), fmt.Sprintf("bench-par-%d.bin", i))

			config := Config{
				URL:         server.URL + "/file",
				Filepath:    tempFile,
				Concurrency: 4,
			}

			dl, _ := New(config)
			ctx := context.Background()
			_ = dl.Download(ctx)
		}
	})
}
