// Пакет proxyf реализует HTTP-прокси с маршрутизацией по целевому имени.
package proxyf

import (
	"fmt"
	"net"
	"net/url"
	"strconv"

	"overnode/common/pkg/config/schema"
	"overnode/common/pkg/network"
)

// Config задаёт слушатель и адреса назначения маршрутов.
type Config struct {
	ListenOn  string `mapstructure:"listen_on"`
	HTTPProxy string `mapstructure:"http_proxy"`
	Yggstack  string `mapstructure:"yggstack"`
	LocalSite string `mapstructure:"local_site"`
}

// Schema возвращает параметры компонента.
func Schema() []schema.Field {
	return []schema.Field{
		schema.String("proxyf.listen_on", ":2080", "HTTP proxy listen address"),
		schema.String("proxyf.http_proxy", "", "optional upstream HTTP proxy host:port"),
		schema.String("proxyf.yggstack", "", "optional Yggstack SOCKS5 host:port"),
		schema.String("proxyf.local_site", "", "optional local HTTP site host:port"),
	}
}

func httpEndpointURL(value string) (*url.URL, error) {
	if value == "" {
		return nil, nil
	}
	host, port, err := network.ParseAuthority(value, true)
	if err != nil {
		return nil, fmt.Errorf("expected host:port without scheme, credentials or path")
	}
	return &url.URL{Scheme: "http", Host: net.JoinHostPort(host, port)}, nil
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
	if _, err := httpEndpointURL(c.HTTPProxy); err != nil {
		return fmt.Errorf("proxyf.http_proxy: %w", err)
	}
	if _, err := httpEndpointURL(c.LocalSite); err != nil {
		return fmt.Errorf("proxyf.local_site: %w", err)
	}
	if c.Yggstack != "" {
		if _, _, err := network.ParseAuthority(c.Yggstack, true); err != nil {
			return fmt.Errorf("proxyf.yggstack: expected host:port without scheme, credentials or path")
		}
	}
	return nil
}
