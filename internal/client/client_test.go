package client

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync/atomic"
	"testing"
)

func TestNew_RejectHTTP(t *testing.T) {
	t.Parallel()
	_, err := New("http://example.com", "key")
	if err == nil {
		t.Fatal("expected error for http:// URL")
	}
	if !strings.Contains(err.Error(), "https://") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestNew_AllowHTTPWithFlag(t *testing.T) {
	t.Parallel()
	c, err := New("http://example.com", "key", WithInsecureHTTP(true))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c.baseURL != "http://example.com/api/v1" {
		t.Fatalf("unexpected baseURL: %s", c.baseURL)
	}
}

func TestListAgents(t *testing.T) {
	t.Parallel()
	ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if auth != "Bearer testkey" {
			t.Errorf("expected Bearer testkey, got %q", auth)
		}
		if r.URL.Path != "/api/v1/agents" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		resp := PagedResponse[Agent]{
			Data: []Agent{
				{ID: "a1", DisplayName: "agent-1", Host: "host1", Status: "ACTIVE", Tags: []Tag{{Name: "prod"}}},
			},
			TotalCount: 1,
			PageCount:  1,
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer ts.Close()

	c := &Client{
		httpClient: ts.Client(),
		baseURL:    ts.URL + "/api/v1",
		apiKey:     "testkey",
		maxRetries: 0,
	}

	resp, err := c.ListAgents(50, 0)
	if err != nil {
		t.Fatalf("ListAgents: %v", err)
	}
	if len(resp.Data) != 1 {
		t.Fatalf("expected 1 agent, got %d", len(resp.Data))
	}
	if resp.Data[0].ID != "a1" {
		t.Fatalf("expected ID a1, got %s", resp.Data[0].ID)
	}
}

func TestCreateAction(t *testing.T) {
	t.Parallel()
	ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("missing content-type")
		}

		var req CreateActionRequest
		_ = json.NewDecoder(r.Body).Decode(&req)
		if req.AgentID != "agent1" || req.FileName != "Foo.java" || req.LineNumber != 42 {
			t.Errorf("unexpected request: %+v", req)
		}

		action := Action{
			ID:         "snap-1",
			Type:       "SNAPSHOT",
			AgentID:    "agent1",
			FileName:   "Foo.java",
			LineNumber: 42,
			Status:     "ACTIVE",
		}
		_ = json.NewEncoder(w).Encode(action)
	}))
	defer ts.Close()

	c := &Client{
		httpClient: ts.Client(),
		baseURL:    ts.URL + "/api/v1",
		apiKey:     "testkey",
		maxRetries: 0,
	}

	action, err := c.CreateAction(CreateActionRequest{
		AgentID:    "agent1",
		Type:       "SNAPSHOT",
		FileName:   "Foo.java",
		LineNumber: 42,
	})
	if err != nil {
		t.Fatalf("CreateAction: %v", err)
	}
	if action.ID != "snap-1" {
		t.Fatalf("expected snap-1, got %s", action.ID)
	}
}

func TestRetryOn429(t *testing.T) {
	t.Parallel()
	var attempts int32
	ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := atomic.AddInt32(&attempts, 1)
		if n <= 2 {
			w.Header().Set("Retry-After", "0")
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		resp := PagedResponse[Agent]{Data: []Agent{{ID: "a1"}}, TotalCount: 1, PageCount: 1}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer ts.Close()

	c := &Client{
		httpClient: ts.Client(),
		baseURL:    ts.URL + "/api/v1",
		apiKey:     "testkey",
		maxRetries: 5,
	}

	resp, err := c.ListAgents(50, 0)
	if err != nil {
		t.Fatalf("expected success after retries, got: %v", err)
	}
	if len(resp.Data) != 1 {
		t.Fatalf("expected 1 agent, got %d", len(resp.Data))
	}
	if got := atomic.LoadInt32(&attempts); got != 3 {
		t.Fatalf("expected 3 attempts, got %d", got)
	}
}

func TestRetryExhausted(t *testing.T) {
	t.Parallel()
	ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Retry-After", "0")
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer ts.Close()

	c := &Client{
		httpClient: ts.Client(),
		baseURL:    ts.URL + "/api/v1",
		apiKey:     "testkey",
		maxRetries: 2,
	}

	_, err := c.ListAgents(50, 0)
	if err == nil {
		t.Fatal("expected error after exhausted retries")
	}
	if !strings.Contains(err.Error(), "rate limited") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDebugRedaction(t *testing.T) {
	t.Setenv("LIGHTCTL_DEBUG", "1")
	defer os.Unsetenv("LIGHTCTL_DEBUG")

	ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(PagedResponse[Agent]{Data: nil, TotalCount: 0, PageCount: 0})
	}))
	defer ts.Close()

	c := &Client{
		httpClient: ts.Client(),
		baseURL:    ts.URL + "/api/v1",
		apiKey:     "supersecrettoken",
		maxRetries: 0,
		debug:      true,
	}

	_, err := c.ListAgents(10, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDeleteAction(t *testing.T) {
	t.Parallel()
	ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "DELETE" {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		if r.URL.Path != "/api/v1/actions/act-123" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer ts.Close()

	c := &Client{
		httpClient: ts.Client(),
		baseURL:    ts.URL + "/api/v1",
		apiKey:     "testkey",
		maxRetries: 0,
	}

	if err := c.DeleteAction("act-123"); err != nil {
		t.Fatalf("DeleteAction: %v", err)
	}
}

func TestHTTPError(t *testing.T) {
	t.Parallel()
	ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"error":"not found"}`))
	}))
	defer ts.Close()

	c := &Client{
		httpClient: ts.Client(),
		baseURL:    ts.URL + "/api/v1",
		apiKey:     "testkey",
		maxRetries: 0,
	}

	_, err := c.GetAction("missing")
	if err == nil {
		t.Fatal("expected error for 404")
	}
	if !strings.Contains(err.Error(), "404") {
		t.Fatalf("error should contain 404: %v", err)
	}
}
