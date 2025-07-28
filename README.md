# splunk-go-sdk

This library provides a convenient and idiomatic Go interface for the Splunk REST API. It is designed to simplify the process of integrating Go applications with Splunk, allowing developers to programmatically search, manage, and ingest data without having to handle the underlying HTTP requests and responses manually.

## Features

*   **Simplified API access:** Interact with Splunk endpoints using clean and intuitive Go methods.
*   **Type-safe data handling:** Work with Go structs that map to Splunk API objects.
*   **Error handling:** Consistent error handling for API responses.
*   **Extensible:** Easily extend the library to support additional Splunk endpoints.

## Getting Started

To get started, you need to initialize a new Splunk client. You can do this by providing your Splunk URL and authentication credentials (either a username/password or a token).

```go
package main

import (
	"fmt"
	"log"
	"time"
)

func main() {
	// Splunk instance details
	splunkURL := "https://your-splunk-instance:8089"
	username := "your-username"
	password := "your-password"
	token := "your-splunk-token"

	// Initialize the client with username and password
	client, err := NewClient(splunkURL, username, password, "")
	if err != nil {
		log.Fatalf("Failed to create Splunk client: %v", err)
	}

	// Or, initialize the client with a token
	// client, err := NewClient(splunkURL, "", "", token)
	// if err != nil {
	// 	log.Fatalf("Failed to create Splunk client: %v", err)
	// }

	fmt.Println("Successfully created Splunk client")

    // ... see Usage section for how to use the client
}
```

## Usage

### Searching

To perform a search, use the `Search` method with your Splunk query.

```go
	// Example: Perform a search
	searchQuery := "search index=_internal | head 10"
	results, err := client.Search(searchQuery)
	if err != nil {
		log.Fatalf("Failed to execute search: %v", err)
	}

	fmt.Printf("Search results: %v\n", results)
```

### Sending Events

To send events to a Splunk index, create a slice of `Event` structs and use the `SendEvents` method.

```go
	// Example: Send events to an index
	indexName := "main"
	events := []Event{
		{
			Time:  time.Now().Unix(),
			Event: map[string]interface{}{"message": "This is a test event 1"},
		},
		{
			Time:  time.Now().Unix(),
			Event: "This is a raw string event 2",
		},
	}

	if err := client.SendEvents(indexName, events); err != nil {
		log.Fatalf("Failed to send events: %v", err)
	}

	fmt.Println("Successfully sent events to Splunk")
```