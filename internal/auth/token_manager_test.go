package auth

import (
	"errors"
	"testing"
	"time"
)

func newTestTokenManager(t *testing.T) *TokenManager {
	t.Helper()

	manager, err := NewTokenManagerWithTTL(
		"auth_test_jwt_secret_32_chars_long",
		"go-user-system-test",
		time.Minute,
		time.Hour,
	)
	if err != nil {
		t.Fatalf("new token manager failed: %v", err)
	}
	return manager
}

func TestTokenManagerIssuesAndParsesAccessToken(t *testing.T) {
	manager := newTestTokenManager(t)

	token, err := manager.GenerateAccessToken(7, "alice", 3)
	if err != nil {
		t.Fatalf("generate access token failed: %v", err)
	}

	claims, err := manager.ParseAccessToken(token)
	if err != nil {
		t.Fatalf("parse access token failed: %v", err)
	}

	if claims.UserID != 7 || claims.Username != "alice" {
		t.Fatalf("unexpected claims: userID=%d username=%s", claims.UserID, claims.Username)
	}
	if claims.TokenType != TokenTypeAccess {
		t.Fatalf("expected access token type, got %s", claims.TokenType)
	}
	if claims.JTI == "" {
		t.Fatal("expected jti to be set")
	}
	if claims.AuthVersion != 3 {
		t.Fatalf("expected auth version 3, got %d", claims.AuthVersion)
	}
}

func TestTokenManagerRejectsRefreshTokenAsAccessToken(t *testing.T) {
	manager := newTestTokenManager(t)

	refreshToken, err := manager.GenerateRefreshToken(7, "alice", 1)
	if err != nil {
		t.Fatalf("generate refresh token failed: %v", err)
	}

	_, err = manager.ParseAccessToken(refreshToken.Token)
	if !errors.Is(err, ErrAccessTokenInvalid) {
		t.Fatalf("expected ErrAccessTokenInvalid, got %v", err)
	}
}

func TestTokenManagerRejectsAccessTokenAsRefreshToken(t *testing.T) {
	manager := newTestTokenManager(t)

	accessToken, err := manager.GenerateAccessToken(7, "alice", 1)
	if err != nil {
		t.Fatalf("generate access token failed: %v", err)
	}

	_, err = manager.ParseRefreshToken(accessToken)
	if !errors.Is(err, ErrRefreshTokenInvalid) {
		t.Fatalf("expected ErrRefreshTokenInvalid, got %v", err)
	}
}

func TestTokenManagerRejectsInvalidAuthVersion(t *testing.T) {
	manager := newTestTokenManager(t)

	_, err := manager.GenerateAccessToken(7, "alice", 0)
	if !errors.Is(err, ErrTokenVersionInvalid) {
		t.Fatalf("expected ErrTokenVersionInvalid, got %v", err)
	}
}

func TestHashTokenIsStableAndNotPlainText(t *testing.T) {
	token := "refresh-token-value"

	first := HashToken(token)
	second := HashToken(token)

	if first != second {
		t.Fatal("expected stable hash")
	}
	if first == token {
		t.Fatal("expected hashed value not to equal plain token")
	}
	if len(first) != 64 {
		t.Fatalf("expected sha256 hex length 64, got %d", len(first))
	}
}
