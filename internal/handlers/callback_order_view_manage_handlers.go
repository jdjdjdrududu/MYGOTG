// Файл: internal/handlers/callback_order_view_manage_handlers.go

package handlers

import (
	"Original/internal/formatters"
	"Original/internal/session"
	"database/sql"
	"fmt"
	"log"
	"strconv"
	// "strings" // Not directly used here, but utils might use it

	"Original/internal/constants" //
	"Original/internal/db"
	"Original/internal/models"
	"Original/internal/utils" //

	tgbotapi "github.com/OvyFlash/telegram-bot-api"
)

// dispatchOrderViewManageCallbacks маршрутирует коллбэки, связанные с просмотром и управлением заказами.
// query - объект CallbackQuery от Telegram
// currentCommand - это основная команда (например, "my_orders_page", "view_order_ops").
// parts - это оставшиеся части callback_data.
// data - это полная строка callback_data.
// Возвращает ID нового отправленного/отредактированного сообщения или 0.
func (bh *BotHandler) dispatchOrderViewManageCallbacks(query *tgbotapi.CallbackQuery, currentCommand string, parts []string, data string, chatID int64, user models.User, originalMessageID int) int {
	log.Printf("[CALLBACK_ORDER_VM] Диспетчер: Команда='%s', Части=%v, Data='%s', ChatID=%d", currentCommand, parts, data, chatID)
	var newMenuMessageID int = originalMessageID
	var sentMsg tgbotapi.Message
	var errHelper error
	var queryID string // Объявляем queryID здесь
	if query != nil {
		queryID = query.ID // Присваиваем значение, если query не nil
	}

	switch currentCommand {
	case "manage_orders": // Операторское меню управления заказами
		if !utils.IsOperatorOrHigher(user.Role) {
			sentMsg, _ = bh.sendAccessDenied(chatID, originalMessageID)
			if sentMsg.MessageID != 0 {
				newMenuMessageID = sentMsg.MessageID
			}
			return newMenuMessageID
		}
		bh.SendOrdersMenu(chatID, user, originalMessageID)
		newMenuMessageID = bh.Deps.SessionManager.GetTempOrder(chatID).CurrentMessageID
	case "operator_orders_new":
		page := 0
		if len(parts) > 0 {
			page, _ = strconv.Atoi(parts[0])
		}
		bh.SendOrderListByStatus(chatID, []string{constants.STATUS_NEW, constants.STATUS_AWAITING_COST}, constants.STATUS_NEW, page, originalMessageID, "🆕 Новые заказы", "Нет новых заказов.")
		newMenuMessageID = bh.Deps.SessionManager.GetTempOrder(chatID).CurrentMessageID

	case "operator_orders_awaiting_confirmation":
		page := 0
		if len(parts) > 0 {
			page, _ = strconv.Atoi(parts[0])
		}
		// --- НАЧАЛО ИЗМЕНЕНИЯ ---
		bh.SendOrderListByStatus(
			chatID,
			[]string{constants.STATUS_AWAITING_CONFIRMATION, constants.STATUS_AWAITING_PAYMENT}, // Показываем и те, что ждут оплаты
			constants.STATUS_AWAITING_CONFIRMATION,                                              // Ключ для пагинации
			page,
			originalMessageID,
			"⏳ Заказы, ожидающие действия клиента", // Новый заголовок
			"Нет заказов, ожидающих действия клиента.",
		)
		// --- КОНЕЦ ИЗМЕНЕНИЯ ---
		newMenuMessageID = bh.Deps.SessionManager.GetTempOrder(chatID).CurrentMessageID
	case constants.CALLBACK_PREFIX_PAY_ORDER:
		if len(parts) == 1 {
			_, err := strconv.ParseInt(parts[0], 10, 64)
			if err == nil {
				bh.handlePayOrder(chatID, user, parts[0], originalMessageID)
				newMenuMessageID = bh.Deps.SessionManager.GetTempOrder(chatID).CurrentMessageID
			} else {
				log.Printf("[CALLBACK_ORDER_VM] Ошибка конвертации OrderID для '%s': '%s'. ChatID=%d", currentCommand, parts[0], chatID)
				sentMsg, errHelper = bh.sendErrorMessageHelper(chatID, originalMessageID, "Ошибка: неверный ID заказа.")
				if errHelper == nil && sentMsg.MessageID != 0 {
					newMenuMessageID = sentMsg.MessageID
				}
			}
		} else {
			log.Printf("[CALLBACK_ORDER_VM] Некорректный формат для '%s': %v. ChatID=%d", currentCommand, parts, chatID)
			sentMsg, errHelper = bh.sendErrorMessageHelper(chatID, originalMessageID, "Ошибка: неверный формат команды оплаты.")
			if errHelper == nil && sentMsg.MessageID != 0 {
				newMenuMessageID = sentMsg.MessageID
			}
		}
	case "operator_orders_in_progress":
		page := 0
		if len(parts) > 0 {
			page, _ = strconv.Atoi(parts[0])
		}
		bh.SendOrderListByStatus(chatID, []string{constants.STATUS_INPROGRESS}, constants.STATUS_INPROGRESS, page, originalMessageID, "🚚 Заказы в работе", "Нет заказов в работе.")
		newMenuMessageID = bh.Deps.SessionManager.GetTempOrder(chatID).CurrentMessageID

	case "operator_orders_completed":
		page := 0
		if len(parts) > 0 {
			page, _ = strconv.Atoi(parts[0])
		}
		bh.SendOrderListByStatus(chatID, []string{constants.STATUS_COMPLETED}, constants.STATUS_COMPLETED, page, originalMessageID, "✅ Завершённые заказы", "Нет завершённых заказов.")
		newMenuMessageID = bh.Deps.SessionManager.GetTempOrder(chatID).CurrentMessageID

	case "operator_orders_calculated":
		page := 0
		if len(parts) > 0 {
			page, _ = strconv.Atoi(parts[0])
		}
		bh.SendOrderListByStatus(chatID, []string{constants.STATUS_CALCULATED}, constants.STATUS_CALCULATED, page, originalMessageID, "🧮 Рассчитанные заказы (финансы)", "Нет рассчитанных заказов.")
		newMenuMessageID = bh.Deps.SessionManager.GetTempOrder(chatID).CurrentMessageID

	case "operator_orders_canceled":
		page := 0
		if len(parts) > 0 {
			page, _ = strconv.Atoi(parts[0])
		}
		bh.SendOrderListByStatus(chatID, []string{constants.STATUS_CANCELED}, constants.STATUS_CANCELED, page, originalMessageID, "❌ Отменённые заказы", "Нет отменённых заказов.")
		newMenuMessageID = bh.Deps.SessionManager.GetTempOrder(chatID).CurrentMessageID
	case "my_orders_page":
		if len(parts) == 1 {
			page, err := strconv.Atoi(parts[0])
			if err == nil {
				bh.SendMyOrdersMenu(chatID, user, originalMessageID, page)
				newMenuMessageID = bh.Deps.SessionManager.GetTempOrder(chatID).CurrentMessageID
			} else {
				log.Printf("[CALLBACK_ORDER_VM] Ошибка конвертации страницы для 'my_orders_page': '%s'. ChatID=%d", parts[0], chatID)
				sentMsg, errHelper = bh.sendErrorMessageHelper(chatID, originalMessageID, "Ошибка навигации по заказам.")
				if errHelper == nil && sentMsg.MessageID != 0 {
					newMenuMessageID = sentMsg.MessageID
				}
			}
		} else {
			log.Printf("[CALLBACK_ORDER_VM] Некорректный формат для 'my_orders_page': %v. ChatID=%d", parts, chatID)
			sentMsg, errHelper = bh.sendErrorMessageHelper(chatID, originalMessageID, "Ошибка отображения заказов.")
			if errHelper == nil && sentMsg.MessageID != 0 {
				newMenuMessageID = sentMsg.MessageID
			}
		}

	case "operator_create_order_for_client": // Этот коллбэк теперь устарел, используется CALLBACK_PREFIX_OP_CREATE_NEW_ORDER
		if !utils.IsOperatorOrHigher(user.Role) {
			sentMsg, _ = bh.sendAccessDenied(chatID, originalMessageID)
			if sentMsg.MessageID != 0 {
				newMenuMessageID = sentMsg.MessageID
			}
			return newMenuMessageID
		}
		bh.SendClientSelectionMenu(chatID, originalMessageID)
		newMenuMessageID = bh.Deps.SessionManager.GetTempOrder(chatID).CurrentMessageID

	case "select_client":
		if !utils.IsOperatorOrHigher(user.Role) {
			sentMsg, _ = bh.sendAccessDenied(chatID, originalMessageID)
			if sentMsg.MessageID != 0 {
				newMenuMessageID = sentMsg.MessageID
			}
			return newMenuMessageID
		}
		if len(parts) == 1 {
			clientChatID, err := strconv.ParseInt(parts[0], 10, 64)
			if err == nil {
				bh.handleOperatorSelectClient(chatID, clientChatID, originalMessageID)
				newMenuMessageID = bh.Deps.SessionManager.GetTempOrder(chatID).CurrentMessageID
			} else {
				log.Printf("[CALLBACK_ORDER_VM] Ошибка конвертации ClientChatID для 'select_client': '%s'. ChatID=%d", parts[0], chatID)
				sentMsg, errHelper = bh.sendErrorMessageHelper(chatID, originalMessageID, "Ошибка выбора клиента.")
				if errHelper == nil && sentMsg.MessageID != 0 {
					newMenuMessageID = sentMsg.MessageID
				}
			}
		} else {
			log.Printf("[CALLBACK_ORDER_VM] Некорректный формат для 'select_client': %v. ChatID=%d", parts, chatID)
			sentMsg, errHelper = bh.sendErrorMessageHelper(chatID, originalMessageID, "Ошибка выбора клиента.")
			if errHelper == nil && sentMsg.MessageID != 0 {
				newMenuMessageID = sentMsg.MessageID
			}
		}

	case "view_order":
		if len(parts) == 1 {
			orderID, err := strconv.Atoi(parts[0])
			if err == nil {
				sentMsg, _ = bh.SendViewOrderDetails(chatID, orderID, originalMessageID, false, user)
				if sentMsg.MessageID != 0 {
					newMenuMessageID = sentMsg.MessageID
				}
			} else {
				log.Printf("[CALLBACK_ORDER_VM] Ошибка конвертации OrderID для 'view_order': '%s'. ChatID=%d", parts[0], chatID)
				sentMsg, errHelper = bh.sendErrorMessageHelper(chatID, originalMessageID, "Ошибка просмотра заказа.")
				if errHelper == nil && sentMsg.MessageID != 0 {
					newMenuMessageID = sentMsg.MessageID
				}
			}
		} else {
			log.Printf("[CALLBACK_ORDER_VM] Некорректный формат для 'view_order': %v. ChatID=%d", parts, chatID)
			sentMsg, errHelper = bh.sendErrorMessageHelper(chatID, originalMessageID, "Ошибка просмотра заказа.")
			if errHelper == nil && sentMsg.MessageID != 0 {
				newMenuMessageID = sentMsg.MessageID
			}
		}

	case "view_order_ops":
		if len(parts) == 1 {
			orderID, err := strconv.Atoi(parts[0])
			if err == nil {
				isOpView := utils.IsOperatorOrHigher(user.Role)
				sentMsg, _ = bh.SendViewOrderDetails(chatID, orderID, originalMessageID, isOpView, user)
				if sentMsg.MessageID != 0 {
					newMenuMessageID = sentMsg.MessageID
				}
			} else {
				log.Printf("[CALLBACK_ORDER_VM] Ошибка конвертации OrderID для 'view_order_ops': '%s'. ChatID=%d", parts[0], chatID)
				sentMsg, errHelper = bh.sendErrorMessageHelper(chatID, originalMessageID, "Ошибка просмотра заказа (сотрудник).")
				if errHelper == nil && sentMsg.MessageID != 0 {
					newMenuMessageID = sentMsg.MessageID
				}
			}
		} else {
			log.Printf("[CALLBACK_ORDER_VM] Некорректный формат для 'view_order_ops': %v. ChatID=%d", parts, chatID)
			sentMsg, errHelper = bh.sendErrorMessageHelper(chatID, originalMessageID, "Ошибка просмотра заказа (сотрудник).")
			if errHelper == nil && sentMsg.MessageID != 0 {
				newMenuMessageID = sentMsg.MessageID
			}
		}

	case "set_cost": // Используется для установки стоимости существующего заказа оператором
		if !utils.IsOperatorOrHigher(user.Role) {
			sentMsg, _ = bh.sendAccessDenied(chatID, originalMessageID)
			if sentMsg.MessageID != 0 {
				newMenuMessageID = sentMsg.MessageID
			}
			return newMenuMessageID
		}
		if len(parts) == 1 {
			orderID, err := strconv.Atoi(parts[0])
			if err == nil {
				bh.SendCostInputPrompt(chatID, orderID, originalMessageID) // Устанавливает STATE_COST_INPUT
				newMenuMessageID = bh.Deps.SessionManager.GetTempOrder(chatID).CurrentMessageID
			} else {
				log.Printf("[CALLBACK_ORDER_VM] Ошибка конвертации OrderID для 'set_cost': '%s'. ChatID=%d", parts[0], chatID)
				sentMsg, errHelper = bh.sendErrorMessageHelper(chatID, originalMessageID, "Ошибка установки стоимости.")
				if errHelper == nil && sentMsg.MessageID != 0 {
					newMenuMessageID = sentMsg.MessageID
				}
			}
		} else {
			log.Printf("[CALLBACK_ORDER_VM] Некорректный формат для 'set_cost': %v. ChatID=%d", parts, chatID)
			sentMsg, errHelper = bh.sendErrorMessageHelper(chatID, originalMessageID, "Ошибка установки стоимости.")
			if errHelper == nil && sentMsg.MessageID != 0 {
				newMenuMessageID = sentMsg.MessageID
			}
		}
	// --- НАЧАЛО НОВЫХ ОБРАБОТЧИКОВ ДЛЯ РЕДАКТИРОВАНИЯ ОПЕРАТОРОМ ---
	case constants.CALLBACK_PREFIX_OP_EDIT_ORDER_COST: // op_edit_ord_cost_ORDERID
		if !utils.IsOperatorOrHigher(user.Role) {
			sentMsg, _ = bh.sendAccessDenied(chatID, originalMessageID)
			if sentMsg.MessageID != 0 {
				newMenuMessageID = sentMsg.MessageID
			}
			return newMenuMessageID
		}
		if len(parts) == 1 { // parts[0] = ORDERID
			orderID, err := strconv.Atoi(parts[0])
			if err == nil {
				log.Printf("[CALLBACK_ORDER_VM] Оператор %d редактирует стоимость заказа #%d", chatID, orderID)
				// Убедимся, что мы в контексте редактирования этого заказа
				tempOrder := bh.Deps.SessionManager.GetTempOrder(chatID)
				if tempOrder.ID != int64(orderID) || bh.Deps.SessionManager.GetState(chatID) != constants.STATE_ORDER_EDIT {
					log.Printf("[CALLBACK_ORDER_VM] Контекст редактирования заказа #%d потерян или неверен. Попытка восстановить.", orderID)
					// Пытаемся войти в меню редактирования заказа, чтобы восстановить сессию
					bh.handleEditOrderStart(chatID, user, parts[0], originalMessageID) // parts[0] это orderIDStr
					// После этого, если успешно, CurrentMessageID обновится. Вызываем SendCostInputPrompt уже с ним.
					updatedTempOrder := bh.Deps.SessionManager.GetTempOrder(chatID)
					bh.SendCostInputPrompt(chatID, orderID, updatedTempOrder.CurrentMessageID)
				} else {
					bh.SendCostInputPrompt(chatID, orderID, originalMessageID)
				}
				newMenuMessageID = bh.Deps.SessionManager.GetTempOrder(chatID).CurrentMessageID
			} else {
				log.Printf("[CALLBACK_ORDER_VM] Ошибка конвертации OrderID для '%s': '%s'. ChatID=%d", currentCommand, parts[0], chatID)
				sentMsg, errHelper = bh.sendErrorMessageHelper(chatID, originalMessageID, "Ошибка: неверный ID заказа.")
				if errHelper == nil && sentMsg.MessageID != 0 {
					newMenuMessageID = sentMsg.MessageID
				}
			}
		} else {
			log.Printf("[CALLBACK_ORDER_VM] Некорректный формат для '%s': %v. ChatID=%d", currentCommand, parts, chatID)
			sentMsg, errHelper = bh.sendErrorMessageHelper(chatID, originalMessageID, "Ошибка: неверный формат команды.")
			if errHelper == nil && sentMsg.MessageID != 0 {
				newMenuMessageID = sentMsg.MessageID
			}
		}
	case constants.CALLBACK_PREFIX_OP_EDIT_ORDER_EXECS: // op_edit_ord_execs_ORDERID
		if !utils.IsOperatorOrHigher(user.Role) {
			sentMsg, _ = bh.sendAccessDenied(chatID, originalMessageID)
			if sentMsg.MessageID != 0 {
				newMenuMessageID = sentMsg.MessageID
			}
			return newMenuMessageID
		}
		if len(parts) == 1 { // parts[0] = ORDERID
			orderID, err := strconv.ParseInt(parts[0], 10, 64) // Используем ParseInt
			if err == nil {
				log.Printf("[CALLBACK_ORDER_VM] Оператор %d редактирует исполнителей для заказа #%d", chatID, orderID)
				tempOrder := bh.Deps.SessionManager.GetTempOrder(chatID)
				if tempOrder.ID != orderID || bh.Deps.SessionManager.GetState(chatID) != constants.STATE_ORDER_EDIT {
					log.Printf("[CALLBACK_ORDER_VM] Контекст редактирования заказа #%d для исполнителей потерян или неверен. Попытка восстановить.", orderID)
					bh.handleEditOrderStart(chatID, user, parts[0], originalMessageID)
					updatedTempOrder := bh.Deps.SessionManager.GetTempOrder(chatID)
					bh.SendAssignExecutorsMenu(chatID, orderID, updatedTempOrder.CurrentMessageID)
				} else {
					bh.SendAssignExecutorsMenu(chatID, orderID, originalMessageID)
				}
				newMenuMessageID = bh.Deps.SessionManager.GetTempOrder(chatID).CurrentMessageID
			} else {
				log.Printf("[CALLBACK_ORDER_VM] Ошибка конвертации OrderID для '%s': '%s'. ChatID=%d", currentCommand, parts[0], chatID)
				sentMsg, errHelper = bh.sendErrorMessageHelper(chatID, originalMessageID, "Ошибка: неверный ID заказа.")
				if errHelper == nil && sentMsg.MessageID != 0 {
					newMenuMessageID = sentMsg.MessageID
				}
			}
		} else {
			log.Printf("[CALLBACK_ORDER_VM] Некорректный формат для '%s': %v. ChatID=%d", currentCommand, parts, chatID)
			sentMsg, errHelper = bh.sendErrorMessageHelper(chatID, originalMessageID, "Ошибка: неверный формат команды.")
			if errHelper == nil && sentMsg.MessageID != 0 {
				newMenuMessageID = sentMsg.MessageID
			}
		}
		// --- КОНЕЦ НОВЫХ ОБРАБОТЧИКОВ ---
	case "assign_executors": // Используется для просмотра/управления исполнителями существующего заказа
		if !utils.IsOperatorOrHigher(user.Role) {
			sentMsg, _ = bh.sendAccessDenied(chatID, originalMessageID)
			if sentMsg.MessageID != 0 {
				newMenuMessageID = sentMsg.MessageID
			}
			return newMenuMessageID
		}
		if len(parts) == 1 {
			orderID, err := strconv.ParseInt(parts[0], 10, 64) // ИЗМЕНЕНО: Atoi на ParseInt
			if err == nil {
				bh.SendAssignExecutorsMenu(chatID, orderID, originalMessageID) // ИЗМЕНЕНО: теперь передаем int64
				newMenuMessageID = bh.Deps.SessionManager.GetTempOrder(chatID).CurrentMessageID
			} else {
				log.Printf("[CALLBACK_ORDER_VM] Ошибка конвертации OrderID для 'assign_executors': '%s'. ChatID=%d", parts[0], chatID)
				sentMsg, errHelper = bh.sendErrorMessageHelper(chatID, originalMessageID, "Ошибка назначения исполнителей.")
				if errHelper == nil && sentMsg.MessageID != 0 {
					newMenuMessageID = sentMsg.MessageID
				}
			}
		} else {
			log.Printf("[CALLBACK_ORDER_VM] Некорректный формат для 'assign_executors': %v. ChatID=%d", parts, chatID)
			sentMsg, errHelper = bh.sendErrorMessageHelper(chatID, originalMessageID, "Ошибка назначения исполнителей.")
			if errHelper == nil && sentMsg.MessageID != 0 {
				newMenuMessageID = sentMsg.MessageID
			}
		}
	case "assign_driver", "assign_loader":
		if !utils.IsOperatorOrHigher(user.Role) {
			sentMsg, _ = bh.sendAccessDenied(chatID, originalMessageID)
			if sentMsg.MessageID != 0 {
				newMenuMessageID = sentMsg.MessageID
			}
			return newMenuMessageID
		}
		if len(parts) == 2 {
			orderID, errOrder := strconv.Atoi(parts[0])
			executorChatID, errExec := strconv.ParseInt(parts[1], 10, 64)
			if errOrder == nil && errExec == nil {
				roleToAssign := constants.ROLE_DRIVER
				if currentCommand == "assign_loader" {
					roleToAssign = constants.ROLE_LOADER
				}
				bh.handleAssignExecutor(chatID, orderID, executorChatID, roleToAssign, originalMessageID)
				newMenuMessageID = bh.Deps.SessionManager.GetTempOrder(chatID).CurrentMessageID
			} else {
				log.Printf("[CALLBACK_ORDER_VM] Ошибка конвертации ID для '%s': OrderID='%s', ExecutorChatID='%s'. ChatID=%d", currentCommand, parts[0], parts[1], chatID)
				sentMsg, errHelper = bh.sendErrorMessageHelper(chatID, originalMessageID, "Ошибка назначения исполнителя (неверные ID).")
				if errHelper == nil && sentMsg.MessageID != 0 {
					newMenuMessageID = sentMsg.MessageID
				}
			}
		} else {
			log.Printf("[CALLBACK_ORDER_VM] Некорректный формат для '%s': %v. ChatID=%d", currentCommand, parts, chatID)
			sentMsg, errHelper = bh.sendErrorMessageHelper(chatID, originalMessageID, "Ошибка назначения исполнителя.")
			if errHelper == nil && sentMsg.MessageID != 0 {
				newMenuMessageID = sentMsg.MessageID
			}
		}
	case "unassign_executor":
		if !utils.IsOperatorOrHigher(user.Role) {
			sentMsg, _ = bh.sendAccessDenied(chatID, originalMessageID)
			if sentMsg.MessageID != 0 {
				newMenuMessageID = sentMsg.MessageID
			}
			return newMenuMessageID
		}
		if len(parts) == 2 {
			orderID, errOrder := strconv.Atoi(parts[0])
			executorChatID, errExec := strconv.ParseInt(parts[1], 10, 64)
			if errOrder == nil && errExec == nil {
				bh.handleUnassignExecutor(chatID, orderID, executorChatID, originalMessageID)
				newMenuMessageID = bh.Deps.SessionManager.GetTempOrder(chatID).CurrentMessageID
			} else {
				log.Printf("[CALLBACK_ORDER_VM] Ошибка конвертации ID для 'unassign_executor': OrderID='%s', ExecutorChatID='%s'. ChatID=%d", parts[0], parts[1], chatID)
				sentMsg, errHelper = bh.sendErrorMessageHelper(chatID, originalMessageID, "Ошибка снятия исполнителя (неверные ID).")
				if errHelper == nil && sentMsg.MessageID != 0 {
					newMenuMessageID = sentMsg.MessageID
				}
			}
		} else {
			log.Printf("[CALLBACK_ORDER_VM] Некорректный формат для 'unassign_executor': %v. ChatID=%d", parts, chatID)
			sentMsg, errHelper = bh.sendErrorMessageHelper(chatID, originalMessageID, "Ошибка снятия исполнителя.")
			if errHelper == nil && sentMsg.MessageID != 0 {
				newMenuMessageID = sentMsg.MessageID
			}
		}
	case constants.CALLBACK_PREFIX_ORDER_SET_FINAL_COST:
		if len(parts) == 1 {
			orderIDStr := parts[0]
			orderID, err := strconv.Atoi(orderIDStr)
			if err == nil {
				// --- НАЧАЛО ИЗМЕНЕНИЯ: Проверка прав доступа ---
				isOperator := utils.IsOperatorOrHigher(user.Role)
				isAssignedDriver := false
				if !isOperator && user.Role == constants.ROLE_DRIVER {
					execs, _ := db.GetExecutorsByOrderID(orderID)
					for _, exec := range execs {
						if exec.UserID == user.ID && exec.Role == constants.ROLE_DRIVER {
							isAssignedDriver = true
							break
						}
					}
				}
				if !isOperator && !isAssignedDriver {
					sentMsg, _ = bh.sendAccessDenied(chatID, originalMessageID)
					if sentMsg.MessageID != 0 {
						newMenuMessageID = sentMsg.MessageID
					}
					return newMenuMessageID
				}
				// --- КОНЕЦ ИЗМЕНЕНИЯ ---
				bh.handleSetFinalCostPrompt(chatID, user, orderID, originalMessageID)
				newMenuMessageID = bh.Deps.SessionManager.GetTempOrder(chatID).CurrentMessageID
			} else {
				log.Printf("[CALLBACK_DISPATCHER] Ошибка конвертации OrderID для '%s': '%s'. ChatID=%d", currentCommand, orderIDStr, chatID)
				sentMsg, errHelper = bh.sendErrorMessageHelper(chatID, originalMessageID, "Ошибка: неверный ID заказа.")
				if errHelper == nil && sentMsg.MessageID != 0 {
					newMenuMessageID = sentMsg.MessageID
				}
			}
		} else {
			log.Printf("[CALLBACK_DISPATCHER] Некорректный формат для '%s': %v. Ожидался ID заказа. ChatID=%d", currentCommand, parts, chatID)
			sentMsg, errHelper = bh.sendErrorMessageHelper(chatID, originalMessageID, "Ошибка: неверный формат команды.")
			if errHelper == nil && sentMsg.MessageID != 0 {
				newMenuMessageID = sentMsg.MessageID
			}
		}

	case constants.CALLBACK_PREFIX_ORDER_RESUME:
		if !utils.IsOperatorOrHigher(user.Role) {
			sentMsg, _ = bh.sendAccessDenied(chatID, originalMessageID)
			if sentMsg.MessageID != 0 {
				newMenuMessageID = sentMsg.MessageID
			}
			return newMenuMessageID
		}
		if len(parts) == 1 {
			orderIDStr := parts[0]
			orderID, err := strconv.Atoi(orderIDStr)
			if err == nil {
				bh.handleResumeOrder(chatID, user, orderID, originalMessageID)
				newMenuMessageID = bh.Deps.SessionManager.GetTempOrder(chatID).CurrentMessageID
			} else {
				log.Printf("[CALLBACK_DISPATCHER] Ошибка конвертации OrderID для '%s': '%s'. ChatID=%d", currentCommand, orderIDStr, chatID)
				sentMsg, errHelper = bh.sendErrorMessageHelper(chatID, originalMessageID, "Ошибка: неверный ID заказа для возобновления.")
				if errHelper == nil && sentMsg.MessageID != 0 {
					newMenuMessageID = sentMsg.MessageID
				}
			}
		} else {
			log.Printf("[CALLBACK_DISPATCHER] Некорректный формат для '%s': %v. Ожидался ID заказа. ChatID=%d", currentCommand, parts, chatID)
			sentMsg, errHelper = bh.sendErrorMessageHelper(chatID, originalMessageID, "Ошибка: неверный формат команды возобновления.")
			if errHelper == nil && sentMsg.MessageID != 0 {
				newMenuMessageID = sentMsg.MessageID
			}
		}
	case constants.CALLBACK_PREFIX_MARK_ORDER_DONE:
		if len(parts) == 1 {
			orderID, err := strconv.Atoi(parts[0])
			if err == nil {
				bh.handleMarkOrderCompleted(chatID, user, orderID, originalMessageID)
				newMenuMessageID = bh.Deps.SessionManager.GetTempOrder(chatID).CurrentMessageID
			} else {
				log.Printf("[CALLBACK_ORDER_VM] Ошибка конвертации OrderID для '%s': '%s'. ChatID=%d", currentCommand, parts[0], chatID)
				sentMsg, errHelper = bh.sendErrorMessageHelper(chatID, originalMessageID, "Ошибка: неверный ID заказа.")
				if errHelper == nil && sentMsg.MessageID != 0 {
					newMenuMessageID = sentMsg.MessageID
				}
			}
		} else {
			log.Printf("[CALLBACK_ORDER_VM] Некорректный формат для '%s': %v. ChatID=%d", currentCommand, parts, chatID)
			sentMsg, errHelper = bh.sendErrorMessageHelper(chatID, originalMessageID, "Ошибка отметки заказа как выполненного.")
			if errHelper == nil && sentMsg.MessageID != 0 {
				newMenuMessageID = sentMsg.MessageID
			}
		}
	case constants.CALLBACK_PREFIX_EXECUTOR_NOTIFIED: // exec_notified_ORDERID_EXECUTORUSERID
		if query == nil { // Добавлена проверка на nil
			log.Printf("[CALLBACK_ORDER_VM] Ошибка: query is nil для '%s'. ChatID=%d", currentCommand, chatID)
			// Отправляем ответ по умолчанию, так как queryID неизвестен
			// Либо можно просто выйти, если это некритично
			return newMenuMessageID
		}
		if len(parts) == 2 {
			orderID, errOrder := strconv.Atoi(parts[0])
			executorUserID, errExec := strconv.ParseInt(parts[1], 10, 64)

			if errOrder == nil && errExec == nil {
				errNotify := db.MarkExecutorAsNotified(orderID, executorUserID)
				answerCallbackText := ""
				if errNotify == nil {
					answerCallbackText = "✅ Вы отмечены как уведомленный по этому заказу."
					// Обновляем только кнопку, текст сообщения не меняем
					if query.Message != nil { // Используем query.Message
						editedMarkup := tgbotapi.NewInlineKeyboardMarkup(
							tgbotapi.NewInlineKeyboardRow(
								tgbotapi.NewInlineKeyboardButtonData("✅ Уведомление подтверждено", "noop_informational"),
							),
							tgbotapi.NewInlineKeyboardRow(
								tgbotapi.NewInlineKeyboardButtonData("📋 Детали заказа", fmt.Sprintf("view_order_ops_%d", orderID)),
							),
						)
						editMsg := tgbotapi.NewEditMessageReplyMarkup(chatID, originalMessageID, editedMarkup)
						_, errEdit := bh.Deps.BotClient.Request(editMsg)
						if errEdit != nil {
							log.Printf("[CALLBACK_HANDLER] Ошибка изменения клавиатуры после уведомления исполнителя: %v", errEdit)
						}
					}
				} else {
					log.Printf("[CALLBACK_HANDLER] Ошибка отметки уведомления исполнителя UserID %d для заказа #%d: %v", executorUserID, orderID, errNotify)
					answerCallbackText = "⚠️ Ошибка при подтверждении уведомления."
				}
				// Отвечаем на callback query
				cbAns := tgbotapi.NewCallback(queryID, answerCallbackText) // Используем queryID
				cbAns.ShowAlert = false                                    // Можно true, если это важное уведомление
				if _, errAns := bh.Deps.BotClient.Request(cbAns); errAns != nil {
					log.Printf("[CALLBACK_HANDLER] Ошибка ответа на CallbackQuery ID %s для exec_notified: %v", queryID, errAns)
				}
			} else {
				log.Printf("[CALLBACK_ORDER_VM] Ошибка парсинга ID для '%s': %v", currentCommand, parts)
				if queryID != "" { // Проверяем, что queryID есть
					cbAns := tgbotapi.NewCallback(queryID, "❌ Ошибка: неверные параметры подтверждения.")
					bh.Deps.BotClient.Request(cbAns)
				}
			}
		} else {
			log.Printf("[CALLBACK_ORDER_VM] Некорректный формат для '%s': %v", currentCommand, parts)
			if queryID != "" { // Проверяем, что queryID есть
				cbAns := tgbotapi.NewCallback(queryID, "❌ Ошибка: неверный формат команды подтверждения.")
				bh.Deps.BotClient.Request(cbAns)
			}
		}
		// newMenuMessageID остается originalMessageID, так как мы только редактируем его клавиатуру или отвечаем на коллбэк

	default:
		log.Printf("[CALLBACK_ORDER_VM] ОШИБКА: Неизвестная команда '%s' передана в dispatchOrderViewManageCallbacks. Data: '%s', ChatID=%d", currentCommand, data, chatID)
		sentMsg, errHelper = bh.sendErrorMessageHelper(chatID, originalMessageID, "Неизвестная команда управления заказами.")
		if errHelper == nil && sentMsg.MessageID != 0 {
			newMenuMessageID = sentMsg.MessageID
		}
	}

	log.Printf("[CALLBACK_ORDER_VM] Диспетчер просмотра/управления заказами завершен. Команда='%s', ChatID=%d, ID нового меню=%d", currentCommand, chatID, newMenuMessageID)
	return newMenuMessageID
}

// handleOperatorSelectClient обрабатывает выбор клиента оператором для создания заказа.
func (bh *BotHandler) handleOperatorSelectClient(operatorChatID int64, clientChatID int64, originalMessageID int) {
	log.Printf("[ORDER_VM_HANDLER] Оператор %d выбрал клиента %d для создания заказа.", operatorChatID, clientChatID)

	clientUser, ok := bh.getUserFromDB(clientChatID)
	if !ok {
		_, _ = bh.sendErrorMessageHelper(operatorChatID, originalMessageID, "Не удалось получить данные выбранного клиента.")
		bh.SendClientSelectionMenu(operatorChatID, originalMessageID)
		return
	}

	// Очищаем предыдущее состояние и временный заказ, если он был
	bh.Deps.SessionManager.ClearState(operatorChatID)
	bh.Deps.SessionManager.ClearTempOrder(operatorChatID)

	// Начинаем новый временный заказ для этого оператора, но с указанием UserChatID клиента
	tempOrderForClient := session.NewTempOrder(clientChatID)                   // Устанавливаем UserChatID клиента
	tempOrderForClient.OrderAction = "operator_creating_order"                 // Ставим флаг, что это заказ от оператора
	bh.Deps.SessionManager.UpdateTempOrder(operatorChatID, tempOrderForClient) // Сохраняем под ChatID оператора

	// Устанавливаем состояние, указывающее, что оператор в процессе создания заказа
	bh.Deps.SessionManager.SetState(operatorChatID, constants.STATE_OP_CREATE_ORDER_FLOW)

	// Имя для приветствия в SendCategoryMenu будет имя клиента
	bh.SendCategoryMenu(operatorChatID, clientUser.FirstName, originalMessageID)
}

// handleAssignExecutor назначает исполнителя на заказ.
func (bh *BotHandler) handleAssignExecutor(operatorChatID int64, orderID int, executorChatID int64, role string, originalMessageID int) {
	log.Printf("[ORDER_VM_HANDLER] Оператор %d назначает исполнителя %d (роль: %s) на заказ #%d", operatorChatID, executorChatID, role, orderID)

	executorUser, okUser := bh.getUserFromDB(executorChatID)
	if !okUser {
		log.Printf("[ORDER_VM_HANDLER] Не удалось получить данные исполнителя (ChatID: %d) для назначения.", executorChatID)
		_, _ = bh.sendErrorMessageHelper(operatorChatID, originalMessageID, fmt.Sprintf("❌ Ошибка: не найден исполнитель с ChatID %d.", executorChatID))
		bh.SendAssignExecutorsMenu(operatorChatID, int64(orderID), originalMessageID)
		return
	}

	err := db.AssignExecutor(orderID, executorChatID, role)
	if err != nil {
		log.Printf("[ORDER_VM_HANDLER] Ошибка назначения исполнителя: %v", err)
		_, _ = bh.sendErrorMessageHelper(operatorChatID, originalMessageID, fmt.Sprintf("❌ Ошибка назначения: %s", err.Error()))
		bh.SendAssignExecutorsMenu(operatorChatID, int64(orderID), originalMessageID)
		return
	}

	// Отправляем детализированное уведомление исполнителю
	go bh.sendTaskNotificationToExecutor(executorUser, orderID)

	// Обновляем меню оператора
	bh.SendAssignExecutorsMenu(operatorChatID, int64(orderID), originalMessageID)
}

// sendTaskNotificationToExecutor формирует и отправляет сообщение с заданием для исполнителя.
func (bh *BotHandler) sendTaskNotificationToExecutor(executor models.User, orderID int) {
	order, errOrder := db.GetOrderByID(orderID)
	if errOrder != nil {
		log.Printf("[TASK_NOTIFY] Ошибка получения деталей заказа #%d для уведомления исполнителя %d: %v", orderID, executor.ChatID, errOrder)
		return
	}

	client, errClient := db.GetUserByID(order.UserID)
	if errClient != nil {
		log.Printf("[TASK_NOTIFY] Ошибка получения клиента заказа #%d для уведомления исполнителя %d: %v", orderID, executor.ChatID, errClient)
		// Продолжаем без данных клиента, если это приемлемо
	}

	brigade, errBrigade := db.GetExecutorsByOrderID(orderID)
	if errBrigade != nil {
		log.Printf("[TASK_NOTIFY] Ошибка получения бригады для заказа #%d: %v", orderID, errBrigade)
	}

	// Формируем сообщение с помощью нового форматера
	notificationText := formatters.FormatTaskForExecutor(order, client, brigade)

	confirmNotificationCallback := fmt.Sprintf("%s_%d_%d", constants.CALLBACK_PREFIX_EXECUTOR_NOTIFIED, orderID, executor.ID)
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("✅ Уведомление получил", confirmNotificationCallback),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📋 Посмотреть детали в боте", fmt.Sprintf("view_order_ops_%d", orderID)),
		),
	)

	msgToSend := tgbotapi.NewMessage(executor.ChatID, notificationText)
	msgToSend.ParseMode = tgbotapi.ModeMarkdown
	msgToSend.ReplyMarkup = keyboard
	_, errSend := bh.Deps.BotClient.Send(msgToSend)

	if errSend != nil {
		log.Printf("[TASK_NOTIFY] Ошибка отправки уведомления о задании исполнителю %d по заказу #%d: %v", executor.ChatID, orderID, errSend)
	} else {
		log.Printf("[TASK_NOTIFY] Уведомление о задании по заказу #%d успешно отправлено исполнителю %s (ChatID: %d)", orderID, executor.FirstName, executor.ChatID)
	}
}

// handleUnassignExecutor снимает исполнителя с заказа.
func (bh *BotHandler) handleUnassignExecutor(operatorChatID int64, orderID int, executorChatID int64, originalMessageID int) {
	log.Printf("[ORDER_VM_HANDLER] Оператор %d снимает исполнителя %d с заказа #%d", operatorChatID, executorChatID, orderID)

	err := db.RemoveExecutor(orderID, executorChatID)
	if err != nil {
		log.Printf("[ORDER_VM_HANDLER] Ошибка снятия исполнителя: %v", err)
		_, _ = bh.sendErrorMessageHelper(operatorChatID, originalMessageID, fmt.Sprintf("❌ Ошибка снятия: %s", err.Error()))
		bh.SendAssignExecutorsMenu(operatorChatID, int64(orderID), originalMessageID)
		return
	}
	_, ok := bh.getUserFromDB(executorChatID)
	if ok {
		bh.sendMessage(executorChatID, fmt.Sprintf("ℹ️ Вас сняли с заказа №%d.", orderID))
	}
	bh.SendAssignExecutorsMenu(operatorChatID, int64(orderID), originalMessageID)
}

// handleMarkOrderCompleted обрабатывает нажатие кнопки "Заказ выполнен".
func (bh *BotHandler) handleMarkOrderCompleted(chatID int64, user models.User, orderID int, originalMessageID int) {
	log.Printf("[ORDER_VM_HANDLER] Пользователь UserID %d (Роль: %s) отметил заказ #%d как выполненный.", user.ID, user.Role, orderID)

	order, err := db.GetOrderByID(orderID)
	if err != nil {
		log.Printf("handleMarkOrderCompleted: Ошибка получения заказа #%d: %v", orderID, err)
		bh.sendErrorMessageHelper(chatID, originalMessageID, "❌ Ошибка: не удалось найти заказ.")
		return
	}

	isAssignedDriver := false
	if user.Role == constants.ROLE_DRIVER {
		execs, _ := db.GetExecutorsByOrderID(orderID)
		for _, exec := range execs {
			if exec.UserID == user.ID && exec.Role == constants.ROLE_DRIVER {
				isAssignedDriver = true
				break
			}
		}
	}

	if !(utils.IsOperatorOrHigher(user.Role) || isAssignedDriver) {
		log.Printf("handleMarkOrderCompleted: Пользователь UserID %d (Роль: %s) не имеет прав отметить заказ #%d как выполненный.", user.ID, user.Role, orderID)
		bh.sendAccessDenied(chatID, originalMessageID)
		return
	}

	if order.Status != constants.STATUS_INPROGRESS {
		log.Printf("handleMarkOrderCompleted: Попытка отметить выполненным заказ #%d, который не в статусе 'В работе' (статус: %s).", orderID, order.Status)
		bh.sendInfoMessage(chatID, originalMessageID, fmt.Sprintf("ℹ️ Заказ №%d уже находится в статусе '%s'.", orderID, constants.StatusDisplayMap[order.Status]), fmt.Sprintf("view_order_ops_%d", orderID))
		return
	}

	errUpdate := db.UpdateOrderStatus(int64(orderID), constants.STATUS_COMPLETED)
	if errUpdate != nil {
		log.Printf("handleMarkOrderCompleted: Ошибка обновления статуса заказа #%d на ВЫПОЛНЕН: %v", orderID, errUpdate)
		bh.sendErrorMessageHelper(chatID, originalMessageID, "❌ Ошибка при обновлении статуса заказа.")
		return
	}

	log.Printf("Заказ #%d успешно переведен в статус ВЫПОЛНЕН пользователем UserID %d.", orderID, user.ID)

	if order.UserChatID != 0 && order.UserChatID != chatID {
		clientMsg := fmt.Sprintf("✅ Ваш заказ №%d выполнен! Спасибо за использование нашего сервиса!", orderID)
		bh.sendMessage(order.UserChatID, clientMsg)
	}

	if user.Role != constants.ROLE_OWNER { // Уведомляем владельца и гл.операторов, если не они сами закрыли
		ownerAndMainOps, _ := db.GetUsersByRole(constants.ROLE_OWNER, constants.ROLE_MAINOPERATOR)
		notificationText := fmt.Sprintf("✅ Заказ №%d был отмечен как выполненный пользователем %s (Роль: %s, ChatID: %d).",
			orderID, utils.GetUserDisplayName(user), utils.GetRoleDisplayName(user.Role), user.ChatID)
		for _, op := range ownerAndMainOps {
			if op.ChatID != chatID { // Не отправляем самому себе, если это гл.оператор
				bh.sendMessage(op.ChatID, notificationText)
			}
		}
		// Уведомление в группу, если она есть и это не тот же чат, откуда пришло действие
		if bh.Deps.Config.GroupChatID != 0 && bh.Deps.Config.GroupChatID != chatID {
			bh.sendMessage(bh.Deps.Config.GroupChatID, notificationText)
		}
	}

	bh.sendInfoMessage(chatID, originalMessageID, fmt.Sprintf("✅ Заказ №%d успешно отмечен как выполненный!", orderID), fmt.Sprintf("view_order_ops_%d", orderID))
	bh.SendViewOrderDetails(chatID, orderID, originalMessageID, true, user)
}

// handleSetFinalCostPrompt запрашивает у оператора итоговую стоимость выполненного заказа.
func (bh *BotHandler) handleSetFinalCostPrompt(chatID int64, user models.User, orderID int, originalMessageID int) {
	log.Printf("handleSetFinalCostPrompt: Запрос на изменение итоговой стоимости для заказа #%d оператором %d", orderID, chatID)

	order, err := db.GetOrderByID(orderID)
	if err != nil || order.Status != constants.STATUS_COMPLETED {
		bh.sendErrorMessageHelper(chatID, originalMessageID, "❌ Заказ не найден или еще не выполнен.")
		return
	}

	bh.Deps.SessionManager.SetState(chatID, constants.STATE_ORDER_FINAL_COST_INPUT)
	tempData := bh.Deps.SessionManager.GetTempOrder(chatID)
	tempData.ID = int64(orderID)
	tempData.CurrentMessageID = originalMessageID // Важно для редактирования этого сообщения
	bh.Deps.SessionManager.UpdateTempOrder(chatID, tempData)

	promptText := fmt.Sprintf("💰 Введите новую итоговую стоимость для заказа №%d (текущая: %.0f ₽).\nЭто изменение только для внутреннего учета, клиенту уведомление не придет.", orderID, order.Cost.Float64)
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("⬅️ Назад к заказу", fmt.Sprintf("view_order_ops_%d", orderID)),
		),
	)
	bh.sendOrEditMessageHelper(chatID, originalMessageID, promptText, &keyboard, "")
}

// handleResumeOrder возобновляет отмененный заказ.
func (bh *BotHandler) handleResumeOrder(chatID int64, user models.User, orderID int, originalMessageID int) {
	log.Printf("handleResumeOrder: Возобновление заказа #%d оператором %d", orderID, chatID)

	order, err := db.GetOrderByID(orderID)
	if err != nil || order.Status != constants.STATUS_CANCELED {
		bh.sendErrorMessageHelper(chatID, originalMessageID, "❌ Заказ не найден или не находится в статусе 'отменен'.")
		return
	}

	// При возобновлении, заказ переходит в "новые", чтобы оператор мог его обработать заново
	errUpdate := db.UpdateOrderStatusAndReason(int64(orderID), constants.STATUS_NEW, sql.NullString{Valid: false}) // Сбрасываем причину отмены
	if errUpdate != nil {
		log.Printf("handleResumeOrder: Ошибка возобновления заказа #%d: %v", orderID, errUpdate)
		bh.sendErrorMessageHelper(chatID, originalMessageID, "❌ Ошибка возобновления заказа.")
		return
	}

	bh.sendInfoMessage(chatID, originalMessageID, fmt.Sprintf("✅ Заказ №%d успешно возобновлен и переведен в 'новые'.", orderID), fmt.Sprintf("view_order_ops_%d", orderID))
	bh.SendViewOrderDetails(chatID, orderID, originalMessageID, true, user)
}
