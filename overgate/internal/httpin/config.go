// Пакет httpin принимает HTTP из overlay и пересылает фиксированному локальному сайту.
package httpin

import (
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"

	"overnode/common/pkg/config/schema"
)

// Config задаёт входящий слушатель и единственный адрес сайта.
type Config struct {
	ListenOn  string `mapstructure:"listen_on"`
	LocalSite string `mapstructure:"local_site"`
}

// Schema описывает настройки; пустой listen_on отключает компонент.
func Schema() []schema.Field {
	return []schema.Field{
		schema.String("httpin.listen_on", "", "optional incoming HTTP IP:port; empty disables listener"),
		schema.String("httpin.local_site", "", "incoming HTTP destination host:port"),
	}
}

// Validate проверяет адреса до открытия слушателей.
func (c Config) Validate() error {
	if c.ListenOn != "" {
		host, port, err := net.SplitHostPort(c.ListenOn)
		if err != nil || (host != "" && net.ParseIP(host) == nil) {
			return fmt.Errorf("httpin.listen_on must be an IP:port or :port pair")
		}
		if _, err := strconv.ParseUint(port, 10, 16); err != nil {
			return fmt.Errorf("httpin.listen_on has an invalid port")
		}
		if c.LocalSite == "" {
			return fmt.Errorf("httpin.local_site is required when httpin is enabled")
		}
	}
	if c.LocalSite != "" {
		u, err := url.Parse("http://" + c.LocalSite)
		if err != nil || u.Host != c.LocalSite || u.User != nil || strings.ContainsAny(c.LocalSite, " /?#@\\\t\r\n") {
			return fmt.Errorf("httpin.local_site must be host:port")
		}
		host, port, err := net.SplitHostPort(c.LocalSite)
		if err != nil || host == "" {
			return fmt.Errorf("httpin.local_site must be host:port")
		}
		if strings.Contains(host, ":") && net.ParseIP(host) == nil {
			return fmt.Errorf("httpin.local_site has an invalid IP address")
		}
		n, err := strconv.ParseUint(port, 10, 16)
		if err != nil || n == 0 {
			return fmt.Errorf("httpin.local_site has an invalid port")
		}
	}
	return nil
}
