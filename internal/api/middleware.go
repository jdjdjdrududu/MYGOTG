// Файл: Original/internal/api/middleware.go
package api

import (
	"Original/internal/models"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"database/sql"
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

			// Для development режима разрешаем fallback авторизацию
			if authHeader == "fallback-development-mode" {
				// Создаем тестового пользователя для development
				testUser := models.User{
					ID:        1263060321,
					ChatID:    1263060321,
					FirstName: "Оператор",
					LastName:  "Сервис-Крым",
					Nickname:  sql.NullString{String: "Demontaj_Crimea", Valid: true},
					Role:      "operator",
					Phone:     sql.NullString{String: "+79781234567", Valid: true},
					IsBlocked: false,
				}

				log.Printf("AuthMiddleware: Using fallback development user for testing")
				ctx := context.WithValue(r.Context(), UserContextKey, testUser)
				next.ServeHTTP(w, r.WithContext(ctx))
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
func validateInitData(initData, botToken string) (bool, telegramUserData, error) {
	var userData telegramUserData

	// Parse the initData string
	q, err := url.ParseQuery(initData)
	if err != nil {
		return false, userData, fmt.Errorf("failed to parse initData: %w", err)
	}

	// Get the hash to verify
	hash := q.Get("hash")
	if hash == "" {
		return false, userData, fmt.Errorf("hash is not present in initData")
	}

	// Remove hash from the data before checking
	q.Del("hash")

	// Sort the remaining parameters
	var pairs []string
	for k, v := range q {
		pairs = append(pairs, fmt.Sprintf("%s=%s", k, v[0]))
	}
	sort.Strings(pairs)
	dataCheckString := strings.Join(pairs, "\n")

	// Generate secret key using bot token
	secretKey := hmac.New(sha256.New, []byte("WebAppData"))
	secretKey.Write([]byte(botToken))
	secret := secretKey.Sum(nil)

	// Calculate hash
	h := hmac.New(sha256.New, secret)
	h.Write([]byte(dataCheckString))
	calculatedHash := hex.EncodeToString(h.Sum(nil))

	// Extract user data
	userJSON := q.Get("user")
	if userJSON == "" {
		return false, userData, fmt.Errorf("user data is not present in initData")
	}
	if err := json.Unmarshal([]byte(userJSON), &userData); err != nil {
		return false, userData, fmt.Errorf("failed to unmarshal user data: %w", err)
	}

	return calculatedHash == hash, userData, nil
}
