package auth

import (
	"errors"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var (
	ErrAccessTokenInvalid   = errors.New("invalid access token")
	ErrJWTSecretTooShort    = errors.New("jwt secret must be at least 32 characters")
	ErrJWTIssuerEmpty       = errors.New("jwt issuer empty")
	ErrJWTExpireInvalid     = errors.New("jwt expire invalid")
	ErrInvalidJWTIssuer     = errors.New("invalid jwt issuer")
	ErrTokenIssuedAtMissing = errors.New("jwt token issued at missing")
	ErrTokenUserInvalid     = errors.New("jwt token user invalid")
	ErrTokenUsernameInvalid = errors.New("jwt token user name invalid")
)

type TokenManager struct {
	secret []byte
	issuer string
	ttl    time.Duration
	now    func() time.Time
}

type UserClaims struct {
	Username string `json:"username"`
	UserID   int64  `json:"user_id"`
	jwt.RegisteredClaims
}

func NewTokenManager(
	secret string,
	issuer string,
	ttl time.Duration,
) (*TokenManager, error) {
	secret = strings.TrimSpace(secret)

	if len(secret) < 32 {
		return nil, ErrJWTSecretTooShort
	}

	if issuer == "" {
		return nil, ErrJWTIssuerEmpty
	}

	if ttl <= 0 {
		return nil, ErrJWTExpireInvalid
	}

	return &TokenManager{
		secret: []byte(secret),
		issuer: issuer,
		ttl:    ttl,
		now:    time.Now,
	}, nil
}

func (m *TokenManager) GenerateAccessToken(
	userID int64,
	username string,
) (string, error) {

	claims := UserClaims{
		UserID:   userID,
		Username: username,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    m.issuer,
			IssuedAt:  jwt.NewNumericDate(m.now()),
			ExpiresAt: jwt.NewNumericDate(m.now().Add(m.ttl)),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(m.secret)
}

func (m *TokenManager) ParseAccessToken(tokenString string) (*UserClaims, error) {
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

	return claimsFromToken(token)
}

func claimsFromToken(token *jwt.Token) (*UserClaims, error) {
	claims, ok := token.Claims.(*UserClaims)
	if !ok || !token.Valid {
		return nil, ErrAccessTokenInvalid
	}

	return claims, nil
}
