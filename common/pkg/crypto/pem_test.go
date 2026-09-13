package crypto

import (
	standard "crypto/ed25519"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"testing"

	"golang.org/x/crypto/ssh"
)

func testPrivateKey(t *testing.T) standard.PrivateKey {
	t.Helper()
	_, privateKey, err := standard.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	return privateKey
}

func TestPKCS8PEM(t *testing.T) {
	privateKey := testPrivateKey(t)
	der, err := x509.MarshalPKCS8PrivateKey(privateKey)
	if err != nil {
		t.Fatal(err)
	}
	value := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der})

	gotPrivateKey, err := privateKeyFromPEM_PKCS8(value)
	if err != nil {
		t.Fatal(err)
	}
	if !gotPrivateKey.Equal(privateKey) {
		t.Fatal("PKCS#8 private key mismatch")
	}
	gotPublicKey, err := publicKeyFromPEM_PKCS8(value)
	if err != nil {
		t.Fatal(err)
	}
	if !gotPublicKey.Equal(privateKey.Public()) {
		t.Fatal("PKCS#8 public key mismatch")
	}
}

func TestOpenSSHPEM(t *testing.T) {
	privateKey := testPrivateKey(t)
	block, err := ssh.MarshalPrivateKey(privateKey, "test")
	if err != nil {
		t.Fatal(err)
	}
	value := pem.EncodeToMemory(block)

	gotPrivateKey, err := privateKeyFromPEM_OpenSSH(value)
	if err != nil {
		t.Fatal(err)
	}
	if !gotPrivateKey.Equal(privateKey) {
		t.Fatal("OpenSSH private key mismatch")
	}
	gotPublicKey, err := publicKeyFromPEM_OpenSSH(value)
	if err != nil {
		t.Fatal(err)
	}
	if !gotPublicKey.Equal(privateKey.Public()) {
		t.Fatal("OpenSSH public key mismatch")
	}
}

func TestPEMDecodersRejectInvalidInput(t *testing.T) {
	values := [][]byte{
		[]byte("not PEM"),
		pem.EncodeToMemory(&pem.Block{Type: "OTHER", Bytes: []byte("invalid")}),
		append(pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: []byte("invalid")}), []byte("extra")...),
	}
	for _, value := range values {
		if _, err := publicKeyFromPEM_PKCS8(value); err == nil {
			t.Errorf("publicKeyFromPEM_PKCS8 accepted invalid input")
		}
		if _, err := publicKeyFromPEM_OpenSSH(value); err == nil {
			t.Errorf("publicKeyFromPEM_OpenSSH accepted invalid input")
		}
	}
}
