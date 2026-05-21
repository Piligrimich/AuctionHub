package service

import (
	"AuthService/internal/domain"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type tokenService struct {
	secret     []byte
	accessTTL  time.Duration
	refreshTTL time.Duration
}

func NewTokenService(secret string, accessTTL, refreshTTL time.Duration) *tokenService {
	return &tokenService{
		secret:     []byte(secret),
		accessTTL:  accessTTL,
		refreshTTL: refreshTTL,
	}
}

type jwtClaims struct {
	UserID string `json:"user_id"`
	Email  string `json:"email"`
	jwt.RegisteredClaims
}

func (s *tokenService) GenerateAccessToken(userID, email string) (string, error) {
	claims := jwtClaims{
		UserID: userID,
		Email:  email,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(s.accessTTL)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "auth-service",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(s.secret)
	if err != nil {
		return "", fmt.Errorf("sign access token: %w", err)
	}
	return signed, nil
}

func (s *tokenService) ValidateAccessToken(tokenStr string) (*domain.Claims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &jwtClaims{},
		func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return s.secret, nil
		})

	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, domain.ErrTokenExpired
		}
		return nil, domain.ErrInvalidToken
	}
	claims, ok := token.Claims.(*jwtClaims)
	if !ok || !token.Valid {
		return nil, domain.ErrInvalidToken
	}
	return &domain.Claims{
		UserID: claims.UserID,
		Email:  claims.Email,
	}, nil
}

func (s *tokenService) GenerateRefreshToken() (tokenStr, tokenHash string, expiresAt time.Time, err error) {
	b := make([]byte, 32)
	if _, err = rand.Read(b); err != nil {
		return "", "", time.Time{}, fmt.Errorf("generate refresh token: %w", err)
	}

	tokenStr = base64.URLEncoding.EncodeToString(b)
	hash := sha256.Sum256([]byte(tokenStr))
	tokenHash = hex.EncodeToString(hash[:])
	expiresAt = time.Now().Add(s.refreshTTL)
	return
}

func HashToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}
