package ed25519_test

import (
	standard "crypto/ed25519"
	"crypto/x509"
	"encoding/pem"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"overnode/common/pkg/crypto/ed25519"

	"golang.org/x/crypto/ssh"
)

const seedHex = "9d61b19deffd5a60ba844af492ec2cc44449c5697b326919703bac031cae7f60"

func TestSeedAndPublicKey(t *testing.T) {
	seed, err := ed25519.SeedFromHex(seedHex)
	if err != nil {
		t.Fatal(err)
	}
	if len(seed) != standard.SeedSize {
		t.Fatalf("seed length = %d, want %d", len(seed), standard.SeedSize)
	}

	privateKey, err := ed25519.PrivateKeyFromSeedHex(seedHex)
	if err != nil {
		t.Fatal(err)
	}
	publicKey, err := ed25519.PublicKeyFromSeedHex(seedHex)
	if err != nil {
		t.Fatal(err)
	}
	if !privateKey.Public().(standard.PublicKey).Equal(publicKey) {
		t.Fatal("public key does not match private key")
	}

	encoded, err := ed25519.PublicKeyToHex(publicKey)
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := ed25519.PublicKeyFromHex(strings.ToUpper(encoded))
	if err != nil {
		t.Fatal(err)
	}
	if !decoded.Equal(publicKey) {
		t.Fatal("decoded public key does not match original")
	}
}

func TestInvalidValues(t *testing.T) {
	for _, value := range []string{"", "00", strings.Repeat("0", 62), strings.Repeat("0", 66), strings.Repeat("g", 64)} {
		if _, err := ed25519.SeedFromHex(value); err == nil {
			t.Errorf("SeedFromHex(%q) succeeded", value)
		}
		if _, err := ed25519.PublicKeyFromHex(value); err == nil {
			t.Errorf("PublicKeyFromHex(%q) succeeded", value)
		}
	}
	for _, value := range [][]byte{nil, make([]byte, standard.PublicKeySize-1), make([]byte, standard.PublicKeySize+1)} {
		if _, err := ed25519.PublicKeyToHex(value); err == nil {
			t.Errorf("PublicKeyToHex(%d bytes) succeeded", len(value))
		}
	}
}

func TestPEMKey(t *testing.T) {
	privateKey, err := ed25519.PrivateKeyFromSeedHex(seedHex)
	if err != nil {
		t.Fatal(err)
	}
	der, err := x509.MarshalPKCS8PrivateKey(privateKey)
	if err != nil {
		t.Fatal(err)
	}
	value := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der})
	got, err := ed25519.PublicKeyFromPEM(value)
	if err != nil || !got.Equal(privateKey.Public().(standard.PublicKey)) {
		t.Fatalf("PKCS#8 key: %x, %v", got, err)
	}
	seed, err := ed25519.SeedFromPEM(pem.EncodeToMemory(&pem.Block{Type: "ED25519 SEED", Bytes: privateKey.Seed()}))
	if err != nil || string(seed) != string(privateKey.Seed()) {
		t.Fatalf("raw seed: %x, %v", seed, err)
	}
	openSSHBlock, err := ssh.MarshalPrivateKey(privateKey, "test")
	if err != nil {
		t.Fatal(err)
	}
	got, err = ed25519.PublicKeyFromPEM(pem.EncodeToMemory(openSSHBlock))
	if err != nil || !got.Equal(privateKey.Public().(standard.PublicKey)) {
		t.Fatalf("OpenSSH key: %x, %v", got, err)
	}
	path := filepath.Join(t.TempDir(), "local-key.pem")
	if err := os.WriteFile(path, value, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := ed25519.PublicKeyFromPEMFile(path); err != nil {
		t.Fatal(err)
	}
}

func TestInvalidPEM(t *testing.T) {
	for _, value := range [][]byte{
		[]byte("not pem"),
		pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: []byte("invalid")}),
		pem.EncodeToMemory(&pem.Block{Type: "OTHER", Bytes: make([]byte, standard.SeedSize)}),
		pem.EncodeToMemory(&pem.Block{Type: "ED25519 SEED", Bytes: make([]byte, standard.SeedSize-1)}),
	} {
		if _, err := ed25519.SeedFromPEM(value); err == nil {
			t.Errorf("SeedFromPEM accepted %q", value)
		}
	}
}
