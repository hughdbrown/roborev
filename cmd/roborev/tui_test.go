package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/wesm/roborev/internal/storage"
)

func TestTUIFetchJobsSuccess(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/jobs" {
			t.Errorf("Expected /api/jobs, got %s", r.URL.Path)
		}
		jobs := []storage.ReviewJob{{ID: 1, GitRef: "abc123", Agent: "test"}}
		json.NewEncoder(w).Encode(map[string]interface{}{"jobs": jobs})
	}))
	defer ts.Close()

	m := newTuiModel(ts.URL)
	cmd := m.fetchJobs()
	msg := cmd()

	jobs, ok := msg.(tuiJobsMsg)
	if !ok {
		t.Fatalf("Expected tuiJobsMsg, got %T: %v", msg, msg)
	}
	if len(jobs) != 1 || jobs[0].ID != 1 {
		t.Errorf("Unexpected jobs: %+v", jobs)
	}
}

func TestTUIFetchJobsError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer ts.Close()

	m := newTuiModel(ts.URL)
	cmd := m.fetchJobs()
	msg := cmd()

	_, ok := msg.(tuiErrMsg)
	if !ok {
		t.Fatalf("Expected tuiErrMsg for 500, got %T: %v", msg, msg)
	}
}

func TestTUIFetchReviewNotFound(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer ts.Close()

	m := newTuiModel(ts.URL)
	cmd := m.fetchReview(999)
	msg := cmd()

	errMsg, ok := msg.(tuiErrMsg)
	if !ok {
		t.Fatalf("Expected tuiErrMsg for 404, got %T: %v", msg, msg)
	}
	if errMsg.Error() != "no review found" {
		t.Errorf("Expected 'no review found', got: %v", errMsg)
	}
}

func TestTUIFetchReviewServerError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer ts.Close()

	m := newTuiModel(ts.URL)
	cmd := m.fetchReview(1)
	msg := cmd()

	errMsg, ok := msg.(tuiErrMsg)
	if !ok {
		t.Fatalf("Expected tuiErrMsg for 500, got %T: %v", msg, msg)
	}
	if errMsg.Error() != "fetch review: 500 Internal Server Error" {
		t.Errorf("Expected status in error, got: %v", errMsg)
	}
}

func TestTUIAddressReviewSuccess(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("Expected POST, got %s", r.Method)
		}
		var req map[string]interface{}
		json.NewDecoder(r.Body).Decode(&req)
		if req["review_id"].(float64) != 42 {
			t.Errorf("Expected review_id 42, got %v", req["review_id"])
		}
		if req["addressed"].(bool) != true {
			t.Errorf("Expected addressed true, got %v", req["addressed"])
		}
		json.NewEncoder(w).Encode(map[string]bool{"success": true})
	}))
	defer ts.Close()

	m := newTuiModel(ts.URL)
	cmd := m.addressReview(42, true)
	msg := cmd()

	addressed, ok := msg.(tuiAddressedMsg)
	if !ok {
		t.Fatalf("Expected tuiAddressedMsg, got %T: %v", msg, msg)
	}
	if !bool(addressed) {
		t.Error("Expected addressed to be true")
	}
}

func TestTUIAddressReviewNotFound(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer ts.Close()

	m := newTuiModel(ts.URL)
	cmd := m.addressReview(999, true)
	msg := cmd()

	errMsg, ok := msg.(tuiErrMsg)
	if !ok {
		t.Fatalf("Expected tuiErrMsg for 404, got %T: %v", msg, msg)
	}
	if errMsg.Error() != "review not found" {
		t.Errorf("Expected 'review not found', got: %v", errMsg)
	}
}

func TestTUIToggleAddressedForJobSuccess(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/review" {
			review := storage.Review{ID: 10, Addressed: false}
			json.NewEncoder(w).Encode(review)
		} else if r.URL.Path == "/api/review/address" {
			json.NewEncoder(w).Encode(map[string]bool{"success": true})
		}
	}))
	defer ts.Close()

	m := newTuiModel(ts.URL)
	currentState := false
	cmd := m.toggleAddressedForJob(1, &currentState)
	msg := cmd()

	addressed, ok := msg.(tuiAddressedMsg)
	if !ok {
		t.Fatalf("Expected tuiAddressedMsg, got %T: %v", msg, msg)
	}
	if !bool(addressed) {
		t.Error("Expected toggled state to be true (was false)")
	}
}

func TestTUIToggleAddressedNoReview(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer ts.Close()

	m := newTuiModel(ts.URL)
	cmd := m.toggleAddressedForJob(999, nil)
	msg := cmd()

	errMsg, ok := msg.(tuiErrMsg)
	if !ok {
		t.Fatalf("Expected tuiErrMsg, got %T: %v", msg, msg)
	}
	if errMsg.Error() != "no review for this job" {
		t.Errorf("Expected 'no review for this job', got: %v", errMsg)
	}
}

func TestTUIHTTPTimeout(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Delay longer than client timeout
		time.Sleep(200 * time.Millisecond)
		json.NewEncoder(w).Encode(map[string]interface{}{"jobs": []storage.ReviewJob{}})
	}))
	defer ts.Close()

	m := newTuiModel(ts.URL)
	// Override with short timeout for test
	m.client.Timeout = 50 * time.Millisecond

	cmd := m.fetchJobs()
	msg := cmd()

	_, ok := msg.(tuiErrMsg)
	if !ok {
		t.Fatalf("Expected tuiErrMsg for timeout, got %T: %v", msg, msg)
	}
}
