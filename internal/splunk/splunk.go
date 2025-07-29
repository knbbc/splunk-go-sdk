package splunk

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Client represents a Splunk client.
type Client struct {
	BaseURL    string
	Username   string
	Password   string
	Token      string
	HTTPClient *http.Client
}

// Event represents a single event to be sent to Splunk.
type Event struct {
	Time  int64          `json:"time"`
	Event map[string]any `json:"event"`
}

// NewClient creates a new Splunk client.
// It requires either a username and password or a token for authentication.
// If both are provided, the token will take precedence.
func NewClient(baseURL string, username, password, token string) (*Client, error) {
	if token == "" && (username == "" || password == "") {
		return nil, fmt.Errorf("either a token or a username and password must be provided")
	}

	return &Client{
		BaseURL:  baseURL,
		Username: username,
		Token:    token,
		HTTPClient: &http.Client{
			Timeout: 30 * time.Second, // Set a timeout to avoid unbounded request durations
		},
	}, nil
}

func (c *Client) Search(query string) (map[string]any, error) {
	req, err := c.prepareHttpRequest(query)
	if err != nil {
		return nil, err
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("search request failed: %s", resp.Status)
	}

	return parseSplunkSearchResults(resp.Body)
}

// SendEvents sends events to a Splunk index using the HTTP Event Collector (HEC) API.
func (c *Client) SendEvents(indexName string, events []Event) error {
	if c.Token == "" {
		return fmt.Errorf("HEC requires a token for authentication")
	}

	hecURL := strings.TrimRight(c.BaseURL, "/") + "/services/collector/event"
	for _, event := range events {
		payload := map[string]any{
			"index": indexName,
			"time":  event.Time,
			"event": event.Event,
		}
		body, err := json.Marshal(payload)
		if err != nil {
			return fmt.Errorf("failed to marshal event: %w", err)
		}

		req, err := http.NewRequest("POST", hecURL, strings.NewReader(string(body)))
		if err != nil {
			return fmt.Errorf("failed to create request: %w", err)
		}
		req.Header.Set("Authorization", "Splunk "+c.Token)
		req.Header.Set("Content-Type", "application/json")

		resp, err := c.HTTPClient.Do(req)
		if err != nil {
			return fmt.Errorf("failed to send event: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			respBody, _ := io.ReadAll(resp.Body)
			return fmt.Errorf("failed to send event: %s - %s", resp.Status, string(respBody))
		}
	}
	return nil

}

// prepareHttpRequest prepares the HTTP request for a Splunk search job.
func (c *Client) prepareHttpRequest(query string) (*http.Request, error) {
	searchURL := c.BaseURL + "/services/search/jobs"
	reqBody := fmt.Sprintf("search=%s&exec_mode=oneshot&output_mode=json", query)

	req, err := http.NewRequest("POST", searchURL, strings.NewReader(reqBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	if err := c.setAuthHeader(req); err != nil {
		return nil, err
	}
	return req, nil
}

// setAuthHeader sets the appropriate authentication header for the request.
func (c *Client) setAuthHeader(req *http.Request) error {
	if c.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.Token)
		return nil
	}
	if c.Username != "" && c.Password != "" {
		req.SetBasicAuth(c.Username, c.Password)
		return nil
	}
	return fmt.Errorf("no authentication credentials provided")
}

// parseSplunkSearchResults parses the Splunk search results from the response body.
func parseSplunkSearchResults(body io.Reader) (map[string]any, error) {
	var result map[string]any
	decoder := json.NewDecoder(body)
	if err := decoder.Decode(&result); err != nil {
		return nil, err
	}
	return result, nil
}
