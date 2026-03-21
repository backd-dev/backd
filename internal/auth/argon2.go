package auth

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"log/slog"
	"strings"

	"golang.org/x/crypto/argon2"
)

const (
	argonMemory      = 64 * 1024 // 64MB
	argonIterations  = 3
	argonParallelism = 4
	argonKeyLen      = 32
	argonSaltLen     = 16
)

// HashPassword hashes a password using Argon2id with random salt
func HashPassword(password string) (string, error) {
	if password == "" {
		return "", fmt.Errorf("password cannot be empty")
	}

	// Generate random salt
	salt := make([]byte, argonSaltLen)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("failed to generate salt: %w", err)
	}

	// Hash the password
	hash := argon2.IDKey([]byte(password), salt, uint32(argonIterations), uint32(argonMemory/argonParallelism), uint8(argonParallelism), uint32(argonKeyLen))

	// Encode salt and hash for storage
	saltB64 := base64.RawStdEncoding.EncodeToString(salt)
	hashB64 := base64.RawStdEncoding.EncodeToString(hash)

	// Format: $argon2id$v=19$m=65536,t=3,p=4$<salt>$<hash>
	encodedHash := fmt.Sprintf("$argon2id$v=19$m=%d,t=%d,p=%d$%s$%s",
		argonMemory, argonIterations, argonParallelism, saltB64, hashB64)

	slog.Debug("password hashed", "memory", argonMemory, "iterations", argonIterations, "parallelism", argonParallelism)

	return encodedHash, nil
}

// VerifyPassword verifies a password against its hash using constant-time comparison
func VerifyPassword(password, hash string) bool {
	if password == "" || hash == "" {
		return false
	}

	// Parse the hash format
	parts := strings.Split(hash, "$")
	if len(parts) != 6 || parts[0] != "" || parts[1] != "argon2id" || parts[2] != "v=19" {
		slog.Warn("invalid hash format")
		return false
	}

	// Parse parameters
	var memory, iterations, parallelism int
	_, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &memory, &iterations, &parallelism)
	if err != nil {
		slog.Warn("failed to parse hash parameters", "error", err)
		return false
	}

	// Decode salt and hash
	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		slog.Warn("failed to decode salt", "error", err)
		return false
	}

	expectedHash, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		slog.Warn("failed to decode hash", "error", err)
		return false
	}

	// Verify parameters match our constants
	if memory != argonMemory || iterations != argonIterations || parallelism != argonParallelism {
		slog.Warn("hash parameters don't match expected values",
			"memory", memory, "iterations", iterations, "parallelism", parallelism)
		return false
	}

	// Hash the provided password with the same salt
	computedHash := argon2.IDKey([]byte(password), salt, uint32(iterations), uint32(memory/parallelism), uint8(parallelism), uint32(len(expectedHash)))

	// Use constant-time comparison to prevent timing attacks
	return subtle.ConstantTimeCompare(computedHash, expectedHash) == 1
}
