package utils

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"os" // Для получения ключа из переменных окружения
)

// encryptionKey stores the global encryption key.
// It's initialized by InitEncryptionKey().
var encryptionKey []byte

// InitEncryptionKey initializes the global encryptionKey from an environment variable.
// It should be called once at application startup.
func InitEncryptionKey() error {
	keyHex := os.Getenv("CARD_ENCRYPTION_KEY_HEX") // 32-byte key in HEX (64 characters)
	if keyHex == "" {
		// В реальном приложении это должно быть фатальной ошибкой,
		// так как без ключа шифрование/дешифрование невозможно.
		log.Println("КРИТИЧЕСКАЯ ОШИБКА: Ключ шифрования CARD_ENCRYPTION_KEY_HEX не установлен в переменных окружения.")
		return fmt.Errorf("ключ шифрования CARD_ENCRYPTION_KEY_HEX не установлен")
	}

	var err error
	encryptionKey, err = hex.DecodeString(keyHex)
	if err != nil {
		log.Printf("КРИТИЧЕСКАЯ ОШИБКА: Не удалось декодировать CARD_ENCRYPTION_KEY_HEX: %v", err)
		return fmt.Errorf("некорректный формат ключа шифрования (не HEX): %w", err)
	}

	if len(encryptionKey) != 32 { // AES-256 requires a 32-byte key.
		log.Printf("КРИТИЧЕСКАЯ ОШИБКА: Длина ключа шифрования должна быть 32 байта (64 HEX символа), получено %d байт.", len(encryptionKey))
		return fmt.Errorf("некорректная длина ключа шифрования, требуется 32 байта, получено %d", len(encryptionKey))
	}

	log.Println("Ключ шифрования успешно инициализирован.")
	return nil
}

// EncryptCardNumber encrypts a plaintext card number using AES-256-GCM.
// Returns the hex-encoded ciphertext.
func EncryptCardNumber(plainTextCardNumber string) (string, error) {
	if len(encryptionKey) == 0 { // Проверка, что ключ был инициализирован
		log.Println("Ошибка шифрования: ключ шифрования не инициализирован. Вызовите InitEncryptionKey().")
		return "", fmt.Errorf("ключ шифрования не инициализирован")
	}

	block, err := aes.NewCipher(encryptionKey)
	if err != nil {
		log.Printf("Ошибка создания шифра AES: %v", err)
		return "", fmt.Errorf("ошибка создания шифра: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		log.Printf("Ошибка создания GCM: %v", err)
		return "", fmt.Errorf("ошибка создания GCM: %w", err)
	}

	// Never use more than 2^32 random nonces with a given key because of the risk of a repeat.
	nonce := make([]byte, gcm.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		log.Printf("Ошибка генерации nonce: %v", err)
		return "", fmt.Errorf("ошибка генерации nonce: %w", err)
	}

	// Seal will append the nonce to the beginning of the ciphertext.
	cipherText := gcm.Seal(nonce, nonce, []byte(plainTextCardNumber), nil)
	return hex.EncodeToString(cipherText), nil
}

// DecryptCardNumber decrypts a hex-encoded ciphertext card number using AES-256-GCM.
// Returns the plaintext card number.
func DecryptCardNumber(cipherTextCardNumberHex string) (string, error) {
	if len(encryptionKey) == 0 { // Проверка, что ключ был инициализирован
		log.Println("Ошибка дешифрования: ключ шифрования не инициализирован. Вызовите InitEncryptionKey().")
		return "", fmt.Errorf("ключ шифрования не инициализирован")
	}

	cipherText, err := hex.DecodeString(cipherTextCardNumberHex)
	if err != nil {
		log.Printf("Ошибка декодирования HEX зашифрованного номера карты: %v", err)
		return "", fmt.Errorf("не удалось декодировать зашифрованный номер карты из hex: %w", err)
	}

	block, err := aes.NewCipher(encryptionKey)
	if err != nil {
		log.Printf("Ошибка создания шифра AES при дешифровании: %v", err)
		return "", fmt.Errorf("ошибка создания шифра при дешифровании: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		log.Printf("Ошибка создания GCM при дешифровании: %v", err)
		return "", fmt.Errorf("ошибка создания GCM при дешифровании: %w", err)
	}

	if len(cipherText) < gcm.NonceSize() {
		log.Println("Ошибка дешифрования: размер зашифрованного текста меньше размера nonce.")
		return "", fmt.Errorf("размер зашифрованного текста меньше размера nonce")
	}

	// The nonce is prefixed to the ciphertext.
	nonce, actualCipherText := cipherText[:gcm.NonceSize()], cipherText[gcm.NonceSize():]

	plainText, err := gcm.Open(nil, nonce, actualCipherText, nil)
	if err != nil {
		log.Printf("Ошибка дешифрования номера карты (возможно, неверный ключ или поврежденные данные): %v", err)
		return "", fmt.Errorf("ошибка дешифрования номера карты: %w", err)
	}

	return string(plainText), nil
}
