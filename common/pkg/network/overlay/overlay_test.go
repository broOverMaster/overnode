package overlay_test

import (
	"crypto/ed25519"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"overnode/common/pkg/network/overlay"
)

const (
	vectorKey = "bdbacfd82240de3dcd123924cbb55256fb8dab08aa98e305528ab84f419e6efb"
	vectorIP  = "200:848a:604f:bb7e:4384:65db:8db6:6895"
	localKey  = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
)

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
}

func TestNormalizeAddress(t *testing.T) {
	for _, suffix := range []string{"ygg", "overspace"} {
		got, err := overlay.NormalizeAddress(vectorKey + "." + suffix)
		if err != nil || got.String() != vectorIP {
			t.Fatalf("%s: %v, %v", suffix, got, err)
		}
	}
	for _, host := range []string{"human.overspace", strings.Repeat("g", 64) + ".ygg", vectorKey + ".extra.ygg", vectorKey, ".ygg"} {
		if _, err := overlay.NormalizeAddress(host); err == nil {
			t.Errorf("accepted %s", host)
		}
	}
}

func TestResolveName(t *testing.T) {
	directory := t.TempDir()
	path := filepath.Join(directory, "overlay.hosts")
	content := vectorKey + " alice alice.home\n" + localKey + " bob\n"
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	localPublicKey := make([]byte, ed25519.PublicKeySize)
	for index := range localPublicKey {
		localPublicKey[index] = byte(index)
	}
	resolver, err := overlay.NewResolver(localPublicKey, path)
	if err != nil {
		t.Fatal(err)
	}
	for _, testCase := range []struct {
		name string
		key  string
	}{
		{"alice.overspace", vectorKey},
		{"ALICE.HOME.OVERSPACE", vectorKey},
		{vectorKey + ".overspace", vectorKey},
	} {
		got, err := resolver.ResolveName(testCase.name)
		if err != nil || hex.EncodeToString(got) != testCase.key {
			t.Errorf("ResolveName(%q) = %x, %v", testCase.name, got, err)
		}
	}
	got, err := resolver.ResolveName("local.overspace")
	if err != nil || !equalBytes(got, localPublicKey) {
		t.Fatalf("local key = %x, %v", got, err)
	}
	if _, err := resolver.ResolveName("missing.overspace"); !errors.Is(err, overlay.ErrNameNotFound) {
		t.Fatalf("missing name error = %v", err)
	}
	for _, name := range []string{"alice.ygg", "alice.overspace:80", "http://alice.overspace", "alice@home.overspace"} {
		if _, err := resolver.ResolveName(name); err == nil {
			t.Errorf("ResolveName(%q) succeeded", name)
		}
	}
}

func TestHostsErrors(t *testing.T) {
	localPublicKey := make([]byte, ed25519.PublicKeySize)
	for _, content := range []string{
		"invalid-key alice\n",
		vectorKey + " bob@blackhole\n",
		vectorKey + " alice\n" + localKey + " alice\n",
		"only-key\n",
	} {
		path := filepath.Join(t.TempDir(), "overlay.hosts")
		if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
		if _, err := overlay.NewResolver(localPublicKey, path); err == nil {
			t.Errorf("NewResolver accepted %q", content)
		}
	}
}

func equalBytes(left, right []byte) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}
