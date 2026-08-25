package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHealthReportsBuildMetadata(t *testing.T) {
	version, commit = "1.2.3", "abc1234"

	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(health{Status: "ok", Version: version, Commit: commit})
	})

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/health", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var got health
	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatalf("decoding body: %v", err)
	}

	// The point of the endpoint is telling you which build answered, so an
	// empty version is a failure even though the request succeeded.
	if got.Version != "1.2.3" || got.Commit != "abc1234" {
		t.Errorf("health = %+v, want version 1.2.3 and commit abc1234", got)
	}
}
