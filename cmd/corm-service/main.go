package main

import (
	"fmt"
	"os"
)

// This variable is set by the CI/CD build process
var Version = "dev"

func run() error {
	// 1. Load Configuration
	// 2. Initialize Dependencies (Logger, Database)
	// 3. Start Router/Server

	fmt.Printf("Service Version: %s\n", Version)
	fmt.Println("Service running...")
	
	// Simulate a successful startup
	return nil 
}

func main() {
	if err := run(); err != nil {
		// Log the error before exiting
		fmt.Printf("Fatal error during startup: %v\n", err) 
		os.Exit(1)
	}
}
