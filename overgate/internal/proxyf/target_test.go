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
	"testing"
	"time"
)

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
