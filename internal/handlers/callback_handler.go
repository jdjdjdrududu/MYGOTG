package handlers

import (
	"Original/internal/constants"
	"Original/internal/models"
	"Original/internal/utils"
	"fmt"
	"log"
	"sort"
	"strconv"
	"strings"
	"time"

	tgbotapi "github.com/OvyFlash/telegram-bot-api"
)

// HandleCallback обрабатывает входящие callback query от Telegram.
func (bh *BotHandler) HandleCallback(update tgbotapi.Update) {
	query := update.CallbackQuery // query объявлен здесь
	if query == nil {
		log.Println("[CALLBACK_HANDLER] Получен пустой CallbackQuery.")
		return
	}

	chatID := query.Message.Chat.ID
	originalMessageID := query.Message.MessageID
	data := query.Data
	queryID := query.ID // queryID объявлен здесь

	log.Printf("[CALLBACK_HANDLER] START: ChatID=%d, User=%s, OriginalMsgID=%d, Data='%s'",
		chatID, query.From.UserName, originalMessageID, data)

	answerText := ""
	if strings.HasPrefix(data, constants.CALLBACK_PREFIX_EXECUTOR_NOTIFIED) {
		// Ответ будет дан после обработки в dispatchOrderViewManageCallbacks
	} else if data == "noop" {
		answerText = "" // Не показываем текст для noop
	} else if data == "noop_informational" {
		answerText = "✅" // Для информационных noop, которые просто убирают кнопки
	} else {
		// Для большинства других коллбэков можно отправить пустой ответ
		// или специфичный текст, если это уместно.
		// answerText = "Processing..."
	}

	if !strings.HasPrefix(data, constants.CALLBACK_PREFIX_EXECUTOR_NOTIFIED) {
		callbackAns := tgbotapi.NewCallback(queryID, answerText)
		if _, err := bh.Deps.BotClient.Request(callbackAns); err != nil {
			log.Printf("[CALLBACK_HANDLER] Ошибка ответа на CallbackQuery ID %s: %v. Продолжаем.", queryID, err)
		}
	}

	tempDataForEphemeralCleanup := bh.Deps.SessionManager.GetTempOrder(chatID)
	if len(tempDataForEphemeralCleanup.EphemeralMediaMessageIDs) > 0 {
		log.Printf("[CALLBACK_HANDLER] Обнаружено %d ephemeral media сообщений для ChatID=%d. Попытка удаления.", len(tempDataForEphemeralCleanup.EphemeralMediaMessageIDs), chatID)
		for _, ephemeralMsgID := range tempDataForEphemeralCleanup.EphemeralMediaMessageIDs {
			if ephemeralMsgID != originalMessageID {
				bh.deleteMessageHelper(chatID, ephemeralMsgID)
			}
		}
		tempDataForEphemeralCleanup.EphemeralMediaMessageIDs = make([]int, 0)
		bh.Deps.SessionManager.UpdateTempOrder(chatID, tempDataForEphemeralCleanup)
		log.Printf("[CALLBACK_HANDLER] Ephemeral media сообщения для ChatID=%d очищены.", chatID)
	}

	user, ok := bh.getUserFromDB(chatID)
	if !ok {
		log.Printf("[CALLBACK_HANDLER] КРИТИЧЕСКАЯ ОШИБКА: не удалось получить пользователя для ChatID=%d. Data: '%s'.", chatID, data)
		bh.sendErrorMessageHelper(chatID, 0, "Произошла ошибка с данными пользователя. Попробуйте /start.")
		return
	}

	if user.IsBlocked {
		log.Printf("[CALLBACK_HANDLER] Пользователь ChatID=%d (Роль: %s) заблокирован. Коллбэк '%s' проигнорирован.", chatID, user.Role, data)
		bh.sendErrorMessageHelper(chatID, originalMessageID, "Ваш аккаунт заблокирован.")
		return
	}

	var finalActiveMessageID int = originalMessageID

	currentStateForCleanup := bh.Deps.SessionManager.GetState(chatID)
	isOrderCreationFlowForCleanup := strings.HasPrefix(currentStateForCleanup, "order_") ||
		utils.IsCommandInCategory(data, []string{"view_uploaded_media", "reset_photo_upload", "finish_photo_upload"}) ||
		(strings.HasPrefix(currentStateForCleanup, "edit_") && !strings.HasPrefix(currentStateForCleanup, "edit_field_"))

	if isOrderCreationFlowForCleanup {
		tempOrderDataSession := bh.Deps.SessionManager.GetTempOrder(chatID)
		if originalMessageID != 0 && tempOrderDataSession.CurrentMessageID == originalMessageID && len(tempOrderDataSession.MediaMessageIDs) > 0 {
			var idsToDeleteThisTurn []int
			for _, mediaID := range tempOrderDataSession.MediaMessageIDs {
				if mediaID != originalMessageID && mediaID != 0 {
					if !bh.Deps.SessionManager.IsMessageDeleted(chatID, mediaID) {
						idsToDeleteThisTurn = append(idsToDeleteThisTurn, mediaID)
					}
				}
			}
			if len(idsToDeleteThisTurn) > 0 {
				log.Printf("[CALLBACK_HANDLER] Удаление %d вспомогательных TempOrder сообщений для ChatID=%d. Список ID: %v", len(idsToDeleteThisTurn), chatID, idsToDeleteThisTurn)
				for _, msgIDToDelete := range idsToDeleteThisTurn {
					bh.deleteMessageHelper(chatID, msgIDToDelete)
				}
			}
			tempOrderDataSession.MediaMessageIDs = []int{originalMessageID}
			tempOrderDataSession.MediaMessageIDsMap = make(map[string]bool)
			tempOrderDataSession.MediaMessageIDsMap[fmt.Sprintf("%d", originalMessageID)] = true
			bh.Deps.SessionManager.UpdateTempOrder(chatID, tempOrderDataSession)
		}
	}

	partsRaw := strings.Split(data, "_")
	if len(partsRaw) == 0 {
		log.Printf("[CALLBACK_HANDLER] Ошибка: Некорректный формат callback (пусто): '%s' для ChatID=%d", data, chatID)
		bh.sendErrorMessageHelper(chatID, originalMessageID, "❌ Некорректный запрос.")
		return
	}

	var currentCommand string
	var remainingParts []string
	foundCmd := false

	explicitCompleteCommands := map[string]bool{
		// --- НАЧАЛО ИЗМЕНЕНИЯ 1 ---
		// Добавляем новую команду для обработки нажатия кнопки "Продолжить в боте"
		constants.CALLBACK_CONTINUE_IN_BOT: true,
		// --- КОНЕЦ ИЗМЕНЕНИЯ 1 ---
		"category_waste": true, "category_demolition": true,
		"subcategory_construct": true, "subcategory_household": true, "subcategory_metal": true, "subcategory_junk": true,
		"subcategory_greenery": true, "subcategory_tires": true, "subcategory_other_waste": true,
		"subcategory_walls": true, "subcategory_partitions": true, "subcategory_floors": true, "subcategory_ceilings": true,
		"subcategory_plumbing": true, "subcategory_tiles": true, "subcategory_other_demo": true,
		"use_profile_name_for_order": true, "enter_another_name_for_order": true,
		"skip_photo_initial": true, "finish_photo_upload": true, "reset_photo_upload": true,
		"payment_now": true, "payment_later": true, "send_location_prompt": true,
		"confirm_order_name": true, "confirm_order_phone": true,
		"confirm_order_description_placeholder": true,
		"skip_order_description_placeholder":    true,
		"change_order_phone":                    true, "view_uploaded_media": true,
		"invite_friend": true, "contact_operator": true, "contact_chat": true, "contact_phone_options": true,
		"client_chats": true, "materials_soon": true, "subscribe_materials_updates": true,
		"referral_link": true, "referral_qr": true, "referral_my": true, "request_referral_payout": true,
		"phone_action_request_call": true, "phone_action_call_self": true,
		"staff_menu": true, "staff_list_menu": true, "staff_add_prompt_name": true, "staff_add_prompt_card_number": true,
		"stats_menu": true, "stats_basic_periods": true,
		"stats_select_custom_date": true, "stats_select_custom_period": true,
		"stats_get_today":         true,
		"stats_get_yesterday":     true,
		"stats_get_current_week":  true,
		"stats_get_current_month": true,
		"stats_get_last_week":     true,
		"stats_get_last_month":    true,
		"send_excel_menu":         true, "excel_generate_orders": true, "excel_generate_referrals": true, "excel_generate_salaries": true,
		"block_user_menu": true, "block_user_list_prompt": true, "unblock_user_list_prompt": true,
		constants.CALLBACK_PREFIX_MY_SALARY:          true,
		constants.CALLBACK_PREFIX_OWNER_STAFF_PAYOUT: true,
		"manage_orders":                                                              true,
		"operator_create_order_for_client":                                           true, // Устарело, заменено на CALLBACK_PREFIX_OP_CREATE_NEW_ORDER
		constants.CALLBACK_PREFIX_OWNER_FINANCIALS:                                   true,
		constants.CALLBACK_PREFIX_OWNER_CASH_MANAGEMENT_MAIN:                         true,
		"back_to_main_confirm_cancel_order":                                          true,
		"back_to_main_confirm_cancel_driver_settlement":                              true,
		"back_to_main_confirmed_cancel_final":                                        true,
		"noop":                                                                       true,
		"noop_informational":                                                         true,
		"select_date_asap":                                                           true,
		constants.CALLBACK_PREFIX_DRIVER_SETTLEMENT:                                  true,
		constants.CALLBACK_PREFIX_DRIVER_REPORT_OVERALL_MENU:                         true,
		constants.CALLBACK_PREFIX_DRIVER_REPORT_SET_FUEL:                             true,
		constants.CALLBACK_PREFIX_DRIVER_REPORT_OTHER_EXPENSES_MENU:                  true,
		constants.CALLBACK_PREFIX_DRIVER_REPORT_ADD_OTHER_EXPENSE_DESCRIPTION_PROMPT: true,
		constants.CALLBACK_PREFIX_DRIVER_REPORT_LOADERS_MENU:                         true,
		constants.CALLBACK_PREFIX_DRIVER_REPORT_ADD_LOADER_PROMPT:                    true,
		constants.CALLBACK_PREFIX_DRIVER_REPORT_SAVE_FINAL:                           true,
		constants.CALLBACK_PREFIX_DRIVER_REPORT_CANCEL_ALL:                           true,
		// Новые операторские команды
		constants.CALLBACK_PREFIX_OP_CREATE_NEW_ORDER: true,
		constants.CALLBACK_PREFIX_DRIVER_CREATE_ORDER: true,
	}

	prefixedCommandsWithParams := map[string]int{
		"confirm_order_final": 3, "select_date": 2, "edit_order": 2, "edit_field_description": 3,
		"edit_field_name": 3, "edit_field_subcategory": 3, "edit_field_date": 3, "edit_field_time": 3, "edit_field_phone": 3,
		"edit_field_address": 3, "edit_field_media": 3, "edit_field_payment": 3, "accept_cost": 2, "reject_cost": 2,
		"cancel_order_operator": 3, "cancel_order_confirm": 3, "operator_orders_new": 3, "operator_orders_awaiting_confirmation": 4,
		"operator_orders_in_progress": 4, "operator_orders_completed": 3, "operator_orders_canceled": 3, "operator_orders_calculated": 3,
		"my_orders_page": 3, "select_client": 2, "view_order": 2, "view_order_ops": 3, "set_cost": 2,
		"assign_executors": 2, "assign_driver": 2, "assign_loader": 2, "unassign_executor": 2,
		constants.CALLBACK_PREFIX_MARK_ORDER_DONE: 3, constants.CALLBACK_PREFIX_ORDER_SET_FINAL_COST: 4, constants.CALLBACK_PREFIX_ORDER_RESUME: 2,
		"view_chat_history": 3, "referral_details": 2, "staff_list_by_role": 4, "staff_add_role_final": 4, "staff_info": 2,
		"staff_edit_menu": 3, "staff_edit_field_name": 4, "staff_edit_field_surname": 4, "staff_edit_field_nickname": 4,
		"staff_edit_field_phone": 4, "staff_edit_field_card_number": 5, "staff_edit_field_role": 4, "staff_edit_role_final": 4,
		"staff_block_reason_prompt": 4, "staff_unblock_confirm": 3, "staff_delete_confirm": 3, "stats_select_month": 3,
		"stats_select_day": 3, "stats_year_nav": 3, "block_user_info": 3, "block_user_reason_prompt": 4,
		"block_user_final": 3, "unblock_user_info": 3, "unblock_user_final": 3,
		fmt.Sprintf("%s_owed", constants.CALLBACK_PREFIX_MY_SALARY):                          3,
		fmt.Sprintf("%s_earned_stats", constants.CALLBACK_PREFIX_MY_SALARY):                  4,
		fmt.Sprintf("%s_page", constants.CALLBACK_PREFIX_OWNER_STAFF_PAYOUT):                 3,
		fmt.Sprintf("%s_select", constants.CALLBACK_PREFIX_OWNER_STAFF_PAYOUT):               3,
		fmt.Sprintf("%s_confirm", constants.CALLBACK_PREFIX_OWNER_STAFF_PAYOUT):              3,
		fmt.Sprintf("%s_date", constants.CALLBACK_PREFIX_OWNER_FINANCIALS):                   3,
		fmt.Sprintf("%s_view", constants.CALLBACK_PREFIX_OWNER_FINANCIALS):                   3,
		fmt.Sprintf("%s_edit_settlement", constants.CALLBACK_PREFIX_OWNER_FINANCIALS):        4,
		fmt.Sprintf("%s_edit_field", constants.CALLBACK_PREFIX_OWNER_FINANCIALS):             4,
		fmt.Sprintf("%s_save_edited_settlement", constants.CALLBACK_PREFIX_OWNER_FINANCIALS): 5,
		constants.CALLBACK_PREFIX_OWNER_CASH_ACTUAL_LIST:                                     4,
		constants.CALLBACK_PREFIX_OWNER_CASH_SETTLED_LIST:                                    4,
		constants.CALLBACK_PREFIX_OWNER_CASH_MARK_PAID:                                       4,
		constants.CALLBACK_PREFIX_OWNER_CASH_MARK_UNPAID:                                     4,
		constants.CALLBACK_PREFIX_OWNER_VIEW_DRIVER_SETTLEMENTS:                              5,
		constants.CALLBACK_PREFIX_OWNER_EDIT_SETTLEMENT:                                      4,
		constants.CALLBACK_PREFIX_OWNER_CASH_MARK_SALARY_PAID:                                5,
		constants.CALLBACK_PREFIX_OWNER_CASH_MARK_SALARY_UNPAID:                              5,
		constants.CALLBACK_PREFIX_OWNER_MARK_ALL_SALARY_PAID:                                 5,
		constants.CALLBACK_PREFIX_OWNER_MARK_ALL_MONEY_DEPOSITED:                             5,
		constants.CALLBACK_PREFIX_OPERATOR_VIEW_DRIVER_SETTLEMENT:                            4,
		"date_page": 2, "resume_order_creation": 3,
		constants.CALLBACK_PREFIX_DRIVER_REPORT_EDIT_LOADER_PROMPT:                5,
		constants.CALLBACK_PREFIX_DRIVER_REPORT_DELETE_LOADER_CONFIRM:             5,
		constants.CALLBACK_PREFIX_DRIVER_REPORT_EDIT_OTHER_EXPENSE_PROMPT:         5,
		constants.CALLBACK_PREFIX_DRIVER_REPORT_DELETE_OTHER_EXPENSE_SHOW_CONFIRM: 5,
		constants.CALLBACK_PREFIX_DRIVER_REPORT_DELETE_OTHER_EXPENSE_CONFIRM:      5,
		// Новые операторские команды с параметрами
		constants.CALLBACK_PREFIX_OP_CONFIRM_ORDER_SIMPLE_CREATE: 3, // op_confirm_simple (3 части) + _ORDERID
		constants.CALLBACK_PREFIX_OP_CONFIRM_ORDER_SET_COST:      4, // op_confirm_set_cost (4 части) + _ORDERID
		constants.CALLBACK_PREFIX_OP_CONFIRM_ORDER_ASSIGN_EXEC:   4, // op_confirm_assign_exec (4 части) + _ORDERID
		constants.CALLBACK_PREFIX_OP_SKIP_COST:                   3, // op_skip_cost (3 части) + _ORDERID
		constants.CALLBACK_PREFIX_OP_SKIP_ASSIGN_EXEC:            4, // op_skip_assign_exec (4 части) + _ORDERID
		constants.CALLBACK_PREFIX_OP_FINALIZE_ORDER_CREATION:     3, // op_finalize_creation (3 части) + _ORDERID
		constants.CALLBACK_PREFIX_OP_EDIT_ORDER_COST:             4, // op_edit_ord_cost (4 части) + _ORDERID
		constants.CALLBACK_PREFIX_OP_EDIT_ORDER_EXECS:            4, // op_edit_ord_execs (4 части) + _ORDERID
		constants.CALLBACK_PREFIX_EXECUTOR_NOTIFIED:              2, // exec_notified_ORDERID_EXECUTORUSERID (старый префикс, новая логика)
		constants.CALLBACK_PREFIX_SELECT_HOUR:                    2, // select_hour_HOUR
		"select_time":                                            2, // select_time_HH:MM
		constants.CALLBACK_PREFIX_PAY_ORDER:                      2, // pay_order_ORDERID
	}

	if explicitCompleteCommands[data] {
		currentCommand = data
		remainingParts = []string{}
		foundCmd = true
	} else if strings.HasPrefix(data, "back_to_") {
		currentCommand = "back"
		// "back_to_" это 8 символов. Если data = "back_to_main", то partsRaw[2:] будет ошибкой.
		// Нужно правильно извлекать параметры для "back_to_".
		if len(data) > len("back_to_") {
			destinationAndParams := data[len("back_to_"):]
			remainingParts = strings.Split(destinationAndParams, "_")
		} else {
			remainingParts = []string{} // на случай если просто "back_to_"
		}
		foundCmd = true
	} else {
		sortedPrefixedKeys := make([]string, 0, len(prefixedCommandsWithParams))
		for k := range prefixedCommandsWithParams {
			sortedPrefixedKeys = append(sortedPrefixedKeys, k)
		}
		sort.Slice(sortedPrefixedKeys, func(i, j int) bool {
			lenPartsI := prefixedCommandsWithParams[sortedPrefixedKeys[i]]
			lenPartsJ := prefixedCommandsWithParams[sortedPrefixedKeys[j]]
			if lenPartsI != lenPartsJ {
				return lenPartsI > lenPartsJ
			}
			return len(sortedPrefixedKeys[i]) > len(sortedPrefixedKeys[j])
		})

		for _, cmdPrefix := range sortedPrefixedKeys {
			if strings.HasPrefix(data, cmdPrefix+"_") || data == cmdPrefix {
				// Проверяем, совпадает ли начало data с cmdPrefix
				// и что количество частей в data соответствует или больше, чем в cmdPrefix
				// (с учетом, что параметры идут после '_')

				// Собираем префикс из partsRaw для сравнения
				numPartsInPrefixKey := strings.Count(cmdPrefix, "_") + 1
				if len(partsRaw) >= numPartsInPrefixKey {
					potentialCmdFromData := strings.Join(partsRaw[:numPartsInPrefixKey], "_")
					if potentialCmdFromData == cmdPrefix {
						currentCommand = cmdPrefix
						if len(partsRaw) > numPartsInPrefixKey {
							remainingParts = partsRaw[numPartsInPrefixKey:]
						} else {
							remainingParts = []string{}
						}
						foundCmd = true
						break
					}
				}
			}
		}

		if !foundCmd {
			if explicitCompleteCommands[strings.Join(partsRaw, "_")] {
				currentCommand = strings.Join(partsRaw, "_")
				remainingParts = []string{}
				foundCmd = true
			} else {
				currentCommand = partsRaw[0]
				if len(partsRaw) > 1 {
					remainingParts = partsRaw[1:]
				} else {
					remainingParts = []string{}
				}
			}
		}
	}

	log.Printf("[CALLBACK_HANDLER] Определена команда: '%s', Оставшиеся части: %v, ChatID=%d", currentCommand, remainingParts, chatID)

	isDispatched := false
	switch currentCommand {
	// --- НАЧАЛО ИЗМЕНЕНИЯ 2 ---
	// Добавляем обработку нового коллбэка
	case constants.CALLBACK_CONTINUE_IN_BOT:
		// Пользователь нажал "Продолжить в боте", показываем ему главное меню
		bh.SendMainMenu(chatID, user, originalMessageID)
		isDispatched = true
	// --- КОНЕЦ ИЗМЕНЕНИЯ 2 ---
	case "back":
		// destination - это то, что идет после "back_to_"
		destination := ""
		if len(remainingParts) > 0 {
			destination = strings.Join(remainingParts, "_")
		}
		bh.handleBackCallback(chatID, user, "back_to_"+destination, originalMessageID)
		isDispatched = true
	case "back_to_main_confirm_cancel_order":
		// Если отмену инициировал оператор или выше, пропускаем подтверждение
		// и сразу возвращаем в главное меню.
		if utils.IsOperatorOrHigher(user.Role) {
			log.Printf("[CALLBACK_HANDLER] Отмена создания заказа оператором (ChatID=%d, Role=%s). Пропуск подтверждения.", chatID, user.Role)
			bh.Deps.SessionManager.ClearState(chatID)
			bh.Deps.SessionManager.ClearTempOrder(chatID)
			bh.Deps.SessionManager.ClearTempDriverSettlement(chatID)
			bh.SendMainMenu(chatID, user, originalMessageID)
		} else {
			// Для обычных пользователей оставляем шаг подтверждения.
			bh.SendAskToCancelOrderConfirmation(chatID, originalMessageID, originalMessageID)
		}
		isDispatched = true
	case "back_to_main_confirm_cancel_driver_settlement":
		bh.SendAskToCancelDriverSettlementConfirmation(chatID, originalMessageID, originalMessageID)
		isDispatched = true
	case "back_to_main_confirmed_cancel_final":
		log.Printf("[CALLBACK_HANDLER] Отмена и возврат в главное меню подтверждены для chatID %d.", chatID)
		bh.Deps.SessionManager.ClearState(chatID)
		bh.Deps.SessionManager.ClearTempOrder(chatID)
		bh.Deps.SessionManager.ClearTempDriverSettlement(chatID)
		bh.SendMainMenu(chatID, user, originalMessageID)
		isDispatched = true
	case "resume_order_creation":
		if len(remainingParts) == 1 {
			originalStepMessageIDFromCallback, err := strconv.Atoi(remainingParts[0])
			if err == nil {
				bh.resumeOrderCreation(chatID, user, originalStepMessageIDFromCallback, originalMessageID)
				isDispatched = true
			} else {
				log.Printf("[CALLBACK_HANDLER] Ошибка resume_order_creation: некорректный originalStepMessageID. Data: '%s'", data)
			}
		} else {
			log.Printf("[CALLBACK_HANDLER] Ошибка resume_order_creation: неверное количество параметров. Data: '%s'", data)
		}
		if !isDispatched {
			bh.sendErrorMessageHelper(chatID, originalMessageID, "Ошибка возобновления операции.")
			bh.SendMainMenu(chatID, user, originalMessageID)
			isDispatched = true
		}
	case "noop", "noop_informational":
		log.Printf("[CALLBACK_HANDLER] Команда 'noop' или 'noop_informational'. ChatID=%d, Data='%s'. Действий не требуется.", chatID, data)
		isDispatched = true
	case constants.CALLBACK_PREFIX_OP_CREATE_NEW_ORDER:
		bh.handleOperatorStartOrderCreation(chatID, user, originalMessageID)
		isDispatched = true
	case constants.CALLBACK_PREFIX_DRIVER_CREATE_ORDER:
		bh.handleDriverStartOrderCreation(chatID, user, originalMessageID)
		isDispatched = true
	case constants.CALLBACK_PREFIX_OP_CONFIRM_ORDER_SIMPLE_CREATE,
		constants.CALLBACK_PREFIX_OP_CONFIRM_ORDER_SET_COST,
		constants.CALLBACK_PREFIX_OP_CONFIRM_ORDER_ASSIGN_EXEC,
		constants.CALLBACK_PREFIX_OP_SKIP_COST,
		constants.CALLBACK_PREFIX_OP_SKIP_ASSIGN_EXEC,
		constants.CALLBACK_PREFIX_OP_FINALIZE_ORDER_CREATION:
		finalActiveMessageID = bh.dispatchOrderCallbacks(currentCommand, remainingParts, data, chatID, user, originalMessageID)
		isDispatched = true

	case constants.CALLBACK_PREFIX_EXECUTOR_NOTIFIED:
		// Передаем query в dispatchOrderViewManageCallbacks
		finalActiveMessageID = bh.dispatchOrderViewManageCallbacks(query, currentCommand, remainingParts, data, chatID, user, originalMessageID)
		isDispatched = true
	default:
		adminDispatchableItems := []string{
			constants.CALLBACK_PREFIX_MY_SALARY, constants.CALLBACK_PREFIX_OWNER_STAFF_PAYOUT,
			constants.CALLBACK_PREFIX_OWNER_FINANCIALS,
			constants.CALLBACK_PREFIX_OWNER_CASH_MANAGEMENT_MAIN,
			constants.CALLBACK_PREFIX_OWNER_CASH_ACTUAL_LIST,
			constants.CALLBACK_PREFIX_OWNER_CASH_SETTLED_LIST,
			constants.CALLBACK_PREFIX_OWNER_CASH_MARK_PAID,
			constants.CALLBACK_PREFIX_OWNER_CASH_MARK_UNPAID,
			constants.CALLBACK_PREFIX_OWNER_VIEW_DRIVER_SETTLEMENTS,
			constants.CALLBACK_PREFIX_OWNER_EDIT_SETTLEMENT,
			constants.CALLBACK_PREFIX_OWNER_CASH_MARK_SALARY_PAID,
			constants.CALLBACK_PREFIX_OWNER_CASH_MARK_SALARY_UNPAID,
			constants.CALLBACK_PREFIX_OWNER_MARK_ALL_SALARY_PAID,
			constants.CALLBACK_PREFIX_OWNER_MARK_ALL_MONEY_DEPOSITED,
			constants.CALLBACK_PREFIX_OPERATOR_VIEW_DRIVER_SETTLEMENT,
			"staff_menu", "staff_list_menu", "staff_list_by_role", "staff_add_prompt_name", "staff_add_role_final", "staff_info", "staff_edit_menu", "staff_edit_role_final", "staff_block_reason_prompt", "staff_unblock_confirm", "staff_delete_confirm", "staff_edit_field_name", "staff_edit_field_surname", "staff_edit_field_nickname", "staff_edit_field_phone", "staff_edit_field_card_number", "staff_edit_field_role", "staff_add_prompt_card_number",
			"stats_menu", "stats_basic_periods", "stats_get_today", "stats_get_yesterday", "stats_get_current_week", "stats_get_current_month", "stats_get_last_week", "stats_get_last_month", "stats_select_custom_date", "stats_select_custom_period", "stats_select_month", "stats_select_day", "stats_year_nav",
			"send_excel_menu", "excel_generate_orders", "excel_generate_referrals", "excel_generate_salaries",
			"block_user_menu", "block_user_list_prompt", "block_user_info", "block_user_reason_prompt", "block_user_final", "unblock_user_list_prompt", "unblock_user_info", "unblock_user_final",
			constants.CALLBACK_PREFIX_DRIVER_SETTLEMENT,
			constants.CALLBACK_PREFIX_DRIVER_REPORT_OVERALL_MENU, constants.CALLBACK_PREFIX_DRIVER_REPORT_SET_FUEL,
			constants.CALLBACK_PREFIX_DRIVER_REPORT_OTHER_EXPENSES_MENU,
			constants.CALLBACK_PREFIX_DRIVER_REPORT_ADD_OTHER_EXPENSE_DESCRIPTION_PROMPT,
			constants.CALLBACK_PREFIX_DRIVER_REPORT_EDIT_OTHER_EXPENSE_PROMPT,
			constants.CALLBACK_PREFIX_DRIVER_REPORT_DELETE_OTHER_EXPENSE_SHOW_CONFIRM,
			constants.CALLBACK_PREFIX_DRIVER_REPORT_DELETE_OTHER_EXPENSE_CONFIRM,
			constants.CALLBACK_PREFIX_DRIVER_REPORT_LOADERS_MENU,
			constants.CALLBACK_PREFIX_DRIVER_REPORT_ADD_LOADER_PROMPT,
			constants.CALLBACK_PREFIX_DRIVER_REPORT_EDIT_LOADER_PROMPT,
			constants.CALLBACK_PREFIX_DRIVER_REPORT_DELETE_LOADER_CONFIRM,
			constants.CALLBACK_PREFIX_DRIVER_REPORT_SAVE_FINAL,
			constants.CALLBACK_PREFIX_DRIVER_REPORT_CANCEL_ALL,
		}

		ownerCashManagementCommands := []string{ // Эти команды уже есть в adminDispatchableItems, но для ясности можно оставить
			constants.CALLBACK_PREFIX_OWNER_CASH_MANAGEMENT_MAIN,
			constants.CALLBACK_PREFIX_OWNER_CASH_ACTUAL_LIST,
			constants.CALLBACK_PREFIX_OWNER_CASH_SETTLED_LIST,
			constants.CALLBACK_PREFIX_OWNER_CASH_MARK_PAID,
			constants.CALLBACK_PREFIX_OWNER_CASH_MARK_UNPAID,
			constants.CALLBACK_PREFIX_OWNER_VIEW_DRIVER_SETTLEMENTS,
			constants.CALLBACK_PREFIX_OWNER_EDIT_SETTLEMENT,
			constants.CALLBACK_PREFIX_OWNER_CASH_MARK_SALARY_PAID,
			constants.CALLBACK_PREFIX_OWNER_CASH_MARK_SALARY_UNPAID,
			constants.CALLBACK_PREFIX_OWNER_MARK_ALL_SALARY_PAID,
			constants.CALLBACK_PREFIX_OWNER_MARK_ALL_MONEY_DEPOSITED,
		}

		orderCreationDispatchableItems := []string{
			constants.CALLBACK_PREFIX_SELECT_HOUR,
			"date_page",
			"category_waste", "category_demolition", "subcategory_construct", "subcategory_household", "subcategory_metal", "subcategory_junk", "subcategory_greenery", "subcategory_tires", "subcategory_other_waste", "subcategory_walls", "subcategory_partitions", "subcategory_floors", "subcategory_ceilings", "subcategory_plumbing", "subcategory_tiles", "subcategory_other_demo",
			"use_profile_name_for_order", "enter_another_name_for_order",
			"skip_photo_initial", "finish_photo_upload", "reset_photo_upload",
			"payment_now", "payment_later", "send_location_prompt",
			"confirm_order_name", "confirm_order_phone",
			"confirm_order_description_placeholder", "skip_order_description_placeholder",
			"change_order_phone", "view_uploaded_media",
			"select_date_asap", "select_date", "select_time",
			"edit_order", "edit_field_description", "edit_field_name", "edit_field_subcategory", "edit_field_date", "edit_field_time",
			"edit_field_phone", "edit_field_address", "edit_field_media", "edit_field_payment",
			"confirm_order_final", "accept_cost", "reject_cost", "cancel_order_operator", "cancel_order_confirm",
			constants.CALLBACK_PREFIX_PAY_ORDER,
		}
		orderViewManageDispatchableItems := []string{
			"manage_orders", "operator_create_order_for_client", "select_client", "view_order", "view_order_ops",
			"set_cost", "assign_executors", "assign_driver", "assign_loader", "unassign_executor",
			"operator_orders_new", "operator_orders_awaiting_confirmation", "operator_orders_in_progress",
			"operator_orders_completed", "operator_orders_canceled", "operator_orders_calculated",
			constants.CALLBACK_PREFIX_MARK_ORDER_DONE, constants.CALLBACK_PREFIX_ORDER_SET_FINAL_COST,
			constants.CALLBACK_PREFIX_ORDER_RESUME, "my_orders_page",
			// Новые коллбэки для редактирования заказа оператором
			constants.CALLBACK_PREFIX_OP_EDIT_ORDER_COST,
			constants.CALLBACK_PREFIX_OP_EDIT_ORDER_EXECS,
		}
		infoCommsDispatchableItems := []string{
			"invite_friend", "contact_operator", "contact_chat", "contact_phone_options",
			"client_chats", "materials_soon", "subscribe_materials_updates", "referral_link",
			"referral_qr", "referral_my", "referral_details", "request_referral_payout",
			"phone_action_request_call", "phone_action_call_self", "view_chat_history",
		}

		if currentCommand == constants.CALLBACK_PREFIX_DRIVER_SETTLEMENT {
			if user.Role == constants.ROLE_DRIVER {
				log.Printf("[CALLBACK_HANDLER] Маршрутизация в StartDriverInlineReport для команды '%s'", currentCommand)
				bh.StartDriverInlineReport(chatID, user, originalMessageID)
				isDispatched = true
			} else {
				log.Printf("[CALLBACK_HANDLER] ОШИБКА: Команда '%s' предназначена для водителя, но вызвана пользователем с ролью '%s'. ChatID=%d", currentCommand, user.Role, chatID)
				bh.sendAccessDenied(chatID, originalMessageID)
				isDispatched = true
			}
		} else if utils.IsCommandInCategory(currentCommand, adminDispatchableItems) ||
			strings.HasPrefix(currentCommand, fmt.Sprintf("%s_", constants.CALLBACK_PREFIX_MY_SALARY)) ||
			strings.HasPrefix(currentCommand, fmt.Sprintf("%s_", constants.CALLBACK_PREFIX_OWNER_STAFF_PAYOUT)) ||
			strings.HasPrefix(currentCommand, fmt.Sprintf("%s_", constants.CALLBACK_PREFIX_OWNER_FINANCIALS)) ||
			utils.IsCommandInCategory(currentCommand, ownerCashManagementCommands) {
			finalActiveMessageID = bh.dispatchAdminCallbacks(currentCommand, remainingParts, data, chatID, user, originalMessageID)
			isDispatched = true
		} else if utils.IsCommandInCategory(currentCommand, orderCreationDispatchableItems) {
			finalActiveMessageID = bh.dispatchOrderCallbacks(currentCommand, remainingParts, data, chatID, user, originalMessageID)
			isDispatched = true
		} else if utils.IsCommandInCategory(currentCommand, orderViewManageDispatchableItems) {
			// ИЗМЕНЕНИЕ: передаем 'query' в dispatchOrderViewManageCallbacks
			finalActiveMessageID = bh.dispatchOrderViewManageCallbacks(query, currentCommand, remainingParts, data, chatID, user, originalMessageID)
			isDispatched = true
		} else if utils.IsCommandInCategory(currentCommand, infoCommsDispatchableItems) {
			finalActiveMessageID = bh.dispatchInfoCommsCallbacks(currentCommand, remainingParts, data, chatID, user, originalMessageID)
			isDispatched = true
		}

		if !isDispatched {
			var sentHelperMsg tgbotapi.Message // Для избежания конфликта с sentMsg из внешнего скоупа
			var errHelperDispatch error
			log.Printf("[CALLBACK_HANDLER] ОШИБКА (default): Неизвестная команда '%s' (data: '%s') от ChatID=%d.", currentCommand, data, chatID)
			sentHelperMsg, errHelperDispatch = bh.sendErrorMessageHelper(chatID, originalMessageID, "Неизвестная команда.")
			if errHelperDispatch == nil && sentHelperMsg.MessageID != 0 {
				finalActiveMessageID = sentHelperMsg.MessageID
			}
		}
	}

	sessionMsgIDAfterDispatch := 0
	currentHandlerStateAfterDispatch := bh.Deps.SessionManager.GetState(chatID)

	activeSessionIsDriverSettlement := strings.HasPrefix(currentHandlerStateAfterDispatch, "driver_report_") ||
		(strings.HasPrefix(currentHandlerStateAfterDispatch, "driver_expense_") && currentHandlerStateAfterDispatch != constants.STATE_DRIVER_REPORT_OTHER_EXPENSES_MENU) ||
		currentHandlerStateAfterDispatch == constants.STATE_OWNER_FINANCIAL_EDIT_FIELD ||
		currentHandlerStateAfterDispatch == constants.STATE_OWNER_FINANCIAL_EDIT_RECORD ||
		strings.HasPrefix(currentHandlerStateAfterDispatch, "owner_cash_")

	if activeSessionIsDriverSettlement {
		sessionMsgIDAfterDispatch = bh.Deps.SessionManager.GetTempDriverSettlement(chatID).CurrentMessageID
	} else {
		sessionMsgIDAfterDispatch = bh.Deps.SessionManager.GetTempOrder(chatID).CurrentMessageID
	}

	if isDispatched && sessionMsgIDAfterDispatch != 0 && finalActiveMessageID == originalMessageID && sessionMsgIDAfterDispatch != originalMessageID {
		finalActiveMessageID = sessionMsgIDAfterDispatch
		log.Printf("[CALLBACK_HANDLER] finalActiveMessageID обновлен из сессии на: %d", finalActiveMessageID)
	}

	if finalActiveMessageID != originalMessageID &&
		originalMessageID != 0 &&
		finalActiveMessageID != 0 &&
		currentCommand != "noop" && currentCommand != "noop_informational" &&
		!strings.HasPrefix(currentCommand, "excel_generate_") &&
		!strings.HasPrefix(currentCommand, constants.CALLBACK_PREFIX_EXECUTOR_NOTIFIED) { // Не удаляем сообщение, если это был ответ на exec_notified
		log.Printf("[CALLBACK_HANDLER] Удаление старого сообщения: ChatID=%d, Старый MsgID=%d (Новый активный MsgID: %d)",
			chatID, originalMessageID, finalActiveMessageID)
		bh.deleteMessageHelper(chatID, originalMessageID)
	}

	log.Printf("[CALLBACK_HANDLER] END: ChatID=%d, User=%s, Data='%s', Final Active Menu/Msg ID: %d (Session CurrentMsgID: %d)",
		chatID, query.From.UserName, data, finalActiveMessageID, sessionMsgIDAfterDispatch)
}

// sendAskToCancelDriverSettlementConfirmation запрашивает подтверждение отмены формирования отчета водителя.
func (bh *BotHandler) SendAskToCancelDriverSettlementConfirmation(chatID int64, messageIDToEdit int, originalStepMessageID int) {
	log.Printf("BotHandler.SendAskToCancelDriverSettlementConfirmation для chatID %d, ред.сообщ: %d, исх.сообщ.шага: %d", chatID, messageIDToEdit, originalStepMessageID)

	msgText := "Вы уверены, что хотите отменить формирование отчета и вернуться в главное меню?\n\n⚠️ Все введенные данные для этого отчета будут потеряны."
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("✅ Да, отменить", "back_to_main_confirmed_cancel_final"),
			tgbotapi.NewInlineKeyboardButtonData("❌ Нет, продолжить", fmt.Sprintf("resume_order_creation_%d", originalStepMessageID)),
		),
	)

	sentMsg, err := bh.sendOrEditMessageHelper(chatID, messageIDToEdit, msgText, &keyboard, tgbotapi.ModeMarkdown)
	if err == nil && sentMsg.MessageID != 0 {
		tempData := bh.Deps.SessionManager.GetTempDriverSettlement(chatID)
		tempData.CurrentMessageID = sentMsg.MessageID
		bh.Deps.SessionManager.UpdateTempDriverSettlement(chatID, tempData)
		log.Printf("SendAskToCancelDriverSettlementConfirmation: CurrentMessageID в TempDriverSettlement обновлен на %d", sentMsg.MessageID)
	} else if err != nil {
		log.Printf("SendAskToCancelDriverSettlementConfirmation: Ошибка отправки/редактирования: %v", err)
	}
}

// resumeOrderCreation восстанавливает предыдущее состояние пользователя.
func (bh *BotHandler) resumeOrderCreation(chatID int64, user models.User, originalStepMessageID int, currentDialogMessageID int) {
	log.Printf("[CALLBACK_RESUME] Начало возобновления. ChatID=%d, Исходный MsgID шага=%d, MsgID диалога отмены=%d", chatID, originalStepMessageID, currentDialogMessageID)

	history := bh.Deps.SessionManager.GetHistory(chatID)
	var previousMeaningfulState string

	for i := len(history) - 1; i >= 0; i-- {
		state := history[i]
		if state != constants.STATE_IDLE &&
			!strings.HasPrefix(state, "back_to_main_confirm_") &&
			state != constants.STATE_DRIVER_REPORT_CONFIRM_DELETE_LOADER &&
			state != constants.STATE_DRIVER_REPORT_CONFIRM_DELETE_OTHER_EXPENSE { // Добавлено
			previousMeaningfulState = state
			break
		}
	}

	if previousMeaningfulState == "" {
		if len(history) > 0 {
			previousMeaningfulState = history[len(history)-1]
		} else {
			previousMeaningfulState = constants.STATE_IDLE
		}
	}

	isDriverReportContext := strings.HasPrefix(previousMeaningfulState, "driver_report_") ||
		(strings.HasPrefix(previousMeaningfulState, "driver_expense_") && previousMeaningfulState != constants.STATE_DRIVER_REPORT_OTHER_EXPENSES_MENU) ||
		utils.IsCommandInCategory(previousMeaningfulState, []string{constants.CALLBACK_PREFIX_DRIVER_SETTLEMENT})

	if strings.HasPrefix(previousMeaningfulState, "back_to_main_confirm_") || originalStepMessageID == 0 {
		if isDriverReportContext {
			log.Printf("[CALLBACK_RESUME] Не удалось восстановить состояние для отчета водителя. Возврат к началу отчета. ChatID=%d", chatID)
			bh.StartDriverInlineReport(chatID, user, currentDialogMessageID)
		} else {
			log.Printf("[CALLBACK_RESUME] Ошибка: Не удалось найти корректное состояние для возобновления. ChatID=%d. Возврат в главное меню.", chatID)
			bh.SendMainMenu(chatID, user, currentDialogMessageID)
		}
		return
	}

	bh.Deps.SessionManager.SetState(chatID, previousMeaningfulState)
	log.Printf("[CALLBACK_RESUME] Восстановление состояния '%s' для ChatID=%d. Сообщение для редактирования/восстановления: MsgID=%d", previousMeaningfulState, chatID, originalStepMessageID)

	if isDriverReportContext {
		tempDriverSettlement := bh.Deps.SessionManager.GetTempDriverSettlement(chatID)
		tempDriverSettlement.CurrentMessageID = originalStepMessageID
		bh.Deps.SessionManager.UpdateTempDriverSettlement(chatID, tempDriverSettlement)
	} else {
		tempOrder := bh.Deps.SessionManager.GetTempOrder(chatID)
		tempOrder.CurrentMessageID = originalStepMessageID
		if originalStepMessageID != 0 {
			if tempOrder.MediaMessageIDsMap == nil { // Инициализация карты, если она nil
				tempOrder.MediaMessageIDsMap = make(map[string]bool)
			}
			foundInMedia := false
			for _, mid := range tempOrder.MediaMessageIDs {
				if mid == originalStepMessageID {
					foundInMedia = true
					break
				}
			}
			if !foundInMedia {
				tempOrder.MediaMessageIDs = append(tempOrder.MediaMessageIDs, originalStepMessageID)
			}
			tempOrder.MediaMessageIDsMap[fmt.Sprintf("%d", originalStepMessageID)] = true
		}
		bh.Deps.SessionManager.UpdateTempOrder(chatID, tempOrder)
	}

	if currentDialogMessageID != 0 && currentDialogMessageID != originalStepMessageID {
		log.Printf("[CALLBACK_RESUME] Удаление сообщения диалога отмены MsgID=%d. ChatID=%d", currentDialogMessageID, chatID)
		bh.deleteMessageHelper(chatID, currentDialogMessageID)
	}

	log.Printf("[CALLBACK_RESUME] Вызов Send...Menu для состояния '%s' с MsgID=%d. ChatID=%d", previousMeaningfulState, originalStepMessageID, chatID)

	switch previousMeaningfulState {
	case constants.STATE_ORDER_CATEGORY:
		bh.SendCategoryMenu(chatID, user.FirstName, originalStepMessageID)
	case constants.STATE_ORDER_SUBCATEGORY:
		bh.SendSubcategoryMenu(chatID, bh.Deps.SessionManager.GetTempOrder(chatID).Category, originalStepMessageID)
	case constants.STATE_ORDER_DESCRIPTION:
		bh.SendDescriptionInputMenu(chatID, originalStepMessageID)
	case constants.STATE_ORDER_NAME:
		bh.SendNameInputMenu(chatID, originalStepMessageID)
	case constants.STATE_ORDER_DATE:
		bh.SendDateSelectionMenu(chatID, originalStepMessageID, 0)
	case constants.STATE_ORDER_TIME:
		bh.SendTimeSelectionMenu(chatID, originalStepMessageID)
	case constants.STATE_ORDER_MINUTE_SELECTION: // Добавлено
		tempData := bh.Deps.SessionManager.GetTempOrder(chatID)
		selectedHour := tempData.SelectedHourForMinuteView
		if selectedHour == -1 { // Если час не был сохранен, возвращаем к выбору часа
			bh.SendTimeSelectionMenu(chatID, originalStepMessageID)
		} else {
			bh.SendMinuteSelectionMenu(chatID, selectedHour, originalStepMessageID)
		}
	case constants.STATE_ORDER_PHONE:
		bh.SendPhoneInputMenu(chatID, user, originalStepMessageID)
	case constants.STATE_ORDER_ADDRESS:
		bh.SendAddressInputMenu(chatID, originalStepMessageID)
	case constants.STATE_ORDER_ADDRESS_LOCATION:
		bh.SendAddressInputMenu(chatID, originalStepMessageID) // Возвращаем к общему вводу адреса
	case constants.STATE_ORDER_PHOTO:
		bh.SendPhotoInputMenu(chatID, originalStepMessageID)
	case constants.STATE_ORDER_PAYMENT:
		bh.SendPaymentSelectionMenu(chatID, originalStepMessageID)
	case constants.STATE_ORDER_CONFIRM:
		bh.SendOrderConfirmationMenu(chatID, originalStepMessageID)
	case constants.STATE_ORDER_EDIT:
		bh.SendEditOrderMenu(chatID, originalStepMessageID)
	case constants.STATE_OP_ORDER_CONFIRMATION_OPTIONS: // Для оператора
		bh.SendOrderConfirmationMenu(chatID, originalStepMessageID)
	case constants.STATE_OP_ORDER_COST_INPUT: // Для оператора
		tempOrder := bh.Deps.SessionManager.GetTempOrder(chatID)
		bh.SendOpOrderCostInputMenu(chatID, tempOrder.ID, originalStepMessageID)
	case constants.STATE_OP_ORDER_ASSIGN_EXEC_MENU: // Для оператора
		tempOrder := bh.Deps.SessionManager.GetTempOrder(chatID)
		bh.SendOpAssignExecutorsMenu(chatID, tempOrder.ID, originalStepMessageID)
	case constants.STATE_OP_ORDER_FINAL_CONFIRM: // Для оператора
		tempOrder := bh.Deps.SessionManager.GetTempOrder(chatID)
		bh.SendOpOrderFinalConfirmMenu(chatID, tempOrder.ID, originalStepMessageID)

	case constants.STATE_DRIVER_REPORT_OVERALL_MENU, constants.CALLBACK_PREFIX_DRIVER_SETTLEMENT:
		bh.SendDriverReportOverallMenu(chatID, user, originalStepMessageID)
	case constants.STATE_DRIVER_REPORT_INPUT_FUEL:
		bh.SendDriverReportFuelInputPrompt(chatID, user, originalStepMessageID)
	case constants.STATE_DRIVER_REPORT_OTHER_EXPENSES_MENU: // Добавлено
		bh.SendDriverReportOtherExpensesMenu(chatID, user, originalStepMessageID)
	case constants.STATE_DRIVER_REPORT_INPUT_OTHER_EXPENSE_DESCRIPTION: // Добавлено
		tempData := bh.Deps.SessionManager.GetTempDriverSettlement(chatID)
		bh.SendDriverReportOtherExpenseDescriptionPrompt(chatID, user, originalStepMessageID, tempData.EditingOtherExpenseIndex != -1, tempData.EditingOtherExpenseIndex)
	case constants.STATE_DRIVER_REPORT_INPUT_OTHER_EXPENSE_AMOUNT: // Добавлено
		tempData := bh.Deps.SessionManager.GetTempDriverSettlement(chatID)
		bh.SendDriverReportOtherExpenseAmountPrompt(chatID, user, originalStepMessageID, tempData.TempOtherExpenseDescription, tempData.EditingOtherExpenseIndex != -1, tempData.EditingOtherExpenseIndex)
	case constants.STATE_DRIVER_REPORT_LOADERS_MENU:
		bh.SendDriverReportLoadersSubMenu(chatID, user, originalStepMessageID)
	case constants.STATE_DRIVER_REPORT_INPUT_LOADER_NAME:
		bh.SendDriverReportLoaderNameInputPrompt(chatID, user, originalStepMessageID)
	case constants.STATE_DRIVER_REPORT_INPUT_LOADER_SALARY:
		tempData := bh.Deps.SessionManager.GetTempDriverSettlement(chatID)
		bh.SendDriverReportLoaderSalaryInputPrompt(chatID, user, originalStepMessageID, tempData.TempLoaderNameInput, false, -1)
	case constants.STATE_DRIVER_REPORT_EDIT_LOADER_SALARY:
		tempData := bh.Deps.SessionManager.GetTempDriverSettlement(chatID)
		if tempData.EditingLoaderIndex >= 0 && tempData.EditingLoaderIndex < len(tempData.LoaderPayments) {
			loaderToEdit := tempData.LoaderPayments[tempData.EditingLoaderIndex]
			bh.SendDriverReportLoaderSalaryInputPrompt(chatID, user, originalStepMessageID, loaderToEdit.LoaderIdentifier, true, tempData.EditingLoaderIndex)
		} else {
			bh.SendDriverReportLoadersSubMenu(chatID, user, originalStepMessageID)
		}
	case constants.STATE_DRIVER_REPORT_CONFIRM_DELETE_LOADER: // После подтверждения удаления, возвращаемся к списку
		bh.SendDriverReportLoadersSubMenu(chatID, user, originalStepMessageID)
	case constants.STATE_DRIVER_REPORT_CONFIRM_DELETE_OTHER_EXPENSE: // После подтверждения удаления, возвращаемся к списку
		bh.SendDriverReportOtherExpensesMenu(chatID, user, originalStepMessageID)

	case constants.STATE_CONTACT_METHOD:
		bh.SendContactOperatorMenu(chatID, user, originalStepMessageID)
	default:
		log.Printf("[CALLBACK_RESUME] Ошибка: Неизвестное или необрабатываемое состояние '%s' для возобновления. Возврат в главное меню. ChatID=%d", previousMeaningfulState, chatID)
		bh.SendMainMenu(chatID, user, originalStepMessageID)
	}

	var restoredMenuID int
	if isDriverReportContext {
		restoredMenuID = bh.Deps.SessionManager.GetTempDriverSettlement(chatID).CurrentMessageID
	} else {
		restoredMenuID = bh.Deps.SessionManager.GetTempOrder(chatID).CurrentMessageID
	}
	log.Printf("[CALLBACK_RESUME] Завершение возобновления. ChatID=%d. Восстановленное/новое меню MsgID=%d.", chatID, restoredMenuID)
}

// handleBackCallback обрабатывает коллбэк "Назад".
func (bh *BotHandler) handleBackCallback(chatID int64, user models.User, data string, originalMessageID int) {
	log.Printf("[CALLBACK_BACK] Начало обработки 'Назад'. ChatID=%d, Data='%s', Исходный MsgID=%d", chatID, data, originalMessageID)

	currentState := bh.Deps.SessionManager.GetState(chatID)
	if currentState == constants.STATE_ORDER_PHONE ||
		currentState == constants.STATE_PHONE_AWAIT_INPUT ||
		currentState == constants.STATE_ORDER_ADDRESS_LOCATION {
		replyMarkupRemove := tgbotapi.NewRemoveKeyboard(true)
		msgToRemoveKb := tgbotapi.NewMessage(chatID, constants.InvisibleMessage)
		msgToRemoveKb.ReplyMarkup = replyMarkupRemove
		sentInvisibleMsg, errSendInvisible := bh.Deps.BotClient.Send(msgToRemoveKb)
		if errSendInvisible == nil {
			go func(id int) {
				time.Sleep(200 * time.Millisecond)
				bh.deleteMessageHelper(chatID, id)
			}(sentInvisibleMsg.MessageID)
		} else {
			log.Printf("[CALLBACK_BACK] Ошибка отправки сообщения для удаления ReplyKeyboard. ChatID=%d: %v", chatID, errSendInvisible)
		}
		if currentState == constants.STATE_ORDER_ADDRESS_LOCATION {
			tempOrderForLocPrompt := bh.Deps.SessionManager.GetTempOrder(chatID)
			if tempOrderForLocPrompt.LocationPromptMessageID != 0 {
				bh.deleteMessageHelper(chatID, tempOrderForLocPrompt.LocationPromptMessageID)
				tempOrderForLocPrompt.LocationPromptMessageID = 0
				bh.Deps.SessionManager.UpdateTempOrder(chatID, tempOrderForLocPrompt)
			}
		}
	}

	bh.Deps.SessionManager.PopState(chatID)
	destinationWithParams := strings.TrimPrefix(data, "back_to_")
	partsOfDestination := strings.Split(destinationWithParams, "_")

	var destinationCommand string
	var destinationParams []string

	knownBackTargets := map[string]int{
		"main": 1, "category": 1, "subcategory": 1, "description": 1, "name": 1,
		"date": 1, "time": 1, "phone": 1, "address": 1, "photo": 1, "payment": 1,
		"edit_menu_direct": 1, // ORDERID будет из сессии
		"staff_menu":       2, "staff_list_menu": 3, "staff_info": 2, "staff_edit_menu": 3,
		"stats_menu": 2, "stats_basic_periods": 3, "stats_select_custom_date": 4,
		"stats_select_custom_period": 4, "block_user_menu": 3, "block_user_list_prompt": 4,
		"unblock_user_list_prompt": 4, "contact_operator": 2, "contact_phone_options": 3,
		"invite_friend": 2, "referral_my": 2, "manage_orders": 2,
		constants.CALLBACK_PREFIX_MY_SALARY:                         1,
		constants.CALLBACK_PREFIX_OWNER_STAFF_PAYOUT:                1,
		constants.CALLBACK_PREFIX_OWNER_FINANCIALS:                  1,
		constants.CALLBACK_PREFIX_OWNER_CASH_MANAGEMENT_MAIN:        1,
		constants.CALLBACK_PREFIX_DRIVER_REPORT_OVERALL_MENU:        3,
		constants.CALLBACK_PREFIX_DRIVER_REPORT_LOADERS_MENU:        4,
		constants.CALLBACK_PREFIX_DRIVER_REPORT_OTHER_EXPENSES_MENU: 4, // Добавлено
		"staff_add_prompt_name":                                     1, "staff_add_prompt_surname": 1, "staff_add_prompt_nickname": 1,
		"staff_add_prompt_phone": 1, "staff_add_prompt_chatid": 1, "staff_add_prompt_card_number": 1,
		"staff_add_role_final":                                    1,
		constants.CALLBACK_PREFIX_OWNER_CASH_ACTUAL_LIST:          1,
		constants.CALLBACK_PREFIX_OWNER_CASH_SETTLED_LIST:         1,
		constants.CALLBACK_PREFIX_OWNER_VIEW_DRIVER_SETTLEMENTS:   1,
		constants.CALLBACK_PREFIX_OPERATOR_VIEW_DRIVER_SETTLEMENT: 1,
		// Для операторского потока создания заказа
		"op_confirm_options": 2, // back_to_op_confirm_options_ORDERID
		"op_cost_input":      2, // back_to_op_cost_input_ORDERID
		"op_assign_exec":     2, // back_to_op_assign_exec_ORDERID
	}

	leafBackTargets := map[string]bool{ // Полные команды, которые не имеют доп. параметров в ключе
		"staff_add_prompt_name": true, "staff_add_prompt_surname": true, "staff_add_prompt_nickname": true,
		"staff_add_prompt_phone": true, "staff_add_prompt_chatid": true, "staff_add_prompt_card_number": true,
		"staff_add_role_final": true,
	}

	foundKnownTarget := false
	if leafBackTargets[destinationWithParams] {
		destinationCommand = destinationWithParams
		foundKnownTarget = true
	} else {
		sortedKnownKeys := make([]string, 0, len(knownBackTargets))
		for k := range knownBackTargets {
			sortedKnownKeys = append(sortedKnownKeys, k)
		}
		sort.Slice(sortedKnownKeys, func(i, j int) bool {
			numPartsI := knownBackTargets[sortedKnownKeys[i]]
			numPartsJ := knownBackTargets[sortedKnownKeys[j]]
			if numPartsI != numPartsJ {
				return numPartsI > numPartsJ
			}
			return len(sortedKnownKeys[i]) > len(sortedKnownKeys[j])
		})

		for _, prefixCandidate := range sortedKnownKeys {
			numPartsInPrefixKey := knownBackTargets[prefixCandidate]
			if len(partsOfDestination) >= numPartsInPrefixKey {
				potentialCmd := strings.Join(partsOfDestination[:numPartsInPrefixKey], "_")
				if potentialCmd == prefixCandidate {
					destinationCommand = prefixCandidate
					if len(partsOfDestination) > numPartsInPrefixKey {
						destinationParams = partsOfDestination[numPartsInPrefixKey:]
					}
					foundKnownTarget = true
					break
				}
			}
		}
	}

	if !foundKnownTarget {
		destinationCommand = partsOfDestination[0]
		if len(partsOfDestination) > 1 {
			destinationParams = partsOfDestination[1:]
		}
	}

	log.Printf("[CALLBACK_BACK] Разобранный пункт назначения: Command='%s', Params=%v. ChatID=%d", destinationCommand, destinationParams, chatID)

	var categoryForSubmenu string
	tempOrderForBack := bh.Deps.SessionManager.GetTempOrder(chatID)

	if destinationCommand == "subcategory" {
		categoryForSubmenu = tempOrderForBack.Category
		if categoryForSubmenu == "" {
			log.Printf("[CALLBACK_BACK] Предупреждение: категория не найдена в сессии для возврата к подкатегории. ChatID=%d", chatID)
			bh.SendCategoryMenu(chatID, user.FirstName, originalMessageID)
			return
		}
	}
	// Устанавливаем CurrentMessageID для редактирования перед вызовом меню
	tempOrderForBack.CurrentMessageID = originalMessageID
	bh.Deps.SessionManager.UpdateTempOrder(chatID, tempOrderForBack)

	switch destinationCommand {
	case "main":
		bh.Deps.SessionManager.ClearState(chatID)
		bh.Deps.SessionManager.ClearTempOrder(chatID)
		bh.Deps.SessionManager.ClearTempDriverSettlement(chatID)
		bh.SendMainMenu(chatID, user, originalMessageID)
	case "staff_menu":
		bh.SendStaffMenu(chatID, originalMessageID)
	case "staff_list_menu":
		bh.SendStaffListMenu(chatID, originalMessageID)
	case "staff_info":
		if len(destinationParams) == 1 {
			targetID, _ := strconv.ParseInt(destinationParams[0], 10, 64)
			bh.SendStaffInfo(chatID, targetID, originalMessageID)
		} else {
			bh.SendStaffMenu(chatID, originalMessageID)
		}
	case "staff_edit_menu":
		if len(destinationParams) == 1 {
			targetID, _ := strconv.ParseInt(destinationParams[0], 10, 64)
			bh.SendStaffEditMenu(chatID, targetID, originalMessageID)
		} else {
			bh.SendStaffMenu(chatID, originalMessageID)
		}
	case "staff_add_prompt_name":
		bh.SendStaffAddPrompt(chatID, constants.STATE_STAFF_ADD_NAME, "👤 Введите имя сотрудника:", "staff_menu", originalMessageID)
	case "staff_add_prompt_surname":
		bh.SendStaffAddPrompt(chatID, constants.STATE_STAFF_ADD_SURNAME, "👤 Введите фамилию:", "staff_add_prompt_name", originalMessageID)
	case "staff_add_prompt_nickname":
		bh.SendStaffAddPrompt(chatID, constants.STATE_STAFF_ADD_NICKNAME, "📛 Введите позывной (никнейм) сотрудника (можно пропустить, отправив '-'):", "staff_add_prompt_surname", originalMessageID)
	case "staff_add_prompt_phone":
		bh.SendStaffAddPrompt(chatID, constants.STATE_STAFF_ADD_PHONE, "📱 Введите телефон сотрудника (например, +79001234567):", "staff_add_prompt_nickname", originalMessageID)
	case "staff_add_prompt_chatid":
		bh.SendStaffAddPrompt(chatID, constants.STATE_STAFF_ADD_CHATID, "🆔 Введите Telegram ChatID сотрудника (числовой ID):", "staff_add_prompt_phone", originalMessageID)
	case "staff_add_prompt_card_number":
		bh.SendStaffAddPrompt(chatID, constants.STATE_STAFF_ADD_CARD_NUMBER, "💳 Введите номер карты сотрудника (16-19 цифр, без пробелов). Если карты нет, отправьте '-'.", "staff_add_prompt_chatid", originalMessageID)
	case "staff_add_role_final":
		bh.SendStaffAddPrompt(chatID, constants.STATE_STAFF_ADD_CARD_NUMBER, "💳 Введите номер карты сотрудника (16-19 цифр, без пробелов). Если карты нет, отправьте '-'.", "staff_add_prompt_chatid", originalMessageID)
	case "stats_menu":
		bh.SendStatsMenu(chatID, originalMessageID)
	case "stats_basic_periods":
		bh.SendBasicStatsPeriodsMenu(chatID, originalMessageID)
	case "stats_select_custom_date", "stats_select_custom_period":
		bh.SendStatsMenu(chatID, originalMessageID)
	case "block_user_menu":
		bh.SendBlockUserMenu(chatID, originalMessageID)
	case "block_user_list_prompt":
		bh.SendUserListForBlocking(chatID, originalMessageID)
	case "unblock_user_list_prompt":
		bh.SendUserListForUnblocking(chatID, originalMessageID)
	case "contact_operator":
		bh.SendContactOperatorMenu(chatID, user, originalMessageID)
	case "contact_phone_options":
		bh.SendPhoneOptionsMenu(chatID, originalMessageID)
	case "invite_friend":
		bh.SendInviteFriendMenu(chatID, originalMessageID)
	case "referral_my":
		bh.SendMyReferralsMenu(chatID, originalMessageID)
	case "manage_orders":
		bh.SendOrdersMenu(chatID, user, originalMessageID)
	case "category":
		bh.SendCategoryMenu(chatID, user.FirstName, originalMessageID)
	case "subcategory":
		bh.SendSubcategoryMenu(chatID, categoryForSubmenu, originalMessageID)
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
	case "address": // Это кейс для back_to_address
		// Если предыдущее состояние было связано с фото, очищаем ActiveMediaGroupID
		// Это нужно делать более надежно, проверяя предыдущее состояние из истории.
		// Пока что, если мы идем на "address" с кнопки "Назад", и текущее состояние (до PopState) было STATE_ORDER_PHOTO,
		// то можно сбросить.
		if currentState == constants.STATE_ORDER_PHOTO { // currentState - это состояние *до* PopState
			log.Printf("[CALLBACK_BACK] Возврат с шага фото на адрес. Сброс ActiveMediaGroupID. ChatID=%d", chatID)
			tempOrderForBack := bh.Deps.SessionManager.GetTempOrder(chatID) // Уже будет с PopState, но мы работаем с данными ДО PopState
			tempOrderForBack.ActiveMediaGroupID = ""
			bh.Deps.SessionManager.UpdateTempOrder(chatID, tempOrderForBack)
		}
		bh.SendAddressInputMenu(chatID, originalMessageID) // Эта функция обновит CurrentMessageID в сессии
		// ...
	case "photo":
		bh.SendPhotoInputMenu(chatID, originalMessageID)
	case "payment":
		bh.SendPaymentSelectionMenu(chatID, originalMessageID)
	case "edit_menu_direct":
		bh.SendEditOrderMenu(chatID, originalMessageID)
	case "confirm":
		orderIDForConfirm := tempOrderForBack.ID // Берем ID из сессии, если не передан
		if len(destinationParams) == 1 {
			id, err := strconv.ParseInt(destinationParams[0], 10, 64)
			if err == nil {
				orderIDForConfirm = id
			}
		}
		if orderIDForConfirm != 0 {
			tempOrderForBack.ID = orderIDForConfirm
			bh.Deps.SessionManager.UpdateTempOrder(chatID, tempOrderForBack)
			bh.SendOrderConfirmationMenu(chatID, originalMessageID)
		} else {
			log.Printf("[CALLBACK_BACK] Ошибка confirm: ID заказа не найден. ChatID=%d", chatID)
			bh.SendMainMenu(chatID, user, originalMessageID)
		}
	case constants.CALLBACK_PREFIX_MY_SALARY:
		bh.SendMySalaryMenu(chatID, user, originalMessageID)
	case constants.CALLBACK_PREFIX_OWNER_STAFF_PAYOUT:
		bh.SendOwnerStaffListForPayout(chatID, user, originalMessageID, 0)
	case constants.CALLBACK_PREFIX_OWNER_FINANCIALS:
		bh.SendOwnerFinancialsMainMenu(chatID, user, originalMessageID)
	case constants.CALLBACK_PREFIX_OWNER_CASH_MANAGEMENT_MAIN:
		bh.SendOwnerCashManagementMenu(chatID, user, originalMessageID)
	case constants.CALLBACK_PREFIX_OWNER_CASH_ACTUAL_LIST:
		page := 0
		if len(destinationParams) > 0 {
			page, _ = strconv.Atoi(destinationParams[0])
		}
		bh.SendOwnerActualDebtsList(chatID, user, originalMessageID, page)
	case constants.CALLBACK_PREFIX_OWNER_CASH_SETTLED_LIST:
		page := 0
		if len(destinationParams) > 0 {
			page, _ = strconv.Atoi(destinationParams[0])
		}
		bh.SendOwnerSettledPaymentsList(chatID, user, originalMessageID, page)
	case constants.CALLBACK_PREFIX_OWNER_VIEW_DRIVER_SETTLEMENTS:
		if len(destinationParams) == 3 {
			driverID, _ := strconv.ParseInt(destinationParams[0], 10, 64)
			viewType := destinationParams[1]
			page, _ := strconv.Atoi(destinationParams[2])
			bh.SendOwnerDriverIndividualSettlementsList(chatID, user, originalMessageID, driverID, viewType, page)
		} else {
			log.Printf("[CALLBACK_BACK] Ошибка %s: неверное количество параметров: %v", destinationCommand, destinationParams)
			bh.SendOwnerCashManagementMenu(chatID, user, originalMessageID)
		}
	case constants.CALLBACK_PREFIX_OPERATOR_VIEW_DRIVER_SETTLEMENT:
		// При возврате из просмотра отчета оператором, возвращаем в главное меню оператора
		bh.SendMainMenu(chatID, user, originalMessageID)

	case constants.CALLBACK_PREFIX_DRIVER_REPORT_OVERALL_MENU:
		bh.SendDriverReportOverallMenu(chatID, user, originalMessageID)
	case constants.CALLBACK_PREFIX_DRIVER_REPORT_LOADERS_MENU:
		bh.SendDriverReportLoadersSubMenu(chatID, user, originalMessageID)
	case constants.CALLBACK_PREFIX_DRIVER_REPORT_OTHER_EXPENSES_MENU: // Добавлено
		bh.SendDriverReportOtherExpensesMenu(chatID, user, originalMessageID)

	// Обработка "Назад" для операторского потока создания заказа
	case "op_confirm_options": // back_to_op_confirm_options_ORDERID
		orderIDForOpConfirm := tempOrderForBack.ID
		if len(destinationParams) == 1 {
			id, err := strconv.ParseInt(destinationParams[0], 10, 64)
			if err == nil {
				orderIDForOpConfirm = id
			}
		}
		if orderIDForOpConfirm != 0 {
			tempOrderForBack.ID = orderIDForOpConfirm
			bh.Deps.SessionManager.UpdateTempOrder(chatID, tempOrderForBack)
			bh.SendOrderConfirmationMenu(chatID, originalMessageID) // Эта функция обработает OrderAction
		} else {
			log.Printf("[CALLBACK_BACK] Ошибка op_confirm_options: ID заказа не найден. ChatID=%d", chatID)
			bh.SendMainMenu(chatID, user, originalMessageID)
		}
	case "op_cost_input": // back_to_op_cost_input_ORDERID
		orderIDForOpCost := tempOrderForBack.ID
		if len(destinationParams) == 1 {
			id, err := strconv.ParseInt(destinationParams[0], 10, 64)
			if err == nil {
				orderIDForOpCost = id
			}
		}
		if orderIDForOpCost != 0 {
			bh.SendOpOrderCostInputMenu(chatID, orderIDForOpCost, originalMessageID)
		} else {
			log.Printf("[CALLBACK_BACK] Ошибка op_cost_input: ID заказа не найден. ChatID=%d", chatID)
			bh.SendMainMenu(chatID, user, originalMessageID)
		}
	case "op_assign_exec": // back_to_op_assign_exec_ORDERID
		orderIDForOpAssign := tempOrderForBack.ID
		if len(destinationParams) == 1 {
			id, err := strconv.ParseInt(destinationParams[0], 10, 64)
			if err == nil {
				orderIDForOpAssign = id
			}
		}
		if orderIDForOpAssign != 0 {
			bh.SendOpAssignExecutorsMenu(chatID, orderIDForOpAssign, originalMessageID)
		} else {
			log.Printf("[CALLBACK_BACK] Ошибка op_assign_exec: ID заказа не найден. ChatID=%d", chatID)
			bh.SendMainMenu(chatID, user, originalMessageID)
		}

	default:
		log.Printf("[CALLBACK_BACK] ОШИБКА: Неизвестное место назначения для возврата: '%s' (data: '%s'). Возврат в Главное меню. ChatID=%d", destinationCommand, data, chatID)
		bh.SendMainMenu(chatID, user, originalMessageID)
	}
	log.Printf("[CALLBACK_BACK] Завершение обработки 'Назад'. ChatID=%d, Data='%s'", chatID, data)
}

// SendSimpleMessage - это публичный метод для отправки простого текстового сообщения.
// Его можно вызывать из других пакетов, например, из пакета api.
func (bh *BotHandler) SendSimpleMessage(chatID int64, text string) {
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = tgbotapi.ModeMarkdown // Используем Markdown по умолчанию для форматирования
	if _, err := bh.Deps.BotClient.Send(msg); err != nil {
		log.Printf("[BotHandler.SendSimpleMessage] Ошибка отправки сообщения в чат %d: %v", chatID, err)
	}
}
