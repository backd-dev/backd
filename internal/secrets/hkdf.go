package secrets

import (
	"crypto/sha256"
	"io"

	"golang.org/x/crypto/hkdf"
)

// hkdfSalt is fixed and versioned — change version string to rotate all keys
const hkdfSalt = "backd-app-key-v1"

// DeriveAppKey derives a 32-byte key for an app using HKDF-SHA256
// This is the ONLY file that imports golang.org/x/crypto/hkdf
func DeriveAppKey(masterKey []byte, appName string) []byte {
	r := hkdf.New(sha256.New, masterKey, []byte(hkdfSalt), []byte(appName))
	key := make([]byte, 32)
	_, err := io.ReadFull(r, key)
	if err != nil {
		// This should never happen with HKDF
		panic("failed to derive app key: " + err.Error())
	}
	return key
}

// DeriveDomainKey derives a 32-byte key for a domain using HKDF-SHA256
// Uses separate namespace from app keys
func DeriveDomainKey(masterKey []byte, domainName string) []byte {
	info := "_domain_" + domainName
	r := hkdf.New(sha256.New, masterKey, []byte(hkdfSalt), []byte(info))
	key := make([]byte, 32)
	_, err := io.ReadFull(r, key)
	if err != nil {
		// This should never happen with HKDF
		panic("failed to derive domain key: " + err.Error())
	}
	return key
}
