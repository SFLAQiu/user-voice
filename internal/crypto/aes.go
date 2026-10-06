package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
)

const encPrefix = "ENC:"

// Cipher provides AES-256-GCM encryption helpers.
type Cipher struct {
	aead cipher.AEAD
}

// New creates a Cipher from a base64-encoded 32-byte key.
func New(b64Key string) (*Cipher, error) {
	key, err := base64.StdEncoding.DecodeString(b64Key)
	if err != nil {
		return nil, fmt.Errorf("decode key: %w", err)
	}
	if len(key) != 32 {
		return nil, fmt.Errorf("encryption key must decode to 32 bytes, got %d", len(key))
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	return &Cipher{aead: aead}, nil
}

// Encrypt returns "ENC:<base64(nonce|ciphertext)>".
func (c *Cipher) Encrypt(plaintext string) (string, error) {
	if plaintext == "" {
		return "", nil
	}
	nonce := make([]byte, c.aead.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	ct := c.aead.Seal(nil, nonce, []byte(plaintext), nil)
	out := make([]byte, 0, len(nonce)+len(ct))
	out = append(out, nonce...)
	out = append(out, ct...)
	return encPrefix + base64.StdEncoding.EncodeToString(out), nil
}

// Decrypt accepts either an "ENC:..." string or returns the input as-is.
func (c *Cipher) Decrypt(s string) (string, error) {
	if s == "" {
		return "", nil
	}
	if len(s) < len(encPrefix) || s[:len(encPrefix)] != encPrefix {
		return s, nil
	}
	raw, err := base64.StdEncoding.DecodeString(s[len(encPrefix):])
	if err != nil {
		return "", fmt.Errorf("decode cipher: %w", err)
	}
	ns := c.aead.NonceSize()
	if len(raw) < ns+1 {
		return "", errors.New("ciphertext too short")
	}
	nonce, ct := raw[:ns], raw[ns:]
	pt, err := c.aead.Open(nil, nonce, ct, nil)
	if err != nil {
		return "", fmt.Errorf("decrypt: %w", err)
	}
	return string(pt), nil
}

// IsEncrypted reports whether s already carries the encrypted prefix.
func IsEncrypted(s string) bool {
	return len(s) >= len(encPrefix) && s[:len(encPrefix)] == encPrefix
}
