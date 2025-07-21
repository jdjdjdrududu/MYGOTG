package telegram_api

import (
	"fmt"
	"log"

	tgbotapi "github.com/OvyFlash/telegram-bot-api"
)

// BotClient представляет собой обертку для Telegram Bot API.
// Он будет содержать экземпляр бота и, возможно, конфигурацию.
// BotClient represents a wrapper for the Telegram Bot API.
// It will contain the bot instance and, possibly, configuration.
type BotClient struct {
	api   *tgbotapi.BotAPI
	Debug bool
}

// Global Bot instance for the package
// Глобальный экземпляр бота для пакета
var Client *BotClient

// InitBot инициализирует Telegram бота.
// token - API токен вашего бота.
// debug - флаг для включения режима отладки.
// botUsername - имя пользователя бота, может использоваться для генерации ссылок.
// InitBot initializes the Telegram bot.
// token - API token of your bot.
// debug - flag to enable debug mode.
// botUsername - bot's username, can be used for link generation.
func InitBot(token string, debug bool, botUsername string) error {
	if token == "" {
		return fmt.Errorf("токен Telegram API не предоставлен")
	}

	api, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		return fmt.Errorf("ошибка инициализации Telegram Bot API: %w", err)
	}

	api.Debug = debug // Используем флаг debug / Use debug flag

	log.Printf("Авторизован как аккаунт %s", api.Self.UserName)

	// Отключаем вебхук, если он активен (важно для getUpdates)
	// Disable webhook if active (important for getUpdates)
	deleteWebhookConfig := tgbotapi.DeleteWebhookConfig{
		DropPendingUpdates: true, // Сбрасываем ожидающие обновления / Drop pending updates
	}
	_, err = api.Request(deleteWebhookConfig)
	if err != nil {
		// Ошибка может возникнуть, если вебхука и не было.
		// Логируем, но не прерываем инициализацию.
		// An error might occur if no webhook was set.
		// Log, but do not interrupt initialization.
		log.Printf("Предупреждение или ошибка при отключении вебхука: %v. Это может быть нормально, если вебхук не был установлен.", err)
	} else {
		log.Println("Вебхук успешно отключен (или не был установлен).")
	}

	Client = &BotClient{
		api:   api,
		Debug: debug,
	}
	return nil
}

// GetAPI возвращает нижележащий экземпляр *tgbotapi.BotAPI.
// GetAPI returns the underlying *tgbotapi.BotAPI instance.
func (bc *BotClient) GetAPI() *tgbotapi.BotAPI {
	if bc == nil || bc.api == nil {
		// Это критическая ошибка, если пытаемся использовать неинициализированный клиент
		// This is a critical error if trying to use an uninitialized client
		log.Fatal("BotClient или его API не инициализирован.")
	}
	return bc.api
}

// GetUpdatesChan возвращает канал обновлений от Telegram.
// GetUpdatesChan returns the update channel from Telegram.
func (bc *BotClient) GetUpdatesChan(config tgbotapi.UpdateConfig) tgbotapi.UpdatesChannel {
	if bc == nil || bc.api == nil {
		log.Fatal("BotClient или его API не инициализирован перед запросом обновлений.")
	}
	if bc.Debug {
		log.Printf("Запрос канала обновлений с конфигурацией: %+v", config)
	}
	return bc.api.GetUpdatesChan(config)
}

// Send отправляет сообщение через BotClient.
// Send sends a message via BotClient.
func (bc *BotClient) Send(c tgbotapi.Chattable) (tgbotapi.Message, error) {
	if bc == nil || bc.api == nil {
		return tgbotapi.Message{}, fmt.Errorf("BotClient или его API не инициализирован")
	}
	if bc.Debug {
		// Логирование может быть очень объемным, особенно для сложных структур.
		// Logging can be very verbose, especially for complex structures.
		if msg, ok := c.(tgbotapi.MessageConfig); ok {
			log.Printf("Отправка сообщения: ChatID=%d, Text='%.50s...'", msg.ChatID, msg.Text)
		} else if editMsg, ok := c.(tgbotapi.EditMessageTextConfig); ok {
			log.Printf("Редактирование сообщения: ChatID=%d, MessageID=%d, Text='%.50s...'", editMsg.ChatID, editMsg.MessageID, editMsg.Text)
		} else if photoMsg, ok := c.(tgbotapi.PhotoConfig); ok {
			log.Printf("Отправка фото: ChatID=%d, Caption='%.50s...'", photoMsg.ChatID, photoMsg.Caption)
		} else {
			log.Printf("Отправка/запрос типа %T", c)
		}
	}
	return bc.api.Send(c)
}

// Request выполняет запрос через BotClient.
// Request performs a request via BotClient.
func (bc *BotClient) Request(c tgbotapi.Chattable) (*tgbotapi.APIResponse, error) {
	if bc == nil || bc.api == nil {
		return nil, fmt.Errorf("BotClient или его API не инициализирован")
	}
	if bc.Debug {
		if delMsg, ok := c.(tgbotapi.DeleteMessageConfig); ok {
			log.Printf("Запрос на удаление: ChatID=%d, MessageID=%d", delMsg.ChatID, delMsg.MessageID)
		} else if cbAns, ok := c.(tgbotapi.CallbackConfig); ok {
			log.Printf("Запрос ответа на коллбэк: CallbackQueryID=%s, Text='%.50s...'", cbAns.CallbackQueryID, cbAns.Text)
		} else {
			log.Printf("Выполнение запроса типа %T", c)
		}
	}
	return bc.api.Request(c)
}

// MakeRequest выполняет произвольный запрос к API Telegram.
// Этот метод полезен для вызовов API, не обернутых в стандартные методы tgbotapi.
// MakeRequest performs an arbitrary request to the Telegram API.
// This method is useful for API calls not wrapped in standard tgbotapi methods.
func (bc *BotClient) MakeRequest(endpoint string, params tgbotapi.Params) (*tgbotapi.APIResponse, error) {
	if bc == nil || bc.api == nil {
		return nil, fmt.Errorf("BotClient или его API не инициализирован")
	}
	if bc.Debug {
		log.Printf("Выполнение MakeRequest: endpoint=%s, params=%v", endpoint, params)
	}
	return bc.api.MakeRequest(endpoint, params)
}
