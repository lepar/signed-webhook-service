package http

import (
	"context"
	"net/http"
	"time"

	"github.com/google/uuid"
	"kii.com/internal/infrastructure/logger"
)

// responseWriter wraps http.ResponseWriter to capture status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// RequestIDMiddleware adds a request ID to each request
func RequestIDMiddleware(next http.HandlerFunc, logger logger.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		requestID := r.Header.Get("X-Request-ID")
		if requestID == "" {
			requestID = uuid.New().String()
		}

		ctx := context.WithValue(r.Context(), "request_id", requestID)
		r = r.WithContext(ctx)

		w.Header().Set("X-Request-ID", requestID)

		// Create logger with request ID
		requestLogger := logger.WithRequestID(requestID)
		ctx = context.WithValue(ctx, "logger", requestLogger)
		r = r.WithContext(ctx)

		next(w, r)
	}
}

// LoggingMiddleware logs request details
func LoggingMiddleware(next http.HandlerFunc, logger logger.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		requestID := r.Context().Value("request_id").(string)
		requestLogger := logger.WithRequestID(requestID)

		requestLogger.LogInfo(r.Context(), "Incoming request",
			"method", r.Method,
			"path", r.URL.Path,
			"remote_addr", r.RemoteAddr)

		// Wrap response writer to capture status code
		wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		next(wrapped, r)

		duration := time.Since(start)
		requestLogger.LogInfo(r.Context(), "Request completed",
			"method", r.Method,
			"path", r.URL.Path,
			"status", wrapped.statusCode,
			"duration_ms", duration.Milliseconds())
	}
}
