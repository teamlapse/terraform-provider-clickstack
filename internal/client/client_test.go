// Copyright (c) Lapse Technologies, Inc.
// SPDX-License-Identifier: MPL-2.0

package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func testServer(t *testing.T, handler http.Handler) *Client {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	return NewClient(srv.URL, "org-1", "svc-1", "key-id", "key-secret")
}

func jsonResponse(t *testing.T, w http.ResponseWriter, status int, result any) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	resp := APIResponse[any]{Status: status, RequestID: "req-123", Result: result}
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		t.Fatal(err)
	}
}

func jsonError(t *testing.T, w http.ResponseWriter, status int, msg string) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	resp := APIResponse[any]{Status: status, RequestID: "req-err", Error: msg}
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		t.Fatal(err)
	}
}

func TestClient_RetriesOn429(t *testing.T) {
	var attempts atomic.Int32
	c := testServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		n := attempts.Add(1)
		if n < 3 {
			w.Header().Set("Retry-After", "0")
			jsonError(t, w, http.StatusTooManyRequests, "TOO_MANY_REQUESTS")
			return
		}
		jsonResponse(t, w, 200, []Dashboard{{ID: "dash-1", Name: "OK"}})
	}))

	dashs, err := c.ListDashboards(context.Background())
	if err != nil {
		t.Fatalf("expected success after retries, got: %v", err)
	}
	if len(dashs) != 1 {
		t.Errorf("expected 1 dashboard, got %d", len(dashs))
	}
	if attempts.Load() != 3 {
		t.Errorf("expected 3 attempts, got %d", attempts.Load())
	}
}

func TestClient_ExhaustsRetries(t *testing.T) {
	var attempts atomic.Int32
	c := testServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		attempts.Add(1)
		w.Header().Set("Retry-After", "0")
		jsonError(t, w, http.StatusTooManyRequests, "TOO_MANY_REQUESTS")
	}))

	_, err := c.ListDashboards(context.Background())
	if err == nil {
		t.Fatal("expected error after exhausting retries")
	}
	if attempts.Load() != maxRetries+1 {
		t.Errorf("expected %d attempts, got %d", maxRetries+1, attempts.Load())
	}
}

func TestClient_RespectsRetryAfterHeader(t *testing.T) {
	var attempts atomic.Int32
	var firstCallTime time.Time
	retryAfterSecs := "0"

	c := testServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		n := attempts.Add(1)
		if n == 1 {
			firstCallTime = time.Now()
			w.Header().Set("Retry-After", retryAfterSecs)
			jsonError(t, w, http.StatusTooManyRequests, "TOO_MANY_REQUESTS")
			return
		}
		jsonResponse(t, w, 200, []Dashboard{})
	}))

	_, err := c.ListDashboards(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if firstCallTime.IsZero() {
		t.Fatal("first call time not recorded")
	}
	_ = firstCallTime
}

func TestRetryAfterDelay_Seconds(t *testing.T) {
	w := httptest.NewRecorder()
	w.Header().Set("Retry-After", "2")
	resp := w.Result()

	got := retryAfterDelay(resp, 0)
	if got != 2*time.Second {
		t.Errorf("expected 2s, got %v", got)
	}
}

func TestRetryAfterDelay_HTTPDate(t *testing.T) {
	w := httptest.NewRecorder()
	future := time.Now().Add(3 * time.Second)
	w.Header().Set("Retry-After", future.UTC().Format(http.TimeFormat))
	resp := w.Result()

	got := retryAfterDelay(resp, 0)
	if got < 2*time.Second || got > 4*time.Second {
		t.Errorf("expected ~3s, got %v", got)
	}
}

func TestRetryAfterDelay_ExponentialFallback(t *testing.T) {
	cases := []struct {
		attempt  int
		expected time.Duration
	}{
		{0, 500 * time.Millisecond},
		{1, 1 * time.Second},
		{2, 2 * time.Second},
		{3, 4 * time.Second},
	}
	for _, tc := range cases {
		t.Run(fmt.Sprintf("attempt_%d", tc.attempt), func(t *testing.T) {
			w := httptest.NewRecorder()
			resp := w.Result()
			got := retryAfterDelay(resp, tc.attempt)
			if got != tc.expected {
				t.Errorf("attempt %d: expected %v, got %v", tc.attempt, tc.expected, got)
			}
		})
	}
}

func TestRateLimiter_ContextCancellation(t *testing.T) {
	rl := &rateLimiter{
		tokens:   0,
		max:      1,
		rate:     0.001,
		lastFill: time.Now(),
	}
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	err := rl.wait(ctx)
	if err == nil {
		t.Fatal("expected context cancellation error")
	}
}

func TestClient_BasicAuth(t *testing.T) {
	var gotUser, gotPass string
	c := testServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUser, gotPass, _ = r.BasicAuth()
		jsonResponse(t, w, 200, []Source{})
	}))

	_, _ = c.ListSources(context.Background())

	if gotUser != "key-id" || gotPass != "key-secret" {
		t.Errorf("expected basic auth key-id:key-secret, got %s:%s", gotUser, gotPass)
	}
}

func TestClient_BasePath(t *testing.T) {
	var gotPath string
	c := testServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		jsonResponse(t, w, 200, []Source{})
	}))

	_, _ = c.ListSources(context.Background())

	expected := "/v1/organizations/org-1/services/svc-1/clickstack/sources"
	if gotPath != expected {
		t.Errorf("expected path %s, got %s", expected, gotPath)
	}
}

func TestClient_NotFoundError(t *testing.T) {
	c := testServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		jsonError(t, w, 404, "dashboard not found")
	}))

	_, err := c.GetDashboard(context.Background(), "missing-id")

	if !IsNotFound(err) {
		t.Fatalf("expected NotFoundError, got %v", err)
	}
}

func TestClient_APIError(t *testing.T) {
	c := testServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		jsonError(t, w, 500, "internal server error")
	}))

	_, err := c.ListDashboards(context.Background())

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if IsNotFound(err) {
		t.Fatal("expected non-404 error")
	}
}

func TestIsNotFound_Nil(t *testing.T) {
	if IsNotFound(nil) {
		t.Error("IsNotFound(nil) should be false")
	}
}

func TestClient_CreateDashboard(t *testing.T) {
	c := testServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		var d Dashboard
		if err := json.NewDecoder(r.Body).Decode(&d); err != nil {
			t.Fatal(err)
		}
		if d.Name != "Test Dashboard" {
			t.Errorf("expected name 'Test Dashboard', got %q", d.Name)
		}
		d.ID = "dash-1"
		jsonResponse(t, w, 200, d)
	}))

	d, err := c.CreateDashboard(context.Background(), Dashboard{
		Name:  "Test Dashboard",
		Tiles: []Tile{{Name: "Tile 1", X: 0, Y: 0, W: 12, H: 4}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if d.ID != "dash-1" {
		t.Errorf("expected id dash-1, got %s", d.ID)
	}
}

func TestClient_GetDashboard(t *testing.T) {
	c := testServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/organizations/org-1/services/svc-1/clickstack/dashboards/dash-1" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		jsonResponse(t, w, 200, Dashboard{ID: "dash-1", Name: "My Dash"})
	}))

	d, err := c.GetDashboard(context.Background(), "dash-1")
	if err != nil {
		t.Fatal(err)
	}
	if d.Name != "My Dash" {
		t.Errorf("expected name 'My Dash', got %q", d.Name)
	}
}

func TestClient_UpdateDashboard(t *testing.T) {
	c := testServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("expected PUT, got %s", r.Method)
		}
		jsonResponse(t, w, 200, Dashboard{ID: "dash-1", Name: "Updated"})
	}))

	d, err := c.UpdateDashboard(context.Background(), "dash-1", Dashboard{Name: "Updated"})
	if err != nil {
		t.Fatal(err)
	}
	if d.Name != "Updated" {
		t.Errorf("expected name 'Updated', got %q", d.Name)
	}
}

func TestClient_DeleteDashboard(t *testing.T) {
	c := testServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		jsonResponse(t, w, 200, nil)
	}))

	if err := c.DeleteDashboard(context.Background(), "dash-1"); err != nil {
		t.Fatal(err)
	}
}

func TestClient_ListDashboards(t *testing.T) {
	c := testServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		jsonResponse(t, w, 200, []Dashboard{
			{ID: "dash-1", Name: "First"},
			{ID: "dash-2", Name: "Second"},
		})
	}))

	dashboards, err := c.ListDashboards(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(dashboards) != 2 {
		t.Errorf("expected 2 dashboards, got %d", len(dashboards))
	}
}

func TestClient_CreateAlert(t *testing.T) {
	name := "High Errors"
	numConsecutiveWindows := int64(3)
	c := testServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		var got Alert
		if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
			t.Fatal(err)
		}
		if got.NumConsecutiveWindows == nil || *got.NumConsecutiveWindows != 3 {
			t.Errorf("expected numConsecutiveWindows 3, got %v", got.NumConsecutiveWindows)
		}
		jsonResponse(t, w, 200, Alert{
			ID:                    "alert-1",
			Name:                  &name,
			Source:                "saved_search",
			Threshold:             100,
			ThresholdType:         "above",
			Interval:              "5m",
			NumConsecutiveWindows: &numConsecutiveWindows,
			State:                 "OK",
			Channel:               AlertChannel{Type: "email"},
		})
	}))

	a, err := c.CreateAlert(context.Background(), Alert{
		Name:                  &name,
		Source:                "saved_search",
		Threshold:             100,
		ThresholdType:         "above",
		Interval:              "5m",
		NumConsecutiveWindows: &numConsecutiveWindows,
		Channel:               AlertChannel{Type: "email", EmailRecipients: []string{"test@example.com"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if a.ID != "alert-1" {
		t.Errorf("expected id alert-1, got %s", a.ID)
	}
	if a.NumConsecutiveWindows == nil || *a.NumConsecutiveWindows != 3 {
		t.Errorf("expected response numConsecutiveWindows 3, got %v", a.NumConsecutiveWindows)
	}
}

func TestAlert_NumConsecutiveWindowsJSON(t *testing.T) {
	numConsecutiveWindows := int64(3)
	payload, err := json.Marshal(Alert{
		Source:                "tile",
		NumConsecutiveWindows: &numConsecutiveWindows,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(payload), `"numConsecutiveWindows":3`) {
		t.Fatalf("expected numConsecutiveWindows JSON field, got %s", payload)
	}

	unsetPayload, err := json.Marshal(Alert{Source: "tile"})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(unsetPayload), "numConsecutiveWindows") {
		t.Fatalf("expected unset numConsecutiveWindows to be omitted, got %s", unsetPayload)
	}
}

func TestClient_GetAlert(t *testing.T) {
	numConsecutiveWindows := int64(3)
	c := testServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		jsonResponse(t, w, 200, Alert{
			ID:                    "alert-1",
			Source:                "tile",
			NumConsecutiveWindows: &numConsecutiveWindows,
			State:                 "OK",
		})
	}))

	a, err := c.GetAlert(context.Background(), "alert-1")
	if err != nil {
		t.Fatal(err)
	}
	if a.State != "OK" {
		t.Errorf("expected state OK, got %q", a.State)
	}
	if a.NumConsecutiveWindows == nil || *a.NumConsecutiveWindows != 3 {
		t.Errorf("expected numConsecutiveWindows 3, got %v", a.NumConsecutiveWindows)
	}
}

func TestClient_UpdateAlert(t *testing.T) {
	numConsecutiveWindows := int64(5)
	c := testServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("expected PUT, got %s", r.Method)
		}
		if r.URL.Path != "/v1/organizations/org-1/services/svc-1/clickstack/alerts/alert-1" {
			t.Errorf("unexpected path %s", r.URL.Path)
		}
		var got Alert
		if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
			t.Fatal(err)
		}
		if got.NumConsecutiveWindows == nil || *got.NumConsecutiveWindows != 5 {
			t.Errorf("expected numConsecutiveWindows 5, got %v", got.NumConsecutiveWindows)
		}
		got.ID = "alert-1"
		jsonResponse(t, w, 200, got)
	}))

	a, err := c.UpdateAlert(context.Background(), "alert-1", Alert{
		Source:                "tile",
		Threshold:             100,
		ThresholdType:         "above",
		Interval:              "5m",
		NumConsecutiveWindows: &numConsecutiveWindows,
		Channel:               AlertChannel{Type: "email"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if a.NumConsecutiveWindows == nil || *a.NumConsecutiveWindows != 5 {
		t.Errorf("expected response numConsecutiveWindows 5, got %v", a.NumConsecutiveWindows)
	}
}

func TestClient_DeleteAlert(t *testing.T) {
	c := testServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		jsonResponse(t, w, 200, nil)
	}))

	if err := c.DeleteAlert(context.Background(), "alert-1"); err != nil {
		t.Fatal(err)
	}
}

func TestClient_CreateSavedSearch(t *testing.T) {
	c := testServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/v1/organizations/org-1/services/svc-1/clickstack/saved-searches" {
			t.Errorf("unexpected path %s", r.URL.Path)
		}
		var got SavedSearch
		if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
			t.Fatal(err)
		}
		if got.SourceID != "src-logs" || got.Where != "SeverityText:ERROR" || got.WhereLanguage != "lucene" {
			t.Errorf("unexpected request: %+v", got)
		}
		jsonResponse(t, w, 200, SavedSearch{ID: "ss-1", Name: "Errors", SourceID: "src-logs", Where: "SeverityText:ERROR", WhereLanguage: "lucene"})
	}))

	s, err := c.CreateSavedSearch(context.Background(), SavedSearch{Name: "Errors", SourceID: "src-logs", Where: "SeverityText:ERROR", WhereLanguage: "lucene"})
	if err != nil {
		t.Fatal(err)
	}
	if s.ID != "ss-1" {
		t.Errorf("expected id ss-1, got %s", s.ID)
	}
}

func TestClient_GetSavedSearch(t *testing.T) {
	c := testServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/organizations/org-1/services/svc-1/clickstack/saved-searches/ss-1" {
			t.Errorf("unexpected path %s", r.URL.Path)
		}
		jsonResponse(t, w, 200, SavedSearch{ID: "ss-1", Name: "Errors", SourceID: "src-logs"})
	}))

	s, err := c.GetSavedSearch(context.Background(), "ss-1")
	if err != nil {
		t.Fatal(err)
	}
	if s.Name != "Errors" {
		t.Errorf("expected name 'Errors', got %q", s.Name)
	}
}

func TestClient_DeleteSavedSearch(t *testing.T) {
	c := testServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		if r.URL.Path != "/v1/organizations/org-1/services/svc-1/clickstack/saved-searches/ss-1" {
			t.Errorf("unexpected path %s", r.URL.Path)
		}
		jsonResponse(t, w, 200, nil)
	}))

	if err := c.DeleteSavedSearch(context.Background(), "ss-1"); err != nil {
		t.Fatal(err)
	}
}

func TestClient_UpdateSavedSearch(t *testing.T) {
	c := testServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("expected PUT, got %s", r.Method)
		}
		if r.URL.Path != "/v1/organizations/org-1/services/svc-1/clickstack/saved-searches/ss-1" {
			t.Errorf("unexpected path %s", r.URL.Path)
		}
		jsonResponse(t, w, 200, SavedSearch{ID: "ss-1", Name: "Errors", SourceID: "src-logs"})
	}))

	search, err := c.UpdateSavedSearch(context.Background(), "ss-1", SavedSearch{Name: "Errors", SourceID: "src-logs"})
	if err != nil {
		t.Fatal(err)
	}
	if search.ID != "ss-1" {
		t.Errorf("expected id ss-1, got %s", search.ID)
	}
}

func TestClient_ListSources(t *testing.T) {
	c := testServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		jsonResponse(t, w, 200, []Source{
			{ID: "src-1", Name: "logs", Kind: "log"},
			{ID: "src-2", Name: "traces", Kind: "trace"},
		})
	}))

	sources, err := c.ListSources(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(sources) != 2 {
		t.Errorf("expected 2 sources, got %d", len(sources))
	}
	if sources[0].Kind != "log" {
		t.Errorf("expected kind 'log', got %q", sources[0].Kind)
	}
}

func TestClient_ListWebhooks(t *testing.T) {
	c := testServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		jsonResponse(t, w, 200, []Webhook{
			{ID: "wh-1", Name: "Slack", Service: "slack", URL: "https://hooks.slack.com/xxx"},
		})
	}))

	webhooks, err := c.ListWebhooks(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(webhooks) != 1 {
		t.Errorf("expected 1 webhook, got %d", len(webhooks))
	}
	if webhooks[0].Service != "slack" {
		t.Errorf("expected service 'slack', got %q", webhooks[0].Service)
	}
}
