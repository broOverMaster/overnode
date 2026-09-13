package httpin

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"overnode/gate/internal/lifecycle"
	"strings"
	"sync"
	"time"
)

// Server обслуживает входящие HTTP-запросы без выбора маршрута клиентом.
type Server struct {
	config    Config
	logger    *slog.Logger
	transport *http.Transport
	proxy     *httputil.ReverseProxy
	mu        sync.Mutex
	stopping  bool
	requests  sync.WaitGroup
}

// New проверяет конфигурацию и создаёт транспорт без открытия слушателя.
// Обработчик доступен всегда; при пустом ListenOn сервис в срез не добавляется.
func New(c Config, logger *slog.Logger, lifeCycles *[]lifecycle.LifeCycle) (http.Handler, error) {
	if lifeCycles == nil {
		return nil, fmt.Errorf("httpin lifecycle slice must not be nil")
	}
	if logger == nil {
		return nil, fmt.Errorf("httpin logger must not be nil")
	}
	if err := c.Validate(); err != nil {
		return nil, err
	}
	s := &Server{config: c, logger: logger.With("component", "httpin")}
	s.transport = &http.Transport{DialContext: (&net.Dialer{Timeout: 10 * time.Second, KeepAlive: 30 * time.Second}).DialContext, ResponseHeaderTimeout: 30 * time.Second, ExpectContinueTimeout: time.Second, IdleConnTimeout: 90 * time.Second, MaxIdleConns: 100, DisableCompression: true}
	s.proxy = &httputil.ReverseProxy{
		Transport: s.transport, FlushInterval: -1,
		Rewrite: func(p *httputil.ProxyRequest) {
			// Адрес соединения фиксирован; логическое имя сайта и URL сохраняются.
			p.Out.URL.Scheme = "http"
			p.Out.URL.Host = c.LocalSite
			p.Out.URL.RawQuery = p.In.URL.RawQuery
			p.Out.Host = p.In.Host
			p.Out.Trailer = p.In.Trailer
			p.Out.Header.Del("Upgrade")
			p.Out.Header.Del("Connection")
			p.Out.Header.Del("Te")
		},
		ModifyResponse: func(r *http.Response) error {
			if r.StatusCode == 101 {
				return fmt.Errorf("protocol upgrade is not supported")
			}
			return nil
		},
		ErrorHandler: func(w http.ResponseWriter, r *http.Request, err error) {
			code := 502
			var ne net.Error
			if errors.Is(err, context.DeadlineExceeded) || (errors.As(err, &ne) && ne.Timeout()) {
				code = 504
			}
			s.logger.Warn("incoming HTTP backend failed", "status", code)
			http.Error(w, http.StatusText(code), code)
		},
	}
	if c.ListenOn == "" {
		s.logger.Info("component skipped", "reason", "httpin.listen_on is empty")
		return s, nil
	}
	*lifeCycles = append(*lifeCycles, s)
	return s, nil
}

// ServeHTTP допускает только origin-form; Host не влияет на выбор backend.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	if s.stopping {
		s.mu.Unlock()
		http.Error(w, "httpin stopping", 503)
		return
	}
	s.requests.Add(1)
	s.mu.Unlock()
	defer s.requests.Done()
	if r.Method == http.MethodConnect {
		w.Header().Set("Allow", "GET, HEAD, POST, PUT, DELETE, OPTIONS, PATCH, TRACE")
		http.Error(w, "CONNECT is not supported", 405)
		return
	}
	u, err := url.ParseRequestURI(r.RequestURI)
	if err != nil || !strings.HasPrefix(r.RequestURI, "/") || u.IsAbs() || u.Host != "" || strings.Contains(r.RequestURI, "#") {
		http.Error(w, "expected origin-form HTTP request", 400)
		return
	}
	if s.config.LocalSite == "" {
		http.Error(w, "incoming HTTP is not configured", 503)
		return
	}
	s.logger.Debug("incoming HTTP request", "method", r.Method, "host", r.Host)
	s.proxy.ServeHTTP(w, r)
}

// Run открывает слушатель и закрывает активные запросы при отмене контекста.
func (s *Server) Run(ctx context.Context) (err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("run httpin: %w", err)
		}
	}()
	if s.config.ListenOn == "" {
		return fmt.Errorf("httpin listener is disabled")
	}
	l, err := net.Listen("tcp", s.config.ListenOn)
	if err != nil {
		return fmt.Errorf("listen incoming HTTP: %w", err)
	}
	return s.serve(ctx, l)
}

func (s *Server) serve(ctx context.Context, l net.Listener) error {
	ctx, cancel := context.WithCancel(ctx)
	h := &http.Server{Handler: s, ReadHeaderTimeout: 10 * time.Second, IdleTimeout: 90 * time.Second, MaxHeaderBytes: 1 << 20, BaseContext: func(net.Listener) context.Context { return ctx }}
	defer func() {
		s.mu.Lock()
		s.stopping = true
		s.mu.Unlock()
		cancel()
		_ = h.Close()
		s.requests.Wait()
		s.transport.CloseIdleConnections()
	}()
	s.logger.Info("incoming HTTP started", "listen_on", l.Addr().String())
	done := make(chan error, 1)
	go func() { done <- h.Serve(l) }()
	var err error
	select {
	case err = <-done:
	case <-ctx.Done():
		_ = h.Close()
		err = <-done
	}
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("serve incoming HTTP: %w", err)
	}
	s.logger.Info("incoming HTTP stopped")
	return nil
}
