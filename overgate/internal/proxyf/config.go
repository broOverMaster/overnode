// Пакет proxyf реализует HTTP-прокси с маршрутизацией по целевому имени.
package proxyf

import (
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"

	"overnode/common/pkg/config/schema"
)

// Config задаёт слушатель и адреса назначения маршрутов.
type Config struct {
	ListenOn      string `mapstructure:"listen_on"`
	HTTPProxy     string `mapstructure:"http_proxy"`
	YggstackProxy string `mapstructure:"yggstack_proxy"`
	LocalSite     string `mapstructure:"local_site"`
}

// Schema возвращает параметры компонента.
func Schema() []schema.Field {
	return []schema.Field{
		schema.String("proxyf.listen_on", ":2080", "HTTP proxy listen address"),
		schema.String("proxyf.http_proxy", "", "optional upstream HTTP proxy URL"),
		schema.String("proxyf.yggstack_proxy", "", "optional Yggstack SOCKS5 proxy URL"),
		schema.String("proxyf.local_site", "", "optional local HTTP site URL"),
	}
}

func authority(value string, requirePort bool) (string, string, error) {
	u, err := url.Parse("http://" + value)
	if err != nil || u.Host != value || u.User != nil || u.Hostname() == "" || strings.ContainsAny(value, " /?#@\\\t\r\n") {
		return "", "", fmt.Errorf("invalid authority")
	}
	host := u.Hostname()
	if strings.Contains(host, ":") && !strings.HasPrefix(value, "[") {
		return "", "", fmt.Errorf("IPv6 address must be bracketed")
	}
	if strings.Contains(host, ":") && net.ParseIP(host) == nil {
		return "", "", fmt.Errorf("invalid IP address")
	}
	if strings.HasPrefix(value, "[") && net.ParseIP(host) == nil {
		return "", "", fmt.Errorf("invalid IP literal")
	}
	port := u.Port()
	if port == "" {
		if requirePort || strings.HasSuffix(value, ":") {
			return "", "", fmt.Errorf("missing port")
		}
		port = "80"
	}
	n, err := strconv.ParseUint(port, 10, 16)
	if err != nil || n == 0 {
		return "", "", fmt.Errorf("invalid port")
	}
	return host, port, nil
}

func endpoint(value, scheme string) (*url.URL, error) {
	if value == "" {
		return nil, nil
	}
	u, err := url.Parse(value)
	if err != nil || u.Scheme != scheme || u.Opaque != "" || u.User != nil || u.Path != "" || u.RawQuery != "" || u.ForceQuery || u.Fragment != "" || strings.Contains(value, "#") {
		return nil, fmt.Errorf("expected %s://host:port without credentials, path, query or fragment", scheme)
	}
	host, port, err := authority(u.Host, scheme == "socks5")
	if err != nil {
		return nil, err
	}
	u.Host = net.JoinHostPort(host, port)
	return u, nil
}

// Validate проверяет конфигурацию до открытия слушателя.
func (c Config) Validate() error {
	host, port, err := net.SplitHostPort(c.ListenOn)
	if err != nil || (host != "" && net.ParseIP(host) == nil) {
		return fmt.Errorf("proxyf.listen_on must be an IP:port or :port pair")
	}
	if _, err := strconv.ParseUint(port, 10, 16); err != nil {
		return fmt.Errorf("proxyf.listen_on has an invalid port")
	}
	for _, item := range []struct{ key, value, scheme string }{
		{"http_proxy", c.HTTPProxy, "http"}, {"yggstack_proxy", c.YggstackProxy, "socks5"}, {"local_site", c.LocalSite, "http"},
	} {
		if _, err := endpoint(item.value, item.scheme); err != nil {
			return fmt.Errorf("proxyf.%s: %w", item.key, err)
		}
	}
	return nil
}
