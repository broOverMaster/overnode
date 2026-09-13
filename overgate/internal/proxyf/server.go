package proxyf

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"overnode/common/pkg/lifecycle"
	"overnode/common/pkg/network"
	"overnode/gate/internal/proxyf/forwarding"
	"overnode/gate/internal/proxyf/tunnel"
	"sync"
	"time"

	"golang.org/x/net/proxy"
)

// Server управляет слушателем HTTP-прокси.
type Server struct {
	configuration     Config
	logger            *slog.Logger
	localTransport    *http.Transport
	internetTransport *http.Transport
	overlayTransport  *http.Transport
	forwarder         *forwarding.Forwarder
	tunneler          *tunnel.Tunneler
	tunnels           *tunnel.Tracker
	mu                sync.Mutex
	stopping          bool
	requests          sync.WaitGroup
	lifetime          context.Context
}

// New проверяет зависимости без открытия слушателя.
// Возвращает обработчик и добавляет сервис в общий срез после успешной инициализации.
func New(configuration Config, logger *slog.Logger, lifeCycles *[]lifecycle.LifeCycle) (http.Handler, error) {
	if lifeCycles == nil {
		return nil, fmt.Errorf("proxyf lifecycle slice must not be nil")
	}
	if logger == nil {
		return nil, fmt.Errorf("proxyf logger must not be nil")
	}
	if err := configuration.Validate(); err != nil {
		return nil, err
	}
	upstream, _ := httpEndpointURL(configuration.HTTPProxy)
	internet := newTransport()
	if upstream != nil {
		internet.Proxy = http.ProxyURL(upstream)
	}
	overlay := newTransport()
	if configuration.Yggstack != "" {
		host, port, _ := network.ParseAuthority(configuration.Yggstack, true)
		dialer, err := proxy.SOCKS5("tcp", net.JoinHostPort(host, port), nil, &net.Dialer{Timeout: 10 * time.Second, KeepAlive: 30 * time.Second})
		if err != nil {
			return nil, fmt.Errorf("create SOCKS5 dialer: %w", err)
		}
		contextDialer, ok := dialer.(proxy.ContextDialer)
		if !ok {
			return nil, fmt.Errorf("SOCKS5 dialer must support context")
		}
		overlay.DialContext = func(ctx context.Context, network, address string) (net.Conn, error) {
			ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
			defer cancel()
			return contextDialer.DialContext(ctx, network, address)
		}
	}
	componentLogger := logger.With("component", "proxyf")
	localTransport := newTransport()
	tracker := tunnel.NewTracker()
	server := &Server{
		configuration:     configuration,
		logger:            componentLogger,
		localTransport:    localTransport,
		internetTransport: internet,
		overlayTransport:  overlay,
		forwarder:         forwarding.New(httpEndpointURLOrNil(configuration.LocalSite), localTransport, internet, overlay, componentLogger),
		tunneler:          tunnel.New(upstream, componentLogger, tracker),
		tunnels:           tracker,
		lifetime:          context.Background(),
	}
	*lifeCycles = append(*lifeCycles, server)
	return server, nil
}

func httpEndpointURLOrNil(value string) *url.URL {
	endpoint, _ := httpEndpointURL(value)
	return endpoint
}

func newTransport() *http.Transport {
	return &http.Transport{
		Proxy:                 nil,
		DialContext:           (&net.Dialer{Timeout: 10 * time.Second, KeepAlive: 30 * time.Second}).DialContext,
		ResponseHeaderTimeout: 30 * time.Second,
		ExpectContinueTimeout: time.Second,
		IdleConnTimeout:       90 * time.Second,
		MaxIdleConns:          100,
		DisableCompression:    true,
	}
}

// Run открывает слушатель и блокируется до отмены контекста.
func (server *Server) Run(ctx context.Context) (err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("run proxyf: %w", err)
		}
	}()
	listener, err := net.Listen("tcp", server.configuration.ListenOn)
	if err != nil {
		return fmt.Errorf("listen on %q: %w", server.configuration.ListenOn, err)
	}
	return server.serve(ctx, listener)
}

func (server *Server) serve(ctx context.Context, listener net.Listener) error {
	ctx, cancel := context.WithCancel(ctx)
	server.lifetime = ctx
	httpServer := &http.Server{
		Handler:           server,
		ReadHeaderTimeout: 10 * time.Second,
		IdleTimeout:       90 * time.Second,
		MaxHeaderBytes:    1 << 20,
		BaseContext:       func(net.Listener) context.Context { return ctx },
	}
	defer func() {
		server.mu.Lock()
		server.stopping = true
		server.mu.Unlock()
		server.tunnels.Close()
		cancel()
		_ = httpServer.Close()
		server.requests.Wait()
		server.localTransport.CloseIdleConnections()
		server.internetTransport.CloseIdleConnections()
		server.overlayTransport.CloseIdleConnections()
	}()
	server.logger.Info("HTTP proxy started", "listen_on", listener.Addr().String())
	result := make(chan error, 1)
	go func() { result <- httpServer.Serve(listener) }()
	select {
	case err := <-result:
		if !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("serve HTTP proxy: %w", err)
		}
	case <-ctx.Done():
		if err := httpServer.Close(); err != nil {
			return fmt.Errorf("close HTTP proxy: %w", err)
		}
		if err := <-result; !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("serve HTTP proxy: %w", err)
		}
	}
	server.logger.Info("HTTP proxy stopped")
	return nil
}
