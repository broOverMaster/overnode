package app

import (
	"context"
	"io"
	"log/slog"
	"net"
	"overnode/gate/internal/httpin"
	"overnode/gate/internal/proxyf"
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
