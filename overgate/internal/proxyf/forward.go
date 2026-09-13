package proxyf

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httputil"
)

func (server *Server) forward(w http.ResponseWriter, r *http.Request, t target) {
	destination := *t.url
	host := t.authority
	transport := server.internetTransport
	if t.route == routeLocal {
		local, _ := httpEndpointURL(server.configuration.LocalSite)
		destination.Host = local.Host
		host = "local.overspace"
		transport = server.localTransport
	} else {
		destination.Host = net.JoinHostPort(t.hostname, t.port)
		if t.route == routeYgg || t.route == routeOverspace {
			transport = server.overlayTransport
		}
	}
	proxy := &httputil.ReverseProxy{
		Transport:     transport,
		FlushInterval: -1,
		Rewrite: func(p *httputil.ProxyRequest) {
			// Прокси сохраняет исходные path/query и не добавляет Forwarded-заголовки.
			p.Out.URL = &destination
			p.Out.Host = host
			// Значения входящих трейлеров появляются только после чтения Body до EOF.
			p.Out.Trailer = p.In.Trailer
			p.Out.Header.Del("Upgrade")
			p.Out.Header.Del("Connection")
			p.Out.Header.Del("Te")
		},
		ModifyResponse: func(resp *http.Response) error {
			if resp.StatusCode == http.StatusSwitchingProtocols {
				return fmt.Errorf("protocol upgrade is not supported")
			}
			return nil
		},
		ErrorHandler: func(w http.ResponseWriter, r *http.Request, err error) {
			code := http.StatusBadGateway
			var ne net.Error
			if errors.Is(err, context.DeadlineExceeded) || (errors.As(err, &ne) && ne.Timeout()) {
				code = http.StatusGatewayTimeout
			}
			// Ошибка транспорта может содержать URL с query или credentials.
			server.logger.Warn("proxy upstream failed", "route", string(t.route), "status", code)
			http.Error(w, http.StatusText(code), code)
		},
	}
	proxy.ServeHTTP(w, r)
}
