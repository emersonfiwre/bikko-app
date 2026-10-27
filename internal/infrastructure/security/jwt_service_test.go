package security

import (
	"github.com/golang-jwt/jwt/v5"
	"testing"
	"time"
)

func TestJWTRequiresExpiryIdentityAndExactAlgorithm(t *testing.T) {
	secret := "test-secret"
	service := NewJWTService(secret)
	for _, tc := range []struct {
		name   string
		method jwt.SigningMethod
		claims jwt.MapClaims
	}{{"missing expiry", jwt.SigningMethodHS256, jwt.MapClaims{"user_id": "lucas"}}, {"empty identity", jwt.SigningMethodHS256, jwt.MapClaims{"user_id": "", "exp": time.Now().Add(time.Hour).Unix()}}, {"other HMAC algorithm", jwt.SigningMethodHS512, jwt.MapClaims{"user_id": "lucas", "exp": time.Now().Add(time.Hour).Unix()}}, {"expired", jwt.SigningMethodHS256, jwt.MapClaims{"user_id": "lucas", "exp": time.Now().Add(-time.Hour).Unix()}}} {
		t.Run(tc.name, func(t *testing.T) {
			token, err := jwt.NewWithClaims(tc.method, tc.claims).SignedString([]byte(secret))
			if err != nil {
				t.Fatal(err)
			}
			if _, err := service.ValidateToken(token); err == nil {
				t.Fatal("accepted invalid session")
			}
		})
	}
	token, err := service.GenerateToken("lucas")
	if err != nil {
		t.Fatal(err)
	}
	id, err := service.ValidateToken(token)
	if err != nil || id != "lucas" {
		t.Fatal(id, err)
	}
	if _, err := service.GenerateToken(""); err == nil {
		t.Fatal("generated anonymous session")
	}
}
