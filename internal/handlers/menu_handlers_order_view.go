// Файл: internal/handlers/menu_handlers_order_view.go
package handlers

import (
	"Original/internal/constants"
	"Original/internal/db"
	"Original/internal/formatters"
	"Original/internal/models"
	"Original/internal/utils" // Для EscapeTelegramMarkdown, FormatDateForDisplay, GetDisplaySubcategory, ValidateDate, GetUserDisplayName
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	tgbotapi "github.com/OvyFlash/telegram-bot-api"
)

// Вспомогательная структура для времени заказа (ранее в formatters.go)
type parsedOrderTimeInternal struct {
	Hour   int
	Minute int
	IsASAP bool
}

// Константы приоритетов (ранее в formatters.go)
const (
	priorityAsapInternal   = 1
	priorityTodayInternal  = 2
	priorityFutureInternal = 3
)

// Вспомогательная функция для парсинга времени заказа (ранее ParseOrderTime в formatters.go)
func parseOrderTimeLogic(timeStr string) parsedOrderTimeInternal {
	trimmedTimeStr := strings.TrimSpace(strings.ToUpper(timeStr))
	if trimmedTimeStr == "СРОЧНО" || trimmedTimeStr == "" {
		return parsedOrderTimeInternal{IsASAP: true}
	}
	if trimmedTimeStr == "В БЛИЖАЙШЕЕ ВРЕМЯ" || trimmedTimeStr == "❗ СРОЧНО (В БЛИЖАЙШЕЕ ВРЕМЯ) ❗" {
		return parsedOrderTimeInternal{IsASAP: true}
	}

	parts := strings.Split(trimmedTimeStr, ":")
	if len(parts) == 2 {
		hour, errH := strconv.Atoi(strings.TrimSpace(parts[0]))
		minute, errM := strconv.Atoi(strings.TrimSpace(parts[1]))
		if errH == nil && errM == nil && hour >= 0 && hour <= 23 && minute >= 0 && minute <= 59 {
			return parsedOrderTimeInternal{Hour: hour, Minute: minute, IsASAP: false}
		}
	}
	log.Printf("[SendOrderListByStatus:parseOrderTimeLogic] не удалось стандартно распарсить время '%s' как ЧЧ:ММ, считаем его эквивалентом ASAP.", timeStr)
	return parsedOrderTimeInternal{IsASAP: true}
}

// Вспомогательная функция для определения приоритета сортировки (ранее GetOrderSortPriority в formatters.go)
func getOrderSortPriorityLogic(order models.Order, today time.Time) (int, time.Time, parsedOrderTimeInternal) {
	log.Printf("[SendOrderListByStatus:getOrderSortPriorityLogic] Start for OrderID=%d, order.Date='%s', order.Time='%s', todayForFunc='%s'", order.ID, order.Date, order.Time, today.Format("2006-01-02"))

	var parsedOrderDate time.Time
	var errParseDate error

	if order.Date != "" {
		parsedOrderDate, errParseDate = utils.ValidateDate(order.Date)
		if errParseDate != nil {
			log.Printf("[SendOrderListByStatus:getOrderSortPriorityLogic] OrderID=%d, ValidateDate for '%s' FAILED: %v. Order.Time is '%s'.", order.ID, order.Date, errParseDate, order.Time)
			if strings.ToUpper(order.Time) == "СРОЧНО" {
				parsedOrderDate = today
				log.Printf("[SendOrderListByStatus:getOrderSortPriorityLogic] OrderID=%d, Date parse failed, Time is СРОЧНО. parsedOrderDate set to today: %s", order.ID, parsedOrderDate.Format("2006-01-02"))
			} else {
				parsedOrderDate = today.AddDate(100, 0, 0)
				log.Printf("[SendOrderListByStatus:getOrderSortPriorityLogic] OrderID=%d, Date parse failed, Time not СРОЧНО. parsedOrderDate set to far future: %s", order.ID, parsedOrderDate.Format("2006-01-02"))
			}
		} else {
			log.Printf("[SendOrderListByStatus:getOrderSortPriorityLogic] OrderID=%d, ValidateDate for '%s' SUCCESS. parsedOrderDate before In/Truncate: %s (Location: %s)", order.ID, order.Date, parsedOrderDate.Format("2006-01-02 15:04:05 MST"), parsedOrderDate.Location().String())
		}
	} else {
		log.Printf("[SendOrderListByStatus:getOrderSortPriorityLogic] OrderID=%d, order.Date is EMPTY. Order.Time is '%s'.", order.ID, order.Time)
		if strings.ToUpper(order.Time) == "СРОЧНО" {
			parsedOrderDate = today
			log.Printf("[SendOrderListByStatus:getOrderSortPriorityLogic] OrderID=%d, Date empty, Time is СРОЧНО. parsedOrderDate set to today: %s", order.ID, parsedOrderDate.Format("2006-01-02"))
		} else {
			parsedOrderDate = today.AddDate(99, 0, 0)
			log.Printf("[SendOrderListByStatus:getOrderSortPriorityLogic] OrderID=%d, Date empty, Time not СРОЧНО. parsedOrderDate set to near future: %s", order.ID, parsedOrderDate.Format("2006-01-02"))
		}
	}

	parsedOrderDate = parsedOrderDate.In(today.Location()).Truncate(24 * time.Hour)
	log.Printf("[SendOrderListByStatus:getOrderSortPriorityLogic] OrderID=%d, Normalized parsedOrderDate: %s (Location: %s). Today for compare: %s (Location: %s)", order.ID, parsedOrderDate.Format("2006-01-02"), parsedOrderDate.Location().String(), today.Format("2006-01-02"), today.Location().String())

	parsedPTime := parseOrderTimeLogic(order.Time)
	log.Printf("[SendOrderListByStatus:getOrderSortPriorityLogic] OrderID=%d, parseOrderTimeLogic('%s') result: IsASAP=%t, Hour=%d, Minute=%d", order.ID, order.Time, parsedPTime.IsASAP, parsedPTime.Hour, parsedPTime.Minute)

	var finalPrio int
	var finalDate time.Time
	var finalPTime parsedOrderTimeInternal

	if parsedPTime.IsASAP {
		finalPrio = priorityAsapInternal
		finalPTime = parsedPTime
		log.Printf("[SendOrderListByStatus:getOrderSortPriorityLogic] OrderID=%d, Path: IsASAP=true. Comparing parsedOrderDate=%s with today=%s", order.ID, parsedOrderDate.Format("2006-01-02"), today.Format("2006-01-02"))
		if parsedOrderDate.Before(today) {
			finalDate = today
			log.Printf("[SendOrderListByStatus:getOrderSortPriorityLogic] OrderID=%d, Assigned prio: %d (priorityAsapInternal - date was past, normalized to today). Date for sort: %s", order.ID, finalPrio, finalDate.Format("2006-01-02"))
		} else {
			finalDate = parsedOrderDate
			log.Printf("[SendOrderListByStatus:getOrderSortPriorityLogic] OrderID=%d, Assigned prio: %d (priorityAsapInternal - date is today or future). Date for sort: %s", order.ID, finalPrio, finalDate.Format("2006-01-02"))
		}
		return finalPrio, finalDate, finalPTime
	}

	log.Printf("[SendOrderListByStatus:getOrderSortPriorityLogic] OrderID=%d, Path: IsASAP=false. Comparing parsedOrderDate=%s with today=%s", order.ID, parsedOrderDate.Format("2006-01-02"), today.Format("2006-01-02"))
	if parsedOrderDate.Equal(today) {
		finalPrio = priorityTodayInternal
		finalDate = parsedOrderDate
		finalPTime = parsedPTime
		log.Printf("[SendOrderListByStatus:getOrderSortPriorityLogic] OrderID=%d, Assigned prio: %d (priorityTodayInternal - date is today). Date for sort: %s", order.ID, finalPrio, finalDate.Format("2006-01-02"))
		return finalPrio, finalDate, finalPTime
	}

	if parsedOrderDate.Before(today) {
		finalPrio = priorityTodayInternal
		finalDate = parsedOrderDate
		finalPTime = parsedPTime
		log.Printf("[SendOrderListByStatus:getOrderSortPriorityLogic] OrderID=%d, Assigned prio: %d (priorityTodayInternal - date was past, non-ASAP). Date for sort: %s", order.ID, finalPrio, finalDate.Format("2006-01-02"))
		return finalPrio, finalDate, finalPTime
	}

	finalPrio = priorityFutureInternal
	finalDate = parsedOrderDate
	finalPTime = parsedPTime
	log.Printf("[SendOrderListByStatus:getOrderSortPriorityLogic] OrderID=%d, Assigned prio: %d (priorityFutureInternal). Date for sort: %s", order.ID, finalPrio, finalDate.Format("2006-01-02"))
	return finalPrio, finalDate, finalPTime
}

// SendMyOrdersMenu отправляет меню "Мои заказы" пользователю с пагинацией.
func (bh *BotHandler) SendMyOrdersMenu(chatID int64, user models.User, messageIDToEdit int, page int) {
	log.Printf("BotHandler.SendMyOrdersMenu для chatID %d (UserID: %d, Роль: %s), страница: %d, messageIDToEdit: %d", chatID, user.ID, user.Role, page, messageIDToEdit)
	bh.Deps.SessionManager.SetState(chatID, constants.STATE_IDLE)

	var orders []models.Order
	var err error
	var callbackViewPrefix string = "view_order_"

	switch user.Role {
	case constants.ROLE_DRIVER:
		orders, err = db.GetOrdersByExecutorIDAndStatuses(user.ID, constants.ROLE_DRIVER, []string{}, page, constants.OrdersPerPage)
		callbackViewPrefix = "view_order_ops_"
	case constants.ROLE_LOADER:
		orders, err = db.GetOrdersByExecutorIDAndStatuses(user.ID, constants.ROLE_LOADER, []string{}, page, constants.OrdersPerPage)
		callbackViewPrefix = "view_order_ops_"
	case constants.ROLE_USER:
		orders, err = db.GetOrdersByChatIDAndStatus(chatID, "", page)
	default: // Для операторов и выше, показываем заказы, где они являются клиентом (если такие есть), или их основной список заказов.
		// Пока что для простоты, операторы и выше увидят свои "клиентские" заказы здесь.
		// Основное управление заказами - через SendOrdersMenu.
		log.Printf("SendMyOrdersMenu: 'Мои заказы' для роли %s (ChatID %d). Показываем как для клиента.", user.Role, chatID)
		orders, err = db.GetOrdersByChatIDAndStatus(chatID, "", page)
	}

	if err != nil {
		log.Printf("SendMyOrdersMenu: ошибка получения заказов для UserID %d (ChatID %d, Role %s): %v", user.ID, chatID, user.Role, err)
		_, _ = bh.sendErrorMessageHelper(chatID, messageIDToEdit, "❌ Ошибка загрузки ваших заказов. Попробуйте позже.")
		return
	}

	var msgText string
	var keyboard tgbotapi.InlineKeyboardMarkup
	var rows [][]tgbotapi.InlineKeyboardButton

	if len(orders) == 0 && page == 0 {
		msgText = "📋 У вас пока нет заказов."
		if user.Role == constants.ROLE_USER {
			msgText += " Создайте новый заказ в главном меню! 🚀\n\n"
		}
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔙 Главное меню", "back_to_main"),
		))
	} else if len(orders) == 0 && page > 0 {
		msgText = "📋 Больше заказов нет."
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("⬅️ Назад", fmt.Sprintf("my_orders_page_%d", page-1)),
		))
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔙 Главное меню", "back_to_main"),
		))
	} else {
		msgText = "📋 Ваши заказы:\n\n💡 Нажмите на заказ, чтобы увидеть подробности!"
		for _, orderItem := range orders {
			statusEmoji := constants.StatusEmojiMap[orderItem.Status]
			if statusEmoji == "" {
				statusEmoji = "🆕"
			}
			categoryEmoji := constants.CategoryEmojiMap[orderItem.Category]
			if categoryEmoji == "" {
				categoryEmoji = "❓"
			}
			buttonText := fmt.Sprintf("%s Заказ №%d | %s %s",
				statusEmoji,
				orderItem.ID,
				categoryEmoji,
				utils.StripEmoji(constants.CategoryDisplayMap[orderItem.Category]),
			)
			formattedDate, _ := utils.FormatDateForDisplay(orderItem.Date)
			buttonText += fmt.Sprintf(" (%s)", formattedDate)

			if len(buttonText) > 60 { // Ограничение Telegram на длину текста кнопки
				buttonText = buttonText[:57] + "..."
			}

			rows = append(rows, tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData(buttonText, fmt.Sprintf("%s%d", callbackViewPrefix, orderItem.ID)),
			))
		}

		navRow := []tgbotapi.InlineKeyboardButton{}
		if page > 0 {
			navRow = append(navRow, tgbotapi.NewInlineKeyboardButtonData("⬅️ Назад", fmt.Sprintf("my_orders_page_%d", page-1)))
		}
		if len(orders) == constants.OrdersPerPage { // Если заказов на странице ровно OrdersPerPage, возможно, есть еще
			navRow = append(navRow, tgbotapi.NewInlineKeyboardButtonData("➡️ Далее", fmt.Sprintf("my_orders_page_%d", page+1)))
		}
		if len(navRow) > 0 {
			rows = append(rows, navRow)
		}

		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔙 Главное меню", "back_to_main"),
		))
	}
	keyboard.InlineKeyboard = rows

	_, errSend := bh.sendOrEditMessageHelper(chatID, messageIDToEdit, msgText, &keyboard, tgbotapi.ModeMarkdown)
	if errSend != nil {
		log.Printf("SendMyOrdersMenu: Ошибка отправки меню для chatID %d: %v", chatID, errSend)
	}
}

// SendOrdersMenu (для оператора) отправляет меню управления заказами.
func (bh *BotHandler) SendOrdersMenu(chatID int64, user models.User, messageIDToEdit int) {
	log.Printf("BotHandler.SendOrdersMenu для оператора chatID %d, messageIDToEdit: %d", chatID, messageIDToEdit)
	bh.Deps.SessionManager.SetState(chatID, constants.STATE_IDLE)

	msgText := "📦 Управление заказами"
	var keyboardRows [][]tgbotapi.InlineKeyboardButton

	keyboardRows = append(keyboardRows, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData(
			fmt.Sprintf("%s %s", constants.StatusEmojiMap[constants.STATUS_NEW], constants.StatusDisplayMap[constants.STATUS_NEW]),
			constants.OrderListCallbackMap[constants.ORDER_LIST_KEY_NEW]),
		tgbotapi.NewInlineKeyboardButtonData(
			fmt.Sprintf("%s %s", constants.StatusEmojiMap[constants.STATUS_AWAITING_CONFIRMATION], constants.StatusDisplayMap[constants.STATUS_AWAITING_CONFIRMATION]),
			constants.OrderListCallbackMap[constants.ORDER_LIST_KEY_AWAITING_CONFIRMATION]),
	))
	keyboardRows = append(keyboardRows, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData(
			fmt.Sprintf("%s %s", constants.StatusEmojiMap[constants.STATUS_INPROGRESS], constants.StatusDisplayMap[constants.STATUS_INPROGRESS]),
			constants.OrderListCallbackMap[constants.ORDER_LIST_KEY_IN_PROGRESS]),
		tgbotapi.NewInlineKeyboardButtonData(
			fmt.Sprintf("%s %s", constants.StatusEmojiMap[constants.STATUS_COMPLETED], constants.StatusDisplayMap[constants.STATUS_COMPLETED]),
			constants.OrderListCallbackMap[constants.ORDER_LIST_KEY_COMPLETED]),
	))
	keyboardRows = append(keyboardRows, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData(
			fmt.Sprintf("%s %s", constants.StatusEmojiMap[constants.STATUS_CALCULATED], constants.StatusDisplayMap[constants.STATUS_CALCULATED]),
			fmt.Sprintf("operator_orders_%s_0", constants.STATUS_CALCULATED)), // Используем прямой коллбэк
		tgbotapi.NewInlineKeyboardButtonData(
			fmt.Sprintf("%s %s", constants.StatusEmojiMap[constants.STATUS_CANCELED], constants.StatusDisplayMap[constants.STATUS_CANCELED]),
			constants.OrderListCallbackMap[constants.ORDER_LIST_KEY_CANCELED]),
	))
	if utils.IsOperatorOrHigher(user.Role) {
		keyboardRows = append(keyboardRows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("➕ Создать заказ для клиента", constants.CALLBACK_PREFIX_OP_CREATE_NEW_ORDER), // Обновленный коллбэк
		))
	}

	keyboardRows = append(keyboardRows, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("🔙 Главное меню", "back_to_main"),
	))
	keyboard := tgbotapi.NewInlineKeyboardMarkup(keyboardRows...)

	_, err := bh.sendOrEditMessageHelper(chatID, messageIDToEdit, msgText, &keyboard, tgbotapi.ModeMarkdown)
	if err != nil {
		log.Printf("SendOrdersMenu: Ошибка для chatID %d: %v", chatID, err)
	}
}

// SendOrderListByStatus (для оператора) отображает список заказов по указанным статусам с пагинацией.
func (bh *BotHandler) SendOrderListByStatus(
	chatID int64,
	statuses []string,
	statusKeyForCallback string, // Например, "new", "in_progress"
	page int,
	messageIDToEdit int,
	menuTitle string,
	noOrdersText string,
) {
	log.Printf("[SendOrderListByStatus] Start for chatID %d, статусы: %v, страница: %d, ключ_cb: %s", chatID, statuses, page, statusKeyForCallback)

	orderByFieldDB := "o.created_at"
	orderByDirectionDB := "DESC"

	isFinalStatusList := false
	for _, s := range statuses {
		if s == constants.STATUS_COMPLETED || s == constants.STATUS_CANCELED || s == constants.STATUS_CALCULATED || s == constants.STATUS_SETTLED {
			isFinalStatusList = true
			break
		}
	}
	if isFinalStatusList {
		orderByFieldDB = "o.updated_at" // Для финальных статусов сортируем по дате последнего обновления
		orderByDirectionDB = "DESC"
	}

	ordersFromDB, err := db.GetOrdersByMultipleStatuses(statuses, page, orderByFieldDB, orderByDirectionDB)
	if err != nil {
		log.Printf("[SendOrderListByStatus] Ошибка получения заказов (статусы %v, sortDB: %s %s) для chatID %d: %v", statuses, orderByFieldDB, orderByDirectionDB, chatID, err)
		bh.sendErrorMessageHelper(chatID, messageIDToEdit, "❌ Ошибка загрузки списка заказов.")
		return
	}

	today := time.Now().In(time.Local).Truncate(24 * time.Hour) // Убедимся, что today в Local и без времени
	orders := ordersFromDB                                      // Используем напрямую из БД, так как сортировка уже там

	// Если нужны специфичные правила сортировки, которых нет в SQL, раскомментировать и адаптировать:
	/*
		if len(orders) > 0 && !isFinalStatusList { // Сортируем только нефинальные статусы по логике приоритета
			sort.Slice(orders, func(i, j int) bool {
				prioI, dateI, timeI := getOrderSortPriorityLogic(orders[i], today)
				prioJ, dateJ, timeJ := getOrderSortPriorityLogic(orders[j], today)

				if prioI != prioJ {
					return prioI < prioJ
				}
				// Дальнейшая сортировка по дате, времени, ID создания...
				if !dateI.Equal(dateJ) {
					return dateI.Before(dateJ)
				}
				if !timeI.IsASAP && !timeJ.IsASAP { // Если оба времени конкретные
					if timeI.Hour != timeJ.Hour {
						return timeI.Hour < timeJ.Hour
					}
					if timeI.Minute != timeJ.Minute {
						return timeI.Minute < timeJ.Minute
					}
				} else if timeI.IsASAP && !timeJ.IsASAP { // Срочный раньше конкретного
					return true
				} else if !timeI.IsASAP && timeJ.IsASAP { // Конкретное позже срочного
					return false
				}
				// Если приоритеты, даты и время (или срочность) одинаковы, сортируем по ID (новые вверху)
				return orders[i].ID > orders[j].ID
			})
		}
	*/

	var msgText string
	var keyboard tgbotapi.InlineKeyboardMarkup
	var rows [][]tgbotapi.InlineKeyboardButton

	if len(orders) == 0 && page == 0 {
		msgText = noOrdersText
	} else if len(orders) == 0 && page > 0 {
		msgText = "📋 Больше заказов в этой категории нет."
	} else {
		msgText = menuTitle + "\n\n💡 Нажмите на заказ для просмотра деталей и действий."
		for _, orderItem := range orders {
			// _, orderDateForDisplayLogic, orderParsedTime := getOrderSortPriorityLogic(orderItem, today)

			categoryEmoji := constants.CategoryEmojiMap[orderItem.Category]
			if categoryEmoji == "" {
				categoryEmoji = "❓"
			}

			clientName := utils.EscapeTelegramMarkdown(orderItem.Name)
			if clientName == "" {
				clientUser, errUser := db.GetUserByID(orderItem.UserID)
				if errUser == nil {
					clientName = utils.GetUserDisplayName(clientUser)
				} else {
					clientName = fmt.Sprintf("Клиент ID %d", orderItem.UserID)
				}
			}

			newOrderIndicator := ""
			if orderItem.Status == constants.STATUS_NEW || orderItem.Status == constants.STATUS_AWAITING_COST {
				newOrderIndicator = constants.StatusEmojiMap[constants.STATUS_NEW]
				if newOrderIndicator == "" {
					newOrderIndicator = "🆕"
				}
			} else {
				newOrderIndicator = constants.StatusEmojiMap[orderItem.Status]
				if newOrderIndicator == "" {
					newOrderIndicator = "⚙️"
				}
			}

			var executorStatuses string
			assignedExecutors, errExec := db.GetExecutorsByOrderID(int(orderItem.ID))
			if errExec == nil && len(assignedExecutors) > 0 {
				notifiedCount := 0
				totalAssigned := len(assignedExecutors)
				for _, exec := range assignedExecutors {
					if exec.IsNotified {
						notifiedCount++
					}
				}
				if totalAssigned > 0 {
					if notifiedCount == totalAssigned {
						executorStatuses = "🟢"
					} else if notifiedCount == 0 {
						executorStatuses = "⭕️"
					} else {
						executorStatuses = fmt.Sprintf("%d/%d 🟡", notifiedCount, totalAssigned)
					}
				}
			} else if errExec != nil {
				log.Printf("[SendOrderListByStatus] Ошибка получения исполнителей для заказа #%d: %v", orderItem.ID, errExec)
			}

			var buttonText string
			var displayParts []string

			displayParts = append(displayParts, newOrderIndicator)
			displayParts = append(displayParts, fmt.Sprintf("№%d", orderItem.ID))
			displayParts = append(displayParts, categoryEmoji)

			displayDate, errDateDisp := utils.FormatDateForDisplay(orderItem.Date)
			if errDateDisp != nil || orderItem.Date == "" {
				displayDate = "??.?? "
			} else {
				// Убираем год, если это текущий год
				parsedOrderDate, _ := utils.ValidateDate(orderItem.Date)
				if parsedOrderDate.Year() == today.Year() {
					displayDate = strings.TrimSuffix(displayDate, fmt.Sprintf(" %d", today.Year())) // Не лучший способ, но для примера
				}
			}

			timeStr := orderItem.Time
			if timeStr == "" || strings.ToUpper(timeStr) == "СРОЧНО" || strings.ToLower(timeStr) == "в ближайшее время" {
				timeStr = "❗️Срочно"
			} else {
				// Убираем :00, если это ровный час
				if strings.HasSuffix(timeStr, ":00") {
					timeStr = timeStr[:2] + "ч"
				}
			}
			displayParts = append(displayParts, displayDate+timeStr)
			displayParts = append(displayParts, clientName)

			if executorStatuses != "" {
				displayParts = append(displayParts, executorStatuses)
			}

			buttonText = strings.Join(displayParts, " | ")

			if len(buttonText) > 64 { // Ограничение Telegram
				buttonText = buttonText[:61] + "..."
			}
			rows = append(rows, tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData(buttonText, fmt.Sprintf("view_order_ops_%d", orderItem.ID)),
			))
		}
	}

	navRow := []tgbotapi.InlineKeyboardButton{}
	if page > 0 {
		navRow = append(navRow, tgbotapi.NewInlineKeyboardButtonData("⬅️ Назад", fmt.Sprintf("operator_orders_%s_%d", statusKeyForCallback, page-1)))
	}
	// Кнопка "Далее" если есть еще заказы (проверяем, было ли получено максимальное количество на этой странице)
	if len(ordersFromDB) == constants.OrdersPerPage {
		navRow = append(navRow, tgbotapi.NewInlineKeyboardButtonData("➡️ Далее", fmt.Sprintf("operator_orders_%s_%d", statusKeyForCallback, page+1)))
	}
	if len(navRow) > 0 {
		rows = append(rows, navRow)
	}

	rows = append(rows, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("📦 Заказы", "manage_orders"),
		tgbotapi.NewInlineKeyboardButtonData("🔙 Главное меню", "back_to_main"),
	))
	keyboard.InlineKeyboard = rows

	_, errSend := bh.sendOrEditMessageHelper(chatID, messageIDToEdit, msgText, &keyboard, tgbotapi.ModeMarkdown)
	if errSend != nil {
		log.Printf("[SendOrderListByStatus] Ошибка отправки для chatID %d: %v", chatID, errSend)
	}
}

// SendViewOrderDetails отображает детали заказа.
// Адаптировано для показа кнопки "Установить стоимость" оператору.
func (bh *BotHandler) SendViewOrderDetails(chatID int64, orderID int, messageIDToEdit int, isOperatorView bool, viewingUser models.User) (tgbotapi.Message, error) {
	log.Printf("BotHandler.SendViewOrderDetails для ChatID %d (UserID %d, Role %s), заказ #%d, операторский просмотр: %v, сообщение для редактирования: %d",
		chatID, viewingUser.ID, viewingUser.Role, orderID, isOperatorView, messageIDToEdit)

	order, err := db.GetOrderByID(orderID)
	if err != nil {
		log.Printf("SendViewOrderDetails: Ошибка загрузки заказа #%d: %v", orderID, err)
		return bh.sendErrorMessageHelper(chatID, messageIDToEdit, "❌ Ошибка загрузки данных заказа.")
	}

	var ephemeralMessagesToStore []int // Сообщения, которые нужно будет удалить при следующем действии

	// Отправка медиа (фото, видео, геолокация)
	for _, photoFileID := range order.Photos {
		if photoFileID == "" {
			continue
		}
		photoMsg := tgbotapi.NewPhoto(chatID, tgbotapi.FileID(photoFileID))
		sentMediaMsg, errPhoto := bh.Deps.BotClient.Send(photoMsg)
		if errPhoto == nil {
			ephemeralMessagesToStore = append(ephemeralMessagesToStore, sentMediaMsg.MessageID)
		} else {
			log.Printf("SendViewOrderDetails: Ошибка отправки фото %s для заказа #%d: %v", photoFileID, orderID, errPhoto)
		}
	}
	for _, videoFileID := range order.Videos {
		if videoFileID == "" {
			continue
		}
		videoMsg := tgbotapi.NewVideo(chatID, tgbotapi.FileID(videoFileID))
		sentMediaMsg, errVideo := bh.Deps.BotClient.Send(videoMsg)
		if errVideo == nil {
			ephemeralMessagesToStore = append(ephemeralMessagesToStore, sentMediaMsg.MessageID)
		} else {
			log.Printf("SendViewOrderDetails: Ошибка отправки видео %s для заказа #%d: %v", videoFileID, orderID, errVideo)
		}
	}
	if order.Latitude != 0 && order.Longitude != 0 {
		locationMsg := tgbotapi.NewLocation(chatID, order.Latitude, order.Longitude)
		sentMediaMsg, errLoc := bh.Deps.BotClient.Send(locationMsg)
		if errLoc == nil {
			ephemeralMessagesToStore = append(ephemeralMessagesToStore, sentMediaMsg.MessageID)
		} else {
			log.Printf("SendViewOrderDetails: Ошибка отправки геолокации для заказа #%d: %v", orderID, errLoc)
		}
	}

	// Формирование текстового описания заказа с помощью новых форматтеров
	var msgText string
	assignedExecutors, _ := db.GetExecutorsByOrderID(orderID)

	if isOperatorView {
		clientUser, _ := db.GetUserByChatID(order.UserChatID)
		title := fmt.Sprintf("ℹ️ *Детали Заказа №%d*", order.ID)
		footer := "Операторский режим просмотра."
		msgText = formatters.FormatOrderDetailsForOperator(order, clientUser, assignedExecutors, title, footer)
	} else {
		msgText = formatters.FormatOrderDetailsForUser(order, assignedExecutors)
	}

	// --- НАЧАЛО ИЗМЕНЕНИЙ В ЛОГИКЕ КЛАВИАТУРЫ ---
	var keyboard tgbotapi.InlineKeyboardMarkup
	var rows [][]tgbotapi.InlineKeyboardButton
	var listCallbackKey string
	var btnToList tgbotapi.InlineKeyboardButton // Вспомогательная переменная для кнопки "К списку"

	isAssignedThisExecutor := false
	isAssignedDriver := false
	thisExecutorIsNotified := true

	if !isOperatorView && (viewingUser.Role == constants.ROLE_DRIVER || viewingUser.Role == constants.ROLE_LOADER) {
		for _, exec := range assignedExecutors {
			if exec.UserID == viewingUser.ID && exec.Role == viewingUser.Role {
				isAssignedThisExecutor = true
				thisExecutorIsNotified = exec.IsNotified
				if viewingUser.Role == constants.ROLE_DRIVER {
					isAssignedDriver = true
				}
				break
			}
		}
	}

	if isOperatorView {
		isAssignedDriverCheckForOp := false
		if viewingUser.Role == constants.ROLE_DRIVER {
			for _, exec := range assignedExecutors {
				if exec.UserID == viewingUser.ID && exec.Role == constants.ROLE_DRIVER {
					isAssignedDriverCheckForOp = true
					break
				}
			}
		}
		switch order.Status {
		case constants.STATUS_NEW, constants.STATUS_AWAITING_COST:
			rows = append(rows, tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("💰 Установить стоимость", fmt.Sprintf("set_cost_%d", order.ID)),
				tgbotapi.NewInlineKeyboardButtonData("✏️ Редактировать", fmt.Sprintf("edit_order_%d", order.ID)),
			))
			rows = append(rows, tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("❌ Отменить заказ", fmt.Sprintf("cancel_order_operator_%d", order.ID)),
				tgbotapi.NewInlineKeyboardButtonData("🚫 Блокировка", fmt.Sprintf("block_user_reason_prompt_%d", order.UserChatID)),
			))
			listCallbackKey = constants.ORDER_LIST_KEY_NEW

		case constants.STATUS_AWAITING_CONFIRMATION:
			rows = append(rows, tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("✏️ Редактировать", fmt.Sprintf("edit_order_%d", order.ID)),
				tgbotapi.NewInlineKeyboardButtonData("❌ Отменить заказ", fmt.Sprintf("cancel_order_operator_%d", order.ID)),
			))
			rows = append(rows, tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("🚫 Блокировка", fmt.Sprintf("block_user_reason_prompt_%d", order.UserChatID)),
			))
			listCallbackKey = constants.ORDER_LIST_KEY_AWAITING_CONFIRMATION

		case constants.STATUS_INPROGRESS:
			var btnCost, btnDone, btnEdit, btnExecs, btnCancel, btnBlock tgbotapi.InlineKeyboardButton
			if (!order.Cost.Valid || order.Cost.Float64 == 0) && utils.IsOperatorOrHigher(viewingUser.Role) {
				btnCost = tgbotapi.NewInlineKeyboardButtonData("💰 Установить стоимость", fmt.Sprintf("set_cost_%d", order.ID))
			}
			if utils.IsOperatorOrHigher(viewingUser.Role) || isAssignedDriverCheckForOp {
				btnDone = tgbotapi.NewInlineKeyboardButtonData("✅ Заказ выполнен", fmt.Sprintf("%s_%d", constants.CALLBACK_PREFIX_MARK_ORDER_DONE, order.ID))
			}
			if utils.IsOperatorOrHigher(viewingUser.Role) {
				btnEdit = tgbotapi.NewInlineKeyboardButtonData("✏️ Редактировать", fmt.Sprintf("edit_order_%d", order.ID))
				btnExecs = tgbotapi.NewInlineKeyboardButtonData("👷 Исполнители", fmt.Sprintf("assign_executors_%d", order.ID))
				btnCancel = tgbotapi.NewInlineKeyboardButtonData("❌ Отменить заказ", fmt.Sprintf("cancel_order_operator_%d", order.ID))
				btnBlock = tgbotapi.NewInlineKeyboardButtonData("🚫 Блокировка", fmt.Sprintf("block_user_reason_prompt_%d", order.UserChatID))
			}
			if btnCost.Text != "" {
				rows = append(rows, tgbotapi.NewInlineKeyboardRow(btnCost))
			}
			if btnExecs.Text != "" {
				rows = append(rows, tgbotapi.NewInlineKeyboardRow(btnExecs))
			}
			if btnDone.Text != "" && btnEdit.Text != "" {
				rows = append(rows, tgbotapi.NewInlineKeyboardRow(btnDone, btnEdit))
			} else if btnDone.Text != "" {
				rows = append(rows, tgbotapi.NewInlineKeyboardRow(btnDone))
			} else if btnEdit.Text != "" {
				rows = append(rows, tgbotapi.NewInlineKeyboardRow(btnEdit))
			}
			if btnCancel.Text != "" && btnBlock.Text != "" {
				rows = append(rows, tgbotapi.NewInlineKeyboardRow(btnCancel, btnBlock))
			} else if btnCancel.Text != "" {
				rows = append(rows, tgbotapi.NewInlineKeyboardRow(btnCancel))
			} else if btnBlock.Text != "" {
				rows = append(rows, tgbotapi.NewInlineKeyboardRow(btnBlock))
			}
			listCallbackKey = constants.ORDER_LIST_KEY_IN_PROGRESS

		case constants.STATUS_COMPLETED:
			if utils.IsOperatorOrHigher(viewingUser.Role) {
				rows = append(rows, tgbotapi.NewInlineKeyboardRow(
					tgbotapi.NewInlineKeyboardButtonData("💲 Изменить итог. стоимость", fmt.Sprintf("%s_%d", constants.CALLBACK_PREFIX_ORDER_SET_FINAL_COST, order.ID)),
					tgbotapi.NewInlineKeyboardButtonData("🚫 Блокировка", fmt.Sprintf("block_user_reason_prompt_%d", order.UserChatID)),
				))
			}
			listCallbackKey = constants.ORDER_LIST_KEY_COMPLETED
		case constants.STATUS_CALCULATED, constants.STATUS_SETTLED:
			if utils.IsOperatorOrHigher(viewingUser.Role) {
				rows = append(rows, tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("🚫 Блокировка", fmt.Sprintf("block_user_reason_prompt_%d", order.UserChatID))))
			}
			listCallbackKey = (map[string]string{constants.STATUS_CALCULATED: constants.STATUS_CALCULATED, constants.STATUS_SETTLED: "manage_orders"})[order.Status]
		case constants.STATUS_CANCELED:
			if utils.IsOperatorOrHigher(viewingUser.Role) {
				rows = append(rows, tgbotapi.NewInlineKeyboardRow(
					tgbotapi.NewInlineKeyboardButtonData("🚫 Блокировка", fmt.Sprintf("block_user_reason_prompt_%d", order.UserChatID)),
					tgbotapi.NewInlineKeyboardButtonData("🔄 Возобновить", fmt.Sprintf("%s_%d", constants.CALLBACK_PREFIX_ORDER_RESUME, order.ID)),
				))
			}
			listCallbackKey = constants.ORDER_LIST_KEY_CANCELED
		default:
			if utils.IsOperatorOrHigher(viewingUser.Role) {
				if order.Status != constants.STATUS_COMPLETED && order.Status != constants.STATUS_CANCELED && order.Status != constants.STATUS_SETTLED && order.Status != constants.STATUS_CALCULATED {
					rows = append(rows, tgbotapi.NewInlineKeyboardRow(
						tgbotapi.NewInlineKeyboardButtonData("✏️ Редактировать", fmt.Sprintf("edit_order_%d", order.ID)),
						tgbotapi.NewInlineKeyboardButtonData("❌ Отменить заказ", fmt.Sprintf("cancel_order_operator_%d", order.ID)),
					))
				}
				rows = append(rows, tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("🚫 Блокировка", fmt.Sprintf("block_user_reason_prompt_%d", order.UserChatID))))
			}
			listCallbackKey = "manage_orders"
		}

		if listDisplayText, textOk := constants.OrderListDisplayMap[listCallbackKey]; textOk {
			if callbackValue, cbOk := constants.OrderListCallbackMap[listCallbackKey]; cbOk {
				btnToList = tgbotapi.NewInlineKeyboardButtonData(listDisplayText, callbackValue)
			}
		} else if listCallbackKey == "manage_orders" {
			btnToList = tgbotapi.NewInlineKeyboardButtonData("📦 Заказы", "manage_orders")
		} else if listCallbackKey == constants.STATUS_CALCULATED {
			btnToList = tgbotapi.NewInlineKeyboardButtonData("📦 К списку заказов", fmt.Sprintf("operator_orders_%s_0", constants.STATUS_CALCULATED))
		}

	} else { // Клиентский просмотр или просмотр исполнителем
		// Кнопки для назначенного водителя
		if isAssignedDriver {
			if order.Status == constants.STATUS_INPROGRESS {
				rows = append(rows, tgbotapi.NewInlineKeyboardRow(
					tgbotapi.NewInlineKeyboardButtonData("✅ Заказ выполнен", fmt.Sprintf("%s_%d", constants.CALLBACK_PREFIX_MARK_ORDER_DONE, order.ID)),
				))
			}
			if order.Status == constants.STATUS_COMPLETED {
				rows = append(rows, tgbotapi.NewInlineKeyboardRow(
					tgbotapi.NewInlineKeyboardButtonData("💲 Изменить итог. стоимость", fmt.Sprintf("%s_%d", constants.CALLBACK_PREFIX_ORDER_SET_FINAL_COST, order.ID)),
				))
			}
		}

		// Кнопки для клиента
		if viewingUser.ChatID == order.UserChatID {
			var clientOrderCostForButton float64
			if order.Cost.Valid {
				clientOrderCostForButton = order.Cost.Float64
			}
			if order.Status == constants.STATUS_DRAFT || (order.Status == constants.STATUS_AWAITING_COST && (!order.Cost.Valid || (order.Cost.Valid && order.Cost.Float64 == 0.0))) {
				rows = append(rows, tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("✏️ Редактировать мой заказ", fmt.Sprintf("edit_order_%d", order.ID))))
			}
			if order.Status == constants.STATUS_AWAITING_CONFIRMATION {
				rows = append(rows, tgbotapi.NewInlineKeyboardRow(
					tgbotapi.NewInlineKeyboardButtonData(fmt.Sprintf("✅ Да, согласен (%.0f ₽)", clientOrderCostForButton), fmt.Sprintf("accept_cost_%d", order.ID)),
					tgbotapi.NewInlineKeyboardButtonData("❌ Отказаться от стоимости", fmt.Sprintf("reject_cost_%d", order.ID)),
				))
			}
			if order.Status == constants.STATUS_DRAFT ||
				(order.Status == constants.STATUS_AWAITING_COST && (!order.Cost.Valid || (order.Cost.Valid && order.Cost.Float64 == 0.0))) ||
				order.Status == constants.STATUS_AWAITING_CONFIRMATION {
				rows = append(rows, tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("❌ Отменить мой заказ", fmt.Sprintf("cancel_order_confirm_%d", order.ID))))
			}
		}

		// Кнопка подтверждения уведомления для любого исполнителя
		if isAssignedThisExecutor && !thisExecutorIsNotified {
			confirmNotificationCallback := fmt.Sprintf("%s_%d_%d", constants.CALLBACK_PREFIX_EXECUTOR_NOTIFIED, order.ID, viewingUser.ID)
			rows = append(rows, tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("✅ Уведомлен о назначении", confirmNotificationCallback),
			))
		}

		btnToList = tgbotapi.NewInlineKeyboardButtonData("📋 Мои заказы", "my_orders_page_0")
	}

	btnMainMenu := tgbotapi.NewInlineKeyboardButtonData("🏢 Главное меню", "back_to_main")
	if btnToList.Text != "" {
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(btnToList, btnMainMenu))
	} else {
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(btnMainMenu))
	}

	keyboard.InlineKeyboard = rows
	// --- КОНЕЦ ИЗМЕНЕНИЙ В ЛОГИКЕ КЛАВИАТУРЫ ---

	var idForMainMessageEdit int
	if len(ephemeralMessagesToStore) > 0 && messageIDToEdit != 0 {
		bh.deleteMessageHelper(chatID, messageIDToEdit)
		idForMainMessageEdit = 0
	} else {
		idForMainMessageEdit = messageIDToEdit
	}

	sentInfoMsg, errSend := bh.sendOrEditMessageHelper(chatID, idForMainMessageEdit, msgText, &keyboard, tgbotapi.ModeMarkdown)
	if errSend != nil {
		if idForMainMessageEdit != 0 {
			sentInfoMsg, errSend = bh.sendOrEditMessageHelper(chatID, 0, msgText, &keyboard, tgbotapi.ModeMarkdown)
		}
		if errSend != nil {
			return tgbotapi.Message{}, errSend
		}
	}

	currentOrderData := bh.Deps.SessionManager.GetTempOrder(chatID)
	currentOrderData.CurrentMessageID = sentInfoMsg.MessageID
	currentOrderData.EphemeralMediaMessageIDs = ephemeralMessagesToStore
	currentOrderData.MediaMessageIDs = []int{sentInfoMsg.MessageID}
	currentOrderData.MediaMessageIDsMap = make(map[string]bool)
	currentOrderData.MediaMessageIDsMap[fmt.Sprintf("%d", sentInfoMsg.MessageID)] = true
	bh.Deps.SessionManager.UpdateTempOrder(chatID, currentOrderData)

	return sentInfoMsg, nil
}

// SendAssignExecutorsMenu отправляет меню назначения исполнителей на заказ.
func (bh *BotHandler) SendAssignExecutorsMenu(chatID int64, orderID int64, messageIDToEdit int) {
	log.Printf("BotHandler.SendAssignExecutorsMenu для chatID %d, заказ #%d, сообщение %d", chatID, orderID, messageIDToEdit)

	// Получаем текущего пользователя для определения его роли и ID
	viewingUser, ok := bh.getUserFromDB(chatID)
	if !ok {
		bh.sendErrorMessageHelper(chatID, messageIDToEdit, "Не удалось получить данные пользователя.")
		return
	}

	tempOrderSession := bh.Deps.SessionManager.GetTempOrder(chatID)
	currentState := bh.Deps.SessionManager.GetState(chatID)

	// Определяем, находится ли пользователь в специальном потоке создания заказа водителем
	isDriverCreatingFlow := currentState == constants.STATE_DRIVER_CREATE_ORDER_FLOW && viewingUser.Role == constants.ROLE_DRIVER

	// Устанавливаем соответствующее состояние для навигации
	if isDriverCreatingFlow {
		// Водитель находится в своем потоке, состояние уже должно быть правильным
	} else if tempOrderSession.OrderAction == "operator_creating_order" {
		bh.Deps.SessionManager.SetState(chatID, constants.STATE_OP_ORDER_ASSIGN_EXEC_MENU)
	} else {
		bh.Deps.SessionManager.SetState(chatID, constants.STATE_IDLE) // Редактирование существующего заказа
	}

	order, err := db.GetOrderByID(int(orderID))
	if err != nil {
		log.Printf("SendAssignExecutorsMenu: Ошибка загрузки заказа #%d: %v", orderID, err)
		_, _ = bh.sendErrorMessageHelper(chatID, messageIDToEdit, "❌ Ошибка загрузки данных заказа.")
		return
	}

	// Если это водитель создает заказ, он должен быть автоматически назначен
	if isDriverCreatingFlow {
		// Проверяем, не назначен ли он уже, чтобы избежать дублирования
		execs, _ := db.GetExecutorsByOrderID(int(orderID))
		alreadyAssigned := false
		for _, exec := range execs {
			if exec.UserID == viewingUser.ID && exec.Role == constants.ROLE_DRIVER {
				alreadyAssigned = true
				break
			}
		}
		if !alreadyAssigned {
			errAssign := db.AssignExecutor(int(orderID), viewingUser.ChatID, constants.ROLE_DRIVER)
			if errAssign != nil {
				log.Printf("SendAssignExecutorsMenu: не удалось автоматически назначить водителя %d на заказ #%d: %v", viewingUser.ID, orderID, errAssign)
				bh.sendErrorMessageHelper(chatID, messageIDToEdit, "Ошибка автоматического назначения вас на заказ.")
				return
			}
			log.Printf("Водитель %d автоматически назначен на создаваемый им заказ #%d", viewingUser.ID, orderID)
		}
	}

	// Получаем обновленный список назначенных исполнителей
	assignedExecutors, err := db.GetExecutorsByOrderID(int(orderID))
	if err != nil {
		log.Printf("SendAssignExecutorsMenu: ошибка получения назначенных исполнителей для заказа #%d: %v", orderID, err)
		_, _ = bh.sendErrorMessageHelper(chatID, messageIDToEdit, "❌ Ошибка загрузки назначенных исполнителей.")
		return
	}

	// Получаем список всех доступных грузчиков
	availableLoaders, _ := db.GetUsersByRole(constants.ROLE_LOADER)

	// Формируем текст сообщения
	msgText := fmt.Sprintf("👷 Назначение исполнителей на Заказ №%d (%s, %s)\n\n",
		orderID, constants.CategoryDisplayMap[order.Category], order.Address)

	var rows [][]tgbotapi.InlineKeyboardButton

	msgText += "*Назначенные исполнители:*\n"
	if len(assignedExecutors) == 0 {
		msgText += "_Нет назначенных исполнителей_\n"
	} else {
		for _, exec := range assignedExecutors {
			execUser, _ := db.GetUserByID(int(exec.UserID))
			displayNameForButton := execUser.FirstName
			if displayNameForButton == "" && execUser.Nickname.Valid {
				displayNameForButton = execUser.Nickname.String
			} else if displayNameForButton == "" {
				displayNameForButton = fmt.Sprintf("ID %d", execUser.ChatID)
			}
			displayNameForText := utils.GetUserDisplayName(execUser)
			roleEmojiForButton := "❓"
			notifiedStatus := "⭕️" // Не уведомлен
			if exec.IsNotified {
				notifiedStatus = "🟢" // Уведомлен
			}
			switch exec.Role {
			case constants.ROLE_DRIVER:
				roleEmojiForButton = "🚚"
			case constants.ROLE_LOADER:
				roleEmojiForButton = "💪"
			}
			msgText += fmt.Sprintf("  - %s %s: %s\n", roleEmojiForButton, notifiedStatus, utils.EscapeTelegramMarkdown(displayNameForText))

			// Водитель, создающий заказ, не может снять себя с него. Оператор - может.
			if !isDriverCreatingFlow || (isDriverCreatingFlow && exec.UserID != viewingUser.ID) {
				rows = append(rows, tgbotapi.NewInlineKeyboardRow(
					tgbotapi.NewInlineKeyboardButtonData(fmt.Sprintf("❌ %s %s %s", notifiedStatus, roleEmojiForButton, utils.StripEmoji(displayNameForButton)), fmt.Sprintf("unassign_executor_%d_%d", orderID, exec.ChatID)),
				))
			}
		}
	}
	msgText += "\n"

	// Логика для отображения доступных водителей (только для операторов)
	if !isDriverCreatingFlow {
		availableDrivers, _ := db.GetUsersByRole(constants.ROLE_DRIVER)
		msgText += "*🚚 Доступные водители для назначения:*\n"
		driversAddedToMenu := 0
		if len(availableDrivers) > 0 {
			for _, driver := range availableDrivers {
				isAssigned := false
				for _, assigned := range assignedExecutors {
					if assigned.UserID == driver.ID && assigned.Role == constants.ROLE_DRIVER {
						isAssigned = true
						break
					}
				}
				if !isAssigned {
					displayNameForButton := driver.FirstName
					if displayNameForButton == "" && driver.Nickname.Valid {
						displayNameForButton = driver.Nickname.String
					} else if displayNameForButton == "" {
						displayNameForButton = fmt.Sprintf("ID %d", driver.ChatID)
					}
					rows = append(rows, tgbotapi.NewInlineKeyboardRow(
						tgbotapi.NewInlineKeyboardButtonData(fmt.Sprintf("✅ 🚚 %s", utils.StripEmoji(displayNameForButton)), fmt.Sprintf("assign_driver_%d_%d", orderID, driver.ChatID)),
					))
					driversAddedToMenu++
				}
			}
		}
		if driversAddedToMenu == 0 && len(availableDrivers) > 0 {
			msgText += "_Все доступные водители уже назначены._\n"
		} else if len(availableDrivers) == 0 {
			msgText += "_Нет доступных водителей в системе._\n"
		}
		msgText += "\n"
	}

	// Логика для отображения доступных грузчиков (для всех)
	msgText += "*💪 Доступные грузчики для назначения:*\n"
	loadersAddedToMenu := 0
	if len(availableLoaders) > 0 {
		for _, loader := range availableLoaders {
			isAssigned := false
			for _, assigned := range assignedExecutors {
				if assigned.UserID == loader.ID && assigned.Role == constants.ROLE_LOADER {
					isAssigned = true
					break
				}
			}
			if !isAssigned {
				displayNameForButton := loader.FirstName
				if displayNameForButton == "" && loader.Nickname.Valid {
					displayNameForButton = loader.Nickname.String
				} else if displayNameForButton == "" {
					displayNameForButton = fmt.Sprintf("ID %d", loader.ChatID)
				}
				rows = append(rows, tgbotapi.NewInlineKeyboardRow(
					tgbotapi.NewInlineKeyboardButtonData(fmt.Sprintf("✅ 💪 %s", utils.StripEmoji(displayNameForButton)), fmt.Sprintf("assign_loader_%d_%d", orderID, loader.ChatID)),
				))
				loadersAddedToMenu++
			}
		}
	}
	if loadersAddedToMenu == 0 && len(availableLoaders) > 0 {
		msgText += "_Все доступные грузчики уже назначены._\n"
	} else if len(availableLoaders) == 0 {
		msgText += "_Нет доступных грузчиков в системе._\n"
	}

	// Кнопки навигации
	isOperatorCreatingRegular := tempOrderSession.OrderAction == "operator_creating_order" && !isDriverCreatingFlow
	if isOperatorCreatingRegular || isDriverCreatingFlow {
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("✅ Далее (к финальному подтверждению)", fmt.Sprintf("%s_%d", constants.CALLBACK_PREFIX_OP_FINALIZE_ORDER_CREATION, orderID))))
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("⬅️ Назад к стоимости", fmt.Sprintf("%s_%d", constants.CALLBACK_PREFIX_OP_CONFIRM_ORDER_SET_COST, orderID))))
	} else { // Редактирование существующего заказа
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("✅ Готово (вернуться к заказу)", fmt.Sprintf("view_order_ops_%d", orderID)),
		))
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("⬅️ Назад к редактированию заказа", fmt.Sprintf("edit_order_%d", orderID)),
		))
	}

	rows = append(rows, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("🏢 Главное меню", "back_to_main"),
	))
	keyboard := tgbotapi.NewInlineKeyboardMarkup(rows...)

	_, errSend := bh.sendOrEditMessageHelper(chatID, messageIDToEdit, msgText, &keyboard, tgbotapi.ModeMarkdown)
	if errSend != nil {
		log.Printf("SendAssignExecutorsMenu: Ошибка отправки для chatID %d: %v", chatID, errSend)
	}
}

// SendClientSelectionMenu (для оператора) предлагает выбрать клиента для создания заказа.
func (bh *BotHandler) SendClientSelectionMenu(chatID int64, messageIDToEdit int) {
	log.Printf("BotHandler.SendClientSelectionMenu для оператора chatID %d, messageIDToEdit: %d", chatID, messageIDToEdit)
	bh.Deps.SessionManager.SetState(chatID, constants.STATE_OPERATOR_SELECT_CLIENT)

	clients, err := db.GetUsersByRole(constants.ROLE_USER)
	if err != nil {
		log.Printf("SendClientSelectionMenu: ошибка получения списка клиентов: %v", err)
		_, _ = bh.sendErrorMessageHelper(chatID, messageIDToEdit, "❌ Ошибка загрузки списка клиентов.")
		return
	}

	var msgText string
	var rows [][]tgbotapi.InlineKeyboardButton

	if len(clients) == 0 {
		msgText = "👥 Клиенты не найдены. Вы можете попросить клиента написать боту для регистрации."
	} else {
		msgText = "👥 Выберите клиента, для которого хотите создать заказ:"
		for _, client := range clients {
			name := utils.GetUserDisplayName(client)
			if len(name) > 50 { // Ограничение длины текста кнопки
				name = name[:47] + "..."
			}
			rows = append(rows, tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData(name, fmt.Sprintf("select_client_%d", client.ChatID)),
			))
		}
	}
	// Кнопка "Назад" должна вести в главное меню оператора, так как это начало нового потока
	rows = append(rows, tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("🔙 Главное меню", "back_to_main")))
	keyboard := tgbotapi.NewInlineKeyboardMarkup(rows...)

	_, errSend := bh.sendOrEditMessageHelper(chatID, messageIDToEdit, msgText, &keyboard, tgbotapi.ModeMarkdown)
	if errSend != nil {
		log.Printf("SendClientSelectionMenu: Ошибка отправки для chatID %d: %v", chatID, errSend)
	}
}
