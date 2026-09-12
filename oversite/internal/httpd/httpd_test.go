package httpd

import (
	"context"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestNewValidatesConfiguration(t *testing.T) {
	directory := t.TempDir()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	for name, configuration := range map[string]Config{
		"empty site path":     {ListenOn: "127.0.0.1:0"},
		"missing site path":   {ListenOn: "127.0.0.1:0", SitePath: filepath.Join(directory, "missing")},
		"site path is a file": {ListenOn: "127.0.0.1:0", SitePath: writeSiteFile(t, directory, "file.txt", "content")},
		"wildcard address":    {ListenOn: "0.0.0.0:8000", SitePath: directory},
		"public address":      {ListenOn: "192.0.2.1:8000", SitePath: directory},
		"name address":        {ListenOn: "localhost:8000", SitePath: directory},
		"missing port":        {ListenOn: "127.0.0.1", SitePath: directory},
		"non-numeric port":    {ListenOn: "127.0.0.1:http", SitePath: directory},
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := New(configuration, logger); err == nil {
				t.Fatal("expected configuration error")
			}
		})
	}

	for _, address := range []string{"127.0.0.1:0", "127.1.2.3:65535", "[::1]:8000"} {
		t.Run(address, func(t *testing.T) {
			if _, err := New(Config{ListenOn: address, SitePath: directory}, logger); err != nil {
				t.Fatalf("expected valid loopback address: %v", err)
			}
		})
	}
}

func TestSchemaRequiresSitePath(t *testing.T) {
	fields := Schema()
	if len(fields) != 2 || fields[0].Default != defaultListenOn || fields[1].Key != "httpd.site_path" || fields[1].Default != "" {
		t.Fatalf("unexpected schema: %#v", fields)
	}
}

func TestServeStaticFileAndStopsOnContextCancellation(t *testing.T) {
	directory := t.TempDir()
	writeSiteFile(t, directory, "index.html", "hello from oversite")
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	server, err := New(Config{ListenOn: "127.0.0.1:0", SitePath: directory}, logger)
	if err != nil {
		t.Fatal(err)
	}
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	result := make(chan error, 1)
	go func() { result <- server.serve(ctx, listener) }()

	response, err := http.Get("http://" + listener.Addr().String() + "/")
	if err != nil {
		cancel()
		t.Fatal(err)
	}
	body, err := io.ReadAll(response.Body)
	response.Body.Close()
	if err != nil {
		cancel()
		t.Fatal(err)
	}
	if response.StatusCode != http.StatusOK || !strings.Contains(string(body), "hello from oversite") {
		cancel()
		t.Fatalf("unexpected response: %d %q", response.StatusCode, body)
	}

	cancel()
	select {
	case err := <-result:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("server did not stop")
	}
}

func writeSiteFile(t *testing.T, directory, name, contents string) string {
	t.Helper()
	path := filepath.Join(directory, name)
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}
