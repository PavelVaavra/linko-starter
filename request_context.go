package main

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

type spyReadCloser struct {
	io.ReadCloser
	bytesRead int
}

func (r *spyReadCloser) Read(p []byte) (int, error) {
	n, err := r.ReadCloser.Read(p)
	r.bytesRead += n
	return n, err
}

type spyResponseWriter struct {
	http.ResponseWriter
	bytesWritten int
	statusCode   int
}

func (w *spyResponseWriter) Write(p []byte) (int, error) {
	if w.statusCode == 0 {
		w.statusCode = http.StatusOK
	}
	n, err := w.ResponseWriter.Write(p)
	w.bytesWritten += n
	return n, err
}

func (w *spyResponseWriter) WriteHeader(statusCode int) {
	w.statusCode = statusCode
	w.ResponseWriter.WriteHeader(statusCode)
}

const logContextKey contextKey = "log_context"

type LogContext struct {
	Username string
	Error    error
}

func httpError(ctx context.Context, w http.ResponseWriter, status int, err error) {
	if logCtx, ok := ctx.Value(logContextKey).(*LogContext); ok {
		logCtx.Error = err
	}
	msg := err.Error()
	switch status {
	case http.StatusUnauthorized, http.StatusForbidden, http.StatusInternalServerError:
		msg = http.StatusText(status)
	}
	http.Error(w, msg, status)
}

func redactIP(ip string) string {
	// Implement your IP redaction logic here
	host, _, err := net.SplitHostPort(ip)
	if err != nil {
		return ""
	}
	addr := net.ParseIP(host)
	if addr == nil {
		return ""
	}
	if addr := addr.To4(); addr != nil {
		return fmt.Sprintf("%d.%d.%d.x", addr[0], addr[1], addr[2])
	}
	return ip
}

// httpRequestsTotal counts requests by method, path and status.
var httpRequestsTotal = promauto.NewCounterVec(
	prometheus.CounterOpts{
		Name: "http_requests_total",
		Help: "Total number of HTTP requests.",
	},
	[]string{"method", "path", "status"},
)

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}
