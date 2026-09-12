package app

import (
	"io"
	"overnode/common/pkg/config"
	"testing"
)

func TestConfigDefaultsAndPrecedence(t *testing.T) {
	t.Setenv("PROXYF__LISTEN_ON", "127.0.0.1:9000")
	var c Config
	load := func(args []string) {
		t.Helper()
		if err := config.Load(args, io.Discard, &c, config.Options{Command: "overgate", Fields: Schema()}); err != nil {
			t.Fatal(err)
		}
	}
	load(nil)
	if c.ProxyF.ListenOn != "127.0.0.1:9000" {
		t.Fatal(c.ProxyF.ListenOn)
	}
	load([]string{"--proxyf.listen_on=127.0.0.1:9001"})
	if c.ProxyF.ListenOn != "127.0.0.1:9001" {
		t.Fatal(c.ProxyF.ListenOn)
	}
	t.Setenv("PROXYF__LISTEN_ON", "")
	load(nil)
	if c.ProxyF.ListenOn != ":2080" {
		t.Fatal(c.ProxyF.ListenOn)
	}
}
