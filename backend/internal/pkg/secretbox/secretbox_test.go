package secretbox

import "testing"

func TestEncryptDecryptRoundTrip(t *testing.T) {
	key := "0123456789abcdef0123456789abcdef"
	box, err := New(key)
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}
	ciphertext, err := box.Encrypt("sk-test")
	if err != nil {
		t.Fatalf("Encrypt returned error: %v", err)
	}
	if ciphertext == "sk-test" {
		t.Fatalf("ciphertext should not equal plaintext")
	}
	plaintext, err := box.Decrypt(ciphertext)
	if err != nil {
		t.Fatalf("Decrypt returned error: %v", err)
	}
	if plaintext != "sk-test" {
		t.Fatalf("plaintext = %q, want sk-test", plaintext)
	}
}

func TestNewRejectsShortKey(t *testing.T) {
	if _, err := New("short"); err == nil {
		t.Fatalf("New accepted short key")
	}
}

func TestMaskKeepsOnlyPrefixAndSuffix(t *testing.T) {
	masked := Mask("sk-1234567890abcdef")
	if masked != "sk-1****cdef" {
		t.Fatalf("masked = %q, want sk-1****cdef", masked)
	}
}
