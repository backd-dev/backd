package secrets

import (
	"crypto/sha256"
	"fmt"
	"io"

	"golang.org/x/crypto/hkdf"
)

// hkdfSalt is fixed and versioned — change version string to rotate all keys
const hkdfSalt = "backd-app-key-v1"

// DeriveAppKey derives a 32-byte key for an app using HKDF-SHA256
// This is the ONLY file that imports golang.org/x/crypto/hkdf
func DeriveAppKey(masterKey []byte, appName string) ([]byte, error) {
	r := hkdf.New(sha256.New, masterKey, []byte(hkdfSalt), []byte(appName))
	key := make([]byte, 32)
	if _, err := io.ReadFull(r, key); err != nil {
		return nil, fmt.Errorf("hkdf.DeriveAppKey: %w", err)
	}
	return key, nil
}

// DeriveDomainKey derives a 32-byte key for a domain using HKDF-SHA256
// Uses separate namespace from app keys
func DeriveDomainKey(masterKey []byte, domainName string) ([]byte, error) {
	info := "_domain_" + domainName
	r := hkdf.New(sha256.New, masterKey, []byte(hkdfSalt), []byte(info))
	key := make([]byte, 32)
	if _, err := io.ReadFull(r, key); err != nil {
		return nil, fmt.Errorf("hkdf.DeriveDomainKey: %w", err)
	}
	return key, nil
}
