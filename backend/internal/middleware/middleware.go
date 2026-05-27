package middleware

import (
	"net/http"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-contrib/requestid"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/bongcloud/chess/pkg/response"
)

// RequestLogger returns a Gin middleware that logs every request using
// structured Zap fields. Skips logging for the /health endpoint.
func RequestLogger(log *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.URL.Path == "/health" {
			c.Next()
			return
		}

		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery

		c.Next()

		latency := time.Since(start)
		status := c.Writer.Status()

		fields := []zap.Field{
			zap.String("request_id", requestid.Get(c)),
			zap.String("method", c.Request.Method),
			zap.String("path", path),
			zap.String("query", query),
			zap.Int("status", status),
			zap.Duration("latency", latency),
			zap.String("ip", c.ClientIP()),
			zap.String("user_agent", c.Request.UserAgent()),
		}

		switch {
		case status >= http.StatusInternalServerError:
			log.Error("server error", fields...)
		case status >= http.StatusBadRequest:
			log.Warn("client error", fields...)
		default:
			log.Info("request", fields...)
		}
	}
}

// Recovery returns a Gin middleware that catches panics, logs them with a
// full stack trace, and returns a clean 500 to the client.
func Recovery(log *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				log.Error("panic recovered",
					zap.Any("error", err),
					zap.String("request_id", requestid.Get(c)),
					zap.String("path", c.Request.URL.Path),
				)
				c.Abort()
				response.InternalServerError(c)
			}
		}()
		c.Next()
	}
}

// CORS returns a permissive CORS middleware suitable for development.
// For production, restrict AllowOrigins to known frontend domains.
func CORS(allowedOrigins []string) gin.HandlerFunc {
	cfg := cors.Config{
		AllowOrigins:     allowedOrigins,
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization", "X-Request-ID"},
		ExposeHeaders:    []string{"X-Request-ID"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}
	return cors.New(cfg)
}

// RequestID injects a unique X-Request-ID header into every request.
// Downstream handlers and logs reference this ID for traceability.
func RequestID() gin.HandlerFunc {
	return requestid.New()
}