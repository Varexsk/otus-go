package internalhttp

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

type fakeLogger struct {
	entries []map[string]any
}

func (f *fakeLogger) Info(_ string, args ...any) {
	entry := make(map[string]any)
	for i := 0; i+1 < len(args); i += 2 {
		entry[args[i].(string)] = args[i+1]
	}
	f.entries = append(f.entries, entry)
}

func (f *fakeLogger) Error(string, ...any) {}

type stubApp struct {
	Application
}

func TestLoggingMiddleware(t *testing.T) {
	tests := []struct {
		name   string
		code   int
		path   string
		method string
		body   string
	}{
		{"hello get", http.StatusOK, "/hello", http.MethodGet, "Hello!"},
		{"wrong method", http.StatusMethodNotAllowed, "/hello", http.MethodPost, ""},
		{"not found", http.StatusNotFound, "/none", http.MethodGet, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			fake := &fakeLogger{}
			s := &Server{
				logg: fake,
				app:  stubApp{},
			}
			s.routes().ServeHTTP(rec, httptest.NewRequest(tt.method, tt.path, nil))

			if rec.Code != tt.code {
				t.Errorf("expected status code %d, got %d", tt.code, rec.Code)
			}

			if tt.body != "" && rec.Body.String() != tt.body {
				t.Errorf("expected body %q, got %q", tt.body, rec.Body.String())
			}

			if len(fake.entries) != 1 {
				t.Fatalf("expected 1 log entry, got %d", len(fake.entries))
			}
			entry := fake.entries[0]

			if entry["status"] != tt.code {
				t.Errorf("logged status = %v, want %d", entry["status"], tt.code)
			}
			if entry["ip"] != "192.0.2.1" {
				t.Errorf("logged ip = %q, want %q", entry["ip"], "192.0.2.1")
			}
			if entry["method"] != tt.method {
				t.Errorf("logged method = %q, want %q", entry["method"], tt.method)
			}
			if entry["path"] != tt.path {
				t.Errorf("logged path = %q, want %q", entry["path"], tt.path)
			}
		})
	}
}
