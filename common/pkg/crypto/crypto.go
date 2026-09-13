// Пакет crypto предоставляет общие криптографические функции common.
package crypto

import (
	standard "crypto/ed25519"
	"fmt"
	"os"

	ed25519 "overnode/common/pkg/crypto/ed25519"
)

// KeyFromHex загружает приватный ключ из seed или полного ключа.
func KeyFromHex(value string) (standard.PrivateKey, error) {
	return ed25519.PrivateKeyFromSeedHex(value)
}

// PubFromHex получает публичный ключ из seed или приватного ключа.
func PubFromHex(value string) (standard.PublicKey, error) {
	return ed25519.PublicKeyFromSeedHex(value)
}

// PublicKeyFromHex загружает публичный ключ из hex-представления.
func PublicKeyFromHex(value string) (standard.PublicKey, error) {
	return ed25519.PublicKeyFromHex(value)
}

// PublicKeyToHex кодирует публичный ключ в hex-представление.
func PublicKeyToHex(value []byte) (string, error) {
	return ed25519.PublicKeyToHex(value)
}

// PrivateKeyFromPEM загружает приватный ключ из PKCS#8 или OpenSSH PEM.
func PrivateKeyFromPEM(value []byte) (standard.PrivateKey, error) {
	key, err := privateKeyFromPEM_OpenSSH(value)
	if err != nil {
		key, err = privateKeyFromPEM_PKCS8(value)
		if err != nil {
			return nil, err
		}
	}
	return key, nil
}

// KeyFromPEMFile загружает приватный ключ из PEM файла.
func KeyFromPEMFile(path string) (standard.PrivateKey, error) {
	value, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read PEM file: %w", err)
	}
	return PrivateKeyFromPEM(value)
}

// PublicKeyFromPEM загружает публичный ключ из приватного PKCS#8 или OpenSSH PEM.
func PublicKeyFromPEM(value []byte) (standard.PublicKey, error) {
	key, err := publicKeyFromPEM_OpenSSH(value)
	if err != nil {
		key, err = publicKeyFromPEM_PKCS8(value)
		if err != nil {
			return nil, err
		}
	}
	return key, nil
}

// PubFromPEMFile получает публичный ключ из PEM файла.
func PubFromPEMFile(path string) (standard.PublicKey, error) {
	value, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read PEM file: %w", err)
	}
	return PublicKeyFromPEM(value)
}
