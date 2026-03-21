package secrets

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"io"
)

// Encrypt encrypts plaintext using AES-256-GCM with a random nonce
// Returns: [12-byte nonce][ciphertext]
// This is the ONLY file that imports crypto/aes and crypto/cipher
func Encrypt(key []byte, plaintext string) (string, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonce := make([]byte, gcm.NonceSize()) // 12 bytes
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return string(ciphertext), nil
}

// Decrypt decrypts ciphertext using AES-256-GCM
// Expects format: [12-byte nonce][ciphertext]
func Decrypt(key []byte, ciphertext string) (string, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return "", io.ErrUnexpectedEOF
	}

	nonce := []byte(ciphertext[:nonceSize])
	ct := []byte(ciphertext[nonceSize:])

	plaintext, err := gcm.Open(nil, nonce, ct, nil)
	if err != nil {
		return "", err
	}

	return string(plaintext), nil
}
