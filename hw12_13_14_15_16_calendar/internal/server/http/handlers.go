package internalhttp

import (
	"net/http"
)

func (s *Server) routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /hello", s.handleHello)
	return chain(mux,
		loggingMiddleware(s.logg),
	)
}

func (s *Server) handleHello(w http.ResponseWriter, _ *http.Request) {
	w.Write([]byte("Hello!"))
}
