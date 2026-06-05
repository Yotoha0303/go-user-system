package utils

import (
	"errors"
	"go-user-system/config"
	"testing"
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
