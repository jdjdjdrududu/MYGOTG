package handlers

import (
	"Original/internal/constants"
	"Original/internal/db"
	"Original/internal/models"
	"Original/internal/session"
	"Original/internal/utils"
	"fmt"
	tgbotapi "github.com/OvyFlash/telegram-bot-api"
	"log"
	"strings"
	"time"
)

// StartDriverInlineReport - существующая функция, без изменений в сигнатуре
func (bh *BotHandler) StartDriverInlineReport(chatID int64, user models.User, messageIDToEdit int) {
	log.Printf("StartDriverInlineReport: для водителя ChatID=%d, UserID=%d, MessageIDToEdit=%d", chatID, user.ID, messageIDToEdit)

	unsettledOrders, err := db.GetUnsettledCompletedOrdersForDriver(user.ID)
	if err != nil {
		log.Printf("StartDriverInlineReport: ошибка получения нерассчитанных заказов для водителя UserID %d: %v", user.ID, err)
		bh.sendErrorMessageHelper(chatID, messageIDToEdit, "❌ Ошибка загрузки заказов для расчета. Попробуйте позже.")
		bh.SendMainMenu(chatID, user, messageIDToEdit)
		return
	}

	if len(unsettledOrders) == 0 {
		log.Printf("StartDriverInlineReport: нет нерассчитанных заказов для водителя UserID %d.", user.ID)
		bh.sendInfoMessage(chatID, messageIDToEdit, "У вас нет выполненных заказов, ожидающих расчета.", "back_to_main")
		bh.Deps.SessionManager.ClearState(chatID)
		bh.Deps.SessionManager.ClearTempDriverSettlement(chatID)
		return
	}

	tempData := session.NewTempDriverSettlement()
	tempData.SettlementCreateTime = time.Now()
	tempData.UnsettledOrders = unsettledOrders
	tempData.CoveredOrdersCount = len(unsettledOrders)
	var totalRevenue float64
	var orderIDs []int64
	for _, order := range unsettledOrders {
		if order.Cost.Valid {
			totalRevenue += order.Cost.Float64
		}
		orderIDs = append(orderIDs, order.ID)
	}
	tempData.CoveredOrdersRevenue = totalRevenue
	tempData.CoveredOrderIDs = orderIDs

	assignedLoadersMap := make(map[int64]string)
	for _, orderID := range tempData.CoveredOrderIDs {
		executors, errExec := db.GetExecutorsByOrderID(int(orderID))
		if errExec != nil {
			log.Printf("StartDriverInlineReport: Ошибка получения исполнителей для заказа #%d: %v", orderID, errExec)
			continue
		}
		for _, executor := range executors {
			if executor.Role == constants.ROLE_LOADER {
				if _, exists := assignedLoadersMap[executor.UserID]; !exists {
					loaderUser, errLoaderUser := db.GetUserByID(int(executor.UserID))
					if errLoaderUser == nil {
						assignedLoadersMap[executor.UserID] = utils.GetUserDisplayName(loaderUser)
					} else {
						assignedLoadersMap[executor.UserID] = fmt.Sprintf("Грузчик ID %d", executor.UserID)
						log.Printf("StartDriverInlineReport: Не удалось получить детали для грузчика UserID %d: %v", executor.UserID, errLoaderUser)
					}
				}
			}
		}
	}

	if len(assignedLoadersMap) > 0 {
		tempData.LoaderPayments = make([]models.LoaderPaymentDetail, 0, len(assignedLoadersMap))
		for loaderUserID, loaderName := range assignedLoadersMap {
			tempData.LoaderPayments = append(tempData.LoaderPayments, models.LoaderPaymentDetail{
				LoaderUserID:     loaderUserID,
				LoaderIdentifier: loaderName,
				Amount:           0,
			})
		}
		log.Printf("StartDriverInlineReport: Предварительно загружено %d грузчиков для отчета.", len(tempData.LoaderPayments))
	}

	if messageIDToEdit != 0 {
		tempData.CurrentMessageID = messageIDToEdit
	}
	bh.Deps.SessionManager.UpdateTempDriverSettlement(chatID, tempData)
	log.Printf("StartDriverInlineReport: Водитель UserID %d начинает инлайн-отчет по %d заказам. Выручка: %.0f. Заказы: %v. Грузчики: %d",
		user.ID, tempData.CoveredOrdersCount, tempData.CoveredOrdersRevenue, tempData.CoveredOrderIDs, len(tempData.LoaderPayments))

	bh.SendDriverReportOverallMenu(chatID, user, messageIDToEdit)
}

// SendDriverReportOverallMenu - отображает основное меню инлайн-отчета
func (bh *BotHandler) SendDriverReportOverallMenu(chatID int64, user models.User, messageIDToEdit int) {
	bh.Deps.SessionManager.SetState(chatID, constants.STATE_DRIVER_REPORT_OVERALL_MENU)
	tempData := bh.Deps.SessionManager.GetTempDriverSettlement(chatID)

	currentMessageIDForThisMenu := messageIDToEdit
	if tempData.CurrentMessageID != 0 && messageIDToEdit == 0 {
		currentMessageIDForThisMenu = tempData.CurrentMessageID
	}

	tempData.RecalculateTotals(bh.Deps.Config.DriverSharePercentage)
	bh.Deps.SessionManager.UpdateTempDriverSettlement(chatID, tempData)

	orderIDsStr := "не указаны"
	if len(tempData.CoveredOrderIDs) > 0 {
		orderIDsStr = strings.Join(utils.Int64SliceToStringSlice(tempData.CoveredOrderIDs), ", ")
	}

	text := fmt.Sprintf("📝 *Отчет по заказам (ID: %s)*\n", orderIDsStr)
	text += fmt.Sprintf("💰 Общая выручка: *%.0f ₽*\n\n", tempData.CoveredOrdersRevenue)
	text += "✏️ *Ваши расходы:*\n"
	fuelTextButton := fmt.Sprintf("⛽️ Топливо: %.0f ₽", tempData.FuelExpense)

	// --- ИЗМЕНЕНИЕ ОТОБРАЖЕНИЯ ПРОЧИХ РАСХОДОВ ---
	var totalOtherExpenses float64
	for _, oe := range tempData.OtherExpenses {
		totalOtherExpenses += oe.Amount
	}
	otherExpensesSummary := "Нет"
	if len(tempData.OtherExpenses) > 0 {
		otherExpensesSummary = fmt.Sprintf("%d шт, %.0f ₽", len(tempData.OtherExpenses), totalOtherExpenses)
	}
	otherTextButton := fmt.Sprintf("🛠️ Прочие: %s", otherExpensesSummary)
	// --- КОНЕЦ ИЗМЕНЕНИЯ ---

	loadersSummary := "Нет назначенных/добавленных"
	totalLoaderSalary := 0.0
	if len(tempData.LoaderPayments) > 0 {
		for _, p := range tempData.LoaderPayments {
			totalLoaderSalary += p.Amount
		}
		loadersSummary = fmt.Sprintf("%d чел, %.0f ₽", len(tempData.LoaderPayments), totalLoaderSalary)
	}
	loadersTextButton := fmt.Sprintf("👷‍♂️ ЗП Грузчикам: %s", loadersSummary)

	text += "\n-------------------------------------\n"
	text += fmt.Sprintf("💸 Ваша зарплата (%.0f%%): *%.0f ₽*\n", bh.Deps.Config.DriverSharePercentage*100, tempData.DriverCalculatedSalary)
	text += fmt.Sprintf("➡️ Сумма к сдаче в кассу: *%.0f ₽*\n", tempData.AmountToCashier)

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(fuelTextButton, constants.CALLBACK_PREFIX_DRIVER_REPORT_SET_FUEL),
		),
		tgbotapi.NewInlineKeyboardRow(
			// --- ИЗМЕНЕНИЕ КОЛЛБЭКА ДЛЯ ПРОЧИХ РАСХОДОВ ---
			tgbotapi.NewInlineKeyboardButtonData(otherTextButton, constants.CALLBACK_PREFIX_DRIVER_REPORT_OTHER_EXPENSES_MENU),
			// --- КОНЕЦ ИЗМЕНЕНИЯ ---
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(loadersTextButton, constants.CALLBACK_PREFIX_DRIVER_REPORT_LOADERS_MENU),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("💾 Сохранить отчет", constants.CALLBACK_PREFIX_DRIVER_REPORT_SAVE_FINAL),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🚫 Отменить и выйти", constants.CALLBACK_PREFIX_DRIVER_REPORT_CANCEL_ALL),
		),
	)

	sentMsg, err := bh.sendOrEditMessageHelper(chatID, currentMessageIDForThisMenu, text, &keyboard, tgbotapi.ModeMarkdown)
	if err == nil && sentMsg.MessageID != 0 {
		updatedTempData := bh.Deps.SessionManager.GetTempDriverSettlement(chatID)
		updatedTempData.CurrentMessageID = sentMsg.MessageID
		bh.Deps.SessionManager.UpdateTempDriverSettlement(chatID, updatedTempData)
		log.Printf("SendDriverReportOverallMenu: CurrentMessageID обновлен на %d", sentMsg.MessageID)
	} else if err != nil {
		log.Printf("SendDriverReportOverallMenu: Ошибка отправки/редактирования меню отчета: %v", err)
	}
}

// SendDriverReportFuelInputPrompt - запрос суммы топлива (без изменений)
func (bh *BotHandler) SendDriverReportFuelInputPrompt(chatID int64, user models.User, messageIDToEdit int) {
	bh.Deps.SessionManager.SetState(chatID, constants.STATE_DRIVER_REPORT_INPUT_FUEL)
	tempData := bh.Deps.SessionManager.GetTempDriverSettlement(chatID)
	tempData.CurrentMessageID = messageIDToEdit
	bh.Deps.SessionManager.UpdateTempDriverSettlement(chatID, tempData)

	text := "⛽ Введите сумму расходов на *топливо* (₽) по этим заказам:"
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔙 Назад к отчету", constants.CALLBACK_PREFIX_DRIVER_REPORT_OVERALL_MENU),
		),
	)
	bh.sendOrEditMessageHelper(chatID, messageIDToEdit, text, &keyboard, tgbotapi.ModeMarkdown)
}

// --- НАЧАЛО НОВЫХ ФУНКЦИЙ ДЛЯ ПРОЧИХ РАСХОДОВ ---

// SendDriverReportOtherExpensesMenu - отображает меню управления прочими расходами.
func (bh *BotHandler) SendDriverReportOtherExpensesMenu(chatID int64, user models.User, messageIDToEdit int) {
	bh.Deps.SessionManager.SetState(chatID, constants.STATE_DRIVER_REPORT_OTHER_EXPENSES_MENU) // Новое состояние для этого меню
	tempData := bh.Deps.SessionManager.GetTempDriverSettlement(chatID)
	tempData.CurrentMessageID = messageIDToEdit
	// Очищаем временные поля для ввода нового расхода
	tempData.TempOtherExpenseDescription = ""
	bh.Deps.SessionManager.UpdateTempDriverSettlement(chatID, tempData)

	var text strings.Builder
	text.WriteString("🛠️ *Прочие расходы по заказам:*\n\n")

	if len(tempData.OtherExpenses) == 0 {
		text.WriteString("_Прочих расходов пока не добавлено._\n")
	} else {
		for i, expense := range tempData.OtherExpenses {
			text.WriteString(fmt.Sprintf("%d. %s: *%.0f ₽*\n", i+1, utils.EscapeTelegramMarkdown(expense.Description), expense.Amount))
			// TODO: Можно добавить кнопки для редактирования/удаления каждого расхода, если потребуется
			// CALLBACK_PREFIX_DRIVER_REPORT_EDIT_OTHER_EXPENSE_PROMPT_i
			// CALLBACK_PREFIX_DRIVER_REPORT_DELETE_OTHER_EXPENSE_CONFIRM_i
		}
	}
	text.WriteString("\nНажмите, чтобы добавить новый расход или вернуться к общему отчету.")

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("➕ Добавить еще расход", constants.CALLBACK_PREFIX_DRIVER_REPORT_ADD_OTHER_EXPENSE_DESCRIPTION_PROMPT),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔙 Назад к общему отчету", constants.CALLBACK_PREFIX_DRIVER_REPORT_OVERALL_MENU),
		),
	)

	sentMsg, err := bh.sendOrEditMessageHelper(chatID, messageIDToEdit, text.String(), &keyboard, tgbotapi.ModeMarkdown)
	if err == nil && sentMsg.MessageID != 0 {
		updatedTempData := bh.Deps.SessionManager.GetTempDriverSettlement(chatID)
		updatedTempData.CurrentMessageID = sentMsg.MessageID
		bh.Deps.SessionManager.UpdateTempDriverSettlement(chatID, updatedTempData)
	} else if err != nil {
		log.Printf("SendDriverReportOtherExpensesMenu: Ошибка отправки/редактирования: %v", err)
	}
}

// SendDriverReportOtherExpenseDescriptionPrompt - запрос описания прочего расхода.
func (bh *BotHandler) SendDriverReportOtherExpenseDescriptionPrompt(chatID int64, user models.User, messageIDToEdit int, isEditing bool, expenseIndex int) {
	var stateToSet string
	var currentDescription string
	if isEditing {
		stateToSet = constants.STATE_DRIVER_REPORT_EDIT_OTHER_EXPENSE_DESCRIPTION
		tempData := bh.Deps.SessionManager.GetTempDriverSettlement(chatID)
		if expenseIndex >= 0 && expenseIndex < len(tempData.OtherExpenses) {
			currentDescription = tempData.OtherExpenses[expenseIndex].Description
		}
	} else {
		stateToSet = constants.STATE_DRIVER_REPORT_INPUT_OTHER_EXPENSE_DESCRIPTION
	}
	bh.Deps.SessionManager.SetState(chatID, stateToSet)

	tempData := bh.Deps.SessionManager.GetTempDriverSettlement(chatID)
	tempData.CurrentMessageID = messageIDToEdit
	tempData.EditingOtherExpenseIndex = expenseIndex // Сохраняем индекс, если редактируем, или -1 если добавляем
	if isEditing {
		tempData.TempOtherExpenseDescription = currentDescription // Предзаполняем для редактирования
	} else {
		tempData.TempOtherExpenseDescription = "" // Очищаем для нового
	}
	bh.Deps.SessionManager.UpdateTempDriverSettlement(chatID, tempData)

	text := "📝 Введите *описание* прочего расхода (например, 'Парковка', 'Штраф')"
	if isEditing && currentDescription != "" {
		text = fmt.Sprintf("📝 Введите новое *описание* для '%s' (или оставьте текущее, отправив его снова):", utils.EscapeTelegramMarkdown(currentDescription))
	} else if isEditing {
		text = "📝 Введите новое *описание* для этого расхода:"
	}

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔙 Назад к списку прочих расходов", constants.CALLBACK_PREFIX_DRIVER_REPORT_OTHER_EXPENSES_MENU),
		),
	)
	bh.sendOrEditMessageHelper(chatID, messageIDToEdit, text, &keyboard, tgbotapi.ModeMarkdown)
}

// SendDriverReportOtherExpenseAmountPrompt - запрос суммы для прочего расхода.
func (bh *BotHandler) SendDriverReportOtherExpenseAmountPrompt(chatID int64, user models.User, messageIDToEdit int, description string, isEditing bool, expenseIndex int) {
	var stateToSet string
	var currentAmount float64
	if isEditing {
		stateToSet = constants.STATE_DRIVER_REPORT_EDIT_OTHER_EXPENSE_AMOUNT
		tempData := bh.Deps.SessionManager.GetTempDriverSettlement(chatID)
		if expenseIndex >= 0 && expenseIndex < len(tempData.OtherExpenses) {
			// Убедимся, что описание соответствует, если мы редактируем
			// Это важно, если описание было изменено на предыдущем шаге редактирования
			if tempData.TempOtherExpenseDescription == tempData.OtherExpenses[expenseIndex].Description || description == tempData.OtherExpenses[expenseIndex].Description {
				currentAmount = tempData.OtherExpenses[expenseIndex].Amount
			} else { // Описание изменилось, значит это новый ввод суммы для измененного описания
				currentAmount = 0 // или можно не показывать текущую сумму, если описание новое
			}
		}
	} else {
		stateToSet = constants.STATE_DRIVER_REPORT_INPUT_OTHER_EXPENSE_AMOUNT
	}
	bh.Deps.SessionManager.SetState(chatID, stateToSet)

	tempData := bh.Deps.SessionManager.GetTempDriverSettlement(chatID)
	tempData.CurrentMessageID = messageIDToEdit
	// TempOtherExpenseDescription уже должен быть установлен из предыдущего шага (или взят из existing item при редактировании)
	// EditingOtherExpenseIndex тоже уже должен быть установлен, если isEditing = true
	bh.Deps.SessionManager.UpdateTempDriverSettlement(chatID, tempData)

	text := fmt.Sprintf("💰 Введите *сумму* (₽) для расхода '%s'", utils.EscapeTelegramMarkdown(description))
	if isEditing {
		text = fmt.Sprintf("💰 Введите новую *сумму* (₽) для '%s' (текущая: %.0f):", utils.EscapeTelegramMarkdown(description), currentAmount)
	}

	backCallbackData := constants.CALLBACK_PREFIX_DRIVER_REPORT_ADD_OTHER_EXPENSE_DESCRIPTION_PROMPT // Для нового расхода
	if isEditing {
		// При редактировании суммы, "Назад" должно вести к редактированию описания ЭТОГО ЖЕ расхода.
		backCallbackData = fmt.Sprintf("%s_%d", constants.CALLBACK_PREFIX_DRIVER_REPORT_EDIT_OTHER_EXPENSE_PROMPT, expenseIndex)
	}

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔙 Назад к описанию", backCallbackData),
		),
	)
	bh.sendOrEditMessageHelper(chatID, messageIDToEdit, text, &keyboard, tgbotapi.ModeMarkdown)
}

// SendDriverReportConfirmAddOtherExpense - запрос на добавление еще одного прочего расхода.
func (bh *BotHandler) SendDriverReportConfirmAddOtherExpense(chatID int64, user models.User, messageIDToEdit int, addedDescription string, addedAmount float64) {
	bh.Deps.SessionManager.SetState(chatID, constants.STATE_DRIVER_REPORT_CONFIRM_ADD_OTHER_EXPENSE) // Это состояние для кнопок ниже
	tempData := bh.Deps.SessionManager.GetTempDriverSettlement(chatID)
	tempData.CurrentMessageID = messageIDToEdit
	bh.Deps.SessionManager.UpdateTempDriverSettlement(chatID, tempData)

	text := fmt.Sprintf("✅ Расход '%s: %.0f ₽' добавлен.\n\nДобавить еще один прочий расход?",
		utils.EscapeTelegramMarkdown(addedDescription), addedAmount)

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("➕ Да, добавить еще", constants.CALLBACK_PREFIX_DRIVER_REPORT_ADD_OTHER_EXPENSE_DESCRIPTION_PROMPT),
		),
		tgbotapi.NewInlineKeyboardRow(
			// Возвращаемся в меню списка прочих расходов, где будет виден новый расход
			tgbotapi.NewInlineKeyboardButtonData("↪️ Нет, завершить прочие расходы", constants.CALLBACK_PREFIX_DRIVER_REPORT_OTHER_EXPENSES_MENU),
		),
	)
	bh.sendOrEditMessageHelper(chatID, messageIDToEdit, text, &keyboard, tgbotapi.ModeMarkdown)
}

// --- КОНЕЦ НОВЫХ ФУНКЦИЙ ДЛЯ ПРОЧИХ РАСХОДОВ ---

// SendDriverReportLoadersSubMenu - меню управления грузчиками (без изменений)
func (bh *BotHandler) SendDriverReportLoadersSubMenu(chatID int64, user models.User, messageIDToEdit int) {
	bh.Deps.SessionManager.SetState(chatID, constants.STATE_DRIVER_REPORT_LOADERS_MENU)
	tempData := bh.Deps.SessionManager.GetTempDriverSettlement(chatID)
	tempData.CurrentMessageID = messageIDToEdit
	bh.Deps.SessionManager.UpdateTempDriverSettlement(chatID, tempData)

	text := "👷‍♂️ *Зарплаты грузчикам по данным заказам:*\n(Укажите ЗП для назначенных грузчиков)\n\n"
	var rows [][]tgbotapi.InlineKeyboardButton

	if len(tempData.LoaderPayments) == 0 {
		text += "_Нет назначенных грузчиков по данным заказам, или они еще не загружены в отчет._\n"
	} else {
		for i, loaderPayment := range tempData.LoaderPayments {
			loaderRowText := fmt.Sprintf("%s: %.0f ₽", utils.EscapeTelegramMarkdown(loaderPayment.LoaderIdentifier), loaderPayment.Amount)
			rows = append(rows, tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData(fmt.Sprintf("✏️ %s", loaderRowText), fmt.Sprintf("%s_%d", constants.CALLBACK_PREFIX_DRIVER_REPORT_EDIT_LOADER_PROMPT, i)),
			))
		}
	}
	rows = append(rows, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("🔙 Назад к отчету", constants.CALLBACK_PREFIX_DRIVER_REPORT_OVERALL_MENU),
	))
	keyboard := tgbotapi.NewInlineKeyboardMarkup(rows...)
	bh.sendOrEditMessageHelper(chatID, messageIDToEdit, text, &keyboard, tgbotapi.ModeMarkdown)
}

// SendDriverReportLoaderNameInputPrompt - запрос имени нового грузчика (без изменений)
func (bh *BotHandler) SendDriverReportLoaderNameInputPrompt(chatID int64, user models.User, messageIDToEdit int) {
	bh.Deps.SessionManager.SetState(chatID, constants.STATE_DRIVER_REPORT_INPUT_LOADER_NAME)
	tempData := bh.Deps.SessionManager.GetTempDriverSettlement(chatID)
	tempData.CurrentMessageID = messageIDToEdit
	tempData.TempLoaderNameInput = ""
	tempData.EditingLoaderIndex = -1
	bh.Deps.SessionManager.UpdateTempDriverSettlement(chatID, tempData)

	text := "🧑‍🔧 Введите имя или идентификатор нового грузчика (если он не был назначен на заказ):"
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔙 Назад к списку грузчиков", constants.CALLBACK_PREFIX_DRIVER_REPORT_LOADERS_MENU),
		),
	)
	bh.sendOrEditMessageHelper(chatID, messageIDToEdit, text, &keyboard, tgbotapi.ModeMarkdown)
}

// SendDriverReportLoaderSalaryInputPrompt - запрос ЗП для грузчика (без изменений)
func (bh *BotHandler) SendDriverReportLoaderSalaryInputPrompt(chatID int64, user models.User, messageIDToEdit int, loaderIdentifier string, isEditing bool, loaderIndex int) {
	var stateToSet string
	var backCallback string

	if isEditing {
		stateToSet = constants.STATE_DRIVER_REPORT_EDIT_LOADER_SALARY
		backCallback = constants.CALLBACK_PREFIX_DRIVER_REPORT_LOADERS_MENU
	} else {
		stateToSet = constants.STATE_DRIVER_REPORT_INPUT_LOADER_SALARY
		backCallback = constants.CALLBACK_PREFIX_DRIVER_REPORT_LOADERS_MENU
	}

	bh.Deps.SessionManager.SetState(chatID, stateToSet)
	tempData := bh.Deps.SessionManager.GetTempDriverSettlement(chatID)
	tempData.CurrentMessageID = messageIDToEdit
	if isEditing {
		tempData.EditingLoaderIndex = loaderIndex
	} else {
		tempData.EditingLoaderIndex = -1
	}
	bh.Deps.SessionManager.UpdateTempDriverSettlement(chatID, tempData)

	currentSalary := 0.0
	if isEditing && loaderIndex >= 0 && loaderIndex < len(tempData.LoaderPayments) {
		currentSalary = tempData.LoaderPayments[loaderIndex].Amount
	}

	text := fmt.Sprintf("💸 Введите сумму зарплаты (₽) для грузчика *%s* (текущая: %.0f ₽):",
		utils.EscapeTelegramMarkdown(loaderIdentifier), currentSalary)
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔙 Назад", backCallback),
		),
	)
	bh.sendOrEditMessageHelper(chatID, messageIDToEdit, text, &keyboard, tgbotapi.ModeMarkdown)
}

// SendDriverReportConfirmDeleteLoaderPrompt - запрос подтверждения удаления грузчика (без изменений)
func (bh *BotHandler) SendDriverReportConfirmDeleteLoaderPrompt(chatID int64, user models.User, messageIDToEdit int, loaderIndex int) {
	tempData := bh.Deps.SessionManager.GetTempDriverSettlement(chatID)
	tempData.CurrentMessageID = messageIDToEdit
	tempData.EditingLoaderIndex = loaderIndex
	bh.Deps.SessionManager.UpdateTempDriverSettlement(chatID, tempData)

	if loaderIndex < 0 || loaderIndex >= len(tempData.LoaderPayments) {
		bh.sendErrorMessageHelper(chatID, messageIDToEdit, "Ошибка: грузчик для удаления не найден в текущем отчете.")
		bh.SendDriverReportLoadersSubMenu(chatID, user, messageIDToEdit)
		return
	}
	loaderToDelete := tempData.LoaderPayments[loaderIndex]
	text := fmt.Sprintf("🗑️ Вы уверены, что хотите удалить запись о ЗП для грузчика *%s* (Текущая ЗП: %.0f ₽) из этого отчета?",
		utils.EscapeTelegramMarkdown(loaderToDelete.LoaderIdentifier), loaderToDelete.Amount)

	confirmCallback := fmt.Sprintf("%s_%d", constants.CALLBACK_PREFIX_DRIVER_REPORT_DELETE_LOADER_CONFIRM, loaderIndex)

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("✅ Да, удалить из отчета", confirmCallback),
			tgbotapi.NewInlineKeyboardButtonData("❌ Нет, назад", constants.CALLBACK_PREFIX_DRIVER_REPORT_LOADERS_MENU),
		),
	)
	bh.sendOrEditMessageHelper(chatID, messageIDToEdit, text, &keyboard, tgbotapi.ModeMarkdown)
}
func (bh *BotHandler) SendDriverReportConfirmDeleteOtherExpensePrompt(chatID int64, user models.User, messageIDToEdit int, expenseIndex int) {
	// Устанавливаем состояние в обработчике коллбэка перед вызовом этой функции,
	// поэтому здесь можно не менять, либо установить на STATE_DRIVER_REPORT_CONFIRM_DELETE_OTHER_EXPENSE, если такое состояние есть
	// bh.Deps.SessionManager.SetState(chatID, constants.STATE_DRIVER_REPORT_CONFIRM_DELETE_OTHER_EXPENSE)
	tempData := bh.Deps.SessionManager.GetTempDriverSettlement(chatID)
	tempData.CurrentMessageID = messageIDToEdit
	tempData.EditingOtherExpenseIndex = expenseIndex // Сохраняем индекс для действия удаления
	bh.Deps.SessionManager.UpdateTempDriverSettlement(chatID, tempData)

	if expenseIndex < 0 || expenseIndex >= len(tempData.OtherExpenses) {
		bh.sendErrorMessageHelper(chatID, messageIDToEdit, "Ошибка: расход для удаления не найден.")
		bh.SendDriverReportOtherExpensesMenu(chatID, user, messageIDToEdit) // Возвращаем к списку
		return
	}
	expenseToDelete := tempData.OtherExpenses[expenseIndex]
	text := fmt.Sprintf("🗑️ Вы уверены, что хотите удалить прочий расход:\n*%s: %.0f ₽*?",
		utils.EscapeTelegramMarkdown(expenseToDelete.Description), expenseToDelete.Amount)

	// Коллбэк для подтверждения должен содержать индекс и вести на CALLBACK_PREFIX_DRIVER_REPORT_DELETE_OTHER_EXPENSE_CONFIRM
	confirmCallback := fmt.Sprintf("%s_%d", constants.CALLBACK_PREFIX_DRIVER_REPORT_DELETE_OTHER_EXPENSE_CONFIRM, expenseIndex)

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("✅ Да, удалить", confirmCallback),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("❌ Нет, назад", constants.CALLBACK_PREFIX_DRIVER_REPORT_OTHER_EXPENSES_MENU),
		),
	)
	bh.sendOrEditMessageHelper(chatID, messageIDToEdit, text, &keyboard, tgbotapi.ModeMarkdown)
}

// --- КОНЕЦ ФАЙЛА internal/handlers/menu_handlers_driver_expenses.go ---
