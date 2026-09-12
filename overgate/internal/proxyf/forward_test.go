package proxyf

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

func startProxy(t *testing.T, c Config) (*Server, *http.Client, context.CancelFunc, <-chan error) {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	c.ListenOn = l.Addr().String()
	s, err := New(c, testLogger())
	if err != nil {
		l.Close()
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- s.serve(ctx, l) }()
	u, _ := url.Parse("http://" + l.Addr().String())
	tr := &http.Transport{Proxy: http.ProxyURL(u)}
	client := &http.Client{Transport: tr, Timeout: 3 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	t.Cleanup(func() {
		tr.CloseIdleConnections()
		cancel()
		select {
		case err := <-done:
			if err != nil {
				t.Error(err)
			}
		case <-time.After(3 * time.Second):
			t.Error("proxy shutdown blocked")
		}
	})
	return s, client, cancel, done
}

func TestForwardHTTP(t *testing.T) {
	t.Setenv("HTTP_PROXY", "http://127.0.0.1:1")
	t.Setenv("HTTPS_PROXY", "http://127.0.0.1:1")
	type observed struct{ host, uri, method, body, secret, hop, trailer string }
	seen := make(chan observed, 8)
	origin := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		seen <- observed{r.Host, r.RequestURI, r.Method, string(body), r.Header.Get("Proxy-Authorization"), r.Header.Get("X-Hop"), r.Trailer.Get("X-Request-End")}
		w.Header().Set("Connection", "X-Response-Hop")
		w.Header().Set("X-Response-Hop", "remove")
		w.Header().Set("X-End-To-End", "keep")
		w.Header().Set("Trailer", "X-Response-End")
		w.WriteHeader(201)
		_, _ = io.WriteString(w, "response-marker")
		w.Header().Set("X-Response-End", "finished")
	}))
	defer origin.Close()
	_, client, _, _ := startProxy(t, Config{LocalSite: origin.Listener.Addr().String()})
	for _, local := range []bool{true, false} {
		base := origin.URL
		expectedHost := strings.TrimPrefix(origin.URL, "http://")
		if local {
			base = "http://LOCAL.OVERSPACE.:9999"
			expectedHost = "local.overspace"
		}
		for _, method := range []string{"GET", "POST"} {
			req, _ := http.NewRequest(method, base+"/a%2Fb?x=%2F&raw=a;b", strings.NewReader("request-marker"))
			req.ContentLength = -1
			req.Trailer = http.Header{"X-Request-End": []string{"finished"}}
			req.Header.Set("Connection", "X-Hop")
			req.Header.Set("X-Hop", "remove")
			req.Header.Set("Proxy-Authorization", "secret")
			resp, err := client.Do(req)
			if err != nil {
				t.Fatal(err)
			}
			body, err := io.ReadAll(resp.Body)
			resp.Body.Close()
			if err != nil || resp.StatusCode != 201 || string(body) != "response-marker" {
				t.Fatalf("response: %d %q %v", resp.StatusCode, body, err)
			}
			if resp.Header.Get("X-Response-Hop") != "" || resp.Header.Get("X-End-To-End") != "keep" || resp.Trailer.Get("X-Response-End") != "finished" {
				t.Fatalf("headers/trailers: %v %v", resp.Header, resp.Trailer)
			}
			got := <-seen
			if got.host != expectedHost || got.uri != "/a%2Fb?x=%2F&raw=a;b" || got.method != method || got.body != "request-marker" || got.secret != "" || got.hop != "" || got.trailer != "finished" {
				t.Fatalf("origin request: %+v", got)
			}
		}
	}
}

func TestRedirectAndIsolation(t *testing.T) {
	origin := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Location", "http://unreachable.invalid/")
		w.WriteHeader(302)
	}))
	defer origin.Close()
	_, client, _, _ := startProxy(t, Config{LocalSite: origin.Listener.Addr().String(), HTTPProxy: "127.0.0.1:1"})
	for _, tc := range []struct {
		url    string
		status int
	}{{"http://local.overspace/", 302}, {origin.URL, 502}, {"http://key.ygg/", 501}, {"http://key.overspace/", 501}} {
		resp, err := client.Get(tc.url)
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
		if resp.StatusCode != tc.status {
			t.Fatalf("%s: %d", tc.url, resp.StatusCode)
		}
	}
}

func TestUpstreamTimeoutAndShutdown(t *testing.T) {
	entered := make(chan struct{}, 2)
	canceled := make(chan struct{}, 2)
	origin := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		entered <- struct{}{}
		<-r.Context().Done()
		canceled <- struct{}{}
	}))
	defer origin.Close()
	s, client, cancel, _ := startProxy(t, Config{LocalSite: origin.Listener.Addr().String()})
	s.localTransport.ResponseHeaderTimeout = 30 * time.Millisecond
	resp, err := client.Get("http://local.overspace/")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != 504 {
		t.Fatal(resp.StatusCode)
	}
	<-entered
	select {
	case <-canceled:
	case <-time.After(time.Second):
		t.Fatal("backend not canceled after timeout")
	}
	result := make(chan error, 1)
	go func() {
		resp, err := client.Get(origin.URL)
		if resp != nil {
			resp.Body.Close()
		}
		result <- err
	}()
	<-entered
	cancel()
	select {
	case <-result:
	case <-time.After(time.Second):
		t.Fatal("request not canceled on shutdown")
	}
	select {
	case <-canceled:
	case <-time.After(time.Second):
		t.Fatal("backend not canceled on shutdown")
	}
}

func TestFailureAndRecovery(t *testing.T) {
	origin := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { _, _ = io.WriteString(w, "recovered") }))
	address := origin.Listener.Addr().String()
	s, client, _, _ := startProxy(t, Config{LocalSite: origin.Listener.Addr().String()})
	for _, destination := range []string{"http://local.overspace/", origin.URL} {
		resp, err := client.Get(destination)
		if err != nil {
			t.Fatal(err)
		}
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
	}
	origin.Close()
	s.localTransport.CloseIdleConnections()
	s.internetTransport.CloseIdleConnections()
	for _, destination := range []string{"http://local.overspace/", "http://" + address} {
		resp, err := client.Get(destination)
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
		if resp.StatusCode != 502 {
			t.Fatal(resp.StatusCode)
		}
	}
	l, err := net.Listen("tcp", address)
	if err != nil {
		t.Fatal(err)
	}
	recovered := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { fmt.Fprint(w, "recovered") }))
	recovered.Listener.Close()
	recovered.Listener = l
	recovered.Start()
	defer recovered.Close()
	for _, destination := range []string{"http://local.overspace/", "http://" + address} {
		resp, err := client.Get(destination)
		if err != nil {
			t.Fatal(err)
		}
		b, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		if resp.StatusCode != 200 || string(b) != "recovered" {
			t.Fatalf("%d %s", resp.StatusCode, b)
		}
	}
}

func TestDefaultInternetPort(t *testing.T) {
	origin := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { fmt.Fprint(w, r.Host) }))
	defer origin.Close()
	s, client, _, _ := startProxy(t, Config{})
	dialed := make(chan string, 1)
	s.internetTransport.DialContext = func(ctx context.Context, network, address string) (net.Conn, error) {
		dialed <- address
		return (&net.Dialer{}).DialContext(ctx, network, origin.Listener.Addr().String())
	}
	resp, err := client.Get("http://example.test/")
	if err != nil {
		t.Fatal(err)
	}
	b, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if got := <-dialed; got != "example.test:80" {
		t.Fatal(got)
	}
	if string(b) != "example.test" {
		t.Fatal(string(b))
	}
}
