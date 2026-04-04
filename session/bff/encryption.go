package bff

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"errors"
	"io"
)

var (
	// ErrInvalidKeySize is returned when the encryption key is not 32 bytes.
	ErrInvalidKeySize = errors.New("encryption key must be 32 bytes for AES-256")
	// ErrCiphertextTooShort is returned when ciphertext is shorter than the nonce.
	ErrCiphertextTooShort = errors.New("ciphertext too short")
)

// Encryptor provides AES-256-GCM encryption for tokens at rest.
// It is safe for concurrent use.
type Encryptor struct {
	gcm cipher.AEAD
}

// NewEncryptor creates a new encryptor with the given 32-byte key.
// The key must be exactly 32 bytes for AES-256.
func NewEncryptor(key []byte) (*Encryptor, error) {
	if len(key) != 32 {
		return nil, ErrInvalidKeySize
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	return &Encryptor{gcm: gcm}, nil
}

// Encrypt encrypts plaintext using AES-256-GCM.
// The nonce is prepended to the ciphertext.
func (e *Encryptor) Encrypt(plaintext []byte) ([]byte, error) {
	nonce := make([]byte, e.gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	// Seal appends the ciphertext to nonce
	return e.gcm.Seal(nonce, nonce, plaintext, nil), nil
}

// Decrypt decrypts ciphertext that was encrypted with Encrypt.
// The ciphertext must have the nonce prepended.
func (e *Encryptor) Decrypt(ciphertext []byte) ([]byte, error) {
	if len(ciphertext) < e.gcm.NonceSize() {
		return nil, ErrCiphertextTooShort
	}

	nonce, ciphertext := ciphertext[:e.gcm.NonceSize()], ciphertext[e.gcm.NonceSize():]
	return e.gcm.Open(nil, nonce, ciphertext, nil)
}

// EncryptString encrypts a string and returns the ciphertext.
func (e *Encryptor) EncryptString(plaintext string) ([]byte, error) {
	return e.Encrypt([]byte(plaintext))
}

// DecryptString decrypts ciphertext and returns the plaintext string.
func (e *Encryptor) DecryptString(ciphertext []byte) (string, error) {
	plaintext, err := e.Decrypt(ciphertext)
	if err != nil {
		return "", err
	}
	return string(plaintext), nil
}
