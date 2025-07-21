package handlers

import (
	// "database/sql" // Not used directly here
	"fmt"
	"log"
	// "os"   // Not used here
	"strconv"
	// "strings" // Used in utils
	"time"

	tgbotapi "github.com/OvyFlash/telegram-bot-api"
	// "github.com/xuri/excelize/v2" // Not used here

	"Original/internal/constants"
	"Original/internal/models"
	// "Original/internal/session" // Access via bh.Deps
	"Original/internal/utils"
)

// SendStatsMenu отправляет главное меню статистики.
// SendStatsMenu sends the main statistics menu.
func (bh *BotHandler) SendStatsMenu(chatID int64, messageIDToEdit int) {
	log.Printf("BotHandler.SendStatsMenu для chatID %d, messageIDToEdit: %d", chatID, messageIDToEdit)
	bh.Deps.SessionManager.SetState(chatID, constants.STATE_STATS_MENU)

	// Права доступа проверяются в callback_handler перед вызовом этой функции
	// Access rights are checked in callback_handler before calling this function

	msgText := "📊 Статистика:\n\nВыберите период или действие:"
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Основные периоды", "stats_basic_periods"),
			tgbotapi.NewInlineKeyboardButtonData("Выбрать дату", "stats_select_custom_date"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Выбрать период", "stats_select_custom_period"),
		),
		// Кнопка для генерации Excel отчетов может быть здесь или в отдельном админском меню
		// Button for generating Excel reports can be here or in a separate admin menu
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📑 Excel отчеты", "send_excel_menu"), // Переход в меню Excel / Go to Excel menu
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔙 Главное меню", "back_to_main"),
		),
	)
	_, err := bh.sendOrEditMessageHelper(chatID, messageIDToEdit, msgText, &keyboard, tgbotapi.ModeMarkdown)
	if err != nil {
		log.Printf("SendStatsMenu: Ошибка для chatID %d: %v", chatID, err)
	}
}

// SendBasicStatsPeriodsMenu отправляет меню выбора основных периодов для статистики.
// SendBasicStatsPeriodsMenu sends the basic period selection menu for statistics.
func (bh *BotHandler) SendBasicStatsPeriodsMenu(chatID int64, messageIDToEdit int) {
	log.Printf("BotHandler.SendBasicStatsPeriodsMenu для chatID %d, messageIDToEdit: %d", chatID, messageIDToEdit)
	bh.Deps.SessionManager.SetState(chatID, constants.STATE_STATS_DATE) // Состояние ожидания выбора конкретного периода / State awaiting selection of a specific period

	msgText := "📊 Основные периоды для статистики:"
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Сегодня", "stats_get_today"),
			tgbotapi.NewInlineKeyboardButtonData("Вчера", "stats_get_yesterday"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Текущая неделя", "stats_get_current_week"),
			tgbotapi.NewInlineKeyboardButtonData("Текущий месяц", "stats_get_current_month"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Прошлая неделя", "stats_get_last_week"),
			tgbotapi.NewInlineKeyboardButtonData("Прошлый месяц", "stats_get_last_month"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔙 Назад в меню статистики", "stats_menu"),
		),
	)
	_, err := bh.sendOrEditMessageHelper(chatID, messageIDToEdit, msgText, &keyboard, tgbotapi.ModeMarkdown)
	if err != nil {
		log.Printf("SendBasicStatsPeriodsMenu: Ошибка для chatID %d: %v", chatID, err)
	}
}

// SendMonthSelectionMenu отправляет меню выбора месяца для статистики.
// year - год для выбора месяца.
// context - "custom_date", "period_start", "period_end" для формирования правильных коллбэков.
// SendMonthSelectionMenu sends the month selection menu for statistics.
// year - year for month selection.
// context - "custom_date", "period_start", "period_end" for forming correct callbacks.
func (bh *BotHandler) SendMonthSelectionMenu(chatID int64, messageIDToEdit int, year int, context string) {
	log.Printf("BotHandler.SendMonthSelectionMenu для chatID %d, год: %d, контекст: %s, messageIDToEdit: %d", chatID, year, context, messageIDToEdit)

	var stateToSet string
	switch context {
	case "custom_date":
		stateToSet = constants.STATE_STATS_MONTH // Для выбора месяца конкретной даты / For selecting month of a specific date
	case "period_start":
		stateToSet = constants.STATE_STATS_PERIOD // Для выбора начала периода (месяц) / For selecting start of period (month)
	case "period_end":
		stateToSet = constants.STATE_STATS_PERIOD_END // Для выбора конца периода (месяц) / For selecting end of period (month)
	default:
		log.Printf("SendMonthSelectionMenu: Неизвестный контекст '%s'", context)
		bh.sendErrorMessageHelper(chatID, messageIDToEdit, "Ошибка выбора контекста даты.")
		return
	}
	bh.Deps.SessionManager.SetState(chatID, stateToSet)

	msgText := fmt.Sprintf("📅 Выберите месяц для %d года:", year)
	if context == "period_start" {
		msgText = fmt.Sprintf("📅 Выберите *НАЧАЛЬНЫЙ* месяц для %d года:", year)
	} else if context == "period_end" {
		msgText = fmt.Sprintf("📅 Выберите *КОНЕЧНЫЙ* месяц для %d года:", year)
	}

	var rows [][]tgbotapi.InlineKeyboardButton
	months := []time.Month{
		time.January, time.February, time.March, time.April,
		time.May, time.June, time.July, time.August,
		time.September, time.October, time.November, time.December,
	}

	currentMonthButtons := []tgbotapi.InlineKeyboardButton{}
	for i, month := range months {
		// Коллбэк: stats_select_month_КОНТЕКСТ_ГОД_НОМЕРМЕСЯЦА
		// Callback: stats_select_month_CONTEXT_YEAR_MONTHNUMBER
		callbackData := fmt.Sprintf("stats_select_month_%s_%d_%d", context, year, int(month))
		currentMonthButtons = append(currentMonthButtons, tgbotapi.NewInlineKeyboardButtonData(constants.MonthMap[month], callbackData))
		if (i+1)%3 == 0 || i == len(months)-1 { // По 3 кнопки в ряду / 3 buttons per row
			rows = append(rows, currentMonthButtons)
			currentMonthButtons = []tgbotapi.InlineKeyboardButton{}
		}
	}

	// Кнопки навигации по годам / Year navigation buttons
	yearNavRow := []tgbotapi.InlineKeyboardButton{}
	currentSystemYear := time.Now().Year()
	if year > currentSystemYear-5 { // Позволяем выбрать до 5 лет назад / Allow selection up to 5 years back
		yearNavRow = append(yearNavRow, tgbotapi.NewInlineKeyboardButtonData(fmt.Sprintf("⬅️ %d год", year-1), fmt.Sprintf("stats_year_nav_%s_%d", context, year-1)))
	}
	if year < currentSystemYear { // Позволяем выбрать до текущего года (если начали с прошлого) / Allow selection up to current year (if started from past)
		yearNavRow = append(yearNavRow, tgbotapi.NewInlineKeyboardButtonData(fmt.Sprintf("%d год ➡️", year+1), fmt.Sprintf("stats_year_nav_%s_%d", context, year+1)))
	}
	if len(yearNavRow) > 0 {
		rows = append(rows, yearNavRow)
	}

	var backToMenuCallback string
	if context == "custom_date" {
		backToMenuCallback = "stats_select_custom_date" // Возврат к выбору "выбрать дату/период" / Return to "select date/period"
	} else if context == "period_start" {
		backToMenuCallback = "stats_select_custom_period"
	} else { // period_end
		// При выборе конечного месяца, "назад" должно вести к выбору начального месяца того же года
		// или к общему меню статистики, если что-то пошло не так.
		// When selecting end month, "back" should lead to start month selection of the same year
		// or to general statistics menu if something went wrong.
		// Для простоты, пока ведем в общее меню статистики. / For simplicity, lead to general statistics menu for now.
		backToMenuCallback = "stats_menu"
	}

	rows = append(rows, tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("🔙 Назад", backToMenuCallback)))
	keyboard := tgbotapi.NewInlineKeyboardMarkup(rows...)

	_, err := bh.sendOrEditMessageHelper(chatID, messageIDToEdit, msgText, &keyboard, tgbotapi.ModeMarkdown)
	if err != nil {
		log.Printf("SendMonthSelectionMenu: Ошибка для chatID %d: %v", chatID, err)
	}
}

// SendDaySelectionMenu отправляет меню выбора дня для статистики.
// context - "custom_date", "period_start", "period_end"
// SendDaySelectionMenu sends the day selection menu for statistics.
// context - "custom_date", "period_start", "period_end"
func (bh *BotHandler) SendDaySelectionMenu(chatID int64, messageIDToEdit int, year int, month time.Month, context string) {
	log.Printf("BotHandler.SendDaySelectionMenu для chatID %d, %d-%s, контекст: %s, messageIDToEdit: %d", chatID, year, month, context, messageIDToEdit)

	var stateToSet string
	switch context {
	case "custom_date":
		stateToSet = constants.STATE_STATS_DAY
	case "period_start":
		stateToSet = constants.STATE_STATS_PERIOD // Остаемся в этом состоянии, пока не выберем день / Remain in this state until day is selected
	case "period_end":
		stateToSet = constants.STATE_STATS_PERIOD_END
	default:
		log.Printf("SendDaySelectionMenu: Неизвестный контекст '%s'", context)
		bh.sendErrorMessageHelper(chatID, messageIDToEdit, "Ошибка выбора контекста дня.")
		return
	}
	bh.Deps.SessionManager.SetState(chatID, stateToSet)

	msgText := fmt.Sprintf("📅 Выберите день для %s %d года:", constants.MonthMap[month], year)
	if context == "period_start" {
		msgText = fmt.Sprintf("📅 Выберите *НАЧАЛЬНЫЙ* день для %s %d:", constants.MonthMap[month], year)
	} else if context == "period_end" {
		msgText = fmt.Sprintf("📅 Выберите *КОНЕЧНЫЙ* день для %s %d:", constants.MonthMap[month], year)
	}

	daysInMonth := time.Date(year, month+1, 0, 0, 0, 0, 0, time.UTC).Day() // Количество дней в месяце / Number of days in month
	var rows [][]tgbotapi.InlineKeyboardButton
	currentDayButtons := []tgbotapi.InlineKeyboardButton{}

	for day := 1; day <= daysInMonth; day++ {
		// Коллбэк: stats_select_day_КОНТЕКСТ_ГОД_НОМЕРМЕСЯЦА_ДЕНЬ
		// Callback: stats_select_day_CONTEXT_YEAR_MONTHNUMBER_DAY
		callbackData := fmt.Sprintf("stats_select_day_%s_%d_%d_%d", context, year, int(month), day)
		currentDayButtons = append(currentDayButtons, tgbotapi.NewInlineKeyboardButtonData(strconv.Itoa(day), callbackData))
		if day%7 == 0 || day == daysInMonth { // По 7 дней в ряду / 7 days per row
			rows = append(rows, currentDayButtons)
			currentDayButtons = []tgbotapi.InlineKeyboardButton{}
		}
	}
	if len(currentDayButtons) > 0 { // Добавляем оставшиеся кнопки, если есть / Add remaining buttons if any
		rows = append(rows, currentDayButtons)
	}

	// Кнопка "Назад" к выбору месяца того же года
	// "Back" button to month selection of the same year
	rows = append(rows, tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("🔙 Назад к выбору месяца", fmt.Sprintf("stats_year_nav_%s_%d", context, year))))
	keyboard := tgbotapi.NewInlineKeyboardMarkup(rows...)

	_, err := bh.sendOrEditMessageHelper(chatID, messageIDToEdit, msgText, &keyboard, tgbotapi.ModeMarkdown)
	if err != nil {
		log.Printf("SendDaySelectionMenu: Ошибка для chatID %d: %v", chatID, err)
	}
}

// DisplayStats отображает полученную статистику.
// DisplayStats displays the retrieved statistics.
func (bh *BotHandler) DisplayStats(chatID int64, messageIDToEdit int, stats models.Stats, periodDescription string) {
	log.Printf("BotHandler.DisplayStats для chatID %d, период: %s, messageIDToEdit: %d", chatID, periodDescription, messageIDToEdit)
	bh.Deps.SessionManager.SetState(chatID, constants.STATE_STATS_MENU) // Возвращаем в меню статистики после просмотра / Return to statistics menu after viewing

	msgText := fmt.Sprintf("📊 Статистика за *%s*:\n\n", utils.EscapeTelegramMarkdown(periodDescription))
	msgText += "📦 *Заказы:*\n"
	if stats.TotalOrders > 0 {
		msgText += fmt.Sprintf("  Всего: %d\n", stats.TotalOrders)
		if stats.NewOrders > 0 {
			msgText += fmt.Sprintf("  %s Новых: %d\n", constants.StatusEmojiMap[constants.STATUS_NEW], stats.NewOrders)
		}
		if stats.InProgressOrders > 0 {
			msgText += fmt.Sprintf("  %s В работе: %d\n", constants.StatusEmojiMap[constants.STATUS_INPROGRESS], stats.InProgressOrders)
		}
		if stats.CompletedOrders > 0 {
			msgText += fmt.Sprintf("  %s Выполненных: %d\n", constants.StatusEmojiMap[constants.STATUS_COMPLETED], stats.CompletedOrders)
		}
		if stats.CanceledOrders > 0 {
			msgText += fmt.Sprintf("  %s Отменённых: %d\n", constants.StatusEmojiMap[constants.STATUS_CANCELED], stats.CanceledOrders)
		}
	} else {
		msgText += "  Заказов нет\n"
	}

	msgText += "\n🗂️ *По категориям (из всех заказов периода):*\n"
	if stats.WasteOrders > 0 {
		msgText += fmt.Sprintf("  %s Мусор: %d\n", constants.CategoryEmojiMap[constants.CAT_WASTE], stats.WasteOrders)
	}
	if stats.DemolitionOrders > 0 {
		msgText += fmt.Sprintf("  %s Демонтаж: %d\n", constants.CategoryEmojiMap[constants.CAT_DEMOLITION], stats.DemolitionOrders)
	}
	if stats.MaterialOrders > 0 {
		msgText += fmt.Sprintf("  %s Стройматериалы: %d\n", constants.CategoryEmojiMap[constants.CAT_MATERIALS], stats.MaterialOrders)
	}
	if stats.WasteOrders == 0 && stats.DemolitionOrders == 0 && stats.MaterialOrders == 0 {
		msgText += "  Нет заказов по этим категориям\n"
	}

	// Отображение финансовых показателей / Display of financial indicators
	// --- MODIFICATION FOR POINT 10 ---
	msgText += fmt.Sprintf("\n💰 Выручка (по завершенным заказам): *%.0f ₽*\n", stats.Revenue)            // Changed from %.0f
	msgText += fmt.Sprintf("📉 Затраты (по завершенным заказам, включая ЗП): *%.0f ₽*\n", stats.Expenses) // Changed from %.0f
	msgText += fmt.Sprintf("📈 Чистая прибыль (Выручка - Затраты): *%.0f ₽*\n", stats.Profit)             // Changed from %.0f
	// --- END MODIFICATION FOR POINT 10 ---
	// msgText += fmt.Sprintf("💸 Долги (не реализовано): %.0f ₽\n", stats.Debts) // Поле Debts пока не используется / Debts field not used yet
	msgText += fmt.Sprintf("\n👥 Новых клиентов за период: *%d*", stats.NewClients)

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔙 Назад в меню статистики", "stats_menu"),
			tgbotapi.NewInlineKeyboardButtonData("🏢 Главное меню", "back_to_main"),
		),
	)

	_, err := bh.sendOrEditMessageHelper(chatID, messageIDToEdit, msgText, &keyboard, tgbotapi.ModeMarkdown)
	if err != nil {
		log.Printf("DisplayStats: Ошибка для chatID %d: %v", chatID, err)
	}
}
