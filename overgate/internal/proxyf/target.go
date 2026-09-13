package proxyf

import (
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
)

type route string

const (
	routeInternet  route = "internet"
	routeLocal     route = "local"
	routeYgg       route = "ygg"
	routeOverspace route = "overspace"
)

// target разделяет исходный HTTP-адрес и нормализованную цель маршрутизации.
type target struct {
	route     route
	url       *url.URL
	authority string
	hostname  string
	port      string
}

func parseTarget(r *http.Request) (target, error) {
	var t target
	if r.Method == http.MethodConnect {
		t.authority = r.RequestURI
	} else {
		u, err := url.ParseRequestURI(r.RequestURI)
		if err != nil || u.Scheme != "http" || u.Host == "" || u.User != nil || u.Opaque != "" || strings.Contains(r.RequestURI, "#") {
			return t, fmt.Errorf("expected an absolute HTTP URL")
		}
		t.url, t.authority = u, u.Host
	}
	host, port, err := authority(t.authority, r.Method == http.MethodConnect)
	if err != nil {
		return t, err
	}
	host = strings.TrimSuffix(strings.ToLower(host), ".")
	if net.ParseIP(host) == nil {
		// Hex-ключ занимает 64 символа: лимит DNS-метки 63 здесь неприменим.
		for _, label := range strings.Split(host, ".") {
			if label == "" || label[0] == '-' || label[len(label)-1] == '-' {
				return t, fmt.Errorf("invalid target hostname")
			}
			for _, c := range label {
				if !(c >= 'a' && c <= 'z' || c >= '0' && c <= '9' || c == '-') {
					return t, fmt.Errorf("invalid target hostname")
				}
			}
		}
	}
	t.hostname, t.port, t.route = host, port, routeInternet
	switch {
	case host == "local.overspace":
		t.route = routeLocal
	case strings.HasSuffix(host, ".overspace"):
		t.route = routeOverspace
	case strings.HasSuffix(host, ".ygg"):
		t.route = routeYgg
	}
	return t, nil
}

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
	t, err := parseTarget(r)
	if err != nil {
		http.Error(w, "invalid proxy request", http.StatusBadRequest)
		return
	}
	server.logger.Debug("proxy request", "route", string(t.route), "method", r.Method, "target", t.hostname, "port", t.port)
	if r.Method == http.MethodConnect && t.route != routeInternet {
		w.Header().Set("Allow", "GET, HEAD, POST, PUT, DELETE, OPTIONS, PATCH, TRACE")
		http.Error(w, "CONNECT is only supported for internet", http.StatusMethodNotAllowed)
		return
	}
	if r.Method == http.MethodConnect {
		server.connect(w, r, net.JoinHostPort(t.hostname, t.port))
		return
	}
	switch t.route {
	case routeLocal:
		if server.configuration.LocalSite == "" {
			http.Error(w, "local route is not configured", 503)
			return
		}
		server.forward(w, r, t)
	case routeInternet:
		server.forward(w, r, t)
	case routeYgg, routeOverspace:
		ip, err := overlayAddress(t.hostname)
		if err != nil {
			http.Error(w, "invalid public key", 400)
			return
		}
		if server.configuration.Yggstack == "" {
			http.Error(w, "Yggstack is not configured", 503)
			return
		}
		t.hostname = ip
		server.forward(w, r, t)
	}
}
