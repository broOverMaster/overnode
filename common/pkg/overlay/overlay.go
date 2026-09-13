// Пакет overlay предоставляет преобразование адресов overlay-сетей OverNode.
package overlay

import (
	"crypto/ed25519"
	"encoding/hex"
	"fmt"
	"net/netip"
	"strings"

	"github.com/yggdrasil-network/yggdrasil-go/src/address"
)

// NormalizeAddress преобразует hex-ключ из имени узла в IPv6 по правилам Yggdrasil.
// Имя должно быть нормализовано: суффикс в нижнем регистре, без завершающей точки.
func NormalizeAddress(host string) (netip.Addr, error) {
	label, _, ok := strings.Cut(host, ".")
	if !ok || len(label) != ed25519.PublicKeySize*2 {
		return netip.Addr{}, fmt.Errorf("expected a 64-digit hexadecimal public key")
	}
	key, err := hex.DecodeString(label)
	if err != nil || (host != label+".ygg" && host != label+".overspace") {
		return netip.Addr{}, fmt.Errorf("invalid public key hostname")
	}
	return PublicKey2Address(key)
}

// PublicKey2Address проверяет размер публичного ключа Ed25519 и вычисляет IPv6 Yggdrasil.
func PublicKey2Address(pub []byte) (netip.Addr, error) {
	if len(pub) != ed25519.PublicKeySize {
		return netip.Addr{}, fmt.Errorf("expected a %d-byte Ed25519 public key", ed25519.PublicKeySize)
	}
	return netip.AddrFrom16([16]byte(*address.AddrForKey(ed25519.PublicKey(pub)))), nil
}
