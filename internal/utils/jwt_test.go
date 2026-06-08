package utils

import (
	"errors"
	"go-user-system/config"
	"testing"

	"github.com/golang-jwt/jwt/v5"
)

func TestInitJWTKeyRequiresSecret(t *testing.T) {
	t.Setenv("JWT_SECRET", "")

	err := InitJWTKey(&config.Config{
		JWT: config.JWTConfig{ExpireHours: 24},
	})

	if !errors.Is(err, ErrJWTSecretNotFound) {
		t.Fatalf("expected ErrJWTSecretNotFound, got %v", err)
	}
}

func TestInitJWTKeyRejectsShortSecret(t *testing.T) {
	t.Setenv("JWT_SECRET", "short")

	err := InitJWTKey(&config.Config{
		JWT: config.JWTConfig{ExpireHours: 24},
	})

	if !errors.Is(err, ErrJWTSecretTooShort) {
		t.Fatalf("expected ErrJWTSecretTooShort, got %v", err)
	}
}

func TestInitJWTKeyRequiresExpireHours(t *testing.T) {
	t.Setenv("JWT_SECRET", "test_jwt_secret_with_32_plus_chars")

	err := InitJWTKey(&config.Config{
		JWT: config.JWTConfig{ExpireHours: 0},
	})

	if !errors.Is(err, ErrJWTExpireHoursNotSet) {
		t.Fatalf("expected ErrJWTExpireHoursNotSet, got %v", err)
	}
}

func TestGenerateAndParseToken(t *testing.T) {
	t.Setenv("JWT_SECRET", "test_jwt_secret_with_32_plus_chars")

	err := InitJWTKey(&config.Config{
		JWT: config.JWTConfig{ExpireHours: 24},
	})
	if err != nil {
		t.Fatalf("init jwt key failed: %v", err)
	}

	token, err := GenerateToken(1, "alice")
	if err != nil {
		t.Fatalf("generate token failed: %v", err)
	}

	claims, err := ParseToken(token)
	if err != nil {
		t.Fatalf("parse token failed: %v", err)
	}
	if claims.UserID != 1 {
		t.Fatalf("expected user id 1, got %d", claims.UserID)
	}
	if claims.Username != "alice" {
		t.Fatalf("expected username alice, got %s", claims.Username)
	}
	if claims.Issuer != InitIssuer {
		t.Fatalf("expected issuer %s, got %s", InitIssuer, claims.Issuer)
	}
}

func TestParseTokenRejectsMalformedToken(t *testing.T) {
	t.Setenv("JWT_SECRET", "test_jwt_secret_with_32_plus_chars")

	err := InitJWTKey(&config.Config{
		JWT: config.JWTConfig{ExpireHours: 24},
	})
	if err != nil {
		t.Fatalf("init jwt key failed: %v", err)
	}

	_, err = ParseToken("malformed.token")

	if err == nil {
		t.Fatal("expected malformed token error")
	}
}

func TestParseTokenRejectsInvalidSigningMethod(t *testing.T) {
	t.Setenv("JWT_SECRET", "test_jwt_secret_with_32_plus_chars")

	err := InitJWTKey(&config.Config{
		JWT: config.JWTConfig{ExpireHours: 24},
	})
	if err != nil {
		t.Fatalf("init jwt key failed: %v", err)
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS512, UserClaims{
		UserID:   1,
		Username: "alice",
	})
	tokenString, err := token.SignedString(jwtKey)
	if err != nil {
		t.Fatalf("sign token failed: %v", err)
	}

	_, err = ParseToken(tokenString)

	if err == nil {
		t.Fatal("expected invalid signing method error")
	}
}

func TestClaimsFromTokenRejectsUnexpectedClaimsType(t *testing.T) {
	_, err := claimsFromToken(&jwt.Token{
		Claims: jwt.MapClaims{},
		Valid:  true,
	})

	if !errors.Is(err, ErrAccessTokenInvalid) {
		t.Fatalf("expected ErrAccessTokenInvalid, got %v", err)
	}
}

func TestClaimsFromTokenRejectsInvalidToken(t *testing.T) {
	_, err := claimsFromToken(&jwt.Token{
		Claims: &UserClaims{UserID: 1, Username: "alice"},
		Valid:  false,
	})

	if !errors.Is(err, ErrAccessTokenInvalid) {
		t.Fatalf("expected ErrAccessTokenInvalid, got %v", err)
	}
}
