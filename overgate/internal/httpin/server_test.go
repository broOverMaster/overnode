package httpin

import (
	"context"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"overnode/gate/internal/lifecycle"
	"strings"
	"testing"
	"time"
)

func logger() *slog.Logger { return slog.New(slog.NewTextHandler(io.Discard, nil)) }

func TestConfiguration(t *testing.T) {
	for _, tc := range []struct {
		c     Config
		valid bool
	}{
		{Config{}, true}, {Config{":8080", "site:8000"}, true}, {Config{"[::1]:0", "[::1]:8000"}, true},
		{Config{":8080", ""}, false}, {Config{"localhost:8000", "site:8000"}, false},
		{Config{":65536", "site:8000"}, false}, {Config{":8080", "http://site:8000"}, false},
		{Config{":8080", "site"}, false}, {Config{":8080", "user:secret@site:8000"}, false},
	} {
		if err := tc.c.Validate(); (err == nil) != tc.valid {
			t.Fatalf("%+v: %v", tc.c, err)
		}
		var services []lifecycle.LifeCycle
		h, err := New(tc.c, logger(), &services)
		if tc.valid {
			if err != nil || h == nil || (len(services) == 0) != (tc.c.ListenOn == "") {
				t.Fatalf("unexpected constructor result for %+v: %v, %v, %v", tc.c, h, services, err)
			}
		} else if err == nil || h != nil || len(services) != 0 {
			t.Fatalf("invalid config returned component: %+v", tc.c)
		}
	}
	var services []lifecycle.LifeCycle
	if h, err := New(Config{}, nil, &services); err == nil || h != nil || len(services) != 0 {
		t.Fatal("nil logger accepted")
	}
}

func TestForwardAndForms(t *testing.T) {
	t.Setenv("HTTP_PROXY", "http://127.0.0.1:1")
	seen := make(chan string, 4)
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		seen <- r.Host + "|" + r.RequestURI + "|" + r.Method + "|" + string(b) + "|" + r.Trailer.Get("X-End")
		if r.Header.Get("X-Hop") != "" || r.Header.Get("Proxy-Authorization") != "" {
			t.Error("hop header leaked")
		}
		w.Header().Set("Trailer", "X-End")
		w.Header().Set("Location", "http://different.invalid/")
		w.Header().Set("Connection", "X-Hop")
		w.Header().Set("X-Hop", "remove")
		w.WriteHeader(302)
		io.WriteString(w, "marker")
		w.Header().Set("X-End", "response-end")
	}))
	defer backend.Close()
	var services []lifecycle.LifeCycle
	s, err := New(Config{LocalSite: backend.Listener.Addr().String()}, logger(), &services)
	if err != nil {
		t.Fatal(err)
	}
	front := httptest.NewServer(s)
	if s == nil || len(services) != 0 {
		t.Fatal("disabled listener must retain handler without lifecycle")
	}
	defer front.Close()
	defer s.(*Server).transport.CloseIdleConnections()
	tr := &http.Transport{}
	defer tr.CloseIdleConnections()
	client := &http.Client{Transport: tr, Timeout: time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	req, _ := http.NewRequest("POST", front.URL+"/a%2Fb?x=a;b", strings.NewReader("payload"))
	req.Host = "untrusted.example:99"
	req.ContentLength = -1
	req.Trailer = http.Header{"X-End": []string{"request-end"}}
	req.Header.Set("Connection", "X-Hop")
	req.Header.Set("X-Hop", "remove")
	req.Header.Set("Proxy-Authorization", "secret")
	resp, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	b, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil || resp.StatusCode != 302 || string(b) != "marker" || resp.Header.Get("X-Hop") != "" || resp.Trailer.Get("X-End") != "response-end" {
		t.Fatalf("response %d %s %v", resp.StatusCode, b, err)
	}
	if got := <-seen; got != "untrusted.example:99|/a%2Fb?x=a;b|POST|payload|request-end" {
		t.Fatal(got)
	}
	for _, tc := range []struct {
		method, target string
		status         int
	}{{"CONNECT", "example.com:443", 405}, {"GET", "http://example.com/", 400}, {"OPTIONS", "*", 400}} {
		r := httptest.NewRequest(tc.method, tc.target, nil)
		w := httptest.NewRecorder()
		s.ServeHTTP(w, r)
		if w.Code != tc.status {
			t.Fatalf("%s: %d", tc.target, w.Code)
		}
	}
	select {
	case got := <-seen:
		t.Fatalf("invalid request forwarded: %s", got)
	default:
	}
	backend.Close()
	r := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	s.ServeHTTP(w, r)
	if w.Code != 502 {
		t.Fatal(w.Code)
	}
}

func TestCancellationAndTimeout(t *testing.T) {
	entered := make(chan struct{}, 2)
	exited := make(chan struct{}, 2)
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		entered <- struct{}{}
		<-r.Context().Done()
		exited <- struct{}{}
	}))
	defer backend.Close()
	var services []lifecycle.LifeCycle
	handler, err := New(Config{ListenOn: "127.0.0.1:0", LocalSite: backend.Listener.Addr().String()}, logger(), &services)
	if err != nil {
		t.Fatal(err)
	}
	s := handler.(*Server)
	s.transport.ResponseHeaderTimeout = 20 * time.Millisecond
	front := httptest.NewServer(s)
	resp, err := front.Client().Get(front.URL)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != 504 {
		t.Fatal(resp.StatusCode)
	}
	<-entered
	select {
	case <-exited:
	case <-time.After(time.Second):
		t.Fatal("timeout did not cancel backend")
	}
	front.Close()
	s.transport.CloseIdleConnections()
	handler, err = New(Config{ListenOn: "127.0.0.1:0", LocalSite: backend.Listener.Addr().String()}, logger(), &services)
	if err != nil {
		t.Fatal(err)
	}
	s = handler.(*Server)
	if len(services) != 2 || services[1] != s || services[0] == services[1] {
		t.Fatal("handler and lifecycle must share the same component")
	}
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() { done <- s.serve(ctx, l) }()
	requestDone := make(chan struct{})
	go func() {
		defer close(requestDone)
		tr := &http.Transport{}
		defer tr.CloseIdleConnections()
		c := &http.Client{Transport: tr, Timeout: time.Second}
		resp, _ := c.Get("http://" + l.Addr().String())
		if resp != nil {
			resp.Body.Close()
		}
	}()
	<-entered
	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("shutdown blocked")
	}
	select {
	case <-exited:
	case <-time.After(time.Second):
		t.Fatal("backend not canceled")
	}
	<-requestDone
}
