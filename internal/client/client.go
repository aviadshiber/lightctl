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
		},
	}

	return c, nil
}

// --- public helpers ---

// ListAgents returns a page of agents.
func (c *Client) ListAgents(limit, page int) (*PagedResponse[Agent], error) {
	url := fmt.Sprintf("%s/agents?limit=%d&page=%d", c.baseURL, limit, page)
	var resp PagedResponse[Agent]
	if err := c.doJSON("GET", url, nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// ListActions returns actions for an agent (optionally filtered by type).
func (c *Client) ListActions(agentID string, actionType string, limit, page int) (*PagedResponse[Action], error) {
	url := fmt.Sprintf("%s/actions?agentId=%s&limit=%d&page=%d", c.baseURL, agentID, limit, page)
	if actionType != "" {
		url += "&type=" + actionType
	}
	var resp PagedResponse[Action]
	if err := c.doJSON("GET", url, nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// CreateAction creates a new action and returns it.
func (c *Client) CreateAction(req CreateActionRequest) (*Action, error) {
	url := c.baseURL + "/actions"
	var action Action
	if err := c.doJSON("POST", url, req, &action); err != nil {
		return nil, err
	}
	return &action, nil
}

// GetAction returns a single action.
func (c *Client) GetAction(actionID string) (*Action, error) {
	url := fmt.Sprintf("%s/actions/%s", c.baseURL, actionID)
	var action Action
	if err := c.doJSON("GET", url, nil, &action); err != nil {
		return nil, err
	}
	return &action, nil
}

// DeleteAction deletes an action by ID.
func (c *Client) DeleteAction(actionID string) error {
	url := fmt.Sprintf("%s/actions/%s", c.baseURL, actionID)
	return c.doJSON("DELETE", url, nil, nil)
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

		defer resp.Body.Close()

		if resp.StatusCode >= 400 {
			respBody, _ := io.ReadAll(resp.Body)
			return fmt.Errorf("%s %s: HTTP %d: %s", method, url, resp.StatusCode, string(respBody))
		}

		if result != nil {
			if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
				return fmt.Errorf("decoding response: %w", err)
			}
		} else {
			// Drain body
			io.Copy(io.Discard, resp.Body)
		}

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
