package main

import (
	"fmt"
	"os"
	"splunk-go-sdk/internal/splunk"
)

func main() {
	// Example usage:
	// To use with a token:
	// export SPLUNK_TOKEN="your_token"
	// To use with username and password:
	// export SPLUNK_USERNAME="your_username"
	// export SPLUNK_PASSWORD="your_password"

	splunkURL := "https://localhost:8089" // Replace with your Splunk URL

	token := os.Getenv("SPLUNK_TOKEN")
	username := os.Getenv("SPLUNK_USERNAME")
	password := os.Getenv("SPLUNK_PASSWORD")

	client, err := splunk.NewClient(splunkURL, username, password, token)
	if err != nil {
		fmt.Println("Error creating Splunk client:", err)
		return
	}

	fmt.Println("Splunk client created successfully!")
	if client.Token != "" {
		fmt.Println("Authentication method: Token")
	} else {
		fmt.Println("Authentication method: Username/Password")
	}
}
