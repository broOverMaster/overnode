// Пакет forwarding пересылает HTTP-запросы выбранному маршруту proxyf.
package forwarding

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"

	"overnode/gate/internal/proxyf/routing"
)

// Forwarder пересылает HTTP-запросы через транспорт выбранного маршрута.
type Forwarder struct {
	localSite         *url.URL
	localTransport    http.RoundTripper
	internetTransport http.RoundTripper
	overlayTransport  http.RoundTripper
	logger            *slog.Logger
}

// New создаёт пересылку с готовыми транспортами маршрутов.
func New(localSite *url.URL, localTransport, internetTransport, overlayTransport http.RoundTripper, logger *slog.Logger) *Forwarder {
	return &Forwarder{localSite: localSite, localTransport: localTransport, internetTransport: internetTransport, overlayTransport: overlayTransport, logger: logger}
}

// Forward пересылает обычный HTTP-запрос.
func (forwarder *Forwarder) Forward(writer http.ResponseWriter, request *http.Request, target routing.Target) {
	destination := *target.URL
	host := target.Authority
	transport := forwarder.internetTransport
	if target.Route == routing.Local {
		destination.Host = forwarder.localSite.Host
		host = "local.overspace"
		transport = forwarder.localTransport
	} else {
		destination.Host = net.JoinHostPort(target.Hostname, target.Port)
		if target.Route == routing.Ygg || target.Route == routing.Overspace {
			transport = forwarder.overlayTransport
		}
	}
	proxy := &httputil.ReverseProxy{
		Transport:     transport,
		FlushInterval: -1,
		Rewrite: func(proxyRequest *httputil.ProxyRequest) {
			// Прокси сохраняет исходные path/query и не добавляет Forwarded-заголовки.
			proxyRequest.Out.URL = &destination
			proxyRequest.Out.Host = host
			// Значения входящих трейлеров появляются только после чтения Body до EOF.
			proxyRequest.Out.Trailer = proxyRequest.In.Trailer
			proxyRequest.Out.Header.Del("Upgrade")
			proxyRequest.Out.Header.Del("Connection")
			proxyRequest.Out.Header.Del("Te")
		},
		ModifyResponse: func(response *http.Response) error {
			if response.StatusCode == http.StatusSwitchingProtocols {
				return fmt.Errorf("protocol upgrade is not supported")
			}
			return nil
		},
		ErrorHandler: func(writer http.ResponseWriter, request *http.Request, err error) {
			code := http.StatusBadGateway
			var networkError net.Error
			if errors.Is(err, context.DeadlineExceeded) || (errors.As(err, &networkError) && networkError.Timeout()) {
				code = http.StatusGatewayTimeout
			}
			// Ошибка транспорта может содержать URL с query или credentials.
			forwarder.logger.Warn("proxy upstream failed", "route", string(target.Route), "status", code)
			http.Error(writer, http.StatusText(code), code)
		},
	}
	proxy.ServeHTTP(writer, request)
}
