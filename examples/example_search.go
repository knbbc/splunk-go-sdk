package examples

import (
	"fmt"
	"os"
	"splunk-go-sdk/pkg/splunk"
)

func ExampleSearch() {
	splunkURL := "https://localhost:8089" // Replace with your Splunk URL

	token := os.Getenv("SPLUNK_TOKEN")
	username := os.Getenv("SPLUNK_USERNAME")
	password := os.Getenv("SPLUNK_PASSWORD")

	client, err := splunk.NewClient(splunkURL, username, password, token)
	if err != nil {
		fmt.Println("Error creating Splunk client:", err)
		return
	}

	// Perform a search
	query := "search index=_internal | head 10"
	results, err := client.Search(query)
	if err != nil {
		fmt.Println("Error performing search:", err)
		return
	}

	// Print the search results
	for _, result := range results {
		fmt.Println(result)
	}
}
