package handlers

import (
	// "database/sql" // Not used directly here, but might be needed for others
	"fmt"
	tgbotapi "github.com/OvyFlash/telegram-bot-api"
	"log"
	"time" // Added for Point 1 fix

	// "github.com/xuri/excelize/v2" // For Excel generation, not needed here

	"Original/internal/constants"
	"Original/internal/db"
	"Original/internal/models"
	// "Original/internal/session" // Access via bh.Deps
	"Original/internal/utils"
)

// --- Меню контактов и связи ---
// --- Contact and Communication Menus ---

// SendContactOperatorMenu отправляет меню выбора способа связи с оператором.
// SendContactOperatorMenu sends the menu for choosing how to contact the operator.
func (bh *BotHandler) SendContactOperatorMenu(chatID int64, user models.User, messageIDToEdit int) {
	log.Printf("BotHandler.SendContactOperatorMenu для chatID %d, messageIDToEdit: %d", chatID, messageIDToEdit)
	bh.Deps.SessionManager.SetState(chatID, constants.STATE_CONTACT_METHOD)

	msgText := "📞 Как хотите связаться с оператором? Мы всегда на связи! 😊\n\n" +
		"💡 Выберите чат для быстрого ответа или звонок для личной консультации!"
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("💬 Связь в чате", "contact_chat"),
			tgbotapi.NewInlineKeyboardButtonData("📱 Связь по телефону", "contact_phone_options"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔙 Главное меню", "back_to_main"),
		),
	)
	_, err := bh.sendOrEditMessageHelper(chatID, messageIDToEdit, msgText, &keyboard, tgbotapi.ModeMarkdown)
	if err != nil {
		log.Printf("SendContactOperatorMenu: Ошибка для chatID %d: %v", chatID, err)
	}
}

// SendPhoneOptionsMenu отправляет меню выбора действий по телефону (позвонить мне / сам позвоню).
// SendPhoneOptionsMenu sends the menu for phone action selection (call me / I'll call).
func (bh *BotHandler) SendPhoneOptionsMenu(chatID int64, messageIDToEdit int) {
	log.Printf("BotHandler.SendPhoneOptionsMenu для chatID %d, messageIDToEdit: %d", chatID, messageIDToEdit)
	bh.Deps.SessionManager.SetState(chatID, constants.STATE_PHONE_OPTIONS) // Новое состояние для этого меню / New state for this menu

	msgText := "📱 Как вам удобнее связаться с оператором?\n\n" +
		"💡 Запросите звонок, и мы свяжемся с вами за 5 минут! 😊\n"
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📲 Позвоните мне", "phone_action_request_call"),
			tgbotapi.NewInlineKeyboardButtonData("☎️ Сам позвоню", "phone_action_call_self"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔙 Назад к выбору связи", "contact_operator"),
			tgbotapi.NewInlineKeyboardButtonData("🔙 Главное меню", "back_to_main"),
		),
	)
	_, err := bh.sendOrEditMessageHelper(chatID, messageIDToEdit, msgText, &keyboard, tgbotapi.ModeMarkdown)
	if err != nil {
		log.Printf("SendPhoneOptionsMenu: Ошибка для chatID %d: %v", chatID, err)
	}
}

// SendRequestPhoneNumberPrompt запрашивает у пользователя номер телефона для обратного звонка.
// SendRequestPhoneNumberPrompt prompts the user for their phone number for a callback.
func (bh *BotHandler) SendRequestPhoneNumberPrompt(chatID int64, messageIDToEdit int) {
	log.Printf("BotHandler.SendRequestPhoneNumberPrompt для chatID %d, messageIDToEdit: %d", chatID, messageIDToEdit)
	bh.Deps.SessionManager.SetState(chatID, constants.STATE_PHONE_AWAIT_INPUT) // Ожидание ввода номера / Awaiting number input

	msgText := "📱 Пожалуйста, отправьте ваш номер телефона в формате +79991234567, чтобы мы могли вам перезвонить:\n\n" +
		"💡 Или свяжитесь с оператором другим способом, если передумали."

	// ReplyKeyboard для кнопки "Поделиться номером" / ReplyKeyboard for "Share contact" button
	replyKeyboard := tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButtonContact("📞 Поделиться моим номером из Telegram"),
		),
	)
	replyKeyboard.OneTimeKeyboard = true
	replyKeyboard.ResizeKeyboard = true

	// InlineKeyboard для кнопок "Назад" / InlineKeyboard for "Back" buttons
	inlineKeyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔙 Назад к опциям телефона", "contact_phone_options"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔙 Главное меню", "back_to_main"),
		),
	)

	// Сначала отправляем/редактируем сообщение с инлайн-клавиатурой
	// First, send/edit message with inline keyboard
	sentInlineMsg, err := bh.sendOrEditMessageHelper(chatID, messageIDToEdit, msgText, &inlineKeyboard, tgbotapi.ModeMarkdown)
	if err != nil {
		log.Printf("SendRequestPhoneNumberPrompt: Ошибка отправки/редактирования основного сообщения для chatID %d: %v", chatID, err)
		return
	}

	// Затем отправляем сообщение с ReplyKeyboard (если оно еще не отображается)
	// Then send message with ReplyKeyboard (if not already displayed)
	tempMsgConfig := tgbotapi.NewMessage(chatID, "Вы также можете использовать кнопку ниже 👇")
	tempMsgConfig.ReplyMarkup = replyKeyboard

	sentReplyMsg, errKb := bh.Deps.BotClient.Send(tempMsgConfig)
	if errKb != nil {
		log.Printf("SendRequestPhoneNumberPrompt: Ошибка отправки ReplyKeyboard для chatID %d: %v", chatID, errKb)
	} else {
		// Сохраняем ID сообщения с ReplyKeyboard, чтобы его можно было удалить
		// Save ID of message with ReplyKeyboard to delete it later
		tempData := bh.Deps.SessionManager.GetTempOrder(chatID)
		tempData.LocationPromptMessageID = sentReplyMsg.MessageID // Переиспользуем это поле / Reuse this field
		tempData.CurrentMessageID = sentInlineMsg.MessageID       // Убедимся, что CurrentMessageID - это инлайн / Ensure CurrentMessageID is inline
		bh.Deps.SessionManager.UpdateTempOrder(chatID, tempData)
	}
}

// SendOperatorContactInfo отправляет пользователю контактную информацию оператора.
// SendOperatorContactInfo sends operator contact information to the user.
func (bh *BotHandler) SendOperatorContactInfo(chatID int64, messageIDToEdit int) {
	log.Printf("BotHandler.SendOperatorContactInfo для chatID %d, messageIDToEdit: %d", chatID, messageIDToEdit)
	bh.Deps.SessionManager.SetState(chatID, constants.STATE_IDLE) // После показа контактов, состояние сбрасывается / After showing contacts, state is reset

	// --- MODIFICATION FOR POINT 1 ---
	// Ensure ReplyKeyboard is removed if it was present (e.g., from "Позвоните мне")
	tempData := bh.Deps.SessionManager.GetTempOrder(chatID)
	if tempData.LocationPromptMessageID != 0 { // LocationPromptMessageID might have been used for phone prompt message
		bh.deleteMessageHelper(chatID, tempData.LocationPromptMessageID)
		tempData.LocationPromptMessageID = 0
		// No need to update session just for this, as state is reset anyway.
		// If other fields were modified in tempData, then update.
	}
	// Send a message with ReplyKeyboardRemove to be sure.
	// This message can be very short-lived.
	replyMarkupRemove := tgbotapi.NewRemoveKeyboard(true)
	msgToRemoveKb := tgbotapi.NewMessage(chatID, constants.InvisibleMessage) // "⌨️" or similar
	msgToRemoveKb.ReplyMarkup = replyMarkupRemove

	// Send and schedule deletion of the invisible message
	if sentKbRemovalMsg, errKb := bh.Deps.BotClient.Send(msgToRemoveKb); errKb == nil {
		go func(id int) {
			time.Sleep(200 * time.Millisecond) // Brief delay
			bh.deleteMessageHelper(chatID, id)
		}(sentKbRemovalMsg.MessageID)
	} else {
		log.Printf("SendOperatorContactInfo: Ошибка при попытке убрать ReplyKeyboard: %v", errKb)
	}
	// --- END MODIFICATION FOR POINT 1 ---

	operatorName, operatorPhone, err := db.GetOperatorForContact()
	if err != nil {
		_, _ = bh.sendErrorMessageHelper(chatID, messageIDToEdit, "📞 К сожалению, не удалось получить контакты оператора. Попробуйте позже или напишите в чат.")
		return
	}

	formattedPhone := utils.FormatPhoneNumber(operatorPhone)
	msgText := fmt.Sprintf(
		"📞 Свяжитесь с оператором:\n\n👨‍💼 %s\n📱 %s\n\nЗвоните прямо сейчас, и мы решим ваш вопрос за 5 минут! 😊\n🔥 Только сегодня: получите скидку 200 ₽ на заказ после разговора! 🎁",
		utils.EscapeTelegramMarkdown(operatorName), utils.EscapeTelegramMarkdown(formattedPhone))
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔙 Назад к опциям телефона", "contact_phone_options"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔙 Главное меню", "back_to_main"),
		),
	)
	_, errSend := bh.sendOrEditMessageHelper(chatID, messageIDToEdit, msgText, &keyboard, tgbotapi.ModeMarkdown)
	if errSend != nil {
		log.Printf("SendOperatorContactInfo: Ошибка для chatID %d: %v", chatID, errSend)
	}
}

// SendChatMessageInputPrompt предлагает пользователю ввести сообщение для чата с оператором.
// SendChatMessageInputPrompt prompts the user to enter a message for the chat with the operator.
func (bh *BotHandler) SendChatMessageInputPrompt(chatID int64, messageIDToEdit int) {
	log.Printf("BotHandler.SendChatMessageInputPrompt для chatID %d, messageIDToEdit: %d", chatID, messageIDToEdit)
	bh.Deps.SessionManager.SetState(chatID, constants.STATE_CHAT_MESSAGE_INPUT)

	msgText := "💬 Напишите ваше сообщение оператору:\n\n🔥 Получите ответ в течение 5 минут! 😊"
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔙 Назад к выбору связи", "contact_operator"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔙 Главное меню", "back_to_main"),
		),
	)
	_, err := bh.sendOrEditMessageHelper(chatID, messageIDToEdit, msgText, &keyboard, tgbotapi.ModeMarkdown)
	if err != nil {
		log.Printf("SendChatMessageInputPrompt: Ошибка для chatID %d: %v", chatID, err)
	}
}

// SendChatConfirmation отправляет подтверждение после отправки сообщения в чат.
// SendChatConfirmation sends confirmation after sending a message to the chat.
func (bh *BotHandler) SendChatConfirmation(chatID int64, messageIDToEdit int) {
	log.Printf("BotHandler.SendChatConfirmation для chatID %d, messageIDToEdit: %d", chatID, messageIDToEdit)
	bh.Deps.SessionManager.SetState(chatID, constants.STATE_IDLE) // Сбрасываем состояние после отправки / Reset state after sending

	msgText := "✅ Сообщение отправлено! Хотите отправить ещё? 😊\n🔥 Получите ответ в течение 5 минут и бонус 200 ₽ за активность! 🎁"
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📩 Отправить ещё", "contact_chat"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔙 Главное меню", "back_to_main"),
		),
	)
	_, err := bh.sendOrEditMessageHelper(chatID, messageIDToEdit, msgText, &keyboard, tgbotapi.ModeMarkdown)
	if err != nil {
		log.Printf("SendChatConfirmation: Ошибка для chatID %d: %v", chatID, err)
	}
}

// SendPhoneCallRequestConfirmation подтверждает запрос на обратный звонок.
// SendPhoneCallRequestConfirmation confirms a callback request.
func (bh *BotHandler) SendPhoneCallRequestConfirmation(chatID int64, formattedPhone string, messageIDToEdit int) {
	log.Printf("BotHandler.SendPhoneCallRequestConfirmation для chatID %d, телефон: %s, messageIDToEdit: %d", chatID, formattedPhone, messageIDToEdit)
	bh.Deps.SessionManager.SetState(chatID, constants.STATE_IDLE)

	// Убираем ReplyKeyboard, если она была / Remove ReplyKeyboard if it was present
	tempData := bh.Deps.SessionManager.GetTempOrder(chatID)
	if tempData.LocationPromptMessageID != 0 { // LocationPromptMessageID используется для хранения ID сообщения с ReplyKeyboard / LocationPromptMessageID is used to store ID of message with ReplyKeyboard
		bh.deleteMessageHelper(chatID, tempData.LocationPromptMessageID)
		tempData.LocationPromptMessageID = 0
		bh.Deps.SessionManager.UpdateTempOrder(chatID, tempData)
	}
	// Также можно отправить команду на удаление клавиатуры напрямую / Can also send command to remove keyboard directly
	replyMarkup := tgbotapi.NewRemoveKeyboard(true)
	msgToRemoveKb := tgbotapi.NewMessage(chatID, constants.InvisibleMessage) // Используем "невидимое" сообщение / Use "invisible" message
	msgToRemoveKb.ReplyMarkup = replyMarkup
	if sentKbRemovalMsg, errKb := bh.Deps.BotClient.Send(msgToRemoveKb); errKb == nil {
		go func(id int) {
			time.Sleep(200 * time.Millisecond)
			bh.deleteMessageHelper(chatID, id)
		}(sentKbRemovalMsg.MessageID)
	} else {
		log.Printf("SendPhoneCallRequestConfirmation: Ошибка отправки сообщения для удаления ReplyKeyboard: %v", errKb)
	}

	msgText := fmt.Sprintf(
		"📞 Спасибо! Мы перезвоним вам на номер %s в ближайшие 5 минут! 😊\n🔥 Пока ждёте, пригласите друга и получите бонус 500 ₽! 🎁",
		utils.EscapeTelegramMarkdown(formattedPhone))
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("👥 Пригласить друга", "invite_friend"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔙 Главное меню", "back_to_main"),
		),
	)
	_, err := bh.sendOrEditMessageHelper(chatID, messageIDToEdit, msgText, &keyboard, tgbotapi.ModeMarkdown)
	if err != nil {
		log.Printf("SendPhoneCallRequestConfirmation: Ошибка для chatID %d: %v", chatID, err)
	}
}

// SendClientChatsMenu (для оператора) отображает список активных чатов с клиентами.
// SendClientChatsMenu (for operator) displays a list of active chats with clients.
func (bh *BotHandler) SendClientChatsMenu(chatID int64, messageIDToEdit int) {
	log.Printf("BotHandler.SendClientChatsMenu для оператора chatID %d, messageIDToEdit: %d", chatID, messageIDToEdit)
	bh.Deps.SessionManager.SetState(chatID, constants.STATE_OPERATOR_VIEW_CHATS)

	activeChats, err := db.GetActiveClientChats()
	if err != nil {
		log.Printf("SendClientChatsMenu: ошибка получения активных чатов: %v", err)
		_, _ = bh.sendErrorMessageHelper(chatID, messageIDToEdit, "❌ Ошибка загрузки активных чатов.")
		return
	}

	var msgText string
	var keyboard tgbotapi.InlineKeyboardMarkup
	var rows [][]tgbotapi.InlineKeyboardButton

	if len(activeChats) == 0 {
		msgText = "💬 Активных чатов с клиентами пока нет."
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("🔙 Главное меню", "back_to_main")))
	} else {
		msgText = "💬 Активные чаты с клиентами:\n\nНажмите на чат, чтобы просмотреть историю и ответить."
		for _, clientUser := range activeChats {
			name := utils.GetUserDisplayName(clientUser)
			if len(name) > 50 {
				name = name[:47] + "..."
			}
			// Коллбэк должен содержать ID клиента для открытия чата
			// Callback should contain client ID to open chat
			rows = append(rows, tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData(name, fmt.Sprintf("view_chat_history_%d", clientUser.ChatID)),
			))
		}
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("🔙 Главное меню", "back_to_main")))
	}
	keyboard.InlineKeyboard = rows

	_, errSend := bh.sendOrEditMessageHelper(chatID, messageIDToEdit, msgText, &keyboard, tgbotapi.ModeMarkdown)
	if errSend != nil {
		log.Printf("SendClientChatsMenu: Ошибка отправки для chatID %d: %v", chatID, errSend)
	}
}

// --- Меню реферальной программы ---
// --- Referral Program Menus ---

// SendInviteFriendMenu отправляет меню "Пригласить друга".
// SendInviteFriendMenu sends the "Invite a Friend" menu.
func (bh *BotHandler) SendInviteFriendMenu(chatID int64, messageIDToEdit int) {
	log.Printf("BotHandler.SendInviteFriendMenu для chatID %d, messageIDToEdit: %d", chatID, messageIDToEdit)
	bh.Deps.SessionManager.SetState(chatID, constants.STATE_INVITE_FRIEND)

	msgText := "👥 Приглашайте друзей и получайте 500 ₽ за каждый их заказ от 10 000 ₽!\n\n" +
		"🔥 Выберите способ поделиться реферальной ссылкой:"
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📱 Реферальная ссылка", "referral_link"),
			tgbotapi.NewInlineKeyboardButtonData("🔲 QR-код", "referral_qr"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("👥 Мои рефералы", "referral_my"),
			tgbotapi.NewInlineKeyboardButtonData("🔙 Главное меню", "back_to_main"),
		),
	)
	_, err := bh.sendOrEditMessageHelper(chatID, messageIDToEdit, msgText, &keyboard, tgbotapi.ModeMarkdown)
	if err != nil {
		log.Printf("SendInviteFriendMenu: Ошибка для chatID %d: %v", chatID, err)
	}
}

// SendReferralLink отправляет реферальную ссылку пользователю.
// SendReferralLink sends a referral link to the user.
func (bh *BotHandler) SendReferralLink(chatID int64, messageIDToEdit int) {
	log.Printf("BotHandler.SendReferralLink для chatID %d, messageIDToEdit: %d", chatID, messageIDToEdit)
	bh.Deps.SessionManager.SetState(chatID, constants.STATE_REFERRAL_LINK)

	link, err := utils.GenerateReferralLink(bh.Deps.Config.BotUsername, chatID)
	if err != nil {
		log.Printf("SendReferralLink: Ошибка генерации реферальной ссылки для chatID %d: %v", chatID, err)
		_, _ = bh.sendErrorMessageHelper(chatID, messageIDToEdit, "❌ Ошибка создания вашей реферальной ссылки. Попробуйте позже.")
		return
	}
	// Используем Markdown для возможности копирования ссылки по клику
	// Use Markdown for click-to-copy link functionality
	msgText := fmt.Sprintf("🔗 Ваша реферальная ссылка:\n`%s`\n\nСкопируйте и поделитесь с друзьями, чтобы получать бонусы! 🎉", link) // Экранирование не нужно для `...` / Escaping not needed for `...`
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔲 Показать QR-код", "referral_qr"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔙 Назад к 'Пригласить друга'", "invite_friend"),
			tgbotapi.NewInlineKeyboardButtonData("🔙 Главное меню", "back_to_main"),
		),
	)
	_, errSend := bh.sendOrEditMessageHelper(chatID, messageIDToEdit, msgText, &keyboard, tgbotapi.ModeMarkdown)
	if errSend != nil {
		log.Printf("SendReferralLink: Ошибка для chatID %d: %v", chatID, errSend)
	}
}

// SendReferralQRCode отправляет QR-код с реферальной ссылкой.
// SendReferralQRCode sends a QR code with the referral link.
func (bh *BotHandler) SendReferralQRCode(chatID int64, messageIDToEdit int) {
	log.Printf("BotHandler.SendReferralQRCode для chatID %d, messageIDToEdit: %d", chatID, messageIDToEdit)
	bh.Deps.SessionManager.SetState(chatID, constants.STATE_REFERRAL_QR)

	qrCodeBytes, err := utils.GenerateQRCode(bh.Deps.Config.BotUsername, chatID)
	if err != nil {
		log.Printf("SendReferralQRCode: Ошибка генерации QR-кода для chatID %d: %v", chatID, err)
		_, _ = bh.sendErrorMessageHelper(chatID, messageIDToEdit, "❌ Ошибка создания QR-кода. Попробуйте позже.")
		return
	}

	// Удаляем предыдущее сообщение (например, с текстовой ссылкой или меню выбора), если оно было и это не оно же
	// Delete previous message (e.g., with text link or selection menu) if it existed and is not the same
	if messageIDToEdit != 0 {
		bh.deleteMessageHelper(chatID, messageIDToEdit)
	}

	photoFileBytes := tgbotapi.FileBytes{
		Name:  "referral_qr.png",
		Bytes: qrCodeBytes,
	}
	photoMsg := tgbotapi.NewPhoto(chatID, photoFileBytes)
	photoMsg.Caption = "🔲 Ваш реферальный QR-код.\nПокажите его друзьям для сканирования!\nПриглашайте и зарабатывайте бонусы! 🎉"
	photoMsg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📱 Показать текстовую ссылку", "referral_link"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔙 Назад к 'Пригласить друга'", "invite_friend"),
			tgbotapi.NewInlineKeyboardButtonData("🔙 Главное меню", "back_to_main"),
		),
	)

	sentMsg, errSend := bh.Deps.BotClient.Send(photoMsg) // Отправляем новое сообщение с фото / Send new message with photo
	if errSend != nil {
		log.Printf("SendReferralQRCode: Ошибка отправки QR-кода для chatID %d: %v", chatID, errSend)
		// Отправляем новое сообщение об ошибке, так как messageIDToEdit мог быть удален
		// Send new error message as messageIDToEdit might have been deleted
		bh.sendErrorMessageHelper(chatID, 0, "❌ Не удалось отправить QR-код.")
		return
	}
	// Обновляем CurrentMessageID в сессии на ID отправленного фото
	// Update CurrentMessageID in session to the ID of the sent photo
	orderData := bh.Deps.SessionManager.GetTempOrder(chatID)
	orderData.CurrentMessageID = sentMsg.MessageID
	orderData.MediaMessageIDs = []int{sentMsg.MessageID} // Это сообщение теперь главное / This message is now the main one
	orderData.MediaMessageIDsMap = make(map[string]bool) // Очищаем карту / Clear map
	orderData.MediaMessageIDsMap[fmt.Sprintf("%d", sentMsg.MessageID)] = true
	bh.Deps.SessionManager.UpdateTempOrder(chatID, orderData)
}

// SendMyReferralsMenu отображает список приглашенных пользователей и сумму бонусов.
// SendMyReferralsMenu displays a list of invited users and bonus amounts.
func (bh *BotHandler) SendMyReferralsMenu(chatID int64, messageIDToEdit int) {
	log.Printf("BotHandler.SendMyReferralsMenu для chatID %d, messageIDToEdit: %d", chatID, messageIDToEdit)
	bh.Deps.SessionManager.SetState(chatID, constants.STATE_MY_REFERRALS)

	referrals, err := db.GetReferralsByInviterChatID(chatID)
	if err != nil {
		log.Printf("SendMyReferralsMenu: Ошибка получения рефералов для chatID %d: %v", chatID, err)
		_, _ = bh.sendErrorMessageHelper(chatID, messageIDToEdit, "❌ Ошибка загрузки ваших рефералов.")
		return
	}

	var msgText string
	var keyboard tgbotapi.InlineKeyboardMarkup
	var rows [][]tgbotapi.InlineKeyboardButton

	if len(referrals) == 0 {
		msgText = "👥 У вас пока нет приглашенных друзей, которые сделали заказ и принесли вам бонус.\n\nПродолжайте делиться вашей ссылкой!"
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("🔗 Поделиться ссылкой", "referral_link")))
	} else {
		msgText = "👥 Ваши успешные рефералы (друзья, сделавшие заказ и принесшие бонус):\n"
		totalBonus := 0.0
		unpaidBonus := 0.0
		hasUnpaidAndNotRequested := false // Флаг для доступных к запросу бонусов / Flag for bonuses available for request
		for _, r := range referrals {
			dateStr := r.CreatedAt.Format("02.01.2006")
			statusStr := ""
			if r.PaidOut {
				statusStr = " (выплачено)"
			} else {
				if r.PayoutRequestID.Valid {
					statusStr = " (в запросе на выплату)"
				} else {
					unpaidBonus += r.Amount // Суммируем только те, что не выплачены и не в запросе / Sum only those not paid and not in request
					hasUnpaidAndNotRequested = true
				}
			}
			// Отображаем имя приглашенного (r.Name уже содержит ФИО) / Display invitee's name (r.Name already contains full name)
			// POINT 10: Format bonus amount
			rows = append(rows, tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData(fmt.Sprintf("%s (%s) - Бонус: %.0f ₽%s", r.Name, dateStr, r.Amount, statusStr), fmt.Sprintf("referral_details_%d", r.ID)),
			))
			totalBonus += r.Amount // Общий заработанный бонус / Total earned bonus
		}
		// POINT 10: Format total and unpaid bonus amounts
		msgText += fmt.Sprintf("\nОбщий заработанный бонус: *%.0f ₽*", totalBonus)
		if hasUnpaidAndNotRequested && unpaidBonus > 0 {
			msgText += fmt.Sprintf("\nК выплате доступно: *%.0f ₽*", unpaidBonus)
			rows = append(rows, tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("💸 Запросить выплату доступных бонусов", "request_referral_payout")))
		} else if totalBonus > 0 {
			msgText += "\nВсе доступные бонусы выплачены или находятся в обработке."
		}
	}

	rows = append(rows, tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("🔙 Назад к 'Пригласить друга'", "invite_friend")))
	rows = append(rows, tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("🔙 Главное меню", "back_to_main")))
	keyboard.InlineKeyboard = rows

	_, errSend := bh.sendOrEditMessageHelper(chatID, messageIDToEdit, msgText, &keyboard, tgbotapi.ModeMarkdown)
	if errSend != nil {
		log.Printf("SendMyReferralsMenu: Ошибка для chatID %d: %v", chatID, errSend)
	}
}

// SendReferralPayoutConfirmation подтверждает запрос на выплату реферальных бонусов.
// Добавлен requestID для информации.
// SendReferralPayoutConfirmation confirms a referral bonus payout request.
// requestID added for information.
func (bh *BotHandler) SendReferralPayoutConfirmation(chatID int64, messageIDToEdit int, amount float64, requestID int64) {
	log.Printf("BotHandler.SendReferralPayoutConfirmation для chatID %d, сумма %.0f, ID запроса %d, messageIDToEdit: %d", chatID, amount, requestID, messageIDToEdit)
	bh.Deps.SessionManager.SetState(chatID, constants.STATE_IDLE) // Сбрасываем состояние / Reset state

	// POINT 10: Format amount
	msgText := fmt.Sprintf("💸 Ваш запрос №%d на выплату реферальных бонусов на сумму %.0f ₽ отправлен администратору!\n\nМы свяжемся с вами для уточнения деталей выплаты. Спасибо!", requestID, amount)
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("👥 Мои рефералы", "referral_my")), // Кнопка для возврата к списку рефералов / Button to return to referral list
		tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("🔙 Главное меню", "back_to_main")),
	)
	_, err := bh.sendOrEditMessageHelper(chatID, messageIDToEdit, msgText, &keyboard, tgbotapi.ModeMarkdown)
	if err != nil {
		log.Printf("SendReferralPayoutConfirmation: Ошибка для chatID %d: %v", chatID, err)
	}
}

// --- Прочие информационные сообщения ---
// --- Other informational messages ---

// SendMaterialsSoonInfo сообщает, что раздел стройматериалов скоро будет доступен.
// SendMaterialsSoonInfo informs that the construction materials section will be available soon.
func (bh *BotHandler) SendMaterialsSoonInfo(chatID int64, messageIDToEdit int) {
	log.Printf("BotHandler.SendMaterialsSoonInfo для chatID %d, messageIDToEdit: %d", chatID, messageIDToEdit)
	// Состояние можно не менять или установить специфичное, например, STATE_INFO_VIEW
	// State can remain unchanged or set to specific, e.g., STATE_INFO_VIEW
	// bh.Deps.SessionManager.SetState(chatID, constants.STATE_INFO_MATERIALS)

	msgText := "🧱 Раздел 'Стройматериалы' скоро будет доступен! 🚛\n\n" +
		"🔥 Подпишитесь на уведомления и получите скидку 500 ₽ на первый заказ материалов! 🎁"
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔔 Подписаться на уведомления", "subscribe_materials_updates"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔙 Главное меню", "back_to_main"),
		),
	)
	_, err := bh.sendOrEditMessageHelper(chatID, messageIDToEdit, msgText, &keyboard, tgbotapi.ModeMarkdown)
	if err != nil {
		log.Printf("SendMaterialsSoonInfo: Ошибка для chatID %d: %v", chatID, err)
	}
}

// SendSubscriptionConfirmation подтверждает подписку на уведомления.
// SendSubscriptionConfirmation confirms subscription to notifications.
func (bh *BotHandler) SendSubscriptionConfirmation(chatID int64, serviceName string, messageIDToEdit int) {
	log.Printf("BotHandler.SendSubscriptionConfirmation для chatID %d, сервис: %s, messageIDToEdit: %d", chatID, serviceName, messageIDToEdit)
	bh.Deps.SessionManager.SetState(chatID, constants.STATE_IDLE) // Сбрасываем состояние / Reset state

	msgText := fmt.Sprintf("🔔 Вы успешно подписаны на уведомления о '%s'!\n\n"+
		"🔥 Мы сообщим вам, как только появятся новости или раздел станет доступен. Вы также получите обещанный бонус! 🎁", serviceName)
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔙 Главное меню", "back_to_main"),
		),
	)
	_, err := bh.sendOrEditMessageHelper(chatID, messageIDToEdit, msgText, &keyboard, tgbotapi.ModeMarkdown)
	if err != nil {
		log.Printf("SendSubscriptionConfirmation: Ошибка для chatID %d: %v", chatID, err)
	}
}

// SendReferralDetails отображает детали конкретного реферала.
// SendReferralDetails displays details of a specific referral.
func (bh *BotHandler) SendReferralDetails(chatID int64, referralID int64, messageIDToEdit int) {
	log.Printf("BotHandler.SendReferralDetails для chatID %d, referralID: %d, messageIDToEdit: %d", chatID, referralID, messageIDToEdit)
	bh.Deps.SessionManager.SetState(chatID, constants.STATE_MY_REFERRALS) // Остаемся в контексте "Мои рефералы" / Remain in "My Referrals" context

	referral, err := db.GetReferralByID(referralID, chatID) // chatID здесь - это chatID пользователя, который нажал кнопку / chatID here is the chatID of the user who pressed the button
	if err != nil {
		log.Printf("SendReferralDetails: Ошибка получения реферала #%d для chatID %d: %v", referralID, chatID, err)
		_, _ = bh.sendErrorMessageHelper(chatID, messageIDToEdit, "❌ Ошибка загрузки деталей реферала или у вас нет доступа.")
		bh.SendMyReferralsMenu(chatID, messageIDToEdit) // Возврат в меню "Мои рефералы" / Return to "My Referrals" menu
		return
	}

	statusText := "Ожидает выплаты"
	if referral.PaidOut {
		statusText = "Выплачено"
	} else if referral.PayoutRequestID.Valid { // Проверяем, есть ли ID запроса / Check if request ID exists
		statusText = "В запросе на выплату"
	}

	// POINT 10: Format bonus amount
	msgText := fmt.Sprintf(
		"👥 Детали по рефералу:\n\n"+
			"Приглашенный: *%s*\n"+
			"Сумма бонуса: *%.0f ₽*\n"+
			"Дата регистрации заказа реферала: *%s*\n"+
			"ID Заказа реферала: *%d*\n"+
			"Статус выплаты: *%s*",
		utils.EscapeTelegramMarkdown(referral.Name),
		referral.Amount,
		utils.EscapeTelegramMarkdown(referral.CreatedAt.Format("02.01.2006")),
		referral.OrderID,
		utils.EscapeTelegramMarkdown(statusText),
	)

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔙 К списку моих рефералов", "referral_my"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔙 Главное меню", "back_to_main"),
		),
	)

	_, err = bh.sendOrEditMessageHelper(chatID, messageIDToEdit, msgText, &keyboard, tgbotapi.ModeMarkdown)
	if err != nil {
		log.Printf("SendReferralDetails: Ошибка для chatID %d: %v", chatID, err)
	}
}
