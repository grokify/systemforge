package scim

import (
	"golang.org/x/crypto/bcrypt"
)

// PasswordHasher defines the interface for password hashing operations.
type PasswordHasher interface {
	// Hash hashes a plain text password.
	Hash(password string) (string, error)

	// Verify compares a plain text password with a hashed password.
	Verify(password, hash string) error
}

// BcryptHasher implements PasswordHasher using bcrypt.
type BcryptHasher struct {
	// Cost is the bcrypt cost parameter. Default is bcrypt.DefaultCost (10).
	Cost int
}

// NewBcryptHasher creates a new bcrypt password hasher.
func NewBcryptHasher(cost int) *BcryptHasher {
	if cost == 0 {
		cost = bcrypt.DefaultCost
	}
	return &BcryptHasher{Cost: cost}
}

// Hash hashes a password using bcrypt.
func (h *BcryptHasher) Hash(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), h.Cost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

// Verify compares a password with a bcrypt hash.
func (h *BcryptHasher) Verify(password, hash string) error {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
}

// NoOpHasher is a PasswordHasher that does not hash passwords.
// This is useful for testing or when passwords are hashed elsewhere.
type NoOpHasher struct{}

// Hash returns the password as-is.
func (h *NoOpHasher) Hash(password string) (string, error) {
	return password, nil
}

// Verify compares passwords directly.
func (h *NoOpHasher) Verify(password, hash string) error {
	if password == hash {
		return nil
	}
	return ErrUnauthorized("password mismatch")
}

// DefaultPasswordHasher returns the default password hasher (bcrypt).
func DefaultPasswordHasher() PasswordHasher {
	return NewBcryptHasher(bcrypt.DefaultCost)
}
