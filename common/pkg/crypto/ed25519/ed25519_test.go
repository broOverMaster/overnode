package ed25519_test

import (
	standard "crypto/ed25519"
	"strings"
	"testing"

	"overnode/common/pkg/crypto/ed25519"
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
