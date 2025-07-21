package handlers

import (
	"Original/internal/constants"
	"Original/internal/db"
	"Original/internal/models"
	"Original/internal/utils"
	"fmt"
	"log"

	tgbotapi "github.com/OvyFlash/telegram-bot-api"
)

// SendGatewayMenu отправляет пользователю первоначальный выбор: Web App или бот.
// Эту функцию нужно вызывать при старте бота (команда /start).
func (bh *BotHandler) SendGatewayMenu(chatID int64, messageIDToEdit int) {
	log.Printf("BotHandler.SendGatewayMenu для chatID %d, messageIDToEdit: %d", chatID, messageIDToEdit)

	// !!! ВАЖНО: Замените 'https://your-web-app.url' на реальный URL вашего Web App
	webAppURL := "https://xn----ctbinlmxece7i.xn--p1ai/webapp/" // Пример URL

	msgText := "Добро пожаловать! 🚀\n\nВыберите, как вам удобнее продолжить:"

	// --- НАЧАЛО ИЗМЕНЕНИЯ ---
	// Используем конструктор NewInlineKeyboardButtonWebApp, который совместим
	// с более старыми версиями библиотеки.
	// Он принимает текст кнопки и структуру WebAppInfo в качестве аргументов.
	webAppButton := tgbotapi.NewInlineKeyboardButtonWebApp(
		"🌐 Открыть Web App",
		tgbotapi.WebAppInfo{URL: webAppURL},
	)
	// --- КОНЕЦ ИЗМЕНЕНИЯ ---

	// Создаем кнопку для продолжения в боте
	continueInBotButton := tgbotapi.NewInlineKeyboardButtonData("🤖 Продолжить в боте", constants.CALLBACK_CONTINUE_IN_BOT)

	// Собираем клавиатуру
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			webAppButton,
		),
		tgbotapi.NewInlineKeyboardRow(
			continueInBotButton,
		),
	)

	// Отправляем или редактируем сообщение
	_, err := bh.sendOrEditMessageHelper(chatID, messageIDToEdit, msgText, &keyboard, "")
	if err != nil {
		log.Printf("SendGatewayMenu: Ошибка отправки/редактирования меню-шлюза для chatID %d: %v", chatID, err)
	}
}

// SendMainMenu отправляет главное меню пользователю.
func (bh *BotHandler) SendMainMenu(chatID int64, user models.User, messageIDToEdit int) {
	log.Printf("BotHandler.SendMainMenu для chatID %d, messageIDToEdit: %d, роль: %s", chatID, messageIDToEdit, user.Role)

	if messageIDToEdit == 0 && user.MainMenuMessageID != 0 {
		messageIDToEdit = user.MainMenuMessageID
	}

	var rows [][]tgbotapi.InlineKeyboardButton
	log.Printf("SendMainMenu: Для chatID %d, user.FirstName: '[%s]' (длина: %d), user.Role: %s", chatID, user.FirstName, len(user.FirstName), user.Role)
	var greetingName string
	if user.FirstName == "" {
		greetingName = "дорогой друг"
	} else {
		greetingName = utils.EscapeTelegramMarkdown(user.FirstName)
	}
	log.Printf("SendMainMenu: greetingName для chatID %d: '[%s]'", chatID, greetingName)
	// Текст по умолчанию, если роль не определена
	msgText := fmt.Sprintf("👋 Привет, %s! Добро пожаловать! 🚛\nВыберите действие:", greetingName)

	switch user.Role {
	case constants.ROLE_USER:
		msgText = fmt.Sprintf(
			"Привет, %s! 👋\n\n"+
				"Хотите избавиться от мусора или планируете демонтаж? Вы по адресу!\n\n"+
				"Я — ваш персональный помощник от компании «<b>СЕРВИС-КРЫМ</b>». Помогу вам рассчитать предварительную стоимость и оформить заявку всего за пару минут.\n\n"+
				"Начнем?\n\n"+
				"👇 <b>Выберите, что вас интересует:</b>",
			utils.EscapeTelegramMarkdown(greetingName),
		)

		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🗑️ Вывоз мусора", "category_waste"),
			tgbotapi.NewInlineKeyboardButtonData("🛠️ Демонтаж", "category_demolition"),
		))
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🧱 Стройматериалы - скоро + бонус! 🎁", "materials_soon"),
		))
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📋 Мои заказы", "my_orders_page_0"),
			tgbotapi.NewInlineKeyboardButtonData("👥 Пригласить друга", "invite_friend"),
		))
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📞 Связь с оператором", "contact_operator"),
		))

	case constants.ROLE_OPERATOR:
		msgText = fmt.Sprintf("👋 Привет, %s! Добро пожаловать в панель управления! 🚛\nВыберите действие:", greetingName)
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🆕 Создать заказ", constants.CALLBACK_PREFIX_OP_CREATE_NEW_ORDER),
		))
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📦 Заказы", "manage_orders"),
			tgbotapi.NewInlineKeyboardButtonData("💬 Связь с клиентами", "client_chats"),
		))
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🚫 Раз/блокировки", "block_user_menu"),
		))

	case constants.ROLE_MAINOPERATOR:
		msgText = fmt.Sprintf("👋 Привет, %s! Добро пожаловать в панель управления! 🚛\nВыберите действие:", greetingName)
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🆕 Создать заказ", constants.CALLBACK_PREFIX_OP_CREATE_NEW_ORDER),
		))
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📦 Заказы", "manage_orders"),
			tgbotapi.NewInlineKeyboardButtonData("💬 Связь с клиентами", "client_chats"),
		))
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🚫 Раз/блокировки", "block_user_menu"),
		))
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("👷 Сотрудники", "staff_menu"),
			tgbotapi.NewInlineKeyboardButtonData("📊 Статистика", "stats_menu"),
		))
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("💰 Money", constants.CALLBACK_PREFIX_OWNER_CASH_MANAGEMENT_MAIN),
			tgbotapi.NewInlineKeyboardButtonData("📑 Отправить Excel", "send_excel_menu"),
		))

	case constants.ROLE_OWNER:
		msgText = fmt.Sprintf("👑 Владелец %s, добро пожаловать в панель управления!", greetingName)
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🆕 Создать заказ", constants.CALLBACK_PREFIX_OP_CREATE_NEW_ORDER),
			tgbotapi.NewInlineKeyboardButtonData("📦 Заказы", "manage_orders"),
		))
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("💬 Связь с клиентами", "client_chats"),
			tgbotapi.NewInlineKeyboardButtonData("🚫 Раз/блокировки", "block_user_menu"),
		))
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("👷 Сотрудники", "staff_menu"),
			tgbotapi.NewInlineKeyboardButtonData("💰 Money", constants.CALLBACK_PREFIX_OWNER_CASH_MANAGEMENT_MAIN),
		))
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("💸 Выплаты сотрудникам", constants.CALLBACK_PREFIX_OWNER_STAFF_PAYOUT),
		))
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📊 Статистика", "stats_menu"),
			tgbotapi.NewInlineKeyboardButtonData("📑 Отправить Excel", "send_excel_menu"),
		))

	case constants.ROLE_DRIVER:
		msgText = fmt.Sprintf("🚚 Водитель %s, выберите действие:", utils.EscapeTelegramMarkdown(user.FirstName))
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📋 Мои заказы", "my_orders_page_0"),
			tgbotapi.NewInlineKeyboardButtonData("🆕 Создать заказ", constants.CALLBACK_PREFIX_DRIVER_CREATE_ORDER),
		))
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("💰 Моя зарплата", constants.CALLBACK_PREFIX_MY_SALARY),
			tgbotapi.NewInlineKeyboardButtonData("🧾 Расчет по заказам", constants.CALLBACK_PREFIX_DRIVER_SETTLEMENT),
		))
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📞 Связь с оператором", "contact_operator"),
		))

	case constants.ROLE_LOADER:
		msgText = fmt.Sprintf("💪 Грузчик %s, выберите действие:", utils.EscapeTelegramMarkdown(user.FirstName))
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📋 Мои заказы", "my_orders_page_0"),
		))
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("💰 Моя зарплата", constants.CALLBACK_PREFIX_MY_SALARY),
		))
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📞 Связь с оператором", "contact_operator"),
		))

	default:
		log.Printf("SendMainMenu: неизвестная роль '%s' для chatID %d", user.Role, chatID)
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📞 Связь с оператором", "contact_operator"),
		))
		msgText = fmt.Sprintf("👋 Привет, %s! Пожалуйста, свяжитесь с оператором для уточнения ваших возможностей в системе.", greetingName)
	}
	keyboard := tgbotapi.NewInlineKeyboardMarkup(rows...)

	parseMode := ""
	if user.Role == constants.ROLE_USER {
		parseMode = tgbotapi.ModeHTML
	}

	sentMsg, errSend := bh.sendOrEditMessageHelper(chatID, messageIDToEdit, msgText, &keyboard, parseMode)
	if errSend != nil {
		log.Printf("SendMainMenu: Ошибка отправки/редактирования главного меню для chatID %d: %v", chatID, errSend)
		if messageIDToEdit != 0 {
			log.Printf("SendMainMenu: Попытка отправить новое главное меню для chatID %d из-за ошибки редактирования.", chatID)
			sentMsg, errSend = bh.sendOrEditMessageHelper(chatID, 0, msgText, &keyboard, parseMode)
			if errSend != nil {
				log.Printf("SendMainMenu: КРИТИЧЕСКАЯ ОШИБКА отправки нового главного меню для chatID %d: %v", chatID, errSend)
				return
			}
		} else {
			return
		}
	}

	if sentMsg.MessageID != 0 && (user.MainMenuMessageID != sentMsg.MessageID || messageIDToEdit == 0) {
		errDbUpdate := db.UpdateUserMainMenuMessageID(chatID, sentMsg.MessageID)
		if errDbUpdate == nil {
			log.Printf("SendMainMenu: main_menu_message_id %d сохранен для chatID %d", sentMsg.MessageID, chatID)
		} else {
			log.Printf("SendMainMenu: Ошибка сохранения main_menu_message_id %d для chatID %d: %v", sentMsg.MessageID, chatID, errDbUpdate)
		}
	}
}
