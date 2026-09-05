package chimux

import (
	"fmt"
	"net/http"
	"time"
)

// healthCheck is a liveness probe: it always answers 200 regardless of the
// state of any downstream dependency. Liveness failing tells an orchestrator
// to restart the process, which does not help when the process itself is
// fine but, say, a database it depends on is unreachable -- that case
// belongs to readiness (see WithReadyz), which is allowed to fail so the
// orchestrator stops routing traffic here without killing the process.
func healthCheck(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = fmt.Fprintf(w, `{"status":"healthy","timestamp":"%s"}`, time.Now().UTC().Format(time.RFC3339))
}
