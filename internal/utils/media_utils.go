// internal/utils/media_utils.go
package utils

import (
	"path"
	"strings" // Добавляем этот импорт

	tgbotapi "github.com/OvyFlash/telegram-bot-api"
)

// IsVideo проверяет, является ли MIME-тип видео.
// Эта функция нужна для обработчика загрузки файлов.
func IsVideo(mimeType string) bool {
	return strings.HasPrefix(mimeType, "video/")
}

// GetMediaType определяет тип медиа в сообщении.
// Эта функция полезна для анализа входящих сообщений от пользователя в боте.
func GetMediaType(msg *tgbotapi.Message) string {
	if msg == nil {
		return "unknown"
	}
	if msg.Photo != nil && len(msg.Photo) > 0 {
		return "photo"
	} else if msg.Video != nil {
		return "video"
	} else if msg.Document != nil {
		return msg.Document.MimeType
	}
	return "unknown"
}

// ExtractFilenamesFromUrls извлекает только имена файлов из полных или относительных URL.
// Например, из "/api/media/file.jpg" вернет "file.jpg".
func ExtractFilenamesFromUrls(urls []string) []string {
	filenames := make([]string, len(urls))
	for i, url := range urls {
		if strings.Contains(url, "/") {
			filenames[i] = path.Base(url)
		} else {
			filenames[i] = url // Если это уже имя файла
		}
	}
	return filenames
}
