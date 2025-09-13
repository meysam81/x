# Downloader

## Example

```go
package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

	"your-module/downloader" // Replace with your module path
)

func main() {
	// Example 1: Basic download with progress
	basicExample()

	// Example 2: Download with authentication and custom headers
	authenticatedExample()

	// Example 3: Large file with parallel chunks
	parallelExample()

	// Example 4: Download with integrity verification
	verifiedExample()

	// Example 5: Graceful cancellation
	cancellableExample()
}

func basicExample() {
	config := downloader.Config{
		URL:      "https://example.com/file.zip",
		Filepath: "downloads/file.zip",
		OnProgress: func(downloaded, total int64) {
			if total > 0 {
				percent := float64(downloaded) / float64(total) * 100
				fmt.Printf("\rProgress: %.2f%% (%d/%d bytes)", percent, downloaded, total)
			}
		},
		OnRetry: func(attempt int, err error) {
			log.Printf("Retry attempt %d due to: %v", attempt, err)
		},
	}

	dl, err := downloader.New(config)
	if err != nil {
		log.Fatal(err)
	}

	ctx := context.Background()
	if err := dl.Download(ctx); err != nil {
		log.Printf("Download failed: %v", err)
		return
	}

	stats := dl.GetStats()
	fmt.Printf("\nDownload completed: %d bytes in %v with %d retries\n",
		stats.BytesDownloaded, stats.Duration, stats.Retries)
}

func authenticatedExample() {
	config := downloader.Config{
		URL:      "https://api.example.com/private/data.json",
		Filepath: "data.json",
		Headers: map[string]string{
			"Authorization": "Bearer " + os.Getenv("API_TOKEN"),
			"Accept":        "application/json",
		},
		MaxRetries: 5,
		Timeout:    30 * time.Second,
	}

	dl, err := downloader.New(config)
	if err != nil {
		log.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	if err := dl.Download(ctx); err != nil {
		log.Printf("Authenticated download failed: %v", err)
	}
}

func parallelExample() {
	// Custom HTTP client with specific transport settings
	client := &http.Client{
		Transport: &http.Transport{
			MaxIdleConns:        200,
			MaxIdleConnsPerHost: 20,
			IdleConnTimeout:     90 * time.Second,
			DisableCompression:  false,
			// Proxy settings if needed
			// Proxy: http.ProxyFromEnvironment,
		},
		Timeout: 10 * time.Minute,
	}

	config := downloader.Config{
		URL:         "https://releases.ubuntu.com/20.04/ubuntu-20.04.6-desktop-amd64.iso",
		Filepath:    "ubuntu.iso",
		Concurrency: 8, // Download in 8 parallel chunks
		ChunkSize:   5 * 1024 * 1024, // 5MB chunks
		HTTPClient:  client,
		OnProgress: func(downloaded, total int64) {
			if total > 0 {
				mbDownloaded := float64(downloaded) / 1024 / 1024
				mbTotal := float64(total) / 1024 / 1024
				percent := float64(downloaded) / float64(total) * 100
				fmt.Printf("\rProgress: %.2f%% (%.2f MB / %.2f MB)",
					percent, mbDownloaded, mbTotal)
			}
		},
	}

	dl, err := downloader.New(config)
	if err != nil {
		log.Fatal(err)
	}

	ctx := context.Background()
	start := time.Now()

	if err := dl.Download(ctx); err != nil {
		log.Printf("Large file download failed: %v", err)
		return
	}

	elapsed := time.Since(start)
	stats := dl.GetStats()
	speed := float64(stats.BytesDownloaded) / elapsed.Seconds() / 1024 / 1024
	fmt.Printf("\nDownloaded %.2f MB in %v (%.2f MB/s)\n",
		float64(stats.BytesDownloaded)/1024/1024, elapsed, speed)
}

func verifiedExample() {
	config := downloader.Config{
		URL:            "https://example.com/important-file.tar.gz",
		Filepath:       "important-file.tar.gz",
		ExpectedSHA256: "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
		MaxRetries:     3,
		OnProgress: func(downloaded, total int64) {
			// Update UI or log progress
		},
	}

	dl, err := downloader.New(config)
	if err != nil {
		log.Fatal(err)
	}

	ctx := context.Background()
	if err := dl.Download(ctx); err != nil {
		log.Printf("Download or verification failed: %v", err)
		// File is automatically cleaned up if verification fails
	} else {
		log.Println("Download completed and verified successfully")
	}
}

func cancellableExample() {
	config := downloader.Config{
		URL:      "https://example.com/large-file.bin",
		Filepath: "large-file.bin",
		OnProgress: func(downloaded, total int64) {
			if total > 0 {
				percent := float64(downloaded) / float64(total) * 100
				fmt.Printf("\rProgress: %.2f%%", percent)
			}
		},
	}

	dl, err := downloader.New(config)
	if err != nil {
		log.Fatal(err)
	}

	// Create cancellable context
	ctx, cancel := context.WithCancel(context.Background())

	// Handle Ctrl+C
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt)

	downloadDone := make(chan error, 1)

	go func() {
		downloadDone <- dl.Download(ctx)
	}()

	select {
	case <-sigChan:
		fmt.Println("\nDownload cancelled by user")
		cancel()
		<-downloadDone // Wait for download to clean up
	case err := <-downloadDone:
		if err != nil {
			log.Printf("Download failed: %v", err)
		} else {
			fmt.Println("\nDownload completed successfully")
		}
	}
}

// Unit test example
func ExampleDownloader_Download() {
	// This would go in a _test.go file

	// Mock server for testing
	server := createMockServer()
	defer server.Close()

	config := downloader.Config{
		URL:            server.URL + "/test-file",
		Filepath:       "test-output.bin",
		ExpectedSHA256: "abc123...",
		MaxRetries:     2,
		Concurrency:    4,
	}

	dl, _ := downloader.New(config)
	ctx := context.Background()

	err := dl.Download(ctx)
	if err != nil {
		log.Printf("Test failed: %v", err)
	}

	// Clean up
	os.Remove("test-output.bin")
}

func createMockServer() *http.Server {
	// Implementation of mock server for testing
	// Would include handlers for partial content, retries, etc.
	return nil
}
```
