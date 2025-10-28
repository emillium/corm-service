package main

import (
	"fmt"
	"io"
	"os"
	"strings"
	"testing"
)

// This test mocks standard output (stdout) to capture what the run() function prints.
func TestRun(t *testing.T) {
	// Setup: Capture the standard output
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Run the function to be tested
	err := run()

	// Teardown: Restore standard output and close the pipe
	w.Close()
	out, _ := io.ReadAll(r)
	os.Stdout = oldStdout

	// 1. Test for error during execution
	if err != nil {
		t.Fatalf("run() failed unexpectedly with error: %v", err)
	}

	// 2. Test for output content
	output := string(out)

	// Check if the service version was printed
	expectedVersionPrefix := fmt.Sprintf("Service Version: %s", Version)
	if !strings.Contains(output, expectedVersionPrefix) {
		t.Errorf("run() output does not contain expected version string. Got: %q", output)
	}

	// Check for another expected startup message
	expectedRunMessage := "Service running..."
	if !strings.Contains(output, expectedRunMessage) {
		t.Errorf("run() output does not contain expected startup message. Got: %q", output)
	}
}
