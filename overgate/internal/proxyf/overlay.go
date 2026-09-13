package proxyf

import (
	"crypto/ed25519"
	"encoding/hex"
	"fmt"
	"net/netip"
	"strings"

	"github.com/yggdrasil-network/yggdrasil-go/src/address"
)

// overlayAddress преобразует hex-ключ из имени узла в IPv6 по правилам Yggdrasil.
func overlayAddress(host string) (string, error) {
	label, _, ok := strings.Cut(host, ".")
	if !ok || len(label) != ed25519.PublicKeySize*2 {
		return "", fmt.Errorf("expected a 64-digit hexadecimal public key")
	}
	key, err := hex.DecodeString(label)
	if err != nil || (host != label+".ygg" && host != label+".overspace") {
		return "", fmt.Errorf("invalid public key hostname")
	}
	ip := address.AddrForKey(ed25519.PublicKey(key))
	return netip.AddrFrom16([16]byte(*ip)).String(), nil
}
