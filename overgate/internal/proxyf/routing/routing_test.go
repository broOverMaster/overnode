package routing

import (
	"net/http"
	"strings"
	"testing"
)

func TestParse(t *testing.T) {
	for _, tc := range []struct {
		uri, method string
		route       Route
		host, port  string
	}{
		{"http://LOCAL.OVERSPACE.:81/a%2Fb?q=%2F", "GET", Local, "local.overspace", "81"},
		{"http://notlocal.overspace/", "POST", Overspace, "notlocal.overspace", "80"},
		{"http://key.ygg/", "GET", Ygg, "key.ygg", "80"},
		{"http://sub.key.ygg/", "GET", Ygg, "sub.key.ygg", "80"},
		{"http://" + strings.Repeat("a", 64) + ".overspace/", "GET", Overspace, strings.Repeat("a", 64) + ".overspace", "80"},
		{"http://local.overspace.example/", "GET", Internet, "local.overspace.example", "80"},
		{"http://key.ygg.example/", "GET", Internet, "key.ygg.example", "80"},
		{"http://overspace/", "GET", Internet, "overspace", "80"},
		{"http://127.0.0.1/", "GET", Internet, "127.0.0.1", "80"},
		{"http://[::1]:8081/", "GET", Internet, "::1", "8081"},
		{"example.com:443", "CONNECT", Internet, "example.com", "443"},
		{"[::1]:443", "CONNECT", Internet, "::1", "443"},
		{"LOCAL.OVERSPACE.:443", "CONNECT", Local, "local.overspace", "443"},
	} {
		t.Run(tc.uri, func(t *testing.T) {
			request := &http.Request{Method: tc.method, RequestURI: tc.uri, Host: "untrusted.ygg"}
			target, err := Parse(request)
			if err != nil {
				t.Fatal(err)
			}
			if target.Route != tc.route || target.Hostname != tc.host || target.Port != tc.port {
				t.Fatalf("unexpected target: %+v", target)
			}
			if target.URL != nil && target.URL.String() != tc.uri {
				t.Fatalf("URL changed: %s", target.URL)
			}
			if request.Host != "untrusted.ygg" {
				t.Fatal("request mutated")
			}
		})
	}
}

func TestInvalidTargets(t *testing.T) {
	for _, uri := range []string{"/path", "*", "", "//host/path", "https://host/", "ftp://host/", "http:host", "http:///path", "http://u:p@host/", "http://host/#x", "http://host/#", "http://host:0/", "http://host:65536/", "http://host:/", "http://host:abc/", "http://::1:80/", "http://[abc]:80/", "http://a..ygg/", "http://.ygg/", "http://a.ygg../", "http://-a.ygg/", "http://host/%zz"} {
		if _, err := Parse(&http.Request{Method: "GET", RequestURI: uri, Host: "example.com"}); err == nil {
			t.Errorf("accepted %q", uri)
		}
	}
	for _, uri := range []string{"host", "host:", "host:0", "host:65536", "host:443/path", "http://host:443", "user@host:443", "::1:443"} {
		if _, err := Parse(&http.Request{Method: "CONNECT", RequestURI: uri}); err == nil {
			t.Errorf("accepted CONNECT %q", uri)
		}
	}
}
