package proxyf

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"
)

// Вектор взят из upstream address/address_test.go Yggdrasil v0.5.14.
const vectorKey = "bdbacfd82240de3dcd123924cbb55256fb8dab08aa98e305528ab84f419e6efb"
const vectorIP = "200:848a:604f:bb7e:4384:65db:8db6:6895"

func TestOverlayAddress(t *testing.T) {
	for _, suffix := range []string{"ygg", "overspace"} {
		got, err := overlayAddress(vectorKey + "." + suffix)
		if err != nil || got != vectorIP {
			t.Fatalf("%s %v", got, err)
		}
	}
	for _, host := range []string{"human.overspace", strings.Repeat("a", 63) + ".ygg", strings.Repeat("a", 65) + ".ygg", strings.Repeat("g", 64) + ".ygg", vectorKey + ".extra.ygg", vectorKey, ".ygg"} {
		if _, err := overlayAddress(host); err == nil {
			t.Errorf("accepted %s", host)
		}
	}
}

type socksObservation struct {
	ip                      string
	port                    uint16
	host, uri, method, body string
}

func fakeSOCKS(t *testing.T, mode string) (string, <-chan socksObservation) {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	seen := make(chan socksObservation, 16)
	var wg sync.WaitGroup
	closed := make(chan struct{})
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			c, err := l.Accept()
			if err != nil {
				return
			}
			wg.Add(1)
			go func() {
				defer wg.Done()
				defer c.Close()
				_ = c.SetDeadline(time.Now().Add(4 * time.Second))
				if mode == "hang" {
					seen <- socksObservation{}
					<-closed
					return
				}
				var hello [2]byte
				if _, err := io.ReadFull(c, hello[:]); err != nil {
					return
				}
				methods := make([]byte, int(hello[1]))
				if _, err := io.ReadFull(c, methods); err != nil {
					return
				}
				if hello[0] != 5 {
					t.Error("not SOCKS5")
					return
				}
				if mode == "auth" {
					c.Write([]byte{5, 255})
					return
				}
				c.Write([]byte{5, 0})
				var header [4]byte
				if _, err := io.ReadFull(c, header[:]); err != nil {
					return
				}
				if header != [4]byte{5, 1, 0, 4} {
					t.Errorf("expected IPv6 CONNECT, got %v", header)
					return
				}
				var destination [18]byte
				if _, err := io.ReadFull(c, destination[:]); err != nil {
					return
				}
				observation := socksObservation{ip: net.IP(destination[:16]).String(), port: binary.BigEndian.Uint16(destination[16:])}
				if mode == "reject" {
					seen <- observation
					c.Write([]byte{5, 4, 0, 1, 0, 0, 0, 0, 0, 0})
					return
				}
				c.Write([]byte{5, 0, 0, 1, 0, 0, 0, 0, 0, 0})
				r, err := http.ReadRequest(bufio.NewReader(c))
				if err != nil {
					return
				}
				b, err := io.ReadAll(r.Body)
				r.Body.Close()
				if err != nil {
					return
				}
				observation.host = r.Host
				observation.uri = r.RequestURI
				observation.method = r.Method
				observation.body = string(b)
				seen <- observation
				fmt.Fprint(c, "HTTP/1.1 200 OK\r\nContent-Length: 14\r\nConnection: close\r\n\r\noverlay-marker")
			}()
		}
	}()
	t.Cleanup(func() { l.Close(); close(closed); wg.Wait() })
	return l.Addr().String(), seen
}

func TestOverlaySOCKSForward(t *testing.T) {
	address, seen := fakeSOCKS(t, "ok")
	_, client, _, _ := startProxy(t, Config{Yggstack: address, HTTPProxy: "127.0.0.1:1"})
	for _, suffix := range []string{"ygg", "overspace"} {
		for _, port := range []string{"", ":8081"} {
			host := strings.ToUpper(vectorKey+"."+suffix) + "." + port
			r, _ := http.NewRequest("POST", "http://"+host+"/a%2Fb?q=1", strings.NewReader("payload"))
			resp, err := client.Do(r)
			if err != nil {
				t.Fatal(err)
			}
			b, err := io.ReadAll(resp.Body)
			resp.Body.Close()
			if err != nil || resp.StatusCode != 200 || string(b) != "overlay-marker" {
				t.Fatalf("response %d %q %v", resp.StatusCode, b, err)
			}
			got := <-seen
			wantPort := uint16(80)
			if port != "" {
				wantPort = 8081
			}
			if got.ip != vectorIP || got.port != wantPort || got.host != host || got.uri != "/a%2Fb?q=1" || got.method != "POST" || got.body != "payload" {
				t.Fatalf("SOCKS request: %+v", got)
			}
		}
	}
	for _, host := range []string{"human.overspace", "bad.ygg", vectorKey + ".extra.ygg"} {
		resp, err := client.Get("http://" + host + "/")
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
		if resp.StatusCode != 400 {
			t.Fatal(resp.StatusCode)
		}
	}
	select {
	case got := <-seen:
		t.Fatalf("unexpected outbound request: %+v", got)
	default:
	}
}

func TestOverlayUnavailable(t *testing.T) {
	for _, mode := range []string{"reject", "auth"} {
		t.Run(mode, func(t *testing.T) {
			address, _ := fakeSOCKS(t, mode)
			_, client, _, _ := startProxy(t, Config{Yggstack: address})
			resp, err := client.Get("http://" + vectorKey + ".ygg/")
			if err != nil {
				t.Fatal(err)
			}
			resp.Body.Close()
			if resp.StatusCode != 502 {
				t.Fatal(resp.StatusCode)
			}
		})
	}
	_, client, _, _ := startProxy(t, Config{})
	resp, err := client.Get("http://" + vectorKey + ".overspace/")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != 503 {
		t.Fatal(resp.StatusCode)
	}
	address, seen := fakeSOCKS(t, "hang")
	_, client, cancel, _ := startProxy(t, Config{Yggstack: address})
	done := make(chan struct{})
	go func() {
		defer close(done)
		resp, _ := client.Get("http://" + vectorKey + ".ygg/")
		if resp != nil {
			resp.Body.Close()
		}
	}()
	<-seen
	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("SOCKS handshake not canceled")
	}
}
