package service

import (
	"testing"
)

func TestHashService_Password(t *testing.T) {
	hasher := NewHashService()
	password := "SecretPassw0rd!"

	hash, err := hasher.HashPassword(password)
	if err != nil {
		t.Fatalf("unexpected error hashing password: %v", err)
	}

	if hash == password {
		t.Fatalf("hashed password should not equal plaintext")
	}

	if err := hasher.ComparePassword(hash, password); err != nil {
		t.Fatalf("expected password match, got error: %v", err)
	}

	if err := hasher.ComparePassword(hash, "WrongPassword"); err == nil {
		t.Fatalf("expected error comparing incorrect password, got nil")
	}
}

func TestHashService_HashToken(t *testing.T) {
	hasher := NewHashService()
	token := "some-random-plain-token-12345"

	hash1 := hasher.HashToken(token)
	hash2 := hasher.HashToken(token)

	if hash1 != hash2 {
		t.Fatalf("expected deterministic sha256 output, got %s != %s", hash1, hash2)
	}

	if hash1 == token {
		t.Fatalf("token hash should not equal plain token")
	}

	if len(hash1) != 64 { // SHA-256 hex encoded is 64 characters
		t.Fatalf("expected hash length 64, got %d", len(hash1))
	}
}
