package utils

import (
	"fmt"
	"log"

	"github.com/skip2/go-qrcode" // Убедитесь, что этот импорт корректен
)

// GenerateReferralLink генерирует реферальную ссылку для пользователя.
// botUsername должен передаваться, так как это конфигурационное значение.
func GenerateReferralLink(botUsername string, chatID int64) (string, error) {
	if botUsername == "" {
		log.Println("GenerateReferralLink: botUsername не предоставлен.")
		return "", fmt.Errorf("имя пользователя бота не настроено")
	}
	if chatID == 0 { // Или другая проверка на валидность chatID
		log.Printf("GenerateReferralLink: невалидный chatID: %d", chatID)
		return "", fmt.Errorf("невалидный ID пользователя для реферальной ссылки")
	}
	return fmt.Sprintf("https://t.me/%s?start=ref_%d", botUsername, chatID), nil
}

// GenerateQRCode генерирует QR-код для реферальной ссылки.
// botUsername также нужен здесь, так как он используется в GenerateReferralLink.
func GenerateQRCode(botUsername string, chatID int64) ([]byte, error) {
	link, err := GenerateReferralLink(botUsername, chatID)
	if err != nil {
		log.Printf("GenerateQRCode: ошибка генерации реферальной ссылки для QR-кода (chatID %d): %v", chatID, err)
		return nil, err
	}

	// qrcode.Medium - уровень коррекции ошибок, 256 - размер QR-кода в пикселях.
	qrBytes, err := qrcode.Encode(link, qrcode.Medium, 256)
	if err != nil {
		log.Printf("GenerateQRCode: ошибка кодирования QR-кода для ссылки '%s': %v", link, err)
		return nil, err
	}
	return qrBytes, nil
}
