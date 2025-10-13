package service

import (
	"errors"
	"fmt"
	"time"

	"rip/internal/app/models"
	"rip/internal/app/repository"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

// JWTService предоставляет методы для работы с JWT токенами
type JWTService struct {
	config *models.JWTConfig
}

// NewJWTService создает новый JWT сервис
func NewJWTService(secretKey string, expiresIn, refreshExpiresIn time.Duration) *JWTService {
	return &JWTService{
		config: &models.JWTConfig{
			SecretKey:        secretKey,
			ExpiresIn:        expiresIn,
			RefreshExpiresIn: refreshExpiresIn,
		},
	}
}

// GenerateTokens генерирует пару токенов (access + refresh)
func (j *JWTService) GenerateTokens(user *repository.User) (*models.TokenResponse, error) {
	// Создаем access token
	accessClaims := &models.JWTClaims{
		UserID:   user.ID,
		Username: user.Username,
		Role:     models.Role(user.Role),
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(j.config.ExpiresIn)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Issuer:    "rip-api",
			Subject:   fmt.Sprintf("%d", user.ID),
		},
	}

	accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims)
	accessTokenString, err := accessToken.SignedString([]byte(j.config.SecretKey))
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}

	// Создаем refresh token
	refreshClaims := &models.JWTClaims{
		UserID:   user.ID,
		Username: user.Username,
		Role:     models.Role(user.Role),
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(j.config.RefreshExpiresIn)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Issuer:    "rip-api",
			Subject:   fmt.Sprintf("%d", user.ID),
		},
	}

	refreshToken := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims)
	refreshTokenString, err := refreshToken.SignedString([]byte(j.config.SecretKey))
	if err != nil {
		return nil, fmt.Errorf("failed to generate refresh token: %w", err)
	}

	return &models.TokenResponse{
		AccessToken:  accessTokenString,
		RefreshToken: refreshTokenString,
		TokenType:    "Bearer",
		ExpiresIn:    int64(j.config.ExpiresIn.Seconds()),
	}, nil
}

// ValidateToken валидирует JWT токен
func (j *JWTService) ValidateToken(tokenString string) (*models.JWTClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &models.JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(j.config.SecretKey), nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	claims, ok := token.Claims.(*models.JWTClaims)
	if !ok || !token.Valid {
		return nil, errors.New("invalid token")
	}

	return claims, nil
}

// HashPassword хеширует пароль
func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}

// CheckPasswordHash проверяет пароль
func CheckPasswordHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}
