package handlers

import (
	"Original/internal/constants"
	"Original/internal/db"
	"Original/internal/models"
	"Original/internal/utils"
	"fmt"
	"log"
	"math"
	"strings"

	tgbotapi "github.com/OvyFlash/telegram-bot-api"
)

// SendOwnerCashManagementMenu - главное меню "Денежные средства" для владельца (новое).
func (bh *BotHandler) SendOwnerCashManagementMenu(chatID int64, user models.User, messageIDToEdit int) {
	log.Printf("SendOwnerCashManagementMenu: для владельца ChatID=%d, messageIDToEdit=%d", chatID, messageIDToEdit)
	bh.Deps.SessionManager.SetState(chatID, constants.STATE_OWNER_CASH_MANAGEMENT_MENU)

	bh.Deps.SessionManager.ClearTempOrder(chatID)
	bh.Deps.SessionManager.ClearTempDriverSettlement(chatID)

	msgText := "💰 Управление денежными средствами:\n\nВыберите категорию для просмотра:"
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("❗️ Актуальные (кто должен)", fmt.Sprintf("%s_0", constants.CALLBACK_PREFIX_OWNER_CASH_ACTUAL_LIST)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("✅ Рассчитанные (кто внес/кому выплачено)", fmt.Sprintf("%s_0", constants.CALLBACK_PREFIX_OWNER_CASH_SETTLED_LIST)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔙 Главное меню", "back_to_main"),
		),
	)
	sentMsg, err := bh.sendOrEditMessageHelper(chatID, messageIDToEdit, msgText, &keyboard, tgbotapi.ModeMarkdown)
	if err != nil {
		log.Printf("SendOwnerCashManagementMenu: Ошибка для chatID %d: %v", chatID, err)
	} else {
		if sentMsg.MessageID != 0 {
			// CurrentMessageID для этого меню обрабатывается через TempOrder sendOrEditMessageHelper
		}
	}
}

// SendOwnerActualDebtsList - отображает АГРЕГИРОВАННЫЙ список актуальных долгов водителей.
func (bh *BotHandler) SendOwnerActualDebtsList(chatID int64, user models.User, messageIDToEdit int, page int) {
	log.Printf("SendOwnerActualDebtsList: для владельца ChatID=%d, страница %d, messageIDToEdit %d", chatID, page, messageIDToEdit)
	bh.Deps.SessionManager.SetState(chatID, constants.STATE_OWNER_CASH_ACTUAL_LIST)

	aggregatedDriverData, totalDrivers, err := db.GetAggregatedDriverSettlements(constants.VIEW_TYPE_ACTUAL_SETTLEMENTS)
	if err != nil {
		log.Printf("SendOwnerActualDebtsList: ошибка получения агрегированных актуальных долгов: %v", err)
		bh.sendErrorMessageHelper(chatID, messageIDToEdit, "❌ Ошибка загрузки списка должников.")
		return
	}

	var msgText string
	var rows [][]tgbotapi.InlineKeyboardButton

	start := page * constants.CashRecordsPerPage
	end := start + constants.CashRecordsPerPage
	var paginatedAggregatedData []models.AggregatedDriverSettlementInfo

	if start >= len(aggregatedDriverData) {
		paginatedAggregatedData = []models.AggregatedDriverSettlementInfo{}
	} else if end > len(aggregatedDriverData) {
		paginatedAggregatedData = aggregatedDriverData[start:]
	} else {
		paginatedAggregatedData = aggregatedDriverData[start:end]
	}

	if len(paginatedAggregatedData) == 0 && page == 0 {
		msgText = "✅ Все водители рассчитались, актуальных долгов нет (ожидающих внесения денег И выплаты ЗП)."
	} else if len(paginatedAggregatedData) == 0 && page > 0 {
		msgText = "✅ Больше водителей с актуальными долгами нет."
	} else {
		msgText = "❗️ *Актуальные (ожидается внесение в кассу И выплата ЗП водителю):*\n\n"
		for _, aggData := range paginatedAggregatedData {
			driverModelsUser := models.User{
				FirstName: aggData.DriverFirstName.String,
				LastName:  aggData.DriverLastName.String,
				Nickname:  aggData.DriverNickname,
				ChatID:    aggData.DriverUserID,
			}
			driverName := utils.GetUserDisplayName(driverModelsUser)

			buttonText := fmt.Sprintf("👤 %s | *%.0f ₽* (%d отч.)",
				utils.EscapeTelegramMarkdown(driverName),
				aggData.TotalAmountToCashier,
				aggData.TotalReportsCount)

			callbackData := fmt.Sprintf("%s_%d_%s_0",
				constants.CALLBACK_PREFIX_OWNER_VIEW_DRIVER_SETTLEMENTS,
				aggData.DriverUserID,
				constants.VIEW_TYPE_ACTUAL_SETTLEMENTS,
			)
			rows = append(rows, tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData(buttonText, callbackData),
			))
		}
	}

	totalPages := 0
	if totalDrivers > 0 && constants.CashRecordsPerPage > 0 {
		totalPages = int(math.Ceil(float64(totalDrivers) / float64(constants.CashRecordsPerPage)))
	}

	navRow := []tgbotapi.InlineKeyboardButton{}
	if page > 0 {
		navRow = append(navRow, tgbotapi.NewInlineKeyboardButtonData("⬅️ Пред.", fmt.Sprintf("%s_%d", constants.CALLBACK_PREFIX_OWNER_CASH_ACTUAL_LIST, page-1)))
	}
	if page < totalPages-1 {
		navRow = append(navRow, tgbotapi.NewInlineKeyboardButtonData("След. ➡️", fmt.Sprintf("%s_%d", constants.CALLBACK_PREFIX_OWNER_CASH_ACTUAL_LIST, page+1)))
	}
	if len(navRow) > 0 {
		rows = append(rows, navRow)
	}

	rows = append(rows, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("🔙 В меню ДС", constants.CALLBACK_PREFIX_OWNER_CASH_MANAGEMENT_MAIN),
	))
	keyboard := tgbotapi.NewInlineKeyboardMarkup(rows...)

	_, errSend := bh.sendOrEditMessageHelper(chatID, messageIDToEdit, msgText, &keyboard, tgbotapi.ModeMarkdown)
	if errSend != nil {
		log.Printf("SendOwnerActualDebtsList (Aggregated): Ошибка для chatID %d: %v", chatID, errSend)
	}
}

// SendOwnerSettledPaymentsList - отображает АГРЕГИРОВАННЫЙ список уже внесенных/выплаченных сумм по водителям.
func (bh *BotHandler) SendOwnerSettledPaymentsList(chatID int64, user models.User, messageIDToEdit int, page int) {
	log.Printf("SendOwnerSettledPaymentsList: для владельца ChatID=%d, страница %d, messageIDToEdit %d", chatID, page, messageIDToEdit)
	bh.Deps.SessionManager.SetState(chatID, constants.STATE_OWNER_CASH_SETTLED_LIST)

	aggregatedDriverData, totalDrivers, err := db.GetAggregatedDriverSettlements(constants.VIEW_TYPE_SETTLED_SETTLEMENTS)
	if err != nil {
		log.Printf("SendOwnerSettledPaymentsList: ошибка получения агрегированных рассчитанных: %v", err)
		bh.sendErrorMessageHelper(chatID, messageIDToEdit, "❌ Ошибка загрузки списка рассчитанных.")
		return
	}

	var msgText string
	var rows [][]tgbotapi.InlineKeyboardButton

	start := page * constants.CashRecordsPerPage
	end := start + constants.CashRecordsPerPage
	var paginatedAggregatedData []models.AggregatedDriverSettlementInfo

	if start >= len(aggregatedDriverData) {
		paginatedAggregatedData = []models.AggregatedDriverSettlementInfo{}
	} else if end > len(aggregatedDriverData) {
		paginatedAggregatedData = aggregatedDriverData[start:]
	} else {
		paginatedAggregatedData = aggregatedDriverData[start:end]
	}

	if len(paginatedAggregatedData) == 0 && page == 0 {
		msgText = "ℹ️ Рассчитанных (деньги внесены И/ИЛИ ЗП выплачена) отчетов пока нет."
	} else if len(paginatedAggregatedData) == 0 && page > 0 {
		msgText = "ℹ️ Больше рассчитанных отчетов нет."
	} else {
		msgText = "✅ *Рассчитанные (деньги внесены в кассу И/ИЛИ ЗП водителю выплачена):*\n\n"
		for _, aggData := range paginatedAggregatedData {
			driverModelsUser := models.User{
				FirstName: aggData.DriverFirstName.String,
				LastName:  aggData.DriverLastName.String,
				Nickname:  aggData.DriverNickname,
				ChatID:    aggData.DriverUserID,
			}
			driverName := utils.GetUserDisplayName(driverModelsUser)

			buttonText := fmt.Sprintf("👤 %s | *%.0f ₽* (%d отч.)",
				utils.EscapeTelegramMarkdown(driverName),
				aggData.TotalAmountToCashier,
				aggData.TotalReportsCount)

			callbackData := fmt.Sprintf("%s_%d_%s_0",
				constants.CALLBACK_PREFIX_OWNER_VIEW_DRIVER_SETTLEMENTS,
				aggData.DriverUserID,
				constants.VIEW_TYPE_SETTLED_SETTLEMENTS,
			)
			rows = append(rows, tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData(buttonText, callbackData),
			))
		}
	}

	totalPages := 0
	if totalDrivers > 0 && constants.CashRecordsPerPage > 0 {
		totalPages = int(math.Ceil(float64(totalDrivers) / float64(constants.CashRecordsPerPage)))
	}
	navRow := []tgbotapi.InlineKeyboardButton{}
	if page > 0 {
		navRow = append(navRow, tgbotapi.NewInlineKeyboardButtonData("⬅️ Пред.", fmt.Sprintf("%s_%d", constants.CALLBACK_PREFIX_OWNER_CASH_SETTLED_LIST, page-1)))
	}
	if page < totalPages-1 {
		navRow = append(navRow, tgbotapi.NewInlineKeyboardButtonData("След. ➡️", fmt.Sprintf("%s_%d", constants.CALLBACK_PREFIX_OWNER_CASH_SETTLED_LIST, page+1)))
	}
	if len(navRow) > 0 {
		rows = append(rows, navRow)
	}

	rows = append(rows, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("🔙 В меню ДС", constants.CALLBACK_PREFIX_OWNER_CASH_MANAGEMENT_MAIN),
	))
	keyboard := tgbotapi.NewInlineKeyboardMarkup(rows...)

	_, errSend := bh.sendOrEditMessageHelper(chatID, messageIDToEdit, msgText, &keyboard, tgbotapi.ModeMarkdown)
	if errSend != nil {
		log.Printf("SendOwnerSettledPaymentsList (Aggregated): Ошибка для chatID %d: %v", chatID, errSend)
	}
}

// SendOwnerDriverIndividualSettlementsList - отображает список индивидуальных отчетов водителя.
func (bh *BotHandler) SendOwnerDriverIndividualSettlementsList(chatID int64, user models.User, messageIDToEdit int, driverUserID int64, viewType string, page int) {
	log.Printf("SendOwnerDriverIndividualSettlementsList: ChatID=%d, DriverUserID=%d, ViewType=%s, Page=%d, MsgID=%d",
		chatID, driverUserID, viewType, page, messageIDToEdit)

	bh.Deps.SessionManager.SetState(chatID, constants.STATE_OWNER_CASH_VIEW_DRIVER_SETTLEMENTS)
	tempData := bh.Deps.SessionManager.GetTempDriverSettlement(chatID)
	tempData.EditingSettlementID = 0
	tempData.FieldToEditByOwner = ""
	tempData.DriverUserIDForBackNav = driverUserID
	tempData.ViewTypeForBackNav = viewType
	tempData.PageForBackNav = page
	bh.Deps.SessionManager.UpdateTempDriverSettlement(chatID, tempData)

	settlements, totalRecords, err := db.GetDriverSettlementsForOwnerView(driverUserID, viewType, page, constants.CashRecordsPerPage)
	if err != nil {
		log.Printf("SendOwnerDriverIndividualSettlementsList: ошибка получения отчетов для водителя %d (viewType: %s): %v", driverUserID, viewType, err)
		bh.sendErrorMessageHelper(chatID, messageIDToEdit, "❌ Ошибка загрузки отчетов водителя.")
		if viewType == constants.VIEW_TYPE_SETTLED_SETTLEMENTS {
			bh.SendOwnerSettledPaymentsList(chatID, user, messageIDToEdit, 0)
		} else {
			bh.SendOwnerActualDebtsList(chatID, user, messageIDToEdit, 0)
		}
		return
	}

	driver, errDriver := db.GetUserByID(int(driverUserID))
	driverDisplayName := fmt.Sprintf("Водитель ID %d", driverUserID)
	if errDriver == nil {
		driverDisplayName = utils.GetUserDisplayName(driver)
	}

	var msgText string
	var rows [][]tgbotapi.InlineKeyboardButton

	viewTitle := "❗️ Актуальные отчеты (ожидается внесение денег И выплата ЗП)"
	if viewType == constants.VIEW_TYPE_SETTLED_SETTLEMENTS {
		viewTitle = "✅ Частично или полностью рассчитанные отчеты"
	}
	msgText = fmt.Sprintf("%s водителя *%s*:\n\n", viewTitle, utils.EscapeTelegramMarkdown(driverDisplayName))

	if len(settlements) == 0 && page == 0 {
		msgText += "Нет отчетов для отображения в этой категории."
	} else if len(settlements) == 0 && page > 0 {
		msgText += "Больше отчетов нет."
	} else {
		for _, s := range settlements {
			reportDateStr := s.SettlementTimestamp.Format("02.01.06 15:04")

			statusParts := []string{}
			moneyIn := s.PaidToOwnerAt.Valid
			salaryPaid := s.DriverSalaryPaidAt.Valid

			if moneyIn {
				statusParts = append(statusParts, fmt.Sprintf("Деньги ✅: %s", s.PaidToOwnerAt.Time.Format("02.01.06")))
			} else {
				statusParts = append(statusParts, "Деньги ❌")
			}
			if salaryPaid {
				statusParts = append(statusParts, fmt.Sprintf("ЗП ✅: %s", s.DriverSalaryPaidAt.Time.Format("02.01.06")))
			} else {
				statusParts = append(statusParts, "ЗП ❌")
			}
			statusLine := strings.Join(statusParts, ", ")

			if moneyIn && salaryPaid {
				statusLine += " (Рассчитан ✅)"
			}

			msgText += fmt.Sprintf("📝 *Отчет #%d* от %s (%s)\n", s.ID, reportDateStr, statusLine)
			msgText += "📦Заказы: "
			if len(s.CoveredOrderIDs) > 0 {
				for i, orderID := range s.CoveredOrderIDs {
					msgText += fmt.Sprintf("#%d", orderID)
					if i < len(s.CoveredOrderIDs)-1 {
						msgText += ", "
					}
				}
			} else {
				msgText += "не указаны"
			}
			msgText += fmt.Sprintf("\n💰Выручка: %.0f ₽\n⛽️Топливо: %.0f ₽\n", s.CoveredOrdersRevenue, s.FuelExpense)

			// --- ИЗМЕНЕНИЕ ОТОБРАЖЕНИЯ OtherExpenses ---
			if len(s.OtherExpenses) > 0 {
				msgText += "🛠️Прочие:\n"
				for _, oe := range s.OtherExpenses {
					msgText += fmt.Sprintf("    - %s: %.0f ₽\n", utils.EscapeTelegramMarkdown(oe.Description), oe.Amount)
				}
			} else {
				msgText += "🛠️Прочие: 0 ₽\n"
			}
			// --- КОНЕЦ ИЗМЕНЕНИЯ ---

			totalLoadersSalary := 0.0
			for _, lp := range s.LoaderPayments {
				totalLoadersSalary += lp.Amount
			}
			msgText += fmt.Sprintf("\n👷‍ЗП грузчикам: %.0f ₽\n     *Расч. ЗП водителя: %.0f ₽*\n", totalLoadersSalary, s.DriverCalculatedSalary)
			msgText += fmt.Sprintf("     *К сдаче в кассу (было): %.0f ₽*\n\n", s.AmountToCashier)

			var reportActionButtons []tgbotapi.InlineKeyboardButton
			reportActionButtons = append(reportActionButtons, tgbotapi.NewInlineKeyboardButtonData(fmt.Sprintf("✏️ Ред. #%d", s.ID),
				fmt.Sprintf("%s_%d_%d_%s_%d", constants.CALLBACK_PREFIX_OWNER_EDIT_SETTLEMENT, s.ID, driverUserID, viewType, page)))

			if moneyIn {
				reportActionButtons = append(reportActionButtons, tgbotapi.NewInlineKeyboardButtonData("❗️💸 НЕ вносил",
					fmt.Sprintf("%s_%d", constants.CALLBACK_PREFIX_OWNER_CASH_MARK_UNPAID, s.ID)))
			} else {
				reportActionButtons = append(reportActionButtons, tgbotapi.NewInlineKeyboardButtonData("✅💸 внес",
					fmt.Sprintf("%s_%d", constants.CALLBACK_PREFIX_OWNER_CASH_MARK_PAID, s.ID)))
			}

			if salaryPaid {
				reportActionButtons = append(reportActionButtons, tgbotapi.NewInlineKeyboardButtonData("💸 ЗП НЕ получил",
					fmt.Sprintf("%s_%d", constants.CALLBACK_PREFIX_OWNER_CASH_MARK_SALARY_UNPAID, s.ID)))
			} else {
				reportActionButtons = append(reportActionButtons, tgbotapi.NewInlineKeyboardButtonData("💸 ЗП получил",
					fmt.Sprintf("%s_%d", constants.CALLBACK_PREFIX_OWNER_CASH_MARK_SALARY_PAID, s.ID)))
			}
			rows = append(rows, reportActionButtons)
		}
	}

	if len(settlements) > 0 {
		var totalDriverSalary float64
		var totalAmountToCashier float64
		for _, s := range settlements {
			totalDriverSalary += s.DriverCalculatedSalary
			totalAmountToCashier += s.AmountToCashier
		}
		summaryText := fmt.Sprintf("\n--------------------------------------------------\n*Итого по показанным отчетам:*\nЗП водителя - *%.0f ₽*\nДенег в кассу (было) - *%.0f ₽*", totalDriverSalary, totalAmountToCashier)
		msgText += summaryText
	}

	if len(settlements) > 0 {
		var allActionsRow []tgbotapi.InlineKeyboardButton
		if viewType == constants.VIEW_TYPE_ACTUAL_SETTLEMENTS {
			canDepositAll := false
			for _, sItem := range settlements {
				if !sItem.PaidToOwnerAt.Valid {
					canDepositAll = true
					break
				}
			}
			if canDepositAll {
				allActionsRow = append(allActionsRow, tgbotapi.NewInlineKeyboardButtonData("💰 Деньги внес за все (на стр.)",
					fmt.Sprintf("%s_%d_%s", constants.CALLBACK_PREFIX_OWNER_MARK_ALL_MONEY_DEPOSITED, driverUserID, viewType)))
			}
		}

		canPaySalaryForAll := false
		for _, sItem := range settlements {
			if !sItem.DriverSalaryPaidAt.Valid {
				canPaySalaryForAll = true
				break
			}
		}
		if canPaySalaryForAll {
			allActionsRow = append(allActionsRow, tgbotapi.NewInlineKeyboardButtonData("💵 ЗП получил за все (на стр.)",
				fmt.Sprintf("%s_%d_%s", constants.CALLBACK_PREFIX_OWNER_MARK_ALL_SALARY_PAID, driverUserID, viewType)))
		}

		if len(allActionsRow) > 0 {
			rows = append(rows, allActionsRow)
		}
	}

	totalPages := 0
	if totalRecords > 0 && constants.CashRecordsPerPage > 0 {
		totalPages = int(math.Ceil(float64(totalRecords) / float64(constants.CashRecordsPerPage)))
	}
	navRow := []tgbotapi.InlineKeyboardButton{}
	callbackPrefixForPagination := fmt.Sprintf("%s_%d_%s", constants.CALLBACK_PREFIX_OWNER_VIEW_DRIVER_SETTLEMENTS, driverUserID, viewType)

	if page > 0 {
		navRow = append(navRow, tgbotapi.NewInlineKeyboardButtonData("⬅️ Пред.", fmt.Sprintf("%s_%d", callbackPrefixForPagination, page-1)))
	}
	if page < totalPages-1 {
		navRow = append(navRow, tgbotapi.NewInlineKeyboardButtonData("След. ➡️", fmt.Sprintf("%s_%d", callbackPrefixForPagination, page+1)))
	}
	if len(navRow) > 0 {
		rows = append(rows, navRow)
	}

	backToListCallback := constants.CALLBACK_PREFIX_OWNER_CASH_ACTUAL_LIST + "_0"
	if viewType == constants.VIEW_TYPE_SETTLED_SETTLEMENTS {
		backToListCallback = constants.CALLBACK_PREFIX_OWNER_CASH_SETTLED_LIST + "_0"
	}
	rows = append(rows, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("🔙 К списку водителей", backToListCallback),
	))
	rows = append(rows, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("💰 В меню ДС", constants.CALLBACK_PREFIX_OWNER_CASH_MANAGEMENT_MAIN),
	))

	keyboard := tgbotapi.NewInlineKeyboardMarkup(rows...)

	sentMsg, errSend := bh.sendOrEditMessageHelper(chatID, messageIDToEdit, msgText, &keyboard, tgbotapi.ModeMarkdown)
	if errSend != nil {
		log.Printf("SendOwnerDriverIndividualSettlementsList: Ошибка для chatID %d: %v", chatID, errSend)
	} else {
		updatedTempData := bh.Deps.SessionManager.GetTempDriverSettlement(chatID)
		updatedTempData.CurrentMessageID = sentMsg.MessageID
		bh.Deps.SessionManager.UpdateTempDriverSettlement(chatID, updatedTempData)
	}
}
