package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"rip/internal/app/models"
	"rip/internal/app/redis"
	"rip/internal/app/service"
)

// AuthMiddleware представляет middleware для аутентификации
type AuthMiddleware struct {
	jwtService  *service.JWTService
	redisClient *redis.Client
}

// NewAuthMiddleware создает новый middleware аутентификации
func NewAuthMiddleware(jwtService *service.JWTService, redisClient *redis.Client) *AuthMiddleware {
	return &AuthMiddleware{
		jwtService:  jwtService,
		redisClient: redisClient,
	}
}

// RequireAuth требует аутентификации
func (m *AuthMiddleware) RequireAuth(next func(http.ResponseWriter, *http.Request)) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		// Извлекаем токен из заголовка Authorization
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			m.writeError(w, http.StatusUnauthorized, "Authorization header required")
			return
		}

		// Проверяем формат Bearer token
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			m.writeError(w, http.StatusUnauthorized, "Invalid authorization header format")
			return
		}

		token := parts[1]

		// Проверяем, не находится ли токен в черном списке
		isBlacklisted, err := m.redisClient.IsInBlacklist(r.Context(), token)
		if err != nil {
			m.writeError(w, http.StatusInternalServerError, "Failed to check token blacklist")
			return
		}
		if isBlacklisted {
			m.writeError(w, http.StatusUnauthorized, "Token has been revoked")
			return
		}

		// Валидируем токен
		claims, err := m.jwtService.ValidateToken(token)
		if err != nil {
			m.writeError(w, http.StatusUnauthorized, "Invalid token")
			return
		}

		// Добавляем информацию о пользователе в контекст
		ctx := context.WithValue(r.Context(), "user_id", claims.UserID)
		ctx = context.WithValue(ctx, "username", claims.Username)
		ctx = context.WithValue(ctx, "role", claims.Role)
		ctx = context.WithValue(ctx, "token", token)

		next(w, r.WithContext(ctx))
	}
}

// RequireRole требует определенную роль
func (m *AuthMiddleware) RequireRole(requiredRole models.Role) func(func(http.ResponseWriter, *http.Request)) func(http.ResponseWriter, *http.Request) {
	return func(next func(http.ResponseWriter, *http.Request)) func(http.ResponseWriter, *http.Request) {
		return func(w http.ResponseWriter, r *http.Request) {
			role, ok := r.Context().Value("role").(models.Role)
			if !ok {
				m.writeError(w, http.StatusUnauthorized, "User role not found")
				return
			}

			if !role.HasPermission(requiredRole) {
				m.writeError(w, http.StatusForbidden, "Insufficient permissions")
				return
			}

			next(w, r)
		}
	}
}

// OptionalAuth не требует аутентификации, но добавляет информацию о пользователе если токен валиден
func (m *AuthMiddleware) OptionalAuth(next func(http.ResponseWriter, *http.Request)) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader != "" {
			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) == 2 && parts[0] == "Bearer" {
				token := parts[1]

				// Проверяем черный список
				isBlacklisted, err := m.redisClient.IsInBlacklist(r.Context(), token)
				if err == nil && !isBlacklisted {
					// Валидируем токен
					claims, err := m.jwtService.ValidateToken(token)
					if err == nil {
						// Добавляем информацию о пользователе в контекст
						ctx := context.WithValue(r.Context(), "user_id", claims.UserID)
						ctx = context.WithValue(ctx, "username", claims.Username)
						ctx = context.WithValue(ctx, "role", claims.Role)
						ctx = context.WithValue(ctx, "token", token)
						r = r.WithContext(ctx)
					}
				}
			}
		}

		next(w, r)
	}
}

// writeError записывает ошибку в ответ
func (m *AuthMiddleware) writeError(w http.ResponseWriter, statusCode int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(map[string]string{"error": message})
}

// GetUserIDFromContext извлекает ID пользователя из контекста
func GetUserIDFromContext(ctx context.Context) (int, bool) {
	userID, ok := ctx.Value("user_id").(int)
	return userID, ok
}

// GetRoleFromContext извлекает роль пользователя из контекста
func GetRoleFromContext(ctx context.Context) (models.Role, bool) {
	role, ok := ctx.Value("role").(models.Role)
	return role, ok
}

// GetTokenFromContext извлекает токен из контекста
func GetTokenFromContext(ctx context.Context) (string, bool) {
	token, ok := ctx.Value("token").(string)
	return token, ok
}
