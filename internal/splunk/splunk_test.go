package splunk

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
)

// mockRoundTripper implements http.RoundTripper for mocking HTTP requests.
type mockRoundTripper struct {
	fn func(req *http.Request) *http.Response
}

func (m *mockRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	return m.fn(req), nil
}

// mockErrorRoundTripper simulates an error on Do.
type mockErrorRoundTripper struct{}

func (m *mockErrorRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	return nil, errors.New("network error")
}

func newTestClient(rt http.RoundTripper) *Client {
	return &Client{
		BaseURL:    "http://localhost:8089",
		Username:   "user",
		Password:   "pass",
		Token:      "",
		HTTPClient: &http.Client{Transport: rt},
	}
}

func TestSearch_Success(t *testing.T) {
	mockResp := `{"results":[{"field":"value"}]}`
	client := newTestClient(&mockRoundTripper{
		fn: func(req *http.Request) *http.Response {
			// Check method and URL
			if req.Method != "POST" {
				t.Errorf("expected POST, got %s", req.Method)
			}
			if req.URL.Path != "/services/search/jobs" {
				t.Errorf("unexpected path: %s", req.URL.Path)
			}
			// Check Content-Type
			if req.Header.Get("Content-Type") != "application/x-www-form-urlencoded" {
				t.Errorf("unexpected Content-Type: %s", req.Header.Get("Content-Type"))
			}
			// Check body
			body, _ := io.ReadAll(req.Body)
			if !bytes.Contains(body, []byte("search=something")) {
				t.Errorf("body missing search param: %s", string(body))
			}
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewBufferString(mockResp)),
				Header:     make(http.Header),
			}
		},
	})

	result, err := client.Search("something")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	results, ok := result["results"].([]any)
	if !ok || len(results) != 1 {
		t.Fatalf("unexpected results: %v", result)
	}
}

func TestSearch_WithOptions(t *testing.T) {
	client := newTestClient(&mockRoundTripper{
		fn: func(req *http.Request) *http.Response {
			body, _ := io.ReadAll(req.Body)
			b := string(body)
			if !strings.Contains(b, "exec_mode=blocking") {
				t.Errorf("expected exec_mode=blocking, got %s", b)
			}
			if !strings.Contains(b, "earliest_time=1") {
				t.Errorf("expected earliest_time=1, got %s", b)
			}
			if !strings.Contains(b, "latest_time=2") {
				t.Errorf("expected latest_time=2, got %s", b)
			}
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewBufferString(`{"results":[]}`)),
				Header:     make(http.Header),
			}
		},
	})

	_, err := client.Search("foo", "exec_mode=blocking", "earliest_time=1", "latest_time=2")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSearch_InvalidOptionFormat(t *testing.T) {
	client := newTestClient(&mockRoundTripper{
		fn: func(req *http.Request) *http.Response {
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewBufferString(`{"results":[]}`)),
				Header:     make(http.Header),
			}
		},
	})

	_, err := client.Search("foo", "invalidoption")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSearch_AuthHeaderError(t *testing.T) {
	client := &Client{
		BaseURL:    "http://localhost:8089",
		Username:   "",
		Password:   "",
		Token:      "",
		HTTPClient: &http.Client{},
	}
	_, err := client.Search("foo")
	if err == nil || !strings.Contains(err.Error(), "no authentication credentials") {
		t.Fatalf("expected auth error, got %v", err)
	}
}

func TestSearch_RequestError(t *testing.T) {
	client := &Client{
		BaseURL:    "http://localhost:8089",
		Username:   "user",
		Password:   "pass",
		Token:      "",
		HTTPClient: &http.Client{Transport: &mockErrorRoundTripper{}},
	}
	_, err := client.Search("foo")
	if err == nil || !strings.Contains(err.Error(), "network error") {
		t.Fatalf("expected network error, got %v", err)
	}
}

func TestSearch_NonOKStatus(t *testing.T) {
	client := newTestClient(&mockRoundTripper{
		fn: func(req *http.Request) *http.Response {
			return &http.Response{
				StatusCode: http.StatusBadRequest,
				Status:     "400 Bad Request",
				Body:       io.NopCloser(bytes.NewBufferString("bad request")),
				Header:     make(http.Header),
			}
		},
	})
	_, err := client.Search("foo")
	if err == nil || !strings.Contains(err.Error(), "search request failed") {
		t.Fatalf("expected search request failed error, got %v", err)
	}
}

func TestSearch_InvalidJSONResponse(t *testing.T) {
	client := newTestClient(&mockRoundTripper{
		fn: func(req *http.Request) *http.Response {
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewBufferString("not json")),
				Header:     make(http.Header),
			}
		},
	})
	_, err := client.Search("foo")
	if err == nil {
		t.Fatalf("expected JSON decode error, got nil")
	}
}
func TestSendEvents_Success(t *testing.T) {
	var called int
	client := &Client{
		BaseURL: "http://localhost:8089",
		Token:   "test-token",
		HTTPClient: &http.Client{
			Transport: &mockRoundTripper{
				fn: func(req *http.Request) *http.Response {
					called++
					// Check URL
					if req.URL.Path != "/services/collector/event" {
						t.Errorf("unexpected path: %s", req.URL.Path)
					}
					// Check method
					if req.Method != "POST" {
						t.Errorf("expected POST, got %s", req.Method)
					}
					// Check headers
					if req.Header.Get("Authorization") != "Splunk test-token" {
						t.Errorf("unexpected Authorization header: %s", req.Header.Get("Authorization"))
					}
					if req.Header.Get("Content-Type") != "application/json" {
						t.Errorf("unexpected Content-Type: %s", req.Header.Get("Content-Type"))
					}
					// Check body
					body, _ := io.ReadAll(req.Body)
					if !bytes.Contains(body, []byte(`"index":"main"`)) {
						t.Errorf("body missing index: %s", string(body))
					}
					return &http.Response{
						StatusCode: http.StatusOK,
						Body:       io.NopCloser(bytes.NewBufferString(`{"text":"Success"}`)),
						Header:     make(http.Header),
					}
				},
			},
		},
	}
	events := []Event{
		{Time: 1234567890, Event: map[string]any{"foo": "bar"}},
		{Time: 1234567891, Event: map[string]any{"baz": "qux"}},
	}
	err := client.SendEvents("main", events)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if called != len(events) {
		t.Errorf("expected %d calls, got %d", len(events), called)
	}
}

func TestSendEvents_NoToken(t *testing.T) {
	client := &Client{
		BaseURL:    "http://localhost:8089",
		Token:      "",
		HTTPClient: &http.Client{},
	}
	events := []Event{{Time: 1, Event: map[string]any{"foo": "bar"}}}
	err := client.SendEvents("main", events)
	if err == nil || !strings.Contains(err.Error(), "HEC requires a token") {
		t.Fatalf("expected HEC requires a token error, got %v", err)
	}
}

func TestSendEvents_MarshalError(t *testing.T) {
	client := &Client{
		BaseURL: "http://localhost:8089",
		Token:   "test-token",
		HTTPClient: &http.Client{
			Transport: &mockRoundTripper{
				fn: func(req *http.Request) *http.Response {
					return &http.Response{
						StatusCode: http.StatusOK,
						Body:       io.NopCloser(bytes.NewBufferString(`ok`)),
						Header:     make(http.Header),
					}
				},
			},
		},
	}
	// json.Marshal fails on channel type
	events := []Event{
		{Time: 1, Event: map[string]any{"bad": make(chan int)}},
	}
	err := client.SendEvents("main", events)
	if err == nil || !strings.Contains(err.Error(), "failed to marshal event") {
		t.Fatalf("expected marshal error, got %v", err)
	}
}

func TestSendEvents_RequestError(t *testing.T) {
	client := &Client{
		BaseURL: "http://localhost:8089",
		Token:   "test-token",
		HTTPClient: &http.Client{
			Transport: &mockErrorRoundTripper{},
		},
	}
	events := []Event{{Time: 1, Event: map[string]any{"foo": "bar"}}}
	err := client.SendEvents("main", events)
	if err == nil || !strings.Contains(err.Error(), "failed to send event") {
		t.Fatalf("expected failed to send event error, got %v", err)
	}
}

func TestSendEvents_NonOKStatus(t *testing.T) {
	client := &Client{
		BaseURL: "http://localhost:8089",
		Token:   "test-token",
		HTTPClient: &http.Client{
			Transport: &mockRoundTripper{
				fn: func(req *http.Request) *http.Response {
					return &http.Response{
						StatusCode: http.StatusBadRequest,
						Status:     "400 Bad Request",
						Body:       io.NopCloser(bytes.NewBufferString("bad request")),
						Header:     make(http.Header),
					}
				},
			},
		},
	}
	events := []Event{{Time: 1, Event: map[string]any{"foo": "bar"}}}
	err := client.SendEvents("main", events)
	if err == nil || !strings.Contains(err.Error(), "failed to send event: 400 Bad Request") {
		t.Fatalf("expected failed to send event error, got %v", err)
	}
}
func TestSetAuthHeader_WithToken(t *testing.T) {
	client := &Client{
		Token: "my-token",
	}
	req, _ := http.NewRequest("GET", "http://example.com", nil)
	err := client.setAuthHeader(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	auth := req.Header.Get("Authorization")
	if auth != "Bearer my-token" {
		t.Errorf("expected Authorization 'Bearer my-token', got '%s'", auth)
	}
}

func TestSetAuthHeader_WithUsernamePassword(t *testing.T) {
	client := &Client{
		Username: "user",
		Password: "pass",
	}
	req, _ := http.NewRequest("GET", "http://example.com", nil)
	err := client.setAuthHeader(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	username, password, ok := req.BasicAuth()
	if !ok {
		t.Errorf("expected basic auth to be set")
	}
	if username != "user" || password != "pass" {
		t.Errorf("expected basic auth user/pass to be set, got %s/%s", username, password)
	}
}

func TestSetAuthHeader_NoCredentials(t *testing.T) {
	client := &Client{}
	req, _ := http.NewRequest("GET", "http://example.com", nil)
	err := client.setAuthHeader(req)
	if err == nil || err.Error() != "no authentication credentials provided" {
		t.Errorf("expected error for missing credentials, got %v", err)
	}
}

func TestSetAuthHeader_TokenTakesPrecedence(t *testing.T) {
	client := &Client{
		Token:    "token123",
		Username: "user",
		Password: "pass",
	}
	req, _ := http.NewRequest("GET", "http://example.com", nil)
	err := client.setAuthHeader(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	auth := req.Header.Get("Authorization")
	if auth != "Bearer token123" {
		t.Errorf("expected Authorization 'Bearer token123', got '%s'", auth)
	}
	username, password, ok := req.BasicAuth()
	if ok && (username != "" || password != "") {
		t.Errorf("expected no basic auth when token is set")
	}
}
