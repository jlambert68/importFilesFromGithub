package main

import (
	"log"
	"os"
)

// mustGetEnv is a helper function for getting environment variables.
// Displays a warning if the environment variable is not set.
func mustGetenv(k string) string {
	v := os.Getenv(k)
	if v == "" {
		log.Fatalf("Warning: %s environment variable not set.\n", k)
	}
	return v
}

var gitHubApiKey string

func init() {

	// Get Environment variable to tell were Fenix Execution Server is running
	gitHubApiKey = mustGetenv("gitHubApiKey")

}
