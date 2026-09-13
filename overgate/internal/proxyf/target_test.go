package proxyf

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"overnode/common/pkg/lifecycle"
	"strings"
	"testing"
	"time"
)

func TestParseTarget(t *testing.T) {
	for _, tc := range []struct {
		uri, method string
		route       route
		host, port  string
	}{
		{"http://LOCAL.OVERSPACE.:81/a%2Fb?q=%2F", "GET", routeLocal, "local.overspace", "81"},
		{"http://notlocal.overspace/", "POST", routeOverspace, "notlocal.overspace", "80"},
		{"http://key.ygg/", "GET", routeYgg, "key.ygg", "80"},
		{"http://sub.key.ygg/", "GET", routeYgg, "sub.key.ygg", "80"},
		{"http://" + strings.Repeat("a", 64) + ".overspace/", "GET", routeOverspace, strings.Repeat("a", 64) + ".overspace", "80"},
		{"http://local.overspace.example/", "GET", routeInternet, "local.overspace.example", "80"},
		{"http://key.ygg.example/", "GET", routeInternet, "key.ygg.example", "80"},
		{"http://overspace/", "GET", routeInternet, "overspace", "80"},
		{"http://127.0.0.1/", "GET", routeInternet, "127.0.0.1", "80"},
		{"http://[::1]:8081/", "GET", routeInternet, "::1", "8081"},
		{"example.com:443", "CONNECT", routeInternet, "example.com", "443"},
		{"[::1]:443", "CONNECT", routeInternet, "::1", "443"},
		{"LOCAL.OVERSPACE.:443", "CONNECT", routeLocal, "local.overspace", "443"},
	} {
		t.Run(tc.uri, func(t *testing.T) {
			r := &http.Request{Method: tc.method, RequestURI: tc.uri, Host: "untrusted.ygg"}
			got, err := parseTarget(r)
			if err != nil {
				t.Fatal(err)
			}
			if got.route != tc.route || got.hostname != tc.host || got.port != tc.port {
				t.Fatalf("unexpected target: %+v", got)
			}
			if got.url != nil && got.url.String() != tc.uri {
				t.Fatalf("URL changed: %s", got.url)
			}
			if r.Host != "untrusted.ygg" {
				t.Fatal("request mutated")
			}
		})
	}
}

func TestInvalidTargets(t *testing.T) {
	for _, uri := range []string{"/path", "*", "", "//host/path", "https://host/", "ftp://host/", "http:host", "http:///path", "http://u:p@host/", "http://host/#x", "http://host/#", "http://host:0/", "http://host:65536/", "http://host:/", "http://host:abc/", "http://::1:80/", "http://[abc]:80/", "http://a..ygg/", "http://.ygg/", "http://a.ygg../", "http://-a.ygg/", "http://host/%zz"} {
		if _, err := parseTarget(&http.Request{Method: "GET", RequestURI: uri, Host: "example.com"}); err == nil {
			t.Errorf("accepted %q", uri)
		}
	}
	for _, uri := range []string{"host", "host:", "host:0", "host:65536", "host:443/path", "http://host:443", "user@host:443", "::1:443"} {
		if _, err := parseTarget(&http.Request{Method: "CONNECT", RequestURI: uri}); err == nil {
			t.Errorf("accepted CONNECT %q", uri)
		}
	}
}

func TestHandlerStatus(t *testing.T) {
	s, err := New(Config{ListenOn: ":2080", HTTPProxy: "127.0.0.1:1"}, testLogger(), new([]lifecycle.LifeCycle))
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		method, uri string
		code        int
	}{
		{"GET", "/", 400}, {"OPTIONS", "*", 400}, {"GET", "http://example.com/", 502},
		{"GET", "http://local.overspace/", 503}, {"GET", "http://invalid.ygg/", 400},
		{"CONNECT", "example.com:443", 502}, {"CONNECT", "local.overspace:443", 405},
		{"CONNECT", "invalid.ygg:443", 405}, {"CONNECT", "invalid.overspace:443", 405},
	} {
		w := httptest.NewRecorder()
		s.ServeHTTP(w, &http.Request{Method: tc.method, RequestURI: tc.uri, Host: "example.com", Header: make(http.Header), URL: &url.URL{}})
		if w.Code != tc.code {
			t.Errorf("%s %s: %d", tc.method, tc.uri, w.Code)
		}
		if tc.code == 405 && w.Header().Get("Allow") == "" {
			t.Error("missing Allow")
		}
	}
}

func TestWireRequestForms(t *testing.T) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	s, err := New(Config{ListenOn: l.Addr().String()}, testLogger(), new([]lifecycle.LifeCycle))
	if err != nil {
		l.Close()
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- s.(*Server).serve(ctx, l) }()
	defer func() {
		cancel()
		select {
		case err := <-done:
			if err != nil {
				t.Error(err)
			}
		case <-time.After(2 * time.Second):
			t.Error("shutdown timeout")
		}
	}()
	for _, tc := range []struct {
		line string
		code int
	}{
		{"GET / HTTP/1.1", 400},
		{"GET http://local.overspace/a%2Fb?q=1 HTTP/1.1", 503},
		{"CONNECT local.overspace:443 HTTP/1.1", 405},
		{"CONNECT 127.0.0.1:1 HTTP/1.1", 502},
	} {
		c, err := net.DialTimeout("tcp", l.Addr().String(), time.Second)
		if err != nil {
			t.Fatal(err)
		}
		_ = c.SetDeadline(time.Now().Add(time.Second))
		_, err = fmt.Fprintf(c, "%s\r\nHost: different.example\r\nConnection: close\r\n\r\n", tc.line)
		if err != nil {
			c.Close()
			t.Fatal(err)
		}
		resp, err := http.ReadResponse(bufio.NewReader(c), nil)
		if err != nil {
			c.Close()
			t.Fatal(err)
		}
		_, _ = io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
		c.Close()
		if resp.StatusCode != tc.code {
			t.Errorf("%s: %d", tc.line, resp.StatusCode)
		}
	}
}
