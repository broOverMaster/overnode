package proxyf

import (
	"context"
	"io"
	"log/slog"
	"net"
	"net/http"
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
		{"all", Config{"[::1]:8080", "http://proxy:3128", "socks5://stack:1080", "http://localhost"}, true},
		{"bad listener", Config{ListenOn: "localhost:8080"}, false},
		{"bad port", Config{ListenOn: "127.0.0.1:65536"}, false},
		{"bad proxy scheme", Config{ListenOn: "127.0.0.1:0", HTTPProxy: "https://proxy:3128"}, false},
		{"credentials", Config{ListenOn: "127.0.0.1:0", HTTPProxy: "http://user:secret@proxy:3128"}, false},
		{"missing socks port", Config{ListenOn: "127.0.0.1:0", YggstackProxy: "socks5://stack"}, false},
		{"local path", Config{ListenOn: "127.0.0.1:0", LocalSite: "http://site/path"}, false},
		{"unbracketed IPv6", Config{ListenOn: "127.0.0.1:0", LocalSite: "http://::1:80"}, false},
		{"empty host", Config{ListenOn: "127.0.0.1:0", LocalSite: "http://:80"}, false},
		{"fragment", Config{ListenOn: "127.0.0.1:0", LocalSite: "http://site#"}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if err := tc.config.Validate(); (err == nil) != tc.valid {
				t.Fatalf("Validate: %v, valid=%v", err, tc.valid)
			}
		})
	}
	if _, err := New(Config{ListenOn: "127.0.0.1:0"}, nil); err == nil {
		t.Fatal("nil logger accepted")
	}
}

func TestLifecycle(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	server, err := New(Config{ListenOn: listener.Addr().String()}, testLogger())
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() { done <- server.serve(ctx, listener) }()
	transport := &http.Transport{Proxy: nil}
	defer transport.CloseIdleConnections()
	client := &http.Client{Transport: transport, Timeout: time.Second}
	resp, err := client.Get("http://" + listener.Addr().String() + "/")
	if err != nil {
		t.Fatal(err)
	}
	body, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil || resp.StatusCode != 501 || !strings.Contains(string(body), "not implemented") {
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
	server, err := New(Config{ListenOn: listener.Addr().String()}, testLogger())
	if err != nil {
		t.Fatal(err)
	}
	if err := server.Run(context.Background()); err == nil {
		t.Fatal("occupied listener accepted")
	}
}

func TestServeFailure(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	listener.Close()
	server, err := New(Config{ListenOn: "127.0.0.1:0"}, testLogger())
	if err != nil {
		t.Fatal(err)
	}
	if err := server.serve(context.Background(), listener); err == nil {
		t.Fatal("closed listener accepted")
	}
}
