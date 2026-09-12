package httpd

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
)

// Server обслуживает один каталог статических файлов.
type Server struct {
	configuration Config
	logger        *slog.Logger
}

// New проверяет зависимости и создаёт HTTP-сервер.
func New(configuration Config, logger *slog.Logger) (*Server, error) {
	if logger == nil {
		return nil, fmt.Errorf("httpd logger must not be nil")
	}
	listenOn, err := canonicalListenOn(configuration.ListenOn)
	if err != nil {
		return nil, err
	}
	configuration.ListenOn = listenOn
	if err := configuration.Validate(); err != nil {
		return nil, err
	}
	return &Server{configuration: configuration, logger: logger.With("component", "httpd")}, nil
}

// Run открывает слушатель и обслуживает сайт до отмены контекста.
func (server *Server) Run(ctx context.Context) (runErr error) {
	accessLogger, closeAccessLog, err := newAccessLogger(server.configuration.AccessLogPath)
	if err != nil {
		return err
	}
	defer func() {
		if err := closeAccessLog(); err != nil {
			runErr = errors.Join(runErr, fmt.Errorf("close HTTP access log: %w", err))
		}
	}()

	listener, err := net.Listen("tcp", server.configuration.ListenOn)
	if err != nil {
		return fmt.Errorf("listen on %q: %w", server.configuration.ListenOn, err)
	}
	return server.serve(ctx, listener, accessLogger)
}

func (server *Server) serve(ctx context.Context, listener net.Listener, accessLogger *slog.Logger) error {
	handler := accessLogHandler(http.FileServer(http.Dir(server.configuration.SitePath)), accessLogger)
	httpServer := &http.Server{Handler: handler}
	server.logger.Info("HTTP server started", "listen_on", listener.Addr().String(), "site_path", server.configuration.SitePath)

	result := make(chan error, 1)
	go func() { result <- httpServer.Serve(listener) }()

	select {
	case err := <-result:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return fmt.Errorf("serve HTTP: %w", err)
	case <-ctx.Done():
		if err := httpServer.Close(); err != nil {
			return fmt.Errorf("close HTTP server: %w", err)
		}
		if err := <-result; !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("serve HTTP: %w", err)
		}
		server.logger.Info("HTTP server stopped")
		return nil
	}
}
