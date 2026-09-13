// Пакет tunnel обслуживает CONNECT-туннели HTTP-прокси.
package tunnel

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"sync"
	"time"
)

const setupTimeout = 10 * time.Second

// Tracker отслеживает туннельные соединения и закрывает их при остановке сервиса.
type Tracker struct {
	mu          sync.Mutex
	closed      bool
	connections map[net.Conn]struct{}
}

// NewTracker создаёт реестр активных туннелей.
func NewTracker() *Tracker {
	return &Tracker{connections: make(map[net.Conn]struct{})}
}

// Track добавляет соединения и возвращает функцию их удаления.
// Если реестр уже закрыт, возвращает false.
func (tracker *Tracker) Track(connections ...net.Conn) (func(), bool) {
	tracker.mu.Lock()
	defer tracker.mu.Unlock()
	if tracker.closed {
		return nil, false
	}
	for _, connection := range connections {
		tracker.connections[connection] = struct{}{}
	}
	return func() {
		tracker.mu.Lock()
		defer tracker.mu.Unlock()
		for _, connection := range connections {
			delete(tracker.connections, connection)
		}
	}, true
}

// Close прекращает все отслеживаемые туннели.
func (tracker *Tracker) Close() {
	tracker.mu.Lock()
	defer tracker.mu.Unlock()
	tracker.closed = true
	for connection := range tracker.connections {
		_ = connection.Close()
	}
}

// Tunneler открывает прямой или chained CONNECT-туннель.
type Tunneler struct {
	upstream *url.URL
	logger   *slog.Logger
	tracker  *Tracker
}

// New создаёт обработчик туннелей.
func New(upstream *url.URL, logger *slog.Logger, tracker *Tracker) *Tunneler {
	return &Tunneler{upstream: upstream, logger: logger, tracker: tracker}
}

// Connect устанавливает и обслуживает CONNECT-туннель до отмены контекста.
func (tunneler *Tunneler) Connect(lifetime context.Context, writer http.ResponseWriter, target string) {
	ctx, cancel := context.WithTimeout(lifetime, setupTimeout)
	defer cancel()
	destination := target
	if tunneler.upstream != nil {
		destination = tunneler.upstream.Host
	}
	remote, err := (&net.Dialer{}).DialContext(ctx, "tcp", destination)
	if err != nil {
		tunneler.connectError(writer, err)
		return
	}
	defer remote.Close()
	stop := context.AfterFunc(lifetime, func() { _ = remote.Close() })
	defer stop()
	reader := bufio.NewReader(remote)
	if tunneler.upstream != nil {
		deadline, _ := ctx.Deadline()
		_ = remote.SetDeadline(deadline)
		request := &http.Request{Method: http.MethodConnect, URL: &url.URL{Opaque: target}, Host: target, Header: make(http.Header)}
		if err := request.Write(remote); err != nil {
			tunneler.connectError(writer, err)
			return
		}
		// Ограничение заголовков не затрагивает байты туннеля, уже прочитанные буфером.
		limited := &io.LimitedReader{R: remote, N: 1 << 20}
		reader = bufio.NewReader(limited)
		response, err := http.ReadResponse(reader, request)
		if err != nil {
			tunneler.connectError(writer, err)
			return
		}
		if response.StatusCode < 200 || response.StatusCode >= 300 {
			tunneler.connectError(writer, fmt.Errorf("upstream CONNECT rejected"))
			return
		}
		buffered := reader.Buffered()
		prefix := make([]byte, buffered)
		_, _ = io.ReadFull(reader, prefix)
		reader = bufio.NewReader(io.MultiReader(bytes.NewReader(prefix), remote))
		_ = remote.SetDeadline(time.Time{})
	}
	hijacker, ok := writer.(http.Hijacker)
	if !ok {
		http.Error(writer, "HTTP/1.x required", http.StatusHTTPVersionNotSupported)
		return
	}
	client, buffer, err := hijacker.Hijack()
	if err != nil {
		return
	}
	defer client.Close()
	release, ok := tunneler.tracker.Track(client, remote)
	if !ok {
		return
	}
	defer release()
	_ = client.SetDeadline(time.Now().Add(setupTimeout))
	if _, err := buffer.WriteString("HTTP/1.1 200 Connection Established\r\n\r\n"); err != nil {
		return
	}
	if err := buffer.Flush(); err != nil {
		return
	}
	_ = client.SetDeadline(time.Time{})
	done := make(chan struct{})
	copyStream := func(destination net.Conn, source io.Reader) {
		if _, err := io.Copy(destination, source); err != nil {
			_ = client.Close()
			_ = remote.Close()
			return
		}
		if tcp, ok := destination.(*net.TCPConn); ok {
			_ = tcp.CloseWrite()
		}
	}
	go func() { defer close(done); copyStream(remote, buffer.Reader) }()
	copyStream(client, reader)
	<-done
}

func (tunneler *Tunneler) connectError(writer http.ResponseWriter, err error) {
	code := http.StatusBadGateway
	var networkError net.Error
	if errors.Is(err, context.DeadlineExceeded) || (errors.As(err, &networkError) && networkError.Timeout()) {
		code = http.StatusGatewayTimeout
	}
	tunneler.logger.Warn("CONNECT upstream failed", "status", code)
	http.Error(writer, http.StatusText(code), code)
}
