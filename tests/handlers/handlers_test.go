package handlers_test

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/JaimeStill/herald/pkg/handlers"
)

func TestRespondJSON(t *testing.T) {
	tests := []struct {
		name       string
		status     int
		data       any
		wantStatus int
	}{
		{
			name:       "200 with map",
			status:     http.StatusOK,
			data:       map[string]string{"key": "value"},
			wantStatus: http.StatusOK,
		},
		{
			name:       "201 with struct",
			status:     http.StatusCreated,
			data:       struct{ ID int }{ID: 42},
			wantStatus: http.StatusCreated,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			handlers.RespondJSON(rec, tt.status, tt.data)

			res := rec.Result()
			defer res.Body.Close()

			if res.StatusCode != tt.wantStatus {
				t.Errorf("status: got %d, want %d", res.StatusCode, tt.wantStatus)
			}
			if ct := res.Header.Get("Content-Type"); ct != "application/json" {
				t.Errorf("content-type: got %s", ct)
			}

			body, _ := io.ReadAll(res.Body)
			var parsed map[string]any
			if err := json.Unmarshal(body, &parsed); err != nil {
				t.Fatalf("unmarshal failed: %v", err)
			}
		})
	}
}

func TestRespondError(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	rec := httptest.NewRecorder()

	handlers.RespondError(rec, logger, http.StatusBadRequest, errors.New("invalid input"))

	res := rec.Result()
	defer res.Body.Close()

	if res.StatusCode != http.StatusBadRequest {
		t.Errorf("status: got %d, want 400", res.StatusCode)
	}
	if ct := res.Header.Get("Content-Type"); ct != "application/json" {
		t.Errorf("content-type: got %s", ct)
	}

	body, _ := io.ReadAll(res.Body)
	var parsed map[string]string
	if err := json.Unmarshal(body, &parsed); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if parsed["error"] != "invalid input" {
		t.Errorf("error: got %s, want invalid input", parsed["error"])
	}
}
