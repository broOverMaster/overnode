package proxyf

import (
	"bufio"
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestInternetChain(t *testing.T) {
	origin := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { fmt.Fprint(w, "plain marker") }))
	defer origin.Close()
	secure := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { fmt.Fprint(w, "secure marker") }))
	defer secure.Close()
	_, upClient, _, _ := startProxy(t, Config{})
	upstreamURL, _ := upClient.Transport.(*http.Transport).Proxy(&http.Request{})
	for _, chained := range []bool{false, true} {
		t.Run(fmt.Sprint(chained), func(t *testing.T) {
			config := Config{}
			if chained {
				config.HTTPProxy = upstreamURL.Host
			}
			_, client, _, _ := startProxy(t, config)
			roots := x509.NewCertPool()
			roots.AddCert(secure.Certificate())
			client.Transport.(*http.Transport).TLSClientConfig = &tls.Config{RootCAs: roots, MinVersion: tls.VersionTLS12}
			for _, tc := range []struct{ url, body string }{{origin.URL, "plain marker"}, {secure.URL, "secure marker"}} {
				resp, err := client.Get(tc.url)
				if err != nil {
					t.Fatal(err)
				}
				body, err := io.ReadAll(resp.Body)
				resp.Body.Close()
				if err != nil || string(body) != tc.body {
					t.Fatalf("body=%s err=%v", body, err)
				}
			}
		})
	}
}

func proxyConnection(t *testing.T, client *http.Client) net.Conn {
	t.Helper()
	u, _ := client.Transport.(*http.Transport).Proxy(&http.Request{})
	c, err := net.DialTimeout("tcp", u.Host, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	_ = c.SetDeadline(time.Now().Add(3 * time.Second))
	t.Cleanup(func() { c.Close() })
	return c
}

func TestTunnelBufferedDataHalfCloseAndShutdown(t *testing.T) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()
	originDone := make(chan struct{})
	go func() {
		defer close(originDone)
		c, err := l.Accept()
		if err != nil {
			return
		}
		defer c.Close()
		body, _ := io.ReadAll(c)
		_, _ = c.Write(body)
	}()
	_, client, _, _ := startProxy(t, Config{})
	c := proxyConnection(t, client)
	payload := strings.Repeat("buffered-data-", 100000)
	written := make(chan error, 1)
	go func() {
		_, err := fmt.Fprintf(c, "CONNECT %s HTTP/1.1\r\nHost: ignored\r\n\r\n%s", l.Addr(), payload)
		if err == nil {
			err = c.(*net.TCPConn).CloseWrite()
		}
		written <- err
	}()
	reader := bufio.NewReader(c)
	resp, err := http.ReadResponse(reader, &http.Request{Method: "CONNECT"})
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != 200 {
		t.Fatal(resp.StatusCode)
	}
	body, err := io.ReadAll(reader)
	if err != nil || !bytes.Equal(body, []byte(payload)) {
		t.Fatalf("tunnel bytes=%d err=%v", len(body), err)
	}
	if err := <-written; err != nil {
		t.Fatal(err)
	}
	<-originDone

	idle, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer idle.Close()
	accepted := make(chan net.Conn, 1)
	go func() { c, _ := idle.Accept(); accepted <- c }()
	_, client, cancel, _ := startProxy(t, Config{})
	c = proxyConnection(t, client)
	fmt.Fprintf(c, "CONNECT %s HTTP/1.1\r\nHost: ignored\r\n\r\n", idle.Addr())
	reader = bufio.NewReader(c)
	resp, err = http.ReadResponse(reader, &http.Request{Method: "CONNECT"})
	if err != nil || resp.StatusCode != 200 {
		t.Fatalf("CONNECT %v", err)
	}
	backend := <-accepted
	if backend == nil {
		t.Fatal("missing backend")
	}
	defer backend.Close()
	cancel()
	_ = backend.SetReadDeadline(time.Now().Add(time.Second))
	if _, err := backend.Read(make([]byte, 1)); err != io.EOF {
		t.Fatalf("backend not closed: %v", err)
	}
	if _, err := reader.ReadByte(); err != io.EOF {
		t.Fatalf("client not closed: %v", err)
	}
}

func TestUpstreamProtocol(t *testing.T) {
	for _, status := range []int{200, 403, 407, 502} {
		t.Run(fmt.Sprint(status), func(t *testing.T) {
			seen := make(chan string, 1)
			upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				seen <- r.Method + " " + r.RequestURI
				if status != 200 {
					w.WriteHeader(status)
					return
				}
				c, b, err := w.(http.Hijacker).Hijack()
				if err != nil {
					return
				}
				defer c.Close()
				fmt.Fprint(b, "HTTP/1.1 200 Connection Established\r\n\r\nearly upstream bytes")
				b.Flush()
			}))
			defer upstream.Close()
			_, client, _, _ := startProxy(t, Config{HTTPProxy: upstream.Listener.Addr().String()})
			c := proxyConnection(t, client)
			fmt.Fprint(c, "CONNECT unresolvable.invalid:443 HTTP/1.1\r\nHost: ignored\r\n\r\n")
			reader := bufio.NewReader(c)
			resp, err := http.ReadResponse(reader, &http.Request{Method: "CONNECT"})
			if err != nil {
				t.Fatal(err)
			}
			expected := 502
			if status == 200 {
				expected = 200
			}
			if resp.StatusCode != expected {
				t.Fatal(resp.StatusCode)
			}
			if status == 200 {
				b := make([]byte, len("early upstream bytes"))
				if _, err := io.ReadFull(reader, b); err != nil || string(b) != "early upstream bytes" {
					t.Fatalf("%q %v", b, err)
				}
			}
			if got := <-seen; got != "CONNECT unresolvable.invalid:443" {
				t.Fatal(got)
			}
		})
	}
	seen := make(chan string, 1)
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { seen <- r.RequestURI; fmt.Fprint(w, "upstream marker") }))
	defer upstream.Close()
	_, client, _, _ := startProxy(t, Config{HTTPProxy: upstream.Listener.Addr().String()})
	resp, err := client.Get("http://unresolvable.invalid/path?q=1")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if got := <-seen; got != "http://unresolvable.invalid/path?q=1" {
		t.Fatal(got)
	}
}
