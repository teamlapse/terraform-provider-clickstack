// Copyright (c) Lapse Technologies, Inc.
// SPDX-License-Identifier: MPL-2.0

package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"strconv"
	"sync"
	"time"
)

const (
	defaultRateLimit = 5.0
	maxRetries       = 4
	baseRetryDelay   = 500 * time.Millisecond
	maxRetryDelay    = 30 * time.Second
)

type rateLimiter struct {
	mu       sync.Mutex
	tokens   float64
	max      float64
	rate     float64
	lastFill time.Time
}

func newRateLimiter(rps float64) *rateLimiter {
	return &rateLimiter{
		tokens:   rps,
		max:      rps,
		rate:     rps,
		lastFill: time.Now(),
	}
}

func (r *rateLimiter) wait(ctx context.Context) error {
	for {
		r.mu.Lock()
		now := time.Now()
		elapsed := now.Sub(r.lastFill).Seconds()
		r.tokens = math.Min(r.max, r.tokens+elapsed*r.rate)
		r.lastFill = now

		if r.tokens >= 1 {
			r.tokens--
			r.mu.Unlock()
			return nil
		}

		waitDur := time.Duration((1-r.tokens)/r.rate*1000) * time.Millisecond
		r.mu.Unlock()

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(waitDur):
		}
	}
}

type Client struct {
	baseURL        string
	organizationID string
	serviceID      string
	apiKeyID       string
	apiKeySecret   string
	httpClient     *http.Client
	rateLimiter    *rateLimiter
}

func NewClient(baseURL, organizationID, serviceID, apiKeyID, apiKeySecret string) *Client {
	return &Client{
		baseURL:        baseURL,
		organizationID: organizationID,
		serviceID:      serviceID,
		apiKeyID:       apiKeyID,
		apiKeySecret:   apiKeySecret,
		httpClient:     &http.Client{Timeout: 30 * time.Second},
		rateLimiter:    newRateLimiter(defaultRateLimit),
	}
}

func (c *Client) basePath() string {
	return fmt.Sprintf("%s/v1/organizations/%s/services/%s/clickstack", c.baseURL, c.organizationID, c.serviceID)
}

func (c *Client) doRequest(ctx context.Context, method, path string, body any) ([]byte, error) {
	var jsonBody []byte
	if body != nil {
		var err error
		jsonBody, err = json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshaling request body: %w", err)
		}
	}

	var lastErr error
	for attempt := 0; attempt <= maxRetries; attempt++ {
		if err := c.rateLimiter.wait(ctx); err != nil {
			return nil, err
		}

		var reqBody io.Reader
		if jsonBody != nil {
			reqBody = bytes.NewReader(jsonBody)
		}

		req, err := http.NewRequestWithContext(ctx, method, c.basePath()+path, reqBody)
		if err != nil {
			return nil, fmt.Errorf("creating request: %w", err)
		}

		req.SetBasicAuth(c.apiKeyID, c.apiKeySecret)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "application/json")

		resp, err := c.httpClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf("executing request: %w", err)
		}

		respBody, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("reading response body: %w", err)
		}
		err = resp.Body.Close()
		if err != nil {
			return nil, fmt.Errorf("closing response body: %w", err)
		}

		if resp.StatusCode == http.StatusTooManyRequests {
			delay := retryAfterDelay(resp, attempt)
			var apiErr APIResponse[json.RawMessage]
			msg := string(respBody)
			requestID := ""
			if json.Unmarshal(respBody, &apiErr) == nil && apiErr.Error != "" {
				msg = apiErr.Error
				requestID = apiErr.RequestID
			}
			lastErr = fmt.Errorf("API error (status %d, requestId %s): %s", resp.StatusCode, requestID, msg)
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(delay):
				continue
			}
		}

		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			var apiErr APIResponse[json.RawMessage]
			msg := string(respBody)
			requestID := ""
			if json.Unmarshal(respBody, &apiErr) == nil && apiErr.Error != "" {
				msg = apiErr.Error
				requestID = apiErr.RequestID
			}
			if resp.StatusCode == http.StatusNotFound {
				return nil, &NotFoundError{Message: msg, RequestID: requestID}
			}
			return nil, fmt.Errorf("API error (status %d, requestId %s): %s", resp.StatusCode, requestID, msg)
		}

		return respBody, nil
	}

	return nil, fmt.Errorf("request failed after %d retries: %w", maxRetries, lastErr)
}

func retryAfterDelay(resp *http.Response, attempt int) time.Duration {
	if ra := resp.Header.Get("Retry-After"); ra != "" {
		if secs, err := strconv.ParseFloat(ra, 64); err == nil {
			return time.Duration(secs * float64(time.Second))
		}
		if t, err := http.ParseTime(ra); err == nil {
			if d := time.Until(t); d > 0 {
				return d
			}
		}
	}
	delay := time.Duration(float64(baseRetryDelay) * math.Pow(2, float64(attempt)))
	if delay > maxRetryDelay {
		delay = maxRetryDelay
	}
	return delay
}

func unwrapResult[T any](data []byte) (T, error) {
	var resp APIResponse[T]
	if err := json.Unmarshal(data, &resp); err != nil {
		var zero T
		return zero, fmt.Errorf("unmarshaling response: %w", err)
	}
	return resp.Result, nil
}

type NotFoundError struct {
	Message   string
	RequestID string
}

func (e *NotFoundError) Error() string {
	return fmt.Sprintf("not found (requestId %s): %s", e.RequestID, e.Message)
}

func IsNotFound(err error) bool {
	if err == nil {
		return false
	}
	_, ok := err.(*NotFoundError)
	return ok
}

func (c *Client) ListDashboards(ctx context.Context) ([]Dashboard, error) {
	data, err := c.doRequest(ctx, http.MethodGet, "/dashboards", nil)
	if err != nil {
		return nil, err
	}
	return unwrapResult[[]Dashboard](data)
}

func (c *Client) GetDashboard(ctx context.Context, id string) (*Dashboard, error) {
	data, err := c.doRequest(ctx, http.MethodGet, "/dashboards/"+id, nil)
	if err != nil {
		return nil, err
	}
	result, err := unwrapResult[Dashboard](data)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *Client) CreateDashboard(ctx context.Context, dashboard Dashboard) (*Dashboard, error) {
	data, err := c.doRequest(ctx, http.MethodPost, "/dashboards", dashboard)
	if err != nil {
		return nil, err
	}
	result, err := unwrapResult[Dashboard](data)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *Client) UpdateDashboard(ctx context.Context, id string, dashboard Dashboard) (*Dashboard, error) {
	data, err := c.doRequest(ctx, http.MethodPut, "/dashboards/"+id, dashboard)
	if err != nil {
		return nil, err
	}
	result, err := unwrapResult[Dashboard](data)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *Client) DeleteDashboard(ctx context.Context, id string) error {
	_, err := c.doRequest(ctx, http.MethodDelete, "/dashboards/"+id, nil)
	return err
}

func (c *Client) ListAlerts(ctx context.Context) ([]Alert, error) {
	data, err := c.doRequest(ctx, http.MethodGet, "/alerts", nil)
	if err != nil {
		return nil, err
	}
	return unwrapResult[[]Alert](data)
}

func (c *Client) GetAlert(ctx context.Context, id string) (*Alert, error) {
	data, err := c.doRequest(ctx, http.MethodGet, "/alerts/"+id, nil)
	if err != nil {
		return nil, err
	}
	result, err := unwrapResult[Alert](data)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *Client) CreateAlert(ctx context.Context, alert Alert) (*Alert, error) {
	data, err := c.doRequest(ctx, http.MethodPost, "/alerts", alert)
	if err != nil {
		return nil, err
	}
	result, err := unwrapResult[Alert](data)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *Client) UpdateAlert(ctx context.Context, id string, alert Alert) (*Alert, error) {
	data, err := c.doRequest(ctx, http.MethodPut, "/alerts/"+id, alert)
	if err != nil {
		return nil, err
	}
	result, err := unwrapResult[Alert](data)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *Client) DeleteAlert(ctx context.Context, id string) error {
	_, err := c.doRequest(ctx, http.MethodDelete, "/alerts/"+id, nil)
	return err
}

func (c *Client) ListSavedSearches(ctx context.Context) ([]SavedSearch, error) {
	data, err := c.doRequest(ctx, http.MethodGet, "/savedSearches", nil)
	if err != nil {
		return nil, err
	}
	return unwrapResult[[]SavedSearch](data)
}

func (c *Client) GetSavedSearch(ctx context.Context, id string) (*SavedSearch, error) {
	data, err := c.doRequest(ctx, http.MethodGet, "/savedSearches/"+id, nil)
	if err != nil {
		return nil, err
	}
	result, err := unwrapResult[SavedSearch](data)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *Client) CreateSavedSearch(ctx context.Context, search SavedSearch) (*SavedSearch, error) {
	data, err := c.doRequest(ctx, http.MethodPost, "/savedSearches", search)
	if err != nil {
		return nil, err
	}
	result, err := unwrapResult[SavedSearch](data)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *Client) UpdateSavedSearch(ctx context.Context, id string, search SavedSearch) (*SavedSearch, error) {
	data, err := c.doRequest(ctx, http.MethodPut, "/savedSearches/"+id, search)
	if err != nil {
		return nil, err
	}
	result, err := unwrapResult[SavedSearch](data)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *Client) DeleteSavedSearch(ctx context.Context, id string) error {
	_, err := c.doRequest(ctx, http.MethodDelete, "/savedSearches/"+id, nil)
	return err
}

func (c *Client) ListSources(ctx context.Context) ([]Source, error) {
	data, err := c.doRequest(ctx, http.MethodGet, "/sources", nil)
	if err != nil {
		return nil, err
	}
	return unwrapResult[[]Source](data)
}

func (c *Client) ListWebhooks(ctx context.Context) ([]Webhook, error) {
	data, err := c.doRequest(ctx, http.MethodGet, "/webhooks", nil)
	if err != nil {
		return nil, err
	}
	return unwrapResult[[]Webhook](data)
}
