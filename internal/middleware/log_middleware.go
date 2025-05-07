package middleware

import (
	"net/http"
	"time"

	"github.com/sirupsen/logrus"
)

// LogMiddleware logs information about each request
func LogMiddleware(logger *logrus.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			
			// Create a response wrapper to capture the status code
			rw := newStatusResponseWriter(w)
			
			// Process the request
			next.ServeHTTP(rw, r)
			
			// Log the request details
			duration := time.Since(start)
			logger.WithFields(logrus.Fields{
				"method":     r.Method,
				"path":       r.URL.Path,
				"status":     rw.status,
				"duration":   duration.String(),
				"user_agent": r.UserAgent(),
				"ip":         r.RemoteAddr,
			}).Info("HTTP request")
		})
	}
}

// statusResponseWriter is a custom response writer that captures the status code
type statusResponseWriter struct {
	http.ResponseWriter
	status int
}

// newStatusResponseWriter creates a new statusResponseWriter
func newStatusResponseWriter(w http.ResponseWriter) *statusResponseWriter {
	return &statusResponseWriter{
		ResponseWriter: w,
		status:         http.StatusOK, // Default status
	}
}

// WriteHeader captures the status code and forwards it to the wrapped ResponseWriter
func (rw *statusResponseWriter) WriteHeader(status int) {
	rw.status = status
	rw.ResponseWriter.WriteHeader(status)
}

// Write forwards the write to the wrapped ResponseWriter
func (rw *statusResponseWriter) Write(b []byte) (int, error) {
	return rw.ResponseWriter.Write(b)
}

// Header forwards the header to the wrapped ResponseWriter
func (rw *statusResponseWriter) Header() http.Header {
	return rw.ResponseWriter.Header()
}