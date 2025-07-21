// Файл: internal/handlers/message_handler.go

package handlers

import (
	"Original/internal/constants" //
	"Original/internal/db"
	"Original/internal/models"
	"Original/internal/utils" //
	"database/sql"
	"fmt"
	"log"
	"regexp"
	"strconv"
	"strings"
	"time"

	tgbotapi "github.com/OvyFlash/telegram-bot-api"
)

// HandleMessage обрабатывает входящие сообщения от Telegram.
func (bh *BotHandler) HandleMessage(update tgbotapi.Update) {
	if update.Message == nil {
		return
	}

	message := update.Message
	chatID := message.Chat.ID
	userMessageID := message.MessageID
	text := strings.TrimSpace(message.Text)

	log.Printf("HandleMessage: ChatID=%d, UserMessageID=%d, Text='%s', MediaGroupID='%s', Photo: %v, Video: %v, Document: %v, Location: %v, Contact: %v",
		chatID, userMessageID, text, message.MediaGroupID, message.Photo != nil, message.Video != nil, message.Document != nil, message.Location != nil, message.Contact != nil)

	user, userExists := bh.getUserFromDB(chatID)
	if !userExists {
		if message.IsCommand() && message.Command() == "start" {
			// Логика регистрации ниже
		} else {
			log.Printf("HandleMessage: Пользователь с chatID %d не найден и это не команда /start. Сообщение проигнорировано и удалено.", chatID)
			bh.sendMessage(chatID, "Пожалуйста, начните с команды /start, чтобы зарегистрироваться или войти в систему.")
			bh.deleteMessageHelper(chatID, userMessageID)
			return
		}
	} else if user.IsBlocked {
		log.Printf("HandleMessage: Пользователь chatID %d заблокирован. Сообщение проигнорировано и удалено.", chatID)
		bh.sendMessage(chatID, "Ваш аккаунт заблокирован. Обратитесь к администратору.")
		bh.deleteMessageHelper(chatID, userMessageID)
		return
	}

	if message.IsCommand() {
		switch message.Command() {
		case "start":
			log.Printf("HandleMessage: Обработка команды /start для chatID %d, UserMessageID %d", chatID, userMessageID)
			var firstName, lastName string
			if message.From != nil {
				firstName = message.From.FirstName
				lastName = message.From.LastName
			}
			registeredUser, errReg := db.RegisterUser(chatID, firstName, lastName)
			if errReg != nil {
				log.Printf("HandleMessage: /start: Ошибка регистрации/получения пользователя для chatID %d: %v", chatID, errReg)
				bh.sendErrorMessageHelper(chatID, 0, "❌ Произошла ошибка при обработке ваших данных. Попробуйте еще раз.")
				return
			}
			user = registeredUser

			tempOrderDataForStart := bh.Deps.SessionManager.GetTempOrder(chatID)
			currentMenuMsgIDBeforeStart := tempOrderDataForStart.CurrentMessageID
			locationPromptMsgIDBeforeStart := tempOrderDataForStart.LocationPromptMessageID
			ephemeralMessagesBeforeStart := make([]int, len(tempOrderDataForStart.EphemeralMediaMessageIDs))
			copy(ephemeralMessagesBeforeStart, tempOrderDataForStart.EphemeralMediaMessageIDs)

			bh.Deps.SessionManager.ClearState(chatID)
			bh.Deps.SessionManager.ClearDeletedMessagesCacheForChat(chatID)
			bh.Deps.SessionManager.ClearTempOrder(chatID)
			bh.Deps.SessionManager.ClearTempDriverSettlement(chatID)
			errDbReset := db.ResetUserMainMenuMessageID(chatID)
			if errDbReset != nil {
				log.Printf("HandleMessage: /start: Ошибка сброса MainMenuMessageID для chatID %d: %v", chatID, errDbReset)
			}

			if currentMenuMsgIDBeforeStart != 0 && currentMenuMsgIDBeforeStart != message.MessageID {
				bh.deleteMessageHelper(chatID, currentMenuMsgIDBeforeStart)
			}
			if locationPromptMsgIDBeforeStart != 0 {
				bh.deleteMessageHelper(chatID, locationPromptMsgIDBeforeStart)
			}
			for _, ephemeralMsgID := range ephemeralMessagesBeforeStart {
				if ephemeralMsgID != message.MessageID && ephemeralMsgID != currentMenuMsgIDBeforeStart && ephemeralMsgID != locationPromptMsgIDBeforeStart {
					bh.deleteMessageHelper(chatID, ephemeralMsgID)
				}
			}

			// --- НАЧАЛО ИЗМЕНЕНИЯ ---
			// Вместо немедленной отправки главного меню, отправляем меню-шлюз
			// с выбором: Web App или продолжить в боте.
			bh.SendGatewayMenu(chatID, 0)
			// --- КОНЕЦ ИЗМЕНЕНИЯ ---

			bh.deleteMessageHelper(chatID, message.MessageID)
			log.Printf("HandleMessage: /start: Обработка команды /start завершена для chatID %d, отправлено меню-шлюз.", chatID)
			return
		default:
			log.Printf("HandleMessage: Неизвестная команда '%s' от chatID %d", message.Command(), chatID)
			bh.deleteMessageHelper(chatID, userMessageID)
			bh.sendErrorMessageHelper(chatID, 0, "Неизвестная команда.")
			return
		}
	}

	currentState := bh.Deps.SessionManager.GetState(chatID)
	log.Printf("HandleMessage: Текущее состояние для chatID %d: %s", chatID, currentState)

	var botMenuMsgID int
	tempOrderForMenuID := bh.Deps.SessionManager.GetTempOrder(chatID)
	botMenuMsgID = tempOrderForMenuID.CurrentMessageID

	isAlbumItem := message.MediaGroupID != "" && (message.Photo != nil || message.Video != nil)

	shouldProcessThisAlbumItem := false
	if isAlbumItem {
		if currentState == constants.STATE_ORDER_PHOTO {
			shouldProcessThisAlbumItem = true
		} else if tempOrderForMenuID.ActiveMediaGroupID != "" && tempOrderForMenuID.ActiveMediaGroupID == message.MediaGroupID {
			shouldProcessThisAlbumItem = true
			log.Printf("HandleMessage: Обработка элемента альбома для MediaGroupID '%s', несмотря на состояние '%s', так как он совпадает с ActiveMediaGroupID.", message.MediaGroupID, currentState)
		}
	}

	if isAlbumItem {
		if shouldProcessThisAlbumItem {
			var fileID string
			var mediaTypeToAdd string
			if message.Photo != nil && len(message.Photo) > 0 {
				fileID = message.Photo[len(message.Photo)-1].FileID
				mediaTypeToAdd = "photo"
			} else if message.Video != nil {
				fileID = message.Video.FileID
				mediaTypeToAdd = "video"
			} else {
				log.Printf("HandleMessage: Элемент альбома MediaGroupID '%s' (сообщение %d) не содержит фото/видео. Пропуск.", message.MediaGroupID, userMessageID)
				bh.deleteMessageHelper(chatID, userMessageID)
				return
			}

			_, _, errAdd := bh.Deps.SessionManager.AddMediaToTempOrder(chatID, fileID, mediaTypeToAdd, message.MediaGroupID, currentState == constants.STATE_ORDER_PHOTO)
			if errAdd != nil {
				log.Printf("HandleMessage: Не удалось добавить медиа %s (тип: %s, группа: %s) из альбома: %v. ChatID: %d", fileID, mediaTypeToAdd, message.MediaGroupID, errAdd, chatID)
				if strings.Contains(errAdd.Error(), "лимит") {
					bh.sendErrorMessageHelper(chatID, botMenuMsgID, errAdd.Error())
				} else if strings.Contains(errAdd.Error(), "уже добавлено") {
					log.Printf("HandleMessage: Попытка добавить дубликат медиа %s (группа: %s). ChatID: %d", fileID, message.MediaGroupID, chatID)
				}
			}

			if currentState == constants.STATE_ORDER_PHOTO {
				bh.SendPhotoInputMenu(chatID, botMenuMsgID)
			}
		} else {
			log.Printf("HandleMessage: Элемент альбома MediaGroupID '%s' (сообщение %d) не будет обработан. CurrentState: '%s', ActiveMediaGroupID в сессии: '%s'.",
				message.MediaGroupID, userMessageID, currentState, tempOrderForMenuID.ActiveMediaGroupID)
		}
		bh.deleteMessageHelper(chatID, userMessageID)
		return

	}

	switch currentState {
	case constants.STATE_ORDER_FINAL_COST_INPUT:
		if !utils.IsOperatorOrHigher(user.Role) {
			bh.sendAccessDenied(chatID, botMenuMsgID)
			bh.deleteMessageHelper(chatID, userMessageID)
			return
		}
		bh.deleteMessageHelper(chatID, userMessageID)

		finalCost, err := strconv.ParseFloat(strings.Replace(text, ",", ".", -1), 64)
		if err != nil || finalCost < 0 {
			bh.sendErrorMessageHelper(chatID, botMenuMsgID, "❌ Неверный формат стоимости. Введите целое число (например, 123) или число с точкой (123.45).")
			return
		}

		tempOrderData := bh.Deps.SessionManager.GetTempOrder(chatID)
		orderID := int(tempOrderData.ID)

		errUpdate := db.UpdateOrderField(int64(orderID), "cost", finalCost)
		if errUpdate != nil {
			log.Printf("handleFinalCostInput: Ошибка обновления итоговой стоимости для заказа #%d: %v", orderID, errUpdate)
			bh.sendErrorMessageHelper(chatID, botMenuMsgID, "❌ Ошибка сохранения новой стоимости.")
			return
		}
		log.Printf("Итоговая стоимость для заказа #%d обновлена на %.0f оператором %d", orderID, finalCost, chatID)
		bh.sendInfoMessage(chatID, botMenuMsgID, fmt.Sprintf("✅ Итоговая стоимость для заказа №%d обновлена на %.0f ₽.", orderID, finalCost), fmt.Sprintf("view_order_ops_%d", orderID))
		bh.Deps.SessionManager.ClearState(chatID)
		bh.SendViewOrderDetails(chatID, orderID, botMenuMsgID, true, user)

	case constants.STATE_ORDER_DESCRIPTION:
		bh.handleOrderDescriptionInput(chatID, user, text, userMessageID, botMenuMsgID)

	case constants.STATE_ORDER_NAME:
		bh.handleOrderNameInput(chatID, user, text, userMessageID, botMenuMsgID)
	case constants.STATE_ORDER_PHONE:
		phoneInput := text
		if message.Contact != nil {
			phoneInput = message.Contact.PhoneNumber
		}
		bh.handleOrderPhoneInput(chatID, user, phoneInput, userMessageID, botMenuMsgID)
	case constants.STATE_ORDER_ADDRESS:
		if message.Location != nil {
			bh.handleLocationMessage(chatID, user, message.Location, userMessageID, botMenuMsgID)
		} else {
			bh.handleOrderAddressTextInput(chatID, user, text, userMessageID, botMenuMsgID)
		}
	case constants.STATE_ORDER_ADDRESS_LOCATION:
		if message.Location != nil {
			bh.handleLocationMessage(chatID, user, message.Location, userMessageID, botMenuMsgID)
		} else {
			bh.sendErrorMessageHelper(chatID, botMenuMsgID, "Пожалуйста, отправьте ваше местоположение с помощью кнопки '📍 Отправить мое местоположение' или вернитесь назад.")
			bh.deleteMessageHelper(chatID, userMessageID)
		}
	case constants.STATE_ORDER_PHOTO:
		if message.Photo == nil && message.Video == nil && text != "" {
			bh.deleteMessageHelper(chatID, userMessageID)
			bh.sendInfoMessage(chatID, botMenuMsgID, "Пожалуйста, отправьте фото/видео или используйте кнопки.", "")
		} else if message.Photo != nil || message.Video != nil {
			var fileID string
			var mediaTypeToAdd string
			if message.Photo != nil && len(message.Photo) > 0 {
				fileID = message.Photo[len(message.Photo)-1].FileID
				mediaTypeToAdd = "photo"
			} else if message.Video != nil {
				fileID = message.Video.FileID
				mediaTypeToAdd = "video"
			}

			if fileID != "" {
				_, _, errAdd := bh.Deps.SessionManager.AddMediaToTempOrder(chatID, fileID, mediaTypeToAdd, "", true)
				if errAdd != nil {
					log.Printf("HandleMessage (single media): Не удалось добавить медиа %s (тип: %s): %v. ChatID: %d", fileID, mediaTypeToAdd, errAdd, chatID)
					if strings.Contains(errAdd.Error(), "лимит") {
						bh.sendErrorMessageHelper(chatID, botMenuMsgID, errAdd.Error())
					}
				}
				bh.SendPhotoInputMenu(chatID, botMenuMsgID)
			}
			bh.deleteMessageHelper(chatID, userMessageID)
		} else {
			bh.deleteMessageHelper(chatID, userMessageID)
		}
		return
	case constants.STATE_ORDER_TIME:
		bh.deleteMessageHelper(chatID, userMessageID)
		hourOnlyRegex := regexp.MustCompile(`^([01]?[0-9]|2[0-3])$`)
		if hourOnlyRegex.MatchString(text) {
			selectedHour, err := strconv.Atoi(text)
			if err == nil && selectedHour >= 0 && selectedHour <= 23 {
				log.Printf("HandleMessage: STATE_ORDER_TIME, пользователь ввел час: %d. Переход к выбору минут.", selectedHour)
				bh.SendMinuteSelectionMenu(chatID, selectedHour, botMenuMsgID)
				return
			}
		}
		timeRegex := regexp.MustCompile(`^([01]?[0-9]|2[0-3])[:.-]?([0-5][0-9])$`)
		matches := timeRegex.FindStringSubmatch(text)

		if len(matches) == 3 {
			hour, _ := strconv.Atoi(matches[1])
			minute, _ := strconv.Atoi(matches[2])
			selectedTimeStr := fmt.Sprintf("%02d:%02d", hour, minute)
			log.Printf("HandleMessage: STATE_ORDER_TIME, пользователь ввел полное время: %s. Обработка...", selectedTimeStr)
			tempOrder := bh.Deps.SessionManager.GetTempOrder(chatID)
			tempOrder.Time = selectedTimeStr
			if tempOrder.Date == "" {
				log.Printf("HandleMessage: STATE_ORDER_TIME, дата не выбрана, а время %s введено. Возврат к выбору даты.", selectedTimeStr)
				bh.sendErrorMessageHelper(chatID, botMenuMsgID, "Пожалуйста, сначала выберите дату.")
				bh.SendDateSelectionMenu(chatID, botMenuMsgID, 0)
				return
			}
			bh.Deps.SessionManager.UpdateTempOrder(chatID, tempOrder)
			history := bh.Deps.SessionManager.GetHistory(chatID)
			isEditingOrder := false
			if len(history) >= 2 && history[len(history)-2] == constants.STATE_ORDER_EDIT && tempOrder.ID != 0 {
				isEditingOrder = true
			}
			if isEditingOrder {
				log.Printf("Редактирование: сохранение времени '%s' для заказа #%d. ChatID=%d", tempOrder.Time, tempOrder.ID, chatID)
				if errDb := db.UpdateOrderField(tempOrder.ID, "time", tempOrder.Time); errDb != nil {
					log.Printf("Ошибка сохранения времени для заказа #%d: %v. ChatID=%d", tempOrder.ID, errDb, chatID)
					bh.sendErrorMessageHelper(chatID, botMenuMsgID, "❌ Ошибка сохранения времени заказа.")
					return
				}
				bh.SendEditOrderMenu(chatID, botMenuMsgID)
			} else {
				log.Printf("Переход к вводу телефона после ввода времени %s. ChatID=%d", tempOrder.Time, chatID)
				bh.SendPhoneInputMenu(chatID, user, botMenuMsgID)
			}
		} else {
			log.Printf("HandleMessage: STATE_ORDER_TIME, неверный формат времени: '%s'. Повторный запрос.", text)
			bh.sendErrorMessageHelper(chatID, botMenuMsgID, "❌ Неверный формат времени. Введите час (например, 9) или точное время (например, 09:30).")
			bh.SendTimeSelectionMenu(chatID, botMenuMsgID)
		}
	case constants.STATE_ORDER_MINUTE_SELECTION:
		bh.deleteMessageHelper(chatID, userMessageID)
		timeRegex := regexp.MustCompile(`^([01]?[0-9]|2[0-3])[:.-]?([0-5][0-9])$`)
		matches := timeRegex.FindStringSubmatch(text)
		tempOrder := bh.Deps.SessionManager.GetTempOrder(chatID)
		if len(matches) == 3 {
			hour, _ := strconv.Atoi(matches[1])
			minute, _ := strconv.Atoi(matches[2])
			if tempOrder.SelectedHourForMinuteView != -1 && tempOrder.SelectedHourForMinuteView != hour {
				bh.sendErrorMessageHelper(chatID, botMenuMsgID, fmt.Sprintf("❌ Вы выбрали час %02d:xx. Пожалуйста, введите минуты для этого часа или вернитесь назад, чтобы выбрать другой час.", tempOrder.SelectedHourForMinuteView))
				bh.SendMinuteSelectionMenu(chatID, tempOrder.SelectedHourForMinuteView, botMenuMsgID)
				return
			}
			selectedTimeStr := fmt.Sprintf("%02d:%02d", hour, minute)
			log.Printf("HandleMessage: STATE_ORDER_MINUTE_SELECTION, пользователь ввел полное время: %s. Обработка...", selectedTimeStr)
			tempOrder.Time = selectedTimeStr
			bh.Deps.SessionManager.UpdateTempOrder(chatID, tempOrder)
			history := bh.Deps.SessionManager.GetHistory(chatID)
			isEditingOrder := false
			if len(history) >= 2 && history[len(history)-2] == constants.STATE_ORDER_EDIT && tempOrder.ID != 0 {
				isEditingOrder = true
			}
			if isEditingOrder {
				log.Printf("Редактирование: сохранение времени '%s' для заказа #%d. ChatID=%d", tempOrder.Time, tempOrder.ID, chatID)
				if errDb := db.UpdateOrderField(tempOrder.ID, "time", tempOrder.Time); errDb != nil {
					log.Printf("Ошибка сохранения времени для заказа #%d: %v. ChatID=%d", tempOrder.ID, errDb, chatID)
					bh.sendErrorMessageHelper(chatID, botMenuMsgID, "❌ Ошибка сохранения времени заказа.")
					return
				}
				bh.SendEditOrderMenu(chatID, botMenuMsgID)
			} else {
				log.Printf("Переход к вводу телефона после ввода времени %s. ChatID=%d", tempOrder.Time, chatID)
				bh.SendPhoneInputMenu(chatID, user, botMenuMsgID)
			}
		} else {
			log.Printf("HandleMessage: STATE_ORDER_MINUTE_SELECTION, неверный формат времени: '%s'. Повторный запрос.", text)
			bh.sendErrorMessageHelper(chatID, botMenuMsgID, "❌ Неверный формат времени. Введите точное время (например, 09:15) или выберите кнопкой.")
			if tempOrder.SelectedHourForMinuteView != -1 {
				bh.SendMinuteSelectionMenu(chatID, tempOrder.SelectedHourForMinuteView, botMenuMsgID)
			} else {
				bh.SendTimeSelectionMenu(chatID, botMenuMsgID)
			}
		}

	case constants.STATE_CHAT_MESSAGE_INPUT:
		bh.handleChatMessageInput(chatID, user, text, userMessageID, botMenuMsgID)
	case constants.STATE_PHONE_AWAIT_INPUT:
		phoneInput := text
		if message.Contact != nil {
			phoneInput = message.Contact.PhoneNumber
		}
		bh.handlePhoneAwaitInput(chatID, user, phoneInput, userMessageID, botMenuMsgID)
	case constants.STATE_COST_INPUT:
		bh.handleCostInput(chatID, user, text, userMessageID, botMenuMsgID)
	case constants.STATE_CANCEL_REASON:
		bh.handleCancelReasonInput(chatID, user, text, userMessageID, botMenuMsgID)

	case constants.STATE_STAFF_ADD_NAME, constants.STATE_STAFF_ADD_SURNAME, constants.STATE_STAFF_ADD_NICKNAME,
		constants.STATE_STAFF_ADD_PHONE, constants.STATE_STAFF_ADD_CHATID:
		bh.handleStaffAddInput(chatID, user, currentState, text, userMessageID, botMenuMsgID)
	case constants.STATE_STAFF_ADD_CARD_NUMBER:
		bh.handleStaffCardNumberInput(chatID, user, text, userMessageID, botMenuMsgID, true)
	case constants.STATE_STAFF_EDIT_NAME, constants.STATE_STAFF_EDIT_SURNAME, constants.STATE_STAFF_EDIT_NICKNAME,
		constants.STATE_STAFF_EDIT_PHONE:
		bh.handleStaffEditInput(chatID, user, currentState, text, userMessageID, botMenuMsgID)
	case constants.STATE_STAFF_EDIT_CARD_NUMBER:
		bh.handleStaffCardNumberInput(chatID, user, text, userMessageID, botMenuMsgID, false)
	case constants.STATE_STAFF_BLOCK_REASON:
		bh.handleStaffBlockReasonInput(chatID, user, text, userMessageID, botMenuMsgID)
	case constants.STATE_BLOCK_REASON:
		bh.handleUserBlockReasonInput(chatID, user, text, userMessageID, botMenuMsgID)

	case constants.STATE_DRIVER_REPORT_INPUT_FUEL:
		bh.deleteMessageHelper(chatID, userMessageID)
		fuel, err := strconv.ParseFloat(strings.Replace(text, ",", ".", -1), 64)
		if err != nil || fuel < 0 {
			bh.sendErrorMessageHelper(chatID, botMenuMsgID, "Сумма на топливо должна быть числом (минимум 0). Попробуйте снова.")
			return
		}
		tempData := bh.Deps.SessionManager.GetTempDriverSettlement(chatID)
		tempData.FuelExpense = fuel
		bh.Deps.SessionManager.UpdateTempDriverSettlement(chatID, tempData)
		bh.SendDriverReportOverallMenu(chatID, user, botMenuMsgID)

	case constants.STATE_DRIVER_REPORT_INPUT_OTHER_EXPENSE_DESCRIPTION:
		bh.deleteMessageHelper(chatID, userMessageID)
		description := strings.TrimSpace(text)
		if description == "" {
			bh.sendErrorMessageHelper(chatID, botMenuMsgID, "Описание прочего расхода не может быть пустым. Попробуйте снова.")
			bh.SendDriverReportOtherExpenseDescriptionPrompt(chatID, user, botMenuMsgID, false, -1)
			return
		}
		if len(description) > 255 {
			bh.sendErrorMessageHelper(chatID, botMenuMsgID, "Описание слишком длинное (макс. 255 символов). Попробуйте снова.")
			bh.SendDriverReportOtherExpenseDescriptionPrompt(chatID, user, botMenuMsgID, false, -1)
			return
		}
		tempData := bh.Deps.SessionManager.GetTempDriverSettlement(chatID)
		tempData.TempOtherExpenseDescription = description
		bh.Deps.SessionManager.UpdateTempDriverSettlement(chatID, tempData)
		bh.SendDriverReportOtherExpenseAmountPrompt(chatID, user, botMenuMsgID, description, false, -1)

	case constants.STATE_DRIVER_REPORT_INPUT_OTHER_EXPENSE_AMOUNT:
		bh.deleteMessageHelper(chatID, userMessageID)
		amount, err := strconv.ParseFloat(strings.Replace(text, ",", ".", -1), 64)
		if err != nil || amount <= 0 {
			tempData := bh.Deps.SessionManager.GetTempDriverSettlement(chatID)
			bh.sendErrorMessageHelper(chatID, botMenuMsgID, "Сумма расхода должна быть положительным числом. Попробуйте снова.")
			bh.SendDriverReportOtherExpenseAmountPrompt(chatID, user, botMenuMsgID, tempData.TempOtherExpenseDescription, false, -1)
			return
		}
		tempData := bh.Deps.SessionManager.GetTempDriverSettlement(chatID)
		if tempData.TempOtherExpenseDescription == "" {
			bh.sendErrorMessageHelper(chatID, botMenuMsgID, "Ошибка: описание для прочего расхода не найдено. Начните добавление заново.")
			bh.SendDriverReportOtherExpensesMenu(chatID, user, botMenuMsgID)
			return
		}
		tempData.OtherExpenses = append(tempData.OtherExpenses, models.OtherExpenseDetail{
			Description: tempData.TempOtherExpenseDescription,
			Amount:      amount,
		})
		addedDesc := tempData.TempOtherExpenseDescription
		tempData.TempOtherExpenseDescription = ""
		bh.Deps.SessionManager.UpdateTempDriverSettlement(chatID, tempData)
		bh.SendDriverReportConfirmAddOtherExpense(chatID, user, botMenuMsgID, addedDesc, amount)

	case constants.STATE_DRIVER_REPORT_INPUT_LOADER_NAME:
		bh.deleteMessageHelper(chatID, userMessageID)
		loaderName := strings.TrimSpace(text)
		if loaderName == "" {
			bh.sendErrorMessageHelper(chatID, botMenuMsgID, "Имя/идентификатор грузчика не может быть пустым. Попробуйте снова.")
			return
		}
		tempData := bh.Deps.SessionManager.GetTempDriverSettlement(chatID)
		tempData.TempLoaderNameInput = loaderName
		bh.Deps.SessionManager.UpdateTempDriverSettlement(chatID, tempData)
		bh.SendDriverReportLoaderSalaryInputPrompt(chatID, user, botMenuMsgID, loaderName, false, -1)

	case constants.STATE_DRIVER_REPORT_INPUT_LOADER_SALARY:
		bh.deleteMessageHelper(chatID, userMessageID)
		salary, err := strconv.ParseFloat(strings.Replace(text, ",", ".", -1), 64)
		if err != nil || salary <= 0 {
			bh.sendErrorMessageHelper(chatID, botMenuMsgID, "Сумма зарплаты грузчика должна быть положительным числом. Попробуйте снова.")
			tempData := bh.Deps.SessionManager.GetTempDriverSettlement(chatID)
			bh.SendDriverReportLoaderSalaryInputPrompt(chatID, user, botMenuMsgID, tempData.TempLoaderNameInput, false, -1)
			return
		}
		tempData := bh.Deps.SessionManager.GetTempDriverSettlement(chatID)
		if tempData.TempLoaderNameInput == "" {
			bh.sendErrorMessageHelper(chatID, botMenuMsgID, "Ошибка: не найдено имя для нового грузчика. Начните добавление заново.")
			bh.SendDriverReportLoadersSubMenu(chatID, user, botMenuMsgID)
			return
		}
		tempData.LoaderPayments = append(tempData.LoaderPayments, models.LoaderPaymentDetail{
			LoaderIdentifier: tempData.TempLoaderNameInput,
			Amount:           salary,
		})
		tempData.TempLoaderNameInput = ""
		tempData.EditingLoaderIndex = -1
		bh.Deps.SessionManager.UpdateTempDriverSettlement(chatID, tempData)
		bh.SendDriverReportLoadersSubMenu(chatID, user, botMenuMsgID)

	case constants.STATE_DRIVER_REPORT_EDIT_LOADER_SALARY:
		bh.deleteMessageHelper(chatID, userMessageID)
		salary, err := strconv.ParseFloat(strings.Replace(text, ",", ".", -1), 64)
		if err != nil || salary <= 0 {
			tempData := bh.Deps.SessionManager.GetTempDriverSettlement(chatID)
			bh.sendErrorMessageHelper(chatID, botMenuMsgID, "Сумма зарплаты грузчика должна быть положительным числом. Попробуйте снова.")
			if tempData.EditingLoaderIndex >= 0 && tempData.EditingLoaderIndex < len(tempData.LoaderPayments) {
				loaderToEdit := tempData.LoaderPayments[tempData.EditingLoaderIndex]
				bh.SendDriverReportLoaderSalaryInputPrompt(chatID, user, botMenuMsgID, loaderToEdit.LoaderIdentifier, true, tempData.EditingLoaderIndex)
			} else {
				bh.SendDriverReportLoadersSubMenu(chatID, user, botMenuMsgID)
			}
			return
		}
		tempData := bh.Deps.SessionManager.GetTempDriverSettlement(chatID)
		if tempData.EditingLoaderIndex < 0 || tempData.EditingLoaderIndex >= len(tempData.LoaderPayments) {
			bh.sendErrorMessageHelper(chatID, botMenuMsgID, "Ошибка: не выбран грузчик для редактирования зарплаты.")
			bh.SendDriverReportLoadersSubMenu(chatID, user, botMenuMsgID)
			return
		}
		tempData.LoaderPayments[tempData.EditingLoaderIndex].Amount = salary
		tempData.EditingLoaderIndex = -1
		bh.Deps.SessionManager.UpdateTempDriverSettlement(chatID, tempData)
		bh.SendDriverReportLoadersSubMenu(chatID, user, botMenuMsgID)

	case constants.STATE_OWNER_CASH_EDIT_SETTLEMENT_FIELD:
		bh.handleOwnerSaveEditedSettlementFieldInput(chatID, user, text, userMessageID, botMenuMsgID)
	case constants.STATE_OPERATOR_REJECT_REASON_INPUT:
		bh.deleteMessageHelper(chatID, userMessageID)
		if !utils.IsOperatorOrHigher(user.Role) {
			bh.sendAccessDenied(chatID, botMenuMsgID)
			return
		}
		rejectionReason := strings.TrimSpace(text)
		if len(rejectionReason) < 5 {
			bh.sendErrorMessageHelper(chatID, botMenuMsgID, "Причина отклонения должна быть более 5 символов. Попробуйте снова.")
			return
		}

		tempData := bh.Deps.SessionManager.GetTempDriverSettlement(chatID)
		settlementID := tempData.EditingSettlementID
		if settlementID == 0 {
			bh.sendErrorMessageHelper(chatID, botMenuMsgID, "Ошибка: ID отчета для отклонения не найден в сессии.")
			bh.SendMainMenu(chatID, user, botMenuMsgID)
			return
		}

		bh.handleOperatorFinalizeRejection(chatID, user, settlementID, rejectionReason, botMenuMsgID)
	case constants.STATE_OP_ORDER_COST_INPUT:
		bh.deleteMessageHelper(chatID, userMessageID)
		tempData := bh.Deps.SessionManager.GetTempOrder(chatID)
		isOperatorFlow := tempData.OrderAction == "operator_creating_order" && utils.IsOperatorOrHigher(user.Role)
		isDriverFlow := tempData.OrderAction == "driver_creating_order" && user.Role == constants.ROLE_DRIVER

		if !isOperatorFlow && !isDriverFlow {
			log.Printf("HandleMessage: Попытка доступа к STATE_OP_ORDER_COST_INPUT без прав. ChatID: %d, Role: %s, OrderAction: %s", chatID, user.Role, tempData.OrderAction)
			bh.sendAccessDenied(chatID, botMenuMsgID)
			return
		}

		cost, err := strconv.ParseFloat(strings.Replace(text, ",", ".", -1), 64)
		if err != nil || cost <= 0 {
			bh.sendErrorMessageHelper(chatID, botMenuMsgID, "❌ Стоимость должна быть положительным числом (например, 1500).")
			bh.SendOpOrderCostInputMenu(chatID, tempData.ID, botMenuMsgID)
			return
		}

		orderID := tempData.ID
		if orderID == 0 {
			bh.sendErrorMessageHelper(chatID, botMenuMsgID, "❌ Ошибка: ID заказа не определен для установки стоимости.")
			bh.SendMainMenu(chatID, user, botMenuMsgID)
			return
		}
		tempData.Cost.Float64 = cost
		tempData.Cost.Valid = true
		bh.Deps.SessionManager.UpdateTempOrder(chatID, tempData)
		log.Printf("Пользователь %d (Роль: %s) установил стоимость %.2f для заказа #%d (в процессе создания).", chatID, user.Role, cost, orderID)

		bh.SendAssignExecutorsMenu(chatID, orderID, botMenuMsgID)

	default:
		if text != "" {
			log.Printf("HandleMessage: Неожиданный текст '%s' в состоянии '%s' от chatID %d. Сообщение %d удалено.", text, currentState, chatID, userMessageID)
			bh.deleteMessageHelper(chatID, userMessageID)
			if botMenuMsgID != 0 {
				bh.sendInfoMessage(chatID, botMenuMsgID, "Пожалуйста, используйте кнопки для навигации или следуйте инструкциям на экране.", "")
			}
		} else if message.Document != nil || message.Audio != nil || message.Voice != nil || message.Sticker != nil || message.Animation != nil {
			log.Printf("HandleMessage: Получен необрабатываемый тип сообщения в состоянии '%s' от chatID %d. Сообщение %d удалено.", currentState, chatID, userMessageID)
			bh.deleteMessageHelper(chatID, userMessageID)
			if botMenuMsgID != 0 {
				bh.sendInfoMessage(chatID, botMenuMsgID, "Этот тип сообщения не поддерживается на данном шаге.", "")
			}
		} else if message.Photo != nil || message.Video != nil {
			log.Printf("HandleMessage: Получено одиночное фото/видео в состоянии '%s' (ожидалось '%s' или др.) от ChatID %d. Сообщение %d удалено.", currentState, constants.STATE_ORDER_PHOTO, chatID, userMessageID)
			bh.deleteMessageHelper(chatID, userMessageID)
			if botMenuMsgID != 0 {
				bh.sendInfoMessage(chatID, botMenuMsgID, "Фото/видео можно загружать только на соответствующем шаге.", "")
			}
		}
	}
}

// --- Вспомогательные функции ---

func (bh *BotHandler) handleOrderDescriptionInput(chatID int64, user models.User, description string, userMsgID int, botMenuMsgID int) {
	log.Printf("handleOrderDescriptionInput: ChatID=%d, Введенное описание='%s', UserMsgID=%d, BotMenuMsgID=%d", chatID, description, userMsgID, botMenuMsgID)
	bh.deleteMessageHelper(chatID, userMsgID)

	trimmedDescription := strings.TrimSpace(description)
	maxLength := 1000
	if len(trimmedDescription) > maxLength {
		bh.sendErrorMessageHelper(chatID, botMenuMsgID, fmt.Sprintf("❌ Описание слишком длинное (максимум %d символов). Попробуйте снова.", maxLength))
		return
	}

	tempOrder := bh.Deps.SessionManager.GetTempOrder(chatID)
	tempOrder.Description = trimmedDescription
	bh.Deps.SessionManager.UpdateTempOrder(chatID, tempOrder)

	history := bh.Deps.SessionManager.GetHistory(chatID)
	isEditingOrder := len(history) >= 2 && history[len(history)-2] == constants.STATE_ORDER_EDIT && tempOrder.ID != 0

	if isEditingOrder {
		log.Printf("Редактирование: сохранение описания для заказа #%d. ChatID=%d", tempOrder.ID, chatID)
		if errDb := db.UpdateOrderField(tempOrder.ID, "description", trimmedDescription); errDb != nil {
			log.Printf("Ошибка сохранения описания для заказа #%d: %v. ChatID=%d", tempOrder.ID, errDb, chatID)
			bh.sendErrorMessageHelper(chatID, botMenuMsgID, "❌ Ошибка сохранения описания заказа.")
			return
		}
		bh.SendEditOrderMenu(chatID, botMenuMsgID)
	} else {
		log.Printf("Описание '%s' установлено. Переход к вводу имени. ChatID=%d", trimmedDescription, chatID)
		bh.SendNameInputMenu(chatID, botMenuMsgID)
	}
}

func (bh *BotHandler) handleOrderNameInput(chatID int64, user models.User, enteredName string, userMsgID int, botMenuMsgID int) {
	log.Printf("handleOrderNameInput: ChatID=%d, Введенное имя='%s', ID сообщения пользователя=%d, ID меню бота=%d", chatID, enteredName, userMsgID, botMenuMsgID)
	bh.deleteMessageHelper(chatID, userMsgID)

	trimmedName := strings.TrimSpace(enteredName)
	if len(trimmedName) < 2 || len(trimmedName) > 50 {
		currentMsgID := botMenuMsgID
		sentErrorMsg, _ := bh.sendErrorMessageHelper(chatID, botMenuMsgID, "❌ Имя должно содержать от 2 до 50 символов. Попробуйте снова.")
		if sentErrorMsg.MessageID != 0 {
			currentMsgID = sentErrorMsg.MessageID
		}
		tempOrder := bh.Deps.SessionManager.GetTempOrder(chatID)
		tempOrder.CurrentMessageID = currentMsgID
		bh.Deps.SessionManager.UpdateTempOrder(chatID, tempOrder)
		return
	}
	userInDBBeforeUpdate, ok := bh.getUserFromDB(chatID)
	if !ok {
		bh.sendErrorMessageHelper(chatID, botMenuMsgID, "Произошла ошибка с вашими данными. Попробуйте /start")
		return
	}

	tempOrder := bh.Deps.SessionManager.GetTempOrder(chatID)
	history := bh.Deps.SessionManager.GetHistory(chatID)
	isEditingOrder := len(history) >= 2 && history[len(history)-2] == constants.STATE_ORDER_EDIT && tempOrder.ID != 0

	isOperatorCreatingForClient := tempOrder.OrderAction == "operator_creating_order" && tempOrder.UserChatID != chatID && tempOrder.UserChatID != 0
	// --- Новое условие для водителя ---
	isDriverCreating := tempOrder.OrderAction == "driver_creating_order"

	// Обновляем имя в tempOrder для заказа
	tempOrder.Name = trimmedName
	bh.Deps.SessionManager.UpdateTempOrder(chatID, tempOrder)

	// Если это НЕ оператор/водитель создает для клиента И имя текущего пользователя в БД пустое И это НЕ редактирование
	if !isOperatorCreatingForClient && !isDriverCreating && userInDBBeforeUpdate.FirstName == "" && !isEditingOrder {
		errDB := db.UpdateUserField(chatID, "first_name", trimmedName)
		if errDB != nil {
			log.Printf("handleOrderNameInput: Ошибка обновления основного имени для chatID %d: %v", chatID, errDB)
			bh.sendErrorMessageHelper(chatID, botMenuMsgID, "❌ Произошла ошибка при сохранении вашего имени.")
			return
		}
		log.Printf("Основное имя для chatID %d успешно обновлено на '%s' в БД.", chatID, trimmedName)
		// user.FirstName = trimmedName // user - это копия, ее изменение не повлияет на user из вызывающей функции
		confirmationMessage := fmt.Sprintf("✅ Ваше имя '%s' сохранено! Теперь укажите дату заказа.", utils.EscapeTelegramMarkdown(trimmedName))
		sentConfirmMsg, _ := bh.sendOrEditMessageHelper(chatID, botMenuMsgID, confirmationMessage, nil, tgbotapi.ModeMarkdown)
		nextMenuMsgID := botMenuMsgID
		if sentConfirmMsg.MessageID != 0 {
			nextMenuMsgID = sentConfirmMsg.MessageID
		}
		updatedTempOrder := bh.Deps.SessionManager.GetTempOrder(chatID)
		updatedTempOrder.CurrentMessageID = nextMenuMsgID
		bh.Deps.SessionManager.UpdateTempOrder(chatID, updatedTempOrder)
		bh.SendDateSelectionMenu(chatID, nextMenuMsgID, 0)
	} else { // Иначе (оператор для клиента, водитель для клиента, или имя в профиле есть, или это редактирование)
		if isEditingOrder {
			log.Printf("Редактирование: сохранение имени заказа '%s' для заказа #%d. ChatID=%d", trimmedName, tempOrder.ID, chatID)
			if err := db.UpdateOrderField(tempOrder.ID, "name", trimmedName); err != nil {
				log.Printf("Ошибка сохранения имени для заказа #%d: %v. ChatID=%d", tempOrder.ID, err, chatID)
				bh.sendErrorMessageHelper(chatID, botMenuMsgID, "❌ Ошибка сохранения имени заказа.")
				return
			}
			bh.SendEditOrderMenu(chatID, botMenuMsgID)
		} else { // Создание нового заказа (клиентом, оператором для клиента/себя или водителем для клиента)
			log.Printf("Имя для заказа '%s' установлено. Переход к выбору даты. ChatID=%d", trimmedName, chatID)
			bh.SendDateSelectionMenu(chatID, botMenuMsgID, 0)
		}
	}
}

func (bh *BotHandler) handleOrderPhoneInput(chatID int64, user models.User, phoneInput string, userMsgID int, botMenuMsgID int) {
	normalizedPhone, errValidate := utils.ValidatePhoneNumber(phoneInput)
	if errValidate != nil {
		currentMsgID := botMenuMsgID
		sentErrorMsg, _ := bh.sendErrorMessageHelper(chatID, botMenuMsgID, fmt.Sprintf("❌ Неверный формат номера: %s. Попробуйте снова.", errValidate.Error()))
		if sentErrorMsg.MessageID != 0 {
			currentMsgID = sentErrorMsg.MessageID
		}
		bh.deleteMessageHelper(chatID, userMsgID)
		tempOrder := bh.Deps.SessionManager.GetTempOrder(chatID)
		tempOrder.CurrentMessageID = currentMsgID
		bh.Deps.SessionManager.UpdateTempOrder(chatID, tempOrder)
		return
	}
	tempOrder := bh.Deps.SessionManager.GetTempOrder(chatID)
	tempOrder.Phone = normalizedPhone
	bh.Deps.SessionManager.UpdateTempOrder(chatID, tempOrder)
	bh.deleteMessageHelper(chatID, userMsgID)

	userForPhoneUpdate := user // По умолчанию текущий пользователь
	targetChatIDForDBUpdate := chatID

	// Если оператор создает для клиента, обновляем телефон клиента (если он отличается)
	// --- Добавляем проверку для водителя ---
	isOperatorFlow := tempOrder.OrderAction == "operator_creating_order" && tempOrder.UserChatID != chatID && tempOrder.UserChatID != 0
	isDriverFlow := tempOrder.OrderAction == "driver_creating_order"

	// Если водитель создает заказ, то он тоже вводит телефон клиента, но мы не знаем ChatID этого клиента.
	// Поэтому мы не можем обновить профиль клиента. Мы просто сохраняем телефон в заказе.
	// Только если оператор выбирает существующего клиента, мы можем обновить профиль.
	if isOperatorFlow {
		clientUser, clientFound := bh.getUserFromDB(tempOrder.UserChatID)
		if clientFound {
			userForPhoneUpdate = clientUser
			targetChatIDForDBUpdate = tempOrder.UserChatID
		}
	}

	// Обновляем профиль только если это оператор для известного клиента, или пользователь для себя.
	// Водитель, создающий заказ, не должен обновлять чей-то профиль по номеру.
	if !isDriverFlow {
		if !userForPhoneUpdate.Phone.Valid || userForPhoneUpdate.Phone.String != normalizedPhone {
			if errDb := db.UpdateUserPhone(targetChatIDForDBUpdate, normalizedPhone); errDb != nil {
				log.Printf("handleOrderPhoneInput: Ошибка обновления телефона пользователя %d: %v", targetChatIDForDBUpdate, errDb)
			} else {
				log.Printf("handleOrderPhoneInput: Телефон пользователя %s для chatID %d обновлен в профиле.", normalizedPhone, targetChatIDForDBUpdate)
			}
		}
	}

	tempDataForKb := bh.Deps.SessionManager.GetTempOrder(chatID)
	if tempDataForKb.LocationPromptMessageID != 0 {
		bh.deleteMessageHelper(chatID, tempDataForKb.LocationPromptMessageID)
		tempDataForKb.LocationPromptMessageID = 0
		bh.Deps.SessionManager.UpdateTempOrder(chatID, tempDataForKb)
	} else {
		replyMarkupRemove := tgbotapi.NewRemoveKeyboard(true)
		msgToRemoveKb := tgbotapi.NewMessage(chatID, constants.InvisibleMessage)
		msgToRemoveKb.ReplyMarkup = replyMarkupRemove
		if sentKbRemovalMsg, errKb := bh.Deps.BotClient.Send(msgToRemoveKb); errKb == nil {
			go func(id int) { time.Sleep(200 * time.Millisecond); bh.deleteMessageHelper(chatID, id) }(sentKbRemovalMsg.MessageID)
		}
	}

	history := bh.Deps.SessionManager.GetHistory(chatID)
	if len(history) >= 2 && history[len(history)-2] == constants.STATE_ORDER_EDIT && tempOrder.ID != 0 {
		if errDb := db.UpdateOrderField(tempOrder.ID, "phone", normalizedPhone); errDb != nil {
			bh.sendErrorMessageHelper(chatID, botMenuMsgID, "❌ Ошибка сохранения телефона заказа.")
			return
		}
		bh.SendEditOrderMenu(chatID, botMenuMsgID)
	} else {
		bh.SendAddressInputMenu(chatID, botMenuMsgID)
	}
}
func (bh *BotHandler) handleOrderAddressTextInput(chatID int64, user models.User, address string, userMsgID int, botMenuMsgID int) {
	if len(address) < 5 {
		currentMsgID := botMenuMsgID
		sentErrorMsg, _ := bh.sendErrorMessageHelper(chatID, botMenuMsgID, "❌ Адрес должен содержать минимум 5 символов. Попробуйте снова.")
		if sentErrorMsg.MessageID != 0 {
			currentMsgID = sentErrorMsg.MessageID
		}
		bh.deleteMessageHelper(chatID, userMsgID)
		tempOrder := bh.Deps.SessionManager.GetTempOrder(chatID)
		tempOrder.CurrentMessageID = currentMsgID
		bh.Deps.SessionManager.UpdateTempOrder(chatID, tempOrder)
		return
	}
	tempOrder := bh.Deps.SessionManager.GetTempOrder(chatID)
	tempOrder.Address = address
	tempOrder.Latitude = 0
	tempOrder.Longitude = 0
	bh.Deps.SessionManager.UpdateTempOrder(chatID, tempOrder)
	bh.deleteMessageHelper(chatID, userMsgID)

	history := bh.Deps.SessionManager.GetHistory(chatID)
	if len(history) >= 2 && history[len(history)-2] == constants.STATE_ORDER_EDIT && tempOrder.ID != 0 {
		if errDb := db.UpdateOrderAddress(tempOrder.ID, address, 0, 0); errDb != nil {
			bh.sendErrorMessageHelper(chatID, botMenuMsgID, "❌ Ошибка сохранения адреса заказа.")
			return
		}
		bh.SendEditOrderMenu(chatID, botMenuMsgID)
	} else {
		bh.SendPhotoInputMenu(chatID, botMenuMsgID)
	}
}

func (bh *BotHandler) handleLocationMessage(chatID int64, user models.User, location *tgbotapi.Location, userMessageID int, botMenuMsgID int) {
	if location == nil {
		bh.deleteMessageHelper(chatID, userMessageID)
		return
	}
	errValidate := utils.ValidateLocation(location.Latitude, location.Longitude)
	if errValidate != nil {
		currentMsgID := botMenuMsgID
		sentErrorMsg, _ := bh.sendErrorMessageHelper(chatID, botMenuMsgID, "❌ Некорректные координаты. Попробуйте снова или введите адрес текстом.")
		if sentErrorMsg.MessageID != 0 {
			currentMsgID = sentErrorMsg.MessageID
		}
		bh.deleteMessageHelper(chatID, userMessageID)
		tempOrder := bh.Deps.SessionManager.GetTempOrder(chatID)
		tempOrder.CurrentMessageID = currentMsgID
		bh.Deps.SessionManager.UpdateTempOrder(chatID, tempOrder)
		bh.SendAddressInputMenu(chatID, currentMsgID)
		return
	}

	tempOrder := bh.Deps.SessionManager.GetTempOrder(chatID)
	tempOrder.Latitude = location.Latitude
	tempOrder.Longitude = location.Longitude
	tempOrder.Address = "🗺️ (Геометка)"
	if tempOrder.LocationPromptMessageID != 0 {
		bh.deleteMessageHelper(chatID, tempOrder.LocationPromptMessageID)
		tempOrder.LocationPromptMessageID = 0
	}
	bh.Deps.SessionManager.UpdateTempOrder(chatID, tempOrder)
	bh.deleteMessageHelper(chatID, userMessageID)

	history := bh.Deps.SessionManager.GetHistory(chatID)
	isEditingOrder := false
	if len(history) >= 2 && history[len(history)-2] == constants.STATE_ORDER_EDIT && tempOrder.ID != 0 {
		isEditingOrder = true
	} else if len(history) >= 3 && history[len(history)-3] == constants.STATE_ORDER_EDIT && tempOrder.ID != 0 {
		isEditingOrder = true
	}

	if isEditingOrder {
		if errDb := db.UpdateOrderAddress(tempOrder.ID, tempOrder.Address, tempOrder.Latitude, tempOrder.Longitude); errDb != nil {
			bh.sendErrorMessageHelper(chatID, botMenuMsgID, "❌ Ошибка сохранения адреса заказа.")
			return
		}
		bh.SendEditOrderMenu(chatID, botMenuMsgID)
	} else {
		bh.SendPhotoInputMenu(chatID, botMenuMsgID)
	}
}

func (bh *BotHandler) handleMediaMessage(chatID int64, user models.User, message *tgbotapi.Message, userMessageID int, botMenuMsgID int) {
	if message.MediaGroupID != "" {
		log.Printf("handleMediaMessage: Сообщение с MediaGroupID %s ошибочно попало в обработчик одиночных медиа. ChatID: %d. Будет проигнорировано здесь.", message.MediaGroupID, chatID)
		return
	}

	mediaType := utils.GetMediaType(message)
	if mediaType != "photo" && mediaType != "video" {
		log.Printf("handleMediaMessage: Получен не фото/видео контент (%s) в состоянии ожидания медиа. ChatID: %d, UserMsgID: %d", mediaType, chatID, userMessageID)
		currentMsgID := botMenuMsgID
		sentErrorMsg, _ := bh.sendErrorMessageHelper(chatID, botMenuMsgID, "❌ Пожалуйста, отправьте фото или видео, или используйте кнопки.")
		if sentErrorMsg.MessageID != 0 {
			currentMsgID = sentErrorMsg.MessageID
		}
		bh.deleteMessageHelper(chatID, userMessageID)
		tempOrder := bh.Deps.SessionManager.GetTempOrder(chatID)
		tempOrder.CurrentMessageID = currentMsgID
		bh.Deps.SessionManager.UpdateTempOrder(chatID, tempOrder)
		return
	}

	tempOrder := bh.Deps.SessionManager.GetTempOrder(chatID)
	var fileID string
	photoAddedThisTurn := false
	videoAddedThisTurn := false

	if mediaType == "photo" {
		if len(message.Photo) > 0 {
			fileID = message.Photo[len(message.Photo)-1].FileID
			if len(tempOrder.Photos) < constants.MAX_PHOTOS {
				tempOrder.Photos = append(tempOrder.Photos, fileID)
				photoAddedThisTurn = true
			} else {
				bh.sendErrorMessageHelper(chatID, botMenuMsgID, fmt.Sprintf("❌ Максимум %d фото.", constants.MAX_PHOTOS))
				bh.deleteMessageHelper(chatID, userMessageID)
				return
			}
		}
	} else if mediaType == "video" {
		if message.Video != nil {
			fileID = message.Video.FileID
			if len(tempOrder.Videos) < constants.MAX_VIDEOS {
				tempOrder.Videos = append(tempOrder.Videos, fileID)
				videoAddedThisTurn = true
			} else {
				bh.sendErrorMessageHelper(chatID, botMenuMsgID, fmt.Sprintf("❌ Максимум %d видео.", constants.MAX_VIDEOS))
				bh.deleteMessageHelper(chatID, userMessageID)
				return
			}
		}
	}

	if fileID == "" {
		log.Printf("handleMediaMessage: FileID не получен для одиночного медиа. ChatID: %d, UserMsgID: %d", chatID, userMessageID)
		bh.deleteMessageHelper(chatID, userMessageID)
		return
	}

	if photoAddedThisTurn || videoAddedThisTurn {
		bh.Deps.SessionManager.UpdateTempOrder(chatID, tempOrder)
	}
	bh.SendPhotoInputMenu(chatID, botMenuMsgID)
	log.Printf("handleMediaMessage (single): Удаление сообщения пользователя %d (одиночное медиа) сразу после обработки. ChatID: %d", userMessageID, chatID)
	bh.deleteMessageHelper(chatID, userMessageID)
}

func (bh *BotHandler) handleChatMessageInput(chatID int64, user models.User, messageText string, userMsgID int, botMenuMsgID int) {
	if len(messageText) < 1 {
		currentMsgID := botMenuMsgID
		sentErrorMsg, _ := bh.sendErrorMessageHelper(chatID, botMenuMsgID, "❌ Сообщение не может быть пустым.")
		if sentErrorMsg.MessageID != 0 {
			currentMsgID = sentErrorMsg.MessageID
		}
		bh.deleteMessageHelper(chatID, userMsgID)
		tempOrder := bh.Deps.SessionManager.GetTempOrder(chatID)
		tempOrder.CurrentMessageID = currentMsgID
		bh.Deps.SessionManager.UpdateTempOrder(chatID, tempOrder)
		return
	}
	conversationID := utils.GenerateUUID()
	operatorTargetChatID := bh.Deps.Config.OwnerChatID
	if bh.Deps.Config.GroupChatID != 0 {
		operatorTargetChatID = bh.Deps.Config.GroupChatID
	}

	_, err := db.AddChatMessage(chatID, operatorTargetChatID, messageText, true, conversationID)
	if err != nil {
		bh.sendErrorMessageHelper(chatID, botMenuMsgID, "❌ Ошибка отправки сообщения.")
		bh.deleteMessageHelper(chatID, userMsgID)
		return
	}

	operatorMsgText := fmt.Sprintf("💬 Новое сообщение от %s (ChatID: `%d`)\n[conv:%s]\n\n%s",
		utils.GetUserDisplayName(user), chatID, conversationID, messageText)

	bh.NotifyOperator(operatorTargetChatID, operatorMsgText)
	if bh.Deps.Config.GroupChatID != 0 && bh.Deps.Config.OwnerChatID != 0 && bh.Deps.Config.GroupChatID != bh.Deps.Config.OwnerChatID {
		bh.NotifyOperator(bh.Deps.Config.OwnerChatID, operatorMsgText)
	}

	bh.deleteMessageHelper(chatID, userMsgID)
	bh.SendChatConfirmation(chatID, botMenuMsgID)
}

func (bh *BotHandler) handlePhoneAwaitInput(chatID int64, user models.User, phoneInput string, userMsgID int, botMenuMsgID int) {
	normalizedPhone, errValidate := utils.ValidatePhoneNumber(phoneInput)
	if errValidate != nil {
		currentMsgID := botMenuMsgID
		sentErrorMsg, _ := bh.sendErrorMessageHelper(chatID, botMenuMsgID, fmt.Sprintf("❌ Неверный формат номера: %s.", errValidate.Error()))
		if sentErrorMsg.MessageID != 0 {
			currentMsgID = sentErrorMsg.MessageID
		}
		bh.deleteMessageHelper(chatID, userMsgID)
		tempOrder := bh.Deps.SessionManager.GetTempOrder(chatID)
		tempOrder.CurrentMessageID = currentMsgID
		bh.Deps.SessionManager.UpdateTempOrder(chatID, tempOrder)
		return
	}
	if errDb := db.UpdateUserPhone(chatID, normalizedPhone); errDb != nil {
		bh.sendErrorMessageHelper(chatID, botMenuMsgID, "❌ Ошибка сохранения вашего номера.")
		bh.deleteMessageHelper(chatID, userMsgID)
		return
	}
	user.Phone = sql.NullString{String: normalizedPhone, Valid: true}

	operatorMsgText := fmt.Sprintf("📲 Запрос на обратный звонок!\nКлиент: %s\nНомер: %s\n🔥 Оператор, свяжитесь в течение 5 минут!",
		utils.GetUserDisplayName(user), utils.FormatPhoneNumber(normalizedPhone))

	operatorTargetChatID := bh.Deps.Config.OwnerChatID
	if bh.Deps.Config.GroupChatID != 0 {
		operatorTargetChatID = bh.Deps.Config.GroupChatID
	}
	bh.NotifyOperator(operatorTargetChatID, operatorMsgText)
	if bh.Deps.Config.GroupChatID != 0 && bh.Deps.Config.OwnerChatID != 0 && bh.Deps.Config.GroupChatID != bh.Deps.Config.OwnerChatID {
		bh.NotifyOperator(bh.Deps.Config.OwnerChatID, operatorMsgText)
	}

	bh.deleteMessageHelper(chatID, userMsgID)
	bh.SendPhoneCallRequestConfirmation(chatID, normalizedPhone, botMenuMsgID)
}

func (bh *BotHandler) handleCostInput(chatID int64, user models.User, costStr string, userMsgID int, botMenuMsgID int) {
	if !utils.IsOperatorOrHigher(user.Role) {
		bh.sendErrorMessageHelper(chatID, botMenuMsgID, constants.AccessDeniedMessage)
		bh.deleteMessageHelper(chatID, userMsgID)
		return
	}
	cost, errConv := strconv.ParseFloat(strings.Replace(costStr, ",", ".", -1), 64)
	if errConv != nil || cost < 0 {
		bh.sendErrorMessageHelper(chatID, botMenuMsgID, "❌ Стоимость должна быть числом >= 0 (например, 1500 или 1250.5).")
		bh.deleteMessageHelper(chatID, userMsgID)
		return
	}
	tempOrder := bh.Deps.SessionManager.GetTempOrder(chatID)
	orderID := tempOrder.ID
	if orderID == 0 {
		bh.sendErrorMessageHelper(chatID, botMenuMsgID, "❌ ID заказа не найден для установки стоимости.")
		bh.deleteMessageHelper(chatID, userMsgID)
		return
	}

	errDb := db.UpdateOrderCostAndStatus(orderID, cost, constants.STATUS_AWAITING_CONFIRMATION)
	if errDb != nil {
		bh.sendErrorMessageHelper(chatID, botMenuMsgID, "❌ Ошибка обновления стоимости заказа.")
		bh.deleteMessageHelper(chatID, userMsgID)
		return
	}

	orderForClient, errGetOrder := db.GetOrderByID(int(orderID))
	if errGetOrder == nil && orderForClient.UserChatID != 0 {
		bh.SendClientCostConfirmation(orderForClient.UserChatID, int(orderID), cost)
	}

	bh.deleteMessageHelper(chatID, userMsgID)
	bh.sendInfoMessage(chatID, botMenuMsgID, fmt.Sprintf("✅ Стоимость %.0f ₽ для заказа №%d установлена. Клиент уведомлен.", cost, orderID), "manage_orders")
	bh.Deps.SessionManager.ClearState(chatID)
	bh.Deps.SessionManager.ClearTempOrder(chatID)
}

func (bh *BotHandler) handleCancelReasonInput(chatID int64, user models.User, reason string, userMsgID int, botMenuMsgID int) {
	if len(reason) < 5 {
		currentMsgID := botMenuMsgID
		sentErrorMsg, _ := bh.sendErrorMessageHelper(chatID, botMenuMsgID, "❌ Причина должна быть > 5 символов.")
		if sentErrorMsg.MessageID != 0 {
			currentMsgID = sentErrorMsg.MessageID
		}
		bh.deleteMessageHelper(chatID, userMsgID)
		tempOrder := bh.Deps.SessionManager.GetTempOrder(chatID)
		tempOrder.CurrentMessageID = currentMsgID
		bh.Deps.SessionManager.UpdateTempOrder(chatID, tempOrder)
		return
	}
	tempOrder := bh.Deps.SessionManager.GetTempOrder(chatID)
	orderID := tempOrder.ID
	if orderID == 0 {
		bh.sendErrorMessageHelper(chatID, botMenuMsgID, "❌ ID заказа не найден для отмены.")
		bh.deleteMessageHelper(chatID, userMsgID)
		return
	}

	errDb := db.UpdateOrderReasonAndStatus(orderID, reason, constants.STATUS_CANCELED)
	if errDb != nil {
		bh.sendErrorMessageHelper(chatID, botMenuMsgID, "❌ Ошибка отмены заказа.")
		bh.deleteMessageHelper(chatID, userMsgID)
		return
	}

	bh.deleteMessageHelper(chatID, userMsgID)
	orderData, _ := db.GetOrderByID(int(orderID))

	if orderData.UserChatID != 0 && orderData.UserChatID != chatID {
		bh.Deps.BotClient.Send(tgbotapi.NewMessage(orderData.UserChatID, fmt.Sprintf("⚠️ Заказ №%d был отменен оператором. Причина: %s", orderID, reason)))
	} else if orderData.UserChatID == chatID {
		bh.NotifyOperatorsOrderCancelledByClient(orderID, user, reason)
	}
	bh.sendInfoMessage(chatID, botMenuMsgID, fmt.Sprintf("❌ Заказ №%d отменён. Причина: %s", orderID, reason), "back_to_main")
	bh.Deps.SessionManager.ClearTempOrder(chatID)
	bh.Deps.SessionManager.ClearState(chatID)
}

func (bh *BotHandler) handleStaffAddInput(chatID int64, user models.User, currentState, text string, userMsgID int, botMenuMsgID int) {
	if !utils.IsRoleOrHigher(user.Role, constants.ROLE_MAINOPERATOR) {
		bh.sendErrorMessageHelper(chatID, botMenuMsgID, constants.AccessDeniedMessage)
		bh.deleteMessageHelper(chatID, userMsgID)
		return
	}
	bh.deleteMessageHelper(chatID, userMsgID)

	tempData := bh.Deps.SessionManager.GetTempOrder(chatID)
	var nextState, promptText, currentStepCallbackKey string

	switch currentState {
	case constants.STATE_STAFF_ADD_NAME:
		if len(text) < 2 {
			bh.sendErrorMessageHelper(chatID, botMenuMsgID, "Имя должно быть >1 символа.")
			return
		}
		tempData.Name = text
		nextState = constants.STATE_STAFF_ADD_SURNAME
		promptText = "👤 Введите фамилию сотрудника:"
		currentStepCallbackKey = "staff_add_prompt_name"
	case constants.STATE_STAFF_ADD_SURNAME:
		if len(text) < 2 {
			bh.sendErrorMessageHelper(chatID, botMenuMsgID, "Фамилия должна быть >1 символа.")
			return
		}
		tempData.Description = text // Используем Description для фамилии
		nextState = constants.STATE_STAFF_ADD_NICKNAME
		promptText = "📛 Введите позывной (никнейм) сотрудника (можно пропустить, отправив '-'):"
		currentStepCallbackKey = "staff_add_prompt_surname"
	case constants.STATE_STAFF_ADD_NICKNAME:
		if text == "-" {
			tempData.Subcategory = "" // Используем Subcategory для никнейма
		} else {
			tempData.Subcategory = text
		}
		nextState = constants.STATE_STAFF_ADD_PHONE
		promptText = "📱 Введите телефон сотрудника (например, +79001234567):"
		currentStepCallbackKey = "staff_add_prompt_nickname"
	case constants.STATE_STAFF_ADD_PHONE:
		phone, err := utils.ValidatePhoneNumber(text)
		if err != nil {
			bh.sendErrorMessageHelper(chatID, botMenuMsgID, "Неверный формат телефона. "+err.Error())
			return
		}
		tempData.Phone = phone
		nextState = constants.STATE_STAFF_ADD_CHATID
		promptText = "🆔 Введите Telegram ChatID сотрудника (числовой ID):"
		currentStepCallbackKey = "staff_add_prompt_phone"
	case constants.STATE_STAFF_ADD_CHATID:
		targetChatID, err := strconv.ParseInt(text, 10, 64)
		if err != nil {
			bh.sendErrorMessageHelper(chatID, botMenuMsgID, "ChatID должен быть числом.")
			return
		}
		exists, _ := db.UserExists(targetChatID)
		if exists {
			bh.sendErrorMessageHelper(chatID, botMenuMsgID, fmt.Sprintf("Пользователь с ChatID %d уже существует в системе. Если вы хотите изменить его данные, используйте меню редактирования.", targetChatID))
			bh.SendStaffAddPrompt(chatID, constants.STATE_STAFF_ADD_CHATID, "🆔 Введите другой Telegram ChatID сотрудника (числовой ID):", "staff_add_prompt_phone", botMenuMsgID)
			return
		}
		tempData.BlockTargetChatID = targetChatID
		nextState = constants.STATE_STAFF_ADD_CARD_NUMBER
		promptText = "💳 Введите номер карты сотрудника (16-19 цифр, без пробелов). Если карты нет, отправьте '-'."
		currentStepCallbackKey = "staff_add_prompt_chatid"
	default:
		log.Printf("handleStaffAddInput: Неизвестное состояние '%s'", currentState)
		bh.SendStaffMenu(chatID, botMenuMsgID)
		return
	}

	bh.Deps.SessionManager.UpdateTempOrder(chatID, tempData)
	bh.SendStaffAddPrompt(chatID, nextState, promptText, currentStepCallbackKey, botMenuMsgID)
}

func (bh *BotHandler) handleStaffCardNumberInput(chatID int64, user models.User, cardNumberInput string, userMsgID int, botMenuMsgID int, isAdding bool) {
	if !utils.IsRoleOrHigher(user.Role, constants.ROLE_MAINOPERATOR) {
		bh.sendErrorMessageHelper(chatID, botMenuMsgID, constants.AccessDeniedMessage)
		bh.deleteMessageHelper(chatID, userMsgID)
		return
	}
	bh.deleteMessageHelper(chatID, userMsgID)

	trimmedCardInput := strings.TrimSpace(cardNumberInput)
	var validCardNumber string
	if trimmedCardInput == "-" {
		validCardNumber = ""
	} else {
		re := regexp.MustCompile(`^[0-9]{16,19}$`)
		if !re.MatchString(trimmedCardInput) {
			bh.sendErrorMessageHelper(chatID, botMenuMsgID, "❌ Неверный формат номера карты. Введите 16-19 цифр без пробелов или '-' для пропуска.")
			if isAdding {
				bh.SendStaffAddPrompt(chatID, constants.STATE_STAFF_ADD_CARD_NUMBER, "💳 Введите номер карты (16-19 цифр) или '-' для пропуска:", "staff_add_prompt_chatid", botMenuMsgID)
			} else {
				targetChatID := bh.Deps.SessionManager.GetTempOrder(chatID).BlockTargetChatID
				bh.SendStaffEditFieldPrompt(chatID, targetChatID, "card_number", "💳 Введите новый номер карты (16-19 цифр) или '-' для пропуска:", botMenuMsgID)
			}
			return
		}
		validCardNumber = trimmedCardInput
	}

	tempData := bh.Deps.SessionManager.GetTempOrder(chatID)

	if isAdding {
		tempData.Payment = validCardNumber // Используем поле Payment для номера карты в TempOrder
		bh.Deps.SessionManager.UpdateTempOrder(chatID, tempData)
		bh.SendStaffRoleSelectionMenu(chatID, fmt.Sprintf("staff_add_role_final"), botMenuMsgID, "staff_add_prompt_card_number")
	} else {
		targetChatID := tempData.BlockTargetChatID
		if targetChatID == 0 {
			bh.sendErrorMessageHelper(chatID, botMenuMsgID, "Ошибка: не выбран сотрудник для редактирования карты.")
			bh.SendStaffMenu(chatID, botMenuMsgID)
			return
		}
		var valueToStore interface{}
		if validCardNumber == "" {
			valueToStore = sql.NullString{Valid: false}
		} else {
			valueToStore = sql.NullString{String: validCardNumber, Valid: true}
		}

		err := db.UpdateStaffField(targetChatID, "card_number", valueToStore)
		if err != nil {
			log.Printf("handleStaffCardNumberInput: Ошибка обновления номера карты для сотрудника %d: %v", targetChatID, err)
			bh.sendErrorMessageHelper(chatID, botMenuMsgID, "❌ Ошибка обновления номера карты сотрудника.")
			return
		}
		bh.sendInfoMessage(chatID, botMenuMsgID, "✅ Номер карты сотрудника успешно обновлен.", fmt.Sprintf("staff_info_%d", targetChatID))
		bh.Deps.SessionManager.ClearState(chatID)
		bh.Deps.SessionManager.ClearTempOrder(chatID)
		bh.SendStaffInfo(chatID, targetChatID, botMenuMsgID)
	}
}

func (bh *BotHandler) handleStaffEditInput(chatID int64, user models.User, currentState, text string, userMsgID int, botMenuMsgID int) {
	if !utils.IsRoleOrHigher(user.Role, constants.ROLE_MAINOPERATOR) {
		bh.sendErrorMessageHelper(chatID, botMenuMsgID, constants.AccessDeniedMessage)
		bh.deleteMessageHelper(chatID, userMsgID)
		return
	}
	bh.deleteMessageHelper(chatID, userMsgID)

	tempData := bh.Deps.SessionManager.GetTempOrder(chatID)
	targetChatID := tempData.BlockTargetChatID
	if targetChatID == 0 {
		bh.sendErrorMessageHelper(chatID, botMenuMsgID, "Ошибка: не выбран сотрудник для редактирования.")
		bh.SendStaffMenu(chatID, botMenuMsgID)
		return
	}

	var fieldToUpdate string
	var valueToUpdate interface{}
	var successMessage string

	switch currentState {
	case constants.STATE_STAFF_EDIT_NAME:
		if len(text) < 2 {
			bh.sendErrorMessageHelper(chatID, botMenuMsgID, "Имя должно быть >1 символа.")
			return
		}
		fieldToUpdate = "first_name"
		valueToUpdate = text
		successMessage = "Имя сотрудника обновлено."
	case constants.STATE_STAFF_EDIT_SURNAME:
		if len(text) < 2 {
			bh.sendErrorMessageHelper(chatID, botMenuMsgID, "Фамилия должна быть >1 символа.")
			return
		}
		fieldToUpdate = "last_name"
		valueToUpdate = text
		successMessage = "Фамилия сотрудника обновлена."
	case constants.STATE_STAFF_EDIT_NICKNAME:
		fieldToUpdate = "nickname"
		if text == "-" || text == "" {
			valueToUpdate = sql.NullString{Valid: false}
		} else {
			valueToUpdate = sql.NullString{String: text, Valid: true}
		}
		successMessage = "Позывной сотрудника обновлен."
	case constants.STATE_STAFF_EDIT_PHONE:
		fieldToUpdate = "phone"
		if text == "-" || text == "" {
			valueToUpdate = sql.NullString{Valid: false}
			successMessage = "Телефон сотрудника удален."
		} else {
			phone, err := utils.ValidatePhoneNumber(text)
			if err != nil {
				bh.sendErrorMessageHelper(chatID, botMenuMsgID, "Неверный формат телефона. "+err.Error())
				return
			}
			valueToUpdate = sql.NullString{String: phone, Valid: true}
			successMessage = "Телефон сотрудника обновлен."
		}
	default:
		log.Printf("handleStaffEditInput: Неизвестное состояние редактирования %s", currentState)
		bh.SendStaffInfo(chatID, targetChatID, botMenuMsgID)
		return
	}

	err := db.UpdateStaffField(targetChatID, fieldToUpdate, valueToUpdate)
	if err != nil {
		log.Printf("handleStaffEditInput: Ошибка обновления данных сотрудника %d: %v", targetChatID, err)
		bh.sendErrorMessageHelper(chatID, botMenuMsgID, "❌ Ошибка обновления данных сотрудника.")
		return
	}
	bh.sendInfoMessage(chatID, botMenuMsgID, "✅ "+successMessage, fmt.Sprintf("staff_info_%d", targetChatID))
	bh.Deps.SessionManager.ClearState(chatID)
	bh.Deps.SessionManager.ClearTempOrder(chatID)
	bh.SendStaffInfo(chatID, targetChatID, botMenuMsgID)
}

func (bh *BotHandler) handleStaffBlockReasonInput(chatID int64, user models.User, reason string, userMsgID int, botMenuMsgID int) {
	if !utils.IsRoleOrHigher(user.Role, constants.ROLE_MAINOPERATOR) {
		bh.sendErrorMessageHelper(chatID, botMenuMsgID, constants.AccessDeniedMessage)
		bh.deleteMessageHelper(chatID, userMsgID)
		return
	}
	bh.deleteMessageHelper(chatID, userMsgID)

	if len(reason) < 5 {
		bh.sendErrorMessageHelper(chatID, botMenuMsgID, "Причина блокировки должна быть > 5 символов.")
		return
	}
	tempData := bh.Deps.SessionManager.GetTempOrder(chatID)
	targetChatID := tempData.BlockTargetChatID
	if targetChatID == 0 {
		bh.sendErrorMessageHelper(chatID, botMenuMsgID, "Ошибка: сотрудник для блокировки не определен.")
		return
	}

	targetUser, errUser := db.GetUserByChatID(targetChatID)
	if errUser != nil {
		bh.sendErrorMessageHelper(chatID, botMenuMsgID, "Ошибка: не удалось найти сотрудника для блокировки.")
		return
	}
	if targetUser.Role == constants.ROLE_OWNER {
		bh.sendErrorMessageHelper(chatID, botMenuMsgID, "Этого сотрудника нельзя заблокировать.")
		bh.SendStaffInfo(chatID, targetChatID, botMenuMsgID)
		return
	}

	err := db.BlockUser(targetChatID, reason)
	if err != nil {
		log.Printf("handleStaffBlockReasonInput: Ошибка блокировки сотрудника %d: %v", targetChatID, err)
		bh.sendErrorMessageHelper(chatID, botMenuMsgID, "❌ Ошибка блокировки сотрудника.")
		return
	}
	bh.Deps.BotClient.Send(tgbotapi.NewMessage(targetChatID, fmt.Sprintf("🚫 Вы были заблокированы администратором. Причина: %s", reason)))
	bh.sendInfoMessage(chatID, botMenuMsgID, "✅ Сотрудник успешно заблокирован.", fmt.Sprintf("staff_info_%d", targetChatID))
	bh.Deps.SessionManager.ClearState(chatID)
	bh.Deps.SessionManager.ClearTempOrder(chatID)
	bh.SendStaffInfo(chatID, targetChatID, botMenuMsgID)
}

func (bh *BotHandler) handleUserBlockReasonInput(chatID int64, user models.User, reason string, userMsgID int, botMenuMsgID int) {
	if !utils.IsOperatorOrHigher(user.Role) {
		bh.sendErrorMessageHelper(chatID, botMenuMsgID, constants.AccessDeniedMessage)
		bh.deleteMessageHelper(chatID, userMsgID)
		return
	}
	bh.deleteMessageHelper(chatID, userMsgID)

	if len(reason) < 5 {
		bh.sendErrorMessageHelper(chatID, botMenuMsgID, "Причина блокировки должна быть > 5 символов.")
		return
	}
	tempData := bh.Deps.SessionManager.GetTempOrder(chatID)
	targetChatID := tempData.BlockTargetChatID
	if targetChatID == 0 {
		bh.sendErrorMessageHelper(chatID, botMenuMsgID, "Ошибка: пользователь для блокировки не определен.")
		return
	}
	tempData.BlockReason = reason
	bh.Deps.SessionManager.UpdateTempOrder(chatID, tempData)
	bh.handleBlockUserFinal(chatID, user, targetChatID, botMenuMsgID)
}

// NotifyOperator уведомляет оператора (или группу).
func (bh *BotHandler) NotifyOperator(operatorChatID int64, messageText string) {
	if operatorChatID == 0 {
		log.Println("NotifyOperator: operatorChatID равен 0, уведомление не отправлено.")
		return
	}
	msg := tgbotapi.NewMessage(operatorChatID, messageText)
	msg.ParseMode = tgbotapi.ModeMarkdown
	_, err := bh.Deps.BotClient.Send(msg)
	if err != nil {
		log.Printf("NotifyOperator: Ошибка отправки уведомления оператору %d: %v", operatorChatID, err)
	}
}

// NotifyOperatorsAboutNewOrder уведомляет всех операторов и группу о новом заказе.
func (bh *BotHandler) NotifyOperatorsAboutNewOrder(orderID int64, clientChatID int64) {
	orderDetails, err := db.GetFullOrderDetailsForNotification(orderID)
	if err != nil {
		log.Printf("NotifyOperatorsAboutNewOrder: Ошибка получения деталей заказа #%d: %v", orderID, err)
		return
	}
	var clientUser models.User
	var clientDisplayName string
	if clientChatID != 0 {
		clientUser, _ = db.GetUserByChatID(clientChatID)
		clientDisplayName = utils.GetUserDisplayName(clientUser)
	} else {
		// Это может быть случай, когда оператор создает заказ "на себя" или для анонимного клиента
		// В таком случае, используем имя из самого заказа
		if orderDetails.Name != "" {
			clientDisplayName = orderDetails.Name
		} else {
			clientDisplayName = "Клиент не указан"
		}
		// Если UserChatID в заказе тоже 0, это странно, но обрабатываем
		if orderDetails.UserChatID != 0 {
			clientDisplayName += fmt.Sprintf(" (внутр. ID: %d)", orderDetails.UserChatID)
		} else {
			clientDisplayName += " (ID не указан)"
		}

	}

	msgText := fmt.Sprintf(
		"🆕 Новый заказ №%d от %s\n"+
			"Категория: %s (%s)\n"+
			"Имя: %s\nДата: %s, Время: %s\n"+
			"Телефон: %s\nАдрес: %s\n"+
			"Описание: %s",
		orderID, utils.EscapeTelegramMarkdown(clientDisplayName),
		utils.EscapeTelegramMarkdown(constants.CategoryDisplayMap[orderDetails.Category]), utils.EscapeTelegramMarkdown(utils.GetDisplaySubcategory(orderDetails)),
		utils.EscapeTelegramMarkdown(orderDetails.Name), orderDetails.Date, orderDetails.Time,
		utils.FormatPhoneNumber(orderDetails.Phone), utils.EscapeTelegramMarkdown(orderDetails.Address),
		utils.EscapeTelegramMarkdown(orderDetails.Description),
	)
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("💰 Указать стоимость", fmt.Sprintf("set_cost_%d", orderID)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📋 Подробности", fmt.Sprintf("view_order_ops_%d", orderID)),
		),
	)

	operators, _ := db.GetUsersByRole(constants.ROLE_OPERATOR, constants.ROLE_MAINOPERATOR, constants.ROLE_OWNER)
	for _, op := range operators {
		msg := tgbotapi.NewMessage(op.ChatID, msgText)
		msg.ReplyMarkup = keyboard
		msg.ParseMode = tgbotapi.ModeMarkdown
		bh.Deps.BotClient.Send(msg)
	}
	if bh.Deps.Config.GroupChatID != 0 {
		msg := tgbotapi.NewMessage(bh.Deps.Config.GroupChatID, msgText)
		msg.ReplyMarkup = keyboard
		msg.ParseMode = tgbotapi.ModeMarkdown
		bh.Deps.BotClient.Send(msg)
	}
}

// NotifyOperatorsOrderCancelledByClient уведомляет операторов, что клиент отменил заказ.
func (bh *BotHandler) NotifyOperatorsOrderCancelledByClient(orderID int64, client models.User, reason string) {
	clientDisplayName := utils.GetUserDisplayName(client)
	msgText := fmt.Sprintf("❌ Клиент %s отменил заказ №%d.\nПричина: %s",
		utils.EscapeTelegramMarkdown(clientDisplayName), orderID, utils.EscapeTelegramMarkdown(reason))

	operators, _ := db.GetUsersByRole(constants.ROLE_OPERATOR, constants.ROLE_MAINOPERATOR, constants.ROLE_OWNER)
	for _, op := range operators {
		bh.NotifyOperator(op.ChatID, msgText)
	}
	if bh.Deps.Config.GroupChatID != 0 {
		bh.NotifyOperator(bh.Deps.Config.GroupChatID, msgText)
	}
}

// SendClientCostConfirmation отправляет клиенту предложение стоимости.
func (bh *BotHandler) SendClientCostConfirmation(clientChatID int64, orderID int, cost float64) {
	log.Printf("SendClientCostConfirmation: Уведомление клиента %d о стоимости заказа #%d", clientChatID, orderID)

	msgText := fmt.Sprintf("💰 Оператор рассчитал стоимость вашего заказа №%d: *%.0f ₽*.\n\nПожалуйста, подтвердите или отклоните предложенную стоимость.", orderID, cost)
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(fmt.Sprintf("✅ Да, согласен (%.0f ₽)", cost), fmt.Sprintf("accept_cost_%d", orderID)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("❌ Отказаться от стоимости", fmt.Sprintf("reject_cost_%d", orderID)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📋 Просмотреть детали заказа", fmt.Sprintf("view_order_%d", orderID)),
		),
	)
	msg := tgbotapi.NewMessage(clientChatID, msgText)
	msg.ReplyMarkup = keyboard
	msg.ParseMode = tgbotapi.ModeMarkdown
	_, err := bh.Deps.BotClient.Send(msg)

	if err != nil {
		log.Printf("SendClientCostConfirmation: Ошибка отправки уведомления клиенту %d: %v", clientChatID, err)
	}
}
