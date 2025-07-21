package handlers

import (
	"Original/internal/constants"
	"Original/internal/db"
	"Original/internal/models"
	"Original/internal/session"
	"Original/internal/utils"
	"database/sql"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	tgbotapi "github.com/OvyFlash/telegram-bot-api"
)

// SendOwnerFinancialsMainMenu - главное меню "Денежные средства" для владельца (СТАРАЯ ВЕРСИЯ - ПО ДАТАМ)
// Эта функция может быть помечена как DEPRECATED или удалена, если новый флоу ее полностью заменяет.
// Пока оставляем для обратной совместимости или если старый флоу еще где-то используется.
func (bh *BotHandler) SendOwnerFinancialsMainMenu(chatID int64, user models.User, messageIDToEdit int) {
	log.Printf("SendOwnerFinancialsMainMenu (DEPRECATED): для владельца ChatID=%d", chatID)
	bh.Deps.SessionManager.SetState(chatID, constants.STATE_OWNER_FINANCIAL_MAIN) // Старое состояние
	// Перенаправляем на новое меню управления денежными средствами
	bh.SendOwnerCashManagementMenu(chatID, user, messageIDToEdit)
}

// SendOwnerFinancialsForDate - отображает суммы к сдаче от водителей за указанную дату (СТАРАЯ ВЕРСИЯ)
// Эта функция также может быть помечена как DEPRECATED.
func (bh *BotHandler) SendOwnerFinancialsForDate(chatID int64, user models.User, targetDate time.Time, messageIDToEdit int) {
	log.Printf("SendOwnerFinancialsForDate (DEPRECATED): для ChatID %d, дата %s", chatID, targetDate.Format("2006-01-02"))
	bh.SendOwnerCashManagementMenu(chatID, user, messageIDToEdit)
}

// handleOwnerViewDriverSettlementsForDate - отображает детализацию по водителю за дату (СТАРАЯ ВЕРСИЯ)
// Может быть DEPRECATED. Новая логика в SendOwnerDriverIndividualSettlementsList.
func (bh *BotHandler) handleOwnerViewDriverSettlementsForDate(chatID int64, user models.User, driverUserID int64, reportDateStr string, messageIDToEdit int) {
	log.Printf("handleOwnerViewDriverSettlementsForDate (DEPRECATED): ChatID %d, DriverUserID %d, ReportDate %s", chatID, driverUserID, reportDateStr)
	// Перенаправляем на новое меню
	bh.SendOwnerDriverIndividualSettlementsList(chatID, user, messageIDToEdit, driverUserID, constants.VIEW_TYPE_ACTUAL_SETTLEMENTS, 0)
}

// handleOwnerEditSettlementStart - начало редактирования отчета владельцем.
// Эта функция будет вызываться новым коллбэком CALLBACK_PREFIX_OWNER_EDIT_SETTLEMENT.
// Параметры viewTypeForBackNav и pageForBackNav передаются для кнопки "Назад".
func (bh *BotHandler) handleOwnerEditSettlementStart(chatID int64, user models.User, settlementID int64, viewTypeForBackNav string, pageForBackNav int, messageIDToEdit int) {
	log.Printf("handleOwnerEditSettlementStart: ChatID=%d, SettlementID=%d, ViewTypeBack=%s, PageBack=%d", chatID, settlementID, viewTypeForBackNav, pageForBackNav)
	settlement, err := db.GetDriverSettlementByID(settlementID)
	if err != nil {
		bh.sendErrorMessageHelper(chatID, messageIDToEdit, "Не удалось загрузить отчет для редактирования.")
		bh.SendOwnerCashManagementMenu(chatID, user, messageIDToEdit)
		return
	}

	tempSettleData := session.NewTempDriverSettlement()
	tempSettleData.EditingSettlementID = settlement.ID
	tempSettleData.SettlementCreateTime = settlement.SettlementTimestamp
	tempSettleData.OriginalPaidToOwnerAt = settlement.PaidToOwnerAt

	tempSettleData.CoveredOrdersRevenue = settlement.CoveredOrdersRevenue
	tempSettleData.FuelExpense = settlement.FuelExpense
	// tempSettleData.OtherExpense = settlement.OtherExpense // УДАЛЕНО, ТАК КАК OtherExpense БОЛЬШЕ НЕТ
	tempSettleData.OtherExpenses = make([]models.OtherExpenseDetail, len(settlement.OtherExpenses)) // ИЗМЕНЕНО
	copy(tempSettleData.OtherExpenses, settlement.OtherExpenses)                                    // ИЗМЕНЕНО

	tempSettleData.LoaderPayments = make([]models.LoaderPaymentDetail, len(settlement.LoaderPayments))
	copy(tempSettleData.LoaderPayments, settlement.LoaderPayments)
	tempSettleData.CoveredOrdersCount = settlement.CoveredOrdersCount
	tempSettleData.CoveredOrderIDs = make([]int64, len(settlement.CoveredOrderIDs))
	copy(tempSettleData.CoveredOrderIDs, settlement.CoveredOrderIDs)
	tempSettleData.CurrentMessageID = messageIDToEdit
	tempSettleData.DriverCalculatedSalary = settlement.DriverCalculatedSalary
	tempSettleData.AmountToCashier = settlement.AmountToCashier

	tempSettleData.DriverUserIDForBackNav = settlement.DriverUserID
	tempSettleData.ViewTypeForBackNav = viewTypeForBackNav
	tempSettleData.PageForBackNav = pageForBackNav

	bh.Deps.SessionManager.UpdateTempDriverSettlement(chatID, tempSettleData)
	bh.SendOwnerEditSettlementFieldSelectMenu(chatID, settlement, messageIDToEdit)
}

// SendOwnerEditSettlementFieldSelectMenu - меню выбора поля для редактирования отчета владельцем.
func (bh *BotHandler) SendOwnerEditSettlementFieldSelectMenu(chatID int64, settlement models.DriverSettlement, messageIDToEdit int) {
	currentState := bh.Deps.SessionManager.GetState(chatID)
	if currentState != constants.STATE_OWNER_CASH_EDIT_SETTLEMENT_FIELD_SELECT {
		bh.Deps.SessionManager.SetState(chatID, constants.STATE_OWNER_CASH_EDIT_SETTLEMENT_FIELD_SELECT)
	}

	tempData := bh.Deps.SessionManager.GetTempDriverSettlement(chatID)
	displayRevenue := settlement.CoveredOrdersRevenue
	displayFuel := settlement.FuelExpense
	displayLoaders := settlement.LoaderPayments

	// --- ИЗМЕНЕНИЕ ДЛЯ OtherExpenses ---
	var totalOtherExpenses float64
	var currentOtherExpensesInSession []models.OtherExpenseDetail

	if tempData.EditingSettlementID == settlement.ID {
		displayRevenue = tempData.CoveredOrdersRevenue
		displayFuel = tempData.FuelExpense
		currentOtherExpensesInSession = tempData.OtherExpenses // Берем из сессии
		displayLoaders = tempData.LoaderPayments
	} else {
		currentOtherExpensesInSession = settlement.OtherExpenses // Берем из полученного объекта settlement
	}

	for _, oe := range currentOtherExpensesInSession {
		totalOtherExpenses += oe.Amount
	}
	otherExpensesStr := fmt.Sprintf("%.0f (%d шт.)", totalOtherExpenses, len(currentOtherExpensesInSession))
	if len(currentOtherExpensesInSession) == 0 {
		otherExpensesStr = "0 (нет)"
	}
	// --- КОНЕЦ ИЗМЕНЕНИЯ ДЛЯ OtherExpenses ---

	text := fmt.Sprintf("✏️ Редактирование Отчета Водителя #%d (от %s)\n\nТекущие значения (могут быть изменены в сессии):\n",
		settlement.ID, settlement.SettlementTimestamp.Format("02.01.06 15:04"))
	text += fmt.Sprintf("💰Выручка: %.0f\n", displayRevenue)
	text += fmt.Sprintf("⛽️Топливо: %.0f\n", displayFuel)
	text += fmt.Sprintf("  Прочие: %s\n", otherExpensesStr) // ИЗМЕНЕНО

	if len(displayLoaders) > 0 {
		text += "  Грузчики:\n"
		for _, lp := range displayLoaders {
			text += fmt.Sprintf("    - %s: %.0f ₽\n", utils.EscapeTelegramMarkdown(lp.LoaderIdentifier), lp.Amount)
		}
	}
	text += "\nКакое поле изменить?"

	var rows [][]tgbotapi.InlineKeyboardButton
	rows = append(rows, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("💰 Выручка", fmt.Sprintf("%s_%d_field_revenue", constants.CALLBACK_PREFIX_OWNER_EDIT_SETTLEMENT, settlement.ID)),
		tgbotapi.NewInlineKeyboardButtonData("⛽ Топливо", fmt.Sprintf("%s_%d_field_fuel", constants.CALLBACK_PREFIX_OWNER_EDIT_SETTLEMENT, settlement.ID)),
	))
	rows = append(rows, tgbotapi.NewInlineKeyboardRow(
		// --- ИЗМЕНЕНИЕ: Коллбэк для прочих расходов теперь другой (ведет в меню управления ими) ---
		tgbotapi.NewInlineKeyboardButtonData("🛠️ Прочие (Ред.)", fmt.Sprintf("%s_%d_field_other_menu", constants.CALLBACK_PREFIX_OWNER_EDIT_SETTLEMENT, settlement.ID)),
		// --- КОНЕЦ ИЗМЕНЕНИЯ ---
		tgbotapi.NewInlineKeyboardButtonData("👷 Грузчики (Ред.)", fmt.Sprintf("%s_%d_field_loaders", constants.CALLBACK_PREFIX_OWNER_EDIT_SETTLEMENT, settlement.ID)),
	))
	rows = append(rows, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("✅ Сохранить изменения и пересчитать", fmt.Sprintf("%s_%d_save", constants.CALLBACK_PREFIX_OWNER_EDIT_SETTLEMENT, settlement.ID)),
	))

	backCallback := fmt.Sprintf("%s_%d_%s_%d",
		constants.CALLBACK_PREFIX_OWNER_VIEW_DRIVER_SETTLEMENTS,
		tempData.DriverUserIDForBackNav, // Это должно быть установлено в handleOwnerEditSettlementStart
		tempData.ViewTypeForBackNav,
		tempData.PageForBackNav)
	if tempData.DriverUserIDForBackNav == 0 { // Фоллбэк, если контекст не был установлен
		backCallback = constants.CALLBACK_PREFIX_OWNER_CASH_MANAGEMENT_MAIN
	}

	rows = append(rows, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("⬅️ Назад к отчетам водителя", backCallback),
	))
	rows = append(rows, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("💰 В меню ДС", constants.CALLBACK_PREFIX_OWNER_CASH_MANAGEMENT_MAIN),
	))

	keyboard := tgbotapi.NewInlineKeyboardMarkup(rows...)
	sentMsg, err := bh.sendOrEditMessageHelper(chatID, messageIDToEdit, text, &keyboard, tgbotapi.ModeMarkdown)
	if err == nil && sentMsg.MessageID != 0 {
		currentTempData := bh.Deps.SessionManager.GetTempDriverSettlement(chatID)
		currentTempData.CurrentMessageID = sentMsg.MessageID
		bh.Deps.SessionManager.UpdateTempDriverSettlement(chatID, currentTempData)
	}
}

// handleOwnerEditSettlementFieldPrompt - запрос нового значения для поля отчета (владелец).
func (bh *BotHandler) handleOwnerEditSettlementFieldPrompt(chatID int64, user models.User, settlementID int64, fieldKey string, messageIDToEdit int) {
	bh.Deps.SessionManager.SetState(chatID, constants.STATE_OWNER_CASH_EDIT_SETTLEMENT_FIELD)
	tempData := bh.Deps.SessionManager.GetTempDriverSettlement(chatID)

	if tempData.EditingSettlementID != settlementID && settlementID != 0 {
		log.Printf("handleOwnerEditSettlementFieldPrompt: ID отчета в сессии (%d) не совпадает с ID из коллбэка (%d). Загрузка актуальных данных для отчета #%d.", tempData.EditingSettlementID, settlementID, settlementID)
		settlementFromDB, errDB := db.GetDriverSettlementByID(settlementID)
		if errDB != nil {
			bh.sendErrorMessageHelper(chatID, messageIDToEdit, "Ошибка загрузки данных отчета. Пожалуйста, выберите отчет заново.")
			bh.SendOwnerCashManagementMenu(chatID, user, messageIDToEdit)
			return
		}
		tempData.EditingSettlementID = settlementFromDB.ID
		tempData.SettlementCreateTime = settlementFromDB.SettlementTimestamp
		tempData.OriginalPaidToOwnerAt = settlementFromDB.PaidToOwnerAt
		tempData.DriverUserIDForBackNav = settlementFromDB.DriverUserID
	}

	tempData.FieldToEditByOwner = fieldKey
	tempData.CurrentMessageID = messageIDToEdit
	bh.Deps.SessionManager.UpdateTempDriverSettlement(chatID, tempData)

	var promptText string
	switch fieldKey {
	case "revenue":
		promptText = fmt.Sprintf("✏️ Введите новую общую выручку для Отчета #%d:", settlementID)
	case "fuel":
		promptText = fmt.Sprintf("✏️ Введите новую сумму на топливо для Отчета #%d:", settlementID)
	case "other_menu": // ИЗМЕНЕНО: это теперь переход в меню управления прочими расходами
		log.Printf("Переход в меню управления прочими расходами для отчета #%d (владелец)", settlementID)
		// Здесь мы должны вызвать функцию, аналогичную SendDriverReportOtherExpensesMenu,
		// но адаптированную для владельца и контекста редактирования отчета.
		// Пока такой функции нет, можно временно вывести сообщение или вернуть в меню выбора поля.
		// Для полноценной реализации потребуется новый набор состояний и коллбэков для владельца.
		// ВРЕМЕННО:
		bh.sendInfoMessage(chatID, messageIDToEdit, "Управление детальными прочими расходами здесь пока не реализовано. Используйте старое поле 'Прочие', если оно есть, или отмените.",
			fmt.Sprintf("%s_%d", constants.CALLBACK_PREFIX_OWNER_EDIT_SETTLEMENT, settlementID))
		bh.Deps.SessionManager.SetState(chatID, constants.STATE_OWNER_CASH_EDIT_SETTLEMENT_FIELD_SELECT)
		return

	case "loaders": // Аналогично, для грузчиков может потребоваться отдельное меню
		bh.sendInfoMessage(chatID, messageIDToEdit, "Редактирование зарплат грузчиков в этом интерфейсе пока не реализовано детально.",
			fmt.Sprintf("%s_%d", constants.CALLBACK_PREFIX_OWNER_EDIT_SETTLEMENT, settlementID))
		bh.Deps.SessionManager.SetState(chatID, constants.STATE_OWNER_CASH_EDIT_SETTLEMENT_FIELD_SELECT)
		return
	default:
		// Если пришел старый "other"
		if fieldKey == "other" {
			promptText = fmt.Sprintf("✏️ Введите *ОБЩУЮ* сумму прочих расходов для Отчета #%d (0 если нет). Детальное редактирование прочих расходов через отдельное меню.", settlementID)
		} else {
			bh.sendErrorMessageHelper(chatID, messageIDToEdit, "Неизвестное поле для редактирования.")
			settlementFromDB, _ := db.GetDriverSettlementByID(settlementID)
			bh.SendOwnerEditSettlementFieldSelectMenu(chatID, settlementFromDB, messageIDToEdit)
			return
		}
	}

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("⬅️ Назад к выбору поля", fmt.Sprintf("%s_%d", constants.CALLBACK_PREFIX_OWNER_EDIT_SETTLEMENT, settlementID)),
		),
	)
	sentMsg, err := bh.sendOrEditMessageHelper(chatID, messageIDToEdit, promptText, &keyboard, "")
	if err == nil && sentMsg.MessageID != 0 {
		currentTempData := bh.Deps.SessionManager.GetTempDriverSettlement(chatID)
		currentTempData.CurrentMessageID = sentMsg.MessageID
		bh.Deps.SessionManager.UpdateTempDriverSettlement(chatID, currentTempData)
	}
}

// handleOwnerSaveEditedSettlementFieldInput - обработка ввода нового значения поля отчета (владелец)
func (bh *BotHandler) handleOwnerSaveEditedSettlementFieldInput(chatID int64, user models.User, textInput string, userMsgID int, botMenuMsgID int) {
	bh.deleteMessageHelper(chatID, userMsgID)
	tempData := bh.Deps.SessionManager.GetTempDriverSettlement(chatID)
	settlementID := tempData.EditingSettlementID
	fieldKey := tempData.FieldToEditByOwner

	if settlementID == 0 {
		bh.sendErrorMessageHelper(chatID, botMenuMsgID, "Ошибка: не выбран отчет для редактирования.")
		bh.SendOwnerCashManagementMenu(chatID, user, botMenuMsgID)
		return
	}

	originalSettlement, errDB := db.GetDriverSettlementByID(settlementID)
	if errDB != nil {
		bh.sendErrorMessageHelper(chatID, botMenuMsgID, "Ошибка загрузки данных отчета для редактирования.")
		bh.SendOwnerCashManagementMenu(chatID, user, botMenuMsgID)
		return
	}

	val, err := strconv.ParseFloat(strings.Replace(textInput, ",", ".", -1), 64)
	if err != nil {
		bh.sendErrorMessageHelper(chatID, botMenuMsgID, "Некорректное числовое значение. Попробуйте снова.")
		bh.SendOwnerEditSettlementFieldSelectMenu(chatID, originalSettlement, botMenuMsgID)
		return
	}

	switch fieldKey {
	case "revenue":
		tempData.CoveredOrdersRevenue = val
	case "fuel":
		tempData.FuelExpense = val
	case "other": // Обработка старого поля "other"
		if val == 0 {
			// Если владелец вводит 0 для старого "other", мы очищаем список детальных OtherExpenses.
			// Это упрощение, так как нет интерфейса для ввода ОБЩЕЙ суммы, если есть детальные.
			tempData.OtherExpenses = []models.OtherExpenseDetail{}
		} else {
			// Если вводится ненулевая ОБЩАЯ сумма, а детальных расходов не было,
			// создаем одну запись "Прочие расходы (общая сумма)"
			if len(tempData.OtherExpenses) == 0 {
				tempData.OtherExpenses = []models.OtherExpenseDetail{{Description: "Прочие расходы (общая сумма)", Amount: val}}
			} else {
				// Если уже были детальные расходы, а владелец вводит общую сумму, это конфликт.
				// Пока просто перезаписываем первым элементом. Более сложная логика потребует UX решения.
				log.Printf("ВНИМАНИЕ: Владелец перезаписывает детальные прочие расходы общей суммой для отчета #%d", settlementID)
				tempData.OtherExpenses = []models.OtherExpenseDetail{{Description: "Прочие расходы (общая сумма, перезаписано)", Amount: val}}
			}
		}
	default:
		bh.sendErrorMessageHelper(chatID, botMenuMsgID, "Ошибка: неизвестное поле для сохранения.")
		bh.SendOwnerEditSettlementFieldSelectMenu(chatID, originalSettlement, botMenuMsgID)
		return
	}

	bh.Deps.SessionManager.UpdateTempDriverSettlement(chatID, tempData)
	bh.SendOwnerEditSettlementFieldSelectMenu(chatID, originalSettlement, botMenuMsgID)
}

// handleOwnerSaveAllSettlementChanges - сохранение всех изменений в отчете и пересчет.
func (bh *BotHandler) handleOwnerSaveAllSettlementChanges(chatID int64, user models.User, settlementID int64, messageIDToEdit int) {
	tempData := bh.Deps.SessionManager.GetTempDriverSettlement(chatID)
	if tempData.EditingSettlementID != settlementID || settlementID == 0 {
		bh.sendErrorMessageHelper(chatID, messageIDToEdit, "Ошибка: контекст редактирования отчета потерян.")
		bh.SendOwnerCashManagementMenu(chatID, user, messageIDToEdit)
		return
	}

	originalSettlement, errDB := db.GetDriverSettlementByID(settlementID)
	if errDB != nil {
		bh.sendErrorMessageHelper(chatID, messageIDToEdit, "Ошибка загрузки оригинального отчета для сохранения.")
		bh.SendOwnerCashManagementMenu(chatID, user, messageIDToEdit)
		return
	}

	settlementToSave := models.DriverSettlement{
		ID:                     tempData.EditingSettlementID,
		DriverUserID:           originalSettlement.DriverUserID,
		SettlementTimestamp:    tempData.SettlementCreateTime,
		CoveredOrdersRevenue:   tempData.CoveredOrdersRevenue,
		FuelExpense:            tempData.FuelExpense,
		OtherExpenses:          tempData.OtherExpenses, // ИСПОЛЬЗУЕМ ОБНОВЛЕННОЕ ПОЛЕ
		LoaderPayments:         tempData.LoaderPayments,
		CoveredOrdersCount:     originalSettlement.CoveredOrdersCount,
		CoveredOrderIDs:        originalSettlement.CoveredOrderIDs,
		PaidToOwnerAt:          tempData.OriginalPaidToOwnerAt,
		DriverCalculatedSalary: 0,
		AmountToCashier:        0,
	}
	if tempData.SettlementCreateTime.IsZero() {
		settlementToSave.SettlementTimestamp = originalSettlement.SettlementTimestamp
	}

	// Пересчет общей суммы OtherExpense для сохранения, если это все еще старое поле в БД
	// и если tempData.OtherExpenses не пустое. Если OtherExpenses пустое, и в старом other_expense был 0, то это ок.
	// НО! Мы перешли на other_expenses_json, поэтому RecalculateTotals в UpdateDriverSettlement должен это учесть.

	err := db.UpdateDriverSettlement(settlementToSave)
	if err != nil {
		bh.sendErrorMessageHelper(chatID, messageIDToEdit, fmt.Sprintf("Ошибка сохранения изменений в отчете #%d: %v", settlementID, err))
		bh.SendOwnerEditSettlementFieldSelectMenu(chatID, originalSettlement, messageIDToEdit)
		return
	}

	updatedSettlement, _ := db.GetDriverSettlementByID(settlementID)

	backToListCallback := fmt.Sprintf("%s_%d_%s_%d",
		constants.CALLBACK_PREFIX_OWNER_VIEW_DRIVER_SETTLEMENTS,
		updatedSettlement.DriverUserID,
		tempData.ViewTypeForBackNav,
		tempData.PageForBackNav)

	bh.sendInfoMessage(chatID, messageIDToEdit, fmt.Sprintf("✅ Изменения в Отчете #%d сохранены и суммы пересчитаны.", settlementID),
		backToListCallback)

	currentTempData := bh.Deps.SessionManager.GetTempDriverSettlement(chatID)
	currentTempData.EditingSettlementID = 0
	currentTempData.FieldToEditByOwner = ""
	currentTempData.SettlementCreateTime = time.Time{}
	currentTempData.OriginalPaidToOwnerAt = sql.NullTime{}
	// OtherExpenses очищать не нужно, если мы хотим, чтобы они сохранились для следующего входа в редактирование *этого же* отчета
	// Но если это был отчет водителя и владелец его редактировал, то при следующем входе водителя в создание отчета, OtherExpenses должны быть пустыми.
	// Пока оставляем их в сессии, если EditingSettlementID сброшен, они не должны влиять на новый отчет.
	bh.Deps.SessionManager.UpdateTempDriverSettlement(chatID, currentTempData)
	bh.Deps.SessionManager.SetState(chatID, constants.STATE_OWNER_CASH_VIEW_DRIVER_SETTLEMENTS)
}
