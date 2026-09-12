package proxyf

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"time"
)

// Server управляет слушателем HTTP-прокси.
type Server struct {
	configuration Config
	logger        *slog.Logger
}

// New проверяет зависимости без открытия слушателя.
func New(configuration Config, logger *slog.Logger) (*Server, error) {
	if logger == nil {
		return nil, fmt.Errorf("proxyf logger must not be nil")
	}
	if err := configuration.Validate(); err != nil {
		return nil, err
	}
	return &Server{configuration: configuration, logger: logger.With("component", "proxyf")}, nil
}

// Run открывает слушатель и блокируется до отмены контекста.
func (server *Server) Run(ctx context.Context) error {
	listener, err := net.Listen("tcp", server.configuration.ListenOn)
	if err != nil {
		return fmt.Errorf("listen on %q: %w", server.configuration.ListenOn, err)
	}
	return server.serve(ctx, listener)
}

func (server *Server) serve(ctx context.Context, listener net.Listener) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	httpServer := &http.Server{
		Handler:           http.HandlerFunc(pendingHandler),
		ReadHeaderTimeout: 10 * time.Second,
		IdleTimeout:       90 * time.Second,
		MaxHeaderBytes:    1 << 20,
		BaseContext:       func(net.Listener) context.Context { return ctx },
	}
	defer httpServer.Close()
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

// pendingHandler обозначает отсутствие маршрутизации до следующего этапа.
func pendingHandler(w http.ResponseWriter, _ *http.Request) {
	http.Error(w, "proxy routing is not implemented", http.StatusNotImplemented)
}
