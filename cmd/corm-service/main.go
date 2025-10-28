package main

import "fmt"

// Define a variable that will be set at compile time by the CI/CD pipeline
// The default value is used if not set by the build flags.
var Version = "dev" 

func main() {
    // Print the version for logging purposes
    fmt.Printf("Hello, my-enterprise-service! Version: %s\n", Version)
    // ... rest of your service starting logic ...
}
