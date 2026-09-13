// Пакет routing разбирает цель HTTP-прокси и выбирает маршрут по имени узла.
package routing

import (
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"

	"overnode/gate/internal/network"
)

// Route определяет сеть, в которую направляется запрос.
type Route string

const (
	Internet  Route = "internet"
	Local     Route = "local"
	Ygg       Route = "ygg"
	Overspace Route = "overspace"
)

// Target содержит исходную и нормализованную цели прокси-запроса.
type Target struct {
	Route     Route
	URL       *url.URL
	Authority string
	Hostname  string
	Port      string
}

// Parse разбирает request-target, нормализует имя и выбирает маршрут.
func Parse(request *http.Request) (Target, error) {
	var target Target
	if request.Method == http.MethodConnect {
		target.Authority = request.RequestURI
	} else {
		url, err := url.ParseRequestURI(request.RequestURI)
		if err != nil || url.Scheme != "http" || url.Host == "" || url.User != nil || url.Opaque != "" || strings.Contains(request.RequestURI, "#") {
			return target, fmt.Errorf("expected an absolute HTTP URL")
		}
		target.URL, target.Authority = url, url.Host
	}
	host, port, err := network.ParseAuthority(target.Authority, request.Method == http.MethodConnect)
	if err != nil {
		return target, err
	}
	host = strings.TrimSuffix(strings.ToLower(host), ".")
	if net.ParseIP(host) == nil {
		// Hex-ключ занимает 64 символа: лимит DNS-метки 63 здесь неприменим.
		for _, label := range strings.Split(host, ".") {
			if label == "" || label[0] == '-' || label[len(label)-1] == '-' {
				return target, fmt.Errorf("invalid target hostname")
			}
			for _, character := range label {
				if !(character >= 'a' && character <= 'z' || character >= '0' && character <= '9' || character == '-') {
					return target, fmt.Errorf("invalid target hostname")
				}
			}
		}
	}
	target.Hostname, target.Port, target.Route = host, port, Internet
	switch {
	case host == "local.overspace":
		target.Route = Local
	case strings.HasSuffix(host, ".overspace"):
		target.Route = Overspace
	case strings.HasSuffix(host, ".ygg"):
		target.Route = Ygg
	}
	return target, nil
}
