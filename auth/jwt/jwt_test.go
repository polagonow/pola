package jwt_test

import (
	"testing"
	"time"

	jwtv5 "github.com/golang-jwt/jwt/v5"
	"github.com/polagonow/pola/auth/jwt"
)

// strong is a >=32-byte HS256 secret; Sign/Verify now reject weaker ones.
var strong = []byte("test-secret-key-0000000000000000000000")

func TestSignVerify_RoundTrip(t *testing.T) {
	secret := strong
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
	secretOne := []byte("secret-one-000000000000000000000000000")
	secretTwo := []byte("secret-two-000000000000000000000000000")
	tok, _ := jwt.Sign(map[string]any{"a": "b"}, secretOne, time.Hour)
	if _, err := jwt.Verify(tok, secretTwo); err == nil {
		t.Fatal("expected error for wrong secret")
	}
}

func TestVerify_Expired(t *testing.T) {
	// A negative expiry is now rejected at Sign time (no non-expiring or
	// pre-expired tokens are minted). Craft an already-expired token by hand.
	past := time.Now().Add(-time.Hour).Unix()
	tok := signRaw(t, strong, map[string]any{"a": "b", "exp": past, "iat": past})
	if _, err := jwt.Verify(tok, strong); err == nil {
		t.Fatal("expected error for expired token")
	}
}

// TestSign_RejectsWeakSecret and TestSign_RejectsNonPositiveExpiry cover the
// hardened Sign contract.
func TestSign_RejectsWeakSecret(t *testing.T) {
	if _, err := jwt.Sign(map[string]any{"a": "b"}, []byte("short"), time.Hour); err == nil {
		t.Fatal("expected error for weak secret")
	}
}

func TestSign_RejectsNonPositiveExpiry(t *testing.T) {
	if _, err := jwt.Sign(map[string]any{"a": "b"}, strong, 0); err == nil {
		t.Fatal("expected error for zero expiry")
	}
	if _, err := jwt.Sign(map[string]any{"a": "b"}, strong, -time.Hour); err == nil {
		t.Fatal("expected error for negative expiry")
	}
}

// TestVerify_MissingExp ensures tokens lacking an exp claim are rejected.
func TestVerify_MissingExp(t *testing.T) {
	tok := signRaw(t, strong, map[string]any{"a": "b", "iat": time.Now().Unix()})
	if _, err := jwt.Verify(tok, strong); err == nil {
		t.Fatal("expected error for token missing exp")
	}
}

func TestVerify_WeakSecret(t *testing.T) {
	if _, err := jwt.Verify("a.b.c", []byte("short")); err == nil {
		t.Fatal("expected error for weak secret")
	}
}

func TestVerify_Tampered(t *testing.T) {
	if _, err := jwt.Verify("not.a.jwt", strong); err == nil {
		t.Fatal("expected error for malformed token")
	}
}

// signRaw HS256-signs claims directly (bypassing jwt.Sign's guards) so tests can
// craft tokens jwt.Sign would refuse to mint (missing/expired exp).
func signRaw(t *testing.T, secret []byte, claims map[string]any) string {
	t.Helper()
	mc := jwtv5.MapClaims{}
	for k, v := range claims {
		mc[k] = v
	}
	tok, err := jwtv5.NewWithClaims(jwtv5.SigningMethodHS256, mc).SignedString(secret)
	if err != nil {
		t.Fatalf("signRaw: %v", err)
	}
	return tok
}
