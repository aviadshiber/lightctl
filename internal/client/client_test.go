package client

import (
	"bytes"
	"encoding/json"
	"log"
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
	want := "http://example.com/api/v1"
	if c.baseURL != want {
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
			Items:   []Agent{{ID: "a1", Name: "agent-1", Type: "Java", Version: "1.0", VersionStatus: "Active"}},
			Total:   1,
			HasMore: false,
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer ts.Close()

	c := &Client{
		httpClient:  ts.Client(),
		baseURL:     ts.URL + "/api/v1",
		apiKey:      "testkey",
		agentPoolID: "pool-1",
		maxRetries:  0,
	}

	resp, err := c.ListAgents(50, 0)
	if err != nil {
		t.Fatalf("ListAgents: %v", err)
	}
	if len(resp.Items) != 1 {
		t.Fatalf("expected 1 agent, got %d", len(resp.Items))
	}
	if resp.Items[0].ID != "a1" {
		t.Fatalf("expected ID a1, got %s", resp.Items[0].ID)
	}
}

func TestCreateAction(t *testing.T) {
	t.Parallel()
	ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == "POST" && strings.HasPrefix(r.URL.Path, "/athena/company/company-1/1.78/insertCapture/"):
			if r.Header.Get("Content-Type") != "application/json" {
				t.Errorf("missing content-type")
			}
			var body insertCaptureBody
			_ = json.NewDecoder(r.Body).Decode(&body)
			if body.AgentID != "agent1" || body.Filename != "Foo.java" || body.Line != 42 {
				t.Errorf("unexpected body: agentId=%s filename=%s line=%d", body.AgentID, body.Filename, body.Line)
			}
			_ = json.NewEncoder(w).Encode(insertCaptureResponse{Status: "OK", StatusCode: "STATUS_OK", ID: "snap-1"})

		default:
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer ts.Close()

	c := &Client{
		httpClient:  ts.Client(),
		baseURL:     ts.URL + "/api/v1",
		athenaURL:   ts.URL + "/athena",
		apiKey:      "testkey",
		agentPoolID: "pool-1",
		companyID:   "company-1",
		maxRetries:  0,
	}

	action, err := c.CreateAction(CreateActionRequest{
		AgentID:    "agent1",
		ActionType: "CAPTURE",
		Location:   "Foo.java",
		Line:       42,
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
		resp := PagedResponse[Agent]{Items: []Agent{{ID: "a1"}}, Total: 1}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer ts.Close()

	c := &Client{
		httpClient:  ts.Client(),
		baseURL:     ts.URL + "/api/v1",
		apiKey:      "testkey",
		agentPoolID: "pool-1",
		maxRetries:  5,
	}

	resp, err := c.ListAgents(50, 0)
	if err != nil {
		t.Fatalf("expected success after retries, got: %v", err)
	}
	if len(resp.Items) != 1 {
		t.Fatalf("expected 1 agent, got %d", len(resp.Items))
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
		httpClient:  ts.Client(),
		baseURL:     ts.URL + "/api/v1",
		apiKey:      "testkey",
		agentPoolID: "pool-1",
		maxRetries:  2,
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

	var logBuf bytes.Buffer
	log.SetOutput(&logBuf)
	defer log.SetOutput(os.Stderr)

	ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(PagedResponse[Agent]{Items: nil, Total: 0})
	}))
	defer ts.Close()

	c := &Client{
		httpClient:  ts.Client(),
		baseURL:     ts.URL + "/api/v1",
		apiKey:      "supersecrettoken",
		agentPoolID: "pool-1",
		maxRetries:  0,
		debug:       true,
	}

	_, err := c.ListAgents(10, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	logOutput := logBuf.String()
	if strings.Contains(logOutput, "supersecrettoken") {
		t.Fatal("API key leaked in debug log output")
	}
	if !strings.Contains(logOutput, "LR****") {
		t.Fatal("expected redacted key marker in debug output")
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
		httpClient:  ts.Client(),
		baseURL:     ts.URL + "/api/v1",
		apiKey:      "testkey",
		agentPoolID: "pool-1",
		maxRetries:  0,
	}

	_, err := c.ListAgents(10, 0)
	if err == nil {
		t.Fatal("expected error for 404")
	}
	if !strings.Contains(err.Error(), "404") {
		t.Fatalf("error should contain 404: %v", err)
	}
}
