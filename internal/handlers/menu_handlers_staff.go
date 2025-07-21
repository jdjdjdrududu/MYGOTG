package handlers

import (
	"fmt"
	"log"
	"strings"
	// "time" // Not used directly here

	tgbotapi "github.com/OvyFlash/telegram-bot-api"

	// "github.com/xuri/excelize/v2" // Not used here

	"Original/internal/constants"
	"Original/internal/db"
	// "Original/internal/session" // Access via bh.Deps
	"Original/internal/utils"
)

// SendStaffMenu отправляет главное меню управления персоналом.
// SendStaffMenu sends the main staff management menu.
func (bh *BotHandler) SendStaffMenu(chatID int64, messageIDToEdit int) {
	log.Printf("BotHandler.SendStaffMenu для chatID %d, messageIDToEdit: %d", chatID, messageIDToEdit)
	bh.Deps.SessionManager.SetState(chatID, constants.STATE_STAFF_MENU)
	// Очищаем временные данные сотрудника при входе в главное меню штата
	// Clear temporary staff data when entering the main staff menu
	bh.Deps.SessionManager.ClearTempOrder(chatID)

	msgText := "👷 Управление штатом:\n\nВыберите действие:"
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📋 Список сотрудников", "staff_list_menu"),
			tgbotapi.NewInlineKeyboardButtonData("➕ Добавить сотрудника", "staff_add_prompt_name"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔙 Главное меню", "back_to_main"),
		),
	)
	_, err := bh.sendOrEditMessageHelper(chatID, messageIDToEdit, msgText, &keyboard, tgbotapi.ModeMarkdown)
	if err != nil {
		log.Printf("SendStaffMenu: Ошибка для chatID %d: %v", chatID, err)
	}
}

// SendStaffListMenu отправляет меню выбора категории сотрудников для просмотра.
// SendStaffListMenu sends the staff category selection menu for viewing.
func (bh *BotHandler) SendStaffListMenu(chatID int64, messageIDToEdit int) {
	log.Printf("BotHandler.SendStaffListMenu для chatID %d, messageIDToEdit: %d", chatID, messageIDToEdit)
	bh.Deps.SessionManager.SetState(chatID, constants.STATE_STAFF_LIST)

	msgText := "📋 Выберите категорию сотрудников для просмотра:"
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(utils.GetRoleDisplayName(constants.ROLE_MAINOPERATOR), fmt.Sprintf("staff_list_by_role_%s", constants.ROLE_MAINOPERATOR)),
			tgbotapi.NewInlineKeyboardButtonData(utils.GetRoleDisplayName(constants.ROLE_OPERATOR), fmt.Sprintf("staff_list_by_role_%s", constants.ROLE_OPERATOR)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(utils.GetRoleDisplayName(constants.ROLE_DRIVER), fmt.Sprintf("staff_list_by_role_%s", constants.ROLE_DRIVER)),
			tgbotapi.NewInlineKeyboardButtonData(utils.GetRoleDisplayName(constants.ROLE_LOADER), fmt.Sprintf("staff_list_by_role_%s", constants.ROLE_LOADER)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(fmt.Sprintf(utils.GetRoleDisplayName(constants.ROLE_USER)), fmt.Sprintf("staff_list_by_role_%s", constants.ROLE_USER)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔙 Назад в меню штата", "staff_menu"),
			tgbotapi.NewInlineKeyboardButtonData("🔙 Главное меню", "back_to_main"),
		),
	)
	_, err := bh.sendOrEditMessageHelper(chatID, messageIDToEdit, msgText, &keyboard, tgbotapi.ModeMarkdown)
	if err != nil {
		log.Printf("SendStaffListMenu: Ошибка для chatID %d: %v", chatID, err)
	}
}

// SendStaffList отображает список сотрудников по роли.
// SendStaffList displays a list of staff members by role.
func (bh *BotHandler) SendStaffList(chatID int64, role string, messageIDToEdit int) {
	log.Printf("BotHandler.SendStaffList для chatID %d, роль: %s, messageIDToEdit: %d", chatID, role, messageIDToEdit)

	staff, err := db.GetStaffListByRole(role) // Эта функция теперь должна возвращать и CardNumber (дешифрованный)
	// This function should now also return CardNumber (decrypted)
	if err != nil {
		bh.sendErrorMessageHelper(chatID, messageIDToEdit, "❌ Ошибка загрузки списка сотрудников.")
		return
	}

	roleDisplay := utils.GetRoleDisplayName(role)
	msgText := fmt.Sprintf("📋 Список: %s\n", roleDisplay)
	var rows [][]tgbotapi.InlineKeyboardButton

	if len(staff) == 0 {
		msgText += "\nСотрудников в этой категории нет."
	} else {
		for _, s := range staff {
			displayName := utils.GetUserDisplayName(s) // Используем GetUserDisplayName
			phoneDisplay := "тел. не указан"
			if s.Phone.Valid && s.Phone.String != "" {
				phoneDisplay = utils.FormatPhoneNumber(s.Phone.String)
			}

			buttonText := fmt.Sprintf("%s - %s", displayName, phoneDisplay)
			if len(buttonText) > 60 { // Ограничение Telegram на длину текста кнопки / Telegram button text length limit
				buttonText = buttonText[:57] + "..."
			}

			rows = append(rows, tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData(buttonText, fmt.Sprintf("staff_info_%d", s.ChatID)),
			))
		}
	}
	rows = append(rows, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("🔙 К выбору категории штата", "staff_list_menu"),
		tgbotapi.NewInlineKeyboardButtonData("🔙 Главное меню", "back_to_main"),
	))
	keyboard := tgbotapi.NewInlineKeyboardMarkup(rows...)

	_, errSend := bh.sendOrEditMessageHelper(chatID, messageIDToEdit, msgText, &keyboard, "")
	if errSend != nil {
		log.Printf("SendStaffList: Ошибка для chatID %d: %v", chatID, errSend)
	}
}

// SendStaffInfo отображает информацию о сотруднике и опции управления.
// SendStaffInfo displays staff member information and management options.
func (bh *BotHandler) SendStaffInfo(chatID int64, targetChatID int64, messageIDToEdit int) {
	log.Printf("BotHandler.SendStaffInfo для chatID %d, целевой chatID: %d, messageIDToEdit: %d", chatID, targetChatID, messageIDToEdit)
	bh.Deps.SessionManager.SetState(chatID, constants.STATE_STAFF_INFO)
	// Очищаем временные данные сотрудника, так как мы просматриваем конкретного
	// Clear temporary staff data as we are viewing a specific member
	bh.Deps.SessionManager.ClearTempOrder(chatID)

	targetUser, err := db.GetUserByChatID(targetChatID) // db.GetUserByChatID теперь дешифрует карту / db.GetUserByChatID now decrypts the card
	if err != nil {
		bh.sendErrorMessageHelper(chatID, messageIDToEdit, "❌ Ошибка загрузки данных сотрудника.")
		return
	}

	status := "Активен"
	if targetUser.IsBlocked {
		status = fmt.Sprintf("🚫 Заблокирован (Причина: %s)", targetUser.BlockReason.String)
	}
	phone := "не указан"
	if targetUser.Phone.Valid && targetUser.Phone.String != "" {
		phone = utils.FormatPhoneNumber(targetUser.Phone.String)
	}
	nickname := "не указан"
	if targetUser.Nickname.Valid && targetUser.Nickname.String != "" {
		nickname = targetUser.Nickname.String
	}
	cardNumberDisplay := "не указан"
	if targetUser.CardNumber.Valid && targetUser.CardNumber.String != "" {
		// Отображаем полный номер карты для копирования, обернутый в ` ` для Markdown
		// Display full card number for copying, wrapped in ` ` for Markdown
		cardNumberDisplay = fmt.Sprintf("`%s` (нажмите для копирования)", utils.EscapeTelegramMarkdown(targetUser.CardNumber.String))
	}

	msgText := fmt.Sprintf(
		"👤 Сотрудник: *%s %s*\n"+
			"Позывной: *%s*\n"+
			"Telegram ChatID: `%d`\n"+
			"Телефон: *%s*\n"+
			"Карта для выплат: %s\n"+ // Изменено для отображения номера карты / Changed to display card number
			"Роль: *%s*\n"+
			"Статус: *%s*",
		utils.EscapeTelegramMarkdown(targetUser.FirstName), utils.EscapeTelegramMarkdown(targetUser.LastName),
		utils.EscapeTelegramMarkdown(nickname),
		targetUser.ChatID,
		utils.EscapeTelegramMarkdown(phone),
		cardNumberDisplay, // Не используем EscapeTelegramMarkdown, так как уже обернули в ` ` / Do not use EscapeTelegramMarkdown as it's already wrapped in ` `
		utils.EscapeTelegramMarkdown(utils.GetRoleDisplayName(targetUser.Role)),
		utils.EscapeTelegramMarkdown(status),
	)

	var rows [][]tgbotapi.InlineKeyboardButton
	rows = append(rows, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("✏️ Изменить данные", fmt.Sprintf("staff_edit_menu_%d", targetChatID)),
	))
	if targetUser.IsBlocked {
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("🔓 Разблокировать", fmt.Sprintf("staff_unblock_confirm_%d", targetChatID))))
	} else {
		if targetUser.Role != constants.ROLE_OWNER { // Владельца нельзя заблокировать / Owner cannot be blocked
			rows = append(rows, tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("🔒 Заблокировать", fmt.Sprintf("staff_block_reason_prompt_%d", targetChatID))))
		}
	}
	if targetUser.Role != constants.ROLE_OWNER { // Владельца нельзя "удалить" (сменить роль на user) / Owner cannot be "deleted" (role changed to user)
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("🗑️ Удалить (сделать пользователем)", fmt.Sprintf("staff_delete_confirm_%d", targetChatID))))
	}
	rows = append(rows, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData(fmt.Sprintf("🔙 К списку (%s)", utils.GetRoleDisplayName(targetUser.Role)), fmt.Sprintf("staff_list_by_role_%s", targetUser.Role)),
		tgbotapi.NewInlineKeyboardButtonData("🔙 Главное меню", "back_to_main"),
	))
	keyboard := tgbotapi.NewInlineKeyboardMarkup(rows...)

	_, errSend := bh.sendOrEditMessageHelper(chatID, messageIDToEdit, msgText, &keyboard, tgbotapi.ModeMarkdown)
	if errSend != nil {
		log.Printf("SendStaffInfo: Ошибка для chatID %d: %v", chatID, errSend)
	}
}

// SendStaffAddPrompt запрашивает поле для добавления сотрудника.
// SendStaffAddPrompt prompts for a field to add a staff member.
func (bh *BotHandler) SendStaffAddPrompt(chatID int64, stateToSet string, promptText string, prevStateCallbackKey string, messageIDToEdit int) {
	log.Printf("BotHandler.SendStaffAddPrompt для chatID %d, состояние: %s, messageIDToEdit: %d", chatID, stateToSet, messageIDToEdit)
	bh.Deps.SessionManager.SetState(chatID, stateToSet)

	// Сохраняем ID сообщения, чтобы его можно было использовать для редактирования на следующем шаге
	// Save message ID to use for editing in the next step
	tempData := bh.Deps.SessionManager.GetTempOrder(chatID)
	tempData.CurrentMessageID = messageIDToEdit // Важно для корректной работы "Назад" и редактирования / Important for "Back" and editing to work correctly
	bh.Deps.SessionManager.UpdateTempOrder(chatID, tempData)

	// Определяем текст кнопки "Назад" / Determine "Back" button text
	backButtonText := "⬅️ Назад"
	if specificBackText, ok := utils.GetBackText(prevStateCallbackKey); ok {
		backButtonText = specificBackText
	}

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(backButtonText, prevStateCallbackKey),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🚫 Отменить добавление", "staff_menu"),
		),
	)
	_, err := bh.sendOrEditMessageHelper(chatID, messageIDToEdit, promptText, &keyboard, tgbotapi.ModeMarkdown)
	if err != nil {
		log.Printf("SendStaffAddPrompt: Ошибка для chatID %d, состояние %s: %v", chatID, stateToSet, err)
	}
}

// SendStaffEditMenu отображает меню выбора поля для редактирования сотрудника.
// SendStaffEditMenu displays the field selection menu for editing a staff member.
func (bh *BotHandler) SendStaffEditMenu(chatID int64, targetChatID int64, messageIDToEdit int) {
	log.Printf("BotHandler.SendStaffEditMenu для chatID %d, редактируется сотрудник %d, messageIDToEdit: %d", chatID, targetChatID, messageIDToEdit)
	bh.Deps.SessionManager.SetState(chatID, constants.STATE_STAFF_EDIT)

	// Сохраняем targetChatID в сессию для последующих шагов редактирования
	// Save targetChatID in session for subsequent editing steps
	tempData := bh.Deps.SessionManager.GetTempOrder(chatID)
	tempData.BlockTargetChatID = targetChatID // Используем BlockTargetChatID для хранения ID редактируемого сотрудника / Use BlockTargetChatID to store ID of staff member being edited
	tempData.CurrentMessageID = messageIDToEdit
	bh.Deps.SessionManager.UpdateTempOrder(chatID, tempData)

	targetUser, err := db.GetUserByChatID(targetChatID)
	if err != nil {
		bh.sendErrorMessageHelper(chatID, messageIDToEdit, "❌ Ошибка загрузки данных сотрудника для редактирования.")
		return
	}

	msgText := fmt.Sprintf("✏️ Редактирование сотрудника: *%s %s*\n(Позывной: *%s*, ChatID: `%d`)\n\nВыберите поле для изменения:",
		utils.EscapeTelegramMarkdown(targetUser.FirstName), utils.EscapeTelegramMarkdown(targetUser.LastName),
		utils.EscapeTelegramMarkdown(targetUser.Nickname.String), targetUser.ChatID)

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Имя", fmt.Sprintf("staff_edit_field_name_%d", targetChatID)),
			tgbotapi.NewInlineKeyboardButtonData("Фамилия", fmt.Sprintf("staff_edit_field_surname_%d", targetChatID)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Позывной", fmt.Sprintf("staff_edit_field_nickname_%d", targetChatID)),
			tgbotapi.NewInlineKeyboardButtonData("Телефон", fmt.Sprintf("staff_edit_field_phone_%d", targetChatID)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("💳 Карта", fmt.Sprintf("staff_edit_field_card_number_%d", targetChatID)), // Новая кнопка / New button
			tgbotapi.NewInlineKeyboardButtonData("Роль", fmt.Sprintf("staff_edit_field_role_%d", targetChatID)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔙 Назад к инфо о сотруднике", fmt.Sprintf("staff_info_%d", targetChatID)),
		),
	)
	_, errSend := bh.sendOrEditMessageHelper(chatID, messageIDToEdit, msgText, &keyboard, tgbotapi.ModeMarkdown)
	if errSend != nil {
		log.Printf("SendStaffEditMenu: Ошибка для chatID %d: %v", chatID, errSend)
	}
}

// SendStaffEditFieldPrompt запрашивает новое значение для конкретного поля сотрудника.
// SendStaffEditFieldPrompt prompts for a new value for a specific staff member field.
func (bh *BotHandler) SendStaffEditFieldPrompt(chatID int64, targetChatID int64, fieldToEdit string, promptText string, messageIDToEdit int) {
	log.Printf("BotHandler.SendStaffEditFieldPrompt для chatID %d, сотрудник %d, поле %s, messageIDToEdit: %d", chatID, targetChatID, fieldToEdit, messageIDToEdit)

	var stateToSet string
	switch fieldToEdit {
	case "name":
		stateToSet = constants.STATE_STAFF_EDIT_NAME
	case "surname":
		stateToSet = constants.STATE_STAFF_EDIT_SURNAME
	case "nickname":
		stateToSet = constants.STATE_STAFF_EDIT_NICKNAME
	case "phone":
		stateToSet = constants.STATE_STAFF_EDIT_PHONE
	case "card_number": // Новое состояние / New state
		stateToSet = constants.STATE_STAFF_EDIT_CARD_NUMBER
	// "role" обрабатывается через SendStaffRoleSelectionMenu / "role" is handled via SendStaffRoleSelectionMenu
	default:
		log.Printf("SendStaffEditFieldPrompt: Неизвестное поле для редактирования '%s'", fieldToEdit)
		bh.sendErrorMessageHelper(chatID, messageIDToEdit, "❌ Неизвестное поле для редактирования.")
		bh.SendStaffEditMenu(chatID, targetChatID, messageIDToEdit) // Возвращаем в меню редактирования этого сотрудника / Return to this staff member's edit menu
		return
	}
	bh.Deps.SessionManager.SetState(chatID, stateToSet)

	// Убедимся, что targetChatID и CurrentMessageID сохранены в сессии
	// Ensure targetChatID and CurrentMessageID are saved in the session
	tempData := bh.Deps.SessionManager.GetTempOrder(chatID)
	tempData.BlockTargetChatID = targetChatID
	tempData.CurrentMessageID = messageIDToEdit
	bh.Deps.SessionManager.UpdateTempOrder(chatID, tempData)

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("⬅️ Назад к выбору поля", fmt.Sprintf("staff_edit_menu_%d", targetChatID)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🚫 Отменить редактирование", fmt.Sprintf("staff_info_%d", targetChatID)),
		),
	)
	_, err := bh.sendOrEditMessageHelper(chatID, messageIDToEdit, promptText, &keyboard, tgbotapi.ModeMarkdown)
	if err != nil {
		log.Printf("SendStaffEditFieldPrompt: Ошибка для chatID %d: %v", chatID, err)
	}
}

// SendStaffRoleSelectionMenu предлагает выбрать роль для нового или редактируемого сотрудника.
// SendStaffRoleSelectionMenu offers to select a role for a new or edited staff member.
func (bh *BotHandler) SendStaffRoleSelectionMenu(chatID int64, contextPrefix string, messageIDToEdit int, backCallbackKey string) {
	log.Printf("BotHandler.SendStaffRoleSelectionMenu для chatID %d, контекст: %s, messageIDToEdit: %d", chatID, contextPrefix, messageIDToEdit)

	var stateToSet string
	if strings.HasPrefix(contextPrefix, "staff_add_role_final") { // staff_add_role_final_TARGETCHATID_ROLE
		stateToSet = constants.STATE_STAFF_ADD_ROLE
	} else if strings.HasPrefix(contextPrefix, "staff_edit_role_final") { // staff_edit_role_final_TARGETCHATID_ROLE
		stateToSet = constants.STATE_STAFF_EDIT_ROLE
	} else {
		log.Printf("SendStaffRoleSelectionMenu: Неизвестный contextPrefix: %s", contextPrefix)
		bh.sendErrorMessageHelper(chatID, messageIDToEdit, "Ошибка выбора роли.")
		return
	}
	bh.Deps.SessionManager.SetState(chatID, stateToSet)

	// Сохраняем ID сообщения для редактирования / Save message ID for editing
	tempData := bh.Deps.SessionManager.GetTempOrder(chatID)
	tempData.CurrentMessageID = messageIDToEdit
	bh.Deps.SessionManager.UpdateTempOrder(chatID, tempData)

	msgText := "👷 Выберите роль сотрудника:"

	// Формируем callback для роли, извлекая targetChatID из contextPrefix если это редактирование
	// Form callback for role, extracting targetChatID from contextPrefix if editing
	var targetChatIDForCallback string
	if strings.HasPrefix(contextPrefix, "staff_edit_role_final_") {
		parts := strings.Split(contextPrefix, "_") // staff_edit_role_final_TARGETCHATID
		if len(parts) == 5 {                       // 0:staff, 1:edit, 2:role, 3:final, 4:TARGETCHATID
			targetChatIDForCallback = parts[4]
		}
	}

	var rows [][]tgbotapi.InlineKeyboardButton
	roles := []string{
		constants.ROLE_MAINOPERATOR, constants.ROLE_OPERATOR,
		constants.ROLE_DRIVER, constants.ROLE_LOADER, constants.ROLE_USER,
	}

	for _, role := range roles {
		callbackData := ""
		if targetChatIDForCallback != "" { // Редактирование существующего / Editing existing
			callbackData = fmt.Sprintf("staff_edit_role_final_%s_%s", targetChatIDForCallback, role)
		} else { // Добавление нового / Adding new
			callbackData = fmt.Sprintf("staff_add_role_final_%s", role)
		}
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(utils.GetRoleDisplayName(role), callbackData),
		))
	}

	rows = append(rows, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("⬅️ Назад", backCallbackKey),
	))
	keyboard := tgbotapi.NewInlineKeyboardMarkup(rows...)

	_, err := bh.sendOrEditMessageHelper(chatID, messageIDToEdit, msgText, &keyboard, tgbotapi.ModeMarkdown)
	if err != nil {
		log.Printf("SendStaffRoleSelectionMenu: Ошибка для chatID %d: %v", chatID, err)
	}
}

// SendStaffActionConfirmation подтверждает действие над сотрудником (добавление, обновление).
// SendStaffActionConfirmation confirms an action on a staff member (add, update).
func (bh *BotHandler) SendStaffActionConfirmation(chatID int64, messageText string, messageIDToEdit int, targetChatIDIfAvailable int64) {
	log.Printf("BotHandler.SendStaffActionConfirmation для chatID %d: %s, messageIDToEdit: %d", chatID, messageText, messageIDToEdit)
	bh.Deps.SessionManager.SetState(chatID, constants.STATE_STAFF_MENU) // Возвращаем в меню штата / Return to staff menu
	bh.Deps.SessionManager.ClearTempOrder(chatID)                       // Очищаем временные данные после завершения операции / Clear temporary data after operation completion

	var rows [][]tgbotapi.InlineKeyboardButton
	rows = append(rows, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("📋 К списку сотрудников", "staff_list_menu"),
	))
	if targetChatIDIfAvailable != 0 {
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("👤 К карточке сотрудника", fmt.Sprintf("staff_info_%d", targetChatIDIfAvailable)),
		))
	}
	rows = append(rows, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("➕ Добавить еще", "staff_add_prompt_name"),
		tgbotapi.NewInlineKeyboardButtonData("🔙 В меню штата", "staff_menu"),
	))
	keyboard := tgbotapi.NewInlineKeyboardMarkup(rows...)

	_, err := bh.sendOrEditMessageHelper(chatID, messageIDToEdit, messageText, &keyboard, tgbotapi.ModeMarkdown)
	if err != nil {
		log.Printf("SendStaffActionConfirmation: Ошибка для chatID %d: %v", chatID, err)
	}
}

// SendStaffBlockReasonInput запрашивает причину блокировки сотрудника.
// SendStaffBlockReasonInput prompts for the reason for blocking a staff member.
func (bh *BotHandler) SendStaffBlockReasonInput(chatID int64, targetStaffID int64, messageIDToEdit int) {
	log.Printf("BotHandler.SendStaffBlockReasonInput для chatID %d, цель staffID: %d, messageIDToEdit: %d", chatID, targetStaffID, messageIDToEdit)
	bh.Deps.SessionManager.SetState(chatID, constants.STATE_STAFF_BLOCK_REASON)

	tempData := bh.Deps.SessionManager.GetTempOrder(chatID)
	tempData.BlockTargetChatID = targetStaffID // Сохраняем ID сотрудника для блокировки / Save staff ID for blocking
	tempData.CurrentMessageID = messageIDToEdit
	bh.Deps.SessionManager.UpdateTempOrder(chatID, tempData)

	targetUser, err := db.GetUserByChatID(targetStaffID)
	if err != nil {
		bh.sendErrorMessageHelper(chatID, messageIDToEdit, "Не удалось найти сотрудника для блокировки.")
		return
	}

	msgText := fmt.Sprintf("🚫 Укажите причину блокировки сотрудника *%s %s* (ChatID: `%d`):",
		utils.EscapeTelegramMarkdown(targetUser.FirstName),
		utils.EscapeTelegramMarkdown(targetUser.LastName),
		targetStaffID)
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("⬅️ Назад к инфо о сотруднике", fmt.Sprintf("staff_info_%d", targetStaffID)),
		),
	)
	_, errSend := bh.sendOrEditMessageHelper(chatID, messageIDToEdit, msgText, &keyboard, tgbotapi.ModeMarkdown)
	if errSend != nil {
		log.Printf("SendStaffBlockReasonInput: Ошибка для chatID %d: %v", chatID, errSend)
	}
}

// SendStaffUnblockConfirm запрашивает подтверждение разблокировки сотрудника.
// SendStaffUnblockConfirm prompts for confirmation to unblock a staff member.
func (bh *BotHandler) SendStaffUnblockConfirm(chatID int64, targetStaffID int64, messageIDToEdit int) {
	log.Printf("BotHandler.SendStaffUnblockConfirm для chatID %d, цель staffID: %d, messageIDToEdit: %d", chatID, targetStaffID, messageIDToEdit)
	// Состояние не меняем, это диалог подтверждения / Do not change state, this is a confirmation dialog

	targetUser, err := db.GetUserByChatID(targetStaffID)
	if err != nil {
		bh.sendErrorMessageHelper(chatID, messageIDToEdit, "❌ Ошибка загрузки данных сотрудника.")
		return
	}
	if !targetUser.IsBlocked {
		bh.sendInfoMessage(chatID, messageIDToEdit, fmt.Sprintf("ℹ️ Сотрудник %s %s не заблокирован.", targetUser.FirstName, targetUser.LastName), fmt.Sprintf("staff_info_%d", targetStaffID))
		return
	}

	msgText := fmt.Sprintf("🔓 Разблокировать сотрудника *%s %s* (ChatID: `%d`)?\nПричина блокировки: *%s*",
		utils.EscapeTelegramMarkdown(targetUser.FirstName), utils.EscapeTelegramMarkdown(targetUser.LastName),
		targetStaffID, utils.EscapeTelegramMarkdown(targetUser.BlockReason.String))

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("✅ Да, разблокировать", fmt.Sprintf("staff_unblock_confirm_%d", targetStaffID)), // Коллбэк должен быть уникальным для действия / Callback should be unique for the action
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("⬅️ Назад к инфо о сотруднике", fmt.Sprintf("staff_info_%d", targetStaffID)),
		),
	)
	_, errSend := bh.sendOrEditMessageHelper(chatID, messageIDToEdit, msgText, &keyboard, tgbotapi.ModeMarkdown)
	if errSend != nil {
		log.Printf("SendStaffUnblockConfirm: Ошибка для chatID %d: %v", chatID, errSend)
	}
}

// SendStaffDeleteConfirm запрашивает подтверждение "удаления" сотрудника (смены роли на user).
// SendStaffDeleteConfirm prompts for confirmation to "delete" a staff member (change role to user).
func (bh *BotHandler) SendStaffDeleteConfirm(chatID int64, targetStaffID int64, messageIDToEdit int) {
	log.Printf("BotHandler.SendStaffDeleteConfirm для chatID %d, цель staffID: %d, messageIDToEdit: %d", chatID, targetStaffID, messageIDToEdit)
	// Состояние не меняем / Do not change state

	targetUser, err := db.GetUserByChatID(targetStaffID)
	if err != nil {
		bh.sendErrorMessageHelper(chatID, messageIDToEdit, "❌ Ошибка загрузки данных сотрудника.")
		return
	}
	if targetUser.Role == constants.ROLE_OWNER {
		bh.sendInfoMessage(chatID, messageIDToEdit, "🚫 Владельца нельзя удалить этим способом.", fmt.Sprintf("staff_info_%d", targetStaffID))
		return
	}
	if targetUser.Role == constants.ROLE_USER {
		bh.sendInfoMessage(chatID, messageIDToEdit, fmt.Sprintf("ℹ️ %s %s уже является обычным пользователем.", targetUser.FirstName, targetUser.LastName), fmt.Sprintf("staff_info_%d", targetStaffID))
		return
	}

	msgText := fmt.Sprintf("🗑️ Вы уверены, что хотите удалить сотрудника *%s %s* (ChatID: `%d`)?\nЕго роль будет изменена на '%s'. Это действие нельзя будет отменить стандартными средствами.",
		utils.EscapeTelegramMarkdown(targetUser.FirstName), utils.EscapeTelegramMarkdown(targetUser.LastName),
		targetStaffID, utils.GetRoleDisplayName(constants.ROLE_USER))

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("✅ Да, удалить (сменить роль)", fmt.Sprintf("staff_delete_confirm_%d", targetStaffID)), // Коллбэк должен быть уникальным / Callback should be unique
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("⬅️ Назад к инфо о сотруднике", fmt.Sprintf("staff_info_%d", targetStaffID)),
		),
	)
	_, errSend := bh.sendOrEditMessageHelper(chatID, messageIDToEdit, msgText, &keyboard, tgbotapi.ModeMarkdown)
	if errSend != nil {
		log.Printf("SendStaffDeleteConfirm: Ошибка для chatID %d: %v", chatID, errSend)
	}
}
