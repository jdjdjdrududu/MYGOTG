package handlers

import (
	"Original/internal/payments"
	"Original/internal/session"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time" // Added for Point 1 fix

	tgbotapi "github.com/OvyFlash/telegram-bot-api"

	"Original/internal/constants" //
	"Original/internal/db"
	"Original/internal/models" // Нужен для user / Needed for user
	"Original/internal/utils"  //
)

// dispatchOrderCallbacks маршрутизирует коллбэки, связанные с созданием и редактированием заказа.
// currentCommand - это уже определенная основная команда (например, "category_waste", "use_profile_name_for_order").
// parts - это оставшиеся части callback_data после извлечения currentCommand (например, ID заказа).
// data - это полная строка callback_data.
// Возвращает ID нового отправленного/отредактированного сообщения или 0.
func (bh *BotHandler) dispatchOrderCallbacks(currentCommand string, parts []string, data string, chatID int64, user models.User, originalMessageID int) int {
	log.Printf("[CALLBACK_ORDER] Диспетчер: Команда='%s', Части=%v, Data='%s', ChatID=%d", currentCommand, parts, data, chatID)

	var newMenuMessageID int = originalMessageID
	var sentMsg tgbotapi.Message // Объявляем здесь
	var errHelper error          // Объявляем здесь

	switch currentCommand {
	case constants.CALLBACK_PREFIX_PAY_ORDER:
		if len(parts) == 1 {
			newMenuMessageID = bh.handlePayOrder(chatID, user, parts[0], originalMessageID)
		} else {
			log.Printf("[CALLBACK_ORDER] Некорректный формат для '%s': %v. Ожидался ID заказа. ChatID=%d", currentCommand, parts, chatID)
			sentMsg, errHelper = bh.sendErrorMessageHelper(chatID, originalMessageID, "Ошибка формата: оплата заказа.")
			if errHelper == nil && sentMsg.MessageID != 0 {
				newMenuMessageID = sentMsg.MessageID
			}
		}
	case constants.CALLBACK_PREFIX_SELECT_HOUR: // например, select_hour_09
		if len(parts) == 1 {
			hourStr := parts[0]
			selectedHour, err := strconv.Atoi(hourStr)
			if err == nil && selectedHour >= 0 && selectedHour <= 23 {
				bh.SendMinuteSelectionMenu(chatID, selectedHour, originalMessageID)
				newMenuMessageID = bh.Deps.SessionManager.GetTempOrder(chatID).CurrentMessageID
			} else {
				log.Printf("[CALLBACK_ORDER] Ошибка конвертации часа для '%s': '%s'. ChatID=%d, err: %v", currentCommand, hourStr, chatID, err)
				sentMsg, errHelper = bh.sendErrorMessageHelper(chatID, originalMessageID, "Ошибка: неверный формат часа.")
				if errHelper == nil && sentMsg.MessageID != 0 {
					newMenuMessageID = sentMsg.MessageID
				}
			}
		} else {
			log.Printf("[CALLBACK_ORDER] Некорректный формат для '%s': %v. Ожидался ЧАС. ChatID=%d", currentCommand, parts, chatID)
			sentMsg, errHelper = bh.sendErrorMessageHelper(chatID, originalMessageID, "Ошибка: неверный формат команды выбора часа.")
			if errHelper == nil && sentMsg.MessageID != 0 {
				newMenuMessageID = sentMsg.MessageID
			}
		}
	case "category_waste", "category_demolition":
		categoryKey := strings.TrimPrefix(currentCommand, "category_")
		newMenuMessageID = bh.handleCategorySelection(chatID, user, []string{categoryKey}, originalMessageID)

	case "subcategory_construct", "subcategory_household", "subcategory_metal", "subcategory_junk",
		"subcategory_greenery", "subcategory_tires", "subcategory_other_waste",
		"subcategory_walls", "subcategory_partitions", "subcategory_floors", "subcategory_ceilings",
		"subcategory_plumbing", "subcategory_tiles", "subcategory_other_demo":
		subcategoryKey := strings.TrimPrefix(currentCommand, "subcategory_")
		newMenuMessageID = bh.handleSubcategorySelection(chatID, user, []string{subcategoryKey}, originalMessageID)

	case "confirm_order_description_placeholder":
		tempOrder := bh.Deps.SessionManager.GetTempOrder(chatID)
		log.Printf("[ORDER_HANDLER] Подтверждение текущего описания. ChatID=%d", chatID)
		history := bh.Deps.SessionManager.GetHistory(chatID)
		isEditingOrder := len(history) >= 2 && history[len(history)-2] == constants.STATE_ORDER_EDIT && tempOrder.ID != 0
		if isEditingOrder {
			bh.SendEditOrderMenu(chatID, originalMessageID)
		} else {
			bh.SendNameInputMenu(chatID, originalMessageID)
		}
		newMenuMessageID = bh.Deps.SessionManager.GetTempOrder(chatID).CurrentMessageID

	case "skip_order_description_placeholder":
		tempOrder := bh.Deps.SessionManager.GetTempOrder(chatID)
		tempOrder.Description = ""
		bh.Deps.SessionManager.UpdateTempOrder(chatID, tempOrder)
		log.Printf("[ORDER_HANDLER] Пропуск ввода описания. ChatID=%d", chatID)
		history := bh.Deps.SessionManager.GetHistory(chatID)
		isEditingOrder := len(history) >= 2 && history[len(history)-2] == constants.STATE_ORDER_EDIT && tempOrder.ID != 0
		if isEditingOrder {
			if err := db.UpdateOrderField(tempOrder.ID, "description", ""); err != nil {
				log.Printf("[ORDER_HANDLER] Ошибка очистки описания для заказа #%d: %v. ChatID=%d", tempOrder.ID, err, chatID)
				sentMsg, errHelper := bh.sendErrorMessageHelper(chatID, originalMessageID, "Ошибка очистки описания.")
				newMenuMessageID = originalMessageID
				if errHelper == nil && sentMsg.MessageID != 0 {
					newMenuMessageID = sentMsg.MessageID
				}
				bh.SendEditOrderMenu(chatID, newMenuMessageID)
				return newMenuMessageID
			}
			bh.SendEditOrderMenu(chatID, originalMessageID)
		} else {
			bh.SendNameInputMenu(chatID, originalMessageID)
		}
		newMenuMessageID = bh.Deps.SessionManager.GetTempOrder(chatID).CurrentMessageID

	case "use_profile_name_for_order":
		newMenuMessageID = bh.handleUseProfileName(chatID, user, originalMessageID)

	case "enter_another_name_for_order":
		bh.SendNameInputMenu(chatID, originalMessageID)
		newMenuMessageID = bh.Deps.SessionManager.GetTempOrder(chatID).CurrentMessageID

	case "confirm_order_name":
		newMenuMessageID = bh.handleConfirmOrderName(chatID, user, originalMessageID)
	case "confirm_order_phone":
		newMenuMessageID = bh.handleConfirmOrderPhone(chatID, user, originalMessageID)
	case "confirm_order_final": // parts: [ORDERID]
		if len(parts) == 1 {
			orderIDStr := parts[0]
			orderID, err := strconv.ParseInt(orderIDStr, 10, 64)
			if err == nil {
				newMenuMessageID = bh.handleConfirmOrderFinal(chatID, user, orderID, originalMessageID)
			} else {
				log.Printf("[CALLBACK_ORDER] Ошибка конвертации OrderID для 'confirm_order_final': '%s'. ChatID=%d", orderIDStr, chatID)
				sentMsg, errHelper = bh.sendErrorMessageHelper(chatID, originalMessageID, "Ошибка: неверный ID заказа.")
				if errHelper == nil && sentMsg.MessageID != 0 {
					newMenuMessageID = sentMsg.MessageID
				}
			}
		} else {
			log.Printf("[CALLBACK_ORDER] Некорректный формат для 'confirm_order_final': %v. Ожидался ID заказа. ChatID=%d", parts, chatID)
			sentMsg, errHelper = bh.sendErrorMessageHelper(chatID, originalMessageID, "Ошибка: неверный формат команды подтверждения.")
			if errHelper == nil && sentMsg.MessageID != 0 {
				newMenuMessageID = sentMsg.MessageID
			}
		}
	case "select_date_asap":
		newMenuMessageID = bh.handleDateTimeSelection(chatID, user, currentCommand, []string{"asap"}, originalMessageID)
	case "select_date": // select_date_DD_MonthString_YYYY -> parts: [DD, MonthString,<x_bin_880>]
		newMenuMessageID = bh.handleDateTimeSelection(chatID, user, currentCommand, parts, originalMessageID)
	case "select_time": // select_time_HH:MM -> parts: [HH:MM]
		newMenuMessageID = bh.handleDateTimeSelection(chatID, user, currentCommand, parts, originalMessageID)

	case "skip_photo_initial":
		newMenuMessageID = bh.handleSkipPhoto(chatID, user, originalMessageID)

	case "finish_photo_upload":
		newMenuMessageID = bh.handleFinishPhotoUpload(chatID, user, originalMessageID)

	case "reset_photo_upload":
		newMenuMessageID = bh.handleResetPhotoUpload(chatID, user, originalMessageID)

	case "view_uploaded_media":
		newMenuMessageID = bh.handleViewUploadedMedia(chatID, user, originalMessageID)

	case "payment_now", "payment_later":
		paymentType := strings.TrimPrefix(currentCommand, "payment_")
		newMenuMessageID = bh.handlePaymentSelection(chatID, user, []string{paymentType}, originalMessageID)

	case "change_order_phone":
		newMenuMessageID = bh.handleChangeOrderPhone(chatID, user, originalMessageID)

	case "send_location_prompt":
		bh.SendLocationPrompt(chatID, originalMessageID)
		newMenuMessageID = originalMessageID // sendOrEdit не используется напрямую в SendLocationPrompt для основного меню

	case "edit_order":
		if len(parts) == 1 {
			newMenuMessageID = bh.handleEditOrderStart(chatID, user, parts[0], originalMessageID)
		} else {
			log.Printf("[CALLBACK_ORDER] Некорректный формат для 'edit_order': %s. Ожидался ID заказа. ChatID=%d", data, chatID)
			sentMsg, errHelper = bh.sendErrorMessageHelper(chatID, originalMessageID, "Ошибка: неверный формат команды редактирования.")
			if errHelper == nil && sentMsg.MessageID != 0 {
				newMenuMessageID = sentMsg.MessageID
			}
		}
	case "date_page": // e.g., date_page_1
		if len(parts) == 1 {
			page, err := strconv.Atoi(parts[0])
			if err == nil {
				bh.SendDateSelectionMenu(chatID, originalMessageID, page)
				newMenuMessageID = bh.Deps.SessionManager.GetTempOrder(chatID).CurrentMessageID
			} else {
				log.Printf("[CALLBACK_ORDER] Ошибка конвертации страницы для 'date_page': '%s'. ChatID=%d, err: %v", parts[0], chatID, err)
				sentMsg, errHelper = bh.sendErrorMessageHelper(chatID, originalMessageID, "Ошибка: неверный формат страницы.")
				if errHelper == nil && sentMsg.MessageID != 0 {
					newMenuMessageID = sentMsg.MessageID
				}
			}
		} else {
			log.Printf("[CALLBACK_ORDER] Некорректный формат для 'date_page': %v. Ожидался номер страницы. ChatID=%d", parts, chatID)
			sentMsg, errHelper = bh.sendErrorMessageHelper(chatID, originalMessageID, "Ошибка: неверный формат команды пагинации.")
			if errHelper == nil && sentMsg.MessageID != 0 {
				newMenuMessageID = sentMsg.MessageID
			}
		}
	case "edit_field_description":
		if len(parts) == 1 {
			orderIDStr := parts[0]
			newMenuMessageID = bh.handleEditFieldSelection(chatID, user, "description", orderIDStr, originalMessageID)
		} else {
			log.Printf("[CALLBACK_ORDER] Некорректный формат для 'edit_field_description': %v. Ожидался ID заказа. ChatID=%d", parts, chatID)
			sentMsg, errHelper = bh.sendErrorMessageHelper(chatID, originalMessageID, "Ошибка формата: ред. описания заказа.")
			if errHelper == nil && sentMsg.MessageID != 0 {
				newMenuMessageID = sentMsg.MessageID
			}
		}
	case "edit_field_name":
		if len(parts) == 1 {
			orderIDStr := parts[0]
			newMenuMessageID = bh.handleEditFieldSelection(chatID, user, "name", orderIDStr, originalMessageID)
		} else {
			log.Printf("[CALLBACK_ORDER] Некорректный формат для 'edit_field_name': %v. Ожидался ID заказа. ChatID=%d", parts, chatID)
			sentMsg, errHelper = bh.sendErrorMessageHelper(chatID, originalMessageID, "Ошибка формата: ред. имени заказа.")
			if errHelper == nil && sentMsg.MessageID != 0 {
				newMenuMessageID = sentMsg.MessageID
			}
		}
	case "edit_field_subcategory":
		if len(parts) == 1 {
			orderIDStr := parts[0]
			newMenuMessageID = bh.handleEditFieldSelection(chatID, user, "subcategory", orderIDStr, originalMessageID)
		} else {
			log.Printf("[CALLBACK_ORDER] Некорректный формат для 'edit_field_subcategory': %v. Ожидался ID заказа. ChatID=%d", parts, chatID)
			sentMsg, errHelper = bh.sendErrorMessageHelper(chatID, originalMessageID, "Ошибка формата: ред. подкатегории заказа.")
			if errHelper == nil && sentMsg.MessageID != 0 {
				newMenuMessageID = sentMsg.MessageID
			}
		}
	case "edit_field_date":
		if len(parts) == 1 {
			orderIDStr := parts[0]
			newMenuMessageID = bh.handleEditFieldSelection(chatID, user, "date", orderIDStr, originalMessageID)
		} else {
			log.Printf("[CALLBACK_ORDER] Некорректный формат для 'edit_field_date': %v. ChatID=%d", parts, chatID)
			sentMsg, errHelper = bh.sendErrorMessageHelper(chatID, originalMessageID, "Ошибка формата: ред. даты заказа.")
			if errHelper == nil && sentMsg.MessageID != 0 {
				newMenuMessageID = sentMsg.MessageID
			}
		}
	case "edit_field_time":
		if len(parts) == 1 {
			orderIDStr := parts[0]
			newMenuMessageID = bh.handleEditFieldSelection(chatID, user, "time", orderIDStr, originalMessageID)
		} else {
			log.Printf("[CALLBACK_ORDER] Некорректный формат для 'edit_field_time': %v. ChatID=%d", parts, chatID)
			sentMsg, errHelper = bh.sendErrorMessageHelper(chatID, originalMessageID, "Ошибка формата: ред. времени заказа.")
			if errHelper == nil && sentMsg.MessageID != 0 {
				newMenuMessageID = sentMsg.MessageID
			}
		}
	case "edit_field_phone":
		if len(parts) == 1 {
			orderIDStr := parts[0]
			newMenuMessageID = bh.handleEditFieldSelection(chatID, user, "phone", orderIDStr, originalMessageID)
		} else {
			log.Printf("[CALLBACK_ORDER] Некорректный формат для 'edit_field_phone': %v. ChatID=%d", parts, chatID)
			sentMsg, errHelper = bh.sendErrorMessageHelper(chatID, originalMessageID, "Ошибка формата: ред. телефона заказа.")
			if errHelper == nil && sentMsg.MessageID != 0 {
				newMenuMessageID = sentMsg.MessageID
			}
		}
	case "edit_field_address":
		if len(parts) == 1 {
			orderIDStr := parts[0]
			newMenuMessageID = bh.handleEditFieldSelection(chatID, user, "address", orderIDStr, originalMessageID)
		} else {
			log.Printf("[CALLBACK_ORDER] Некорректный формат для 'edit_field_address': %v. ChatID=%d", parts, chatID)
			sentMsg, errHelper = bh.sendErrorMessageHelper(chatID, originalMessageID, "Ошибка формата: ред. адреса заказа.")
			if errHelper == nil && sentMsg.MessageID != 0 {
				newMenuMessageID = sentMsg.MessageID
			}
		}
	case "edit_field_media":
		if len(parts) == 1 {
			orderIDStr := parts[0]
			newMenuMessageID = bh.handleEditFieldSelection(chatID, user, "media", orderIDStr, originalMessageID)
		} else {
			log.Printf("[CALLBACK_ORDER] Некорректный формат для 'edit_field_media': %v. ChatID=%d", parts, chatID)
			sentMsg, errHelper = bh.sendErrorMessageHelper(chatID, originalMessageID, "Ошибка формата: ред. медиа заказа.")
			if errHelper == nil && sentMsg.MessageID != 0 {
				newMenuMessageID = sentMsg.MessageID
			}
		}
	case "edit_field_payment":
		if len(parts) == 1 {
			orderIDStr := parts[0]
			newMenuMessageID = bh.handleEditFieldSelection(chatID, user, "payment", orderIDStr, originalMessageID)
		} else {
			log.Printf("[CALLBACK_ORDER] Некорректный формат для 'edit_field_payment': %v. ChatID=%d", parts, chatID)
			sentMsg, errHelper = bh.sendErrorMessageHelper(chatID, originalMessageID, "Ошибка формата: ред. оплаты заказа.")
			if errHelper == nil && sentMsg.MessageID != 0 {
				newMenuMessageID = sentMsg.MessageID
			}
		}
	case "edit_field": // Общий обработчик, если предыдущие не сработали (должны)
		if len(parts) == 2 {
			newMenuMessageID = bh.handleEditFieldSelection(chatID, user, parts[0], parts[1], originalMessageID)
		} else {
			log.Printf("[CALLBACK_ORDER] Некорректный формат для 'edit_field': %s. Ожидалось ПОЛЕ_IDзаказа. ChatID=%d", data, chatID)
			sentMsg, errHelper = bh.sendErrorMessageHelper(chatID, originalMessageID, "Ошибка: неверный формат команды ред. поля.")
			if errHelper == nil && sentMsg.MessageID != 0 {
				newMenuMessageID = sentMsg.MessageID
			}
		}
	case "accept_cost":
		if len(parts) == 1 {
			newMenuMessageID = bh.handleAcceptCost(chatID, user, parts[0], originalMessageID)
		} else {
			log.Printf("[CALLBACK_ORDER] Некорректный формат для 'accept_cost': %s. Ожидался ID заказа. ChatID=%d", data, chatID)
			sentMsg, errHelper = bh.sendErrorMessageHelper(chatID, originalMessageID, "Ошибка: неверный формат команды согл. стоимости.")
			if errHelper == nil && sentMsg.MessageID != 0 {
				newMenuMessageID = sentMsg.MessageID
			}
		}
	case "reject_cost":
		if len(parts) == 1 {
			newMenuMessageID = bh.handleRejectCost(chatID, user, parts[0], originalMessageID)
		} else {
			log.Printf("[CALLBACK_ORDER] Некорректный формат для 'reject_cost': %s. Ожидался ID заказа. ChatID=%d", data, chatID)
			sentMsg, errHelper = bh.sendErrorMessageHelper(chatID, originalMessageID, "Ошибка: неверный формат команды откл. стоимости.")
			if errHelper == nil && sentMsg.MessageID != 0 {
				newMenuMessageID = sentMsg.MessageID
			}
		}
	case "cancel_order_operator", "cancel_order_confirm":
		actionType := strings.TrimPrefix(currentCommand, "cancel_order_")
		if len(parts) == 1 {
			newMenuMessageID = bh.handleCancelOrder(chatID, user, actionType, parts[0], originalMessageID)
		} else {
			log.Printf("[CALLBACK_ORDER] Некорректный формат для '%s': %s. Ожидался ID заказа. ChatID=%d", currentCommand, data, chatID)
			sentMsg, errHelper = bh.sendErrorMessageHelper(chatID, originalMessageID, "Ошибка: неверный формат команды отмены заказа.")
			if errHelper == nil && sentMsg.MessageID != 0 {
				newMenuMessageID = sentMsg.MessageID
			}
		}

		// --- НОВЫЕ ОБРАБОТЧИКИ ДЛЯ ОПЕРАТОРСКОГО ПОТОКА ---
	case constants.CALLBACK_PREFIX_OP_CONFIRM_ORDER_SET_COST: // parts: [ORDERID]
		if !utils.IsOperatorOrHigher(user.Role) {
			sentMsg, _ = bh.sendAccessDenied(chatID, originalMessageID)
			if sentMsg.MessageID != 0 {
				newMenuMessageID = sentMsg.MessageID
			}
			return newMenuMessageID
		}
		if len(parts) == 1 {
			orderID, err := strconv.ParseInt(parts[0], 10, 64)
			if err == nil {
				tempData := bh.Deps.SessionManager.GetTempOrder(chatID)
				tempData.ID = orderID
				tempData.OrderAction = "operator_creating_order" // Убедимся, что флаг установлен
				bh.Deps.SessionManager.UpdateTempOrder(chatID, tempData)
				// Состояние STATE_OP_ORDER_COST_INPUT будет установлено в SendOpOrderCostInputMenu
				bh.SendOpOrderCostInputMenu(chatID, orderID, originalMessageID)
				newMenuMessageID = bh.Deps.SessionManager.GetTempOrder(chatID).CurrentMessageID
			} else {
				log.Printf("[CALLBACK_ORDER] Ошибка конвертации OrderID для '%s': %v. ChatID=%d", currentCommand, parts, chatID)
				sentMsg, errHelper = bh.sendErrorMessageHelper(chatID, originalMessageID, "Ошибка: неверный ID заказа.")
				if errHelper == nil && sentMsg.MessageID != 0 {
					newMenuMessageID = sentMsg.MessageID
				}
			}
		} else {
			log.Printf("[CALLBACK_ORDER] Некорректный формат для '%s': %v. ChatID=%d", currentCommand, parts, chatID)
			sentMsg, errHelper = bh.sendErrorMessageHelper(chatID, originalMessageID, "Ошибка: неверный формат команды.")
			if errHelper == nil && sentMsg.MessageID != 0 {
				newMenuMessageID = sentMsg.MessageID
			}
		}

	case constants.CALLBACK_PREFIX_OP_SKIP_COST: // parts: [ORDERID]
		if !utils.IsOperatorOrHigher(user.Role) {
			sentMsg, _ = bh.sendAccessDenied(chatID, originalMessageID)
			if sentMsg.MessageID != 0 {
				newMenuMessageID = sentMsg.MessageID
			}
			return newMenuMessageID
		}
		if len(parts) == 1 {
			orderID, err := strconv.ParseInt(parts[0], 10, 64)
			if err == nil {
				tempData := bh.Deps.SessionManager.GetTempOrder(chatID)
				tempData.ID = orderID
				tempData.Cost.Valid = false // Стоимость не установлена или сброшена
				tempData.Cost.Float64 = 0
				bh.Deps.SessionManager.UpdateTempOrder(chatID, tempData)
				log.Printf("[CALLBACK_ORDER] Оператор %d пропустил ввод стоимости для заказа #%d.", chatID, orderID)
				// Переход к назначению исполнителей
				bh.SendOpAssignExecutorsMenu(chatID, orderID, originalMessageID)
				newMenuMessageID = bh.Deps.SessionManager.GetTempOrder(chatID).CurrentMessageID
			} else {
				log.Printf("[CALLBACK_ORDER] Ошибка конвертации OrderID для '%s': %v. ChatID=%d", currentCommand, parts, chatID)
				sentMsg, errHelper = bh.sendErrorMessageHelper(chatID, originalMessageID, "Ошибка: неверный ID заказа.")
				if errHelper == nil && sentMsg.MessageID != 0 {
					newMenuMessageID = sentMsg.MessageID
				}
			}
		} else {
			log.Printf("[CALLBACK_ORDER] Некорректный формат для '%s': %v. ChatID=%d", currentCommand, parts, chatID)
			sentMsg, errHelper = bh.sendErrorMessageHelper(chatID, originalMessageID, "Ошибка: неверный формат команды.")
			if errHelper == nil && sentMsg.MessageID != 0 {
				newMenuMessageID = sentMsg.MessageID
			}
		}

	case constants.CALLBACK_PREFIX_OP_CONFIRM_ORDER_ASSIGN_EXEC: // parts: [ORDERID]
		if !utils.IsOperatorOrHigher(user.Role) {
			sentMsg, _ = bh.sendAccessDenied(chatID, originalMessageID)
			if sentMsg.MessageID != 0 {
				newMenuMessageID = sentMsg.MessageID
			}
			return newMenuMessageID
		}
		if len(parts) == 1 {
			orderID, err := strconv.ParseInt(parts[0], 10, 64)
			if err == nil {
				tempData := bh.Deps.SessionManager.GetTempOrder(chatID)
				tempData.ID = orderID
				bh.Deps.SessionManager.UpdateTempOrder(chatID, tempData)
				// Переход к меню назначения исполнителей
				bh.SendOpAssignExecutorsMenu(chatID, orderID, originalMessageID)
				newMenuMessageID = bh.Deps.SessionManager.GetTempOrder(chatID).CurrentMessageID
			} else {
				log.Printf("[CALLBACK_ORDER] Ошибка конвертации OrderID для '%s': %v. ChatID=%d", currentCommand, parts, chatID)
				sentMsg, errHelper = bh.sendErrorMessageHelper(chatID, originalMessageID, "Ошибка: неверный ID заказа.")
				if errHelper == nil && sentMsg.MessageID != 0 {
					newMenuMessageID = sentMsg.MessageID
				}
			}
		} else {
			log.Printf("[CALLBACK_ORDER] Некорректный формат для '%s': %v. ChatID=%d", currentCommand, parts, chatID)
			sentMsg, errHelper = bh.sendErrorMessageHelper(chatID, originalMessageID, "Ошибка: неверный формат команды.")
			if errHelper == nil && sentMsg.MessageID != 0 {
				newMenuMessageID = sentMsg.MessageID
			}
		}

	case constants.CALLBACK_PREFIX_OP_SKIP_ASSIGN_EXEC: // parts: [ORDERID]
		if !utils.IsOperatorOrHigher(user.Role) {
			sentMsg, _ = bh.sendAccessDenied(chatID, originalMessageID)
			if sentMsg.MessageID != 0 {
				newMenuMessageID = sentMsg.MessageID
			}
			return newMenuMessageID
		}
		if len(parts) == 1 {
			orderID, err := strconv.ParseInt(parts[0], 10, 64)
			if err == nil {
				tempData := bh.Deps.SessionManager.GetTempOrder(chatID)
				tempData.ID = orderID
				// Исполнители не назначаются
				bh.Deps.SessionManager.UpdateTempOrder(chatID, tempData)
				log.Printf("[CALLBACK_ORDER] Оператор %d пропустил назначение исполнителей для заказа #%d.", chatID, orderID)
				// Переход к финальному подтверждению
				bh.SendOpOrderFinalConfirmMenu(chatID, orderID, originalMessageID)
				newMenuMessageID = bh.Deps.SessionManager.GetTempOrder(chatID).CurrentMessageID
			} else {
				log.Printf("[CALLBACK_ORDER] Ошибка конвертации OrderID для '%s': %v. ChatID=%d", currentCommand, parts, chatID)
				sentMsg, errHelper = bh.sendErrorMessageHelper(chatID, originalMessageID, "Ошибка: неверный ID заказа.")
				if errHelper == nil && sentMsg.MessageID != 0 {
					newMenuMessageID = sentMsg.MessageID
				}
			}
		} else {
			log.Printf("[CALLBACK_ORDER] Некорректный формат для '%s': %v. ChatID=%d", currentCommand, parts, chatID)
			sentMsg, errHelper = bh.sendErrorMessageHelper(chatID, originalMessageID, "Ошибка: неверный формат команды.")
			if errHelper == nil && sentMsg.MessageID != 0 {
				newMenuMessageID = sentMsg.MessageID
			}
		}
	case constants.CALLBACK_PREFIX_OP_FINALIZE_ORDER_CREATION: // parts: [ORDERID]
		if !utils.IsOperatorOrHigher(user.Role) {
			sentMsg, _ = bh.sendAccessDenied(chatID, originalMessageID)
			if sentMsg.MessageID != 0 {
				newMenuMessageID = sentMsg.MessageID
			}
			return newMenuMessageID
		}
		if len(parts) == 1 {
			orderID, err := strconv.ParseInt(parts[0], 10, 64)
			if err == nil {
				bh.SendOpOrderFinalConfirmMenu(chatID, orderID, originalMessageID)
				newMenuMessageID = bh.Deps.SessionManager.GetTempOrder(chatID).CurrentMessageID
			} else {
				log.Printf("[CALLBACK_ORDER] Ошибка конвертации OrderID для '%s': %v. ChatID=%d", currentCommand, parts, chatID)
				sentMsg, errHelper = bh.sendErrorMessageHelper(chatID, originalMessageID, "Ошибка: неверный ID заказа.")
				if errHelper == nil && sentMsg.MessageID != 0 {
					newMenuMessageID = sentMsg.MessageID
				}
			}
		} else {
			log.Printf("[CALLBACK_ORDER] Некорректный формат для '%s': %v. ChatID=%d", currentCommand, parts, chatID)
			sentMsg, errHelper = bh.sendErrorMessageHelper(chatID, originalMessageID, "Ошибка: неверный формат команды.")
			if errHelper == nil && sentMsg.MessageID != 0 {
				newMenuMessageID = sentMsg.MessageID
			}
		}

	default:
		log.Printf("[CALLBACK_ORDER] ОШИБКА: Неизвестная команда '%s' передана в dispatchOrderCallbacks. Полные данные: '%s', ChatID=%d", currentCommand, data, chatID)
		sentMsg, errHelper = bh.sendErrorMessageHelper(chatID, originalMessageID, "Неизвестная команда (внутренняя).")
		if errHelper == nil && sentMsg.MessageID != 0 {
			newMenuMessageID = sentMsg.MessageID
		}
	}
	log.Printf("[CALLBACK_ORDER] Диспетчер коллбэков заказа завершен. Команда='%s', ChatID=%d, ID нового меню=%d", currentCommand, chatID, newMenuMessageID)
	return newMenuMessageID
}

// handleAcceptCost обрабатывает подтверждение клиентом предложенной стоимости.
func (bh *BotHandler) handleAcceptCost(chatID int64, user models.User, orderIDStr string, originalMessageID int) int {
	log.Printf("[ORDER_HANDLER] Принятие стоимости заказа: OrderIDStr=%s, ChatID=%d", orderIDStr, chatID)
	var newMenuMessageID int = originalMessageID
	var sentMsg tgbotapi.Message
	var errHelper error

	orderID, err := strconv.ParseInt(orderIDStr, 10, 64)
	if err != nil {
		log.Printf("[ORDER_HANDLER] Ошибка: неверный OrderID в 'accept_cost': '%s'. ChatID=%d", orderIDStr, chatID)
		sentMsg, errHelper = bh.sendErrorMessageHelper(chatID, originalMessageID, "Неверный ID заказа.")
		if errHelper == nil && sentMsg.MessageID != 0 {
			newMenuMessageID = sentMsg.MessageID
		}
		return newMenuMessageID
	}

	orderData, errDb := db.GetOrderByID(int(orderID))
	if errDb != nil {
		log.Printf("[ORDER_HANDLER] Ошибка загрузки заказа #%d: %v. ChatID=%d", orderID, errDb, chatID)
		sentMsg, errHelper = bh.sendErrorMessageHelper(chatID, originalMessageID, "Ошибка загрузки заказа.")
		if errHelper == nil && sentMsg.MessageID != 0 {
			newMenuMessageID = sentMsg.MessageID
		}
		return newMenuMessageID
	}

	if orderData.UserChatID != chatID {
		log.Printf("[ORDER_HANDLER] Ошибка: Попытка принять стоимость для заказа #%d не клиентом. ChatID запроса: %d", orderID, chatID)
		sentMsg, errHelper = bh.sendErrorMessageHelper(chatID, originalMessageID, "Это действие доступно только клиенту заказа.")
		if errHelper == nil && sentMsg.MessageID != 0 {
			newMenuMessageID = sentMsg.MessageID
		}
		return newMenuMessageID
	}
	if orderData.Status != constants.STATUS_AWAITING_CONFIRMATION {
		log.Printf("[ORDER_HANDLER] Попытка принять стоимость для заказа #%d не в статусе AWAITING_CONFIRMATION (статус: %s). ChatID=%d", orderID, orderData.Status, chatID)
		sentMsg, errHelper = bh.sendErrorMessageHelper(chatID, originalMessageID, "Стоимость для этого заказа уже подтверждена или заказ в другом статусе.")
		if errHelper == nil && sentMsg.MessageID != 0 {
			newMenuMessageID = sentMsg.MessageID
		}
		return newMenuMessageID
	}

	if orderData.Payment == "now" {
		log.Printf("[ORDER_HANDLER] Клиент ChatID=%d подтвердил стоимость для заказа #%d. Метод оплаты: 'now'. Переход к оплате.", chatID, orderID)
		errDb = db.UpdateOrderStatus(orderID, constants.STATUS_AWAITING_PAYMENT)
		if errDb != nil {
			log.Printf("[ORDER_HANDLER] Ошибка обновления статуса заказа #%d на AWAITING_PAYMENT: %v. ChatID=%d", orderID, errDb, chatID)
			sentMsg, errHelper = bh.sendErrorMessageHelper(chatID, originalMessageID, "Ошибка подтверждения стоимости.")
			if errHelper == nil && sentMsg.MessageID != 0 {
				newMenuMessageID = sentMsg.MessageID
			}
			return newMenuMessageID
		}
		bh.sendPaymentMenu(chatID, int(orderID), originalMessageID)
		newMenuMessageID = bh.Deps.SessionManager.GetTempOrder(chatID).CurrentMessageID
	} else {
		errDb = db.UpdateOrderStatus(orderID, constants.STATUS_INPROGRESS)
		if errDb != nil {
			log.Printf("[ORDER_HANDLER] Ошибка обновления статуса заказа #%d на INPROGRESS: %v. ChatID=%d", orderID, errDb, chatID)
			sentMsg, errHelper = bh.sendErrorMessageHelper(chatID, originalMessageID, "Ошибка подтверждения стоимости.")
			if errHelper == nil && sentMsg.MessageID != 0 {
				newMenuMessageID = sentMsg.MessageID
			}
			return newMenuMessageID
		}

		log.Printf("[ORDER_HANDLER] Клиент ChatID=%d подтвердил стоимость для заказа #%d. Уведомление операторам...", chatID, orderID)
		operatorMsgText := fmt.Sprintf("✅ Клиент %s (ChatID: `%d`) подтвердил стоимость для заказа №%d.\nЗаказ переведен в статус '%s'.",
			utils.GetUserDisplayName(user), chatID, orderID, constants.StatusDisplayMap[constants.STATUS_INPROGRESS])
		bh.NotifyOperatorsAndGroup(operatorMsgText)

		clientConfirmText := fmt.Sprintf("✅ Заказ №%d подтверждён и скоро будет в работе! 🚚", orderID)
		clientKeyboard := tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("📋 Мои заказы", "my_orders_page_0")),
			tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("🏢 Главное меню", "back_to_main")),
		)
		sentClientMsg, errClientSend := bh.sendOrEditMessageHelper(chatID, originalMessageID, clientConfirmText, &clientKeyboard, "")
		if errClientSend != nil {
			log.Printf("[ORDER_HANDLER] Ошибка отправки сообщения клиенту о подтверждении стоимости заказа #%d: %v", orderID, errClientSend)
			newMenuMessageID = 0
		} else {
			newMenuMessageID = sentClientMsg.MessageID
		}
	}

	bh.Deps.SessionManager.ClearTempOrder(chatID)
	bh.Deps.SessionManager.ClearState(chatID)
	return newMenuMessageID
}

// sendPaymentMenu отправляет клиенту меню с кнопкой для оплаты.
func (bh *BotHandler) sendPaymentMenu(chatID int64, orderID int, messageIDToEdit int) {
	log.Printf("sendPaymentMenu: Заказ #%d готов к оплате. ChatID: %d, MessageID: %d", orderID, chatID, messageIDToEdit)

	orderData, err := db.GetOrderByID(orderID)
	if err != nil {
		bh.sendErrorMessageHelper(chatID, messageIDToEdit, "❌ Ошибка получения данных для оплаты.")
		return
	}

	if !orderData.Cost.Valid || orderData.Cost.Float64 <= 0 {
		bh.sendErrorMessageHelper(chatID, messageIDToEdit, "❌ Не установлена стоимость для оплаты.")
		return
	}

	text := fmt.Sprintf(
		"💳 *Переход к оплате*\n\n"+
			"Заказ: №%d\n"+
			"Сумма к оплате: *%.2f ₽*\n\n"+
			"Нажмите на кнопку ниже, чтобы перейти на страницу безопасной оплаты.",
		orderID,
		orderData.Cost.Float64,
	)

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("💳 Оплатить заказ", fmt.Sprintf("%s_%d", constants.CALLBACK_PREFIX_PAY_ORDER, orderID)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔙 Позже", "my_orders_page_0"),
		),
	)

	_, errSend := bh.sendOrEditMessageHelper(chatID, messageIDToEdit, text, &keyboard, tgbotapi.ModeMarkdown)
	if errSend != nil {
		log.Printf("sendPaymentMenu: ошибка отправки меню оплаты: %v", errSend)
	}
}

// handlePayOrder обрабатывает нажатие кнопки "Оплатить".
func (bh *BotHandler) handlePayOrder(chatID int64, user models.User, orderIDStr string, originalMessageID int) int {
	log.Printf("[ORDER_HANDLER] Начало процесса оплаты: OrderIDStr=%s, ChatID=%d", orderIDStr, chatID)
	var newMenuMessageID int = originalMessageID

	orderID, err := strconv.ParseInt(orderIDStr, 10, 64)
	if err != nil {
		log.Printf("[ORDER_HANDLER] Ошибка: неверный OrderID в '%s': '%s'. ChatID=%d", constants.CALLBACK_PREFIX_PAY_ORDER, orderIDStr, chatID)
		return newMenuMessageID
	}

	orderData, errDb := db.GetOrderByID(int(orderID))
	if errDb != nil {
		bh.sendErrorMessageHelper(chatID, originalMessageID, "❌ Ошибка получения данных заказа для оплаты.")
		return newMenuMessageID
	}

	if orderData.UserChatID != chatID {
		bh.sendAccessDenied(chatID, originalMessageID)
		return newMenuMessageID
	}

	if orderData.Status != constants.STATUS_AWAITING_PAYMENT {
		bh.sendInfoMessage(chatID, originalMessageID, "Этот заказ не ожидает оплаты.", "my_orders_page_0")
		return newMenuMessageID
	}

	if !orderData.Cost.Valid || orderData.Cost.Float64 <= 0 {
		bh.sendErrorMessageHelper(chatID, originalMessageID, "❌ Стоимость заказа не установлена, оплата невозможна.")
		return newMenuMessageID
	}

	// --- Логика создания платежа (YooKassa) ---
	shopID := bh.Deps.Config.YooKassaShopID
	secretKey := bh.Deps.Config.YooKassaSecretKey

	if shopID == "" || secretKey == "" {
		log.Printf("handlePayOrder: YooKassa Shop ID или Secret Key не установлены в конфигурации.")
		bh.sendErrorMessageHelper(chatID, originalMessageID, "❌ Ошибка конфигурации платежной системы.")
		return newMenuMessageID
	}

	description := fmt.Sprintf("Оплата заказа №%d", orderID)
	amount := orderData.Cost.Float64
	// A simple return URL, could be improved to lead back to the bot
	returnURL := fmt.Sprintf("https://t.me/%s", bh.Deps.Config.BotUsername)

	// --- НАЧАЛО ИЗМЕНЕНИЯ: Добавляем телефон клиента в вызов ---
	clientPhone := orderData.Phone
	if clientPhone == "" {
		// Фоллбэк, если в заказе нет телефона, хотя он должен быть на этом этапе
		clientPhone = "79000000000"
		log.Printf("handlePayOrder: ВНИМАНИЕ! Телефон для заказа #%d не найден. Используется номер-заглушка.", orderID)
	}

	paymentURL, errPay := payments.CreatePaymentLink(shopID, secretKey, orderID, amount, "RUB", description, returnURL, clientPhone)
	// --- КОНЕЦ ИЗМЕНЕНИЯ ---

	if errPay != nil {
		log.Printf("handlePayOrder: Ошибка создания платежной ссылки для заказа #%d: %v", orderID, errPay)
		bh.sendErrorMessageHelper(chatID, originalMessageID, "❌ Не удалось создать ссылку на оплату. Пожалуйста, попробуйте позже.")
		return newMenuMessageID
	}

	// --- Отправка ссылки на оплату ---
	text := "✅ Ваша ссылка на оплату готова!\n\nНажмите кнопку ниже, чтобы перейти к странице безопасной оплаты."
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonURL("Перейти к оплате", paymentURL),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔙 Назад к моим заказам", "my_orders_page_0"),
		),
	)

	sentMsg, errSend := bh.sendOrEditMessageHelper(chatID, originalMessageID, text, &keyboard, "")
	if errSend != nil {
		log.Printf("handlePayOrder: Ошибка отправки ссылки на оплату: %v", errSend)
	} else {
		newMenuMessageID = sentMsg.MessageID
	}

	return newMenuMessageID
}

// handleCategorySelection обрабатывает выбор категории заказа.
// Возвращает ID нового или отредактированного сообщения меню.
func (bh *BotHandler) handleCategorySelection(chatID int64, user models.User, parts []string, originalMessageID int) int {
	log.Printf("[ORDER_HANDLER] Выбор категории: ChatID=%d, Parts=%v", chatID, parts)
	var newMenuMessageID int = originalMessageID
	var sentMsg tgbotapi.Message // Объявляем здесь
	var errHelper error          // Объявляем здесь

	if len(parts) < 1 {
		log.Printf("[ORDER_HANDLER] Ошибка: недостаточно частей в коллбэке категории: %v. ChatID=%d", parts, chatID)
		sentMsg, errHelper = bh.sendErrorMessageHelper(chatID, originalMessageID, "Ошибка выбора категории.")
		if errHelper == nil && sentMsg.MessageID != 0 {
			newMenuMessageID = sentMsg.MessageID
		}
		return newMenuMessageID
	}
	categoryKey := parts[0]
	var categoryToSet string

	switch categoryKey {
	case "waste":
		categoryToSet = constants.CAT_WASTE
	case "demolition":
		categoryToSet = constants.CAT_DEMOLITION
	default:
		log.Printf("[ORDER_HANDLER] Ошибка: выбрана неизвестная категория '%s'. ChatID=%d", categoryKey, chatID)
		sentMsg, errHelper = bh.sendErrorMessageHelper(chatID, originalMessageID, "Выбрана неизвестная категория.")
		if errHelper == nil && sentMsg.MessageID != 0 {
			newMenuMessageID = sentMsg.MessageID
		}
		return newMenuMessageID
	}

	bh.SendSubcategoryMenu(chatID, categoryToSet, originalMessageID)
	newMenuMessageID = bh.Deps.SessionManager.GetTempOrder(chatID).CurrentMessageID
	return newMenuMessageID
}

// handleSubcategorySelection обрабатывает выбор подкатегории заказа.
// Возвращает ID нового или отредактированного сообщения меню.
func (bh *BotHandler) handleSubcategorySelection(chatID int64, user models.User, parts []string, originalMessageID int) int {
	log.Printf("[ORDER_HANDLER] Выбор подкатегории: ChatID=%d, Parts=%v", chatID, parts)
	var newMenuMessageID int = originalMessageID
	var sentMsg tgbotapi.Message // Объявляем здесь
	var errHelper error          // Объявляем здесь

	if len(parts) < 1 {
		log.Printf("[ORDER_HANDLER] Ошибка: недостаточно частей в коллбэке подкатегории: %v. ChatID=%d", parts, chatID)
		sentMsg, errHelper = bh.sendErrorMessageHelper(chatID, originalMessageID, "Ошибка выбора подкатегории.")
		if errHelper == nil && sentMsg.MessageID != 0 {
			newMenuMessageID = sentMsg.MessageID
		}
		return newMenuMessageID
	}
	subcategoryKey := parts[0]

	tempOrder := bh.Deps.SessionManager.GetTempOrder(chatID)
	if tempOrder.Category == "" {
		log.Printf("[ORDER_HANDLER] Ошибка: категория не установлена перед выбором подкатегории. ChatID=%d", chatID)
		sentMsg, errHelper = bh.sendErrorMessageHelper(chatID, originalMessageID, "Сначала выберите категорию.")
		currentMsgIDForCategoryMenu := originalMessageID
		if errHelper == nil && sentMsg.MessageID != 0 {
			currentMsgIDForCategoryMenu = sentMsg.MessageID
		}
		bh.SendCategoryMenu(chatID, user.FirstName, currentMsgIDForCategoryMenu)
		newMenuMessageID = bh.Deps.SessionManager.GetTempOrder(chatID).CurrentMessageID
		return newMenuMessageID
	}

	tempOrder.Subcategory = subcategoryKey
	bh.Deps.SessionManager.UpdateTempOrder(chatID, tempOrder)

	history := bh.Deps.SessionManager.GetHistory(chatID)
	isEditingOrder := len(history) >= 2 && history[len(history)-2] == constants.STATE_ORDER_EDIT && tempOrder.ID != 0

	if isEditingOrder {
		log.Printf("[ORDER_HANDLER] Редактирование подкатегории для заказа #%d на '%s'. ChatID=%d", tempOrder.ID, subcategoryKey, chatID)
		if err := db.UpdateOrderField(tempOrder.ID, "subcategory", subcategoryKey); err != nil {
			log.Printf("[ORDER_HANDLER] Ошибка сохранения подкатегории для заказа #%d: %v. ChatID=%d", tempOrder.ID, err, chatID)
			sentMsg, errHelper := bh.sendErrorMessageHelper(chatID, originalMessageID, "Ошибка сохранения подкатегории.")
			currentMsgIDForEditMenu := originalMessageID
			if errHelper == nil && sentMsg.MessageID != 0 {
				currentMsgIDForEditMenu = sentMsg.MessageID
			}
			bh.SendEditOrderMenu(chatID, currentMsgIDForEditMenu)
			newMenuMessageID = bh.Deps.SessionManager.GetTempOrder(chatID).CurrentMessageID
			return newMenuMessageID
		}
		bh.SendEditOrderMenu(chatID, originalMessageID)
	} else {
		log.Printf("[ORDER_HANDLER] Переход к вводу описания после выбора подкатегории '%s'. ChatID=%d", subcategoryKey, chatID)
		bh.SendDescriptionInputMenu(chatID, originalMessageID)
	}
	newMenuMessageID = bh.Deps.SessionManager.GetTempOrder(chatID).CurrentMessageID
	return newMenuMessageID
}

// handleUseProfileName обрабатывает использование имени из профиля для заказа.
func (bh *BotHandler) handleUseProfileName(chatID int64, user models.User, originalMessageID int) int {
	log.Printf("[ORDER_HANDLER] Использование имени из профиля для заказа. ChatID=%d", chatID)
	var newMenuMessageID int = originalMessageID
	var sentMsg tgbotapi.Message // Объявляем здесь
	var errHelper error          // Объявляем здесь

	tempOrder := bh.Deps.SessionManager.GetTempOrder(chatID)
	nameToUse := user.FirstName // По умолчанию имя текущего пользователя (оператора или клиента)

	// Если оператор создает для клиента, и у клиента есть имя
	if tempOrder.OrderAction == "operator_creating_order" && tempOrder.UserChatID != chatID && tempOrder.UserChatID != 0 {
		clientUser, clientFound := bh.getUserFromDB(tempOrder.UserChatID)
		if clientFound && clientUser.FirstName != "" {
			nameToUse = clientUser.FirstName
		} else if clientFound { // Клиент есть, но имя пустое
			log.Printf("[ORDER_HANDLER] Имя в профиле клиента (UserChatID: %d) не заполнено. Оператор введет имя.", tempOrder.UserChatID)
			sentMsg, errHelper = bh.sendErrorMessageHelper(chatID, originalMessageID, "Имя клиента в профиле не заполнено. Введите имя вручную.")
			bh.Deps.SessionManager.SetState(chatID, constants.STATE_ORDER_NAME)
			currentMsgIDForNameInput := originalMessageID
			if errHelper == nil && sentMsg.MessageID != 0 {
				currentMsgIDForNameInput = sentMsg.MessageID
			}
			bh.SendNameInputMenu(chatID, currentMsgIDForNameInput)
			newMenuMessageID = bh.Deps.SessionManager.GetTempOrder(chatID).CurrentMessageID
			return newMenuMessageID
		} else { // Клиент не найден (не должно произойти, если UserChatID установлен)
			log.Printf("[ORDER_HANDLER] Ошибка: клиент (UserChatID: %d) не найден для использования имени. Оператор введет имя.", tempOrder.UserChatID)
			sentMsg, errHelper = bh.sendErrorMessageHelper(chatID, originalMessageID, "Клиент не найден. Введите имя вручную.")
			bh.Deps.SessionManager.SetState(chatID, constants.STATE_ORDER_NAME)
			currentMsgIDForNameInput := originalMessageID
			if errHelper == nil && sentMsg.MessageID != 0 {
				currentMsgIDForNameInput = sentMsg.MessageID
			}
			bh.SendNameInputMenu(chatID, currentMsgIDForNameInput)
			newMenuMessageID = bh.Deps.SessionManager.GetTempOrder(chatID).CurrentMessageID
			return newMenuMessageID
		}
	} else if nameToUse == "" { // Если имя текущего пользователя пустое
		log.Printf("[ORDER_HANDLER] Ошибка: имя в профиле пользователя ChatID=%d не заполнено.", chatID)
		sentMsg, errHelper = bh.sendErrorMessageHelper(chatID, originalMessageID, "Ваше имя в профиле не заполнено. Введите имя вручную.")
		bh.Deps.SessionManager.SetState(chatID, constants.STATE_ORDER_NAME)
		currentMsgIDForNameInput := originalMessageID
		if errHelper == nil && sentMsg.MessageID != 0 {
			currentMsgIDForNameInput = sentMsg.MessageID
		}
		bh.SendNameInputMenu(chatID, currentMsgIDForNameInput)
		newMenuMessageID = bh.Deps.SessionManager.GetTempOrder(chatID).CurrentMessageID
		return newMenuMessageID
	}

	tempOrder.Name = nameToUse
	bh.Deps.SessionManager.UpdateTempOrder(chatID, tempOrder)
	bh.SendDateSelectionMenu(chatID, originalMessageID, 0)
	newMenuMessageID = bh.Deps.SessionManager.GetTempOrder(chatID).CurrentMessageID
	return newMenuMessageID
}

// handleConfirmOrderName обрабатывает подтверждение имени заказа.
func (bh *BotHandler) handleConfirmOrderName(chatID int64, user models.User, originalMessageID int) int {
	log.Printf("[ORDER_HANDLER] Подтверждение имени заказа. ChatID=%d", chatID)
	var newMenuMessageID int = originalMessageID
	var sentMsg tgbotapi.Message // Объявляем здесь
	var errHelper error          // Объявляем здесь

	tempOrder := bh.Deps.SessionManager.GetTempOrder(chatID)
	if tempOrder.Name == "" {
		log.Printf("[ORDER_HANDLER] Ошибка: имя для заказа не было установлено. ChatID=%d", chatID)
		sentMsg, errHelper = bh.sendErrorMessageHelper(chatID, originalMessageID, "Имя не было установлено.")
		currentMsgIDForNameInput := originalMessageID
		if errHelper == nil && sentMsg.MessageID != 0 {
			currentMsgIDForNameInput = sentMsg.MessageID
		}
		bh.SendNameInputMenu(chatID, currentMsgIDForNameInput)
		newMenuMessageID = bh.Deps.SessionManager.GetTempOrder(chatID).CurrentMessageID
		return newMenuMessageID
	}

	history := bh.Deps.SessionManager.GetHistory(chatID)
	isEditingOrder := len(history) >= 2 && history[len(history)-2] == constants.STATE_ORDER_EDIT && tempOrder.ID != 0

	if isEditingOrder {
		log.Printf("[ORDER_HANDLER] Редактирование: сохранение имени '%s' для заказа #%d. ChatID=%d", tempOrder.Name, tempOrder.ID, chatID)
		if err := db.UpdateOrderField(tempOrder.ID, "name", tempOrder.Name); err != nil {
			log.Printf("[ORDER_HANDLER] Ошибка сохранения имени для заказа #%d: %v. ChatID=%d", tempOrder.ID, err, chatID)
			sentMsg, errHelper = bh.sendErrorMessageHelper(chatID, originalMessageID, "Ошибка сохранения имени заказа.")
			currentMsgIDForEditMenu := originalMessageID
			if errHelper == nil && sentMsg.MessageID != 0 {
				currentMsgIDForEditMenu = sentMsg.MessageID
			}
			bh.SendEditOrderMenu(chatID, currentMsgIDForEditMenu)
			newMenuMessageID = bh.Deps.SessionManager.GetTempOrder(chatID).CurrentMessageID
			return newMenuMessageID
		}
		bh.SendEditOrderMenu(chatID, originalMessageID)
	} else {
		log.Printf("[ORDER_HANDLER] Переход к выбору даты после подтверждения имени. ChatID=%d", chatID)
		bh.SendDateSelectionMenu(chatID, originalMessageID, 0)
	}
	newMenuMessageID = bh.Deps.SessionManager.GetTempOrder(chatID).CurrentMessageID
	return newMenuMessageID
}

// handleConfirmOrderPhone обрабатывает подтверждение номера телефона для заказа.
func (bh *BotHandler) handleConfirmOrderPhone(chatID int64, user models.User, originalMessageID int) int {
	log.Printf("[ORDER_HANDLER] Подтверждение телефона для заказа. ChatID=%d", chatID)
	var newMenuMessageID int = originalMessageID
	var sentMsg tgbotapi.Message // Объявляем здесь
	var errHelper error          // Объявляем здесь

	tempOrder := bh.Deps.SessionManager.GetTempOrder(chatID)
	if tempOrder.Phone == "" { // Если телефон в tempOrder пуст (например, после "Изменить номер" -> "Назад")
		// Пытаемся взять из профиля пользователя, для которого создается заказ
		userForPhone := user // По умолчанию текущий пользователь
		if tempOrder.OrderAction == "operator_creating_order" && tempOrder.UserChatID != chatID && tempOrder.UserChatID != 0 {
			clientUser, clientFound := bh.getUserFromDB(tempOrder.UserChatID)
			if clientFound {
				userForPhone = clientUser
			}
		}
		if userForPhone.Phone.Valid && userForPhone.Phone.String != "" {
			tempOrder.Phone = userForPhone.Phone.String
			bh.Deps.SessionManager.UpdateTempOrder(chatID, tempOrder)
			log.Printf("[ORDER_HANDLER] Телефон взят из профиля (%s) для заказа. ChatID=%d", userForPhone.Phone.String, chatID)
		} else {
			log.Printf("[ORDER_HANDLER] Ошибка: номер телефона для заказа не был указан и не найден в профиле. ChatID=%d", chatID)
			sentMsg, errHelper = bh.sendErrorMessageHelper(chatID, originalMessageID, "Номер не был указан.")
			currentMsgIDForPhoneInput := originalMessageID
			if errHelper == nil && sentMsg.MessageID != 0 {
				currentMsgIDForPhoneInput = sentMsg.MessageID
			}
			bh.SendPhoneInputMenu(chatID, user, currentMsgIDForPhoneInput)
			newMenuMessageID = bh.Deps.SessionManager.GetTempOrder(chatID).CurrentMessageID
			return newMenuMessageID
		}
	}
	// Если tempOrder.Phone НЕ пустой, значит он был либо введен, либо подтвержден из профиля, либо оставлен из предыдущего шага "Изменить"

	history := bh.Deps.SessionManager.GetHistory(chatID)
	isEditingOrder := len(history) >= 2 && history[len(history)-2] == constants.STATE_ORDER_EDIT && tempOrder.ID != 0

	if isEditingOrder {
		log.Printf("[ORDER_HANDLER] Редактирование: сохранение телефона '%s' для заказа #%d. ChatID=%d", tempOrder.Phone, tempOrder.ID, chatID)
		if err := db.UpdateOrderField(tempOrder.ID, "phone", tempOrder.Phone); err != nil {
			log.Printf("[ORDER_HANDLER] Ошибка сохранения телефона для заказа #%d: %v. ChatID=%d", tempOrder.ID, err, chatID)
			sentMsg, errHelper = bh.sendErrorMessageHelper(chatID, originalMessageID, "Ошибка сохранения телефона заказа.")
			currentMsgIDForEditMenu := originalMessageID
			if errHelper == nil && sentMsg.MessageID != 0 {
				currentMsgIDForEditMenu = sentMsg.MessageID
			}
			bh.SendEditOrderMenu(chatID, currentMsgIDForEditMenu)
			newMenuMessageID = bh.Deps.SessionManager.GetTempOrder(chatID).CurrentMessageID
			return newMenuMessageID
		}
		bh.SendEditOrderMenu(chatID, originalMessageID)
	} else {
		log.Printf("[ORDER_HANDLER] Переход к вводу адреса после подтверждения телефона. ChatID=%d", chatID)
		bh.SendAddressInputMenu(chatID, originalMessageID)
	}
	newMenuMessageID = bh.Deps.SessionManager.GetTempOrder(chatID).CurrentMessageID
	return newMenuMessageID
}

// handleDateTimeSelection обрабатывает выбор даты или времени.
func (bh *BotHandler) handleDateTimeSelection(chatID int64, user models.User, command string, parts []string, originalMessageID int) int {
	log.Printf("[ORDER_HANDLER] Выбор даты/времени: Command='%s', Parts=%v, ChatID=%d", command, parts, chatID)
	var newMenuMessageID int = originalMessageID
	var sentMsg tgbotapi.Message // Объявляем здесь
	var errHelper error          // Объявляем здесь

	tempOrder := bh.Deps.SessionManager.GetTempOrder(chatID)
	history := bh.Deps.SessionManager.GetHistory(chatID)
	isEditingOrder := len(history) >= 2 && history[len(history)-2] == constants.STATE_ORDER_EDIT && tempOrder.ID != 0

	if command == "select_date_asap" {
		log.Printf("[ORDER_HANDLER] Выбрана дата 'СРОЧНО'. ChatID=%d", chatID)
		tempOrder.Date = time.Now().Format("02 January 2006") // Сохраняем в формате, который потом парсится ValidateDate
		tempOrder.Time = "СРОЧНО"
		bh.Deps.SessionManager.UpdateTempOrder(chatID, tempOrder)

		if isEditingOrder {
			log.Printf("[ORDER_HANDLER] Редактирование: установка даты 'СРОЧНО' для заказа #%d. ChatID=%d", tempOrder.ID, chatID)
			parsedDateForDB, _ := utils.ValidateDate(tempOrder.Date)
			_ = db.UpdateOrderField(tempOrder.ID, "date", parsedDateForDB)
			_ = db.UpdateOrderField(tempOrder.ID, "time", "СРОЧНО")
			bh.SendEditOrderMenu(chatID, originalMessageID)
		} else {
			bh.SendPhoneInputMenu(chatID, user, originalMessageID)
		}
	} else if command == "select_date" && len(parts) == 3 {
		dayStr, monthStr, yearStr := parts[0], parts[1], parts[2]
		dateToParse := fmt.Sprintf("%s %s %s", dayStr, monthStr, yearStr)
		log.Printf("[ORDER_HANDLER] Выбрана дата: '%s'. ChatID=%d", dateToParse, chatID)

		parsedDate, errDate := time.Parse("02 January 2006", dateToParse) // Используем формат, в котором сохраняем
		if errDate != nil {
			parsedDate, errDate = utils.ValidateDate(dateToParse) // Доп. проверка, если формат другой
			if errDate != nil {
				log.Printf("[ORDER_HANDLER] КРИТИЧЕСКАЯ ОШИБКА: Не удалось распознать дату '%s'. ChatID=%d, Error: %v", dateToParse, chatID, errDate)
				sentMsg, errHelper = bh.sendErrorMessageHelper(chatID, originalMessageID, "Некорректный формат даты в коллбэке.")
				currentMsgIDForDateMenu := originalMessageID
				if errHelper == nil && sentMsg.MessageID != 0 {
					currentMsgIDForDateMenu = sentMsg.MessageID
				}
				bh.SendDateSelectionMenu(chatID, currentMsgIDForDateMenu, 0)
				newMenuMessageID = bh.Deps.SessionManager.GetTempOrder(chatID).CurrentMessageID
				return newMenuMessageID
			}
		}
		tempOrder.Date = parsedDate.Format("02 January 2006") // Сохраняем в согласованном формате
		tempOrder.Time = ""                                   // Сбрасываем время при выборе новой даты
		bh.Deps.SessionManager.UpdateTempOrder(chatID, tempOrder)

		if isEditingOrder {
			log.Printf("[ORDER_HANDLER] Редактирование: установка даты '%s' для заказа #%d. ChatID=%d", tempOrder.Date, tempOrder.ID, chatID)
			_ = db.UpdateOrderField(tempOrder.ID, "date", parsedDate)
			_ = db.UpdateOrderField(tempOrder.ID, "time", nil) // Сброс времени в БД
			bh.SendEditOrderMenu(chatID, originalMessageID)
		} else {
			bh.SendTimeSelectionMenu(chatID, originalMessageID)
		}
	} else if command == "select_time" && len(parts) == 1 {
		timeStr := parts[0]
		log.Printf("[ORDER_HANDLER] Выбрано время: '%s'. ChatID=%d", timeStr, chatID)
		tempOrder.Time = timeStr
		bh.Deps.SessionManager.UpdateTempOrder(chatID, tempOrder)

		if isEditingOrder {
			log.Printf("[ORDER_HANDLER] Редактирование: установка времени '%s' для заказа #%d. ChatID=%d", timeStr, tempOrder.ID, chatID)
			_ = db.UpdateOrderField(tempOrder.ID, "time", timeStr)
			bh.SendEditOrderMenu(chatID, originalMessageID)
		} else {
			bh.SendPhoneInputMenu(chatID, user, originalMessageID)
		}
	} else {
		log.Printf("[ORDER_HANDLER] Ошибка: неизвестный тип выбора для 'select...': Command='%s', Parts=%v, ChatID=%d", command, parts, chatID)
		sentMsg, errHelper = bh.sendErrorMessageHelper(chatID, originalMessageID, "Ошибка выбора.")
		if errHelper == nil && sentMsg.MessageID != 0 {
			newMenuMessageID = sentMsg.MessageID
		}
		return newMenuMessageID
	}
	newMenuMessageID = bh.Deps.SessionManager.GetTempOrder(chatID).CurrentMessageID
	return newMenuMessageID
}

// handleSkipPhoto обрабатывает пропуск шага добавления фото.
func (bh *BotHandler) handleSkipPhoto(chatID int64, user models.User, originalMessageID int) int {
	log.Printf("[ORDER_HANDLER] Пропуск шага добавления фото. ChatID=%d", chatID)
	tempOrder := bh.Deps.SessionManager.GetTempOrder(chatID)
	tempOrder.ActiveMediaGroupID = ""                         // <--- СБРОС ActiveMediaGroupID
	bh.Deps.SessionManager.UpdateTempOrder(chatID, tempOrder) // Обновляем сессию с очищенным ID группы

	history := bh.Deps.SessionManager.GetHistory(chatID)
	isEditingOrder := len(history) >= 2 && history[len(history)-2] == constants.STATE_ORDER_EDIT && tempOrder.ID != 0

	if isEditingOrder {
		log.Printf("[ORDER_HANDLER] Редактирование: пропуск фото для заказа #%d. Фото/видео не изменяются. ChatID=%d", tempOrder.ID, chatID)
		bh.SendEditOrderMenu(chatID, originalMessageID)
	} else {
		log.Printf("[ORDER_HANDLER] Переход к выбору способа оплаты после пропуска фото. ChatID=%d", chatID)
		bh.SendPaymentSelectionMenu(chatID, originalMessageID)
	}
	return bh.Deps.SessionManager.GetTempOrder(chatID).CurrentMessageID
}

// handleFinishPhotoUpload обрабатывает завершение добавления фото/видео.
func (bh *BotHandler) handleFinishPhotoUpload(chatID int64, user models.User, originalMessageID int) int {
	log.Printf("[ORDER_HANDLER] Завершение добавления фото/видео. ChatID=%d", chatID)
	var newMenuMessageID int = originalMessageID
	// var sentMsg tgbotapi.Message // Объявляем здесь, если нужно
	// var errHelper error          // Объявляем здесь, если нужно

	tempOrder := bh.Deps.SessionManager.GetTempOrder(chatID)
	tempOrder.ActiveMediaGroupID = "" // <--- СБРОС ActiveMediaGroupID
	// Если TempOrderData как-то еще меняется здесь, то UpdateTempOrder нужен.
	// Но так как 다음 шаги (SendEditOrderMenu/SendPaymentSelectionMenu) вызовут sendOrEditMessageHelper,
	// который сделает GetTempOrder и UpdateTempOrder, то tempData обновится там.
	// Для явности можно сделать Update здесь:
	bh.Deps.SessionManager.UpdateTempOrder(chatID, tempOrder)

	history := bh.Deps.SessionManager.GetHistory(chatID)
	isEditingOrder := len(history) >= 2 && history[len(history)-2] == constants.STATE_ORDER_EDIT && tempOrder.ID != 0

	if isEditingOrder {
		log.Printf("[ORDER_HANDLER] Редактирование: сохранение фото/видео для заказа #%d. Фото: %d, Видео: %d. ChatID=%d", tempOrder.ID, len(tempOrder.Photos), len(tempOrder.Videos), chatID)
		errDb := db.UpdateOrderPhotosAndVideos(tempOrder.ID, tempOrder.Photos, tempOrder.Videos)
		if errDb != nil {
			log.Printf("[ORDER_HANDLER] Ошибка сохранения фото/видео для заказа #%d: %v. ChatID=%d", tempOrder.ID, errDb, chatID)
			// ... (обработка ошибки) ...
			// Убедимся, что CurrentMessageID обновляется даже при ошибке, если сообщение было отправлено
			currentMsgIDForEditMenu := originalMessageID
			sentErrorMsg, _ := bh.sendErrorMessageHelper(chatID, originalMessageID, "Ошибка сохранения фото/видео.")
			if sentErrorMsg.MessageID != 0 {
				currentMsgIDForEditMenu = sentErrorMsg.MessageID
			}
			bh.SendEditOrderMenu(chatID, currentMsgIDForEditMenu) // Эта функция обновит CurrentMessageID в сессии
			newMenuMessageID = bh.Deps.SessionManager.GetTempOrder(chatID).CurrentMessageID
			return newMenuMessageID
		}
		bh.SendEditOrderMenu(chatID, originalMessageID)
	} else {
		log.Printf("[ORDER_HANDLER] Переход к выбору способа оплаты после добавления фото/видео. ChatID=%d", chatID)
		bh.SendPaymentSelectionMenu(chatID, originalMessageID)
	}
	newMenuMessageID = bh.Deps.SessionManager.GetTempOrder(chatID).CurrentMessageID
	return newMenuMessageID
}

// handleResetPhotoUpload обрабатывает сброс всех добавленных медиа.
func (bh *BotHandler) handleResetPhotoUpload(chatID int64, user models.User, originalMessageID int) int {
	log.Printf("[ORDER_HANDLER] Сброс всех добавленных медиа. ChatID=%d", chatID)
	tempOrder := bh.Deps.SessionManager.GetTempOrder(chatID)
	tempOrder.Photos = []string{}
	tempOrder.Videos = []string{}
	tempOrder.ActiveMediaGroupID = ""                         // <--- СБРОС ActiveMediaGroupID
	bh.Deps.SessionManager.UpdateTempOrder(chatID, tempOrder) // Обновляем сессию

	bh.SendPhotoInputMenu(chatID, originalMessageID) // Эта функция обновит CurrentMessageID в сессии
	return bh.Deps.SessionManager.GetTempOrder(chatID).CurrentMessageID
}

// ЗАГЛУШКА: handleViewUploadedMedia обрабатывает просмотр загруженных медиа.
// Вам нужно будет реализовать эту функцию.
func (bh *BotHandler) handleViewUploadedMedia(chatID int64, user models.User, originalMessageID int) int {
	log.Printf("[CALLBACK_ORDER] Заглушка: handleViewUploadedMedia вызвана для ChatID=%d", chatID)
	// TODO: Реализуйте логику просмотра загруженных медиа.
	// Возможно, потребуется отправить пользователю карусель из фото/видео,
	// которые хранятся во временном объекте заказа в сессии (tempOrder.Photos, tempOrder.Videos).
	// Не забудьте обработать случай, когда медиа нет.

	// Пример ответного сообщения:
	tempOrder := bh.Deps.SessionManager.GetTempOrder(chatID)
	photoCount := len(tempOrder.Photos)
	videoCount := len(tempOrder.Videos)

	var mediaMessageText string
	if photoCount == 0 && videoCount == 0 {
		mediaMessageText = "Вы еще не загрузили фото или видео."
	} else {
		mediaMessageText = fmt.Sprintf("Загружено: %d фото, %d видео.", photoCount, videoCount)
		// Здесь может быть логика отправки самих медиа файлов.
		// Для простоты примера, просто выводим количество.
	}
	// Отправляем инфо-сообщение и потом меню загрузки фото
	// Send info message then photo input menu
	sentInfoMsg, _ := bh.sendInfoMessage(chatID, originalMessageID, mediaMessageText, "back_to_photo")

	currentMsgID := originalMessageID
	if sentInfoMsg.MessageID != 0 {
		currentMsgID = sentInfoMsg.MessageID
		// Поскольку sendInfoMessage сам становится CurrentMessageID,
		// то для SendPhotoInputMenu он и будет messageIDToEdit
	}

	// Возвращаем в меню загрузки фото
	bh.SendPhotoInputMenu(chatID, currentMsgID)
	return bh.Deps.SessionManager.GetTempOrder(chatID).CurrentMessageID
}

// handlePaymentSelection обрабатывает выбор способа оплаты.
func (bh *BotHandler) handlePaymentSelection(chatID int64, user models.User, parts []string, originalMessageID int) int {
	log.Printf("[ORDER_HANDLER] Выбор способа оплаты: Parts=%v, ChatID=%d", parts, chatID)
	var newMenuMessageID int = originalMessageID
	var sentMsg tgbotapi.Message // Объявляем здесь
	var errHelper error          // Объявляем здесь

	if len(parts) < 1 {
		log.Printf("[ORDER_HANDLER] Ошибка: недостаточно частей в коллбэке оплаты: %v. ChatID=%d", parts, chatID)
		sentMsg, errHelper = bh.sendErrorMessageHelper(chatID, originalMessageID, "Ошибка выбора способа оплаты.")
		if errHelper == nil && sentMsg.MessageID != 0 {
			newMenuMessageID = sentMsg.MessageID
		}
		return newMenuMessageID
	}
	paymentType := parts[0]

	tempOrder := bh.Deps.SessionManager.GetTempOrder(chatID)
	tempOrder.Payment = paymentType
	bh.Deps.SessionManager.UpdateTempOrder(chatID, tempOrder)

	history := bh.Deps.SessionManager.GetHistory(chatID)
	isEditingOrder := len(history) >= 2 && history[len(history)-2] == constants.STATE_ORDER_EDIT && tempOrder.ID != 0
	isOperatorCreating := tempOrder.OrderAction == "operator_creating_order"

	if isEditingOrder {
		log.Printf("[ORDER_HANDLER] Редактирование: сохранение способа оплаты '%s' для заказа #%d. ChatID=%d", paymentType, tempOrder.ID, chatID)
		errDb := db.UpdateOrderField(tempOrder.ID, "payment", paymentType)
		if errDb != nil {
			log.Printf("[ORDER_HANDLER] Ошибка сохранения способа оплаты для заказа #%d: %v. ChatID=%d", tempOrder.ID, errDb, chatID)
			sentMsg, errHelper = bh.sendErrorMessageHelper(chatID, originalMessageID, "Ошибка сохранения способа оплаты.")
			currentMsgIDForEditMenu := originalMessageID
			if errHelper == nil && sentMsg.MessageID != 0 {
				currentMsgIDForEditMenu = sentMsg.MessageID
			}
			bh.SendEditOrderMenu(chatID, currentMsgIDForEditMenu)
			newMenuMessageID = bh.Deps.SessionManager.GetTempOrder(chatID).CurrentMessageID
			return newMenuMessageID
		}
		bh.SendEditOrderMenu(chatID, originalMessageID)
	} else if isOperatorCreating {
		log.Printf("[ORDER_HANDLER] Оператор %d выбрал оплату '%s' для заказа. Переход к опциям создания.", chatID, paymentType)
		// Возвращаемся к меню с опциями: установить стоимость, назначить исполнителей и т.д.
		// SendOrderConfirmationMenu теперь обработает этот случай.
		bh.SendOrderConfirmationMenu(chatID, originalMessageID)
	} else { // Клиент создает заказ
		log.Printf("[ORDER_HANDLER] Переход к подтверждению заказа после выбора способа оплаты. ChatID=%d", chatID)
		bh.SendOrderConfirmationMenu(chatID, originalMessageID)
	}
	newMenuMessageID = bh.Deps.SessionManager.GetTempOrder(chatID).CurrentMessageID
	return newMenuMessageID
}

// handleChangeOrderPhone обрабатывает запрос на изменение номера телефона в заказе.
func (bh *BotHandler) handleChangeOrderPhone(chatID int64, user models.User, originalMessageID int) int {
	log.Printf("[ORDER_HANDLER] Запрос на изменение номера телефона в заказе. ChatID=%d", chatID)

	// 1. Очищаем телефон в сессии
	tempOrder := bh.Deps.SessionManager.GetTempOrder(chatID)
	tempOrder.Phone = ""
	bh.Deps.SessionManager.UpdateTempOrder(chatID, tempOrder)

	// 2. Устанавливаем правильное состояние, чтобы message_handler его подхватил
	bh.Deps.SessionManager.SetState(chatID, constants.STATE_ORDER_PHONE)

	// 3. Формируем и отправляем сообщение с запросом ввода (аналогично части SendPhoneInputMenu)
	promptEntity := "контактный"
	if tempOrder.OrderAction == "operator_creating_order" {
		promptEntity = "клиента"
	}
	msgText := fmt.Sprintf("📱 Пожалуйста, укажите новый %s номер телефона.\n\n"+
		"Вы можете отправить его текстом (например, +79001234567) или нажать кнопку ниже, чтобы поделиться контактом из Telegram.", promptEntity)

	// Инлайн-клавиатура для навигации
	history := bh.Deps.SessionManager.GetHistory(chatID)
	isEditingOrder := false
	if len(history) >= 2 && history[len(history)-2] == constants.STATE_ORDER_EDIT {
		isEditingOrder = true
	}
	backButtonCallback := "back_to_time" // Возвращаемся к предыдущему шагу до телефона
	if isEditingOrder {
		backButtonCallback = "back_to_edit_menu_direct"
	}

	inlineKeyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("⬅️ Назад", backButtonCallback),
			tgbotapi.NewInlineKeyboardButtonData("🏢 Главное меню", "back_to_main_confirm_cancel_order"),
		),
	)

	// Reply-клавиатура для шаринга контакта
	replyKeyboard := tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButtonContact(fmt.Sprintf("📞 Поделиться моим номером (%s)", utils.GetUserDisplayName(user))),
		),
	)
	replyKeyboard.OneTimeKeyboard = true
	replyKeyboard.ResizeKeyboard = true

	// Отправляем основное сообщение
	sentInlineMsg, err := bh.sendOrEditMessageHelper(chatID, originalMessageID, msgText, &inlineKeyboard, tgbotapi.ModeMarkdown)
	if err != nil {
		log.Printf("handleChangeOrderPhone: Ошибка отправки/редактирования основного сообщения для chatID %d: %v", chatID, err)
		return originalMessageID
	}

	// Отправляем временное сообщение с ReplyKeyboard
	tempMsgConfig := tgbotapi.NewMessage(chatID, "Вы также можете использовать кнопку ниже 👇")
	tempMsgConfig.ReplyMarkup = replyKeyboard

	sentReplyKbMsg, errKb := bh.Deps.BotClient.Send(tempMsgConfig)
	if errKb != nil {
		log.Printf("handleChangeOrderPhone: Ошибка отправки ReplyKeyboard для телефона chatID %d: %v", chatID, errKb)
	} else {
		// Сохраняем ID сообщения с ReplyKeyboard для последующего удаления
		orderDataSess := bh.Deps.SessionManager.GetTempOrder(chatID)
		if orderDataSess.CurrentMessageID != sentInlineMsg.MessageID && sentInlineMsg.MessageID != 0 {
			orderDataSess.CurrentMessageID = sentInlineMsg.MessageID
		}
		orderDataSess.LocationPromptMessageID = sentReplyKbMsg.MessageID
		bh.Deps.SessionManager.UpdateTempOrder(chatID, orderDataSess)
	}

	// Возвращаем ID основного (инлайн) сообщения
	return sentInlineMsg.MessageID
}

// handleEditOrderStart обрабатывает начало редактирования заказа.
func (bh *BotHandler) handleEditOrderStart(chatID int64, user models.User, orderIDStr string, originalMessageID int) int {
	log.Printf("[ORDER_HANDLER] Начало редактирования заказа: OrderIDStr=%s, ChatID=%d", orderIDStr, chatID)
	var newMenuMessageID int = originalMessageID
	var sentMsg tgbotapi.Message // Объявляем здесь
	var errHelper error          // Объявляем здесь

	orderID, err := strconv.ParseInt(orderIDStr, 10, 64)
	if err != nil {
		log.Printf("[ORDER_HANDLER] Ошибка: неверный OrderID в 'edit_order': '%s'. ChatID=%d", orderIDStr, chatID)
		sentMsg, errHelper = bh.sendErrorMessageHelper(chatID, originalMessageID, "Неверный ID заказа для редактирования.")
		if errHelper == nil && sentMsg.MessageID != 0 {
			newMenuMessageID = sentMsg.MessageID
		}
		return newMenuMessageID
	}

	orderData, errDb := db.GetOrderByID(int(orderID))
	if errDb != nil {
		log.Printf("[ORDER_HANDLER] Ошибка загрузки заказа #%d для редактирования: %v. ChatID=%d", orderID, errDb, chatID)
		sentMsg, errHelper = bh.sendErrorMessageHelper(chatID, originalMessageID, "Не удалось загрузить заказ для редактирования.")
		currentMsgIDForMyOrders := originalMessageID
		if errHelper == nil && sentMsg.MessageID != 0 {
			currentMsgIDForMyOrders = sentMsg.MessageID
		}
		bh.SendMyOrdersMenu(chatID, user, currentMsgIDForMyOrders, 0)
		newMenuMessageID = bh.Deps.SessionManager.GetTempOrder(chatID).CurrentMessageID
		return newMenuMessageID
	}

	// Проверка прав: только создатель заказа или оператор/выше могут редактировать
	if !(orderData.UserChatID == chatID || utils.IsOperatorOrHigher(user.Role)) {
		log.Printf("[ORDER_HANDLER] Ошибка: У пользователя ChatID=%d нет прав на редактирование заказа #%d.", chatID, orderID)
		sentMsg, errHelper = bh.sendErrorMessageHelper(chatID, originalMessageID, "У вас нет прав на редактирование этого заказа.")
		currentMsgIDForMyOrders := originalMessageID
		if errHelper == nil && sentMsg.MessageID != 0 {
			currentMsgIDForMyOrders = sentMsg.MessageID
		}
		bh.SendMyOrdersMenu(chatID, user, currentMsgIDForMyOrders, 0)
		newMenuMessageID = bh.Deps.SessionManager.GetTempOrder(chatID).CurrentMessageID
		return newMenuMessageID
	}

	tempOrder := bh.Deps.SessionManager.GetTempOrder(chatID)
	tempOrder.Order = orderData // Заполняем сессию данными из БД
	tempOrder.CurrentMessageID = originalMessageID
	// При входе в редактирование, если CurrentMessageID не 0, он должен стать единственным в MediaMessageIDs
	if tempOrder.CurrentMessageID != 0 {
		tempOrder.MediaMessageIDs = []int{tempOrder.CurrentMessageID}
		tempOrder.MediaMessageIDsMap = make(map[string]bool)
		tempOrder.MediaMessageIDsMap[fmt.Sprintf("%d", tempOrder.CurrentMessageID)] = true
	} else {
		tempOrder.MediaMessageIDs = []int{}
		tempOrder.MediaMessageIDsMap = make(map[string]bool)
	}
	bh.Deps.SessionManager.UpdateTempOrder(chatID, tempOrder)

	bh.Deps.SessionManager.SetState(chatID, constants.STATE_ORDER_EDIT)
	bh.SendEditOrderMenu(chatID, originalMessageID)
	newMenuMessageID = bh.Deps.SessionManager.GetTempOrder(chatID).CurrentMessageID
	return newMenuMessageID
}

// handleEditFieldSelection обрабатывает выбор поля для редактирования.
func (bh *BotHandler) handleEditFieldSelection(chatID int64, user models.User, fieldKey string, orderIDStr string, originalMessageID int) int {
	log.Printf("[ORDER_HANDLER] Выбор поля '%s' для редактирования заказа #%s. ChatID=%d", fieldKey, orderIDStr, chatID)
	var newMenuMessageID int = originalMessageID
	var sentMsg tgbotapi.Message // Объявляем здесь
	var errHelper error          // Объявляем здесь

	orderID, err := strconv.ParseInt(orderIDStr, 10, 64)
	if err != nil {
		log.Printf("[ORDER_HANDLER] Ошибка: неверный OrderID '%s' в 'edit_field'. ChatID=%d", orderIDStr, chatID)
		sentMsg, errHelper = bh.sendErrorMessageHelper(chatID, originalMessageID, "Неверный ID заказа в коллбэке редактирования поля.")
		if errHelper == nil && sentMsg.MessageID != 0 {
			newMenuMessageID = sentMsg.MessageID
		}
		return newMenuMessageID
	}

	tempOrder := bh.Deps.SessionManager.GetTempOrder(chatID)
	if tempOrder.ID != orderID { // Проверяем, что редактируем тот же заказ, что и в сессии
		log.Printf("[ORDER_HANDLER] Ошибка: ID заказа в коллбэке (%d) не совпадает с ID в сессии (%d). ChatID=%d. Перезагрузка заказа для редактирования.", orderID, tempOrder.ID, chatID)
		// Попытка восстановить контекст редактирования
		return bh.handleEditOrderStart(chatID, user, orderIDStr, originalMessageID)
	}

	// Устанавливаем CurrentMessageID для редактирования
	tempOrder.CurrentMessageID = originalMessageID
	if tempOrder.MediaMessageIDsMap == nil { // Инициализация карты, если она nil
		tempOrder.MediaMessageIDsMap = make(map[string]bool)
	}
	// Убедимся, что CurrentMessageID есть в MediaMessageIDs и карте
	foundInSlice := false
	for _, mid := range tempOrder.MediaMessageIDs {
		if mid == originalMessageID {
			foundInSlice = true
			break
		}
	}
	if !foundInSlice && originalMessageID != 0 {
		tempOrder.MediaMessageIDs = append(tempOrder.MediaMessageIDs, originalMessageID)
	}
	if originalMessageID != 0 {
		tempOrder.MediaMessageIDsMap[fmt.Sprintf("%d", originalMessageID)] = true
	}
	bh.Deps.SessionManager.UpdateTempOrder(chatID, tempOrder)

	switch fieldKey {
	case "description":
		bh.SendDescriptionInputMenu(chatID, originalMessageID)
	case "name":
		bh.SendNameInputMenu(chatID, originalMessageID)
	case "date":
		bh.SendDateSelectionMenu(chatID, originalMessageID, 0)
	case "time":
		bh.SendTimeSelectionMenu(chatID, originalMessageID)
	case "phone":
		bh.SendPhoneInputMenu(chatID, user, originalMessageID)
	case "address":
		bh.SendAddressInputMenu(chatID, originalMessageID)
	case "subcategory":
		if tempOrder.Category == "" && tempOrder.Order.Category != "" { // Если категория из сессии пуста, но есть в orderData
			tempOrder.Category = tempOrder.Order.Category
			bh.Deps.SessionManager.UpdateTempOrder(chatID, tempOrder)
		}
		bh.SendSubcategoryMenu(chatID, tempOrder.Category, originalMessageID)
	case "media":
		bh.SendPhotoInputMenu(chatID, originalMessageID)
	case "payment":
		bh.SendPaymentSelectionMenu(chatID, originalMessageID)
	default:
		log.Printf("[ORDER_HANDLER] Ошибка: неизвестное поле для редактирования '%s'. ChatID=%d", fieldKey, chatID)
		sentMsg, errHelper = bh.sendErrorMessageHelper(chatID, originalMessageID, "Неизвестное поле для редактирования.")
		currentMsgIDForEditMenu := originalMessageID
		if errHelper == nil && sentMsg.MessageID != 0 {
			currentMsgIDForEditMenu = sentMsg.MessageID
		}
		bh.SendEditOrderMenu(chatID, currentMsgIDForEditMenu)
	}
	newMenuMessageID = bh.Deps.SessionManager.GetTempOrder(chatID).CurrentMessageID
	return newMenuMessageID
}

// handleConfirmOrderFinal обрабатывает финальное подтверждение заказа.
// Адаптировано для операторского потока.
func (bh *BotHandler) handleConfirmOrderFinal(chatID int64, user models.User, orderID int64, originalMessageID int) int {
	log.Printf("[ORDER_HANDLER] Финальное подтверждение заказа #%d (ChatID=%d, UserRole=%s).", orderID, chatID, user.Role)
	var newMenuMessageID int = originalMessageID
	var sentMsg tgbotapi.Message // Объявляем здесь
	var errHelper error          // Объявляем здесь

	tempOrderSession := bh.Deps.SessionManager.GetTempOrder(chatID)
	isOperatorCreating := tempOrderSession.OrderAction == "operator_creating_order" && utils.IsOperatorOrHigher(user.Role)

	// Загружаем заказ из БД, чтобы убедиться в его существовании и актуальном статусе
	orderFromDB, errDbGet := db.GetOrderByID(int(orderID))
	if errDbGet != nil {
		log.Printf("[ORDER_HANDLER] Ошибка получения данных заказа #%d при финальном подтверждении: %v. ChatID=%d", orderID, errDbGet, chatID)
		sentMsg, errHelper = bh.sendErrorMessageHelper(chatID, originalMessageID, "Ошибка: не удалось найти заказ для подтверждения.")
		if errHelper == nil && sentMsg.MessageID != 0 {
			newMenuMessageID = sentMsg.MessageID
		}
		bh.SendMainMenu(chatID, user, newMenuMessageID)
		return newMenuMessageID
	}

	if isOperatorCreating {
		log.Printf("[ORDER_HANDLER] Оператор ChatID=%d завершает создание заказа #%d.", chatID, orderID)
		// Сохраняем все данные из сессии в БД
		// Категория, Подкатегория, Имя, Дата, Время, Телефон, Адрес, Описание, Фото, Видео, Оплата
		// уже должны быть в orderFromDB после CreateInitialOrder.
		// Нам нужно обновить их, если они менялись в сессии И ЕЩЕ НЕ БЫЛИ СОХРАНЕНЫ через UpdateOrderField

		// Проверяем, что orderID из сессии совпадает с orderID из коллбэка
		if tempOrderSession.ID != orderID {
			log.Printf("[ORDER_HANDLER] КРИТИЧЕСКАЯ ОШИБКА: ID заказа в сессии (%d) не совпадает с ID из коллбэка (%d) при финальном подтверждении оператором. ChatID=%d", tempOrderSession.ID, orderID, chatID)
			// Можно отправить ошибку и вернуть в главное меню оператора.
			bh.sendErrorMessageHelper(chatID, originalMessageID, "Критическая ошибка целостности данных сессии. Пожалуйста, начните заново.")
			bh.SendMainMenu(chatID, user, originalMessageID)
			return originalMessageID
		}

		// Обновляем все поля из сессии в БД
		// Это перезапишет данные, даже если они не менялись, но гарантирует консистентность
		db.UpdateOrderField(orderID, "category", tempOrderSession.Category)
		db.UpdateOrderField(orderID, "subcategory", tempOrderSession.Subcategory)
		db.UpdateOrderField(orderID, "name", tempOrderSession.Name)
		db.UpdateOrderField(orderID, "description", tempOrderSession.Description)
		//if tempOrderSession.Date != "" { // Если дата в сессии НЕ пустая
		//	log.Printf("[DEBUG_DATE_FINAL] handleConfirmOrderFinal (Operator): Заказ #%d. Дата в сессии ('%s') НЕ ПУСТАЯ. Попытка парсинга и обновления.", orderID, tempOrderSession.Date)
		//	parsedDate, errValidate := utils.ValidateDate(tempOrderSession.Date)
		//	if errValidate != nil {
		//		log.Printf("[ERROR_DATE_FINAL] handleConfirmOrderFinal (Operator): Заказ #%d. utils.ValidateDate НЕ СМОГ распарсить дату из сессии '%s': %v. Дата в БД НЕ будет обновлена через этот вызов.", orderID, tempOrderSession.Date, errValidate)
		//		// В этом случае НЕ вызываем db.UpdateOrderField для даты,
		//		// так как она либо уже корректна в БД после CreateInitialOrder,
		//		// либо строка в сессии испорчена, и мы не хотим портить БД нулевой датой.
		//	} else {
		//		log.Printf("[DEBUG_DATE_FINAL] handleConfirmOrderFinal (Operator): Заказ #%d. utils.ValidateDate УСПЕШНО распарсил '%s' в '%v'. Обновление БД.", orderID, tempOrderSession.Date, parsedDate.Format("2006-01-02"))
		//		errDbUpdate := db.UpdateOrderField(orderID, "date", parsedDate)
		//		if errDbUpdate != nil {
		//			log.Printf("[ERROR_DATE_FINAL] handleConfirmOrderFinal (Operator): Заказ #%d. db.UpdateOrderField НЕ СМОГ обновить дату: %v", orderID, errDbUpdate)
		//		} else {
		//			log.Printf("[DEBUG_DATE_FINAL] handleConfirmOrderFinal (Operator): Заказ #%d. db.UpdateOrderField УСПЕШНО обновил дату.", orderID)
		//		}
		//	}
		//} else { // Если дата в сессии ПУСТАЯ
		//	log.Printf("[WARN_DATE_FINAL] handleConfirmOrderFinal (Operator): Заказ #%d. tempOrderSession.Date из сессии ПУСТАЯ. Дата в БД НЕ будет обновлена на NULL через этот вызов (предполагаем, что CreateInitialOrder уже сохранил правильную или NULL).", orderID)
		//	// НЕ вызываем db.UpdateOrderField(orderID, "date", nil),
		//	// чтобы не затереть возможно корректно сохраненную дату из CreateInitialOrder.
		//	// Если CreateInitialOrder тоже не смог сохранить (например, дата и там была пустая), то в БД и так будет NULL.
		//}
		db.UpdateOrderField(orderID, "time", tempOrderSession.Time)
		db.UpdateOrderField(orderID, "phone", tempOrderSession.Phone)
		if tempOrderSession.Latitude != 0 || tempOrderSession.Longitude != 0 {
			db.UpdateOrderField(orderID, "latitude", tempOrderSession.Latitude)
			db.UpdateOrderField(orderID, "longitude", tempOrderSession.Longitude)
		}
		db.UpdateOrderPhotosAndVideos(orderID, tempOrderSession.Photos, tempOrderSession.Videos)
		db.UpdateOrderField(orderID, "payment", tempOrderSession.Payment)

		// Стоимость заказа (если оператор ее установил)
		if tempOrderSession.Cost.Valid && tempOrderSession.Cost.Float64 > 0 {
			db.UpdateOrderField(orderID, "cost", tempOrderSession.Cost.Float64)
		} else if tempOrderSession.OrderAction == "op_set_cost_after_confirm" && !tempOrderSession.Cost.Valid {
			// Если оператор был на шаге установки стоимости, но не ввел ее (маловероятно, т.к. есть skip)
			// или если пропустил, то стоимость не устанавливаем или ставим 0/NULL
			db.UpdateOrderField(orderID, "cost", nil) // или 0, в зависимости от логики
		}

		// Исполнители (если были назначены)
		// TODO: Добавить логику сохранения назначенных исполнителей из tempOrderSession (если они там хранятся)
		// Например:
		// if len(tempOrderSession.AssignedExecutors) > 0 {
		//   db.ClearExecutorsForOrder(orderID) // Сначала удалить старых, если это редактирование
		//   for _, exec := range tempOrderSession.AssignedExecutors {
		//     db.AssignExecutor(int(orderID), exec.ChatID, exec.Role)
		//     go bh.NotifyExecutorAboutAssignment(exec.ChatID, orderID, exec.Role)
		//   }
		// }

		// Устанавливаем статус "В работе"
		errDb := db.UpdateOrderStatus(orderID, constants.STATUS_INPROGRESS)
		if errDb != nil {
			log.Printf("[ORDER_HANDLER] Оператор: Ошибка обновления статуса заказа #%d на IN_PROGRESS: %v. ChatID=%d", orderID, errDb, chatID)
			sentMsg, errHelper = bh.sendErrorMessageHelper(chatID, originalMessageID, "Ошибка установки статуса заказа.")
			if errHelper == nil && sentMsg.MessageID != 0 {
				newMenuMessageID = sentMsg.MessageID
			}
			return newMenuMessageID
		}
		log.Printf("[ORDER_HANDLER] Заказ #%d успешно создан оператором (ChatID=%d) и переведен в статус IN_PROGRESS.", orderID, chatID)

		// Уведомление клиенту (если заказ создавался для конкретного клиента, а не "на себя")
		finalOrderData, _ := db.GetOrderByID(int(orderID))
		if finalOrderData.UserChatID != chatID && finalOrderData.UserChatID != 0 { // Уведомляем, если клиент - не сам оператор
			clientMessageText := fmt.Sprintf("✅ Ваш заказ №%d (для %s, тел: %s) создан оператором и находится в работе!",
				orderID,
				utils.EscapeTelegramMarkdown(finalOrderData.Name),
				utils.EscapeTelegramMarkdown(finalOrderData.Phone))
			if finalOrderData.Cost.Valid && finalOrderData.Cost.Float64 > 0 {
				clientMessageText += fmt.Sprintf("\nСтоимость: *%.0f ₽*", finalOrderData.Cost.Float64)
			}
			bh.sendMessage(finalOrderData.UserChatID, clientMessageText)
		}

		// Уведомляем назначенных исполнителей (если они были назначены на предыдущем шаге)
		// TODO: Реализовать уведомление исполнителей
		// assignedExecs, _ := db.GetExecutorsByOrderID(int(orderID))
		// for _, exec := range assignedExecs { ... bh.NotifyExecutorAboutAssignment ... }

		successMsgText := fmt.Sprintf("✅ Заказ №%d успешно создан и переведен в статус '%s'!", orderID, constants.StatusDisplayMap[constants.STATUS_INPROGRESS])
		successKeyboard := tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("📦 Заказы", "manage_orders")),
			tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("🏢 Главное меню", "back_to_main")),
		)
		sentSuccessMsg, errSend := bh.sendOrEditMessageHelper(chatID, originalMessageID, successMsgText, &successKeyboard, "")
		if errSend != nil {
			log.Printf("Ошибка отправки сообщения оператору об успешном создании заказа: %v", errSend)
			newMenuMessageID = 0
		} else {
			newMenuMessageID = sentSuccessMsg.MessageID
		}
		bh.Deps.SessionManager.ClearTempOrder(chatID)
		bh.Deps.SessionManager.ClearState(chatID)
		return newMenuMessageID

	} else { // Клиент подтверждает свой заказ
		actualClientChatID := orderFromDB.UserChatID
		if chatID != actualClientChatID {
			log.Printf("[ORDER_HANDLER] Ошибка: Попытка подтвердить заказ #%d не клиентом. ChatID запроса: %d, ChatID клиента: %d.", orderID, chatID, actualClientChatID)
			sentMsg, errHelper = bh.sendErrorMessageHelper(chatID, originalMessageID, "Это действие предназначено для клиента заказа.")
			if errHelper == nil && sentMsg.MessageID != 0 {
				newMenuMessageID = sentMsg.MessageID
			}
			return newMenuMessageID
		}

		if orderFromDB.Status != constants.STATUS_DRAFT {
			log.Printf("[ORDER_HANDLER] Попытка подтвердить заказ #%d, который уже не в статусе DRAFT (статус: %s). ChatID=%d", orderID, orderFromDB.Status, chatID)
			statusText := constants.StatusDisplayMap[orderFromDB.Status]
			if statusText == "" {
				statusText = orderFromDB.Status
			}
			sentMsg, errHelper = bh.sendErrorMessageHelper(chatID, originalMessageID, fmt.Sprintf("Ваш заказ уже обрабатывается (статус: %s).", statusText))
			currentMsgIDForMyOrders := originalMessageID
			if errHelper == nil && sentMsg.MessageID != 0 {
				currentMsgIDForMyOrders = sentMsg.MessageID
			}
			bh.SendMyOrdersMenu(chatID, user, currentMsgIDForMyOrders, 0)
			newMenuMessageID = bh.Deps.SessionManager.GetTempOrder(chatID).CurrentMessageID
			return newMenuMessageID
		}

		errDbUpdate := db.UpdateOrderStatus(orderID, constants.STATUS_NEW)
		if errDbUpdate != nil {
			log.Printf("[ORDER_HANDLER] Клиент: Ошибка обновления статуса заказа #%d на NEW: %v. ChatID=%d", orderID, errDbUpdate, chatID)
			sentMsg, errHelper = bh.sendErrorMessageHelper(chatID, originalMessageID, "Ошибка подтверждения заказа.")
			if errHelper == nil && sentMsg.MessageID != 0 {
				newMenuMessageID = sentMsg.MessageID
			}
			return newMenuMessageID
		}

		log.Printf("[ORDER_HANDLER] Заказ #%d успешно подтвержден клиентом (ChatID=%d) и переведен в статус NEW. Уведомление операторам...", orderID, chatID)
		go bh.NotifyOperatorsAboutNewOrder(orderID, actualClientChatID)

		successMsgText := fmt.Sprintf("✅ Ваш заказ №%d успешно подтвержден и отправлен оператору!\n\n"+
			"Оператор свяжется с вами для согласования стоимости, если это потребуется.\n\n"+
			"Спасибо за ваш заказ!", orderID)
		successKeyboard := tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("📋 Мои заказы", "my_orders_page_0")),
			tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("🏢 Главное меню", "back_to_main")),
		)
		sentSuccessMsg, errSend := bh.sendOrEditMessageHelper(chatID, originalMessageID, successMsgText, &successKeyboard, "")
		if errSend != nil {
			log.Printf("Ошибка отправки сообщения клиенту об успешном подтверждении заказа: %v", errSend)
			newMenuMessageID = 0
		} else {
			newMenuMessageID = sentSuccessMsg.MessageID
		}

		bh.Deps.SessionManager.ClearTempOrder(chatID)
		bh.Deps.SessionManager.ClearState(chatID)
		return newMenuMessageID
	}
}

// handleAcceptCost обрабатывает подтверждение клиентом предложенной стоимости.
// func (bh *BotHandler) handleAcceptCost(chatID int64, user models.User, orderIDStr string, originalMessageID int) int // ...

// handleRejectCost обрабатывает отклонение клиентом предложенной стоимости.
func (bh *BotHandler) handleRejectCost(chatID int64, user models.User, orderIDStr string, originalMessageID int) int {
	log.Printf("[ORDER_HANDLER] Отклонение стоимости заказа: OrderIDStr=%s, ChatID=%d", orderIDStr, chatID)
	var newMenuMessageID int = originalMessageID
	var sentMsg tgbotapi.Message // Объявляем здесь
	var errHelper error          // Объявляем здесь

	orderID, err := strconv.ParseInt(orderIDStr, 10, 64)
	if err != nil {
		log.Printf("[ORDER_HANDLER] Ошибка: неверный OrderID в 'reject_cost': '%s'. ChatID=%d", orderIDStr, chatID)
		sentMsg, errHelper = bh.sendErrorMessageHelper(chatID, originalMessageID, "Неверный ID заказа.")
		if errHelper == nil && sentMsg.MessageID != 0 {
			newMenuMessageID = sentMsg.MessageID
		}
		return newMenuMessageID
	}

	orderData, errDb := db.GetOrderByID(int(orderID))
	if errDb != nil {
		log.Printf("[ORDER_HANDLER] Ошибка загрузки заказа #%d: %v. ChatID=%d", orderID, errDb, chatID)
		sentMsg, errHelper = bh.sendErrorMessageHelper(chatID, originalMessageID, "Ошибка загрузки заказа.")
		if errHelper == nil && sentMsg.MessageID != 0 {
			newMenuMessageID = sentMsg.MessageID
		}
		return newMenuMessageID
	}

	if orderData.UserChatID != chatID {
		log.Printf("[ORDER_HANDLER] Ошибка: Попытка отклонить стоимость для заказа #%d не клиентом. ChatID запроса: %d", orderID, chatID)
		sentMsg, errHelper = bh.sendErrorMessageHelper(chatID, originalMessageID, "Это действие доступно только клиенту заказа.")
		if errHelper == nil && sentMsg.MessageID != 0 {
			newMenuMessageID = sentMsg.MessageID
		}
		return newMenuMessageID
	}
	if orderData.Status != constants.STATUS_AWAITING_CONFIRMATION {
		log.Printf("[ORDER_HANDLER] Попытка отклонить стоимость для заказа #%d не в статусе AWAITING_CONFIRMATION (статус: %s). ChatID=%d", orderID, orderData.Status, chatID)
		sentMsg, errHelper = bh.sendErrorMessageHelper(chatID, originalMessageID, "Нельзя отклонить стоимость для этого заказа (неверный статус).")
		if errHelper == nil && sentMsg.MessageID != 0 {
			newMenuMessageID = sentMsg.MessageID
		}
		return newMenuMessageID
	}

	log.Printf("[ORDER_HANDLER] Клиент ChatID=%d отклонил стоимость для заказа #%d. Запрос причины...", chatID, orderID)
	bh.Deps.SessionManager.SetState(chatID, constants.STATE_CANCEL_REASON)
	tempData := bh.Deps.SessionManager.GetTempOrder(chatID)
	tempData.ID = orderID
	tempData.OrderAction = "reject_cost" // Контекст для обработчика причины отмены
	bh.Deps.SessionManager.UpdateTempOrder(chatID, tempData)

	bh.SendCancelReasonInput(chatID, int(orderID), originalMessageID, "reject_cost")
	newMenuMessageID = bh.Deps.SessionManager.GetTempOrder(chatID).CurrentMessageID
	return newMenuMessageID
}

// handleCancelOrder обрабатывает отмену заказа (как клиентом, так и оператором).
func (bh *BotHandler) handleCancelOrder(chatID int64, user models.User, actionType string, orderIDStr string, originalMessageID int) int {
	log.Printf("[ORDER_HANDLER] Отмена заказа: ActionType=%s, OrderIDStr=%s, ChatID=%d", actionType, orderIDStr, chatID)
	var newMenuMessageID int = originalMessageID
	var sentMsg tgbotapi.Message // Объявляем здесь
	var errHelper error          // Объявляем здесь

	orderID, err := strconv.ParseInt(orderIDStr, 10, 64)
	if err != nil {
		log.Printf("[ORDER_HANDLER] Ошибка: неверный OrderID в 'cancel_order': '%s'. ChatID=%d", orderIDStr, chatID)
		sentMsg, errHelper = bh.sendErrorMessageHelper(chatID, originalMessageID, "Неверный ID заказа для отмены.")
		if errHelper == nil && sentMsg.MessageID != 0 {
			newMenuMessageID = sentMsg.MessageID
		}
		return newMenuMessageID
	}

	if actionType == "confirm" { // Клиент нажал "Отменить мой заказ"
		log.Printf("[ORDER_HANDLER] Клиент ChatID=%d инициировал отмену заказа #%d (тип 'confirm').", chatID, orderID)
		orderForCancel, errGet := db.GetOrderByID(int(orderID))
		if errGet != nil || orderForCancel.UserChatID != chatID {
			log.Printf("[ORDER_HANDLER] Ошибка проверки заказа #%d для отмены клиентом ChatID=%d: %v", orderID, chatID, errGet)
			sentMsg, errHelper = bh.sendErrorMessageHelper(chatID, originalMessageID, "Ошибка отмены заказа. Не удалось проверить данные.")
			if errHelper == nil && sentMsg.MessageID != 0 {
				newMenuMessageID = sentMsg.MessageID
			}
			return newMenuMessageID
		}

		canClientCancel := false
		// Клиент может отменить заказ, если он в статусе DRAFT,
		// или AWAITING_COST (если стоимость еще не была предложена),
		// или AWAITING_CONFIRMATION (отклонение стоимости через reject_cost, здесь только отмена)
		if orderForCancel.Status == constants.STATUS_DRAFT {
			canClientCancel = true
		} else if orderForCancel.Status == constants.STATUS_AWAITING_COST && (!orderForCancel.Cost.Valid || (orderForCancel.Cost.Valid && orderForCancel.Cost.Float64 == 0.0)) {
			canClientCancel = true
		}
		// Если статус AWAITING_CONFIRMATION, то отмена идет через reject_cost -> cancel_reason.
		// Эта ветка для прямой отмены.

		if canClientCancel {
			log.Printf("[ORDER_HANDLER] Запрос причины отмены заказа #%d (статус %s) клиентом ChatID=%d.", orderID, orderForCancel.Status, chatID)
			bh.Deps.SessionManager.SetState(chatID, constants.STATE_CANCEL_REASON)
			tempDataForReason := bh.Deps.SessionManager.GetTempOrder(chatID)
			tempDataForReason.ID = orderID
			tempDataForReason.OrderAction = "user_cancel_draft_or_awaiting_cost_no_cost"
			bh.Deps.SessionManager.UpdateTempOrder(chatID, tempDataForReason)
			bh.SendCancelReasonInput(chatID, int(orderID), originalMessageID, "user_cancel_draft_or_awaiting_cost_no_cost")
		} else {
			log.Printf("[ORDER_HANDLER] Клиент ChatID=%d не может отменить заказ #%d через этот коллбэк (статус %s). Предлагаем связаться с оператором.",
				chatID, orderID, orderForCancel.Status)
			var costStr string
			if orderForCancel.Cost.Valid {
				costStr = fmt.Sprintf("(установлена стоимость: %.0f ₽)", orderForCancel.Cost.Float64)
			} else {
				costStr = "(стоимость еще не установлена)"
			}

			errorMsgText := fmt.Sprintf("Для отмены заказа №%d %s, пожалуйста, свяжитесь с оператором.", orderID, costStr)
			sentMsg, errHelper = bh.sendErrorMessageHelper(chatID, originalMessageID, errorMsgText)
			currentMsgIDForContactMenu := originalMessageID
			if errHelper == nil && sentMsg.MessageID != 0 {
				currentMsgIDForContactMenu = sentMsg.MessageID
			}
			bh.SendContactOperatorMenu(chatID, user, currentMsgIDForContactMenu)
		}
	} else if actionType == "operator" { // Оператор нажал "Отменить заказ"
		log.Printf("[ORDER_HANDLER] Оператор ChatID=%d инициировал отмену заказа #%d.", chatID, orderID)
		if !utils.IsOperatorOrHigher(user.Role) {
			log.Printf("[ORDER_HANDLER] Ошибка: Пользователь ChatID=%d (роль %s) не имеет прав на отмену заказа #%d как оператор.", chatID, user.Role, orderID)
			sentMsg, errHelper = bh.sendErrorMessageHelper(chatID, originalMessageID, constants.AccessDeniedMessage)
			if errHelper == nil && sentMsg.MessageID != 0 {
				newMenuMessageID = sentMsg.MessageID
			}
			return newMenuMessageID
		}
		bh.Deps.SessionManager.SetState(chatID, constants.STATE_CANCEL_REASON)
		tempData := bh.Deps.SessionManager.GetTempOrder(chatID)
		tempData.ID = orderID
		tempData.OrderAction = "operator_cancel"
		bh.Deps.SessionManager.UpdateTempOrder(chatID, tempData)
		bh.SendCancelReasonInput(chatID, int(orderID), originalMessageID, "operator_cancel")
	} else {
		log.Printf("[ORDER_HANDLER] Ошибка: неизвестный тип действия для 'cancel_order': %s. ChatID=%d", actionType, chatID)
		sentMsg, errHelper = bh.sendErrorMessageHelper(chatID, originalMessageID, "Неизвестное действие отмены.")
		if errHelper == nil && sentMsg.MessageID != 0 {
			newMenuMessageID = sentMsg.MessageID
		}
		return newMenuMessageID
	}
	newMenuMessageID = bh.Deps.SessionManager.GetTempOrder(chatID).CurrentMessageID
	return newMenuMessageID
}

// NotifyOperatorsAndGroup уведомляет операторов и группу (если настроена).
// Убрал contextKey, так как он не использовался.
func (bh *BotHandler) NotifyOperatorsAndGroup(message string) {
	log.Printf("[NOTIFY_OPS_GROUP] Message: %s", message)

	operators, err := db.GetUsersByRole(constants.ROLE_OPERATOR, constants.ROLE_MAINOPERATOR, constants.ROLE_OWNER)
	if err != nil {
		log.Printf("NotifyOperatorsAndGroup: ошибка получения списка операторов: %v", err)
	} else {
		for _, op := range operators {
			bh.NotifyOperator(op.ChatID, message)
		}
	}

	if bh.Deps.Config.GroupChatID != 0 {
		bh.NotifyOperator(bh.Deps.Config.GroupChatID, message)
	}
}
func (bh *BotHandler) handleOperatorStartOrderCreation(chatID int64, user models.User, messageIDToEdit int) {
	log.Printf("BotHandler.handleOperatorStartOrderCreation: Оператор ChatID=%d (Роль: %s) начинает создание нового заказа. MessageIDToEdit: %d", chatID, user.Role, messageIDToEdit)

	// 1. Очищаем предыдущее состояние и временный заказ оператора, если он был
	bh.Deps.SessionManager.ClearState(chatID)
	bh.Deps.SessionManager.ClearTempOrder(chatID) // Убедимся, что для оператора нет "висячих" данных

	// 2. Устанавливаем состояние, указывающее, что оператор в процессе выбора клиента для заказа
	// или сразу в общем потоке создания заказа, если выбор клиента не первый шаг.
	// STATE_OP_CREATE_ORDER_FLOW - это общее состояние для всего потока.
	// Первым фактическим шагом может быть выбор клиента.
	bh.Deps.SessionManager.SetState(chatID, constants.STATE_OP_CREATE_ORDER_FLOW)

	// 3. Инициализируем TempOrderData для этого оператора.
	// UserChatID самого заказа будет установлен ПОСЛЕ выбора клиента.
	// Пока можно установить его в 0 или ChatID оператора, если заказ может быть "на себя".
	// OrderAction четко указывает, что это заказ от оператора.
	tempOrderForOperator := session.NewTempOrder(chatID) // UserChatID здесь -- это ChatID оператора, который инициирует
	tempOrderForOperator.OrderAction = "operator_creating_order"
	tempOrderForOperator.CurrentMessageID = messageIDToEdit // Сохраняем ID для редактирования
	bh.Deps.SessionManager.UpdateTempOrder(chatID, tempOrderForOperator)

	// 4. Отправляем первое меню этого флоу - ВЫБОР КАТЕГОРИИ.
	// Имя пользователя (user.FirstName) передается для приветствия в SendCategoryMenu.
	bh.SendCategoryMenu(chatID, user.FirstName, messageIDToEdit) // <<< ИЗМЕНЕННАЯ СТРОКА
	log.Printf("BotHandler.handleOperatorStartOrderCreation: Оператору %d отправлено меню выбора категории.", chatID)
}

// handleDriverStartOrderCreation запускает процесс создания заказа для водителя.
func (bh *BotHandler) handleDriverStartOrderCreation(chatID int64, user models.User, messageIDToEdit int) {
	log.Printf("BotHandler.handleDriverStartOrderCreation: Водитель ChatID=%d (Роль: %s) начинает создание нового заказа. MessageIDToEdit: %d", chatID, user.Role, messageIDToEdit)

	// 1. Очищаем предыдущее состояние и временный заказ
	bh.Deps.SessionManager.ClearState(chatID)
	bh.Deps.SessionManager.ClearTempOrder(chatID)

	// 2. Устанавливаем специальное состояние для этого потока
	bh.Deps.SessionManager.SetState(chatID, constants.STATE_DRIVER_CREATE_ORDER_FLOW)

	// 3. Инициализируем TempOrderData.
	// Здесь UserChatID заказа будет 0, так как водитель вводит данные клиента.
	// Мы сохраняем сессию под ChatID самого водителя.
	tempOrderForDriver := session.NewTempOrder(chatID)
	tempOrderForDriver.OrderAction = "driver_creating_order" // Специальный флаг для контекста
	tempOrderForDriver.CurrentMessageID = messageIDToEdit
	bh.Deps.SessionManager.UpdateTempOrder(chatID, tempOrderForDriver)

	// 4. Отправляем первое меню этого флоу - ВЫБОР КАТЕГОРИИ.
	// В SendCategoryMenu имя user.FirstName будет использовано только для приветствия.
	bh.SendCategoryMenu(chatID, user.FirstName, messageIDToEdit)
	log.Printf("BotHandler.handleDriverStartOrderCreation: Водителю %d отправлено меню выбора категории.", chatID)
}
