package gin

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/meysam81/x/logging"
)

func zerologMiddleware(logger *logging.Logger) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		start := time.Now()

		ctx.Next()

		end := time.Since(start).Abs().String()

		status := ctx.Writer.Status()

		var event *logging.Event
		if status > 400 {
			event = logger.Error()
		} else {
			event = logger.Info()
		}

		event.
			Str("method", ctx.Request.Method).
			Str("path", ctx.Request.URL.Path).
			Int("status", status).
			Interface("params", ctx.Request.URL.Query()).
			Str("user-agent", ctx.Request.UserAgent()).
			Int("response-size", ctx.Writer.Size()).
			Str("latency", end).
			Send()
	}
}
