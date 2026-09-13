package crypto

import (
	standard "crypto/ed25519"
	"encoding/pem"
	"fmt"
	"strings"

	"golang.org/x/crypto/ssh"
)

func privateKeyFromPEM_OpenSSH(value []byte) (standard.PrivateKey, error) {
	block, rest := pem.Decode(value)
	if block == nil || len(strings.TrimSpace(string(rest))) != 0 {
		return nil, fmt.Errorf("expected exactly one PEM block")
	}
	if block.Type == "OPENSSH PRIVATE KEY" {
		privateKey, err := ssh.ParseRawPrivateKey(value)
		if err != nil {
			return nil, fmt.Errorf("parse OpenSSH private key: %w", err)
		}
		switch key := privateKey.(type) {
		case standard.PrivateKey:
			return append(standard.PrivateKey(nil), key...), nil
		case *standard.PrivateKey:
			if key == nil {
				break
			}
			return append(standard.PrivateKey(nil), (*key)...), nil
		}
		return nil, fmt.Errorf("OpenSSH private key is not Ed25519")
	}
	return nil, fmt.Errorf("unsupported PEM block type %q", block.Type)
}

func publicKeyFromPEM_OpenSSH(value []byte) (standard.PublicKey, error) {
	privateKey, err := privateKeyFromPEM_OpenSSH(value)
	if err != nil {
		return nil, err
	}
	return append(standard.PublicKey(nil), privateKey.Public().(standard.PublicKey)...), nil
}
