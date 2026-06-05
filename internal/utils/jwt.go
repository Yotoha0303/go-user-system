package utils

import (
	"errors"
	"go-user-system/config"
	"log"
	"os"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var (
	jwtKey      []byte
	expireHours int
)

var (
	ErrJWTSecretNotFound    = errors.New("JWT_SECRET is not set")
	ErrJWTExpireHoursNotSet = errors.New("JWT_EXPIRE_HOURS is not set")
	ErrAccessTokenInvalid   = errors.New("invalid access token")

	ErrJWTSecretTooShort = errors.New("JWT_SECRET must be at least 32 characters")
)

const InitIssuer = "go-user-system"

func InitJWTKey(cfg *config.Config) error {

	key := strings.TrimSpace(os.Getenv("JWT_SECRET"))

	log.Printf("key: %v", key)

	if key == "" {
		return ErrJWTSecretNotFound
	}

	if len(key) < 32 {
		return ErrJWTSecretTooShort
	}

	if cfg.JWT.ExpireHours == 0 {
		return ErrJWTExpireHoursNotSet
	}

	expireHours = cfg.JWT.ExpireHours

	jwtKey = []byte(key)
	return nil
}

type UserClaims struct {
	UserID   int64  `json:"user_id"`
	Username string `json:"username"`
	jwt.RegisteredClaims
}

func GenerateToken(userID int64, username string) (string, error) {

	claims := UserClaims{
		UserID:   userID,
		Username: username,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Duration(expireHours) * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    InitIssuer,
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtKey)
}

func ParseToken(tokenString string) (*UserClaims, error) {
	token, err := jwt.ParseWithClaims(
		tokenString,
		&UserClaims{},
		func(token *jwt.Token) (interface{}, error) {
			return jwtKey, nil
		},
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
	)

	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(*UserClaims)
	if !ok || !token.Valid {
		return nil, ErrAccessTokenInvalid
	}

	return claims, nil
}
