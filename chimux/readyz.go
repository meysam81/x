package chimux

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"
)

// defaultReadyzTimeout bounds how long a single ReadyCheck may run before it
// is treated as failed. Override it with WithReadyzTimeout.
const defaultReadyzTimeout = 2 * time.Second

// ReadyCheck is one dependency probe run by the readiness endpoint. Name
// identifies the check in the failure report; Check reports whether the
// dependency is ready and receives a context bounded by the configured
// per-check timeout.
type ReadyCheck struct {
	Name  string
	Check func(ctx context.Context) error
}

// readyzResponse is the JSON body written by the readyz handler.
type readyzResponse struct {
	Status string            `json:"status"`
	Failed map[string]string `json:"failed,omitempty"`
}

const readyzOKBody = `{"status":"ready"}`

// readyzHandler runs every configured ReadyCheck concurrently on each
// request. All checks passing, or zero checks configured, responds 200; any
// failure responds 503 naming the failing checks and their errors.
func readyzHandler(o *options) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		failed := runReadyChecks(r.Context(), o.readyChecks, o.readyzTimeout)

		w.Header().Set("Content-Type", "application/json")

		if len(failed) == 0 {
			w.WriteHeader(http.StatusOK)
			_, _ = fmt.Fprint(w, readyzOKBody)
			return
		}

		body, err := json.Marshal(readyzResponse{Status: "not ready", Failed: failed})
		if err != nil {
			// readyzResponse holds only strings; Marshal cannot fail on it.
			body = []byte(`{"status":"not ready"}`)
		}

		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = w.Write(body)
	}
}

// runReadyChecks runs checks concurrently, each bounded by timeout, and
// returns the name -> error text of every one that failed.
func runReadyChecks(ctx context.Context, checks []ReadyCheck, timeout time.Duration) map[string]string {
	failed := make(map[string]string)
	if len(checks) == 0 {
		return failed
	}

	var mu sync.Mutex
	var wg sync.WaitGroup

	for _, c := range checks {
		wg.Add(1)
		go func(c ReadyCheck) {
			defer wg.Done()
			if err := runReadyCheck(ctx, timeout, c); err != nil {
				mu.Lock()
				failed[c.Name] = err.Error()
				mu.Unlock()
			}
		}(c)
	}

	wg.Wait()
	return failed
}

// runReadyCheck runs one ReadyCheck with its own timeout derived from ctx.
// A panic inside c.Check is recovered and reported as a failure. If c.Check
// does not honor context cancellation and keeps running past timeout, this
// still returns once the timeout elapses -- the abandoned goroutine exits on
// its own whenever c.Check eventually returns, since done is buffered.
func runReadyCheck(ctx context.Context, timeout time.Duration, c ReadyCheck) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	done := make(chan error, 1)
	go func() {
		defer func() {
			if p := recover(); p != nil {
				done <- fmt.Errorf("panic: %v", p)
			}
		}()
		done <- c.Check(ctx)
	}()

	select {
	case err := <-done:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}
