// Package identity provides identity management for CoreForge applications.
package identity

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"golang.org/x/crypto/argon2"
)

// Argon2idParams holds the parameters for Argon2id password hashing.
type Argon2idParams struct {
	Memory      uint32 // Memory in KiB
	Iterations  uint32 // Number of iterations
	Parallelism uint8  // Degree of parallelism
	SaltLength  uint32 // Salt length in bytes
	KeyLength   uint32 // Hash length in bytes
}

// DefaultArgon2idParams returns recommended Argon2id parameters.
// These follow OWASP recommendations for password hashing.
func DefaultArgon2idParams() *Argon2idParams {
	return &Argon2idParams{
		Memory:      64 * 1024, // 64 MiB
		Iterations:  3,
		Parallelism: 2,
		SaltLength:  16,
		KeyLength:   32,
	}
}

var (
	// ErrInvalidHash indicates the hash format is invalid.
	ErrInvalidHash = errors.New("invalid hash format")
	// ErrIncompatibleVersion indicates the Argon2 version is incompatible.
	ErrIncompatibleVersion = errors.New("incompatible argon2 version")
)

// HashPassword hashes a password using Argon2id with the given parameters.
// Returns the encoded hash in the format: $argon2id$v=19$m=65536,t=3,p=2$salt$hash
func HashPassword(password string, params *Argon2idParams) (string, error) {
	if params == nil {
		params = DefaultArgon2idParams()
	}

	salt := make([]byte, params.SaltLength)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("generating salt: %w", err)
	}

	hash := argon2.IDKey(
		[]byte(password),
		salt,
		params.Iterations,
		params.Memory,
		params.Parallelism,
		params.KeyLength,
	)

	// Encode to standard Argon2 format
	b64Salt := base64.RawStdEncoding.EncodeToString(salt)
	b64Hash := base64.RawStdEncoding.EncodeToString(hash)

	encoded := fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version, params.Memory, params.Iterations, params.Parallelism,
		b64Salt, b64Hash)

	return encoded, nil
}

// VerifyPassword verifies a password against an encoded Argon2id hash.
// Returns true if the password matches, false otherwise.
func VerifyPassword(password, encodedHash string) (bool, error) {
	params, salt, hash, err := decodeHash(encodedHash)
	if err != nil {
		return false, err
	}

	otherHash := argon2.IDKey(
		[]byte(password),
		salt,
		params.Iterations,
		params.Memory,
		params.Parallelism,
		params.KeyLength,
	)

	// Constant-time comparison to prevent timing attacks
	return subtle.ConstantTimeCompare(hash, otherHash) == 1, nil
}

// decodeHash decodes an Argon2id hash string into its components.
func decodeHash(encodedHash string) (*Argon2idParams, []byte, []byte, error) {
	parts := strings.Split(encodedHash, "$")
	if len(parts) != 6 {
		return nil, nil, nil, ErrInvalidHash
	}

	if parts[1] != "argon2id" {
		return nil, nil, nil, ErrInvalidHash
	}

	var version int
	if _, err := fmt.Sscanf(parts[2], "v=%d", &version); err != nil {
		return nil, nil, nil, ErrInvalidHash
	}
	if version != argon2.Version {
		return nil, nil, nil, ErrIncompatibleVersion
	}

	params := &Argon2idParams{}
	if _, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d",
		&params.Memory, &params.Iterations, &params.Parallelism); err != nil {
		return nil, nil, nil, ErrInvalidHash
	}

	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return nil, nil, nil, ErrInvalidHash
	}
	params.SaltLength = uint32(len(salt)) //nolint:gosec // G115: salt length is bounded (typically 16 bytes)

	hash, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return nil, nil, nil, ErrInvalidHash
	}
	params.KeyLength = uint32(len(hash)) //nolint:gosec // G115: hash length is bounded (typically 32 bytes)

	return params, salt, hash, nil
}

// NeedsRehash checks if a hash needs to be regenerated with new parameters.
// This is useful when upgrading hash parameters over time.
func NeedsRehash(encodedHash string, params *Argon2idParams) (bool, error) {
	if params == nil {
		params = DefaultArgon2idParams()
	}

	currentParams, _, _, err := decodeHash(encodedHash)
	if err != nil {
		return false, err
	}

	// Check if any parameters differ
	return currentParams.Memory != params.Memory ||
		currentParams.Iterations != params.Iterations ||
		currentParams.Parallelism != params.Parallelism ||
		currentParams.KeyLength != params.KeyLength, nil
}
