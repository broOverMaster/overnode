// Пакет overlay разрешает имена overlay-узлов и преобразует их адреса Yggdrasil.
package overlay

import (
	"bufio"
	"crypto/ed25519"
	"encoding/hex"
	"errors"
	"fmt"
	"net/netip"
	"os"
	"strings"

	keyed25519 "overnode/common/pkg/crypto/ed25519"

	"github.com/yggdrasil-network/yggdrasil-go/src/address"
)

// ErrNameNotFound сообщает, что overlay-имя отсутствует в таблице имён.
var ErrNameNotFound = errors.New("overlay name not found")

// Resolver разрешает полное имя overlay-узла в 32-байтовый публичный ключ.
type Resolver interface {
	ResolveName(name string) ([]byte, error)
}

type resolver struct {
	localPublicKey ed25519.PublicKey
	byName         map[string]ed25519.PublicKey
}

// NewResolver создаёт резолвер с публичным ключом локальной ноды.
// Если hostsPath пуст, таблица имён загружается только для встроенного имени local.
func NewResolver(localPublicKey []byte, hostsPath string) (Resolver, error) {
	if len(localPublicKey) != ed25519.PublicKeySize {
		return nil, fmt.Errorf("local public key must be exactly %d bytes", ed25519.PublicKeySize)
	}
	value := &resolver{
		localPublicKey: append(ed25519.PublicKey(nil), localPublicKey...),
		byName:         make(map[string]ed25519.PublicKey),
	}
	if hostsPath != "" {
		if err := value.loadHosts(hostsPath); err != nil {
			return nil, err
		}
	}
	return value, nil
}

// ResolveName разрешает полное имя вида Name.overspace.
func (value *resolver) ResolveName(name string) ([]byte, error) {
	name = strings.ToLower(name)
	base, suffix, ok := strings.Cut(name, ".")
	if !ok || suffix == "" || !strings.HasSuffix(name, ".overspace") {
		return nil, fmt.Errorf("invalid overspace name")
	}
	base = strings.TrimSuffix(name, ".overspace")
	if len(base) == ed25519.PublicKeySize*2 {
		if publicKey, err := keyed25519.PublicKeyFromHex(base); err == nil {
			return append([]byte(nil), publicKey...), nil
		}
	}
	if !validHostname(base) {
		return nil, fmt.Errorf("invalid overspace name")
	}
	if base == "local" {
		return append([]byte(nil), value.localPublicKey...), nil
	}
	publicKey, ok := value.byName[base]
	if !ok {
		return nil, ErrNameNotFound
	}
	return append([]byte(nil), publicKey...), nil
}

func (value *resolver) loadHosts(path string) error {
	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open overlay hosts file: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	lineNumber := 0
	for scanner.Scan() {
		lineNumber++
		line := strings.TrimSpace(strings.SplitN(scanner.Text(), "#", 2)[0])
		if line == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 2 {
			return fmt.Errorf("overlay hosts file line %d: expected key and name", lineNumber)
		}
		publicKey, err := keyed25519.PublicKeyFromHex(fields[0])
		if err != nil {
			return fmt.Errorf("overlay hosts file line %d: invalid public key: %w", lineNumber, err)
		}
		for _, host := range fields[1:] {
			host = strings.ToLower(host)
			if !validHostname(host) || strings.HasSuffix(host, ".overspace") {
				return fmt.Errorf("overlay hosts file line %d: invalid name %q", lineNumber, host)
			}
			if previous, exists := value.byName[host]; exists && !previous.Equal(publicKey) {
				return fmt.Errorf("overlay hosts file line %d: conflicting name %q", lineNumber, host)
			}
			value.byName[host] = append(ed25519.PublicKey(nil), publicKey...)
		}
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("read overlay hosts file: %w", err)
	}
	return nil
}

func validHostname(value string) bool {
	if value == "" || len(value) > 253 || strings.HasPrefix(value, ".") || strings.HasSuffix(value, ".") {
		return false
	}
	for _, label := range strings.Split(value, ".") {
		if label == "" || len(label) > 63 || label[0] == '-' || label[len(label)-1] == '-' {
			return false
		}
		for _, character := range label {
			if !(character >= 'a' && character <= 'z' || character >= '0' && character <= '9' || character == '-') {
				return false
			}
		}
	}
	return true
}

// PublicKey2Address проверяет размер публичного ключа Ed25519 и вычисляет IPv6 Yggdrasil.
func PublicKey2Address(publicKey []byte) (netip.Addr, error) {
	if len(publicKey) != ed25519.PublicKeySize {
		return netip.Addr{}, fmt.Errorf("expected a %d-byte Ed25519 public key", ed25519.PublicKeySize)
	}
	return netip.AddrFrom16([16]byte(*address.AddrForKey(ed25519.PublicKey(publicKey)))), nil
}

// NormalizeAddress преобразует hex-ключ из имени узла ygg или overspace в IPv6.
func NormalizeAddress(host string) (netip.Addr, error) {
	label, suffix, ok := strings.Cut(host, ".")
	if !ok || len(label) != ed25519.PublicKeySize*2 || (suffix != "ygg" && suffix != "overspace") {
		return netip.Addr{}, fmt.Errorf("invalid public key hostname")
	}
	key, err := hex.DecodeString(label)
	if err != nil {
		return netip.Addr{}, fmt.Errorf("invalid public key hostname: %w", err)
	}
	return PublicKey2Address(key)
}
