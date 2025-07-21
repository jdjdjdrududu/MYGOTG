package handlers

import (
	"Original/internal/formatters"
	"database/sql"
	"fmt"
	"log"
	"strings"
	"time"

	tgbotapi "github.com/OvyFlash/telegram-bot-api"

	"Original/internal/constants" //
	"Original/internal/db"
	"Original/internal/models"
	"Original/internal/session" //
	"Original/internal/utils"   //
)

// --- Меню процесса оформления и редактирования заказа ---

// SendCategoryMenu отправляет меню выбора категории заказа.
func (bh *BotHandler) SendCategoryMenu(chatID int64, userFirstName string, messageIDToEdit int) {
	log.Printf("BotHandler.SendCategoryMenu для chatID %d, messageIDToEdit: %d", chatID, messageIDToEdit)

	tempOrder := bh.Deps.SessionManager.GetTempOrder(chatID)
	user, _ := bh.getUserFromDB(chatID) // Получаем пользователя, чтобы проверить роль

	// Если это начало нового заказа оператором, устанавливаем специальный флаг
	isOperatorInitiating := utils.IsOperatorOrHigher(user.Role) && tempOrder.ID == 0 && tempOrder.OrderAction != "operator_creating_order"
	if isOperatorInitiating && bh.Deps.SessionManager.GetState(chatID) == constants.STATE_OP_CREATE_ORDER_FLOW {
		// Это условие должно было быть установлено ранее, например, при нажатии "Создать заказ" оператором
		// Для нового потока оператора, мы должны убедиться, что OrderAction установлен
		// Если мы сюда попали из mainMenu по кнопке "Создать заказ" (оператор), то OrderAction должен быть уже установлен
		// Если нет, то это может быть обычный пользователь или оператор начинает заказ для себя как User
	}

	if tempOrder.ID == 0 && tempOrder.OrderAction != "operator_creating_order" { // ID из БД еще не присвоен И это не операторский заказ в процессе
		userChatIDForOrder := tempOrder.UserChatID
		if userChatIDForOrder == 0 {
			userChatIDForOrder = chatID
		}
		tempOrder = session.NewTempOrder(userChatIDForOrder)
	} else if tempOrder.ID == 0 && tempOrder.OrderAction == "operator_creating_order" {
		// Оператор создает заказ, UserChatID будет установлен позже или будет ID самого оператора, если для себя
		// Пока оставляем как есть или устанавливаем в chatID оператора
		if tempOrder.UserChatID == 0 {
			tempOrder.UserChatID = chatID
		}
	}
	bh.Deps.SessionManager.UpdateTempOrder(chatID, tempOrder)
	bh.Deps.SessionManager.SetState(chatID, constants.STATE_ORDER_CATEGORY)

	msgText := fmt.Sprintf("👇 Здравствуйте, %s! Какую услугу выберете сегодня?\n\n"+
		"Мы поможем быстро и качественно решить вашу задачу! ✨\n", utils.EscapeTelegramMarkdown(userFirstName))

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(fmt.Sprintf("%s %s", constants.CategoryEmojiMap[constants.CAT_WASTE], constants.CategoryDisplayMap[constants.CAT_WASTE]), "category_waste"),
			tgbotapi.NewInlineKeyboardButtonData(fmt.Sprintf("%s %s", constants.CategoryEmojiMap[constants.CAT_DEMOLITION], constants.CategoryDisplayMap[constants.CAT_DEMOLITION]), "category_demolition"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(fmt.Sprintf("%s %s (скоро + бонус!)", constants.CategoryEmojiMap[constants.CAT_MATERIALS], constants.CategoryDisplayMap[constants.CAT_MATERIALS]), "materials_soon"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🏢 Главное меню", "back_to_main_confirm_cancel_order"),
		),
	)

	_, err := bh.sendOrEditMessageHelper(chatID, messageIDToEdit, msgText, &keyboard, tgbotapi.ModeMarkdown)
	if err != nil {
		log.Printf("SendCategoryMenu: Ошибка для chatID %d: %v", chatID, err)
	}
}

// SendSubcategoryMenu отправляет меню выбора подкатегории.
func (bh *BotHandler) SendSubcategoryMenu(chatID int64, category string, messageIDToEdit int) {
	log.Printf("BotHandler.SendSubcategoryMenu для chatID %d, категория: %s, messageIDToEdit: %d", chatID, category, messageIDToEdit)
	bh.Deps.SessionManager.SetState(chatID, constants.STATE_ORDER_SUBCATEGORY)

	tempOrder := bh.Deps.SessionManager.GetTempOrder(chatID)
	tempOrder.Category = category
	bh.Deps.SessionManager.UpdateTempOrder(chatID, tempOrder)

	var msgText string
	var keyboard tgbotapi.InlineKeyboardMarkup
	currentCategoryDisplay := constants.CategoryDisplayMap[category]
	if currentCategoryDisplay == "" {
		currentCategoryDisplay = category // Fallback
	}

	backButtonRow := tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("⬅️ Назад к Категориям", "back_to_category"),
		tgbotapi.NewInlineKeyboardButtonData("🏢 Главное меню", "back_to_main_confirm_cancel_order"),
	)

	switch category {
	case constants.CAT_WASTE:
		msgText = fmt.Sprintf("Выбрано: *%s*. ♻️ Уточните тип мусора:\n\n"+
			"💡 Точное указание поможет нам быстрее рассчитать стоимость и подобрать подходящий транспорт!", utils.EscapeTelegramMarkdown(currentCategoryDisplay))
		keyboard = tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData(constants.WasteSubcategoryMap["construct"], "subcategory_construct"),
				tgbotapi.NewInlineKeyboardButtonData(constants.WasteSubcategoryMap["household"], "subcategory_household"),
			),
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData(constants.WasteSubcategoryMap["metal"], "subcategory_metal"),
				tgbotapi.NewInlineKeyboardButtonData(constants.WasteSubcategoryMap["junk"], "subcategory_junk"),
			),
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData(constants.WasteSubcategoryMap["greenery"], "subcategory_greenery"),
				tgbotapi.NewInlineKeyboardButtonData(constants.WasteSubcategoryMap["tires"], "subcategory_tires"),
			),
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData(constants.WasteSubcategoryMap["other_waste"], "subcategory_other_waste"),
			),
			backButtonRow,
		)
	case constants.CAT_DEMOLITION:
		msgText = fmt.Sprintf("Выбрано: *%s*. 🛠️ Какой вид демонтажа требуется?\n\n"+
			"💡 Подробности помогут нам подобрать лучших специалистов для вашей задачи!", utils.EscapeTelegramMarkdown(currentCategoryDisplay))
		keyboard = tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData(constants.DemolitionSubcategoryMap["walls"], "subcategory_walls"),
				tgbotapi.NewInlineKeyboardButtonData(constants.DemolitionSubcategoryMap["partitions"], "subcategory_partitions"),
			),
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData(constants.DemolitionSubcategoryMap["floors"], "subcategory_floors"),
				tgbotapi.NewInlineKeyboardButtonData(constants.DemolitionSubcategoryMap["ceilings"], "subcategory_ceilings"),
			),
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData(constants.DemolitionSubcategoryMap["plumbing"], "subcategory_plumbing"),
				tgbotapi.NewInlineKeyboardButtonData(constants.DemolitionSubcategoryMap["tiles"], "subcategory_tiles"),
			),
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData(constants.DemolitionSubcategoryMap["other_demo"], "subcategory_other_demo"),
			),
			backButtonRow,
		)
	default:
		log.Printf("Категория '%s' для chatID %d не требует выбора подкатегории или обрабатывается иначе, переход к вводу описания.", category, chatID)
		tempOrder.Subcategory = "default_for_" + category
		bh.Deps.SessionManager.UpdateTempOrder(chatID, tempOrder)
		bh.SendDescriptionInputMenu(chatID, messageIDToEdit)
		return
	}

	_, err := bh.sendOrEditMessageHelper(chatID, messageIDToEdit, msgText, &keyboard, tgbotapi.ModeMarkdown)
	if err != nil {
		log.Printf("SendSubcategoryMenu: Ошибка для chatID %d: %v", chatID, err)
	}
}

// SendDescriptionInputMenu отправляет меню для ввода описания заказа.
func (bh *BotHandler) SendDescriptionInputMenu(chatID int64, messageIDToEdit int) {
	log.Printf("BotHandler.SendDescriptionInputMenu для chatID %d, messageIDToEdit: %d", chatID, messageIDToEdit)
	bh.Deps.SessionManager.SetState(chatID, constants.STATE_ORDER_DESCRIPTION)

	tempOrder := bh.Deps.SessionManager.GetTempOrder(chatID)
	tempOrder.CurrentMessageID = messageIDToEdit // Это важно для sendOrEditMessageHelper
	bh.Deps.SessionManager.UpdateTempOrder(chatID, tempOrder)

	history := bh.Deps.SessionManager.GetHistory(chatID)
	isEditingOrder := false
	if len(history) >= 2 && history[len(history)-2] == constants.STATE_ORDER_EDIT && tempOrder.ID != 0 {
		isEditingOrder = true
	}

	backButtonCallbackData := "back_to_subcategory"
	if isEditingOrder {
		backButtonCallbackData = "back_to_edit_menu_direct"
	} else if tempOrder.Category == constants.CAT_MATERIALS || tempOrder.Category == constants.CAT_OTHER {
		// Если категория не требует подкатегории, "Назад" ведет к категориям
		backButtonCallbackData = "back_to_category"
	}

	msgText := "📝 Опишите детали заказа (например, объем, этаж, наличие лифта, особые пожелания).\nЭто поможет нам точнее рассчитать стоимость и время.\n\nВы можете пропустить этот шаг."

	var rows [][]tgbotapi.InlineKeyboardButton
	if tempOrder.Description != "" { // Если описание уже есть (например, при редактировании или возврате)
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("✅ Оставить текущее описание", "confirm_order_description_placeholder"),
		))
	}
	rows = append(rows, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("➡️ Пропустить описание", "skip_order_description_placeholder"),
	))
	rows = append(rows, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("⬅️ Назад", backButtonCallbackData),
		tgbotapi.NewInlineKeyboardButtonData("🏢 Главное меню", "back_to_main_confirm_cancel_order"),
	))
	keyboard := tgbotapi.NewInlineKeyboardMarkup(rows...)

	_, err := bh.sendOrEditMessageHelper(chatID, messageIDToEdit, msgText, &keyboard, tgbotapi.ModeMarkdown)
	if err != nil {
		log.Printf("SendDescriptionInputMenu: Ошибка для chatID %d: %v", chatID, err)
	}
}

// SendNameInputMenu отправляет меню для ввода имени клиента.
func (bh *BotHandler) SendNameInputMenu(chatID int64, messageIDToEdit int) {
	log.Printf("BotHandler.SendNameInputMenu для chatID %d, messageIDToEdit: %d", chatID, messageIDToEdit)

	currentUser, ok := bh.getUserFromDB(chatID)
	if !ok {
		bh.sendErrorMessageHelper(chatID, messageIDToEdit, "Произошла ошибка с вашими данными. Попробуйте /start")
		return
	}

	bh.Deps.SessionManager.SetState(chatID, constants.STATE_ORDER_NAME)
	tempOrder := bh.Deps.SessionManager.GetTempOrder(chatID)
	tempOrder.CurrentMessageID = messageIDToEdit
	bh.Deps.SessionManager.UpdateTempOrder(chatID, tempOrder)

	var promptText string
	var keyboard tgbotapi.InlineKeyboardMarkup

	history := bh.Deps.SessionManager.GetHistory(chatID)
	isEditingOrder := len(history) >= 2 && history[len(history)-2] == constants.STATE_ORDER_EDIT && tempOrder.ID != 0

	backButtonCallbackData := "back_to_description"
	if isEditingOrder {
		backButtonCallbackData = "back_to_edit_menu_direct"
	}

	mainMenuButtonCallbackData := "back_to_main_confirm_cancel_order"

	isOperatorCreating := tempOrder.OrderAction == "operator_creating_order"
	isDriverCreating := tempOrder.OrderAction == "driver_creating_order" // Новое условие

	// Если заказ создает оператор или водитель
	if isOperatorCreating || isDriverCreating {
		if tempOrder.Name != "" { // Имя уже введено в сессии (например, при возврате на шаг назад)
			promptText = fmt.Sprintf("👤 Имя клиента для заказа: *%s*. \nЖелаете изменить? Введите новое или подтвердите.", utils.EscapeTelegramMarkdown(tempOrder.Name))
			keyboard = tgbotapi.NewInlineKeyboardMarkup(
				tgbotapi.NewInlineKeyboardRow(
					tgbotapi.NewInlineKeyboardButtonData("✅ Оставить "+utils.EscapeTelegramMarkdown(tempOrder.Name), "confirm_order_name"),
				),
				tgbotapi.NewInlineKeyboardRow(
					tgbotapi.NewInlineKeyboardButtonData("⬅️ Назад", backButtonCallbackData),
					tgbotapi.NewInlineKeyboardButtonData("🏢 Главное меню", mainMenuButtonCallbackData),
				),
			)
		} else { // Первый раз на этом шаге, всегда просим ввести имя текстом.
			promptText = "👤 Пожалуйста, введите имя клиента для этого заказа:"
			keyboard = tgbotapi.NewInlineKeyboardMarkup(
				tgbotapi.NewInlineKeyboardRow(
					tgbotapi.NewInlineKeyboardButtonData("⬅️ Назад", backButtonCallbackData),
					tgbotapi.NewInlineKeyboardButtonData("🏢 Главное меню", mainMenuButtonCallbackData),
				),
			)
		}
	} else { // Логика для обычного пользователя (клиента)
		userForOrderNameSuggestion := currentUser
		if tempOrder.Name != "" { // Клиент уже ввел имя и вернулся на этот шаг.
			promptText = fmt.Sprintf("👤 Имя для заказа: *%s*. \nЖелаете изменить? Введите новое или подтвердите.", utils.EscapeTelegramMarkdown(tempOrder.Name))
			keyboard = tgbotapi.NewInlineKeyboardMarkup(
				tgbotapi.NewInlineKeyboardRow(
					tgbotapi.NewInlineKeyboardButtonData("✅ Оставить "+utils.EscapeTelegramMarkdown(tempOrder.Name), "confirm_order_name"),
				),
				tgbotapi.NewInlineKeyboardRow(
					tgbotapi.NewInlineKeyboardButtonData("⬅️ Назад", backButtonCallbackData),
					tgbotapi.NewInlineKeyboardButtonData("🏢 Главное меню", mainMenuButtonCallbackData),
				),
			)
		} else if userForOrderNameSuggestion.FirstName != "" && !isEditingOrder { // Предлагаем имя из профиля клиента
			promptText = fmt.Sprintf("👤 Будем оформлять заказ на имя *%s*? \nЕсли да, нажмите кнопку. Если нет, введите другое имя.", utils.EscapeTelegramMarkdown(userForOrderNameSuggestion.FirstName))
			keyboard = tgbotapi.NewInlineKeyboardMarkup(
				tgbotapi.NewInlineKeyboardRow(
					tgbotapi.NewInlineKeyboardButtonData(fmt.Sprintf("✅ Да, на %s", utils.EscapeTelegramMarkdown(userForOrderNameSuggestion.FirstName)), "use_profile_name_for_order"),
				),
				tgbotapi.NewInlineKeyboardRow(
					tgbotapi.NewInlineKeyboardButtonData("⬅️ Назад", backButtonCallbackData),
					tgbotapi.NewInlineKeyboardButtonData("🏢 Главное меню", mainMenuButtonCallbackData),
				),
			)
		} else { // Просим клиента ввести имя, если в профиле оно пустое
			promptText = "👤 Пожалуйста, введите ваше имя для этого заказа (контактное лицо):"
			keyboard = tgbotapi.NewInlineKeyboardMarkup(
				tgbotapi.NewInlineKeyboardRow(
					tgbotapi.NewInlineKeyboardButtonData("⬅️ Назад", backButtonCallbackData),
					tgbotapi.NewInlineKeyboardButtonData("🏢 Главное меню", mainMenuButtonCallbackData),
				),
			)
		}
	}

	_, err := bh.sendOrEditMessageHelper(chatID, messageIDToEdit, promptText, &keyboard, tgbotapi.ModeMarkdown)
	if err != nil {
		log.Printf("SendNameInputMenu: Ошибка для chatID %d: %v", chatID, err)
	}
}

// SendDateSelectionMenu отправляет меню выбора даты.
func (bh *BotHandler) SendDateSelectionMenu(chatID int64, messageIDToEdit int, page int) {
	log.Printf("BotHandler.SendDateSelectionMenu для chatID %d, messageIDToEdit: %d, page: %d", chatID, messageIDToEdit, page)
	bh.Deps.SessionManager.SetState(chatID, constants.STATE_ORDER_DATE)

	now := time.Now()
	var startDate time.Time
	daysToShow := 7
	if page == 0 {
		startDate = now
	} else {
		startDate = now.AddDate(0, 0, page*daysToShow)
	}

	var rows [][]tgbotapi.InlineKeyboardButton
	tempOrder := bh.Deps.SessionManager.GetTempOrder(chatID)
	history := bh.Deps.SessionManager.GetHistory(chatID)
	isEditingOrder := false
	if len(history) >= 2 && history[len(history)-2] == constants.STATE_ORDER_EDIT && tempOrder.ID != 0 {
		isEditingOrder = true
	}

	if page == 0 && !isEditingOrder { // Кнопка "Срочно" только при создании нового заказа и на первой странице
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("❗ Срочно (в ближайшее время) ❗", "select_date_asap"),
		))
	}

	weekdayMap := map[time.Weekday]string{time.Monday: "Пн", time.Tuesday: "Вт", time.Wednesday: "Ср", time.Thursday: "Чт", time.Friday: "Пт", time.Saturday: "Сб", time.Sunday: "Вс"}
	monthMapShort := map[time.Month]string{time.January: "Янв", time.February: "Фев", time.March: "Мар", time.April: "Апр", time.May: "Мая", time.June: "Июн", time.July: "Июл", time.August: "Авг", time.September: "Сен", time.October: "Окт", time.November: "Ноя", time.December: "Дек"}

	var dateButtons []tgbotapi.InlineKeyboardButton
	daysAdded := 0
	for i := 0; daysAdded < daysToShow; i++ {
		date := startDate.AddDate(0, 0, i)
		if date.Before(now.Truncate(24*time.Hour)) && !isEditingOrder { // Пропускаем прошедшие даты только при создании
			continue
		}
		dayStr := fmt.Sprintf("%s, %d %s", weekdayMap[date.Weekday()], date.Day(), monthMapShort[date.Month()])
		emoji := "🟢"
		if date.Weekday() == time.Saturday || date.Weekday() == time.Sunday {
			emoji = "⭕️"
		}
		callbackData := fmt.Sprintf("select_date_%02d_%s_%d", date.Day(), date.Month().String(), date.Year())
		dateButtons = append(dateButtons, tgbotapi.NewInlineKeyboardButtonData(emoji+" "+dayStr, callbackData))
		daysAdded++

		if len(dateButtons) == 2 || (daysAdded == daysToShow && len(dateButtons) > 0) {
			rows = append(rows, dateButtons)
			dateButtons = []tgbotapi.InlineKeyboardButton{}
		}
	}
	if len(dateButtons) > 0 {
		rows = append(rows, dateButtons)
	}

	// Ряд для навигации по неделям
	navRow := []tgbotapi.InlineKeyboardButton{}
	if page > 0 {
		navRow = append(navRow, tgbotapi.NewInlineKeyboardButtonData("⬅️ Предыдущая неделя", fmt.Sprintf("date_page_%d", page-1)))
	}
	if page < 51 { // Ограничение на ~1 год вперед
		navRow = append(navRow, tgbotapi.NewInlineKeyboardButtonData("Следующая неделя ➡️", fmt.Sprintf("date_page_%d", page+1)))
	}
	if len(navRow) > 0 {
		rows = append(rows, navRow)
	}

	// Определяем коллбэк для кнопки "Назад"
	backCallback := "back_to_name"
	if isEditingOrder {
		backCallback = "back_to_edit_menu_direct"
	}

	// Последний ряд с кнопками "Назад" и "Главное меню"
	rows = append(rows, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("⬅️ Назад", backCallback),
		tgbotapi.NewInlineKeyboardButtonData("🏢 Главное меню", "back_to_main_confirm_cancel_order"),
	))

	keyboard := tgbotapi.NewInlineKeyboardMarkup(rows...)

	msgText := "📅 Выберите удобную дату для заказа:\n\n" +
		"🚛 Мы готовы приступить к работе в кратчайшие сроки! 😎"

	_, err := bh.sendOrEditMessageHelper(chatID, messageIDToEdit, msgText, &keyboard, tgbotapi.ModeMarkdown)
	if err != nil {
		log.Printf("SendDateSelectionMenu: Ошибка для chatID %d: %v", chatID, err)
	}
}

// SendTimeSelectionMenu отправляет меню выбора времени.
func (bh *BotHandler) SendTimeSelectionMenu(chatID int64, messageIDToEdit int) {
	log.Printf("BotHandler.SendTimeSelectionMenu для chatID %d, messageIDToEdit: %d", chatID, messageIDToEdit)
	bh.Deps.SessionManager.SetState(chatID, constants.STATE_ORDER_TIME)

	var rows [][]tgbotapi.InlineKeyboardButton
	timeSlots := []string{}
	// Изменяем цикл, чтобы он включал 17:00, но не 18:00
	for hour := 9; hour <= 17; hour++ {
		timeSlots = append(timeSlots, fmt.Sprintf("%02d:00", hour))
	}

	var timeButtons []tgbotapi.InlineKeyboardButton
	for i, hourSlot := range timeSlots {
		hourStr := strings.Split(hourSlot, ":")[0]
		callbackData := fmt.Sprintf("%s_%s", constants.CALLBACK_PREFIX_SELECT_HOUR, hourStr)
		timeButtons = append(timeButtons, tgbotapi.NewInlineKeyboardButtonData(hourSlot, callbackData))
		if (i+1)%3 == 0 || i == len(timeSlots)-1 {
			rows = append(rows, timeButtons)
			timeButtons = []tgbotapi.InlineKeyboardButton{}
		}
	}
	if len(timeButtons) > 0 {
		rows = append(rows, timeButtons)
	}

	tempOrder := bh.Deps.SessionManager.GetTempOrder(chatID)
	history := bh.Deps.SessionManager.GetHistory(chatID)
	isEditingOrder := false
	if len(history) >= 2 && history[len(history)-2] == constants.STATE_ORDER_EDIT {
		isEditingOrder = true
	}
	backButtonCallbackData := "back_to_date"
	if isEditingOrder {
		backButtonCallbackData = "back_to_edit_menu_direct"
	}

	rows = append(rows, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("⬅️ Назад", backButtonCallbackData),
		tgbotapi.NewInlineKeyboardButtonData("🏢 Главное меню", "back_to_main_confirm_cancel_order"),
	))
	keyboard := tgbotapi.NewInlineKeyboardMarkup(rows...)

	msgText := "⏰ Выберите удобный *час* для заказа или введите точное время (например, 09:30):\n\n" +
		"🚛 Мы приедем точно в срок 😎"

	tempOrder.SelectedHourForMinuteView = -1
	bh.Deps.SessionManager.UpdateTempOrder(chatID, tempOrder)

	_, err := bh.sendOrEditMessageHelper(chatID, messageIDToEdit, msgText, &keyboard, tgbotapi.ModeMarkdown)
	if err != nil {
		log.Printf("SendTimeSelectionMenu: Ошибка для chatID %d: %v", chatID, err)
	}
}

// SendMinuteSelectionMenu отправляет меню выбора минут для указанного часа.
func (bh *BotHandler) SendMinuteSelectionMenu(chatID int64, selectedHour int, messageIDToEdit int) {
	log.Printf("BotHandler.SendMinuteSelectionMenu для chatID %d, час: %d, messageIDToEdit: %d", chatID, selectedHour, messageIDToEdit)
	bh.Deps.SessionManager.SetState(chatID, constants.STATE_ORDER_MINUTE_SELECTION)

	tempOrder := bh.Deps.SessionManager.GetTempOrder(chatID)
	tempOrder.SelectedHourForMinuteView = selectedHour
	bh.Deps.SessionManager.UpdateTempOrder(chatID, tempOrder)

	var rows [][]tgbotapi.InlineKeyboardButton
	minuteSlots := []int{0, 15, 30, 45}

	var minuteButtons []tgbotapi.InlineKeyboardButton
	for _, minute := range minuteSlots {
		timeStr := fmt.Sprintf("%02d:%02d", selectedHour, minute)
		callbackData := fmt.Sprintf("select_time_%s", timeStr)
		minuteButtons = append(minuteButtons, tgbotapi.NewInlineKeyboardButtonData(timeStr, callbackData))
	}
	rows = append(rows, minuteButtons)

	backButtonCallbackData := "back_to_time" // Возврат к выбору часа

	rows = append(rows, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("⬅️ Назад к выбору часа", backButtonCallbackData),
		tgbotapi.NewInlineKeyboardButtonData("🏢 Главное меню", "back_to_main_confirm_cancel_order"),
	))
	keyboard := tgbotapi.NewInlineKeyboardMarkup(rows...)

	msgText := fmt.Sprintf("⏰ Вы выбрали час: *%02d:xx*. Уточните минуты или введите точное время (например, %02d:10):\n\n"+
		"🚛 Мы приедем точно в срок 😎", selectedHour, selectedHour)

	_, err := bh.sendOrEditMessageHelper(chatID, messageIDToEdit, msgText, &keyboard, tgbotapi.ModeMarkdown)
	if err != nil {
		log.Printf("SendMinuteSelectionMenu: Ошибка для chatID %d: %v", chatID, err)
	}
}

// SendPhoneInputMenu отправляет меню для ввода/подтверждения номера телефона.
func (bh *BotHandler) SendPhoneInputMenu(chatID int64, user models.User, messageIDToEdit int) {
	log.Printf("BotHandler.SendPhoneInputMenu для chatID %d, messageIDToEdit: %d", chatID, messageIDToEdit)
	bh.Deps.SessionManager.SetState(chatID, constants.STATE_ORDER_PHONE)

	tempOrder := bh.Deps.SessionManager.GetTempOrder(chatID)
	isOperatorCreating := tempOrder.OrderAction == "operator_creating_order"
	isDriverCreating := tempOrder.OrderAction == "driver_creating_order" // Новое условие

	// Предзаполняем номер из профиля только для клиента, не для оператора или водителя, создающего заказ
	if !isOperatorCreating && !isDriverCreating && tempOrder.UserChatID == chatID && user.Phone.Valid && tempOrder.Phone == "" {
		tempOrder.Phone = user.Phone.String
		bh.Deps.SessionManager.UpdateTempOrder(chatID, tempOrder)
	}

	var msgText string
	var inlineKeyboard tgbotapi.InlineKeyboardMarkup
	var replyKeyboard tgbotapi.ReplyKeyboardMarkup // По умолчанию пустая

	history := bh.Deps.SessionManager.GetHistory(chatID)
	isEditingOrder := len(history) >= 2 && history[len(history)-2] == constants.STATE_ORDER_EDIT && tempOrder.ID != 0
	backButtonCallbackData := "back_to_time"
	if isEditingOrder {
		backButtonCallbackData = "back_to_edit_menu_direct"
	}
	mainMenuButtonCallbackData := "back_to_main_confirm_cancel_order"

	// Если заказ создает оператор или водитель
	if isOperatorCreating || isDriverCreating {
		currentOrderPhone := tempOrder.Phone
		if currentOrderPhone != "" { // Телефон уже введен в сессии
			formattedPhoneForDisplay := utils.FormatPhoneNumber(currentOrderPhone)
			msgText = fmt.Sprintf(
				"📱 Контактный номер клиента для заказа: *%s*.\n\n"+
					"Это верный номер? Если нет, отправьте новый текстом.",
				utils.EscapeTelegramMarkdown(formattedPhoneForDisplay))
			inlineKeyboard = tgbotapi.NewInlineKeyboardMarkup(
				tgbotapi.NewInlineKeyboardRow(
					tgbotapi.NewInlineKeyboardButtonData(fmt.Sprintf("✅ Да, %s", formattedPhoneForDisplay), "confirm_order_phone"),
				),
				tgbotapi.NewInlineKeyboardRow(
					tgbotapi.NewInlineKeyboardButtonData("✏️ Изменить номер", "change_order_phone"),
				),
				tgbotapi.NewInlineKeyboardRow(
					tgbotapi.NewInlineKeyboardButtonData("⬅️ Назад", backButtonCallbackData),
					tgbotapi.NewInlineKeyboardButtonData("🏢 Главное меню", mainMenuButtonCallbackData),
				),
			)
		} else { // Первый раз на этом шаге, всегда запрашиваем ввод текстом
			msgText = "📱 Пожалуйста, укажите контактный номер телефона клиента.\n\n" +
				"Вы можете отправить его текстом (например, +79001234567)."
			inlineKeyboard = tgbotapi.NewInlineKeyboardMarkup(
				tgbotapi.NewInlineKeyboardRow(
					tgbotapi.NewInlineKeyboardButtonData("⬅️ Назад", backButtonCallbackData),
					tgbotapi.NewInlineKeyboardButtonData("🏢 Главное меню", mainMenuButtonCallbackData),
				),
			)
			// ReplyKeyboard для оператора/водителя не создается
		}
	} else {
		// --- Поток для Клиента (оригинальная логика без изменений) ---
		phoneForSuggestion := ""
		if user.Phone.Valid {
			phoneForSuggestion = user.Phone.String
		}
		currentOrderPhone := tempOrder.Phone

		if currentOrderPhone != "" {
			formattedPhoneForDisplay := utils.FormatPhoneNumber(currentOrderPhone)
			msgText = fmt.Sprintf(
				"📱 Для заказа будет использован номер: *%s*.\n\n"+
					"Это верный номер? Если нет, отправьте новый текстом или нажав кнопку изменить номер✏️",
				utils.EscapeTelegramMarkdown(formattedPhoneForDisplay))
			inlineKeyboard = tgbotapi.NewInlineKeyboardMarkup(
				tgbotapi.NewInlineKeyboardRow(
					tgbotapi.NewInlineKeyboardButtonData(fmt.Sprintf("✅ Да, %s", formattedPhoneForDisplay), "confirm_order_phone"),
				),
				tgbotapi.NewInlineKeyboardRow(
					tgbotapi.NewInlineKeyboardButtonData("✏️ Изменить номер", "change_order_phone"),
				),
				tgbotapi.NewInlineKeyboardRow(
					tgbotapi.NewInlineKeyboardButtonData("⬅️ Назад", backButtonCallbackData),
					tgbotapi.NewInlineKeyboardButtonData("🏢 Главное меню", mainMenuButtonCallbackData),
				),
			)
		} else if phoneForSuggestion != "" && !isEditingOrder {
			formattedPhoneForDisplay := utils.FormatPhoneNumber(phoneForSuggestion)
			msgText = fmt.Sprintf("📱 Использовать ваш номер *%s* для заказа? \nЕсли да, нажмите кнопку. Если нет, введите другой номер или поделитесь контактом.", utils.EscapeTelegramMarkdown(formattedPhoneForDisplay))
			inlineKeyboard = tgbotapi.NewInlineKeyboardMarkup(
				tgbotapi.NewInlineKeyboardRow(
					tgbotapi.NewInlineKeyboardButtonData(fmt.Sprintf("✅ Да, %s", formattedPhoneForDisplay), "confirm_order_phone"),
				),
				tgbotapi.NewInlineKeyboardRow(
					tgbotapi.NewInlineKeyboardButtonData("✏️ Ввести другой номер", "change_order_phone"),
				),
				tgbotapi.NewInlineKeyboardRow(
					tgbotapi.NewInlineKeyboardButtonData("⬅️ Назад", backButtonCallbackData),
					tgbotapi.NewInlineKeyboardButtonData("🏢 Главное меню", mainMenuButtonCallbackData),
				),
			)
			tempOrder.Phone = phoneForSuggestion
			bh.Deps.SessionManager.UpdateTempOrder(chatID, tempOrder)
			replyKeyboard = tgbotapi.NewReplyKeyboard(
				tgbotapi.NewKeyboardButtonRow(
					tgbotapi.NewKeyboardButtonContact(fmt.Sprintf("📞 Поделиться моим номером (%s)", utils.GetUserDisplayName(user))),
				),
			)
			replyKeyboard.OneTimeKeyboard = true
			replyKeyboard.ResizeKeyboard = true
		} else {
			msgText = "📱 Пожалуйста, укажите ваш контактный номер телефона.\n\n" +
				"Вы можете отправить его текстом (например, +79001234567) или нажать кнопку ниже, чтобы поделиться контактом из Telegram.\n\n" +
				"💡 Это поможет нам оперативно связаться с вами для уточнения деталей заказа."
			inlineKeyboard = tgbotapi.NewInlineKeyboardMarkup(
				tgbotapi.NewInlineKeyboardRow(
					tgbotapi.NewInlineKeyboardButtonData("⬅️ Назад", backButtonCallbackData),
					tgbotapi.NewInlineKeyboardButtonData("🏢 Главное меню", mainMenuButtonCallbackData),
				),
			)
			replyKeyboard = tgbotapi.NewReplyKeyboard(
				tgbotapi.NewKeyboardButtonRow(
					tgbotapi.NewKeyboardButtonContact(fmt.Sprintf("📞 Поделиться моим номером (%s)", utils.GetUserDisplayName(user))),
				),
			)
			replyKeyboard.OneTimeKeyboard = true
			replyKeyboard.ResizeKeyboard = true
		}
	}

	sentInlineMsg, err := bh.sendOrEditMessageHelper(chatID, messageIDToEdit, msgText, &inlineKeyboard, tgbotapi.ModeMarkdown)
	if err != nil {
		log.Printf("SendPhoneInputMenu: Ошибка для chatID %d: %v", chatID, err)
		return
	}

	// Этот блок теперь будет выполняться только для клиента, так как у оператора/водителя replyKeyboard.Keyboard будет nil
	if replyKeyboard.Keyboard != nil {
		tempOrderForClean := bh.Deps.SessionManager.GetTempOrder(chatID)
		if tempOrderForClean.LocationPromptMessageID != 0 {
			bh.deleteMessageHelper(chatID, tempOrderForClean.LocationPromptMessageID)
			updatedTempOrder := bh.Deps.SessionManager.GetTempOrder(chatID)
			updatedTempOrder.LocationPromptMessageID = 0
			bh.Deps.SessionManager.UpdateTempOrder(chatID, updatedTempOrder)
		} else {
			replyMarkupRemove := tgbotapi.NewRemoveKeyboard(true)
			msgToRemoveActiveKb := tgbotapi.NewMessage(chatID, constants.InvisibleMessage)
			msgToRemoveActiveKb.ReplyMarkup = replyMarkupRemove
			if sentInvisible, errSendInvisible := bh.Deps.BotClient.Send(msgToRemoveActiveKb); errSendInvisible == nil {
				go func(id int) {
					time.Sleep(1000 * time.Millisecond)
					bh.deleteMessageHelper(chatID, id)
				}(sentInvisible.MessageID)
			} else {
				log.Printf("SendPhoneInputMenu: Ошибка отправки сообщения для удаления активной ReplyKeyboard: %v", errSendInvisible)
			}
			time.Sleep(300 * time.Millisecond)
		}

		tempMsgConfig := tgbotapi.NewMessage(chatID, "Вы также можете использовать кнопку ниже 👇")
		tempMsgConfig.ReplyMarkup = replyKeyboard

		sentReplyKbMsg, errKb := bh.Deps.BotClient.Send(tempMsgConfig)
		if errKb != nil {
			log.Printf("SendPhoneInputMenu: Ошибка отправки ReplyKeyboard для телефона chatID %d: %v", chatID, errKb)
		} else {
			orderDataSess := bh.Deps.SessionManager.GetTempOrder(chatID)
			if orderDataSess.CurrentMessageID != sentInlineMsg.MessageID && sentInlineMsg.MessageID != 0 {
				orderDataSess.CurrentMessageID = sentInlineMsg.MessageID
			}
			orderDataSess.LocationPromptMessageID = sentReplyKbMsg.MessageID
			bh.Deps.SessionManager.UpdateTempOrder(chatID, orderDataSess)
		}
	}
}

// SendAddressInputMenu отправляет меню для ввода адреса.
func (bh *BotHandler) SendAddressInputMenu(chatID int64, messageIDToEdit int) {
	log.Printf("BotHandler.SendAddressInputMenu для chatID %d, messageIDToEdit: %d", chatID, messageIDToEdit)
	bh.Deps.SessionManager.SetState(chatID, constants.STATE_ORDER_ADDRESS)

	tempData := bh.Deps.SessionManager.GetTempOrder(chatID)
	// Очистка ReplyKeyboard от предыдущего шага (если была)
	if tempData.LocationPromptMessageID != 0 {
		bh.deleteMessageHelper(chatID, tempData.LocationPromptMessageID)
		tempData.LocationPromptMessageID = 0
		bh.Deps.SessionManager.UpdateTempOrder(chatID, tempData)
	} else { // Если LocationPromptMessageID был 0, возможно, клавиатура еще активна
		replyMarkupRemove := tgbotapi.NewRemoveKeyboard(true)
		msgToRemoveKb := tgbotapi.NewMessage(chatID, constants.InvisibleMessage)
		msgToRemoveKb.ReplyMarkup = replyMarkupRemove
		if sentKbRemovalMsg, err := bh.Deps.BotClient.Send(msgToRemoveKb); err == nil {
			go func(id int) { time.Sleep(200 * time.Millisecond); bh.deleteMessageHelper(chatID, id) }(sentKbRemovalMsg.MessageID)
		}
	}

	tempData.CurrentMessageID = messageIDToEdit // Обновляем для sendOrEditMessageHelper
	bh.Deps.SessionManager.UpdateTempOrder(chatID, tempData)

	history := bh.Deps.SessionManager.GetHistory(chatID)
	isEditingOrder := false
	if len(history) >= 2 && history[len(history)-2] == constants.STATE_ORDER_EDIT {
		isEditingOrder = true
	}
	backButtonCallback := "back_to_phone"
	if isEditingOrder {
		backButtonCallback = "back_to_edit_menu_direct"
	}

	msgText := "📍 Укажите адрес или поделитесь местоположением\n\n" +
		"💡 Введите адрес вручную, прикрепите геометку (📎) или нажмите кнопку ниже для отправки текущего местоположения."

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📍 Отправить местоположение", "send_location_prompt"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("⬅️ Назад", backButtonCallback),
			tgbotapi.NewInlineKeyboardButtonData("🏢 Главное меню", "back_to_main_confirm_cancel_order"),
		),
	)

	_, err := bh.sendOrEditMessageHelper(chatID, messageIDToEdit, msgText, &keyboard, tgbotapi.ModeMarkdown)
	if err != nil {
		log.Printf("SendAddressInputMenu: Ошибка для chatID %d: %v", chatID, err)
	}
}

// SendLocationPrompt отправляет запрос на геолокацию с ReplyKeyboard.
func (bh *BotHandler) SendLocationPrompt(chatID int64, originalAddressMenuMessageID int) {
	log.Printf("BotHandler.SendLocationPrompt для chatID %d, originalAddressMenuMessageID: %d", chatID, originalAddressMenuMessageID)
	bh.Deps.SessionManager.SetState(chatID, constants.STATE_ORDER_ADDRESS_LOCATION)

	msgText := "📍 Отправьте свое местоположение с помощью кнопки ниже:\n\n" +
		"💡 Убедитесь, что в настройках Telegram разрешён доступ к геолокации!"

	replyKeyboard := tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButtonLocation("📍 Отправить мое местоположение"),
		),
	)
	replyKeyboard.OneTimeKeyboard = true
	replyKeyboard.ResizeKeyboard = true

	promptMsgConfig := tgbotapi.NewMessage(chatID, msgText)
	promptMsgConfig.ReplyMarkup = replyKeyboard

	sentPromptMsg, err := bh.Deps.BotClient.Send(promptMsgConfig)
	if err != nil {
		log.Printf("SendLocationPrompt: Ошибка отправки запроса местоположения для chatID %d: %v", chatID, err)
		bh.sendErrorMessageHelper(chatID, originalAddressMenuMessageID, "❌ Ошибка запроса местоположения. Попробуйте снова.")
		bh.Deps.SessionManager.SetState(chatID, constants.STATE_ORDER_ADDRESS)
		bh.SendAddressInputMenu(chatID, originalAddressMenuMessageID)
		return
	}

	tempOrder := bh.Deps.SessionManager.GetTempOrder(chatID)
	tempOrder.LocationPromptMessageID = sentPromptMsg.MessageID
	tempOrder.CurrentMessageID = originalAddressMenuMessageID
	bh.Deps.SessionManager.UpdateTempOrder(chatID, tempOrder)
	log.Printf("Сообщение-промпт для местоположения (ID: %d) отправлено для chatID %d. CurrentMessageID сессии для след. шага: %d", sentPromptMsg.MessageID, chatID, originalAddressMenuMessageID)
}

// SendPhotoInputMenu отправляет меню для добавления фото/видео.
func (bh *BotHandler) SendPhotoInputMenu(chatID int64, messageIDToEdit int) {
	log.Printf("BotHandler.SendPhotoInputMenu для chatID %d, messageIDToEdit: %d", chatID, messageIDToEdit)
	bh.Deps.SessionManager.SetState(chatID, constants.STATE_ORDER_PHOTO)

	tempOrder := bh.Deps.SessionManager.GetTempOrder(chatID)
	photoCount := len(tempOrder.Photos)
	videoCount := len(tempOrder.Videos)

	msgTextFormat := "📸 Отправьте нам несколько фото или видео.\n" +
		"Это поможет нам точнее оценить объем работ."
	msgText := fmt.Sprintf(msgTextFormat)

	history := bh.Deps.SessionManager.GetHistory(chatID)
	isEditingOrder := false
	if len(history) >= 2 && history[len(history)-2] == constants.STATE_ORDER_EDIT {
		isEditingOrder = true
	}

	var rows [][]tgbotapi.InlineKeyboardButton
	if photoCount > 0 || videoCount > 0 {
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(fmt.Sprintf("👍 Готово (%d фото, %d видео)", photoCount, videoCount), "finish_photo_upload"),
		))
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🖼️ Просмотреть загруженное медиа", "view_uploaded_media"),
		))
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🗑 Сбросить всё медиа", "reset_photo_upload"),
		))
	} else {
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("➡️ Пропустить этот шаг", "skip_photo_initial"),
		))
	}

	backCallback := "back_to_address"
	if isEditingOrder {
		backCallback = "back_to_edit_menu_direct"
	}
	rows = append(rows, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("⬅️ Назад", backCallback),
		tgbotapi.NewInlineKeyboardButtonData("🏢 Главное меню", "back_to_main_confirm_cancel_order"),
	))
	keyboard := tgbotapi.NewInlineKeyboardMarkup(rows...)

	_, err := bh.sendOrEditMessageHelper(chatID, messageIDToEdit, msgText, &keyboard, tgbotapi.ModeMarkdown)
	if err != nil {
		log.Printf("SendPhotoInputMenu: Ошибка для chatID %d: %v", chatID, err)
	}
}

// SendPaymentSelectionMenu отправляет меню выбора способа оплаты.
func (bh *BotHandler) SendPaymentSelectionMenu(chatID int64, messageIDToEdit int) {
	log.Printf("BotHandler.SendPaymentSelectionMenu для chatID %d, messageIDToEdit: %d", chatID, messageIDToEdit)
	bh.Deps.SessionManager.SetState(chatID, constants.STATE_ORDER_PAYMENT)

	tempOrder := bh.Deps.SessionManager.GetTempOrder(chatID)
	history := bh.Deps.SessionManager.GetHistory(chatID)
	isEditingOrder := false
	if len(history) >= 2 && history[len(history)-2] == constants.STATE_ORDER_EDIT {
		isEditingOrder = true
	}

	backButtonCallback := "back_to_photo"
	if len(history) >= 2 {
		prevState := history[len(history)-2]
		if prevState == constants.STATE_ORDER_ADDRESS || prevState == constants.STATE_ORDER_ADDRESS_LOCATION {
			if len(tempOrder.Photos) == 0 && len(tempOrder.Videos) == 0 {
				backButtonCallback = "back_to_address"
			}
		}
	}
	if isEditingOrder {
		backButtonCallback = "back_to_edit_menu_direct"
	}

	msgText := "💳 Выберите способ оплаты. При оплате сразу — скидка 5%"

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("💳 Оплатить сразу (скидка 5%)", "payment_now")),
		tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("💵 Оплатить по выполнению", "payment_later")),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("⬅️ Назад", backButtonCallback),
			tgbotapi.NewInlineKeyboardButtonData("🏢 Главное меню", "back_to_main_confirm_cancel_order"),
		),
	)
	_, err := bh.sendOrEditMessageHelper(chatID, messageIDToEdit, msgText, &keyboard, tgbotapi.ModeMarkdown)
	if err != nil {
		log.Printf("SendPaymentSelectionMenu: Ошибка для chatID %d: %v", chatID, err)
	}
}

// SendOrderConfirmationMenu отправляет меню подтверждения заказа.
func (bh *BotHandler) SendOrderConfirmationMenu(chatID int64, messageIDToEdit int) {
	log.Printf("BotHandler.SendOrderConfirmationMenu для chatID %d, messageIDToEdit: %d", chatID, messageIDToEdit)

	tempOrder := bh.Deps.SessionManager.GetTempOrder(chatID)
	viewingUser, ok := bh.getUserFromDB(chatID)
	if !ok {
		bh.sendErrorMessageHelper(chatID, messageIDToEdit, "Не удалось получить данные пользователя.")
		return
	}

	isOperatorCreatingFlow := tempOrder.OrderAction == "operator_creating_order" && utils.IsOperatorOrHigher(viewingUser.Role)
	isDriverCreatingFlow := tempOrder.OrderAction == "driver_creating_order" && viewingUser.Role == constants.ROLE_DRIVER // Новое условие

	var orderID int64 = tempOrder.ID
	var orderStatus string = constants.STATUS_DRAFT
	actualClientChatID := tempOrder.UserChatID

	if tempOrder.ID == 0 { // Если это новый заказ (черновик еще не создан в БД)
		// Если создает водитель, то UserChatID и UserID в таблице orders будут NULL,
		// так как мы не знаем ID клиента.
		if isDriverCreatingFlow {
			tempOrder.UserChatID = 0 // Указываем, что у заказа нет привязанного пользователя в системе
		} else if actualClientChatID == 0 {
			actualClientChatID = chatID
			tempOrder.UserChatID = chatID
		}

		newOrderID, errCreate := db.CreateInitialOrder(tempOrder.Order)
		if errCreate != nil {
			log.Printf("Ошибка создания черновика заказа для chatID %d (клиент: %d): %v", chatID, actualClientChatID, errCreate)
			bh.sendErrorMessageHelper(chatID, messageIDToEdit, "❌ Ошибка сохранения данных заказа.")
			bh.Deps.SessionManager.SetState(chatID, constants.STATE_ORDER_PAYMENT)
			bh.SendPaymentSelectionMenu(chatID, messageIDToEdit)
			return
		}
		tempOrder.ID = newOrderID
		orderID = newOrderID
		bh.Deps.SessionManager.UpdateTempOrder(chatID, tempOrder)
		log.Printf("Черновик заказа #%d создан. ClientChatID в заказе: %d. Текущий chatID: %d", orderID, actualClientChatID, chatID)
	} else { // Заказ уже существует в БД
		statusFromDB, clientChatIDFromDB, errDb := db.GetOrderStatusAndClientChatID(tempOrder.ID)
		if errDb != nil {
			log.Printf("Ошибка получения статуса/клиента заказа #%d: %v", tempOrder.ID, errDb)
			bh.sendErrorMessageHelper(chatID, messageIDToEdit, "❌ Ошибка загрузки данных заказа.")
			bh.SendMainMenu(chatID, viewingUser, 0)
			return
		}
		orderStatus = statusFromDB
		actualClientChatID = clientChatIDFromDB
		tempOrder.UserChatID = actualClientChatID
		bh.Deps.SessionManager.UpdateTempOrder(chatID, tempOrder)
	}

	var keyboardRows [][]tgbotapi.InlineKeyboardButton
	var msgText string

	if isOperatorCreatingFlow || isDriverCreatingFlow {
		bh.Deps.SessionManager.SetState(chatID, constants.STATE_OP_ORDER_CONFIRMATION_OPTIONS)

		var client models.User
		if actualClientChatID != 0 {
			client, _ = db.GetUserByChatID(actualClientChatID)
		} else {
			// Для заказа, созданного водителем, создаем "виртуального" клиента из данных заказа
			client = models.User{
				FirstName: tempOrder.Name,
				Phone:     sql.NullString{String: tempOrder.Phone, Valid: true},
				ChatID:    0, // У клиента нет ChatID в нашей системе
			}
		}

		execs, _ := db.GetExecutorsByOrderID(int(orderID))
		title := fmt.Sprintf("✨ *Подтверждение Заказа №%d*", orderID)
		footer := "⚙️ *Опции для создателя заказа:*"
		msgText = formatters.FormatOrderDetailsForOperator(tempOrder.Order, client, execs, title, footer)

		keyboardRows = append(keyboardRows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("💰 Установить стоимость заказа", fmt.Sprintf("%s_%d", constants.CALLBACK_PREFIX_OP_CONFIRM_ORDER_SET_COST, orderID)),
		))
		keyboardRows = append(keyboardRows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("➡️ Назначить исполнителей", fmt.Sprintf("%s_%d", constants.CALLBACK_PREFIX_OP_SKIP_COST, orderID)),
		))
		keyboardRows = append(keyboardRows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("✅ Без стоимости и исполнителей", fmt.Sprintf("%s_%d", constants.CALLBACK_PREFIX_OP_CONFIRM_ORDER_SIMPLE_CREATE, orderID)),
		))
		keyboardRows = append(keyboardRows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("✏️ Редактировать", fmt.Sprintf("edit_order_%d", orderID)),
			tgbotapi.NewInlineKeyboardButtonData("🏢 Главное меню", "back_to_main_confirm_cancel_order"),
		))
	} else { // Клиент подтверждает свой заказ
		bh.Deps.SessionManager.SetState(chatID, constants.STATE_ORDER_CONFIRM)
		msgText = formatters.FormatOrderConfirmationForUser(tempOrder.Order)

		var confirmButtonText, confirmCallbackData string
		if orderStatus == constants.STATUS_DRAFT {
			confirmButtonText = "✅ Подтвердить и отправить оператору"
			confirmCallbackData = fmt.Sprintf("confirm_order_final_%d", orderID)
		} else { // AWAITING_CONFIRMATION или другой статус
			confirmButtonText = "👍 К моим заказам"
			confirmCallbackData = "my_orders_page_0"
		}
		keyboardRows = append(keyboardRows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(confirmButtonText, confirmCallbackData),
		))
		if orderStatus == constants.STATUS_DRAFT {
			keyboardRows = append(keyboardRows, tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("✏️ Редактировать мой заказ", fmt.Sprintf("edit_order_%d", orderID))))
			keyboardRows = append(keyboardRows, tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("❌ Отменить заказ", fmt.Sprintf("cancel_order_confirm_%d", orderID))))
		}
		keyboardRows = append(keyboardRows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🏢 Главное меню", "back_to_main"),
		))
	}

	keyboard := tgbotapi.NewInlineKeyboardMarkup(keyboardRows...)
	_, err := bh.sendOrEditMessageHelper(chatID, messageIDToEdit, msgText, &keyboard, tgbotapi.ModeMarkdown)
	if err != nil {
		log.Printf("SendOrderConfirmationMenu: Ошибка для chatID %d: %v", chatID, err)
	}
}

// SendOpOrderFinalConfirmMenu отображает оператору или водителю финальное подтверждение создаваемого заказа.
func (bh *BotHandler) SendOpOrderFinalConfirmMenu(chatID int64, orderID int64, messageIDToEdit int) {
	log.Printf("SendOpOrderFinalConfirmMenu: Пользователь %d подтверждает заказ #%d", chatID, orderID)
	bh.Deps.SessionManager.SetState(chatID, constants.STATE_OP_ORDER_FINAL_CONFIRM)

	tempOrder := bh.Deps.SessionManager.GetTempOrder(chatID)

	var client models.User
	if tempOrder.UserChatID != 0 {
		client, _ = db.GetUserByChatID(tempOrder.UserChatID)
	} else {
		// Для заказа, созданного водителем, UserChatID может быть 0
		client = models.User{
			FirstName: tempOrder.Name,
			Phone:     sql.NullString{String: tempOrder.Phone, Valid: true},
		}
	}

	assignedExecutors, _ := db.GetExecutorsByOrderID(int(orderID))

	title := fmt.Sprintf("✨ *Подтверждение Заказа №%d*", orderID)
	footer := "Заказ будет создан со статусом 'В работе'."
	msgText := formatters.FormatOrderDetailsForOperator(tempOrder.Order, client, assignedExecutors, title, footer)

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🚀 Подтвердить и создать", fmt.Sprintf("confirm_order_final_%d", orderID)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("⬅️ Назад к назначению исполнителей", fmt.Sprintf("%s_%d", constants.CALLBACK_PREFIX_OP_CONFIRM_ORDER_ASSIGN_EXEC, orderID)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🏢 Отменить создание", "back_to_main_confirm_cancel_order"),
		),
	)

	_, err := bh.sendOrEditMessageHelper(chatID, messageIDToEdit, msgText, &keyboard, tgbotapi.ModeMarkdown)
	if err != nil {
		log.Printf("SendOpOrderFinalConfirmMenu: Ошибка для chatID %d: %v", chatID, err)
	}
}

// SendEditOrderMenu отправляет меню редактирования заказа.
func (bh *BotHandler) SendEditOrderMenu(chatID int64, messageIDToEdit int) {
	log.Printf("BotHandler.SendEditOrderMenu для chatID %d, messageIDToEdit: %d", chatID, messageIDToEdit)
	bh.Deps.SessionManager.SetState(chatID, constants.STATE_ORDER_EDIT)

	tempOrder := bh.Deps.SessionManager.GetTempOrder(chatID)
	if tempOrder.ID == 0 {
		bh.sendErrorMessageHelper(chatID, messageIDToEdit, "❌ Нет активного заказа для редактирования.")
		user, ok := bh.getUserFromDB(chatID)
		if ok {
			bh.SendMainMenu(chatID, user, 0)
		}
		return
	}
	// Перезагружаем данные из БД, чтобы гарантировать актуальность перед редактированием
	orderFromDB, errDB := db.GetOrderByID(int(tempOrder.ID))
	if errDB != nil {
		bh.sendErrorMessageHelper(chatID, messageIDToEdit, "❌ Ошибка загрузки данных заказа для редактирования.")
		return
	}
	// Сохраняем ID текущего сообщения, если оно было передано и отличается от того, что в сессии
	currentMsgIDFromSession := tempOrder.CurrentMessageID
	if messageIDToEdit != 0 && messageIDToEdit != currentMsgIDFromSession {
		currentMsgIDFromSession = messageIDToEdit
	}

	// Обновляем данные в сессии из БД, сохраняя CurrentMessageID
	tempOrder.Order = orderFromDB
	tempOrder.CurrentMessageID = currentMsgIDFromSession // Восстанавливаем/устанавливаем актуальный ID для редактирования

	// Если CurrentMessageID обновился или еще не был в MediaMessageIDs, добавляем его.
	// Это нужно, чтобы sendOrEditMessageHelper корректно работал с этим сообщением как с "главным".
	if tempOrder.CurrentMessageID != 0 {
		found := false
		for _, id := range tempOrder.MediaMessageIDs {
			if id == tempOrder.CurrentMessageID {
				found = true
				break
			}
		}
		if !found {
			// Если CurrentMessageID (новое/текущее сообщение меню) не было среди MediaMessageIDs,
			// это означает, что мы перешли в новое меню. Очищаем старые медиа ID и ставим текущее.
			tempOrder.MediaMessageIDs = []int{tempOrder.CurrentMessageID}
			tempOrder.MediaMessageIDsMap = make(map[string]bool)
			tempOrder.MediaMessageIDsMap[fmt.Sprintf("%d", tempOrder.CurrentMessageID)] = true
		}
	}
	bh.Deps.SessionManager.UpdateTempOrder(chatID, tempOrder)

	timeStr := tempOrder.Time
	if timeStr == "" {
		timeStr = "В ближайшее время"
	}
	paymentStr := "По выполнению"
	if tempOrder.Payment == "now" {
		paymentStr = "Сразу (скидка 5%)"
	}
	formattedDate, _ := utils.FormatDateForDisplay(tempOrder.Date)
	formattedPhone := utils.FormatPhoneNumber(tempOrder.Phone)
	displaySubcategory := utils.GetDisplaySubcategory(tempOrder.Order)

	lines := []string{
		fmt.Sprintf("📋 Подкатегория: %s", displaySubcategory),
		fmt.Sprintf("📝 Описание: %s", utils.EscapeTelegramMarkdown(tempOrder.Description)),
		fmt.Sprintf("👤 Имя: %s", tempOrder.Name),
		fmt.Sprintf("📅 Дата: %s", formattedDate), fmt.Sprintf("⏰ Время: %s", timeStr),
		fmt.Sprintf("📱 Телефон: %s", formattedPhone), fmt.Sprintf("📍 Адрес: %s", tempOrder.Address),
	}
	if len(tempOrder.Photos) > 0 {
		lines = append(lines, fmt.Sprintf("📸 Фото: %d", len(tempOrder.Photos)))
	}
	if len(tempOrder.Videos) > 0 {
		lines = append(lines, fmt.Sprintf("🎥 Видео: %d", len(tempOrder.Videos)))
	}
	lines = append(lines, fmt.Sprintf("💳 Оплата: %s", paymentStr))

	viewingUser, _ := bh.getUserFromDB(chatID)
	isOperator := utils.IsOperatorOrHigher(viewingUser.Role)

	// Стоимость показывается всегда, если установлена
	costDisplay := "не установлена"
	if tempOrder.Cost.Valid && tempOrder.Cost.Float64 > 0 {
		costDisplay = fmt.Sprintf("%.0f ₽", tempOrder.Cost.Float64)
	}
	lines = append(lines, fmt.Sprintf("💰 Стоимость: *%s*", costDisplay))

	msgText := fmt.Sprintf("✏️ Редактировать заказ №%d:\n%s\n%s\n%s\nВыберите, что изменить:", tempOrder.ID, strings.Repeat("_", 30), strings.Join(lines, "\n"), strings.Repeat("_", 30))

	var keyboardRows [][]tgbotapi.InlineKeyboardButton
	keyboardRows = append(keyboardRows, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("📋 Подкатегория", fmt.Sprintf("edit_field_subcategory_%d", tempOrder.ID)),
		tgbotapi.NewInlineKeyboardButtonData("📝 Описание", fmt.Sprintf("edit_field_description_%d", tempOrder.ID)),
	))
	keyboardRows = append(keyboardRows, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("👤 Имя", fmt.Sprintf("edit_field_name_%d", tempOrder.ID)),
		tgbotapi.NewInlineKeyboardButtonData("📅 Дата", fmt.Sprintf("edit_field_date_%d", tempOrder.ID)),
	))
	keyboardRows = append(keyboardRows, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("⏰ Время", fmt.Sprintf("edit_field_time_%d", tempOrder.ID)),
		tgbotapi.NewInlineKeyboardButtonData("📱 Телефон", fmt.Sprintf("edit_field_phone_%d", tempOrder.ID)),
	))
	keyboardRows = append(keyboardRows, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("📍 Адрес", fmt.Sprintf("edit_field_address_%d", tempOrder.ID)),
		tgbotapi.NewInlineKeyboardButtonData("🖼️ Фото/Видео", fmt.Sprintf("edit_field_media_%d", tempOrder.ID)),
	))
	var paymentAndCostRow []tgbotapi.InlineKeyboardButton
	paymentAndCostRow = append(paymentAndCostRow, tgbotapi.NewInlineKeyboardButtonData("💳 Оплата", fmt.Sprintf("edit_field_payment_%d", tempOrder.ID)))

	if isOperator {
		// Кнопка "Стоимость" для оператора
		paymentAndCostRow = append(paymentAndCostRow, tgbotapi.NewInlineKeyboardButtonData("💰 Стоимость", fmt.Sprintf("%s_%d", constants.CALLBACK_PREFIX_OP_EDIT_ORDER_COST, tempOrder.ID)))
	}
	if len(paymentAndCostRow) > 0 {
		keyboardRows = append(keyboardRows, paymentAndCostRow)
	}

	if isOperator {
		// Кнопка "Исполнители" для оператора
		keyboardRows = append(keyboardRows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("👷 Исполнители", fmt.Sprintf("%s_%d", constants.CALLBACK_PREFIX_OP_EDIT_ORDER_EXECS, tempOrder.ID)),
		))
	}

	// Для оператора, создающего заказ, кнопка "Назад к опциям"
	if tempOrder.OrderAction == "operator_creating_order" {
		keyboardRows = append(keyboardRows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("✅ Сохранить и к опциям создания", fmt.Sprintf("back_to_op_confirm_options_%d", tempOrder.ID)),
		))
	} else { // Для клиента или оператора, редактирующего существующий заказ
		keyboardRows = append(keyboardRows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("✅ Сохранить и к подтверждению", fmt.Sprintf("back_to_confirm_%d", tempOrder.ID)),
		))
	}

	keyboard := tgbotapi.NewInlineKeyboardMarkup(keyboardRows...)

	finalMessageIDToEditForMenu := tempOrder.CurrentMessageID // Используем обновленный ID из сессии
	if finalMessageIDToEditForMenu == 0 {
		finalMessageIDToEditForMenu = messageIDToEdit // Фоллбэк, если в сессии 0
	}

	_, err := bh.sendOrEditMessageHelper(chatID, finalMessageIDToEditForMenu, msgText, &keyboard, tgbotapi.ModeMarkdown)
	if err != nil {
		log.Printf("SendEditOrderMenu: Ошибка для chatID %d: %v", chatID, err)
	}
}

// SendAskToCancelOrderConfirmation запрашивает подтверждение отмены заказа.
func (bh *BotHandler) SendAskToCancelOrderConfirmation(chatID int64, messageIDToEdit int, originalStepMessageID int) {
	log.Printf("BotHandler.SendAskToCancelOrderConfirmation для chatID %d, редактируемое сообщение: %d, исходное сообщение шага: %d", chatID, messageIDToEdit, originalStepMessageID)

	msgText := "Вы уверены, что хотите отменить создание/редактирование заказа и вернуться в главное меню?\n\n⚠️ Все введенные данные для этого заказа будут потеряны."
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("✅ Да, отменить", "back_to_main_confirmed_cancel_final"),
			tgbotapi.NewInlineKeyboardButtonData("❌ Нет, продолжить", fmt.Sprintf("resume_order_creation_%d", originalStepMessageID)),
		),
	)
	// Используем sendOrEditMessageHelper, чтобы он обновил CurrentMessageID
	sentMsg, err := bh.sendOrEditMessageHelper(chatID, messageIDToEdit, msgText, &keyboard, tgbotapi.ModeMarkdown)
	if err != nil {
		log.Printf("SendAskToCancelOrderConfirmation: Ошибка для chatID %d: %v", chatID, err)
	} else if sentMsg.MessageID != 0 {
		// CurrentMessageID обновлен в sendOrEditMessageHelper
		log.Printf("SendAskToCancelOrderConfirmation: CurrentMessageID обновлен на %d", sentMsg.MessageID)
	}
}

// SendCostInputPrompt запрашивает у оператора ввод стоимости заказа.
// Эта функция может использоваться как при создании заказа оператором, так и при редактировании.
func (bh *BotHandler) SendCostInputPrompt(chatID int64, orderID int, messageIDToEdit int) {
	log.Printf("BotHandler.SendCostInputPrompt для заказа #%d, пользователь chatID %d, messageIDToEdit %d", orderID, chatID, messageIDToEdit)

	tempOrder := bh.Deps.SessionManager.GetTempOrder(chatID)
	tempOrder.ID = int64(orderID)

	var backCallbackKey string
	// Определяем, откуда мы пришли, чтобы кнопка "Назад" работала корректно
	if tempOrder.OrderAction == "operator_creating_order" || tempOrder.OrderAction == "driver_creating_order" {
		// Если мы в потоке создания заказа, "Назад" ведет к опциям подтверждения
		bh.Deps.SessionManager.SetState(chatID, constants.STATE_OP_ORDER_COST_INPUT)
		backCallbackKey = fmt.Sprintf("back_to_op_confirm_options_%d", orderID)
	} else {
		// Если редактируем существующий заказ или устанавливаем стоимость для уже созданного
		bh.Deps.SessionManager.SetState(chatID, constants.STATE_COST_INPUT)
		backCallbackKey = fmt.Sprintf("view_order_ops_%d", orderID)
		history := bh.Deps.SessionManager.GetHistory(chatID)
		if len(history) > 1 && history[len(history)-2] == constants.STATE_ORDER_EDIT {
			backCallbackKey = fmt.Sprintf("back_to_edit_menu_direct_%d", orderID)
		}
	}

	bh.Deps.SessionManager.UpdateTempOrder(chatID, tempOrder)

	msgText := fmt.Sprintf("💰 Введите стоимость для заказа №%d (в рублях, например, 1500):", orderID)
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("⬅️ Назад", backCallbackKey),
		),
	)
	_, err := bh.sendOrEditMessageHelper(chatID, messageIDToEdit, msgText, &keyboard, tgbotapi.ModeMarkdown)
	if err != nil {
		log.Printf("SendCostInputPrompt: Ошибка для chatID %d: %v", chatID, err)
	}
}

// SendCancelReasonInput запрашивает причину отмены заказа.
func (bh *BotHandler) SendCancelReasonInput(chatID int64, orderID int, messageIDToEdit int, context string) {
	log.Printf("BotHandler.SendCancelReasonInput для заказа #%d, chatID %d, контекст: %s", orderID, chatID, context)
	bh.Deps.SessionManager.SetState(chatID, constants.STATE_CANCEL_REASON)

	tempData := bh.Deps.SessionManager.GetTempOrder(chatID)
	tempData.ID = int64(orderID)
	tempData.OrderAction = context // 'reject_cost', 'operator_cancel', 'user_cancel_draft_or_awaiting_cost_no_cost'
	// tempData.CurrentMessageID = messageIDToEdit; // sendOrEditMessageHelper обновит
	bh.Deps.SessionManager.UpdateTempOrder(chatID, tempData)

	msgText := fmt.Sprintf("📝 Пожалуйста, укажите причину отмены/отклонения для заказа №%d:", orderID)

	var backCallback string
	if context == "reject_cost" { // Клиент отклоняет стоимость
		backCallback = fmt.Sprintf("confirm_order_final_%d", orderID) // Вернуться к подтверждению заказа (где были кнопки принять/отклонить)
	} else if context == "operator_cancel" { // Оператор отменяет
		backCallback = fmt.Sprintf("view_order_ops_%d", orderID) // Вернуться к деталям заказа
	} else { // Клиент отменяет черновик/новый без стоимости
		backCallback = fmt.Sprintf("confirm_order_final_%d", orderID) // Вернуться к подтверждению заказа
	}

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("⬅️ Назад", backCallback),
		),
	)
	_, err := bh.sendOrEditMessageHelper(chatID, messageIDToEdit, msgText, &keyboard, tgbotapi.ModeMarkdown)
	if err != nil {
		log.Printf("SendCancelReasonInput: Ошибка для chatID %d: %v", chatID, err)
	}
}

// --- Новые функции для операторского потока ---

// SendOpOrderCostInputMenu запрашивает у оператора ввод стоимости для создаваемого заказа.
func (bh *BotHandler) SendOpOrderCostInputMenu(chatID int64, orderID int64, messageIDToEdit int) {
	log.Printf("SendOpOrderCostInputMenu: Оператор %d вводит стоимость для заказа #%d", chatID, orderID)
	bh.Deps.SessionManager.SetState(chatID, constants.STATE_OP_ORDER_COST_INPUT)

	tempOrder := bh.Deps.SessionManager.GetTempOrder(chatID)
	tempOrder.ID = orderID // Убедимся, что ID заказа есть в сессии
	bh.Deps.SessionManager.UpdateTempOrder(chatID, tempOrder)

	msgText := fmt.Sprintf("💰 *Установка стоимости*\nВведите стоимость для заказа №%d (например, 1500).\nЭто значение будет показано клиенту (если применимо).", orderID)

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("➡️ Пропустить этот шаг", fmt.Sprintf("%s_%d", constants.CALLBACK_PREFIX_OP_SKIP_COST, orderID)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("⬅️ Назад к опциям создания", fmt.Sprintf("back_to_op_confirm_options_%d", orderID)),
		),
	)
	_, err := bh.sendOrEditMessageHelper(chatID, messageIDToEdit, msgText, &keyboard, tgbotapi.ModeMarkdown)
	if err != nil {
		log.Printf("SendOpOrderCostInputMenu: Ошибка для chatID %d: %v", chatID, err)
	}
}

// SendOpAssignExecutorsMenu отправляет оператору меню назначения исполнителей для создаваемого заказа.
func (bh *BotHandler) SendOpAssignExecutorsMenu(chatID int64, orderID int64, messageIDToEdit int) {
	// Эта функция теперь вызывает основную реализацию из callback_order_view_manage_handlers
	bh.SendAssignExecutorsMenu(chatID, orderID, messageIDToEdit)
}
