package examples

import (
	"fmt"
	"os"
	"splunk-go-sdk/pkg/splunk"
	"time"
)

func ExampleSendEvents() {
	splunkURL := "https://localhost:8089" // Replace with your Splunk URL

	token := os.Getenv("SPLUNK_TOKEN")
	username := os.Getenv("SPLUNK_USERNAME")
	password := os.Getenv("SPLUNK_PASSWORD")

	client, err := splunk.NewClient(splunkURL, username, password, token)
	if err != nil {
		fmt.Println("Error creating Splunk client:", err)
		return
	}
	// Send an event
	t, err := time.Parse(time.RFC3339, "2023-01-01T00:00:00Z")
	if err != nil {
		fmt.Println("Error parsing time:", err)
		return
	}
	event := splunk.Event{
		Time:  t.Unix(),
		Event: map[string]interface{}{"message": "Hello, Splunk!"},
	}

	err = client.SendEvents("test_index", []splunk.Event{event})
	if err != nil {
		fmt.Println("Error sending event:", err)
		return
	}

	fmt.Println("Event sent successfully!")
}
