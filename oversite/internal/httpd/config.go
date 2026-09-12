// Пакет httpd обслуживает статические файлы по loopback HTTP-адресу.
package httpd

import (
	"fmt"
	"net"
	"net/netip"
	"os"
	"strconv"
	"strings"

	"overnode/common/pkg/config/schema"
)

const defaultListenOn = "127.0.0.1:8000"

// Config описывает адрес прослушивания, каталог статического сайта и access log.
type Config struct {
	ListenOn      string `mapstructure:"listen_on" json:"listen_on" yaml:"listen_on" toml:"listen_on"`
	SitePath      string `mapstructure:"site_path" json:"site_path" yaml:"site_path" toml:"site_path"`
	AccessLogPath string `mapstructure:"access_log_path" json:"access_log_path" yaml:"access_log_path" toml:"access_log_path"`
}

// Schema возвращает внешние параметры HTTP-компонента.
func Schema() []schema.Field {
	return []schema.Field{
		schema.String("httpd.listen_on", defaultListenOn, "HTTP loopback listen address"),
		schema.String("httpd.site_path", "", "required static site directory"),
		schema.String("httpd.access_log_path", "", "optional HTTP access log file path"),
	}
}

// Validate проверяет адрес loopback и существующий каталог со статикой.
func (configuration Config) Validate() error {
	if _, err := canonicalListenOn(configuration.ListenOn); err != nil {
		return err
	}
	if strings.TrimSpace(configuration.SitePath) == "" {
		return fmt.Errorf("httpd.site_path must not be empty")
	}
	info, err := os.Stat(configuration.SitePath)
	if err != nil {
		return fmt.Errorf("inspect httpd.site_path %q: %w", configuration.SitePath, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("httpd.site_path %q must be a directory", configuration.SitePath)
	}
	return nil
}

func canonicalListenOn(value string) (string, error) {
	host, port, err := net.SplitHostPort(value)
	if err != nil {
		return "", fmt.Errorf("httpd.listen_on %q must be a host:port pair", value)
	}
	address, err := netip.ParseAddr(host)
	if err != nil || !address.IsLoopback() {
		return "", fmt.Errorf("httpd.listen_on %q must use a loopback IP address", value)
	}
	number, err := strconv.ParseUint(port, 10, 16)
	if err != nil {
		return "", fmt.Errorf("httpd.listen_on %q must use a numeric port between 0 and 65535", value)
	}
	return net.JoinHostPort(address.String(), strconv.FormatUint(number, 10)), nil
}
