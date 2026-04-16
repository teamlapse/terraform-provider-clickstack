// Copyright (c) Lapse Technologies, Inc.
// SPDX-License-Identifier: MPL-2.0

// Package testmock provides an in-memory ClickStack API server for unit tests.
package testmock

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/teamlapse/terraform-provider-clickstack/internal/client"
)

// Server is a mock ClickStack API backed by in-memory maps.
type Server struct {
	*httptest.Server

	mu            sync.Mutex
	nextID        int
	dashboards    map[string]client.Dashboard
	alerts        map[string]client.Alert
	savedSearches map[string]client.SavedSearch
	sources       []client.Source
	webhooks      []client.Webhook
}

// NewServer creates and starts a mock ClickStack API server.
// It returns the server URL and a cleanup function.
func NewServer(t *testing.T) *Server {
	t.Helper()

	s := &Server{
		dashboards:    make(map[string]client.Dashboard),
		alerts:        make(map[string]client.Alert),
		savedSearches: make(map[string]client.SavedSearch),
		sources: []client.Source{
			{ID: "src-log", Name: "Log Source", Kind: "log"},
			{ID: "src-trace", Name: "Trace Source", Kind: "trace"},
			{ID: "src-metric", Name: "Metric Source", Kind: "metric"},
			{ID: "src-session", Name: "Session Source", Kind: "session"},
		},
		webhooks: []client.Webhook{
			{ID: "wh-slack", Name: "Slack Alerts", Service: "slack", URL: "https://hooks.slack.com/test"},
			{ID: "wh-pd", Name: "PagerDuty", Service: "pagerduty_api", URL: "https://events.pagerduty.com/test"},
		},
	}

	mux := http.NewServeMux()

	// The API prefix we expect.
	prefix := "/v1/organizations/test-org/services/test-svc/clickstack"

	mux.HandleFunc(prefix+"/dashboards", func(w http.ResponseWriter, r *http.Request) {
		// Strip trailing slashes for simpler matching.
		switch r.Method {
		case http.MethodGet:
			s.listDashboards(t, w)
		case http.MethodPost:
			s.createDashboard(t, w, r)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})
	mux.HandleFunc(prefix+"/dashboards/", func(w http.ResponseWriter, r *http.Request) {
		id := strings.TrimPrefix(r.URL.Path, prefix+"/dashboards/")
		switch r.Method {
		case http.MethodGet:
			s.getDashboard(t, w, id)
		case http.MethodPut:
			s.updateDashboard(t, w, r, id)
		case http.MethodDelete:
			s.deleteDashboard(t, w, id)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc(prefix+"/alerts", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			s.listAlerts(t, w)
		case http.MethodPost:
			s.createAlert(t, w, r)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})
	mux.HandleFunc(prefix+"/alerts/", func(w http.ResponseWriter, r *http.Request) {
		id := strings.TrimPrefix(r.URL.Path, prefix+"/alerts/")
		switch r.Method {
		case http.MethodGet:
			s.getAlert(t, w, id)
		case http.MethodPut:
			s.updateAlert(t, w, r, id)
		case http.MethodDelete:
			s.deleteAlert(t, w, id)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc(prefix+"/savedSearches", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			s.listSavedSearches(t, w)
		case http.MethodPost:
			s.createSavedSearch(t, w, r)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})
	mux.HandleFunc(prefix+"/savedSearches/", func(w http.ResponseWriter, r *http.Request) {
		id := strings.TrimPrefix(r.URL.Path, prefix+"/savedSearches/")
		switch r.Method {
		case http.MethodGet:
			s.getSavedSearch(t, w, id)
		case http.MethodPut:
			s.updateSavedSearch(t, w, r, id)
		case http.MethodDelete:
			s.deleteSavedSearch(t, w, id)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc(prefix+"/sources", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		writeJSON(t, w, 200, s.sources)
	})

	mux.HandleFunc(prefix+"/webhooks", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		writeJSON(t, w, 200, s.webhooks)
	})

	s.Server = httptest.NewServer(mux)
	t.Cleanup(s.Close)
	return s
}

// ProviderConfig returns an HCL provider block pointing at this mock server.
func (s *Server) ProviderConfig() string {
	return fmt.Sprintf(`
provider "clickstack" {
  base_url        = %q
  organization_id = "test-org"
  service_id      = "test-svc"
  api_key_id      = "test-key"
  api_key_secret  = "test-secret"
}
`, s.URL)
}

func (s *Server) genID() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.nextID++
	return fmt.Sprintf("mock-%d", s.nextID)
}

// --- Dashboard handlers ---

func (s *Server) listDashboards(t *testing.T, w http.ResponseWriter) {
	s.mu.Lock()
	defer s.mu.Unlock()
	dashboards := make([]client.Dashboard, 0, len(s.dashboards))
	for _, d := range s.dashboards {
		dashboards = append(dashboards, d)
	}
	writeJSON(t, w, 200, dashboards)
}

func (s *Server) createDashboard(t *testing.T, w http.ResponseWriter, r *http.Request) {
	var d client.Dashboard
	if err := json.NewDecoder(r.Body).Decode(&d); err != nil {
		writeError(t, w, 400, "invalid request body")
		return
	}
	d.ID = s.genID()
	for i := range d.Tiles {
		if d.Tiles[i].ID == "" {
			d.Tiles[i].ID = s.genID()
		}
	}
	for i := range d.Filters {
		if d.Filters[i].ID == "" {
			d.Filters[i].ID = s.genID()
		}
	}
	if d.SavedQueryLanguage == nil {
		lang := "lucene"
		d.SavedQueryLanguage = &lang
	}
	s.mu.Lock()
	s.dashboards[d.ID] = d
	s.mu.Unlock()
	writeJSON(t, w, 200, d)
}

func (s *Server) getDashboard(t *testing.T, w http.ResponseWriter, id string) {
	s.mu.Lock()
	d, ok := s.dashboards[id]
	s.mu.Unlock()
	if !ok {
		writeError(t, w, 404, "dashboard not found")
		return
	}
	writeJSON(t, w, 200, d)
}

func (s *Server) updateDashboard(t *testing.T, w http.ResponseWriter, r *http.Request, id string) {
	s.mu.Lock()
	_, ok := s.dashboards[id]
	s.mu.Unlock()
	if !ok {
		writeError(t, w, 404, "dashboard not found")
		return
	}
	var d client.Dashboard
	if err := json.NewDecoder(r.Body).Decode(&d); err != nil {
		writeError(t, w, 400, "invalid request body")
		return
	}
	d.ID = id
	for i := range d.Tiles {
		if d.Tiles[i].ID == "" {
			d.Tiles[i].ID = s.genID()
		}
	}
	for i := range d.Filters {
		if d.Filters[i].ID == "" {
			d.Filters[i].ID = s.genID()
		}
	}
	if d.SavedQueryLanguage == nil {
		lang := "lucene"
		d.SavedQueryLanguage = &lang
	}
	s.mu.Lock()
	s.dashboards[id] = d
	s.mu.Unlock()
	writeJSON(t, w, 200, d)
}

func (s *Server) deleteDashboard(t *testing.T, w http.ResponseWriter, id string) {
	s.mu.Lock()
	_, ok := s.dashboards[id]
	if ok {
		delete(s.dashboards, id)
	}
	s.mu.Unlock()
	if !ok {
		writeError(t, w, 404, "dashboard not found")
		return
	}
	writeJSON(t, w, 200, nil)
}

// --- Alert handlers ---

func (s *Server) listAlerts(t *testing.T, w http.ResponseWriter) {
	s.mu.Lock()
	defer s.mu.Unlock()
	alerts := make([]client.Alert, 0, len(s.alerts))
	for _, a := range s.alerts {
		alerts = append(alerts, a)
	}
	writeJSON(t, w, 200, alerts)
}

func (s *Server) createAlert(t *testing.T, w http.ResponseWriter, r *http.Request) {
	var a client.Alert
	if err := json.NewDecoder(r.Body).Decode(&a); err != nil {
		writeError(t, w, 400, "invalid request body")
		return
	}
	a.ID = s.genID()
	a.State = "OK"
	s.mu.Lock()
	s.alerts[a.ID] = a
	s.mu.Unlock()
	writeJSON(t, w, 200, a)
}

func (s *Server) getAlert(t *testing.T, w http.ResponseWriter, id string) {
	s.mu.Lock()
	a, ok := s.alerts[id]
	s.mu.Unlock()
	if !ok {
		writeError(t, w, 404, "alert not found")
		return
	}
	writeJSON(t, w, 200, a)
}

func (s *Server) updateAlert(t *testing.T, w http.ResponseWriter, r *http.Request, id string) {
	s.mu.Lock()
	existing, ok := s.alerts[id]
	s.mu.Unlock()
	if !ok {
		writeError(t, w, 404, "alert not found")
		return
	}
	var a client.Alert
	if err := json.NewDecoder(r.Body).Decode(&a); err != nil {
		writeError(t, w, 400, "invalid request body")
		return
	}
	a.ID = id
	a.State = existing.State
	s.mu.Lock()
	s.alerts[id] = a
	s.mu.Unlock()
	writeJSON(t, w, 200, a)
}

func (s *Server) deleteAlert(t *testing.T, w http.ResponseWriter, id string) {
	s.mu.Lock()
	_, ok := s.alerts[id]
	if ok {
		delete(s.alerts, id)
	}
	s.mu.Unlock()
	if !ok {
		writeError(t, w, 404, "alert not found")
		return
	}
	writeJSON(t, w, 200, nil)
}

// --- SavedSearch handlers ---

func (s *Server) listSavedSearches(t *testing.T, w http.ResponseWriter) {
	s.mu.Lock()
	defer s.mu.Unlock()
	searches := make([]client.SavedSearch, 0, len(s.savedSearches))
	for _, ss := range s.savedSearches {
		searches = append(searches, ss)
	}
	writeJSON(t, w, 200, searches)
}

func (s *Server) createSavedSearch(t *testing.T, w http.ResponseWriter, r *http.Request) {
	var ss client.SavedSearch
	if err := json.NewDecoder(r.Body).Decode(&ss); err != nil {
		writeError(t, w, 400, "invalid request body")
		return
	}
	ss.ID = s.genID()
	s.mu.Lock()
	s.savedSearches[ss.ID] = ss
	s.mu.Unlock()
	writeJSON(t, w, 200, ss)
}

func (s *Server) getSavedSearch(t *testing.T, w http.ResponseWriter, id string) {
	s.mu.Lock()
	ss, ok := s.savedSearches[id]
	s.mu.Unlock()
	if !ok {
		writeError(t, w, 404, "saved search not found")
		return
	}
	writeJSON(t, w, 200, ss)
}

func (s *Server) updateSavedSearch(t *testing.T, w http.ResponseWriter, r *http.Request, id string) {
	s.mu.Lock()
	_, ok := s.savedSearches[id]
	s.mu.Unlock()
	if !ok {
		writeError(t, w, 404, "saved search not found")
		return
	}
	var ss client.SavedSearch
	if err := json.NewDecoder(r.Body).Decode(&ss); err != nil {
		writeError(t, w, 400, "invalid request body")
		return
	}
	ss.ID = id
	s.mu.Lock()
	s.savedSearches[id] = ss
	s.mu.Unlock()
	writeJSON(t, w, 200, ss)
}

func (s *Server) deleteSavedSearch(t *testing.T, w http.ResponseWriter, id string) {
	s.mu.Lock()
	_, ok := s.savedSearches[id]
	if ok {
		delete(s.savedSearches, id)
	}
	s.mu.Unlock()
	if !ok {
		writeError(t, w, 404, "saved search not found")
		return
	}
	writeJSON(t, w, 200, nil)
}

// --- Helpers ---

func writeJSON(t *testing.T, w http.ResponseWriter, status int, result any) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	resp := client.APIResponse[any]{Status: status, RequestID: "mock-req", Result: result}
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		t.Fatal(err)
	}
}

func writeError(t *testing.T, w http.ResponseWriter, status int, msg string) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	resp := client.APIResponse[any]{Status: status, RequestID: "mock-req", Error: msg}
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		t.Fatal(err)
	}
}
