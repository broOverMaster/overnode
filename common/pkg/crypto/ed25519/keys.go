// Пакет ed25519 предоставляет проверку и преобразование ключей Ed25519.
package ed25519

import (
	standard "crypto/ed25519"
	"encoding/hex"
	"fmt"
)

// PrivateKeyFromSeedHex создаёт приватный ключ Ed25519 из hex-представления
// seed или полного приватного ключа.
func PrivateKeyFromSeedHex(value string) (standard.PrivateKey, error) {
	if len(value) != standard.SeedSize*2 && len(value) != standard.PrivateKeySize*2 {
		return nil, fmt.Errorf("Ed25519 key must be %d or %d hex characters", standard.SeedSize*2, standard.PrivateKeySize*2)
	}
	decoded, err := hex.DecodeString(value)
	if err != nil {
		return nil, fmt.Errorf("decode Ed25519 private key: %w", err)
	}
	if len(decoded) == standard.SeedSize {
		return standard.NewKeyFromSeed(decoded), nil
	}
	return standard.PrivateKey(decoded), nil
}

// PrivateKeyFromHex — краткое имя для PrivateKeyFromSeedHex.
func PrivateKeyFromHex(value string) (standard.PrivateKey, error) {
	return PrivateKeyFromSeedHex(value)
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
