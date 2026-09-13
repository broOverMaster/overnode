package crypto

import (
	standard "crypto/ed25519"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"strings"
)

func privateKeyFromPEM_PKCS8(value []byte) (standard.PrivateKey, error) {
	block, rest := pem.Decode(value)
	if block == nil || len(strings.TrimSpace(string(rest))) != 0 {
		return nil, fmt.Errorf("expected exactly one PEM block")
	}
	if block.Type == "PRIVATE KEY" {
		privateKey, err := x509.ParsePKCS8PrivateKey(block.Bytes)
		if err != nil {
			return nil, fmt.Errorf("parse PKCS#8 private key: %w", err)
		}
		key, ok := privateKey.(standard.PrivateKey)
		if !ok {
			return nil, fmt.Errorf("PKCS#8 private key is not Ed25519")
		}
		return append(standard.PrivateKey(nil), key...), nil
	}
	return nil, fmt.Errorf("unsupported PEM block type %q", block.Type)
}

func publicKeyFromPEM_PKCS8(value []byte) (standard.PublicKey, error) {
	privateKey, err := privateKeyFromPEM_PKCS8(value)
	if err != nil {
		return nil, err
	}
	return append(standard.PublicKey(nil), privateKey.Public().(standard.PublicKey)...), nil
}
