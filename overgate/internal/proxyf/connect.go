package proxyf

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"time"
)

const tunnelSetupTimeout = 10 * time.Second

func (server *Server) connect(w http.ResponseWriter, r *http.Request, target string) {
	ctx, cancel := context.WithTimeout(server.lifetime, tunnelSetupTimeout)
	defer cancel()
	destination := target
	upstream, _ := httpEndpointURL(server.configuration.HTTPProxy)
	if upstream != nil {
		destination = upstream.Host
	}
	remote, err := (&net.Dialer{}).DialContext(ctx, "tcp", destination)
	if err != nil {
		server.connectError(w, err)
		return
	}
	defer remote.Close()
	stop := context.AfterFunc(server.lifetime, func() { _ = remote.Close() })
	defer stop()
	reader := bufio.NewReader(remote)
	if upstream != nil {
		deadline, _ := ctx.Deadline()
		_ = remote.SetDeadline(deadline)
		request := &http.Request{Method: http.MethodConnect, URL: &url.URL{Opaque: target}, Host: target, Header: make(http.Header)}
		if err := request.Write(remote); err != nil {
			server.connectError(w, err)
			return
		}
		// Ограничение заголовков не затрагивает байты туннеля, уже прочитанные буфером.
		limited := &io.LimitedReader{R: remote, N: 1 << 20}
		reader = bufio.NewReader(limited)
		response, err := http.ReadResponse(reader, request)
		if err != nil {
			server.connectError(w, err)
			return
		}
		if response.StatusCode < 200 || response.StatusCode >= 300 {
			server.connectError(w, fmt.Errorf("upstream CONNECT rejected"))
			return
		}
		buffered := reader.Buffered()
		prefix := make([]byte, buffered)
		_, _ = io.ReadFull(reader, prefix)
		reader = bufio.NewReader(io.MultiReader(bytes.NewReader(prefix), remote))
		_ = remote.SetDeadline(time.Time{})
	}
	hijacker, ok := w.(http.Hijacker)
	if !ok {
		http.Error(w, "HTTP/1.x required", 505)
		return
	}
	client, buffer, err := hijacker.Hijack()
	if err != nil {
		return
	}
	defer client.Close()
	server.mu.Lock()
	if server.stopping {
		server.mu.Unlock()
		return
	}
	server.tunnels[client] = struct{}{}
	server.tunnels[remote] = struct{}{}
	server.mu.Unlock()
	defer func() {
		server.mu.Lock()
		delete(server.tunnels, client)
		delete(server.tunnels, remote)
		server.mu.Unlock()
	}()
	_ = client.SetDeadline(time.Now().Add(tunnelSetupTimeout))
	if _, err := buffer.WriteString("HTTP/1.1 200 Connection Established\r\n\r\n"); err != nil {
		return
	}
	if err := buffer.Flush(); err != nil {
		return
	}
	_ = client.SetDeadline(time.Time{})
	done := make(chan struct{})
	copyStream := func(dst net.Conn, src io.Reader) {
		if _, err := io.Copy(dst, src); err != nil {
			_ = client.Close()
			_ = remote.Close()
			return
		}
		if tcp, ok := dst.(*net.TCPConn); ok {
			_ = tcp.CloseWrite()
		}
	}
	go func() { defer close(done); copyStream(remote, buffer.Reader) }()
	copyStream(client, reader)
	<-done
}

func (server *Server) connectError(w http.ResponseWriter, err error) {
	code := http.StatusBadGateway
	var ne net.Error
	if errors.Is(err, context.DeadlineExceeded) || (errors.As(err, &ne) && ne.Timeout()) {
		code = http.StatusGatewayTimeout
	}
	server.logger.Warn("CONNECT upstream failed", "status", code)
	http.Error(w, http.StatusText(code), code)
}
