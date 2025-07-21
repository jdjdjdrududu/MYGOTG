package handlers

import (
	"database/sql"
	"fmt"
	"log"
	"os" // Для работы с файлами Excel / For working with Excel files
	// "strconv" // Not used directly here
	// "strings" // Used in utils
	"time"

	tgbotapi "github.com/OvyFlash/telegram-bot-api"
	"github.com/xuri/excelize/v2" // Для генерации Excel / For Excel generation

	"Original/internal/constants"
	"Original/internal/db"
	"Original/internal/models"
	// "Original/internal/session" // Access via bh.Deps
	"Original/internal/utils"
)

// SendExcelMenu отправляет меню выбора типа Excel-отчета.
// SendExcelMenu sends the Excel report type selection menu.
func (bh *BotHandler) SendExcelMenu(chatID int64, messageIDToEdit int) {
	log.Printf("BotHandler.SendExcelMenu для chatID %d, messageIDToEdit: %d", chatID, messageIDToEdit)
	// Состояние можно не менять, так как действие выполняется сразу
	// State may not need to be changed as action is performed immediately
	// bh.Deps.SessionManager.SetState(chatID, constants.STATE_ADMIN_ACTION)

	// Права доступа проверяются в callback_handler перед вызовом этой функции
	// Access rights are checked in callback_handler before calling this function

	msgText := "📑 Выберите тип Excel-отчета для генерации:"
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📋 Заказы (за сегодня)", "excel_generate_orders"),
			tgbotapi.NewInlineKeyboardButtonData("👥 Рефералы (за сегодня)", "excel_generate_referrals"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("💰 Зарплаты (за сегодня)", "excel_generate_salaries"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔙 Назад в меню статистики", "stats_menu"), // Или back_to_main, если вызывается не из статистики
			// Or back_to_main if not called from statistics
		),
	)
	_, err := bh.sendOrEditMessageHelper(chatID, messageIDToEdit, msgText, &keyboard, tgbotapi.ModeMarkdown)
	if err != nil {
		log.Printf("SendExcelMenu: Ошибка для chatID %d: %v", chatID, err)
	}
}

// SendExcelFile отправляет сгенерированный Excel-файл пользователю.
// SendExcelFile sends the generated Excel file to the user.
func (bh *BotHandler) SendExcelFile(chatID int64, filePath string, caption string) {
	log.Printf("BotHandler.SendExcelFile: отправка файла %s для chatID %d", filePath, chatID)

	if filePath == "" {
		bh.sendErrorMessageHelper(chatID, 0, "❌ Не удалось сгенерировать Excel-файл.") // Отправляем новое сообщение об ошибке / Send new error message
		return
	}

	doc := tgbotapi.NewDocument(chatID, tgbotapi.FilePath(filePath))
	doc.Caption = caption
	_, err := bh.Deps.BotClient.Send(doc) // Используем прямой Send из BotClient, так как это файл / Use direct Send from BotClient as it's a file

	if err != nil {
		log.Printf("SendExcelFile: Ошибка отправки Excel-файла %s для chatID %d: %v", filePath, chatID, err)
		bh.sendErrorMessageHelper(chatID, 0, "❌ Ошибка при отправке Excel-файла.")
	}

	// Удаляем временный файл после отправки / Delete temporary file after sending
	errRemove := os.Remove(filePath)
	if errRemove != nil {
		log.Printf("SendExcelFile: Ошибка удаления временного Excel-файла %s: %v", filePath, errRemove)
	}
}

// generateAndSendOrdersExcel генерирует и отправляет Excel отчет по заказам.
// messageIDToEdit - ID сообщения с кнопками выбора Excel, которое нужно удалить после отправки файла.
// generateAndSendOrdersExcel generates and sends an Excel report on orders.
// messageIDToEdit - ID of the message with Excel selection buttons, to be deleted after sending the file.
func (bh *BotHandler) generateAndSendOrdersExcel(chatID int64, messageIDToEdit int) {
	rows, err := db.GetOrdersForExcel() // За сегодня / For today
	if err != nil {
		log.Printf("generateAndSendOrdersExcel: Ошибка получения данных заказов из БД: %v", err)
		bh.sendErrorMessageHelper(chatID, messageIDToEdit, "❌ Ошибка при получении данных для Excel отчета по заказам.")
		return
	}
	defer rows.Close()

	f := excelize.NewFile()
	sheetName := "Заказы"
	index, _ := f.NewSheet(sheetName) // Игнорируем ошибку, если лист уже существует (NewFile создает Sheet1) / Ignore error if sheet already exists (NewFile creates Sheet1)
	f.DeleteSheet("Sheet1")           // Удаляем стандартный лист / Delete default sheet
	f.SetActiveSheet(index)

	headers := []string{"ID Заказа", "Клиент Имя", "Клиент Фамилия", "Клиент Никнейм", "Категория", "Подкатегория", "Дата заказа", "Время заказа", "Телефон клиента", "Адрес", "Статус", "Стоимость"}
	for i, header := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		f.SetCellValue(sheetName, cell, header)
	}

	rowIndex := 2
	for rows.Next() {
		var id int
		var firstName, lastName, category, subcategory, phone, address, status string
		var nickname sql.NullString
		var date time.Time // db.GetOrdersForExcel должен возвращать time.Time для даты / db.GetOrdersForExcel should return time.Time for date
		var timeStr sql.NullString
		var cost sql.NullFloat64

		// Порядок сканирования должен соответствовать SELECT в db.GetOrdersForExcel()
		// Scan order must match SELECT in db.GetOrdersForExcel()
		if errScan := rows.Scan(&id, &firstName, &lastName, &nickname, &category, &subcategory, &date, &timeStr, &phone, &address, &status, &cost); errScan != nil {
			log.Printf("generateAndSendOrdersExcel: Ошибка сканирования строки заказа: %v", errScan)
			continue
		}

		f.SetCellValue(sheetName, fmt.Sprintf("A%d", rowIndex), id)
		f.SetCellValue(sheetName, fmt.Sprintf("B%d", rowIndex), firstName)
		f.SetCellValue(sheetName, fmt.Sprintf("C%d", rowIndex), lastName)
		if nickname.Valid {
			f.SetCellValue(sheetName, fmt.Sprintf("D%d", rowIndex), nickname.String)
		}
		f.SetCellValue(sheetName, fmt.Sprintf("E%d", rowIndex), constants.CategoryDisplayMap[category])
		f.SetCellValue(sheetName, fmt.Sprintf("F%d", rowIndex), utils.GetDisplaySubcategory(models.Order{Category: category, Subcategory: subcategory}))
		f.SetCellValue(sheetName, fmt.Sprintf("G%d", rowIndex), date.Format("02.01.2006")) // Форматируем дату / Format date
		if timeStr.Valid {
			f.SetCellValue(sheetName, fmt.Sprintf("H%d", rowIndex), timeStr.String)
		} else {
			f.SetCellValue(sheetName, fmt.Sprintf("H%d", rowIndex), "в ближайшее время")
		}
		f.SetCellValue(sheetName, fmt.Sprintf("I%d", rowIndex), phone)
		f.SetCellValue(sheetName, fmt.Sprintf("J%d", rowIndex), address)
		f.SetCellValue(sheetName, fmt.Sprintf("K%d", rowIndex), constants.StatusDisplayMap[status])
		if cost.Valid {
			f.SetCellValue(sheetName, fmt.Sprintf("L%d", rowIndex), cost.Float64)
		} else {
			f.SetCellValue(sheetName, fmt.Sprintf("L%d", rowIndex), 0.0)
		}
		rowIndex++
	}
	if err = rows.Err(); err != nil {
		log.Printf("generateAndSendOrdersExcel: Ошибка после итерации по заказам: %v", err)
		bh.sendErrorMessageHelper(chatID, messageIDToEdit, "❌ Ошибка при обработке данных заказов для Excel.")
		return
	}

	filePath := fmt.Sprintf("orders_report_%s.xlsx", time.Now().Format("20060102_150405"))
	if errSave := f.SaveAs(filePath); errSave != nil {
		log.Printf("generateAndSendOrdersExcel: Ошибка сохранения Excel файла: %v", errSave)
		bh.sendErrorMessageHelper(chatID, messageIDToEdit, "❌ Ошибка при создании Excel файла.")
		return
	}

	bh.SendExcelFile(chatID, filePath, fmt.Sprintf("Отчет по заказам за %s", time.Now().Format("02.01.2006")))
	if messageIDToEdit != 0 { // Удаляем сообщение с кнопками выбора Excel, если оно было / Delete message with Excel selection buttons if it existed
		bh.deleteMessageHelper(chatID, messageIDToEdit)
	}
}

// generateAndSendReferralsExcel генерирует и отправляет Excel отчет по рефералам.
// generateAndSendReferralsExcel generates and sends an Excel report on referrals.
func (bh *BotHandler) generateAndSendReferralsExcel(chatID int64, messageIDToEdit int) {
	rows, err := db.GetReferralsForExcel() // За сегодня / For today
	if err != nil {
		log.Printf("generateAndSendReferralsExcel: Ошибка получения данных рефералов: %v", err)
		bh.sendErrorMessageHelper(chatID, messageIDToEdit, "❌ Ошибка при получении данных для Excel отчета по рефералам.")
		return
	}
	defer rows.Close()

	f := excelize.NewFile()
	sheetName := "Рефералы"
	index, _ := f.NewSheet(sheetName)
	f.DeleteSheet("Sheet1")
	f.SetActiveSheet(index)

	headers := []string{"Пригласивший Имя", "Пригласивший Фамилия", "Приглашенный Имя", "Приглашенный Фамилия", "Сумма Бонуса", "Дата Регистрации Реферала", "Статус Выплаты"}
	for i, header := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		f.SetCellValue(sheetName, cell, header)
	}
	rowIndex := 2
	for rows.Next() {
		var inviterFirstName, inviterLastName, inviteeFirstName, inviteeLastName string
		var amount float64
		var createdAt time.Time
		var paidOut bool // Добавлено для статуса выплаты / Added for payout status

		// Порядок сканирования должен соответствовать SELECT в db.GetReferralsForExcel()
		// Scan order must match SELECT in db.GetReferralsForExcel()
		if errScan := rows.Scan(&inviterFirstName, &inviterLastName, &inviteeFirstName, &inviteeLastName, &amount, &createdAt, &paidOut); errScan != nil {
			log.Printf("generateAndSendReferralsExcel: Ошибка сканирования строки реферала: %v", errScan)
			continue
		}
		f.SetCellValue(sheetName, fmt.Sprintf("A%d", rowIndex), inviterFirstName)
		f.SetCellValue(sheetName, fmt.Sprintf("B%d", rowIndex), inviterLastName)
		f.SetCellValue(sheetName, fmt.Sprintf("C%d", rowIndex), inviteeFirstName)
		f.SetCellValue(sheetName, fmt.Sprintf("D%d", rowIndex), inviteeLastName)
		f.SetCellValue(sheetName, fmt.Sprintf("E%d", rowIndex), amount)
		f.SetCellValue(sheetName, fmt.Sprintf("F%d", rowIndex), createdAt.Format("02.01.2006 15:04"))
		payoutStatusText := "Не выплачено"
		if paidOut {
			payoutStatusText = "Выплачено"
		}
		f.SetCellValue(sheetName, fmt.Sprintf("G%d", rowIndex), payoutStatusText)
		rowIndex++
	}
	if err = rows.Err(); err != nil {
		log.Printf("generateAndSendReferralsExcel: Ошибка после итерации по рефералам: %v", err)
		bh.sendErrorMessageHelper(chatID, messageIDToEdit, "❌ Ошибка при обработке данных рефералов для Excel.")
		return
	}
	filePath := fmt.Sprintf("referrals_report_%s.xlsx", time.Now().Format("20060102_150405"))
	if errSave := f.SaveAs(filePath); errSave != nil {
		log.Printf("generateAndSendReferralsExcel: Ошибка сохранения Excel файла: %v", errSave)
		bh.sendErrorMessageHelper(chatID, messageIDToEdit, "❌ Ошибка при создании Excel файла по рефералам.")
		return
	}
	bh.SendExcelFile(chatID, filePath, fmt.Sprintf("Отчет по рефералам за %s", time.Now().Format("02.01.2006")))
	if messageIDToEdit != 0 {
		bh.deleteMessageHelper(chatID, messageIDToEdit)
	}
}

// generateAndSendSalariesExcel генерирует и отправляет Excel отчет по зарплатам.
// generateAndSendSalariesExcel generates and sends an Excel report on salaries.
func (bh *BotHandler) generateAndSendSalariesExcel(chatID int64, messageIDToEdit int) {
	rows, err := db.GetSalariesForExcel() // Использует обновленную db.GetSalariesForExcel / Uses updated db.GetSalariesForExcel
	if err != nil {
		log.Printf("generateAndSendSalariesExcel: Ошибка получения данных о зарплатах: %v", err)
		bh.sendErrorMessageHelper(chatID, messageIDToEdit, "❌ Ошибка при получении данных для Excel отчета по зарплатам.")
		return
	}
	defer rows.Close()

	f := excelize.NewFile()
	sheetName := "Зарплаты"
	index, _ := f.NewSheet(sheetName)
	f.DeleteSheet("Sheet1")
	f.SetActiveSheet(index)

	headers := []string{"Сотрудник Имя", "Сотрудник Фамилия", "Позывной", "Роль", "Тип ЗП", "Сумма ЗП", "ID Заказа", "Дата Заказа", "Дата Расчета/Начисления", "Карта Сотрудника"}
	for i, header := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		f.SetCellValue(sheetName, cell, header)
	}
	rowIndex := 2
	for rows.Next() {
		var firstName, lastName, role, salaryType string
		var nickname, encryptedCardNumber sql.NullString // encryptedCardNumber для зашифрованной карты / encryptedCardNumber for encrypted card
		var salaryAmount sql.NullFloat64
		var orderID sql.NullInt64
		var orderDate, calculationOrPayoutDate sql.NullTime

		// Порядок и типы полей должны соответствовать SELECT в db.GetSalariesForExcel()
		// Scan order and types must match SELECT in db.GetSalariesForExcel()
		if errScan := rows.Scan(&firstName, &lastName, &nickname, &role, &salaryAmount, &orderID, &orderDate, &calculationOrPayoutDate, &salaryType, &encryptedCardNumber); errScan != nil {
			log.Printf("generateAndSendSalariesExcel: Ошибка сканирования строки зарплаты: %v", errScan)
			continue
		}
		f.SetCellValue(sheetName, fmt.Sprintf("A%d", rowIndex), firstName)
		f.SetCellValue(sheetName, fmt.Sprintf("B%d", rowIndex), lastName)
		if nickname.Valid {
			f.SetCellValue(sheetName, fmt.Sprintf("C%d", rowIndex), nickname.String)
		}
		f.SetCellValue(sheetName, fmt.Sprintf("D%d", rowIndex), utils.GetRoleDisplayName(role))
		f.SetCellValue(sheetName, fmt.Sprintf("E%d", rowIndex), salaryType) // Тип ЗП (driver_share, loader_salary) / Salary type

		if salaryAmount.Valid {
			f.SetCellValue(sheetName, fmt.Sprintf("F%d", rowIndex), salaryAmount.Float64)
		} else {
			f.SetCellValue(sheetName, fmt.Sprintf("F%d", rowIndex), 0.0)
		}
		if orderID.Valid {
			f.SetCellValue(sheetName, fmt.Sprintf("G%d", rowIndex), orderID.Int64)
		}
		if orderDate.Valid {
			f.SetCellValue(sheetName, fmt.Sprintf("H%d", rowIndex), orderDate.Time.Format("02.01.2006"))
		}
		if calculationOrPayoutDate.Valid {
			f.SetCellValue(sheetName, fmt.Sprintf("I%d", rowIndex), calculationOrPayoutDate.Time.Format("02.01.2006"))
		}
		if encryptedCardNumber.Valid && encryptedCardNumber.String != "" {
			decryptedCard, errDecrypt := utils.DecryptCardNumber(encryptedCardNumber.String)
			if errDecrypt == nil {
				f.SetCellValue(sheetName, fmt.Sprintf("J%d", rowIndex), decryptedCard)
			} else {
				log.Printf("generateAndSendSalariesExcel: Ошибка дешифрования карты для отчета: %v", errDecrypt)
				f.SetCellValue(sheetName, fmt.Sprintf("J%d", rowIndex), "[ошибка дешифр.]")
			}
		}

		rowIndex++
	}
	if err = rows.Err(); err != nil {
		log.Printf("generateAndSendSalariesExcel: Ошибка после итерации по зарплатам: %v", err)
		bh.sendErrorMessageHelper(chatID, messageIDToEdit, "❌ Ошибка при обработке данных зарплат для Excel.")
		return
	}
	filePath := fmt.Sprintf("salaries_report_%s.xlsx", time.Now().Format("20060102_150405"))
	if errSave := f.SaveAs(filePath); errSave != nil {
		log.Printf("generateAndSendSalariesExcel: Ошибка сохранения Excel файла: %v", errSave)
		bh.sendErrorMessageHelper(chatID, messageIDToEdit, "❌ Ошибка при создании Excel файла по зарплатам.")
		return
	}
	bh.SendExcelFile(chatID, filePath, fmt.Sprintf("Отчет по зарплатам за %s", time.Now().Format("02.01.2006")))
	if messageIDToEdit != 0 {
		bh.deleteMessageHelper(chatID, messageIDToEdit)
	}
}

// --- Меню блокировки пользователей (Block/Unblock) ---
// --- User Blocking Menu (Block/Unblock) ---
// (Эти функции остаются без изменений, если они не затрагивают новую логику зарплат/выплат)
// (These functions remain unchanged if they do not affect the new salary/payout logic)

// SendBlockUserMenu отправляет меню выбора действия (блокировать/разблокировать).
// SendBlockUserMenu sends the action selection menu (block/unblock).
func (bh *BotHandler) SendBlockUserMenu(chatID int64, messageIDToEdit int) {
	log.Printf("BotHandler.SendBlockUserMenu для chatID %d, messageIDToEdit: %d", chatID, messageIDToEdit)
	bh.Deps.SessionManager.SetState(chatID, constants.STATE_BLOCK_USER_MENU)

	msgText := "🚫 Управление блокировкой пользователей:\n\nВыберите действие:"
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔒 Заблокировать пользователя", "block_user_list_prompt"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔓 Разблокировать пользователя", "unblock_user_list_prompt"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔙 Главное меню", "back_to_main"),
		),
	)
	_, err := bh.sendOrEditMessageHelper(chatID, messageIDToEdit, msgText, &keyboard, tgbotapi.ModeMarkdown)
	if err != nil {
		log.Printf("SendBlockUserMenu: Ошибка для chatID %d: %v", chatID, err)
	}
}

// SendUserListForBlocking отправляет список пользователей для выбора кого заблокировать.
// SendUserListForBlocking sends a list of users to select whom to block.
func (bh *BotHandler) SendUserListForBlocking(chatID int64, messageIDToEdit int) {
	log.Printf("BotHandler.SendUserListForBlocking для chatID %d, messageIDToEdit: %d", chatID, messageIDToEdit)
	bh.Deps.SessionManager.SetState(chatID, constants.STATE_BLOCK_USER_SELECT)

	users, err := db.GetUsersForBlocking()
	if err != nil {
		bh.sendErrorMessageHelper(chatID, messageIDToEdit, "❌ Ошибка загрузки списка пользователей.")
		return
	}

	msgText := "🔒 Выберите пользователя для блокировки:"
	var rows [][]tgbotapi.InlineKeyboardButton
	if len(users) == 0 {
		msgText += "\n\nНет пользователей для блокировки (роль 'user', не заблокированы)."
	} else {
		for _, u := range users {
			displayName := utils.GetUserDisplayName(u)
			if len(displayName) > 50 {
				displayName = displayName[:47] + "..."
			}
			rows = append(rows, tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData(displayName, fmt.Sprintf("block_user_info_%d", u.ChatID)),
			))
		}
	}
	rows = append(rows, tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("🔙 Назад к упр. блокировками", "block_user_menu")))
	keyboard := tgbotapi.NewInlineKeyboardMarkup(rows...)

	_, errSend := bh.sendOrEditMessageHelper(chatID, messageIDToEdit, msgText, &keyboard, "")
	if errSend != nil {
		log.Printf("SendUserListForBlocking: Ошибка для chatID %d: %v", chatID, errSend)
	}
}

// SendBlockUserInfo показывает инфо перед блокировкой.
// SendBlockUserInfo shows info before blocking.
func (bh *BotHandler) SendBlockUserInfo(chatID int64, targetChatID int64, messageIDToEdit int) {
	log.Printf("BotHandler.SendBlockUserInfo для chatID %d, цель: %d, messageIDToEdit: %d", chatID, targetChatID, messageIDToEdit)
	bh.Deps.SessionManager.SetState(chatID, constants.STATE_BLOCK_USER_CONFIRM_INFO)

	targetUser, err := db.GetUserByChatID(targetChatID)
	if err != nil {
		bh.sendErrorMessageHelper(chatID, messageIDToEdit, "❌ Ошибка загрузки данных пользователя.")
		return
	}
	if targetUser.IsBlocked {
		bh.sendInfoMessage(chatID, messageIDToEdit, fmt.Sprintf("ℹ️ Пользователь %s %s уже заблокирован.", targetUser.FirstName, targetUser.LastName), "block_user_list_prompt")
		return
	}
	if targetUser.Role != constants.ROLE_USER { // Проверка, что это обычный пользователь / Check if it's a regular user
		bh.sendInfoMessage(chatID, messageIDToEdit, fmt.Sprintf("🚫 Нельзя заблокировать сотрудника (%s) через это меню. Используйте управление штатом.", utils.GetRoleDisplayName(targetUser.Role)), "block_user_list_prompt")
		return
	}

	phone := "не указан"
	if targetUser.Phone.Valid {
		phone = utils.FormatPhoneNumber(targetUser.Phone.String)
	}
	nickname := "не указан"
	if targetUser.Nickname.Valid {
		nickname = targetUser.Nickname.String
	}

	msgText := fmt.Sprintf(
		"👤 Пользователь для блокировки: *%s %s*\n"+
			"Никнейм: *%s*\n"+
			"Telegram ChatID: `%d`\n"+
			"Телефон: *%s*\n"+
			"Текущая роль: *%s*\n\n"+
			"Действительно заблокировать этого пользователя?",
		utils.EscapeTelegramMarkdown(targetUser.FirstName), utils.EscapeTelegramMarkdown(targetUser.LastName),
		utils.EscapeTelegramMarkdown(nickname), targetUser.ChatID,
		utils.EscapeTelegramMarkdown(phone), utils.EscapeTelegramMarkdown(utils.GetRoleDisplayName(targetUser.Role)),
	)
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔒 Да, заблокировать", fmt.Sprintf("block_user_reason_prompt_%d", targetChatID)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("⬅️ Нет, назад к списку", "block_user_list_prompt"),
		),
	)
	_, errSend := bh.sendOrEditMessageHelper(chatID, messageIDToEdit, msgText, &keyboard, tgbotapi.ModeMarkdown)
	if errSend != nil {
		log.Printf("SendBlockUserInfo: Ошибка для chatID %d: %v", chatID, errSend)
	}
}

// SendBlockReasonInput запрашивает причину блокировки пользователя (не сотрудника).
// SendBlockReasonInput prompts for the reason for blocking a user (not staff).
func (bh *BotHandler) SendBlockReasonInput(chatID int64, targetChatID int64, messageIDToEdit int) {
	log.Printf("BotHandler.SendBlockReasonInput для chatID %d, цель: %d, messageIDToEdit: %d", chatID, targetChatID, messageIDToEdit)
	bh.Deps.SessionManager.SetState(chatID, constants.STATE_BLOCK_REASON)

	tempData := bh.Deps.SessionManager.GetTempOrder(chatID)
	tempData.BlockTargetChatID = targetChatID // Сохраняем ID пользователя для блокировки / Save user ID for blocking
	tempData.CurrentMessageID = messageIDToEdit
	bh.Deps.SessionManager.UpdateTempOrder(chatID, tempData)

	msgText := fmt.Sprintf("🚫 Укажите причину блокировки пользователя (ChatID: `%d`):", targetChatID)
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("⬅️ Назад к информации о пользователе", fmt.Sprintf("block_user_info_%d", targetChatID)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🚫 Отменить блокировку", "block_user_list_prompt"), // Возврат к списку / Return to list
		),
	)
	_, err := bh.sendOrEditMessageHelper(chatID, messageIDToEdit, msgText, &keyboard, tgbotapi.ModeMarkdown)
	if err != nil {
		log.Printf("SendBlockReasonInput: Ошибка для chatID %d: %v", chatID, err)
	}
}

// SendUserListForUnblocking отправляет список заблокированных пользователей для выбора.
// SendUserListForUnblocking sends a list of blocked users for selection.
func (bh *BotHandler) SendUserListForUnblocking(chatID int64, messageIDToEdit int) {
	log.Printf("BotHandler.SendUserListForUnblocking для chatID %d, messageIDToEdit: %d", chatID, messageIDToEdit)
	bh.Deps.SessionManager.SetState(chatID, constants.STATE_UNBLOCK_USER_SELECT)

	blockedUsers, err := db.GetBlockedUsers()
	if err != nil {
		bh.sendErrorMessageHelper(chatID, messageIDToEdit, "❌ Ошибка загрузки списка заблокированных пользователей.")
		return
	}
	msgText := "🔓 Выберите пользователя для разблокировки:"
	var rows [][]tgbotapi.InlineKeyboardButton
	if len(blockedUsers) == 0 {
		msgText += "\n\nНет заблокированных пользователей."
	} else {
		for _, u := range blockedUsers {
			displayName := utils.GetUserDisplayName(u)
			reason := "не указана"
			if u.BlockReason.Valid {
				reason = u.BlockReason.String
			}
			if len(reason) > 20 {
				reason = reason[:17] + "..."
			}
			dateStr := ""
			if u.BlockDate.Valid {
				dateStr = u.BlockDate.Time.Format("02.01.06")
			}

			buttonText := fmt.Sprintf("%s (Забл: %s, Причина: %s)", displayName, dateStr, reason)
			if len(buttonText) > 60 {
				buttonText = buttonText[:57] + "..."
			}

			rows = append(rows, tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData(buttonText, fmt.Sprintf("unblock_user_info_%d", u.ChatID)),
			))
		}
	}
	rows = append(rows, tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("🔙 Назад к упр. блокировками", "block_user_menu")))
	keyboard := tgbotapi.NewInlineKeyboardMarkup(rows...)

	_, errSend := bh.sendOrEditMessageHelper(chatID, messageIDToEdit, msgText, &keyboard, "")
	if errSend != nil {
		log.Printf("SendUserListForUnblocking: Ошибка для chatID %d: %v", chatID, errSend)
	}
}

// SendUnblockUserInfo показывает информацию о заблокированном пользователе перед разблокировкой.
// SendUnblockUserInfo shows information about a blocked user before unblocking.
func (bh *BotHandler) SendUnblockUserInfo(chatID int64, targetChatID int64, messageIDToEdit int) {
	log.Printf("BotHandler.SendUnblockUserInfo для chatID %d, цель: %d, messageIDToEdit: %d", chatID, targetChatID, messageIDToEdit)
	bh.Deps.SessionManager.SetState(chatID, constants.STATE_UNBLOCK_USER_CONFIRM_INFO)

	targetUser, err := db.GetUserByChatID(targetChatID)
	if err != nil {
		bh.sendErrorMessageHelper(chatID, messageIDToEdit, "❌ Ошибка загрузки данных пользователя.")
		return
	}
	if !targetUser.IsBlocked {
		bh.sendInfoMessage(chatID, messageIDToEdit, fmt.Sprintf("ℹ️ Пользователь %s %s не заблокирован.", targetUser.FirstName, targetUser.LastName), "unblock_user_list_prompt")
		return
	}

	phone := "не указан"
	if targetUser.Phone.Valid {
		phone = utils.FormatPhoneNumber(targetUser.Phone.String)
	}
	nickname := "не указан"
	if targetUser.Nickname.Valid {
		nickname = targetUser.Nickname.String
	}
	reason := "не указана"
	if targetUser.BlockReason.Valid {
		reason = targetUser.BlockReason.String
	}
	blockDate := "неизвестно"
	if targetUser.BlockDate.Valid {
		blockDate = targetUser.BlockDate.Time.Format("02.01.2006 в 15:04")
	}

	msgText := fmt.Sprintf(
		"👤 Пользователь для разблокировки: *%s %s*\n"+
			"Никнейм: *%s*\n"+
			"Telegram ChatID: `%d`\n"+
			"Телефон: *%s*\n"+
			"Роль: *%s*\n"+
			"Заблокирован: *%s*\n"+
			"Причина: *%s*\n\n"+
			"Действительно разблокировать этого пользователя?",
		utils.EscapeTelegramMarkdown(targetUser.FirstName), utils.EscapeTelegramMarkdown(targetUser.LastName),
		utils.EscapeTelegramMarkdown(nickname), targetUser.ChatID,
		utils.EscapeTelegramMarkdown(phone), utils.EscapeTelegramMarkdown(utils.GetRoleDisplayName(targetUser.Role)),
		utils.EscapeTelegramMarkdown(blockDate), utils.EscapeTelegramMarkdown(reason),
	)
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔓 Да, разблокировать", fmt.Sprintf("unblock_user_final_%d", targetChatID)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("⬅️ Нет, назад к списку", "unblock_user_list_prompt"),
		),
	)
	_, errSend := bh.sendOrEditMessageHelper(chatID, messageIDToEdit, msgText, &keyboard, tgbotapi.ModeMarkdown)
	if errSend != nil {
		log.Printf("SendUnblockUserInfo: Ошибка для chatID %d: %v", chatID, errSend)
	}
}
