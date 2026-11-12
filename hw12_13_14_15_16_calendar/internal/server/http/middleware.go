package internalhttp

import (
	"net"
	"net/http"
	"slices"
	"time"
)

type statusRecorder struct {
	http.ResponseWriter
	status int
}

type middleware func(http.Handler) http.Handler

func (r *statusRecorder) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}

func chain(h http.Handler, mws ...middleware) http.Handler {
	for _, mw := range slices.Backward(mws) {
		h = mw(h)
	}
	return h
}

func loggingMiddleware(logg Logger) middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			ip, _, _ := net.SplitHostPort(r.RemoteAddr)
			rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
			next.ServeHTTP(rec, r)
			logg.Info(
				"http request",
				"ip", ip,
				"method", r.Method,
				"path", r.URL.RequestURI(),
				"start", start.Format(time.RFC3339),
				"status", rec.status,
				"duration", time.Since(start),
				"user-agent", r.UserAgent(),
				"proto", r.Proto,
			)
		})
	}
}
