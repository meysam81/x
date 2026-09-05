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
//
// Implementations should honour ctx and return promptly once it is done.
// Concurrent /readyz requests naming the same check share one in-flight
// run: if Check ignores cancellation and keeps blocking, at most one
// goroutine (and whatever it holds open, e.g. a pooled connection) stays
// alive on its behalf no matter how many requests arrive or time out while
// it is still running.
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

// readyRun is one execution of a named ReadyCheck, shared by every
// concurrent /readyz request that names the same check while it runs.
type readyRun struct {
	done chan struct{}
	err  error
}

// readyzRunner deduplicates concurrent executions of the same ReadyCheck by
// name. Without it, a check that ignores context cancellation would get a
// fresh goroutine (and, for something like db.PingContext, a fresh pooled
// connection) on every request that times out waiting on it -- at a 2s
// probe period that exhausts a connection pool in under a minute. With it,
// a wedged check accumulates at most one goroutine total, regardless of how
// many requests arrive while it is still running.
type readyzRunner struct {
	mu       sync.Mutex
	inflight map[string]*readyRun
}

// run waits for check c's result, starting it if no run is already in
// flight for c.Name. The check itself runs against reqCtx with
// cancellation stripped (context.WithoutCancel) so that one caller
// disconnecting never fails a check other callers are waiting on; the
// check still stops being waited on via its own timeout, and the run
// naturally ends whenever c.Check itself returns. The wait below, however,
// is always bounded by timeout, so run returns promptly regardless of how
// long the underlying check takes.
func (rr *readyzRunner) run(reqCtx context.Context, timeout time.Duration, c ReadyCheck) error {
	rr.mu.Lock()
	run, ok := rr.inflight[c.Name]
	if !ok {
		run = &readyRun{done: make(chan struct{})}
		rr.inflight[c.Name] = run

		checkCtx, cancel := context.WithTimeout(context.WithoutCancel(reqCtx), timeout)
		go func() {
			defer cancel()

			run.err = runCheckRecovered(checkCtx, c)
			close(run.done)

			rr.mu.Lock()
			delete(rr.inflight, c.Name)
			rr.mu.Unlock()
		}()
	}
	rr.mu.Unlock()

	waitCtx, waitCancel := context.WithTimeout(reqCtx, timeout)
	defer waitCancel()

	select {
	case <-run.done:
		return run.err
	case <-waitCtx.Done():
		return waitCtx.Err()
	}
}

// runCheckRecovered runs c.Check, converting a panic into an error instead
// of letting it crash the server.
func runCheckRecovered(ctx context.Context, c ReadyCheck) (err error) {
	defer func() {
		if p := recover(); p != nil {
			err = fmt.Errorf("panic: %v", p)
		}
	}()
	return c.Check(ctx)
}

// readyzHandler runs every configured ReadyCheck concurrently on each
// request. All checks passing, or zero checks configured, responds 200; any
// failure responds 503 naming the failing checks and their errors.
func readyzHandler(o *options) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		failed := runReadyChecks(r.Context(), o.readyChecks, o.readyzTimeout, o.readyzRunner)

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

// runReadyChecks runs checks concurrently -- each via rr, which dedupes
// against any run already in flight for the same check name -- and returns
// the name -> error text of every one that failed.
func runReadyChecks(ctx context.Context, checks []ReadyCheck, timeout time.Duration, rr *readyzRunner) map[string]string {
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
			if err := rr.run(ctx, timeout, c); err != nil {
				mu.Lock()
				failed[c.Name] = err.Error()
				mu.Unlock()
			}
		}(c)
	}

	wg.Wait()
	return failed
}
