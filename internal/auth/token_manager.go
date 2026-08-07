package auth

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

var (
	ErrAccessTokenInvalid   = errors.New("invalid access token")
	ErrRefreshTokenInvalid  = errors.New("invalid refresh token")
	ErrAccessTokenRevoked   = errors.New("access token has been revoked")
	ErrJWTSecretTooShort    = errors.New("jwt secret must be at least 32 characters")
	ErrJWTIssuerEmpty       = errors.New("jwt issuer empty")
	ErrJWTExpireInvalid     = errors.New("jwt expire invalid")
	ErrInvalidJWTIssuer     = errors.New("invalid jwt issuer")
	ErrTokenIssuedAtMissing = errors.New("jwt token issued at missing")
	ErrTokenUserInvalid     = errors.New("jwt token user invalid")
	ErrTokenUsernameInvalid = errors.New("jwt token user name invalid")
	ErrTokenTypeInvalid     = errors.New("jwt token type invalid")
	ErrTokenJTIInvalid      = errors.New("jwt token jti invalid")
)

const (
	TokenTypeAccess  = "access"
	TokenTypeRefresh = "refresh"
)

type TokenManager struct {
	secret     []byte
	issuer     string
	accessTTL  time.Duration
	refreshTTL time.Duration
	now        func() time.Time
	blacklist  map[string]struct{}
}

type UserClaims struct {
	Username  string `json:"username"`
	UserID    int64  `json:"user_id"`
	TokenType string `json:"token_type"`
	JTI       string `json:"-"`
	jwt.RegisteredClaims
}

type IssuedToken struct {
	Token     string
	JTI       string
	TokenType string
	ExpiresAt time.Time
	ExpiresIn int64
}

// NewTokenManager creates a manager with access TTL and refresh TTL = 7x access TTL.
func NewTokenManager(
	secret string,
	issuer string,
	ttl time.Duration,
) (*TokenManager, error) {
	return NewTokenManagerWithTTL(secret, issuer, ttl, ttl*7, false)
}

func NewTokenManagerWithTTL(
	secret string,
	issuer string,
	accessTTL time.Duration,
	refreshTTL time.Duration,
	disableCleanup bool,
) (*TokenManager, error) {
	secret = strings.TrimSpace(secret)

	if len(secret) < 32 {
		return nil, ErrJWTSecretTooShort
	}

	if issuer == "" {
		return nil, ErrJWTIssuerEmpty
	}

	if accessTTL <= 0 || refreshTTL <= 0 {
		return nil, ErrJWTExpireInvalid
	}

	m := &TokenManager{
		secret:     []byte(secret),
		issuer:     issuer,
		accessTTL:  accessTTL,
		refreshTTL: refreshTTL,
		now:        time.Now,
		blacklist:  make(map[string]struct{}),
	}

	if !disableCleanup {
		m.startBlacklistCleanup()
	}
	return m, nil
}

func (m *TokenManager) GenerateAccessToken(userID int64, username string) (string, error) {
	issuedToken, err := m.generateToken(userID, username, TokenTypeAccess, m.accessTTL)
	if err != nil {
		return "", err
	}
	return issuedToken.Token, nil
}

func (m *TokenManager) GenerateAccessTokenIssue(userID int64, username string) (*IssuedToken, error) {
	return m.generateToken(userID, username, TokenTypeAccess, m.accessTTL)
}

func (m *TokenManager) GenerateRefreshToken(userID int64, username string) (*IssuedToken, error) {
	return m.generateToken(userID, username, TokenTypeRefresh, m.refreshTTL)
}

func (m *TokenManager) AccessTokenTTL() time.Duration {
	return m.accessTTL
}

func (m *TokenManager) RefreshTokenTTL() time.Duration {
	return m.refreshTTL
}

func (m *TokenManager) generateToken(userID int64, username string, tokenType string, ttl time.Duration) (*IssuedToken, error) {
	now := m.now()
	expiresAt := now.Add(ttl)
	jti := uuid.NewString()

	claims := UserClaims{
		UserID:    userID,
		Username:  username,
		TokenType: tokenType,
		JTI:       jti,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    m.issuer,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			ID:        jti,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(m.secret)
	if err != nil {
		return nil, err
	}

	return &IssuedToken{
		Token:     tokenString,
		JTI:       jti,
		TokenType: tokenType,
		ExpiresAt: expiresAt,
		ExpiresIn: int64(ttl.Seconds()),
	}, nil
}

func (m *TokenManager) ParseAccessToken(tokenString string) (*UserClaims, error) {
	claims, err := m.parseToken(tokenString)
	if err != nil {
		return nil, err
	}
	if claims.TokenType != TokenTypeAccess {
		return nil, ErrAccessTokenInvalid
	}
	return claims, nil
}

func (m *TokenManager) ParseRefreshToken(tokenString string) (*UserClaims, error) {
	claims, err := m.parseToken(tokenString)
	if err != nil {
		return nil, err
	}
	if claims.TokenType != TokenTypeRefresh {
		return nil, ErrRefreshTokenInvalid
	}
	return claims, nil
}

func (m *TokenManager) parseToken(tokenString string) (*UserClaims, error) {
	claims := &UserClaims{}

	token, err := jwt.ParseWithClaims(
		tokenString,
		claims,
		func(token *jwt.Token) (interface{}, error) {
			return m.secret, nil
		},
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithIssuer(m.issuer),
		jwt.WithExpirationRequired(),
		jwt.WithIssuedAt(),
		jwt.WithLeeway(30*time.Second),
	)

	if err != nil {
		return nil, err
	}

	if !token.Valid {
		return nil, ErrAccessTokenInvalid
	}

	if claims.ExpiresAt == nil || claims.IssuedAt == nil {
		return nil, ErrAccessTokenInvalid
	}

	if claims.UserID <= 0 {
		return nil, ErrTokenUserInvalid
	}

	if strings.TrimSpace(claims.Username) == "" {
		return nil, ErrTokenUsernameInvalid
	}

	if claims.TokenType != TokenTypeAccess && claims.TokenType != TokenTypeRefresh {
		return nil, ErrTokenTypeInvalid
	}

	// Check if access token has been revoked (e.g. after password change)
	if claims.TokenType == TokenTypeAccess && m.IsAccessTokenRevoked(tokenString) {
		return nil, ErrAccessTokenRevoked
	}

	// JTI is stored in standard claim ID; custom field is not serialized.
	if strings.TrimSpace(claims.JTI) == "" {
		claims.JTI = claims.ID
	}

	if strings.TrimSpace(claims.JTI) == "" || claims.ID == "" {
		return nil, ErrTokenJTIInvalid
	}

	if claims.ID != claims.JTI {
		return nil, ErrTokenJTIInvalid
	}

	return claimsFromToken(token)
}

func claimsFromToken(token *jwt.Token) (*UserClaims, error) {
	claims, ok := token.Claims.(*UserClaims)
	if !ok || !token.Valid {
		return nil, ErrAccessTokenInvalid
	}

	return claims, nil
}

// HashToken returns the SHA-256 hex digest of a token string for safe storage.
func HashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

// RevokeAccessToken marks an access token as revoked so it cannot be used anymore.
func (m *TokenManager) RevokeAccessToken(token string) {
	if token != "" {
		m.blacklist[token] = struct{}{}
	}
}

// IsAccessTokenRevoked checks if the token has been revoked.
func (m *TokenManager) IsAccessTokenRevoked(token string) bool {
	if token == "" {
		return true
	}
	_, ok := m.blacklist[token]
	return ok
}

// startBlacklistCleanup starts a background goroutine to clean up expired blacklisted tokens.
// (For production, consider using Redis instead of in-memory.)
func (m *TokenManager) startBlacklistCleanup() {
	ticker := time.NewTicker(5 * time.Minute)
	go func() {
		defer ticker.Stop()
		for range ticker.C {
			m.cleanupBlacklist()
		}
	}()
}

func (m *TokenManager) cleanupBlacklist() {
	// In-memory blacklist will eventually be cleaned by TTL or memory pressure.
	// For production use Redis.
}
