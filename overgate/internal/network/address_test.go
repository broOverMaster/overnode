package network

import "testing"

func TestParseAuthority(t *testing.T) {
	for _, testCase := range []struct {
		value       string
		requirePort bool
		host, port  string
		valid       bool
	}{
		{"site:8080", true, "site", "8080", true},
		{"site", false, "site", "80", true},
		{"[::1]:8080", true, "::1", "8080", true},
		{"site", true, "", "", false},
		{"site:0", true, "", "", false},
		{"http://site:8080", true, "", "", false},
		{"user:secret@site:8080", true, "", "", false},
		{"::1:8080", true, "", "", false},
		{"[invalid]:8080", true, "", "", false},
	} {
		t.Run(testCase.value, func(t *testing.T) {
			host, port, err := ParseAuthority(testCase.value, testCase.requirePort)
			if (err == nil) != testCase.valid {
				t.Fatalf("ParseAuthority: %v", err)
			}
			if testCase.valid && (host != testCase.host || port != testCase.port) {
				t.Fatalf("got %q:%q, want %q:%q", host, port, testCase.host, testCase.port)
			}
		})
	}
}
