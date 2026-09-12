package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestRunPrintsHelp(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if code := run([]string{"--help"}, &stdout, &stderr); code != 0 {
		t.Fatalf("got exit code %d", code)
	}
	if !strings.Contains(stdout.String(), "Usage: oversite [flags]") || stderr.Len() != 0 {
		t.Fatalf("unexpected help output: stdout=%q stderr=%q", stdout.String(), stderr.String())
	}
}
