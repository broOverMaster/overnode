package crypto_test

import (
	standard "crypto/ed25519"
	"crypto/x509"
	"encoding/hex"
	"encoding/pem"
	"os"
	"path/filepath"
	"strings"
	"testing"

	crypto "overnode/common/pkg/crypto"

	"golang.org/x/crypto/ssh"
)

const seedHex = "9d61b19deffd5a60ba844af492ec2cc44449c5697b326919703bac031cae7f60"

func TestSeedAndPublicKey(t *testing.T) {
	privateKey, err := crypto.KeyFromHex(seedHex)
	if err != nil {
		t.Fatal(err)
	}
	publicKey, err := crypto.PubFromHex(seedHex)
	if err != nil {
		t.Fatal(err)
	}
	if !privateKey.Public().(standard.PublicKey).Equal(publicKey) {
		t.Fatal("public key does not match private key")
	}

	encoded, err := crypto.PublicKeyToHex(publicKey)
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := crypto.PublicKeyFromHex(strings.ToUpper(encoded))
	if err != nil {
		t.Fatal(err)
	}
	if !decoded.Equal(publicKey) {
		t.Fatal("decoded public key does not match original")
	}
}

func TestPrivateKeyFromHex(t *testing.T) {
	seedKey, err := crypto.KeyFromHex(seedHex)
	if err != nil {
		t.Fatal(err)
	}

	fullKey, err := crypto.KeyFromHex(hex.EncodeToString(seedKey))
	if err != nil {
		t.Fatal(err)
	}
	if !fullKey.Equal(seedKey) {
		t.Fatal("full private key does not match seed-derived key")
	}
}

func TestInvalidValues(t *testing.T) {
	for _, value := range []string{"", "00", strings.Repeat("0", 62), strings.Repeat("0", 66), strings.Repeat("g", 64), strings.Repeat("g", 128)} {
		if _, err := crypto.KeyFromHex(value); err == nil {
			t.Errorf("KeyFromHex(%q) succeeded", value)
		}
		if _, err := crypto.PublicKeyFromHex(value); err == nil {
			t.Errorf("PublicKeyFromHex(%q) succeeded", value)
		}
	}
	for _, value := range [][]byte{nil, make([]byte, standard.PublicKeySize-1), make([]byte, standard.PublicKeySize+1)} {
		if _, err := crypto.PublicKeyToHex(value); err == nil {
			t.Errorf("PublicKeyToHex(%d bytes) succeeded", len(value))
		}
	}
}

func TestPEMKey(t *testing.T) {
	privateKey, err := crypto.KeyFromHex(seedHex)
	if err != nil {
		t.Fatal(err)
	}
	der, err := x509.MarshalPKCS8PrivateKey(privateKey)
	if err != nil {
		t.Fatal(err)
	}
	value := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der})
	got, err := crypto.PublicKeyFromPEM(value)
	if err != nil || !got.Equal(privateKey.Public().(standard.PublicKey)) {
		t.Fatalf("PKCS#8 key: %x, %v", got, err)
	}
	openSSHBlock, err := ssh.MarshalPrivateKey(privateKey, "test")
	if err != nil {
		t.Fatal(err)
	}
	got, err = crypto.PublicKeyFromPEM(pem.EncodeToMemory(openSSHBlock))
	if err != nil || !got.Equal(privateKey.Public().(standard.PublicKey)) {
		t.Fatalf("OpenSSH key: %x, %v", got, err)
	}
	path := filepath.Join(t.TempDir(), "local-key.pem")
	if err := os.WriteFile(path, value, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := crypto.PubFromPEMFile(path); err != nil {
		t.Fatal(err)
	}
}

func TestInvalidPEM(t *testing.T) {
	for _, value := range [][]byte{
		[]byte("not pem"),
		pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: []byte("invalid")}),
		pem.EncodeToMemory(&pem.Block{Type: "OTHER", Bytes: make([]byte, standard.SeedSize)}),
	} {
		if _, err := crypto.PublicKeyFromPEM(value); err == nil {
			t.Errorf("PublicKeyFromPEM accepted %q", value)
		}
	}
}
