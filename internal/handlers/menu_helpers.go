package handlers

import (
	"Original/internal/constants"
	"Original/internal/db" // Для db.GetUserMainMenuMessageID, db.ResetUserMainMenuMessageID
	"Original/internal/models"
	"Original/internal/telegram_api"
	"Original/internal/utils" // Для utils.GetBackText
	"fmt"
	"log"

	tgbotapi "github.com/OvyFlash/telegram-bot-api"
)

// --- Вспомогательные функции для отправки сообщений и управления сессией ---
// --- Helper functions for sending messages and managing session ---

// sendOrEditMessageHelper отправляет или редактирует сообщение и обновляет CurrentMessageID и MediaMessageIDs в сессии.
// sendOrEditMessageHelper sends or edits a message and updates CurrentMessageID and MediaMessageIDs in the session.
func (bh *BotHandler) sendOrEditMessageHelper(
	chatID int64,
	messageIDToTryEdit int,
	text string,
	keyboard *tgbotapi.InlineKeyboardMarkup,
	parseMode string,
) (tgbotapi.Message, error) {
	sentMsg, err := telegram_api.SendOrEditMessage(bh.Deps.BotClient, chatID, messageIDToTryEdit, text, keyboard, parseMode)
	if err != nil {
		return tgbotapi.Message{}, err
	}

	if sentMsg.MessageID != 0 {
		orderData := bh.Deps.SessionManager.GetTempOrder(chatID) // Получаем копию структуры

		// Всегда создаем новые экземпляры карты и среза, чтобы избежать гонок при записи.
		// Копируем существующие данные, если они есть.

		// Создаем новую карту для MediaMessageIDsMap
		newMediaMap := make(map[string]bool)
		if orderData.MediaMessageIDsMap != nil { // Если старая карта существовала, копируем ее содержимое
			for k, v := range orderData.MediaMessageIDsMap {
				newMediaMap[k] = v
			}
		}

		// Создаем новый срез для MediaMessageIDs
		var newMediaIDs []int
		if orderData.MediaMessageIDs != nil { // Если старый срез существовал, копируем его содержимое
			newMediaIDs = make([]int, len(orderData.MediaMessageIDs))
			copy(newMediaIDs, orderData.MediaMessageIDs)
		} else {
			newMediaIDs = []int{} // Инициализируем пустым срезом, если старого не было
		}

		isNewPrimaryMessage := (messageIDToTryEdit == 0) || (messageIDToTryEdit != 0 && messageIDToTryEdit != orderData.CurrentMessageID)

		orderData.CurrentMessageID = sentMsg.MessageID

		if isNewPrimaryMessage {
			// Это сообщение начинает новый контекст/вид.
			// Очищаем предыдущие медиа и устанавливаем это сообщение как единственное.
			newMediaIDs = []int{sentMsg.MessageID} // Новый срез только с текущим ID

			// Очищаем новую карту и добавляем только текущий ID
			for k := range newMediaMap {
				delete(newMediaMap, k)
			}
			newMediaMap[fmt.Sprintf("%d", sentMsg.MessageID)] = true
			log.Printf("sendOrEditMessageHelper: Новое главное сообщение %d. MediaMessageIDs и Map сброшены и установлен ID нового сообщения. ChatID: %d", sentMsg.MessageID, chatID)
		} else {
			// Мы отредактировали существующий CurrentMessageID или другое сообщение стало текущим.
			// Убедимся, что CurrentMessageID есть в карте и срезе.
			newMediaMap[fmt.Sprintf("%d", sentMsg.MessageID)] = true // Добавляем/обновляем ID в новой карте

			foundInSlice := false
			for _, mid := range newMediaIDs { // Проверяем в новом срезе
				if mid == sentMsg.MessageID {
					foundInSlice = true
					break
				}
			}
			if !foundInSlice {
				newMediaIDs = append(newMediaIDs, sentMsg.MessageID) // Добавляем в новый срез
			}
			log.Printf("sendOrEditMessageHelper: Отредактировано/установлено сообщение CurrentMessageID %d. MediaMessageIDsMap и MediaMessageIDs обновлены. ChatID: %d", sentMsg.MessageID, chatID)
		}

		// Присваиваем обновленные (скопированные и измененные) карту и срез обратно в orderData
		orderData.MediaMessageIDsMap = newMediaMap
		orderData.MediaMessageIDs = newMediaIDs

		bh.Deps.SessionManager.UpdateTempOrder(chatID, orderData) // Записываем копию структуры (теперь с ее собственными копиями карты/среза) обратно
	}
	return sentMsg, nil
}

// sendErrorMessageHelper отправляет сообщение об ошибке и обновляет сессию, чтобы это сообщение стало CurrentMessageID.
// sendErrorMessageHelper sends an error message and updates the session so this message becomes CurrentMessageID.
func (bh *BotHandler) sendErrorMessageHelper(chatID int64, messageIDToEdit int, errorText string) (tgbotapi.Message, error) {
	// Вызываем функцию из пакета telegram_api, передавая клиент
	// Call function from telegram_api package, passing the client
	sentMsg, err := telegram_api.SendErrorMessage(bh.Deps.BotClient, chatID, messageIDToEdit, errorText)
	if err != nil {
		// Ошибка уже залогирована в telegram_api.SendErrorMessage
		// Error is already logged in telegram_api.SendErrorMessage
		return tgbotapi.Message{}, err
	}

	// Обновляем сессию, чтобы это сообщение об ошибке стало текущим
	// Update session so this error message becomes the current one
	if sentMsg.MessageID != 0 {
		orderData := bh.Deps.SessionManager.GetTempOrder(chatID)
		orderData.CurrentMessageID = sentMsg.MessageID
		// Сообщение об ошибке заменяет предыдущий контекст
		// Error message replaces previous context
		orderData.MediaMessageIDs = []int{sentMsg.MessageID}
		orderData.MediaMessageIDsMap = make(map[string]bool)
		orderData.MediaMessageIDsMap[fmt.Sprintf("%d", sentMsg.MessageID)] = true
		bh.Deps.SessionManager.UpdateTempOrder(chatID, orderData)
	}
	return sentMsg, nil
}

// deleteMessageHelper удаляет сообщение и помечает его как удаленное в сессии.
// deleteMessageHelper deletes a message and marks it as deleted in the session.
func (bh *BotHandler) deleteMessageHelper(chatID int64, messageID int) bool {
	if messageID == 0 {
		return false
	}
	// Проверяем, не было ли сообщение уже помечено как удаленное, чтобы избежать лишних API вызовов
	// Check if the message was already marked as deleted to avoid redundant API calls
	if bh.Deps.SessionManager.IsMessageDeleted(chatID, messageID) {
		// log.Printf("deleteMessageHelper: Сообщение %d для chatID %d уже было помечено как удаленное, пропуск API вызова.", messageID, chatID)
		return true // Считаем успешным, если уже помечено / Consider successful if already marked
	}

	// Вызываем функцию из пакета telegram_api, передавая клиент
	// Call function from telegram_api package, passing the client
	deleted := telegram_api.DeleteMessage(bh.Deps.BotClient, chatID, messageID)

	// Помечаем в любом случае после попытки, чтобы не пытаться удалить снова, если API вернуло ошибку "not found"
	// Mark anyway after attempt, to avoid trying to delete again if API returned "not found" error
	bh.Deps.SessionManager.MarkMessageAsDeleted(chatID, messageID)

	if deleted {
		// log.Printf("deleteMessageHelper: Сообщение %d для chatID %d успешно удалено через API.", messageID, chatID)
	} else {
		// log.Printf("deleteMessageHelper: Попытка удаления сообщения %d для chatID %d через API не удалась (или сообщение не найдено).", messageID, chatID)
	}
	return deleted
}

// deleteRecentMessagesHelper удаляет недавние сообщения из сессии (MediaMessageIDs), кроме исключенных.
// Также удаляет currentCallbackMessageID, если он не исключен.
// deleteRecentMessagesHelper deletes recent messages from the session (MediaMessageIDs), except excluded ones.
// Also deletes currentCallbackMessageID if it's not excluded.
func (bh *BotHandler) deleteRecentMessagesHelper(chatID int64, currentCallbackMessageID int, excludeIDs ...int) {
	orderData := bh.Deps.SessionManager.GetTempOrder(chatID)

	excludeSet := make(map[int]bool)
	for _, id := range excludeIDs {
		if id != 0 {
			excludeSet[id] = true
		}
	}

	// Собираем ID сообщений для удаления / Collect message IDs for deletion
	var messagesToDelete []int

	// Добавляем currentCallbackMessageID (сообщение с кнопкой) к удалению, если он не исключен
	// Add currentCallbackMessageID (message with button) for deletion if not excluded
	if currentCallbackMessageID != 0 && !excludeSet[currentCallbackMessageID] {
		if !bh.Deps.SessionManager.IsMessageDeleted(chatID, currentCallbackMessageID) {
			messagesToDelete = append(messagesToDelete, currentCallbackMessageID)
		}
	}

	// Добавляем сообщения из MediaMessageIDs к удалению, если они не исключены
	// Add messages from MediaMessageIDs for deletion if not excluded
	var newMediaIDs []int
	newMediaMap := make(map[string]bool)

	for _, id := range orderData.MediaMessageIDs {
		if id != 0 {
			if excludeSet[id] {
				// Если ID исключен, сохраняем его в новом списке
				// If ID is excluded, save it in the new list
				newMediaIDs = append(newMediaIDs, id)
				newMediaMap[fmt.Sprintf("%d", id)] = true
			} else if !bh.Deps.SessionManager.IsMessageDeleted(chatID, id) {
				// Добавляем к удалению, только если еще не помечено как удаленное
				// Add for deletion only if not already marked as deleted
				isAlreadyInDeleteList := false
				for _, delID := range messagesToDelete {
					if delID == id {
						isAlreadyInDeleteList = true
						break
					}
				}
				if !isAlreadyInDeleteList {
					messagesToDelete = append(messagesToDelete, id)
				}
			}
		}
	}
	// Обновляем MediaMessageIDs в сессии, оставляя только исключенные
	// Update MediaMessageIDs in session, keeping only excluded ones
	orderData.MediaMessageIDs = newMediaIDs
	orderData.MediaMessageIDsMap = newMediaMap
	bh.Deps.SessionManager.UpdateTempOrder(chatID, orderData)

	if len(messagesToDelete) > 0 {
		log.Printf("deleteRecentMessagesHelper: %d сообщений помечено для удаления для chatID %d (после фильтрации). Исключенные ID: %v", len(messagesToDelete), chatID, excludeIDs)
	}

	deletedCount := 0
	for _, msgID := range messagesToDelete {
		if bh.deleteMessageHelper(chatID, msgID) {
			deletedCount++
			// Проверяем, не удалили ли мы сохраненное главное меню
			// Check if we deleted the saved main menu
			mainMenuMsgID, errMM := db.GetUserMainMenuMessageID(chatID)
			if errMM == nil && mainMenuMsgID == msgID {
				// Если удаленное сообщение было главным меню, сбрасываем его ID в БД
				// If the deleted message was the main menu, reset its ID in the DB
				// Это важно, чтобы при следующем вызове SendMainMenu не пытаться редактировать удаленное
				// This is important so that SendMainMenu doesn't try to edit a deleted message next time
				if orderData.CurrentMessageID != mainMenuMsgID { // Не сбрасываем, если это текущее активное меню
					// Do not reset if it's the current active menu
					db.ResetUserMainMenuMessageID(chatID)
					log.Printf("deleteRecentMessagesHelper: main_menu_message_id %d сброшен для chatID %d, так как сообщение удалено.", msgID, chatID)
				}
			}
		}
	}
	if deletedCount > 0 {
		// log.Printf("deleteRecentMessagesHelper: Фактически удалено (или предпринята попытка удаления) %d сообщений для chatID %d.", deletedCount, chatID)
	}
}

// sendInfoMessage - общая функция для отправки информационного сообщения с кнопками "Назад" и "Главное меню".
// Теперь возвращает (tgbotapi.Message, error).
// sendInfoMessage - general function for sending an informational message with "Back" and "Main Menu" buttons.
// Now returns (tgbotapi.Message, error).
func (bh *BotHandler) sendInfoMessage(chatID int64, messageIDToEdit int, text string, backCallbackKey string) (tgbotapi.Message, error) {
	backButtonText := "⬅️ Назад" // Текст по умолчанию / Default text
	if specificBackText, ok := utils.GetBackText(backCallbackKey); ok {
		backButtonText = specificBackText
	}

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(backButtonText, backCallbackKey),
			tgbotapi.NewInlineKeyboardButtonData("🏢 Главное меню", "back_to_main"),
		),
	)
	// Используем sendOrEditMessageHelper, так как он обновляет CurrentMessageID
	// Use sendOrEditMessageHelper as it updates CurrentMessageID
	sentMsg, err := bh.sendOrEditMessageHelper(chatID, messageIDToEdit, text, &keyboard, tgbotapi.ModeMarkdown)
	if err != nil {
		log.Printf("sendInfoMessage: Ошибка для chatID %d: %v", chatID, err)
		return tgbotapi.Message{}, err
	}
	return sentMsg, nil
}

// sendMessage отправляет простое текстовое сообщение.
// Возвращает (tgbotapi.Message, error).
// sendMessage sends a simple text message.
// Returns (tgbotapi.Message, error).
func (bh *BotHandler) sendMessage(chatID int64, text string) (tgbotapi.Message, error) {
	if bh.Deps.BotClient == nil || bh.Deps.BotClient.GetAPI() == nil {
		log.Printf("sendMessage: BotClient не инициализирован для chatID %d", chatID)
		return tgbotapi.Message{}, fmt.Errorf("BotClient не инициализирован")
	}
	msg := tgbotapi.NewMessage(chatID, text)
	sentMsg, err := bh.Deps.BotClient.Send(msg)
	if err != nil {
		log.Printf("sendMessage: Ошибка отправки сообщения для chatID %d: %v", chatID, err)
		return tgbotapi.Message{}, err
	}
	return sentMsg, nil
}

// sendMessageWithKeyboard отправляет сообщение с указанной клавиатурой.
// Возвращает (tgbotapi.Message, error).
// sendMessageWithKeyboard sends a message with the specified keyboard.
// Returns (tgbotapi.Message, error).
func (bh *BotHandler) sendMessageWithKeyboard(chatID int64, text string, keyboard *tgbotapi.InlineKeyboardMarkup) (tgbotapi.Message, error) {
	if bh.Deps.BotClient == nil || bh.Deps.BotClient.GetAPI() == nil {
		log.Printf("sendMessageWithKeyboard: BotClient не инициализирован для chatID %d", chatID)
		return tgbotapi.Message{}, fmt.Errorf("BotClient не инициализирован")
	}
	msg := tgbotapi.NewMessage(chatID, text)
	if keyboard != nil {
		msg.ReplyMarkup = keyboard
	}
	msg.ParseMode = tgbotapi.ModeMarkdown // По умолчанию Markdown, если нужно другое, можно добавить параметр
	// Default Markdown, can add parameter if different is needed

	sentMsg, err := bh.Deps.BotClient.Send(msg)
	if err != nil {
		log.Printf("sendMessageWithKeyboard: Ошибка отправки сообщения для chatID %d: %v", chatID, err)
		return tgbotapi.Message{}, err
	}
	return sentMsg, nil
}

// NotifyOperatorsAboutDriverSettlement уведомляет операторов о новом отчете водителя.
func (bh *BotHandler) NotifyOperatorsAboutDriverSettlement(driverUser models.User, settlementID int64) {
	log.Printf("NotifyOperatorsAboutDriverSettlement: Уведомление о новом отчете #%d от водителя %s (ID: %d)", settlementID, driverUser.FirstName, driverUser.ID)

	operatorRoles := []string{constants.ROLE_OPERATOR, constants.ROLE_MAINOPERATOR, constants.ROLE_OWNER}
	operators, err := db.GetUsersByRole(operatorRoles...)
	if err != nil {
		log.Printf("NotifyOperatorsAboutDriverSettlement: Ошибка получения списка операторов: %v", err)
		return
	}

	driverName := utils.GetUserDisplayName(driverUser)
	// --- НАЧАЛО ИЗМЕНЕНИЯ ---
	messageText := fmt.Sprintf("🧾 Водитель *%s* отправил новый отчет (ID: %d) на проверку.",
		utils.EscapeTelegramMarkdown(driverName),
		settlementID,
	)

	// Добавляем кнопки для просмотра и действий
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("👁️ Проверить и решить", fmt.Sprintf("%s_%d", constants.CALLBACK_PREFIX_OPERATOR_VIEW_DRIVER_SETTLEMENT, settlementID)),
		),
	)
	// --- КОНЕЦ ИЗМЕНЕНИЯ ---

	for _, operator := range operators {
		// Не отправляем уведомление самому водителю, если он также оператор (маловероятно, но на всякий случай)
		if operator.ID == driverUser.ID {
			continue
		}
		bh.sendMessageWithKeyboard(operator.ChatID, messageText, &keyboard)
	}

	// Также можно отправить в общую группу, если это релевантно
	if bh.Deps.Config.GroupChatID != 0 {
		bh.sendMessageWithKeyboard(bh.Deps.Config.GroupChatID, messageText, &keyboard)
	}
}
