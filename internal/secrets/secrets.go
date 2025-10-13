package secrets

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

const keyFilename = "secret.key"

// EncryptString encrypts plaintext using the master key located alongside the config file.
func EncryptString(configPath, plaintext string) (string, error) {
	if plaintext == "" {
		return "", nil
	}
	key, err := ensureKey(configPath)
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("init cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("init gcm: %w", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("generate nonce: %w", err)
	}

	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// DecryptString decrypts ciphertext using the master key located alongside the config file.
func DecryptString(configPath, ciphertext string) (string, error) {
	if ciphertext == "" {
		return "", nil
	}

	keyPath := keyPathFromConfig(configPath)
	key, err := os.ReadFile(keyPath)
	if err != nil {
		return "", fmt.Errorf("read secret key: %w", err)
	}

	raw, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", fmt.Errorf("decode ciphertext: %w", err)
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("init cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("init gcm: %w", err)
	}

	if len(raw) < gcm.NonceSize() {
		return "", errors.New("ciphertext too short")
	}

	nonce := raw[:gcm.NonceSize()]
	cipherText := raw[gcm.NonceSize():]

	plain, err := gcm.Open(nil, nonce, cipherText, nil)
	if err != nil {
		return "", fmt.Errorf("decrypt: %w", err)
	}

	return string(plain), nil
}

func ensureKey(configPath string) ([]byte, error) {
	keyPath := keyPathFromConfig(configPath)
	if _, err := os.Stat(keyPath); errors.Is(err, os.ErrNotExist) {
		if err := os.MkdirAll(filepath.Dir(keyPath), 0o700); err != nil {
			return nil, fmt.Errorf("ensure secret dir: %w", err)
		}
		key := make([]byte, 32)
		if _, err := rand.Read(key); err != nil {
			return nil, fmt.Errorf("generate secret key: %w", err)
		}
		if err := os.WriteFile(keyPath, key, 0o600); err != nil {
			return nil, fmt.Errorf("write secret key: %w", err)
		}
		return key, nil
	} else if err != nil {
		return nil, fmt.Errorf("stat secret key: %w", err)
	}

	key, err := os.ReadFile(keyPath)
	if err != nil {
		return nil, fmt.Errorf("read secret key: %w", err)
	}
	if len(key) != 32 {
		return nil, fmt.Errorf("invalid secret key length: %d", len(key))
	}
	return key, nil
}

func keyPathFromConfig(configPath string) string {
	dir := filepath.Dir(configPath)
	return filepath.Join(dir, keyFilename)
}
