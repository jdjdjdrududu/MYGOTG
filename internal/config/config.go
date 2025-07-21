// internal/config/config.go
package config

import (
	"log"
	"net/url"
	"os"
	"strconv"
	"strings"
)

// Config хранит все конфигурационные параметры приложения.
type Config struct {
	TelegramToken         string
	DatabaseURL           string
	AppEnv                string
	OwnerChatID           int64
	AccountingChatID      int64
	GroupChatID           int64
	BotUsername           string
	DBHost                string
	DBPort                string
	DBUser                string
	DBPassword            string
	DBName                string
	DriverSharePercentage float64
	YooKassaShopID        string
	YooKassaSecretKey     string

	// --- ДОБАВЛЕНО: ID канала для хранения файлов ---
	StorageChannelID int64 `yaml:"storage_channel_id"`
}

// LoadConfig загружает конфигурацию из переменных окружения.
func LoadConfig() (*Config, error) {
	cfg := &Config{
		TelegramToken: os.Getenv("TELEGRAM_APITOKEN"),
		DatabaseURL:   os.Getenv("DATABASE_URL"),
		AppEnv:        os.Getenv("ENV"),
		BotUsername:   os.Getenv("BOT_USERNAME"),
	}

	var err error
	cfg.OwnerChatID, err = strconv.ParseInt(os.Getenv("OWNER_CHAT_ID"), 10, 64)
	if err != nil {
		log.Printf("Предупреждение: не удалось прочитать OWNER_CHAT_ID: %v. Установлено в 0.", err)
		cfg.OwnerChatID = 0
	}

	cfg.AccountingChatID, err = strconv.ParseInt(os.Getenv("ACCOUNTING_CHAT_ID"), 10, 64)
	if err != nil {
		log.Printf("Предупреждение: не удалось прочитать ACCOUNTING_CHAT_ID: %v. Установлено в 0.", err)
		cfg.AccountingChatID = 0
	}

	cfg.GroupChatID, err = strconv.ParseInt(os.Getenv("GROUP_CHAT_ID"), 10, 64)
	if err != nil {
		log.Printf("Предупреждение: не удалось прочитать GROUP_CHAT_ID: %v. Установлено в 0.", err)
		cfg.GroupChatID = 0
	}

	// --- ДОБАВЛЕНО: Загрузка ID канала-хранилища ---
	cfg.StorageChannelID, err = strconv.ParseInt(os.Getenv("STORAGE_CHANNEL_ID"), 10, 64)
	if err != nil {
		log.Printf("Критическая ошибка: не удалось прочитать STORAGE_CHANNEL_ID: %v. Функции загрузки файлов не будут работать.", err)
	}
	// --- КОНЕЦ ДОБАВЛЕНИЯ ---

	driverShareStr := os.Getenv("DRIVER_SHARE_PERCENTAGE")
	if driverShareStr == "" {
		log.Printf("Предупреждение: DRIVER_SHARE_PERCENTAGE не установлен, используется значение по умолчанию 0.35 (35%%).")
		cfg.DriverSharePercentage = 0.35
	} else {
		driverShare, errParse := strconv.ParseFloat(driverShareStr, 64)
		if errParse != nil || driverShare <= 0 || driverShare >= 1 {
			log.Printf("Предупреждение: Некорректное значение для DRIVER_SHARE_PERCENTAGE ('%s'): %v. Используется значение по умолчанию 0.35.", driverShareStr, errParse)
			cfg.DriverSharePercentage = 0.35
		} else {
			cfg.DriverSharePercentage = driverShare
		}
	}

	cfg.YooKassaShopID = os.Getenv("YOOKASSA_SHOP_ID")
	cfg.YooKassaSecretKey = os.Getenv("YOOKASSA_SECRET_KEY")

	if cfg.YooKassaShopID == "" {
		log.Println("Предупреждение: YOOKASSA_SHOP_ID не установлен. Функции оплаты картой не будут работать.")
	}
	if cfg.YooKassaSecretKey == "" {
		log.Println("Предупреждение: YOOKASSA_SECRET_KEY не установлен. Функции оплаты картой не будут работать.")
	}

	if cfg.TelegramToken == "" {
		log.Println("Критическая ошибка: TELEGRAM_APITOKEN не установлен.")
	}
	if cfg.DatabaseURL == "" {
		log.Println("Критическая ошибка: DATABASE_URL не установлен.")
	} else {
		parsedURL, parseErr := url.Parse(cfg.DatabaseURL)
		if parseErr != nil {
			log.Printf("Критическая ошибка: ошибка парсинга DATABASE_URL: %v", parseErr)
		} else {
			cfg.DBHost = parsedURL.Hostname()
			cfg.DBPort = parsedURL.Port()
			if cfg.DBPort == "" {
				cfg.DBPort = "5432"
			}
			cfg.DBUser = parsedURL.User.Username()
			cfg.DBPassword, _ = parsedURL.User.Password()
			cfg.DBName = strings.TrimPrefix(parsedURL.Path, "/")
		}
	}
	if cfg.BotUsername == "" {
		log.Println("Предупреждение: BOT_USERNAME не установлен.")
	}

	log.Println("Конфигурация загружена.")
	return cfg, nil
}
