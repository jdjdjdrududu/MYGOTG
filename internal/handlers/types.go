package handlers

import (
	// Импорты ваших пакетов / Imports of your packages
	"Original/internal/config" // Используем Original как имя модуля / Use Original as module name
	"Original/internal/db"     // Для прямого доступа к db.DB, если потребуется / For direct access to db.DB, if needed
	"Original/internal/models"
	"Original/internal/session"      // Менеджер сессий / Session manager
	"Original/internal/telegram_api" // Клиент Telegram API / Telegram API client
	"log"
)

// HandlerDependencies содержит все зависимости, необходимые для обработчиков.
// HandlerDependencies contains all dependencies required for handlers.
type HandlerDependencies struct {
	Config         *config.Config
	BotClient      *telegram_api.BotClient
	SessionManager *session.SessionManager
	// DB *sql.DB // Можно передавать db.DB напрямую, если обработчики часто к нему обращаются,
	// но лучше, если они будут использовать функции из пакета db.
	// Глобальный db.DB все еще доступен из пакета db.
	// DB *sql.DB // Can pass db.DB directly if handlers access it frequently,
	// but it's better if they use functions from the db package.
	// Global db.DB is still accessible from the db package.
}

// BotHandler инкапсулирует логику обработки сообщений и коллбэков.
// BotHandler encapsulates the logic for handling messages and callbacks.
type BotHandler struct {
	Deps HandlerDependencies
}

// NewBotHandler создает новый экземпляр BotHandler.
// NewBotHandler creates a new instance of BotHandler.
func NewBotHandler(deps HandlerDependencies) *BotHandler {
	if deps.Config == nil || deps.BotClient == nil || deps.SessionManager == nil {
		// Это критическая ошибка конфигурации, приложение не сможет работать корректно.
		// В реальном приложении здесь должна быть более строгая обработка.
		// This is a critical configuration error; the application will not work correctly.
		// In a real application, there should be stricter handling here.
		panic("Не все зависимости для BotHandler были предоставлены.")
	}
	return &BotHandler{Deps: deps}
}

// Вспомогательная структура для передачи параметров при отправке меню.
// Может быть расширена по мере необходимости.
// Helper structure for passing parameters when sending a menu.
// Can be extended as needed.
type menuContext struct {
	ChatID    int64
	UserID    int64 // ID пользователя из нашей БД (не chat_id) / User ID from our DB (not chat_id)
	UserRole  string
	FirstName string
	MessageID int                   // ID сообщения, которое нужно отредактировать или на которое ответить / ID of the message to edit or reply to
	Page      int                   // Для пагинации / For pagination
	OrderData session.TempOrderData // Временные данные заказа из сессии / Temporary order data from session
	// ... другие поля, если нужны ... / ... other fields if needed ...
}

// Helper to get user from DB or handle error
// Вспомогательная функция для получения пользователя из БД или обработки ошибки
func (bh *BotHandler) getUserFromDB(chatID int64) (models.User, bool) {
	user, err := db.GetUserByChatID(chatID) // Предполагается, что db.GetUserByChatID доступна / Assumes db.GetUserByChatID is available
	if err != nil {
		log.Printf("Ошибка получения пользователя %d из БД: %v", chatID, err)
		// Отправляем сообщение об ошибке пользователю, если это уместно
		// Send an error message to the user if appropriate
		// bh.Deps.BotClient.Send(tgbotapi.NewMessage(chatID, "Произошла ошибка с вашими данными. Попробуйте /start"))
		return models.User{}, false
	}
	return user, true
}
