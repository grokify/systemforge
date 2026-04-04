package bff

import (
	"bytes"
	"crypto/rand"
	"testing"
)

func TestEncryptor_RoundTrip(t *testing.T) {
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		t.Fatal(err)
	}

	enc, err := NewEncryptor(key)
	if err != nil {
		t.Fatalf("NewEncryptor: %v", err)
	}

	plaintext := []byte("hello world, this is a secret token")

	ciphertext, err := enc.Encrypt(plaintext)
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}

	// Ciphertext should be different from plaintext
	if bytes.Equal(ciphertext, plaintext) {
		t.Error("ciphertext should not equal plaintext")
	}

	decrypted, err := enc.Decrypt(ciphertext)
	if err != nil {
		t.Fatalf("Decrypt: %v", err)
	}

	if !bytes.Equal(decrypted, plaintext) {
		t.Errorf("decrypted = %q, want %q", decrypted, plaintext)
	}
}

func TestEncryptor_StringRoundTrip(t *testing.T) {
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		t.Fatal(err)
	}

	enc, err := NewEncryptor(key)
	if err != nil {
		t.Fatalf("NewEncryptor: %v", err)
	}

	plaintext := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIn0.dozjgNryP4J3jVmNHl0w5N_XgL0n3I9PlFUP0THsR8U"

	ciphertext, err := enc.EncryptString(plaintext)
	if err != nil {
		t.Fatalf("EncryptString: %v", err)
	}

	decrypted, err := enc.DecryptString(ciphertext)
	if err != nil {
		t.Fatalf("DecryptString: %v", err)
	}

	if decrypted != plaintext {
		t.Errorf("decrypted = %q, want %q", decrypted, plaintext)
	}
}

func TestEncryptor_DifferentNonces(t *testing.T) {
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		t.Fatal(err)
	}

	enc, err := NewEncryptor(key)
	if err != nil {
		t.Fatalf("NewEncryptor: %v", err)
	}

	plaintext := []byte("same plaintext")

	// Encrypt the same plaintext twice
	ciphertext1, err := enc.Encrypt(plaintext)
	if err != nil {
		t.Fatalf("first Encrypt: %v", err)
	}

	ciphertext2, err := enc.Encrypt(plaintext)
	if err != nil {
		t.Fatalf("second Encrypt: %v", err)
	}

	// Different encryptions should produce different ciphertext (due to random nonce)
	if bytes.Equal(ciphertext1, ciphertext2) {
		t.Error("encrypting same plaintext twice should produce different ciphertext")
	}

	// But both should decrypt to the same plaintext
	decrypted1, _ := enc.Decrypt(ciphertext1)
	decrypted2, _ := enc.Decrypt(ciphertext2)

	if !bytes.Equal(decrypted1, decrypted2) {
		t.Error("both ciphertexts should decrypt to the same plaintext")
	}
}

func TestEncryptor_InvalidKeySize(t *testing.T) {
	tests := []struct {
		name    string
		keySize int
	}{
		{"too short", 16},
		{"too long", 64},
		{"empty", 0},
		{"just under", 31},
		{"just over", 33},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key := make([]byte, tt.keySize)
			_, err := NewEncryptor(key)
			if err != ErrInvalidKeySize {
				t.Errorf("NewEncryptor with %d bytes: got err %v, want ErrInvalidKeySize", tt.keySize, err)
			}
		})
	}
}

func TestEncryptor_CiphertextTooShort(t *testing.T) {
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		t.Fatal(err)
	}

	enc, err := NewEncryptor(key)
	if err != nil {
		t.Fatalf("NewEncryptor: %v", err)
	}

	// GCM nonce is 12 bytes, so anything less should fail
	shortCiphertext := []byte("short")
	_, err = enc.Decrypt(shortCiphertext)
	if err != ErrCiphertextTooShort {
		t.Errorf("Decrypt with short ciphertext: got err %v, want ErrCiphertextTooShort", err)
	}
}

func TestEncryptor_CorruptedCiphertext(t *testing.T) {
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		t.Fatal(err)
	}

	enc, err := NewEncryptor(key)
	if err != nil {
		t.Fatalf("NewEncryptor: %v", err)
	}

	plaintext := []byte("secret data")
	ciphertext, err := enc.Encrypt(plaintext)
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}

	// Corrupt the ciphertext (flip some bits)
	corruptedCiphertext := make([]byte, len(ciphertext))
	copy(corruptedCiphertext, ciphertext)
	corruptedCiphertext[len(corruptedCiphertext)-1] ^= 0xFF

	_, err = enc.Decrypt(corruptedCiphertext)
	if err == nil {
		t.Error("Decrypt with corrupted ciphertext should fail")
	}
}

func TestEncryptor_WrongKey(t *testing.T) {
	key1 := make([]byte, 32)
	key2 := make([]byte, 32)
	if _, err := rand.Read(key1); err != nil {
		t.Fatal(err)
	}
	if _, err := rand.Read(key2); err != nil {
		t.Fatal(err)
	}

	enc1, _ := NewEncryptor(key1)
	enc2, _ := NewEncryptor(key2)

	plaintext := []byte("secret data")
	ciphertext, _ := enc1.Encrypt(plaintext)

	// Decrypting with a different key should fail
	_, err := enc2.Decrypt(ciphertext)
	if err == nil {
		t.Error("Decrypt with wrong key should fail")
	}
}

func TestEncryptor_EmptyPlaintext(t *testing.T) {
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		t.Fatal(err)
	}

	enc, err := NewEncryptor(key)
	if err != nil {
		t.Fatalf("NewEncryptor: %v", err)
	}

	plaintext := []byte{}
	ciphertext, err := enc.Encrypt(plaintext)
	if err != nil {
		t.Fatalf("Encrypt empty: %v", err)
	}

	decrypted, err := enc.Decrypt(ciphertext)
	if err != nil {
		t.Fatalf("Decrypt empty: %v", err)
	}

	if len(decrypted) != 0 {
		t.Errorf("decrypted empty should be empty, got %q", decrypted)
	}
}
