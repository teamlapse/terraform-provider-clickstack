// Copyright (c) Lapse Technologies, Inc.
// SPDX-License-Identifier: MPL-2.0

package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
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

// --- Auth ---

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

// --- Error handling ---

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

// --- Dashboards ---

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

// --- Alerts ---

func TestClient_CreateAlert(t *testing.T) {
	name := "High Errors"
	c := testServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		jsonResponse(t, w, 200, Alert{
			ID:            "alert-1",
			Name:          &name,
			Source:        "saved_search",
			Threshold:     100,
			ThresholdType: "above",
			Interval:      "5m",
			State:         "OK",
			Channel:       AlertChannel{Type: "email"},
		})
	}))

	a, err := c.CreateAlert(context.Background(), Alert{
		Name:          &name,
		Source:        "saved_search",
		Threshold:     100,
		ThresholdType: "above",
		Interval:      "5m",
		Channel:       AlertChannel{Type: "email", EmailRecipients: []string{"test@example.com"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if a.ID != "alert-1" {
		t.Errorf("expected id alert-1, got %s", a.ID)
	}
}

func TestClient_GetAlert(t *testing.T) {
	c := testServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		jsonResponse(t, w, 200, Alert{ID: "alert-1", Source: "tile", State: "OK"})
	}))

	a, err := c.GetAlert(context.Background(), "alert-1")
	if err != nil {
		t.Fatal(err)
	}
	if a.State != "OK" {
		t.Errorf("expected state OK, got %q", a.State)
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

// --- Saved Searches ---

func TestClient_CreateSavedSearch(t *testing.T) {
	c := testServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		jsonResponse(t, w, 200, SavedSearch{ID: "ss-1", Name: "Errors", Query: "level:error", Source: "log"})
	}))

	s, err := c.CreateSavedSearch(context.Background(), SavedSearch{Name: "Errors", Query: "level:error", Source: "log"})
	if err != nil {
		t.Fatal(err)
	}
	if s.ID != "ss-1" {
		t.Errorf("expected id ss-1, got %s", s.ID)
	}
}

func TestClient_GetSavedSearch(t *testing.T) {
	c := testServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		jsonResponse(t, w, 200, SavedSearch{ID: "ss-1", Name: "Errors", Query: "level:error", Source: "log"})
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
		jsonResponse(t, w, 200, nil)
	}))

	if err := c.DeleteSavedSearch(context.Background(), "ss-1"); err != nil {
		t.Fatal(err)
	}
}

// --- Sources ---

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

// --- Webhooks ---

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
