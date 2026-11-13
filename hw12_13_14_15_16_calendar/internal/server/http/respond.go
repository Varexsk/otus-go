package internalhttp

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/varexsk/hw12_13_14_15_calendar/internal/app"
	"github.com/varexsk/hw12_13_14_15_calendar/internal/storage"
)

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func (s *Server) writeError(w http.ResponseWriter, err error) {
	var code int
	switch {
	case errors.Is(err, errBadRequest), errors.Is(err, app.ErrInvalidEvent):
		code = http.StatusBadRequest
	case errors.Is(err, storage.ErrEventNotFound):
		code = http.StatusNotFound
	case errors.Is(err, storage.ErrEventAlreadyExist), errors.Is(err, storage.ErrDateBusy):
		code = http.StatusConflict
	default:
		s.logg.Error("internal error", "error", err)
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "internal error"})
		return
	}
	writeJSON(w, code, errorResponse{Error: err.Error()})
}
