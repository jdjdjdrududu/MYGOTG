package handlers

import (
	"Original/internal/constants"
	"Original/internal/db"
	"Original/internal/models"
	"Original/internal/utils"
	// "database/sql" // Убрано, если не используется напрямую после удаления старых функций
	"fmt"
	tgbotapi "github.com/OvyFlash/telegram-bot-api"
	"log"
	// "strconv" // Убрано, если не используется
)

// SendMySalaryMenu отправляет сотруднику (грузчику, водителю) меню "Моя зарплата".
func (bh *BotHandler) SendMySalaryMenu(chatID int64, user models.User, messageIDToEdit int) {
	log.Printf("SendMySalaryMenu: для ChatID=%d, Роль=%s, MessageIDToEdit=%d", chatID, user.Role, messageIDToEdit)
	bh.Deps.SessionManager.SetState(chatID, constants.STATE_MY_SALARY_MENU)

	if user.Role == constants.ROLE_USER {
		bh.sendAccessDenied(chatID, messageIDToEdit)
		return
	}

	text := "💰 Моя зарплата\n\nВыберите, что вас интересует:"
	var rows [][]tgbotapi.InlineKeyboardButton

	rows = append(rows, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("❓ Сколько мне должны?", fmt.Sprintf("%s_owed", constants.CALLBACK_PREFIX_MY_SALARY)),
	))
	rows = append(rows, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("📊 Сколько я заработал (всего)?", fmt.Sprintf("%s_earned_stats", constants.CALLBACK_PREFIX_MY_SALARY)),
	))

	rows = append(rows, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("🔙 Главное меню", "back_to_main"),
	))

	keyboard := tgbotapi.NewInlineKeyboardMarkup(rows...)
	_, err := bh.sendOrEditMessageHelper(chatID, messageIDToEdit, text, &keyboard, "")
	if err != nil {
		log.Printf("SendMySalaryMenu: Ошибка отправки меню для ChatID %d: %v", chatID, err)
	}
}

// HandleShowAmountOwed показывает пользователю, сколько ему должны.
func (bh *BotHandler) HandleShowAmountOwed(chatID int64, user models.User, messageIDToEdit int) {
	log.Printf("HandleShowAmountOwed: для ChatID=%d, Роль=%s", chatID, user.Role)
	bh.Deps.SessionManager.SetState(chatID, constants.STATE_VIEW_SALARY_OWED)

	amountOwed, err := db.GetAmountOwedToUser(user.ID, user.Role)
	if err != nil {
		log.Printf("HandleShowAmountOwed: Ошибка получения суммы к выплате для UserID %d (ChatID %d): %v", user.ID, chatID, err)
		bh.sendErrorMessageHelper(chatID, messageIDToEdit, "❌ Не удалось получить данные о сумме к выплате.")
		return
	}

	var cardNumberDisplay string
	if user.CardNumber.Valid && user.CardNumber.String != "" {
		cardNumberDisplay = fmt.Sprintf("\n💳 Карта для выплат: `%s`", utils.EscapeTelegramMarkdown(user.CardNumber.String))
	} else {
		cardNumberDisplay = "\n⚠️ Номер карты для выплат не указан. Обратитесь к администратору."
	}

	text := fmt.Sprintf("💸 Вам должны выплатить: *%.0f ₽*%s", amountOwed, cardNumberDisplay)
	if amountOwed <= 0 {
		text = "✅ На данный момент все выплаты произведены." + cardNumberDisplay
	}

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔙 Назад в 'Моя зарплата'", constants.CALLBACK_PREFIX_MY_SALARY),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🏢 Главное меню", "back_to_main"),
		),
	)

	_, errSend := bh.sendOrEditMessageHelper(chatID, messageIDToEdit, text, &keyboard, tgbotapi.ModeMarkdown)
	if errSend != nil {
		log.Printf("HandleShowAmountOwed: Ошибка отправки сообщения для ChatID %d: %v", chatID, errSend)
	}
}

// HandleShowEarnedStats показывает пользователю, сколько он заработал (общая сумма).
func (bh *BotHandler) HandleShowEarnedStats(chatID int64, user models.User, messageIDToEdit int) {
	log.Printf("HandleShowEarnedStats: для ChatID=%d, Роль=%s", chatID, user.Role)
	bh.Deps.SessionManager.SetState(chatID, constants.STATE_VIEW_SALARY_EARNED)

	totalEarned, err := db.GetTotalEarnedForUser(user.ID, user.Role)
	if err != nil {
		log.Printf("HandleShowEarnedStats: Ошибка получения общего заработка для UserID %d (ChatID %d): %v", user.ID, chatID, err)
		bh.sendErrorMessageHelper(chatID, messageIDToEdit, "❌ Не удалось получить данные о заработанной сумме.")
		return
	}
	text := fmt.Sprintf("📊 Всего заработано (за всё время): *%.0f ₽*", totalEarned)

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔙 Назад в 'Моя зарплата'", constants.CALLBACK_PREFIX_MY_SALARY),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🏢 Главное меню", "back_to_main"),
		),
	)

	_, errSend := bh.sendOrEditMessageHelper(chatID, messageIDToEdit, text, &keyboard, tgbotapi.ModeMarkdown)
	if errSend != nil {
		log.Printf("HandleShowEarnedStats: Ошибка отправки сообщения для ChatID %d: %v", chatID, errSend)
	}
}

// --- УДАЛЕНЫ УСТАРЕВШИЕ ФУНКЦИИ ---
// SendDriverExpensesMainMenu
// SendDriverSelectOrderForExpenses
// SendDriverExpenseInputMenu
// --- КОНЕЦ УДАЛЕННЫХ ФУНКЦИЙ ---

// SendOwnerPayoutsMainMenu отправляет главное меню выплат сотрудникам для Владельца.
func (bh *BotHandler) SendOwnerPayoutsMainMenu(chatID int64, user models.User, messageIDToEdit int) {
	log.Printf("SendOwnerPayoutsMainMenu: для Владельца ChatID=%d", chatID)
	if user.Role != constants.ROLE_OWNER {
		bh.sendAccessDenied(chatID, messageIDToEdit)
		return
	}
	bh.Deps.SessionManager.SetState(chatID, constants.STATE_OWNER_STAFF_PAYOUTS_MENU)
	bh.SendOwnerStaffListForPayout(chatID, user, messageIDToEdit, 0)
}

// SendOwnerStaffListForPayout отображает список сотрудников с суммами к выплате Владельцу.
func (bh *BotHandler) SendOwnerStaffListForPayout(chatID int64, user models.User, messageIDToEdit int, page int) {
	log.Printf("SendOwnerStaffListForPayout: для Владельца ChatID=%d, страница %d", chatID, page)
	bh.Deps.SessionManager.SetState(chatID, constants.STATE_OWNER_SELECT_STAFF_FOR_PAYOUT)

	staffRoles := []string{constants.ROLE_DRIVER, constants.ROLE_LOADER, constants.ROLE_OPERATOR, constants.ROLE_MAINOPERATOR}
	allStaff, err := db.GetUsersByRole(staffRoles...)
	if err != nil {
		log.Printf("SendOwnerStaffListForPayout: Ошибка получения списка сотрудников: %v", err)
		bh.sendErrorMessageHelper(chatID, messageIDToEdit, "❌ Ошибка загрузки списка сотрудников.")
		return
	}

	var staffWithDebt []models.User
	var staffDebtMap = make(map[int64]float64)

	for _, staffMember := range allStaff {
		amountOwed, errOwed := db.GetAmountOwedToUser(staffMember.ID, staffMember.Role)
		if errOwed != nil {
			log.Printf("SendOwnerStaffListForPayout: Ошибка получения суммы к выплате для UserID %d: %v", staffMember.ID, errOwed)
			continue
		}
		if amountOwed > 0 {
			staffWithDebt = append(staffWithDebt, staffMember)
			staffDebtMap[staffMember.ID] = amountOwed
		}
	}

	start := page * constants.PayoutsPerPage
	end := start + constants.PayoutsPerPage
	var paginatedStaff []models.User
	if start >= len(staffWithDebt) {
		paginatedStaff = []models.User{}
	} else if end > len(staffWithDebt) {
		paginatedStaff = staffWithDebt[start:]
	} else {
		paginatedStaff = staffWithDebt[start:end]
	}

	var msgText string
	var keyboard tgbotapi.InlineKeyboardMarkup
	var rows [][]tgbotapi.InlineKeyboardButton

	if len(paginatedStaff) == 0 && page == 0 {
		msgText = "💸 На данный момент нет сотрудников с задолженностью по зарплате."
	} else if len(paginatedStaff) == 0 && page > 0 {
		msgText = "💸 Больше сотрудников с задолженностью нет."
	} else {
		msgText = "💸 Выплаты сотрудникам:\n\nВыберите сотрудника для осуществления выплаты:"
		for _, staffMember := range paginatedStaff { // Итерируемся по пагинированному списку
			amountOwed := staffDebtMap[staffMember.ID]
			displayName := utils.GetUserDisplayName(staffMember)
			cardDisplay := "Карта не указана"
			if staffMember.CardNumber.Valid && staffMember.CardNumber.String != "" {
				cardDisplay = fmt.Sprintf("Карта: `%s`", utils.EscapeTelegramMarkdown(staffMember.CardNumber.String))
			}

			buttonText := fmt.Sprintf("%s (%s) - %.0f ₽ [%s]", displayName, utils.GetRoleDisplayName(staffMember.Role), amountOwed, cardDisplay)
			if len(buttonText) > 60 {
				buttonText = buttonText[:57] + "..."
			}
			callbackData := fmt.Sprintf("%s_select_%d_%.0f", constants.CALLBACK_PREFIX_OWNER_STAFF_PAYOUT, staffMember.ID, amountOwed)
			rows = append(rows, tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData(buttonText, callbackData),
			))
		}
	}

	navRow := []tgbotapi.InlineKeyboardButton{}
	if page > 0 {
		navRow = append(navRow, tgbotapi.NewInlineKeyboardButtonData("⬅️ Назад", fmt.Sprintf("%s_page_%d", constants.CALLBACK_PREFIX_OWNER_STAFF_PAYOUT, page-1)))
	}
	// Условие для кнопки "Далее"
	if end < len(staffWithDebt) { // Сравниваем end с общей длиной списка staffWithDebt (до пагинации)
		navRow = append(navRow, tgbotapi.NewInlineKeyboardButtonData("➡️ Далее", fmt.Sprintf("%s_page_%d", constants.CALLBACK_PREFIX_OWNER_STAFF_PAYOUT, page+1)))
	}
	if len(navRow) > 0 {
		rows = append(rows, navRow)
	}

	rows = append(rows, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("🔙 Главное меню", "back_to_main"),
	))
	keyboard.InlineKeyboard = rows

	_, errSend := bh.sendOrEditMessageHelper(chatID, messageIDToEdit, msgText, &keyboard, tgbotapi.ModeMarkdown)
	if errSend != nil {
		log.Printf("SendOwnerStaffListForPayout: Ошибка отправки меню для ChatID %d: %v", chatID, errSend)
	}
}

// SendOwnerConfirmPayoutToStaff отправляет Владельцу диалог подтверждения выплаты сотруднику.
func (bh *BotHandler) SendOwnerConfirmPayoutToStaff(chatID int64, user models.User, targetUserID int64, amountOwed float64, messageIDToEdit int) {
	log.Printf("SendOwnerConfirmPayoutToStaff: Владелец %d подтверждает выплату %.0f сотруднику UserID %d", chatID, amountOwed, targetUserID)
	bh.Deps.SessionManager.SetState(chatID, constants.STATE_OWNER_CONFIRM_STAFF_PAYOUT)

	targetStaff, err := db.GetUserByID(int(targetUserID))
	if err != nil {
		log.Printf("SendOwnerConfirmPayoutToStaff: Ошибка получения данных сотрудника UserID %d: %v", targetUserID, err)
		bh.sendErrorMessageHelper(chatID, messageIDToEdit, "❌ Ошибка: не удалось найти данные сотрудника.")
		return
	}

	cardDisplay := "Карта не указана"
	if targetStaff.CardNumber.Valid && targetStaff.CardNumber.String != "" {
		cardDisplay = fmt.Sprintf("`%s`", utils.EscapeTelegramMarkdown(targetStaff.CardNumber.String))
	}
	confirmText := fmt.Sprintf("❓ Вы уверены, что хотите выплатить *%.0f ₽* сотруднику %s (%s) на карту %s?",
		amountOwed, utils.GetUserDisplayName(targetStaff), utils.GetRoleDisplayName(targetStaff.Role), cardDisplay)

	confirmCallback := fmt.Sprintf("%s_confirm_%d_%.0f", constants.CALLBACK_PREFIX_OWNER_STAFF_PAYOUT, targetUserID, amountOwed)
	cancelCallback := fmt.Sprintf("%s_page_0", constants.CALLBACK_PREFIX_OWNER_STAFF_PAYOUT)

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("✅ Да, выплатить", confirmCallback),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("⬅️ Нет, назад к списку", cancelCallback),
		),
	)
	_, errSend := bh.sendOrEditMessageHelper(chatID, messageIDToEdit, confirmText, &keyboard, tgbotapi.ModeMarkdown)
	if errSend != nil {
		log.Printf("SendOwnerConfirmPayoutToStaff: Ошибка отправки сообщения для ChatID %d: %v", chatID, errSend)
	}
}
