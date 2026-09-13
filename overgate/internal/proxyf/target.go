package proxyf

import (
	"net"
	"net/http"

	"overnode/common/pkg/network/overlay"
	"overnode/gate/internal/proxyf/routing"
)

// ServeHTTP классифицирует запрос и выбирает обработчик маршрута.
func (server *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	server.mu.Lock()
	if server.stopping {
		server.mu.Unlock()
		http.Error(w, "proxy stopping", 503)
		return
	}
	server.requests.Add(1)
	server.mu.Unlock()
	defer server.requests.Done()
	target, err := routing.Parse(r)
	if err != nil {
		http.Error(w, "invalid proxy request", http.StatusBadRequest)
		return
	}
	server.logger.Debug("proxy request", "route", string(target.Route), "method", r.Method, "target", target.Hostname, "port", target.Port)
	if r.Method == http.MethodConnect && target.Route != routing.Internet {
		w.Header().Set("Allow", "GET, HEAD, POST, PUT, DELETE, OPTIONS, PATCH, TRACE")
		http.Error(w, "CONNECT is only supported for internet", http.StatusMethodNotAllowed)
		return
	}
	if r.Method == http.MethodConnect {
		server.tunneler.Connect(server.lifetime, w, net.JoinHostPort(target.Hostname, target.Port))
		return
	}
	switch target.Route {
	case routing.Local:
		if server.configuration.LocalSite == "" {
			http.Error(w, "local route is not configured", 503)
			return
		}
		server.forwarder.Forward(w, r, target)
	case routing.Internet:
		server.forwarder.Forward(w, r, target)
	case routing.Ygg, routing.Overspace:
		ip, err := overlay.NormalizeAddress(target.Hostname)
		if err != nil {
			http.Error(w, "invalid public key", 400)
			return
		}
		if server.configuration.Yggstack == "" {
			http.Error(w, "Yggstack is not configured", 503)
			return
		}
		target.Hostname = ip.String()
		server.forwarder.Forward(w, r, target)
	}
}
