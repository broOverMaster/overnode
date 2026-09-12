package logging

import (
	"bytes"
	"strings"
	"testing"
)

func TestTextLoggingPreservesErrorFields(t *testing.T) {
	var output bytes.Buffer
	logger, closeLogger, err := New(Config{Level: "info", Format: "text"}, &output)
	if err != nil {
		t.Fatal(err)
	}
	defer closeLogger()
	logger.With("component", "relay").Error("name lookup failed", "operation", "resolve", "error_code", "name_not_found", "host", "alice.overspace")
	for _, field := range []string{"level=ERROR", "component=relay", "operation=resolve", "error_code=name_not_found", "host=alice.overspace"} {
		if !strings.Contains(output.String(), field) {
			t.Errorf("missing %s: %s", field, output.String())
		}
	}
}
