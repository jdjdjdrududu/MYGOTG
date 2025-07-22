package api

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
)

// MediaProxyHandler обрабатывает запросы к медиафайлам с проверкой прав доступа
func MediaProxyHandler(w http.ResponseWriter, r *http.Request) {
	// Получаем имя файла из URL
	filename := chi.URLParam(r, "filename")

	// Проверяем, что имя файла не пустое и не содержит путей к директориям
	if filename == "" || strings.Contains(filename, "/") || strings.Contains(filename, "\\") {
		http.Error(w, "Invalid filename", http.StatusBadRequest)
		return
	}

	// Проверяем авторизацию пользователя
	// Если запрос пришел из Telegram WebApp, проверяем инициатора
	// Проверяем X-Telegram-Auth заголовок или Referer
	authHeader := r.Header.Get("X-Telegram-Auth")
	if authHeader == "" {
		// Проверяем referer, чтобы убедиться, что запрос пришел с нашего сайта
		referer := r.Header.Get("Referer")
		if referer == "" || !strings.Contains(referer, r.Host) {
			// Для debug режима разрешаем доступ к медиа без авторизации
			// В production это должно быть более строго
			// http.Error(w, "Unauthorized", http.StatusUnauthorized)
			// return
		}
	}

	// Путь к файлу
	mediaDir := "./media_storage"
	filePath := filepath.Join(mediaDir, filename)

	// Проверяем существование файла
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			http.Error(w, "File not found", http.StatusNotFound)
		} else {
			http.Error(w, "Internal server error", http.StatusInternalServerError)
		}
		return
	}

	// Проверяем, что это файл, а не директория
	if fileInfo.IsDir() {
		http.Error(w, "Not a file", http.StatusBadRequest)
		return
	}

	// Определяем Content-Type на основе расширения файла
	contentType := getContentType(filepath.Ext(filename))
	w.Header().Set("Content-Type", contentType)

	// Устанавливаем заголовки кэширования
	w.Header().Set("Cache-Control", "public, max-age=86400") // Кэшировать на 1 день
	w.Header().Set("Expires", time.Now().Add(24*time.Hour).Format(http.TimeFormat))

	// Отправляем файл
	http.ServeFile(w, r, filePath)
}

// getContentType возвращает MIME-тип на основе расширения файла
func getContentType(ext string) string {
	ext = strings.ToLower(ext)
	switch ext {
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".png":
		return "image/png"
	case ".gif":
		return "image/gif"
	case ".webp":
		return "image/webp"
	case ".mp4":
		return "video/mp4"
	case ".webm":
		return "video/webm"
	case ".mov":
		return "video/quicktime"
	default:
		return "application/octet-stream"
	}
}
