package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math"
	"math/rand"
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

// Client talks to the LightRun REST API.
type Client struct {
	httpClient   *http.Client
	baseURL      string
	apiKey       string
	agentPoolID  string
	debug        bool
	maxRetries   int
	insecureHTTP bool
}

// Option configures a Client.
type Option func(*Client)

// WithInsecureHTTP allows http:// URLs.
func WithInsecureHTTP(allow bool) Option {
	return func(c *Client) { c.insecureHTTP = allow }
}

// New creates a Client for the given server and API key.
func New(server, apiKey string, opts ...Option) (*Client, error) {
	c := &Client{
		apiKey:     apiKey,
		maxRetries: 5,
		debug:      os.Getenv("LIGHTCTL_DEBUG") == "1",
	}
	for _, o := range opts {
		o(c)
	}

	if !c.insecureHTTP && !strings.HasPrefix(server, "https://") {
		return nil, fmt.Errorf("server URL must use https:// (got %q); use --insecure-http to allow plain HTTP", server)
	}

	c.baseURL = strings.TrimRight(server, "/") + "/api/v1"

	c.httpClient = &http.Client{
		Timeout: 10 * time.Second,
		Transport: &http.Transport{
			Proxy: http.ProxyFromEnvironment,
			DialContext: (&net.Dialer{
				Timeout: 3 * time.Second,
			}).DialContext,
			MaxIdleConns:        100,
			MaxIdleConnsPerHost: 10,
			IdleConnTimeout:     90 * time.Second,
		},
	}

	return c, nil
}

// SetAgentPoolID overrides the agent pool used for all requests.
func (c *Client) SetAgentPoolID(id string) {
	c.agentPoolID = id
}

// AgentPoolID returns the currently configured agent pool ID.
func (c *Client) AgentPoolID() string {
	return c.agentPoolID
}

// AutoDiscoverPool fetches the list of agent pools and sets the first one.
// Returns an error if no pools are found.
func (c *Client) AutoDiscoverPool() error {
	resp, err := c.GetAgentPools(20, 0)
	if err != nil {
		return fmt.Errorf("auto-discovering agent pool: %w", err)
	}
	if len(resp.Items) == 0 {
		return fmt.Errorf("no agent pools found; set agent_pool_id in config")
	}
	c.agentPoolID = resp.Items[0].ID
	return nil
}

// --- public helpers ---

// GetAgentPools returns a page of agent pools.
func (c *Client) GetAgentPools(limit, offset int) (*PagedResponse[AgentPool], error) {
	reqURL := fmt.Sprintf("%s/agent-pools?limit=%d&offset=%d", c.baseURL, limit, offset)
	var resp PagedResponse[AgentPool]
	if err := c.doJSON("GET", reqURL, nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// ListAgents returns a page of agents.
func (c *Client) ListAgents(limit, offset int) (*PagedResponse[Agent], error) {
	params := url.Values{}
	params.Set("agentPoolId", c.agentPoolID)
	params.Set("limit", strconv.Itoa(limit))
	params.Set("offset", strconv.Itoa(offset))
	reqURL := c.baseURL + "/agents?" + params.Encode()
	var resp PagedResponse[Agent]
	if err := c.doJSON("GET", reqURL, nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// ListActions returns actions for an agent pool (optionally filtered by type and agentId).
func (c *Client) ListActions(agentID string, actionType string, limit, offset int) (*PagedResponse[Action], error) {
	params := url.Values{}
	params.Set("agentPoolId", c.agentPoolID)
	params.Set("limit", strconv.Itoa(limit))
	params.Set("offset", strconv.Itoa(offset))
	if agentID != "" {
		params.Set("agentId", agentID)
	}
	if actionType != "" {
		params.Set("type", actionType)
	}
	reqURL := c.baseURL + "/actions?" + params.Encode()
	var resp PagedResponse[Action]
	if err := c.doJSON("GET", reqURL, nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// CreateAction creates a new action and returns it.
func (c *Client) CreateAction(req CreateActionRequest) (*Action, error) {
	params := url.Values{}
	params.Set("agentPoolId", c.agentPoolID)
	reqURL := c.baseURL + "/actions?" + params.Encode()
	var action Action
	if err := c.doJSON("POST", reqURL, req, &action); err != nil {
		return nil, err
	}
	return &action, nil
}

// GetAction finds an action by ID by listing and filtering.
// The LightRun REST API does not support GET /actions/{id} directly.
func (c *Client) GetAction(actionID string) (*Action, error) {
	offset := 0
	for {
		resp, err := c.ListActions("", "", 100, offset)
		if err != nil {
			return nil, fmt.Errorf("searching for action %s: %w", actionID, err)
		}
		for _, a := range resp.Items {
			if a.ID == actionID {
				return &a, nil
			}
		}
		if !resp.HasMore {
			return nil, fmt.Errorf("action %s not found", actionID)
		}
		offset += len(resp.Items)
	}
}

// DeleteAction deletes an action by ID.
func (c *Client) DeleteAction(actionID string) error {
	reqURL := fmt.Sprintf("%s/actions/%s", c.baseURL, actionID)
	return c.doJSON("DELETE", reqURL, nil, nil)
}

// --- internal ---

func (c *Client) doJSON(method, url string, body interface{}, result interface{}) error {
	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("marshalling request body: %w", err)
		}
		bodyReader = bytes.NewReader(data)
	}

	var lastErr error
	for attempt := 0; attempt <= c.maxRetries; attempt++ {
		req, err := http.NewRequest(method, url, bodyReader)
		if err != nil {
			return fmt.Errorf("creating request: %w", err)
		}
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
		if body != nil {
			req.Header.Set("Content-Type", "application/json")
		}

		if c.debug {
			log.Printf("[DEBUG] %s %s  Authorization: Bearer LR****", method, url)
		}

		resp, err := c.httpClient.Do(req)
		if err != nil {
			return fmt.Errorf("%s %s: %w", method, url, err)
		}

		if c.debug {
			log.Printf("[DEBUG] %s %s → %d", method, url, resp.StatusCode)
		}

		if resp.StatusCode == http.StatusTooManyRequests {
			resp.Body.Close()
			if attempt == c.maxRetries {
				return fmt.Errorf("%s %s: rate limited after %d retries", method, url, c.maxRetries)
			}
			wait := retryDelay(resp, attempt)
			if c.debug {
				log.Printf("[DEBUG] 429 retry in %v (attempt %d/%d)", wait, attempt+1, c.maxRetries)
			}
			time.Sleep(wait)
			// Reset reader for retry
			if body != nil {
				data, _ := json.Marshal(body)
				bodyReader = bytes.NewReader(data)
			}
			lastErr = fmt.Errorf("rate limited")
			continue
		}

		if resp.StatusCode >= 400 {
			respBody, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			return fmt.Errorf("%s %s: HTTP %d: %s", method, url, resp.StatusCode, string(respBody))
		}

		if result != nil {
			if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
				resp.Body.Close()
				return fmt.Errorf("decoding response: %w", err)
			}
		} else {
			// Drain body to allow connection reuse
			_, _ = io.Copy(io.Discard, resp.Body)
		}
		resp.Body.Close()
		return nil
	}

	return lastErr
}

// retryDelay computes the delay for a 429 retry, honouring the Retry-After
// header when present, otherwise using exponential backoff with jitter.
func retryDelay(resp *http.Response, attempt int) time.Duration {
	if ra := resp.Header.Get("Retry-After"); ra != "" {
		if secs, err := strconv.Atoi(ra); err == nil && secs > 0 {
			d := time.Duration(secs) * time.Second
			if d > 30*time.Second {
				d = 30 * time.Second
			}
			return d
		}
	}
	base := float64(time.Second)
	backoff := base*math.Pow(2, float64(attempt)) + float64(rand.Int63n(int64(base)))
	if backoff > float64(30*time.Second) {
		backoff = float64(30 * time.Second)
	}
	return time.Duration(backoff)
}
