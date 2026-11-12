package internalhttp

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/varexsk/hw12_13_14_15_calendar/internal/storage"
)

const dateLayout = "2006-01-02"

func (s *Server) routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /hello", s.handleHello)

	mux.HandleFunc("POST /events", s.handleCreate)
	mux.HandleFunc("PUT /events/{id}", s.handleUpdate)
	mux.HandleFunc("DELETE /events/{id}", s.handleDelete)

	mux.HandleFunc("GET /events/day", s.handleList(s.app.ListDay))
	mux.HandleFunc("GET /events/week", s.handleList(s.app.ListWeek))
	mux.HandleFunc("GET /events/month", s.handleList(s.app.ListMonth))

	return chain(mux,
		loggingMiddleware(s.logg),
	)
}

func (s *Server) handleHello(w http.ResponseWriter, _ *http.Request) {
	w.Write([]byte("Hello!"))
}

func (s *Server) handleCreate(w http.ResponseWriter, r *http.Request) {
	event, err := decodeEvent(r)
	if err != nil {
		s.writeError(w, err)
		return
	}

	id, err := s.app.CreateEvent(r.Context(), event)
	if err != nil {
		s.writeError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, createEventResponse{ID: id.String()})
}

func (s *Server) handleUpdate(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		s.writeError(w, err)
		return
	}

	event, err := decodeEvent(r)
	if err != nil {
		s.writeError(w, err)
		return
	}

	if err := s.app.UpdateEvent(r.Context(), id, event); err != nil {
		s.writeError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleDelete(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		s.writeError(w, err)
		return
	}

	if err := s.app.DeleteEvent(r.Context(), id); err != nil {
		s.writeError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

type listFunc func(ctx context.Context, date time.Time) ([]storage.Event, error)

func (s *Server) handleList(list listFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		date, err := time.Parse(dateLayout, r.URL.Query().Get("date"))
		if err != nil {
			s.writeError(w, fmt.Errorf("%w: date must be in format YYYY-MM-DD: %w", errBadRequest, err))
			return
		}

		events, err := list(r.Context(), date)
		if err != nil {
			s.writeError(w, err)
			return
		}

		writeJSON(w, http.StatusOK, listEventsResponse{Events: toDTOs(events)})
	}
}

func decodeEvent(r *http.Request) (storage.Event, error) {
	var req eventDTO
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return storage.Event{}, fmt.Errorf("%w: invalid json: %w", errBadRequest, err)
	}
	return req.toEvent()
}

func parseID(r *http.Request) (uuid.UUID, error) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		return uuid.Nil, fmt.Errorf("%w: invalid id: %w", errBadRequest, err)
	}
	return id, nil
}
