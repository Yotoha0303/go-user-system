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
	ErrJWTSecretTooShort    = errors.New("jwt secret must be at least 32 characters")
	ErrJWTIssuerEmpty       = errors.New("jwt issuer empty")
	ErrJWTExpireInvalid     = errors.New("jwt expire invalid")
	ErrInvalidJWTIssuer     = errors.New("invalid jwt issuer")
	ErrTokenIssuedAtMissing = errors.New("jwt token issued at missing")
	ErrTokenUserInvalid     = errors.New("jwt token user invalid")
	ErrTokenUsernameInvalid = errors.New("jwt token user name invalid")
	ErrTokenTypeInvalid     = errors.New("jwt token type invalid")
	ErrTokenJTIInvalid      = errors.New("jwt token jti invalid")
	ErrTokenVersionInvalid  = errors.New("jwt token auth version invalid")
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
}

type UserClaims struct {
	Username    string `json:"username"`
	UserID      int64  `json:"user_id"`
	TokenType   string `json:"token_type"`
	AuthVersion int64  `json:"auth_version"`
	JTI         string `json:"-"`
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
	return NewTokenManagerWithTTL(secret, issuer, ttl, ttl*7)
}

func NewTokenManagerWithTTL(
	secret string,
	issuer string,
	accessTTL time.Duration,
	refreshTTL time.Duration,
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
	}
	return m, nil
}

func (m *TokenManager) GenerateAccessToken(userID int64, username string, authVersion int64) (string, error) {
	issuedToken, err := m.generateToken(userID, username, authVersion, TokenTypeAccess, m.accessTTL)
	if err != nil {
		return "", err
	}
	return issuedToken.Token, nil
}

func (m *TokenManager) GenerateAccessTokenIssue(userID int64, username string, authVersion int64) (*IssuedToken, error) {
	return m.generateToken(userID, username, authVersion, TokenTypeAccess, m.accessTTL)
}

func (m *TokenManager) GenerateRefreshToken(userID int64, username string, authVersion int64) (*IssuedToken, error) {
	return m.generateToken(userID, username, authVersion, TokenTypeRefresh, m.refreshTTL)
}

func (m *TokenManager) AccessTokenTTL() time.Duration {
	return m.accessTTL
}

func (m *TokenManager) RefreshTokenTTL() time.Duration {
	return m.refreshTTL
}

func (m *TokenManager) generateToken(userID int64, username string, authVersion int64, tokenType string, ttl time.Duration) (*IssuedToken, error) {
	if authVersion <= 0 {
		return nil, ErrTokenVersionInvalid
	}
	now := m.now()
	expiresAt := now.Add(ttl)
	jti := uuid.NewString()

	claims := UserClaims{
		UserID:      userID,
		Username:    username,
		TokenType:   tokenType,
		AuthVersion: authVersion,
		JTI:         jti,
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

	if claims.AuthVersion <= 0 {
		return nil, ErrTokenVersionInvalid
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
