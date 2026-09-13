// Пакет network содержит сетевые утилиты, общие для компонентов overgate.
package network

import (
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"
)

// ParseAuthority проверяет host[:port] или host:port и возвращает host и port.
// Если порт не обязателен и отсутствует, возвращается HTTP-порт по умолчанию 80.
func ParseAuthority(value string, requirePort bool) (string, string, error) {
	url, err := url.Parse("http://" + value)
	if err != nil || url.Host != value || url.User != nil || url.Hostname() == "" || strings.ContainsAny(value, " /?#@\\\t\r\n") {
		return "", "", fmt.Errorf("invalid authority")
	}
	host := url.Hostname()
	if strings.Contains(host, ":") && !strings.HasPrefix(value, "[") {
		return "", "", fmt.Errorf("IPv6 address must be bracketed")
	}
	if strings.Contains(host, ":") && net.ParseIP(host) == nil {
		return "", "", fmt.Errorf("invalid IP address")
	}
	if strings.HasPrefix(value, "[") && net.ParseIP(host) == nil {
		return "", "", fmt.Errorf("invalid IP literal")
	}
	port := url.Port()
	if port == "" {
		if requirePort || strings.HasSuffix(value, ":") {
			return "", "", fmt.Errorf("missing port")
		}
		port = "80"
	}
	portNumber, err := strconv.ParseUint(port, 10, 16)
	if err != nil || portNumber == 0 {
		return "", "", fmt.Errorf("invalid port")
	}
	return host, port, nil
}
