package models

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// JWTClaims содержит данные JWT токена
type JWTClaims struct {
	UserID   int    `json:"user_id"`
	Username string `json:"username"`
	Role     Role   `json:"role"`
	jwt.RegisteredClaims
}

// JWTConfig содержит конфигурацию JWT
type JWTConfig struct {
	SecretKey        string
	ExpiresIn        time.Duration
	RefreshExpiresIn time.Duration
}

// SessionData содержит данные сессии в Redis
type SessionData struct {
	UserID    int       `json:"user_id"`
	Username  string    `json:"username"`
	Role      Role      `json:"role"`
	ExpiresAt time.Time `json:"expires_at"`
}

// TokenResponse содержит ответ с токенами
type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int64  `json:"expires_in"`
}
