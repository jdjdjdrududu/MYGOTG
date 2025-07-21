package telegram_api

import (
	"fmt"
	"log"
	"strings"

	tgbotapi "github.com/OvyFlash/telegram-bot-api"
)

// SendOrEditMessage пытается отредактировать существующее сообщение или отправляет новое.
// Возвращает отправленное/отредактированное сообщение или пустое сообщение при ошибке.
// Если редактирование не удалось из-за "message is not modified", возвращает "фиктивный"
// Message объект с ID оригинального сообщения и nil в качестве ошибки.
func SendOrEditMessage(
	botClient *BotClient,
	chatID int64,
	messageIDToTryEdit int,
	text string,
	keyboard *tgbotapi.InlineKeyboardMarkup,
	parseMode string,
) (tgbotapi.Message, error) {
	if botClient == nil || botClient.api == nil {
		log.Println("SendOrEditMessage: BotClient или его API не инициализирован.")
		return tgbotapi.Message{}, fmt.Errorf("BotClient не инициализирован")
	}

	// Создаем "фиктивный" объект Message для возврата в случае успешного no-op редактирования
	// или успешного реального редактирования.
	var originalMsgObject tgbotapi.Message
	if messageIDToTryEdit != 0 {
		var chatObj tgbotapi.Chat
		chatObj.ID = chatID
		originalMsgObject.Chat = chatObj // <- ИСПРАВЛЕНО
		originalMsgObject.MessageID = messageIDToTryEdit
		originalMsgObject.Text = text // Возвращаем запрошенный текст
		if keyboard != nil {
			originalMsgObject.ReplyMarkup = keyboard // Возвращаем запрошенную клавиатуру
		}
	}

	if messageIDToTryEdit != 0 {
		var editMsgConfig tgbotapi.EditMessageTextConfig
		if keyboard != nil {
			editMsgConfig = tgbotapi.NewEditMessageTextAndMarkup(chatID, messageIDToTryEdit, text, *keyboard)
		} else {
			editMsgConfig = tgbotapi.NewEditMessageText(chatID, messageIDToTryEdit, text)
		}
		if parseMode != "" {
			editMsgConfig.ParseMode = parseMode
		}

		_, err := botClient.Request(editMsgConfig)
		if err == nil {
			// Успешное редактирование.
			return originalMsgObject, nil
		}

		// Если ошибка "message is not modified", это не фатальная ошибка для нас,
		// просто означает, что контент не изменился.
		// Возвращаем оригинальный ID сообщения и nil ошибку.
		if strings.Contains(err.Error(), "message is not modified") {
			log.Printf("SendOrEditMessage: Сообщение не изменено (ожидаемо): chatID=%d, MessageID=%d. Текст: '%.50s...'", chatID, messageIDToTryEdit, text)
			return originalMsgObject, nil // Возвращаем объект оригинального сообщения и nil ошибку
		}

		// Если ошибка "message to edit not found", это также может быть ожидаемо,
		// если сообщение было удалено. В этом случае мы отправим новое сообщение.
		if strings.Contains(err.Error(), "message to edit not found") {
			log.Printf("SendOrEditMessage: Ошибка редактирования (сообщение не найдено): chatID=%d, MessageID=%d: %v. Будет отправлено новое.", chatID, messageIDToTryEdit, err)
			// Продолжаем для отправки нового сообщения
		} else {
			// Другие ошибки редактирования логируем как неожиданные
			log.Printf("SendOrEditMessage: НЕОЖИДАННАЯ ОШИБКА редактирования сообщения chatID=%d, MessageID=%d: %v. Будет отправлено новое.", chatID, messageIDToTryEdit, err)
			// Продолжаем для отправки нового сообщения
		}
	}

	// Отправка нового сообщения, если редактирование не удалось (кроме "not modified") или не требовалось
	newMsg := tgbotapi.NewMessage(chatID, text)
	if keyboard != nil {
		newMsg.ReplyMarkup = keyboard
	}
	if parseMode != "" {
		newMsg.ParseMode = parseMode
	}

	actualSentMsg, err := botClient.Send(newMsg)
	if err != nil {
		log.Printf("SendOrEditMessage: ОШИБКА отправки нового сообщения для chatID %d: %v", chatID, err)
		return tgbotapi.Message{}, err
	}
	log.Printf("SendOrEditMessage: Отправлено новое сообщение ID %d для chatID %d. Text: '%.50s...'", actualSentMsg.MessageID, chatID, text)
	return actualSentMsg, nil
}

// SendErrorMessage отправляет стандартизированное сообщение об ошибке пользователю.
func SendErrorMessage(
	botClient *BotClient,
	chatID int64,
	messageIDToTryEdit int,
	errorText string,
) (tgbotapi.Message, error) {
	log.Printf("Отправка сообщения об ошибке для chatID %d: %s", chatID, errorText)
	if botClient == nil || botClient.api == nil {
		log.Println("SendErrorMessage: BotClient или его API не инициализирован.")
		return tgbotapi.Message{}, fmt.Errorf("BotClient не инициализирован")
	}

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔙 Главное меню", "back_to_main"),
		),
	)
	return SendOrEditMessage(botClient, chatID, messageIDToTryEdit, errorText, &keyboard, tgbotapi.ModeMarkdown)
}

// DeleteMessage удаляет сообщение.
func DeleteMessage(botClient *BotClient, chatID int64, messageID int) bool {
	if botClient == nil || botClient.api == nil {
		log.Println("DeleteMessage: BotClient или его API не инициализирован.")
		return false
	}
	if messageID == 0 {
		return false
	}

	deleteConfig := tgbotapi.NewDeleteMessage(chatID, messageID)
	response, err := botClient.Request(deleteConfig)

	if err != nil {
		log.Printf("DeleteMessage API Call Error: ChatID=%d, MessageID=%d, Error: %v", chatID, messageID, err)
		return false
	}
	if !response.Ok {
		if response.Description != "Bad Request: message to delete not found" &&
			response.Description != "Bad Request: message can't be deleted" &&
			!strings.Contains(response.Description, "MESSAGE_ID_INVALID") {
			log.Printf("DeleteMessage: Telegram API не смог удалить сообщение %d для chatID %d: %s (ErrorCode: %d)", messageID, chatID, response.Description, response.ErrorCode)
		}
		return false
	}
	// log.Printf("DeleteMessage API Response OK: ChatID=%d, MessageID=%d successfully marked for deletion by API.", chatID, messageID)
	return true
}

// DeleteMessages удаляет список сообщений.
// Возвращает количество успешно удаленных сообщений.
func DeleteMessages(botClient *BotClient, chatID int64, messageIDs []int) int {
	if botClient == nil || botClient.api == nil {
		log.Println("DeleteMessages: BotClient или его API не инициализирован.")
		return 0
	}
	if len(messageIDs) == 0 {
		return 0
	}
	successfullyDeleted := 0
	for _, msgID := range messageIDs {
		if DeleteMessage(botClient, chatID, msgID) {
			successfullyDeleted++
		}
	}
	if successfullyDeleted > 0 {
		// log.Printf("DeleteMessages: Успешно удалено %d из %d сообщений для chatID %d.", successfullyDeleted, len(messageIDs), chatID)
	}
	return successfullyDeleted
}
