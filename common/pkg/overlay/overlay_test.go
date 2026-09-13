package overlay_test

import (
	"crypto/ed25519"
	"encoding/hex"
	"overnode/common/pkg/overlay"
	"strings"
	"testing"
)

// Вектор взят из upstream address/address_test.go Yggdrasil v0.5.14.
const vectorKey = "bdbacfd82240de3dcd123924cbb55256fb8dab08aa98e305528ab84f419e6efb"
const vectorIP = "200:848a:604f:bb7e:4384:65db:8db6:6895"

func TestPublicKey2Address(t *testing.T) {
	key, err := hex.DecodeString(vectorKey)
	if err != nil {
		t.Fatal(err)
	}
	ip, err := overlay.PublicKey2Address(key)
	if err != nil || !ip.IsValid() {
		t.Fatalf("address: %v, %v", ip, err)
	}
	if got := ip.String(); got != vectorIP {
		t.Fatalf("address: %s, want %s", got, vectorIP)
	}
	for _, size := range []int{0, ed25519.PublicKeySize - 1, ed25519.PublicKeySize + 1, ed25519.PrivateKeySize} {
		if ip, err := overlay.PublicKey2Address(make([]byte, size)); err == nil || ip.IsValid() {
			t.Errorf("accepted key size %d", size)
		}
	}
	if ip, err := overlay.PublicKey2Address(nil); err == nil || ip.IsValid() {
		t.Error("accepted nil key")
	}
}

func TestOverlayAddress(t *testing.T) {
	for _, suffix := range []string{"ygg", "overspace"} {
		got, err := overlay.NormalizeAddress(vectorKey + "." + suffix)
		if err != nil || got.String() != vectorIP {
			t.Fatalf("%s %v", got, err)
		}
	}
	for _, host := range []string{"human.overspace", strings.Repeat("a", 63) + ".ygg", strings.Repeat("a", 65) + ".ygg", strings.Repeat("g", 64) + ".ygg", vectorKey + ".extra.ygg", vectorKey, ".ygg"} {
		if _, err := overlay.NormalizeAddress(host); err == nil {
			t.Errorf("accepted %s", host)
		}
	}
}
