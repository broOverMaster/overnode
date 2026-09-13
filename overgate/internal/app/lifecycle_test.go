package app

import (
	"context"
	"io"
	"log/slog"
	"net"
	"net/http"
	"overnode/common/pkg/lifecycle"
	"overnode/gate/internal/httpin"
	"overnode/gate/internal/proxyf"
	"strings"
	"testing"
	"time"
)

func TestComponentFailureStopsSibling(t *testing.T) {
	for _, incomingFails := range []bool{false, true} {
		occupied, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			t.Fatal(err)
		}
		available, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			occupied.Close()
			t.Fatal(err)
		}
		free := available.Addr().String()
		available.Close()
		c := Config{ProxyF: proxyf.Config{ListenOn: free}, HTTPIn: httpin.Config{ListenOn: occupied.Addr().String(), LocalSite: "127.0.0.1:1"}}
		if !incomingFails {
			c.ProxyF.ListenOn, c.HTTPIn.ListenOn = c.HTTPIn.ListenOn, c.ProxyF.ListenOn
		}
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		done := make(chan error, 1)
		go func() { done <- Run(ctx, c, slog.New(slog.NewTextHandler(io.Discard, nil))) }()
		select {
		case err := <-done:
			if err == nil {
				t.Error("expected component failure")
			} else {
				name := "proxyf"
				if incomingFails {
					name = "httpin"
				}
				if !strings.Contains(err.Error(), "run "+name+":") {
					t.Errorf("missing component name: %v", err)
				}
			}
		case <-ctx.Done():
			t.Error("sibling did not stop")
		}
		cancel()
		occupied.Close()
		rebound, err := net.Listen("tcp", free)
		if err != nil {
			t.Fatal("sibling listener leaked", err)
		}
		rebound.Close()
	}
}

func TestLifeCycleRegistration(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	var services []lifecycle.LifeCycle
	proxy, err := proxyf.New(proxyf.Config{ListenOn: ":0"}, logger, &services)
	if err != nil || len(services) != 1 || services[0] != proxy.(lifecycle.LifeCycle) {
		t.Fatalf("proxyf registration: %v, %v", services, err)
	}
	first := services[0]
	incoming, err := httpin.New(httpin.Config{ListenOn: ":0", LocalSite: "127.0.0.1:1"}, logger, &services)
	if err != nil || len(services) != 2 || services[0] != first || services[1] != incoming.(lifecycle.LifeCycle) {
		t.Fatalf("httpin registration: %v, %v", services, err)
	}
	second := services[1]
	handler, err := httpin.New(httpin.Config{}, logger, &services)
	if err != nil || handler == nil || len(services) != 2 || services[0] != first || services[1] != second {
		t.Fatalf("disabled component changed registry: %v", err)
	}
	for _, construct := range []func(*[]lifecycle.LifeCycle) (http.Handler, error){
		func(list *[]lifecycle.LifeCycle) (http.Handler, error) {
			return proxyf.New(proxyf.Config{ListenOn: "invalid"}, logger, list)
		},
		func(list *[]lifecycle.LifeCycle) (http.Handler, error) {
			return httpin.New(httpin.Config{ListenOn: "invalid"}, logger, list)
		},
	} {
		h, err := construct(&services)
		if err == nil || h != nil || len(services) != 2 || services[0] != first || services[1] != second {
			t.Fatal("failed initialization changed registry")
		}
	}
	if _, err := proxyf.New(proxyf.Config{ListenOn: ":0"}, logger, nil); err == nil {
		t.Fatal("proxyf accepted nil registry pointer")
	}
	if _, err := httpin.New(httpin.Config{}, logger, nil); err == nil {
		t.Fatal("httpin accepted nil registry pointer")
	}
}

type lifecycleLog chan string

func (l lifecycleLog) Write(p []byte) (int, error) {
	l <- string(p)
	return len(p), nil
}

func TestRunCancellation(t *testing.T) {
	for _, name := range []string{"proxyf_only", "both"} {
		t.Run(name, func(t *testing.T) {
			c := Config{ProxyF: proxyf.Config{ListenOn: "127.0.0.1:0"}}
			if name == "both" {
				c.HTTPIn = httpin.Config{ListenOn: "127.0.0.1:0", LocalSite: "127.0.0.1:1"}
			}
			logs := make(lifecycleLog, 16)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			done := make(chan error, 1)
			go func() { done <- Run(ctx, c, slog.New(slog.NewTextHandler(logs, nil))) }()
			timer := time.NewTimer(3 * time.Second)
			defer timer.Stop()
			proxyStarted, incomingAccounted := false, false
			for !proxyStarted || !incomingAccounted {
				select {
				case line := <-logs:
					proxyStarted = proxyStarted || strings.Contains(line, `msg="HTTP proxy started"`)
					if name == "both" {
						incomingAccounted = incomingAccounted || strings.Contains(line, `msg="incoming HTTP started"`)
					} else {
						incomingAccounted = incomingAccounted || (strings.Contains(line, `msg="component skipped"`) && strings.Contains(line, "component=httpin"))
					}
				case err := <-done:
					t.Fatalf("Run exited before cancellation: %v", err)
				case <-timer.C:
					t.Fatal("components did not start or report skip")
				}
			}
			cancel()
			select {
			case err := <-done:
				if err != nil {
					t.Fatal(err)
				}
			case <-time.After(3 * time.Second):
				t.Fatal("shutdown blocked")
			}
		})
	}
}
