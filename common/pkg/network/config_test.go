package network_test

import (
	"crypto/x509"
	"encoding/hex"
	"encoding/pem"
	"os"
	"path/filepath"
	"testing"

	keyed25519 "overnode/common/pkg/crypto/ed25519"
	"overnode/common/pkg/network"
)

const testSeed = "9d61b19deffd5a60ba844af492ec2cc44449c5697b326919703bac031cae7f60"

func TestLocalKeySources(t *testing.T) {
	privateKey, err := keyed25519.PrivateKeyFromSeedHex(testSeed)
	if err != nil {
		t.Fatal(err)
	}
	der, err := x509.MarshalPKCS8PrivateKey(privateKey)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "local-key.pem")
	if err := os.WriteFile(path, pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der}), 0o600); err != nil {
		t.Fatal(err)
	}

	for _, configuration := range []network.Config{
		{LocalKey: testSeed},
		{LocalKeyPath: path},
		{LocalKey: testSeed, LocalKeyPath: filepath.Join(t.TempDir(), "missing.pem")},
	} {
		resolver, publicKey, err := configuration.NewOverlayResolver()
		if err != nil {
			t.Fatal(err)
		}
		resolved, err := resolver.ResolveName("local.overspace")
		if err != nil || hex.EncodeToString(publicKey) != hex.EncodeToString(resolved) {
			t.Fatalf("local resolution: %x, %v", resolved, err)
		}
	}
}

func TestMissingOrInvalidLocalKey(t *testing.T) {
	for _, configuration := range []network.Config{
		{},
		{LocalKey: "invalid"},
		{LocalKeyPath: filepath.Join(t.TempDir(), "missing.pem")},
	} {
		if _, _, err := configuration.NewOverlayResolver(); err == nil {
			t.Errorf("NewOverlayResolver accepted %+v", configuration)
		}
	}
}
