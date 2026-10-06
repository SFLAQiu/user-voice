package crypto

import (
	"crypto/rand"
	"encoding/base64"
	"testing"
)

func newTestCipher(t *testing.T) *Cipher {
	t.Helper()
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		t.Fatalf("rand: %v", err)
	}
	c, err := New(base64.StdEncoding.EncodeToString(key))
	if err != nil {
		t.Fatalf("new cipher: %v", err)
	}
	return c
}

func TestEncryptDecryptRoundTrip(t *testing.T) {
	c := newTestCipher(t)
	cases := []string{"", "hello", "中文与emoji😀", "cookie=abc; sess=xyz"}
	for _, in := range cases {
		ct, err := c.Encrypt(in)
		if err != nil {
			t.Fatalf("encrypt %q: %v", in, err)
		}
		if in != "" && !IsEncrypted(ct) {
			t.Fatalf("expected ENC prefix, got %q", ct)
		}
		pt, err := c.Decrypt(ct)
		if err != nil {
			t.Fatalf("decrypt %q: %v", in, err)
		}
		if pt != in {
			t.Fatalf("round-trip mismatch: got %q want %q", pt, in)
		}
	}
}

func TestDecryptPassthroughPlain(t *testing.T) {
	c := newTestCipher(t)
	out, err := c.Decrypt("not-encrypted")
	if err != nil {
		t.Fatalf("decrypt plain: %v", err)
	}
	if out != "not-encrypted" {
		t.Fatalf("expected passthrough, got %q", out)
	}
}

func TestNewRejectsBadKey(t *testing.T) {
	if _, err := New("not-base64!"); err == nil {
		t.Fatal("expected error for invalid base64")
	}
	if _, err := New(base64.StdEncoding.EncodeToString([]byte("short"))); err == nil {
		t.Fatal("expected error for wrong key size")
	}
}
