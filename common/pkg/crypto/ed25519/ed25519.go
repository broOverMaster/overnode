// Пакет ed25519 предоставляет проверку и преобразование ключей Ed25519.
package ed25519

import (
	standard "crypto/ed25519"
	"crypto/x509"
	"encoding/hex"
	"encoding/pem"
	"fmt"
	"os"
	"strings"

	"golang.org/x/crypto/ssh"
)

// SeedFromHex декодирует 32-байтовый seed Ed25519 из 64 hex-символов.
func SeedFromHex(value string) ([]byte, error) {
	if len(value) != standard.SeedSize*2 {
		return nil, fmt.Errorf("Ed25519 seed must be exactly %d hex characters", standard.SeedSize*2)
	}
	seed, err := hex.DecodeString(value)
	if err != nil {
		return nil, fmt.Errorf("decode Ed25519 seed: %w", err)
	}
	return seed, nil
}

// SeedFromPEM декодирует seed Ed25519 из PEM-блока PRIVATE KEY формата PKCS#8
// или из блока ED25519 SEED с 32 необработанными байтами.
func SeedFromPEM(value []byte) ([]byte, error) {
	block, rest := pem.Decode(value)
	if block == nil || len(strings.TrimSpace(string(rest))) != 0 {
		return nil, fmt.Errorf("expected exactly one Ed25519 PEM block")
	}
	if block.Type == "ED25519 SEED" {
		if len(block.Bytes) != standard.SeedSize {
			return nil, fmt.Errorf("Ed25519 PEM seed must be exactly %d bytes", standard.SeedSize)
		}
		return append([]byte(nil), block.Bytes...), nil
	}
	if block.Type == "OPENSSH PRIVATE KEY" {
		privateKey, err := ssh.ParseRawPrivateKey(value)
		if err != nil {
			return nil, fmt.Errorf("parse OpenSSH private key: %w", err)
		}
		switch key := privateKey.(type) {
		case standard.PrivateKey:
			return append([]byte(nil), key.Seed()...), nil
		case *standard.PrivateKey:
			if key == nil {
				break
			}
			return append([]byte(nil), key.Seed()...), nil
		}
		return nil, fmt.Errorf("OpenSSH private key is not Ed25519")
	}
	if block.Type != "PRIVATE KEY" {
		return nil, fmt.Errorf("unsupported Ed25519 PEM block type %q", block.Type)
	}
	privateKey, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("parse Ed25519 PKCS#8 private key: %w", err)
	}
	key, ok := privateKey.(standard.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("PEM private key is not Ed25519")
	}
	return append([]byte(nil), key.Seed()...), nil
}

// SeedFromPEMFile читает PEM-файл и декодирует seed Ed25519.
func SeedFromPEMFile(path string) ([]byte, error) {
	value, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read Ed25519 PEM file: %w", err)
	}
	return SeedFromPEM(value)
}

// PrivateKeyFromPEM декодирует приватный ключ Ed25519 из PEM и возвращает
// полный ключ стандартного формата crypto/ed25519.
func PrivateKeyFromPEM(value []byte) (standard.PrivateKey, error) {
	seed, err := SeedFromPEM(value)
	if err != nil {
		return nil, err
	}
	return standard.NewKeyFromSeed(seed), nil
}

// PublicKeyFromPEM получает публичный ключ Ed25519 из PEM-приватного ключа.
func PublicKeyFromPEM(value []byte) (standard.PublicKey, error) {
	privateKey, err := PrivateKeyFromPEM(value)
	if err != nil {
		return nil, err
	}
	return privateKey.Public().(standard.PublicKey), nil
}

// PublicKeyFromPEMFile читает PEM-файл и получает из него публичный ключ Ed25519.
func PublicKeyFromPEMFile(path string) (standard.PublicKey, error) {
	value, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read Ed25519 PEM file: %w", err)
	}
	return PublicKeyFromPEM(value)
}

// PrivateKeyFromSeedHex создаёт приватный ключ Ed25519 из hex-представления seed.
func PrivateKeyFromSeedHex(value string) (standard.PrivateKey, error) {
	seed, err := SeedFromHex(value)
	if err != nil {
		return nil, err
	}
	return standard.NewKeyFromSeed(seed), nil
}

// PublicKeyFromSeedHex получает публичный ключ Ed25519 из hex-представления seed.
func PublicKeyFromSeedHex(value string) (standard.PublicKey, error) {
	privateKey, err := PrivateKeyFromSeedHex(value)
	if err != nil {
		return nil, err
	}
	publicKey := privateKey.Public().(standard.PublicKey)
	return publicKey, nil
}

// PublicKeyFromHex декодирует публичный ключ Ed25519 из 64 hex-символов.
func PublicKeyFromHex(value string) (standard.PublicKey, error) {
	if len(value) != standard.PublicKeySize*2 {
		return nil, fmt.Errorf("Ed25519 public key must be exactly %d hex characters", standard.PublicKeySize*2)
	}
	publicKey, err := hex.DecodeString(value)
	if err != nil {
		return nil, fmt.Errorf("decode Ed25519 public key: %w", err)
	}
	return standard.PublicKey(publicKey), nil
}

// PublicKeyToHex кодирует публичный ключ Ed25519 в lowercase hex.
func PublicKeyToHex(value []byte) (string, error) {
	if len(value) != standard.PublicKeySize {
		return "", fmt.Errorf("Ed25519 public key must be exactly %d bytes", standard.PublicKeySize)
	}
	return hex.EncodeToString(value), nil
}
