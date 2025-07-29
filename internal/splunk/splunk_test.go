package splunk

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestNewClient(t *testing.T) {
	_, err := NewClient("http://localhost", "", "", "")
	if err == nil {
		t.Error("expected error when no credentials are provided")
	}

	client, err := NewClient("http://localhost", "user", "pass", "")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if client.Username != "user" {
		t.Error("username not set correctly")
	}

	client, err = NewClient("http://localhost", "", "", "token123")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if client.Token != "token123" {
		t.Error("token not set correctly")
	}
}

func TestSetAuthHeader_Token(t *testing.T) {
	req, _ := http.NewRequest("GET", "http://localhost", nil)
	c := &Client{Token: "abc123"}
	err := c.setAuthHeader(req)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if got := req.Header.Get("Authorization"); got != "Bearer abc123" {
		t.Errorf("expected Bearer token, got %s", got)
	}
}

func TestSetAuthHeader_BasicAuth(t *testing.T) {
	req, _ := http.NewRequest("GET", "http://localhost", nil)
	c := &Client{Username: "user", Password: "pass"}
	err := c.setAuthHeader(req)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	username, password, ok := req.BasicAuth()
	if !ok || username != "user" || password != "pass" {
		t.Error("basic auth not set correctly")
	}
}

func TestSetAuthHeader_NoAuth(t *testing.T) {
	req, _ := http.NewRequest("GET", "http://localhost", nil)
	c := &Client{}
	err := c.setAuthHeader(req)
	if err == nil {
		t.Error("expected error when no credentials are set")
	}
}

func TestPrepareHttpRequest(t *testing.T) {
	c := &Client{BaseURL: "http://localhost", Token: "tok"}
	req, err := c.prepareHttpRequest("search index=_internal")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if req.Method != "POST" {
		t.Error("expected POST method")
	}
	if req.Header.Get("Authorization") != "Bearer tok" {
		t.Error("expected Authorization header")
	}
}

func TestParseSplunkSearchResults(t *testing.T) {
	body := `{"results":[{"field":"value"}]}`
	r := bytes.NewReader([]byte(body))
	result, err := parseSplunkSearchResults(r)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	results, ok := result["results"].([]any)
	if !ok || len(results) != 1 {
		t.Error("failed to parse results")
	}
}

func TestSearch_Success(t *testing.T) {
	// Mock Splunk search endpoint
	handler := func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		io.WriteString(w, `{"results":[{"foo":"bar"}]}`)
	}
	server := httptest.NewServer(http.HandlerFunc(handler))
	defer server.Close()

	client, _ := NewClient(server.URL, "user", "pass", "")
	client.HTTPClient = server.Client()

	result, err := client.Search("search index=_internal")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	results, ok := result["results"].([]any)
	if !ok || len(results) != 1 {
		t.Error("unexpected search results")
	}
}

func TestSearch_Failure(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		io.WriteString(w, "bad request")
	}
	server := httptest.NewServer(http.HandlerFunc(handler))
	defer server.Close()

	client, _ := NewClient(server.URL, "user", "pass", "")
	client.HTTPClient = server.Client()

	_, err := client.Search("search index=_internal")
	if err == nil {
		t.Error("expected error on bad status")
	}
}

func TestSendEvents_Success(t *testing.T) {
	var received []map[string]any
	handler := func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Splunk token123" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		var payload map[string]any
		json.NewDecoder(r.Body).Decode(&payload)
		received = append(received, payload)
		w.WriteHeader(http.StatusOK)
	}
	server := httptest.NewServer(http.HandlerFunc(handler))
	defer server.Close()

	client, _ := NewClient(server.URL, "", "", "token123")
	client.HTTPClient = server.Client()

	events := []Event{
		{Time: time.Now().Unix(), Event: map[string]any{"foo": "bar"}},
	}
	err := client.SendEvents("main", events)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(received) != 1 {
		t.Error("event not received")
	}
	if received[0]["index"] != "main" {
		t.Error("index not set correctly")
	}
}

func TestSendEvents_NoToken(t *testing.T) {
	client, _ := NewClient("http://localhost", "user", "pass", "")
	err := client.SendEvents("main", []Event{{Time: 1, Event: map[string]any{}}})
	if err == nil {
		t.Error("expected error when no token is set")
	}
}

func TestSendEvents_Failure(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "fail", http.StatusBadRequest)
	}
	server := httptest.NewServer(http.HandlerFunc(handler))
	defer server.Close()

	client, _ := NewClient(server.URL, "", "", "token123")
	client.HTTPClient = server.Client()

	events := []Event{
		{Time: time.Now().Unix(), Event: map[string]any{"foo": "bar"}},
	}
	err := client.SendEvents("main", events)
	if err == nil {
		t.Error("expected error on failed send")
	}
}

func TestSendEvents_MarshalError(t *testing.T) {
	client, _ := NewClient("http://localhost", "", "", "token123")
	client.HTTPClient = &http.Client{}
	// Event.Event contains a channel, which cannot be marshaled to JSON
	events := []Event{
		{Time: 1, Event: map[string]any{"bad": make(chan int)}},
	}
	err := client.SendEvents("main", events)
	if err == nil {
		t.Error("expected marshal error")
	}
}

func TestSendEvents_RequestError(t *testing.T) {
	client, _ := NewClient("http://localhost", "", "", "token123")
	client.HTTPClient = &http.Client{}
	// Invalid URL to force request creation error
	client.BaseURL = "http://[::1]:NamedPort"
	events := []Event{
		{Time: 1, Event: map[string]any{"foo": "bar"}},
	}
	err := client.SendEvents("main", events)
	if err == nil {
		t.Error("expected error on request creation")
	}
}

func TestSendEvents_DoError(t *testing.T) {
	client, _ := NewClient("http://localhost", "", "", "token123")
	client.HTTPClient = &http.Client{
		Transport: roundTripperFunc(func(*http.Request) (*http.Response, error) {
			return nil, fmt.Errorf("network error")
		}),
	}
	events := []Event{
		{Time: 1, Event: map[string]any{"foo": "bar"}},
	}
	err := client.SendEvents("main", events)
	if err == nil {
		t.Error("expected error on HTTP Do")
	}
}

// roundTripperFunc is a helper to mock http.RoundTripper
type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}
