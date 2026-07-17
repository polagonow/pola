package jwt_test

import (
	"testing"
	"time"

	"github.com/polagonow/pola/auth/jwt"
)

func TestSignVerify_RoundTrip(t *testing.T) {
	secret := []byte("test-secret-key")
	tok, err := jwt.Sign(map[string]any{"user": map[string]any{"id": float64(42)}}, secret, time.Hour)
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	claims, err := jwt.Verify(tok, secret)
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	user, ok := claims["user"].(map[string]any)
	if !ok || user["id"] != float64(42) {
		t.Fatalf("expected user.id=42, got %#v", claims["user"])
	}
	if _, ok := claims["exp"]; !ok {
		t.Fatal("expected exp claim")
	}
}

func TestVerify_WrongSecret(t *testing.T) {
	tok, _ := jwt.Sign(map[string]any{"a": "b"}, []byte("secret-one"), time.Hour)
	if _, err := jwt.Verify(tok, []byte("secret-two")); err == nil {
		t.Fatal("expected error for wrong secret")
	}
}

func TestVerify_Expired(t *testing.T) {
	tok, _ := jwt.Sign(map[string]any{"a": "b"}, []byte("secret"), -time.Hour)
	if _, err := jwt.Verify(tok, []byte("secret")); err == nil {
		t.Fatal("expected error for expired token")
	}
}

func TestVerify_Tampered(t *testing.T) {
	if _, err := jwt.Verify("not.a.jwt", []byte("secret")); err == nil {
		t.Fatal("expected error for malformed token")
	}
}
