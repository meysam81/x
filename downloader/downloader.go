package downloader

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"
)

type Config struct {
	URL      string
	Filepath string

	MaxRetries       int
	InitialRetryWait time.Duration
	MaxRetryWait     time.Duration
	ChunkSize        int64
	Concurrency      int
	Timeout          time.Duration
	UserAgent        string

	Headers        map[string]string
	ExpectedSHA256 string
	FileMode       os.FileMode
	TempDir        string

	OnProgress func(downloaded, total int64)
	OnRetry    func(attempt int, err error)

	HTTPClient *http.Client
}

type Stats struct {
	BytesDownloaded int64
	BytesTotal      int64
	Duration        time.Duration
	Retries         int
}

type Downloader struct {
	config Config
	client *http.Client
	stats  Stats
}

func New(config Config) (*Downloader, error) {
	if config.URL == "" || config.Filepath == "" {
		return nil, errors.New("URL and Filepath are required")
	}

	if config.MaxRetries == 0 {
		config.MaxRetries = 3
	}
	if config.InitialRetryWait == 0 {
		config.InitialRetryWait = time.Second
	}
	if config.MaxRetryWait == 0 {
		config.MaxRetryWait = 30 * time.Second
	}
	if config.ChunkSize == 0 {
		config.ChunkSize = 1024 * 1024
	}
	if config.Concurrency == 0 {
		config.Concurrency = 4
	}
	if config.Timeout == 0 {
		config.Timeout = 5 * time.Minute
	}
	if config.UserAgent == "" {
		config.UserAgent = "go-downloader/1.0"
	}
	if config.FileMode == 0 {
		config.FileMode = 0644
	}
	if config.TempDir == "" {
		config.TempDir = filepath.Dir(config.Filepath)
	}

	client := config.HTTPClient
	if client == nil {
		client = &http.Client{
			Transport: &http.Transport{
				MaxIdleConns:        100,
				MaxIdleConnsPerHost: 10,
				IdleConnTimeout:     90 * time.Second,
				DisableCompression:  false,
			},
			Timeout: config.Timeout,
		}
	}

	return &Downloader{
		config: config,
		client: client,
	}, nil
}

func (d *Downloader) Download(ctx context.Context) error {
	startTime := time.Now()
	defer func() {
		d.stats.Duration = time.Since(startTime)
	}()

	fileInfo, err := d.getFileInfo(ctx)
	if err != nil {
		return fmt.Errorf("failed to get file info: %w", err)
	}

	d.stats.BytesTotal = fileInfo.Size

	if fileInfo.SupportsResume && fileInfo.Size > 0 {
		if d.config.Concurrency > 1 && fileInfo.Size > d.config.ChunkSize*2 {
			err = d.downloadParallel(ctx, fileInfo)
		} else {
			err = d.downloadWithResume(ctx, fileInfo)
		}
	} else {
		err = d.downloadSimple(ctx)
	}

	if err != nil {
		return err
	}

	if d.config.ExpectedSHA256 != "" {
		if err := d.verifyFile(); err != nil {
			_ = os.Remove(d.config.Filepath)
			return fmt.Errorf("integrity check failed: %w", err)
		}
	}

	return nil
}

type fileInfo struct {
	Size           int64
	SupportsResume bool
	LastModified   time.Time
	ETag           string
}

func (d *Downloader) getFileInfo(ctx context.Context) (*fileInfo, error) {
	req, err := http.NewRequestWithContext(ctx, "HEAD", d.config.URL, nil)
	if err != nil {
		return nil, err
	}

	d.setHeaders(req)

	resp, err := d.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return nil, fmt.Errorf("HEAD request failed: %s", resp.Status)
	}

	info := &fileInfo{
		Size:           resp.ContentLength,
		SupportsResume: resp.Header.Get("Accept-Ranges") == "bytes",
		ETag:           resp.Header.Get("ETag"),
	}

	if lastMod := resp.Header.Get("Last-Modified"); lastMod != "" {
		info.LastModified, _ = time.Parse(http.TimeFormat, lastMod)
	}

	return info, nil
}

func (d *Downloader) downloadParallel(ctx context.Context, info *fileInfo) error {
	tempFile := d.getTempFilePath()

	file, err := os.OpenFile(tempFile, os.O_CREATE|os.O_WRONLY, d.config.FileMode)
	if err != nil {
		return err
	}

	if err := file.Truncate(info.Size); err != nil {
		_ = file.Close()
		return err
	}
	_ = file.Close()

	chunkSize := info.Size / int64(d.config.Concurrency)
	if chunkSize < d.config.ChunkSize {
		chunkSize = d.config.ChunkSize
	}

	var wg sync.WaitGroup
	errChan := make(chan error, d.config.Concurrency)
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	for i := 0; i < d.config.Concurrency; i++ {
		start := int64(i) * chunkSize
		end := start + chunkSize - 1
		if i == d.config.Concurrency-1 {
			end = info.Size - 1
		}
		if start >= info.Size {
			break
		}

		wg.Add(1)
		go func(start, end int64) {
			defer wg.Done()
			if err := d.downloadChunk(ctx, tempFile, start, end); err != nil {
				select {
				case errChan <- err:
					cancel()
				default:
				}
			}
		}(start, end)
	}

	wg.Wait()
	close(errChan)

	for err := range errChan {
		return err
	}

	return d.atomicRename(tempFile)
}

func (d *Downloader) downloadChunk(ctx context.Context, filepath string, start, end int64) error {
	var lastErr error

	for attempt := 0; attempt <= d.config.MaxRetries; attempt++ {
		if attempt > 0 {
			waitTime := d.exponentialBackoff(attempt)

			if d.config.OnRetry != nil {
				d.config.OnRetry(attempt, lastErr)
			}

			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(waitTime):
			}
		}

		err := d.downloadChunkAttempt(ctx, filepath, start, end)
		if err == nil {
			return nil
		}

		lastErr = err

		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return err
		}
	}

	return fmt.Errorf("chunk download failed after %d attempts: %w", d.config.MaxRetries+1, lastErr)
}

func (d *Downloader) downloadChunkAttempt(ctx context.Context, filepath string, start, end int64) error {
	req, err := http.NewRequestWithContext(ctx, "GET", d.config.URL, nil)
	if err != nil {
		return err
	}

	req.Header.Set("Range", fmt.Sprintf("bytes=%d-%d", start, end))
	d.setHeaders(req)

	resp, err := d.client.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusPartialContent && resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status: %s", resp.Status)
	}

	file, err := os.OpenFile(filepath, os.O_WRONLY, 0)
	if err != nil {
		return err
	}
	defer func() { _ = file.Close() }()

	if _, err := file.Seek(start, io.SeekStart); err != nil {
		return err
	}

	return d.copyWithProgress(file, resp.Body, end-start+1)
}

func (d *Downloader) downloadWithResume(ctx context.Context, info *fileInfo) error {
	tempFile := d.getTempFilePath()

	var startPos int64
	if stat, err := os.Stat(tempFile); err == nil {
		startPos = stat.Size()
	}

	if startPos >= info.Size {
		return d.atomicRename(tempFile)
	}

	file, err := os.OpenFile(tempFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, d.config.FileMode)
	if err != nil {
		return err
	}
	defer func() { _ = file.Close() }()

	for attempt := 0; attempt <= d.config.MaxRetries; attempt++ {
		if attempt > 0 {
			waitTime := d.exponentialBackoff(attempt)

			if d.config.OnRetry != nil {
				d.config.OnRetry(attempt, err)
			}

			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(waitTime):
			}
		}

		err = d.attemptResumeDownload(ctx, file, info, tempFile, startPos)
		if err == nil {
			return d.atomicRename(tempFile)
		}

		if stat, statErr := os.Stat(tempFile); statErr == nil {
			startPos = stat.Size()
		}
		d.stats.Retries++
	}

	return fmt.Errorf("download failed after %d attempts", d.config.MaxRetries+1)
}

func (d *Downloader) attemptResumeDownload(ctx context.Context, file *os.File, info *fileInfo, tempFile string, startPos int64) error {
	req, err := http.NewRequestWithContext(ctx, "GET", d.config.URL, nil)
	if err != nil {
		return err
	}

	if startPos > 0 {
		req.Header.Set("Range", fmt.Sprintf("bytes=%d-", startPos))
	}
	d.setHeaders(req)

	resp, err := d.client.Do(req)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusPartialContent && resp.StatusCode != http.StatusOK {
		_ = resp.Body.Close()
		return fmt.Errorf("unexpected status: %s", resp.Status)
	}

	err = d.copyWithProgress(file, resp.Body, info.Size-startPos)
	_ = resp.Body.Close()

	return err
}

func (d *Downloader) downloadSimple(ctx context.Context) error {
	tempFile := d.getTempFilePath()

	for attempt := 0; attempt <= d.config.MaxRetries; attempt++ {
		if attempt > 0 {
			waitTime := d.exponentialBackoff(attempt)

			if d.config.OnRetry != nil {
				d.config.OnRetry(attempt, nil)
			}

			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(waitTime):
			}
		}

		err := d.downloadSimpleAttempt(ctx, tempFile)
		if err == nil {
			return d.atomicRename(tempFile)
		}

		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return err
		}

		d.stats.Retries++
	}

	return fmt.Errorf("simple download failed after %d attempts", d.config.MaxRetries+1)
}

func (d *Downloader) downloadSimpleAttempt(ctx context.Context, tempFile string) error {
	req, err := http.NewRequestWithContext(ctx, "GET", d.config.URL, nil)
	if err != nil {
		return err
	}

	d.setHeaders(req)

	resp, err := d.client.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status: %s", resp.Status)
	}

	file, err := os.OpenFile(tempFile, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, d.config.FileMode)
	if err != nil {
		return err
	}
	defer func() { _ = file.Close() }()

	expectedSize := resp.ContentLength
	if expectedSize < 0 {
		expectedSize = 0
	}

	return d.copyWithProgress(file, resp.Body, expectedSize)
}

func (d *Downloader) copyWithProgress(dst io.Writer, src io.Reader, expectedBytes int64) error {
	buf := make([]byte, 32*1024)
	var written int64

	progressChan := make(chan int64, 100)
	progressDone := make(chan struct{})

	go func() {
		defer close(progressDone)
		for bytesWritten := range progressChan {
			atomic.AddInt64(&d.stats.BytesDownloaded, bytesWritten)
			if d.config.OnProgress != nil {
				d.config.OnProgress(atomic.LoadInt64(&d.stats.BytesDownloaded), d.stats.BytesTotal)
			}
		}
	}()

	defer func() {
		close(progressChan)
		<-progressDone
	}()

	for {
		nr, readErr := src.Read(buf)
		if nr > 0 {
			nw, writeErr := dst.Write(buf[:nr])
			if nw > 0 {
				written += int64(nw)
				select {
				case progressChan <- int64(nw):
				default:

				}
			}
			if writeErr != nil {
				return writeErr
			}
			if nr != nw {
				return io.ErrShortWrite
			}
		}

		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return readErr
		}
	}

	if expectedBytes > 0 && written != expectedBytes {
		return fmt.Errorf("incomplete download: got %d bytes, expected %d", written, expectedBytes)
	}

	return nil
}

func (d *Downloader) verifyFile() error {
	file, err := os.Open(d.config.Filepath)
	if err != nil {
		return err
	}
	defer func() { _ = file.Close() }()

	h := sha256.New()
	if _, err := io.Copy(h, file); err != nil {
		return err
	}

	actualHash := hex.EncodeToString(h.Sum(nil))
	if actualHash != d.config.ExpectedSHA256 {
		return fmt.Errorf("SHA256 mismatch: expected %s, got %s", d.config.ExpectedSHA256, actualHash)
	}

	return nil
}

func (d *Downloader) atomicRename(tempFile string) error {

	file, err := os.Open(tempFile)
	if err != nil {
		return err
	}

	if err := file.Sync(); err != nil {
		_ = file.Close()
		return err
	}
	_ = file.Close()

	if err := os.Rename(tempFile, d.config.Filepath); err != nil {

		if _, statErr := os.Stat(d.config.Filepath); statErr == nil {
			if removeErr := os.Remove(d.config.Filepath); removeErr != nil {
				return fmt.Errorf("failed to remove existing file: %w", removeErr)
			}
			return os.Rename(tempFile, d.config.Filepath)
		}
		return err
	}

	return nil
}

func (d *Downloader) getTempFilePath() string {
	return filepath.Join(d.config.TempDir, fmt.Sprintf(".%s.download", filepath.Base(d.config.Filepath)))
}

func (d *Downloader) setHeaders(req *http.Request) {
	req.Header.Set("User-Agent", d.config.UserAgent)

	for key, value := range d.config.Headers {
		req.Header.Set(key, value)
	}
}

func (d *Downloader) exponentialBackoff(attempt int) time.Duration {
	wait := d.config.InitialRetryWait * time.Duration(math.Pow(2, float64(attempt-1)))
	if wait > d.config.MaxRetryWait {
		wait = d.config.MaxRetryWait
	}
	return wait
}

func (d *Downloader) GetStats() Stats {
	return Stats{
		BytesDownloaded: atomic.LoadInt64(&d.stats.BytesDownloaded),
		BytesTotal:      d.stats.BytesTotal,
		Duration:        d.stats.Duration,
		Retries:         d.stats.Retries,
	}
}
