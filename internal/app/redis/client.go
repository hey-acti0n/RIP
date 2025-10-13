package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"rip/internal/app/models"

	"github.com/redis/go-redis/v9"
)

const (
	// Префиксы для ключей Redis
	sessionPrefix   = "session:"
	blacklistPrefix = "blacklist:"
)

// Client представляет клиент Redis
type Client struct {
	rdb *redis.Client
}

// NewClient создает новый клиент Redis
func NewClient(addr, password string, db int) (*Client, error) {
	rdb := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})

	// Проверяем соединение
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	return &Client{rdb: rdb}, nil
}

// Close закрывает соединение с Redis
func (c *Client) Close() error {
	return c.rdb.Close()
}

// SetSession сохраняет данные сессии
func (c *Client) SetSession(ctx context.Context, sessionID string, data *models.SessionData, expiration time.Duration) error {
	key := sessionPrefix + sessionID
	dataJSON, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal session data: %w", err)
	}

	return c.rdb.Set(ctx, key, dataJSON, expiration).Err()
}

// GetSession получает данные сессии
func (c *Client) GetSession(ctx context.Context, sessionID string) (*models.SessionData, error) {
	key := sessionPrefix + sessionID
	dataJSON, err := c.rdb.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, fmt.Errorf("session not found")
		}
		return nil, fmt.Errorf("failed to get session: %w", err)
	}

	var data models.SessionData
	if err := json.Unmarshal([]byte(dataJSON), &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal session data: %w", err)
	}

	return &data, nil
}

// DeleteSession удаляет сессию
func (c *Client) DeleteSession(ctx context.Context, sessionID string) error {
	key := sessionPrefix + sessionID
	return c.rdb.Del(ctx, key).Err()
}

// AddToBlacklist добавляет токен в черный список
func (c *Client) AddToBlacklist(ctx context.Context, token string, expiration time.Duration) error {
	key := blacklistPrefix + token
	return c.rdb.Set(ctx, key, "1", expiration).Err()
}

// IsInBlacklist проверяет, находится ли токен в черном списке
func (c *Client) IsInBlacklist(ctx context.Context, token string) (bool, error) {
	key := blacklistPrefix + token
	_, err := c.rdb.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return false, nil
		}
		return false, fmt.Errorf("failed to check blacklist: %w", err)
	}
	return true, nil
}

// GetAllSessions возвращает все активные сессии (для отладки)
func (c *Client) GetAllSessions(ctx context.Context) (map[string]*models.SessionData, error) {
	keys, err := c.rdb.Keys(ctx, sessionPrefix+"*").Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get session keys: %w", err)
	}

	sessions := make(map[string]*models.SessionData)
	for _, key := range keys {
		sessionID := key[len(sessionPrefix):]
		data, err := c.GetSession(ctx, sessionID)
		if err != nil {
			continue // Пропускаем невалидные сессии
		}
		sessions[sessionID] = data
	}

	return sessions, nil
}
