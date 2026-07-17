// Package jwt provides HS256 JSON Web Token signing and verification — the
// stateless primitive behind pola's middleware/jwt cookie sessions. It mirrors
// auth/password: pure functions with no request/response coupling.
package jwt

import (
	"errors"
	"time"

	jwtv5 "github.com/golang-jwt/jwt/v5"
)

// ErrInvalidToken is returned when a token fails signature or claim validation.
var ErrInvalidToken = errors.New("jwt: invalid token")

// Sign returns an HS256-signed token embedding the given claims plus the
// standard "iat" and "exp" (now + expiry) registered claims. A zero expiry omits
// "exp" (a non-expiring token); a negative expiry yields an already-expired one.
func Sign(claims map[string]any, secret []byte, expiry time.Duration) (string, error) {
	mc := jwtv5.MapClaims{}
	for k, v := range claims {
		mc[k] = v
	}
	now := time.Now()
	mc["iat"] = now.Unix()
	if expiry != 0 {
		mc["exp"] = now.Add(expiry).Unix()
	}
	return jwtv5.NewWithClaims(jwtv5.SigningMethodHS256, mc).SignedString(secret)
}

// Verify parses and validates an HS256 token (signature + expiry) and returns
// its claims as a plain map. It rejects any token not signed with HS256.
func Verify(token string, secret []byte) (map[string]any, error) {
	parsed, err := jwtv5.Parse(token, func(t *jwtv5.Token) (any, error) {
		if _, ok := t.Method.(*jwtv5.SigningMethodHMAC); !ok {
			return nil, ErrInvalidToken
		}
		return secret, nil
	}, jwtv5.WithValidMethods([]string{"HS256"}))
	if err != nil || !parsed.Valid {
		return nil, ErrInvalidToken
	}
	mc, ok := parsed.Claims.(jwtv5.MapClaims)
	if !ok {
		return nil, ErrInvalidToken
	}
	out := make(map[string]any, len(mc))
	for k, v := range mc {
		out[k] = v
	}
	return out, nil
}
