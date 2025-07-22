// Файл: Original/internal/api/middleware.go
package api

import (
	"Original/internal/models"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"sort"
	"strings"

	"Original/internal/db"
	"Original/internal/utils"
)

// UserContextKey - ключ для сохранения данных пользователя в контексте запроса.
var UserContextKey = &contextKey{"User"}

// BotContextKey - ключ для сохранения бота в контексте запроса.
var BotContextKey = &contextKey{"Bot"}

// ConfigContextKey - ключ для сохранения конфига в контексте запроса.
var ConfigContextKey = &contextKey{"Config"}

type contextKey struct {
	name string
}

// AuthMiddleware проверяет заголовок X-Telegram-Auth с initData.
func AuthMiddleware(secretKey string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("X-Telegram-Auth")
			if authHeader == "" {
				http.Error(w, "Unauthorized: Missing X-Telegram-Auth header", http.StatusUnauthorized)
				return
			}

			// Валидация initData
			isValid, userData, err := validateInitData(authHeader, secretKey)
			if err != nil || !isValid {
				log.Printf("AuthMiddleware: Invalid initData. Error: %v", err)
				http.Error(w, "Unauthorized: Invalid initData", http.StatusUnauthorized)
				return
			}

			// Получаем полную информацию о пользователе из нашей БД
			user, err := db.GetUserByChatID(userData.ID) //
			if err != nil {
				log.Printf("AuthMiddleware: User not found in DB. ChatID: %d. Error: %v", userData.ID, err) //
				http.Error(w, "Unauthorized: User not found", http.StatusUnauthorized)                      //
				return
			}

			// Сохраняем пользователя в контексте запроса для последующих обработчиков
			ctx := context.WithValue(r.Context(), UserContextKey, user)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RoleMiddleware проверяет, соответствует ли роль пользователя требуемой.
func RoleMiddleware(requiredRole string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user, ok := r.Context().Value(UserContextKey).(models.User)
			if !ok {
				http.Error(w, "Forbidden: User data not found in context", http.StatusForbidden)
				return
			}

			if !utils.IsRoleOrHigher(user.Role, requiredRole) {
				http.Error(w, "Forbidden: Insufficient permissions", http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// BotMiddleware добавляет бота в контекст запроса.
func BotMiddleware(bot interface{}) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := context.WithValue(r.Context(), BotContextKey, bot)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// ConfigMiddleware добавляет конфиг в контекст запроса.
func ConfigMiddleware(cfg interface{}) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := context.WithValue(r.Context(), ConfigContextKey, cfg)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// Структура для парсинга JSON из initData
type telegramUserData struct {
	ID        int64  `json:"id"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Username  string `json:"username"`
}

// validateInitData - функция для проверки подлинности данных от Telegram.
func validateInitData(initData, secret string) (bool, telegramUserData, error) {
	var userData telegramUserData

	q, err := url.ParseQuery(initData)
	if err != nil {
		return false, userData, fmt.Errorf("failed to parse initData: %w", err)
	}

	hash := q.Get("hash")
	if hash == "" {
		return false, userData, fmt.Errorf("hash is not present in initData")
	}

	// Извлекаем JSON с данными пользователя
	userJSON := q.Get("user")
	if userJSON == "" {
		return false, userData, fmt.Errorf("user data is not present in initData")
	}
	if err := json.Unmarshal([]byte(userJSON), &userData); err != nil {
		return false, userData, fmt.Errorf("failed to unmarshal user data: %w", err)
	}

	var pairs []string
	for k, v := range q {
		if k != "hash" {
			pairs = append(pairs, fmt.Sprintf("%s=%s", k, v[0]))
		}
	}
	sort.Strings(pairs)
	dataCheckString := strings.Join(pairs, "\n")

	secretKey := hmac.New(sha256.New, []byte("WebAppData"))
	secretKey.Write([]byte(secret))

	h := hmac.New(sha256.New, secretKey.Sum(nil))
	h.Write([]byte(dataCheckString))
	calculatedHash := hex.EncodeToString(h.Sum(nil))

	return calculatedHash == hash, userData, nil
}
