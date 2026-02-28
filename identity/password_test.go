package identity

import (
	"strings"
	"testing"
)

func TestHashPassword(t *testing.T) {
	password := "testPassword123!"
	hash, err := HashPassword(password, nil)
	if err != nil {
		t.Fatalf("HashPassword failed: %v", err)
	}

	// Check hash format
	if !strings.HasPrefix(hash, "$argon2id$v=") {
		t.Errorf("hash has wrong prefix: %s", hash)
	}

	parts := strings.Split(hash, "$")
	if len(parts) != 6 {
		t.Errorf("hash has wrong number of parts: %d", len(parts))
	}
}

func TestVerifyPassword(t *testing.T) {
	password := "testPassword123!"
	hash, err := HashPassword(password, nil)
	if err != nil {
		t.Fatalf("HashPassword failed: %v", err)
	}

	// Correct password should verify
	valid, err := VerifyPassword(password, hash)
	if err != nil {
		t.Fatalf("VerifyPassword failed: %v", err)
	}
	if !valid {
		t.Error("VerifyPassword returned false for correct password")
	}

	// Wrong password should not verify
	valid, err = VerifyPassword("wrongPassword", hash)
	if err != nil {
		t.Fatalf("VerifyPassword failed: %v", err)
	}
	if valid {
		t.Error("VerifyPassword returned true for wrong password")
	}
}

func TestVerifyPassword_InvalidHash(t *testing.T) {
	_, err := VerifyPassword("password", "invalid-hash")
	if err != ErrInvalidHash {
		t.Errorf("expected ErrInvalidHash, got: %v", err)
	}
}

func TestNeedsRehash(t *testing.T) {
	password := "testPassword123!"

	// Hash with default params
	hash, err := HashPassword(password, nil)
	if err != nil {
		t.Fatalf("HashPassword failed: %v", err)
	}

	// Should not need rehash with same params
	needsRehash, err := NeedsRehash(hash, nil)
	if err != nil {
		t.Fatalf("NeedsRehash failed: %v", err)
	}
	if needsRehash {
		t.Error("NeedsRehash returned true with same params")
	}

	// Should need rehash with different params
	newParams := &Argon2idParams{
		Memory:      128 * 1024, // Different memory
		Iterations:  3,
		Parallelism: 2,
		SaltLength:  16,
		KeyLength:   32,
	}
	needsRehash, err = NeedsRehash(hash, newParams)
	if err != nil {
		t.Fatalf("NeedsRehash failed: %v", err)
	}
	if !needsRehash {
		t.Error("NeedsRehash returned false with different params")
	}
}

func TestCustomParams(t *testing.T) {
	password := "testPassword123!"
	params := &Argon2idParams{
		Memory:      32 * 1024,
		Iterations:  2,
		Parallelism: 1,
		SaltLength:  16,
		KeyLength:   32,
	}

	hash, err := HashPassword(password, params)
	if err != nil {
		t.Fatalf("HashPassword with custom params failed: %v", err)
	}

	valid, err := VerifyPassword(password, hash)
	if err != nil {
		t.Fatalf("VerifyPassword failed: %v", err)
	}
	if !valid {
		t.Error("VerifyPassword returned false for correct password with custom params")
	}
}
