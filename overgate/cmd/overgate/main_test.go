package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestRunHelpAndErrors(t *testing.T) {
	for _, tc := range []struct {
		args []string
		code int
	}{
		{[]string{"--help"}, 0},
		{[]string{"--unknown"}, 1},
		{[]string{"--proxyf.listen_on=invalid"}, 1},
		{[]string{"--proxyf.http_proxy=http://user:secret@proxy:80"}, 1},
	} {
		var stdout, stderr bytes.Buffer
		if got := run(tc.args, &stdout, &stderr); got != tc.code {
			t.Fatalf("%v: code %d", tc.args, got)
		}
		if tc.code == 0 && (!strings.Contains(stdout.String(), "proxyf.listen_on") || stderr.Len() != 0) {
			t.Fatalf("invalid help: %s %s", &stdout, &stderr)
		}
		if strings.Contains(stderr.String(), "secret") {
			t.Fatal("credentials leaked")
		}
	}
}
