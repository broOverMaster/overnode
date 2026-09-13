package proxyf

import (
	"context"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"overnode/common/pkg/lifecycle"
	"strings"
	"testing"
	"time"
)

func testLogger() *slog.Logger { return slog.New(slog.NewTextHandler(io.Discard, nil)) }

func TestConfiguration(t *testing.T) {
	for _, tc := range []struct {
		name   string
		config Config
		valid  bool
	}{
		{"minimal", Config{ListenOn: "127.0.0.1:0"}, true},
		{"wildcard default", Config{ListenOn: ":2080"}, true},
		{"local IPv6", Config{ListenOn: ":2080", LocalSite: "[::1]:8000"}, true},
		{"local missing port", Config{ListenOn: ":2080", LocalSite: "site"}, false},
		{"local URL rejected", Config{ListenOn: ":2080", LocalSite: "http://site:8000"}, false},
		{"IPv6 proxy", Config{ListenOn: ":2080", HTTPProxy: "[::1]:1080"}, true},
		{"proxy needs port", Config{ListenOn: ":2080", HTTPProxy: "proxy"}, false},
		{"proxy rejects URL", Config{ListenOn: ":2080", HTTPProxy: "http://proxy:1080"}, false},
		{"all", Config{ListenOn: "[::1]:8080", HTTPProxy: "proxy:3128", Yggstack: "stack:1080", LocalSite: "localhost:80"}, true},
		{"IPv6 yggstack", Config{ListenOn: ":2080", Yggstack: "[::1]:1080"}, true},
		{"yggstack URL rejected", Config{ListenOn: ":2080", Yggstack: "socks5://stack:1080"}, false},
		{"bad listener", Config{ListenOn: "localhost:8080"}, false},
		{"bad port", Config{ListenOn: "127.0.0.1:65536"}, false},
		{"bad proxy scheme", Config{ListenOn: "127.0.0.1:0", HTTPProxy: "https://proxy:3128"}, false},
		{"credentials", Config{ListenOn: "127.0.0.1:0", HTTPProxy: "http://user:secret@proxy:3128"}, false},
		{"missing socks port", Config{ListenOn: "127.0.0.1:0", Yggstack: "stack"}, false},
		{"local path", Config{ListenOn: "127.0.0.1:0", LocalSite: "http://site/path"}, false},
		{"unbracketed IPv6", Config{ListenOn: "127.0.0.1:0", LocalSite: "http://::1:80"}, false},
		{"empty host", Config{ListenOn: "127.0.0.1:0", LocalSite: "http://:80"}, false},
		{"fragment", Config{ListenOn: "127.0.0.1:0", LocalSite: "http://site#"}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if err := tc.config.Validate(); (err == nil) != tc.valid {
				t.Fatalf("Validate: %v, valid=%v", err, tc.valid)
			}
			var services []lifecycle.LifeCycle
			h, err := New(tc.config, testLogger(), &services)
			if tc.valid {
				if err != nil || h == nil || len(services) != 1 {
					t.Fatalf("missing component: %v", err)
				}
			} else if err == nil || h != nil || len(services) != 0 {
				t.Fatal("invalid config returned component")
			}
		})
	}
	var services []lifecycle.LifeCycle
	if h, err := New(Config{ListenOn: "127.0.0.1:0"}, nil, &services); err == nil || h != nil || len(services) != 0 {
		t.Fatal("nil logger accepted")
	}
}

func TestLifecycle(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	var services []lifecycle.LifeCycle
	handler, err := New(Config{ListenOn: listener.Addr().String()}, testLogger(), &services)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	server := handler.(*Server)
	if len(services) != 1 || services[0] != server {
		t.Fatal("handler and lifecycle must share the same component")
	}
	defer cancel()
	done := make(chan error, 1)
	go func() { done <- server.serve(ctx, listener) }()
	proxyURL, err := url.Parse("http://" + listener.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	transport := &http.Transport{Proxy: http.ProxyURL(proxyURL)}
	defer transport.CloseIdleConnections()
	client := &http.Client{Transport: transport, Timeout: time.Second}
	resp, err := client.Get("http://key.ygg/")
	if err != nil {
		t.Fatal(err)
	}
	body, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil || resp.StatusCode != 400 || !strings.Contains(string(body), "invalid public key") {
		t.Fatalf("response: %d %s %v", resp.StatusCode, body, err)
	}
	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("shutdown blocked")
	}
	conn, err := net.DialTimeout("tcp", listener.Addr().String(), time.Second)
	if err == nil {
		conn.Close()
		t.Fatal("listener is still open")
	}
}

func TestListenFailure(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	var services []lifecycle.LifeCycle
	_, err = New(Config{ListenOn: listener.Addr().String()}, testLogger(), &services)
	if err != nil {
		t.Fatal(err)
	}
	if err := services[0].Run(context.Background()); err == nil {
		t.Fatal("occupied listener accepted")
	}
}

func TestServeFailure(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	listener.Close()
	handler, err := New(Config{ListenOn: "127.0.0.1:0"}, testLogger(), new([]lifecycle.LifeCycle))
	if err != nil {
		t.Fatal(err)
	}
	if err := handler.(*Server).serve(context.Background(), listener); err == nil {
		t.Fatal("closed listener accepted")
	}
}
