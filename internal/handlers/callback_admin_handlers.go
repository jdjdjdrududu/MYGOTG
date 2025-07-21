package handlers

import (
	"Original/internal/constants"
	"Original/internal/db"
	"Original/internal/models"
	"Original/internal/utils"
	"database/sql"
	"fmt"
	"log"
	"regexp"
	"strconv"
	"strings"
	"time"

	tgbotapi "github.com/OvyFlash/telegram-bot-api"
)

// dispatchAdminCallbacks маршрутизирует коллбэки, связанные с административными функциями,
// а также функциями сотрудников, такими как "Моя зарплата" и новый флоу "Расчет по заказам" водителя.
func (bh *BotHandler) dispatchAdminCallbacks(currentCommand string, parts []string, data string, chatID int64, user models.User, originalMessageID int) int {
	log.Printf("[CALLBACK_ADMIN] Диспетчер: Команда='%s', Части=%v, ChatID=%d, UserRole=%s, OriginalMsgID=%d", currentCommand, parts, chatID, user.Role, originalMessageID)

	var newMenuMessageID int = originalMessageID
	var sentMsg tgbotapi.Message
	var errHelper error

	isMainOperatorOrHigher := utils.IsRoleOrHigher(user.Role, constants.ROLE_MAINOPERATOR)
	isOwner := user.Role == constants.ROLE_OWNER
	isOperatorOrHigher := utils.IsOperatorOrHigher(user.Role)
	isEmployee := user.Role == constants.ROLE_DRIVER || user.Role == constants.ROLE_LOADER || isOperatorOrHigher
	isDriver := user.Role == constants.ROLE_DRIVER

	// Категории команд для проверки доступа
	staffManagementCommands := []string{
		"staff_menu", "staff_list_menu", "staff_list_by_role", "staff_add_prompt_name",
		"staff_add_prompt_surname", "staff_add_prompt_nickname", "staff_add_prompt_phone",
		"staff_add_prompt_chatid", "staff_add_prompt_card_number",
		"staff_add_role_final", "staff_info", "staff_edit_menu",
		"staff_edit_role_final", "staff_block_reason_prompt", "staff_unblock_confirm", "staff_delete_confirm",
		"staff_edit_field_name", "staff_edit_field_surname", "staff_edit_field_nickname",
		"staff_edit_field_phone", "staff_edit_field_card_number", "staff_edit_field_role",
	}
	statsAndExcelCommands := []string{
		"stats_menu", "stats_basic_periods", "stats_get_today", "stats_get_yesterday",
		"stats_get_current_week", "stats_get_current_month", "stats_get_last_week", "stats_get_last_month",
		"stats_select_custom_date", "stats_select_custom_period", "stats_select_month", "stats_select_day",
		"stats_year_nav", "send_excel_menu", "excel_generate_orders", "excel_generate_referrals",
		"excel_generate_salaries",
	}
	userBlockingCommands := []string{
		"block_user_menu", "block_user_list_prompt", "block_user_info", "block_user_reason_prompt",
		"block_user_final", "unblock_user_list_prompt", "unblock_user_info", "unblock_user_final",
	}
	salaryCommands := []string{
		constants.CALLBACK_PREFIX_MY_SALARY,
	}

	driverInlineReportCommands := []string{
		constants.CALLBACK_PREFIX_DRIVER_REPORT_OVERALL_MENU,
		constants.CALLBACK_PREFIX_DRIVER_REPORT_SET_FUEL,
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

	ownerPayoutCommands := []string{
		constants.CALLBACK_PREFIX_OWNER_STAFF_PAYOUT,
	}
	ownerFinancialsCommands := []string{
		"financial_menu",
		constants.CALLBACK_PREFIX_OWNER_FINANCIALS,
	}
	ownerCashManagementCommands := []string{
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
	}

	// --- НАЧАЛО ИЗМЕНЕНИЯ: Добавляем новые коллбэки в проверку прав ---
	settlementReviewCommands := []string{
		constants.CALLBACK_PREFIX_OPERATOR_APPROVE_SETTLEMENT,
		constants.CALLBACK_PREFIX_OPERATOR_REJECT_SETTLEMENT,
	}
	// --- КОНЕЦ ИЗМЕНЕНИЯ ---

	accessGranted := true
	if utils.IsCommandInCategory(currentCommand, staffManagementCommands) && !isMainOperatorOrHigher {
		accessGranted = false
	} else if utils.IsCommandInCategory(currentCommand, statsAndExcelCommands) && !isMainOperatorOrHigher {
		accessGranted = false
	} else if utils.IsCommandInCategory(currentCommand, userBlockingCommands) && !isOperatorOrHigher {
		accessGranted = false
	} else if (utils.IsCommandInCategory(currentCommand, salaryCommands) || strings.HasPrefix(currentCommand, constants.CALLBACK_PREFIX_MY_SALARY+"_")) && !isEmployee {
		accessGranted = false
	} else if (utils.IsCommandInCategory(currentCommand, driverInlineReportCommands)) && !isDriver {
		accessGranted = false
	} else if (utils.IsCommandInCategory(currentCommand, ownerPayoutCommands) || strings.HasPrefix(currentCommand, constants.CALLBACK_PREFIX_OWNER_STAFF_PAYOUT+"_")) && !isOwner {
		accessGranted = false
	} else if (utils.IsCommandInCategory(currentCommand, ownerFinancialsCommands) || strings.HasPrefix(currentCommand, constants.CALLBACK_PREFIX_OWNER_FINANCIALS+"_")) && !isMainOperatorOrHigher {
		accessGranted = false
	} else if utils.IsCommandInCategory(currentCommand, ownerCashManagementCommands) {
		if currentCommand == constants.CALLBACK_PREFIX_OPERATOR_VIEW_DRIVER_SETTLEMENT && !isOperatorOrHigher {
			accessGranted = false
		} else if currentCommand != constants.CALLBACK_PREFIX_OPERATOR_VIEW_DRIVER_SETTLEMENT && !isOwner {
			accessGranted = false
		}
	} else if utils.IsCommandInCategory(currentCommand, settlementReviewCommands) && !isOperatorOrHigher { // --- НОВОЕ ПРАВИЛО ---
		accessGranted = false
	}

	if !accessGranted {
		log.Printf("[CALLBACK_ADMIN] Отказ в доступе для команды '%s', UserRole=%s", currentCommand, user.Role)
		sentMsg, _ = bh.sendAccessDenied(chatID, originalMessageID)
		if sentMsg.MessageID != 0 && sentMsg.MessageID != originalMessageID {
			newMenuMessageID = sentMsg.MessageID
		}
		return newMenuMessageID
	}

	foundSpecificHandlerInSwitch := true

	switch currentCommand {
	case constants.CALLBACK_PREFIX_OPERATOR_APPROVE_SETTLEMENT: // parts: [SETTLEMENT_ID]
		if len(parts) == 1 {
			settlementID, err := strconv.ParseInt(parts[0], 10, 64)
			if err == nil {
				bh.handleOperatorApproveSettlement(chatID, user, settlementID, originalMessageID)
			} else {
				bh.sendErrorMessageHelper(chatID, originalMessageID, "Ошибка: неверный ID отчета.")
			}
		} else {
			bh.sendErrorMessageHelper(chatID, originalMessageID, "Ошибка: неверный формат команды утверждения.")
		}
		break // Добавляем break, чтобы избежать провала в default
	case constants.CALLBACK_PREFIX_OPERATOR_REJECT_SETTLEMENT: // parts: [SETTLEMENT_ID]
		if len(parts) == 1 {
			settlementID, err := strconv.ParseInt(parts[0], 10, 64)
			if err == nil {
				bh.handleOperatorRejectSettlement(chatID, user, settlementID, originalMessageID)
			} else {
				bh.sendErrorMessageHelper(chatID, originalMessageID, "Ошибка: неверный ID отчета.")
			}
		} else {
			bh.sendErrorMessageHelper(chatID, originalMessageID, "Ошибка: неверный формат команды отклонения.")
		}
		break // Добавляем break

	case constants.CALLBACK_PREFIX_OPERATOR_VIEW_DRIVER_SETTLEMENT:
		if !utils.IsOperatorOrHigher(user.Role) {
			sentMsg, _ = bh.sendAccessDenied(chatID, originalMessageID)
			if sentMsg.MessageID != 0 {
				newMenuMessageID = sentMsg.MessageID
			}
			return newMenuMessageID
		}
		if len(parts) == 1 {
			settlementID, err := strconv.ParseInt(parts[0], 10, 64)
			if err == nil {
				bh.SendOperatorViewDriverSettlementDetails(chatID, user, settlementID, originalMessageID)
			} else {
				log.Printf("CALLBACK_ADMIN: Неверный ID отчета для просмотра оператором: %s", parts[0])
				sentMsg, errHelper = bh.sendErrorMessageHelper(chatID, originalMessageID, "❌ Ошибка: неверный ID отчета.")
			}
		} else {
			log.Printf("CALLBACK_ADMIN: Некорректный формат для просмотра отчета водителем: %v", parts)
			sentMsg, errHelper = bh.sendErrorMessageHelper(chatID, originalMessageID, "❌ Ошибка: неверный формат команды.")
		}
		break // Добавляем break
	// --- Штат ---
	case "staff_menu":
		bh.SendStaffMenu(chatID, originalMessageID)
	case "staff_list_menu":
		bh.SendStaffListMenu(chatID, originalMessageID)
	case "staff_list_by_role": // parts: [ROLE_KEY]
		if len(parts) == 1 {
			bh.SendStaffList(chatID, parts[0], originalMessageID)
		} else {
			sentMsg, errHelper = bh.sendErrorMessageHelper(chatID, originalMessageID, "Ошибка: не указана роль.")
		}
	case "staff_add_prompt_name":
		bh.Deps.SessionManager.ClearTempOrder(chatID)
		bh.SendStaffAddPrompt(chatID, constants.STATE_STAFF_ADD_NAME, "👤 Введите имя сотрудника:", "staff_menu", originalMessageID)
	case "staff_add_prompt_card_number":
		bh.SendStaffAddPrompt(chatID, constants.STATE_STAFF_ADD_CARD_NUMBER, "💳 Введите номер карты сотрудника (16-19 цифр, без пробелов). Если карты нет, отправьте '-'.", "staff_add_prompt_chatid", originalMessageID)
	case "staff_add_role_final": // parts: [ROLE_KEY]
		if len(parts) == 1 {
			bh.handleStaffAddRoleFinal(chatID, parts[0], originalMessageID)
		} else {
			sentMsg, errHelper = bh.sendErrorMessageHelper(chatID, originalMessageID, "Ошибка: не выбрана роль.")
		}
	case "staff_info": // parts: [TARGET_CHAT_ID]
		if len(parts) == 1 {
			targetChatID, _ := strconv.ParseInt(parts[0], 10, 64)
			bh.SendStaffInfo(chatID, targetChatID, originalMessageID)
		}
	case "staff_edit_menu": // parts: [TARGET_CHAT_ID]
		if len(parts) == 1 {
			targetChatID, _ := strconv.ParseInt(parts[0], 10, 64)
			bh.SendStaffEditMenu(chatID, targetChatID, originalMessageID)
		}
	case "staff_edit_field_name", "staff_edit_field_surname", "staff_edit_field_nickname", "staff_edit_field_phone", "staff_edit_field_card_number", "staff_edit_field_role": // parts: [TARGET_CHAT_ID]
		if len(parts) == 1 {
			targetChatID, _ := strconv.ParseInt(parts[0], 10, 64)
			fieldToEdit := strings.TrimPrefix(currentCommand, "staff_edit_field_")
			var promptText string
			backCallback := fmt.Sprintf("staff_edit_menu_%d", targetChatID)
			shouldSendPrompt := true
			switch fieldToEdit {
			case "name":
				promptText = "✏️ Введите новое имя:"
			case "surname":
				promptText = "✏️ Введите новую фамилию:"
			case "nickname":
				promptText = "✏️ Введите новый позывной (или '-' чтобы убрать):"
			case "phone":
				promptText = "✏️ Введите новый телефон (или '-' чтобы убрать):"
			case "card_number":
				promptText = "💳 Введите новый номер карты (16-19 цифр, или '-' чтобы убрать):"
			case "role":
				bh.SendStaffRoleSelectionMenu(chatID, fmt.Sprintf("staff_edit_role_final_%d", targetChatID), originalMessageID, backCallback)
				shouldSendPrompt = false
			default:
				sentMsg, errHelper = bh.sendErrorMessageHelper(chatID, originalMessageID, "Неизвестное поле для редактирования.")
				shouldSendPrompt = false
			}
			if shouldSendPrompt {
				bh.SendStaffEditFieldPrompt(chatID, targetChatID, fieldToEdit, promptText, originalMessageID)
			}
		}
	case "staff_edit_role_final": // parts: [TARGET_CHAT_ID, NEW_ROLE_KEY]
		if len(parts) == 2 {
			targetChatID, _ := strconv.ParseInt(parts[0], 10, 64)
			bh.handleStaffEditRoleFinal(chatID, targetChatID, parts[1], originalMessageID)
		}
	case "staff_block_reason_prompt": // parts: [TARGET_CHAT_ID]
		if len(parts) == 1 {
			targetChatID, _ := strconv.ParseInt(parts[0], 10, 64)
			bh.SendStaffBlockReasonInput(chatID, targetChatID, originalMessageID)
		}
	case "staff_unblock_confirm": // parts: [TARGET_CHAT_ID]
		if len(parts) == 1 {
			targetChatID, _ := strconv.ParseInt(parts[0], 10, 64)
			bh.handleStaffUnblockConfirm(chatID, targetChatID, originalMessageID)
		}
	case "staff_delete_confirm": // parts: [TARGET_CHAT_ID]
		if len(parts) == 1 {
			targetChatID, _ := strconv.ParseInt(parts[0], 10, 64)
			bh.handleStaffDeleteConfirm(chatID, targetChatID, originalMessageID)
		}

	// --- Статистика и Excel ---
	case "stats_menu":
		bh.SendStatsMenu(chatID, originalMessageID)
	case "stats_basic_periods":
		bh.SendBasicStatsPeriodsMenu(chatID, originalMessageID)
	case "stats_get_today", "stats_get_yesterday", "stats_get_current_week", "stats_get_current_month", "stats_get_last_week", "stats_get_last_month":
		periodKey := strings.TrimPrefix(currentCommand, "stats_get_")
		bh.handleStatsGetPeriod(periodKey, data, chatID, originalMessageID)
	case "stats_select_custom_date":
		bh.SendMonthSelectionMenu(chatID, originalMessageID, time.Now().Year(), "custom_date")
	case "stats_select_custom_period":
		bh.Deps.SessionManager.ClearTempOrder(chatID)
		bh.SendMonthSelectionMenu(chatID, originalMessageID, time.Now().Year(), "period_start")
	case "stats_select_month": // parts: [CONTEXT, YEAR, MONTH_INT]
		if len(parts) == 3 {
			context := parts[0]
			year, _ := strconv.Atoi(parts[1])
			monthInt, _ := strconv.Atoi(parts[2])
			bh.SendDaySelectionMenu(chatID, originalMessageID, year, time.Month(monthInt), context)
		}
	case "stats_select_day": // parts: [CONTEXT, YEAR, MONTH_INT, DAY]
		if len(parts) == 4 {
			context := parts[0]
			year, _ := strconv.Atoi(parts[1])
			monthInt, _ := strconv.Atoi(parts[2])
			day, _ := strconv.Atoi(parts[3])
			bh.handleStatsSelectDay(chatID, user, context, year, time.Month(monthInt), day, originalMessageID)
		}
	case "stats_year_nav": // parts: [CONTEXT, YEAR_STR]
		if len(parts) == 2 {
			bh.handleStatsYearNavigation(parts[0], parts[1], data, chatID, originalMessageID)
		}
	case "send_excel_menu":
		bh.SendExcelMenu(chatID, originalMessageID)
	case "excel_generate_orders", "excel_generate_referrals", "excel_generate_salaries":
		reportType := strings.TrimPrefix(currentCommand, "excel_generate_")
		bh.handleExcelGenerate(chatID, user, reportType, originalMessageID)
		foundSpecificHandlerInSwitch = false

	// --- Блокировка пользователей ---
	case "block_user_menu":
		bh.SendBlockUserMenu(chatID, originalMessageID)
	case "block_user_list_prompt":
		bh.SendUserListForBlocking(chatID, originalMessageID)
	case "block_user_info": // parts: [TARGET_CHAT_ID]
		if len(parts) == 1 {
			targetChatID, _ := strconv.ParseInt(parts[0], 10, 64)
			bh.SendBlockUserInfo(chatID, targetChatID, originalMessageID)
		}
	case "block_user_reason_prompt": // parts: [TARGET_CHAT_ID]
		if len(parts) == 1 {
			targetChatID, _ := strconv.ParseInt(parts[0], 10, 64)
			bh.SendBlockReasonInput(chatID, targetChatID, originalMessageID)
		}
	case "block_user_final": // parts: [TARGET_CHAT_ID]
		if len(parts) == 1 {
			targetChatID, _ := strconv.ParseInt(parts[0], 10, 64)
			bh.handleBlockUserFinal(chatID, user, targetChatID, originalMessageID)
		}
	case "unblock_user_list_prompt":
		bh.SendUserListForUnblocking(chatID, originalMessageID)
	case "unblock_user_info": // parts: [TARGET_CHAT_ID]
		if len(parts) == 1 {
			targetChatID, _ := strconv.ParseInt(parts[0], 10, 64)
			bh.SendUnblockUserInfo(chatID, targetChatID, originalMessageID)
		}
	case "unblock_user_final": // parts: [TARGET_CHAT_ID]
		if len(parts) == 1 {
			targetChatID, _ := strconv.ParseInt(parts[0], 10, 64)
			bh.handleUnblockUserFinal(chatID, user, targetChatID, originalMessageID)
		}

	// --- Зарплата сотрудника ---
	case constants.CALLBACK_PREFIX_MY_SALARY:
		bh.SendMySalaryMenu(chatID, user, originalMessageID)
	case fmt.Sprintf("%s_owed", constants.CALLBACK_PREFIX_MY_SALARY):
		bh.HandleShowAmountOwed(chatID, user, originalMessageID)
	case fmt.Sprintf("%s_earned_stats", constants.CALLBACK_PREFIX_MY_SALARY):
		bh.HandleShowEarnedStats(chatID, user, originalMessageID)

	// --- ИНЛАЙН-ОТЧЕТ ВОДИТЕЛЯ ---
	case constants.CALLBACK_PREFIX_DRIVER_REPORT_OVERALL_MENU:
		bh.SendDriverReportOverallMenu(chatID, user, originalMessageID)
	case constants.CALLBACK_PREFIX_DRIVER_REPORT_SET_FUEL:
		bh.SendDriverReportFuelInputPrompt(chatID, user, originalMessageID)
	// case constants.CALLBACK_PREFIX_DRIVER_REPORT_SET_OTHER: // Заменен на MENU
	// 	bh.SendDriverReportOtherExpenseInputPrompt(chatID, user, originalMessageID)
	case constants.CALLBACK_PREFIX_DRIVER_REPORT_OTHER_EXPENSES_MENU: // НОВЫЙ
		bh.SendDriverReportOtherExpensesMenu(chatID, user, originalMessageID)
	case constants.CALLBACK_PREFIX_DRIVER_REPORT_ADD_OTHER_EXPENSE_DESCRIPTION_PROMPT: // НОВЫЙ
		bh.SendDriverReportOtherExpenseDescriptionPrompt(chatID, user, originalMessageID, false, -1)
	case constants.CALLBACK_PREFIX_DRIVER_REPORT_EDIT_OTHER_EXPENSE_PROMPT: // НОВЫЙ, parts: [INDEX]
		if len(parts) == 1 {
			expenseIndex, err := strconv.Atoi(parts[0])
			if err == nil {
				bh.SendDriverReportOtherExpenseDescriptionPrompt(chatID, user, originalMessageID, true, expenseIndex)
			} else {
				sentMsg, errHelper = bh.sendErrorMessageHelper(chatID, originalMessageID, "Ошибка: неверный индекс расхода.")
				bh.SendDriverReportOtherExpensesMenu(chatID, user, originalMessageID)
			}
		} else {
			sentMsg, errHelper = bh.sendErrorMessageHelper(chatID, originalMessageID, "Ошибка: не указан расход для редактирования.")
			bh.SendDriverReportOtherExpensesMenu(chatID, user, originalMessageID)
		}
	case constants.CALLBACK_PREFIX_DRIVER_REPORT_DELETE_OTHER_EXPENSE_SHOW_CONFIRM: // НОВЫЙ, parts: [INDEX]
		if len(parts) == 1 {
			expenseIndex, err := strconv.Atoi(parts[0])
			if err == nil {
				bh.SendDriverReportConfirmDeleteOtherExpensePrompt(chatID, user, originalMessageID, expenseIndex)
			} else {
				sentMsg, errHelper = bh.sendErrorMessageHelper(chatID, originalMessageID, "Ошибка: неверный индекс для удаления.")
				bh.SendDriverReportOtherExpensesMenu(chatID, user, originalMessageID)
			}
		} else {
			sentMsg, errHelper = bh.sendErrorMessageHelper(chatID, originalMessageID, "Ошибка: не указан расход для удаления.")
			bh.SendDriverReportOtherExpensesMenu(chatID, user, originalMessageID)
		}
	case constants.CALLBACK_PREFIX_DRIVER_REPORT_DELETE_OTHER_EXPENSE_CONFIRM: // НОВЫЙ, parts: [INDEX]
		if len(parts) == 1 {
			expenseIndex, err := strconv.Atoi(parts[0])
			if err == nil {
				tempData := bh.Deps.SessionManager.GetTempDriverSettlement(chatID)
				if expenseIndex >= 0 && expenseIndex < len(tempData.OtherExpenses) {
					deletedDesc := tempData.OtherExpenses[expenseIndex].Description
					tempData.OtherExpenses = append(tempData.OtherExpenses[:expenseIndex], tempData.OtherExpenses[expenseIndex+1:]...)
					tempData.EditingOtherExpenseIndex = -1
					bh.Deps.SessionManager.UpdateTempDriverSettlement(chatID, tempData)
					bh.sendInfoMessage(chatID, originalMessageID, fmt.Sprintf("Расход '%s' удален.", deletedDesc), constants.CALLBACK_PREFIX_DRIVER_REPORT_OTHER_EXPENSES_MENU)
					bh.SendDriverReportOtherExpensesMenu(chatID, user, originalMessageID)
				} else {
					sentMsg, errHelper = bh.sendErrorMessageHelper(chatID, originalMessageID, "Ошибка: неверный индекс расхода для удаления.")
					bh.SendDriverReportOtherExpensesMenu(chatID, user, originalMessageID)
				}
			} else {
				sentMsg, errHelper = bh.sendErrorMessageHelper(chatID, originalMessageID, "Ошибка: неверный формат индекса для удаления.")
				bh.SendDriverReportOtherExpensesMenu(chatID, user, originalMessageID)
			}
		} else {
			sentMsg, errHelper = bh.sendErrorMessageHelper(chatID, originalMessageID, "Ошибка: не указан расход для удаления (подтверждение).")
			bh.SendDriverReportOtherExpensesMenu(chatID, user, originalMessageID)
		}
	case constants.CALLBACK_PREFIX_DRIVER_REPORT_LOADERS_MENU:
		bh.SendDriverReportLoadersSubMenu(chatID, user, originalMessageID)
	case constants.CALLBACK_PREFIX_DRIVER_REPORT_ADD_LOADER_PROMPT:
		bh.SendDriverReportLoaderNameInputPrompt(chatID, user, originalMessageID)
	case constants.CALLBACK_PREFIX_DRIVER_REPORT_EDIT_LOADER_PROMPT:
		if len(parts) == 1 {
			loaderIndex, err := strconv.Atoi(parts[0])
			if err == nil {
				tempData := bh.Deps.SessionManager.GetTempDriverSettlement(chatID)
				if loaderIndex >= 0 && loaderIndex < len(tempData.LoaderPayments) {
					loaderToEdit := tempData.LoaderPayments[loaderIndex]
					bh.SendDriverReportLoaderSalaryInputPrompt(chatID, user, originalMessageID, loaderToEdit.LoaderIdentifier, true, loaderIndex)
				} else {
					sentMsg, errHelper = bh.sendErrorMessageHelper(chatID, originalMessageID, "Ошибка: неверный индекс грузчика.")
					bh.SendDriverReportLoadersSubMenu(chatID, user, originalMessageID)
				}
			} else {
				sentMsg, errHelper = bh.sendErrorMessageHelper(chatID, originalMessageID, "Ошибка: неверный формат индекса.")
			}
		} else {
			sentMsg, errHelper = bh.sendErrorMessageHelper(chatID, originalMessageID, "Ошибка: не указан грузчик для редактирования.")
		}
	case constants.CALLBACK_PREFIX_DRIVER_REPORT_DELETE_LOADER_CONFIRM:
		if len(parts) == 1 {
			loaderIndex, err := strconv.Atoi(parts[0])
			if err == nil {
				tempData := bh.Deps.SessionManager.GetTempDriverSettlement(chatID)
				if tempData.EditingLoaderIndex == loaderIndex && tempData.EditingLoaderIndex != -1 {
					if loaderIndex >= 0 && loaderIndex < len(tempData.LoaderPayments) {
						tempData.LoaderPayments = append(tempData.LoaderPayments[:loaderIndex], tempData.LoaderPayments[loaderIndex+1:]...)
						tempData.EditingLoaderIndex = -1
						bh.Deps.SessionManager.UpdateTempDriverSettlement(chatID, tempData)
						bh.SendDriverReportLoadersSubMenu(chatID, user, originalMessageID)
					} else {
						sentMsg, errHelper = bh.sendErrorMessageHelper(chatID, originalMessageID, "Ошибка: неверный индекс грузчика для удаления.")
						bh.SendDriverReportLoadersSubMenu(chatID, user, originalMessageID)
					}
				} else {
					bh.SendDriverReportConfirmDeleteLoaderPrompt(chatID, user, originalMessageID, loaderIndex)
				}
			} else {
				sentMsg, errHelper = bh.sendErrorMessageHelper(chatID, originalMessageID, "Ошибка: неверный формат индекса для удаления.")
			}
		} else {
			sentMsg, errHelper = bh.sendErrorMessageHelper(chatID, originalMessageID, "Ошибка: не указан грузчик для удаления.")
		}
	case constants.CALLBACK_PREFIX_DRIVER_REPORT_SAVE_FINAL:
		tempData := bh.Deps.SessionManager.GetTempDriverSettlement(chatID)
		tempData.RecalculateTotals(bh.Deps.Config.DriverSharePercentage)

		settlement := models.DriverSettlement{
			DriverUserID:           user.ID,
			SettlementTimestamp:    tempData.SettlementCreateTime,
			CoveredOrdersRevenue:   tempData.CoveredOrdersRevenue,
			FuelExpense:            tempData.FuelExpense,
			OtherExpenses:          tempData.OtherExpenses, // Используем новый список
			LoaderPayments:         tempData.LoaderPayments,
			DriverCalculatedSalary: tempData.DriverCalculatedSalary,
			AmountToCashier:        tempData.AmountToCashier,
			CoveredOrdersCount:     tempData.CoveredOrdersCount,
			CoveredOrderIDs:        tempData.CoveredOrderIDs,
			CreatedAt:              time.Now(),
			UpdatedAt:              time.Now(),
		}
		savedSettlementID, err := db.AddDriverSettlement(settlement)
		if err != nil {
			log.Printf("CALLBACK_PREFIX_DRIVER_REPORT_SAVE_FINAL: ошибка сохранения отчета для водителя %d: %v", user.ID, err)
			sentMsg, errHelper = bh.sendErrorMessageHelper(chatID, originalMessageID, "Произошла ошибка при сохранении отчета. Попробуйте снова.")
			bh.SendDriverReportOverallMenu(chatID, user, originalMessageID)
		} else {
			go bh.NotifyOperatorsAboutDriverSettlement(user, savedSettlementID)
			sentMsg, errHelper = bh.sendInfoMessage(chatID, originalMessageID, "✅ Отчет по расходам успешно сохранен!", "back_to_main")
			bh.Deps.SessionManager.ClearState(chatID)
			bh.Deps.SessionManager.ClearTempDriverSettlement(chatID)
		}
	case constants.CALLBACK_PREFIX_DRIVER_REPORT_CANCEL_ALL:
		bh.Deps.SessionManager.ClearState(chatID)
		bh.Deps.SessionManager.ClearTempDriverSettlement(chatID)
		bh.SendMainMenu(chatID, user, originalMessageID)

	// --- Финансы владельца (Старая версия по датам) ---
	case "financial_menu", constants.CALLBACK_PREFIX_OWNER_FINANCIALS:
		bh.SendOwnerFinancialsMainMenu(chatID, user, originalMessageID)
	case fmt.Sprintf("%s_date", constants.CALLBACK_PREFIX_OWNER_FINANCIALS): // parts: [DATE_STR_YYYY-MM-DD]
		if len(parts) == 1 {
			targetDate, _ := time.ParseInLocation("2006-01-02", parts[0], time.Local)
			bh.SendOwnerFinancialsForDate(chatID, user, targetDate, originalMessageID)
		}
	case fmt.Sprintf("%s_view", constants.CALLBACK_PREFIX_OWNER_FINANCIALS): // parts: [DRIVER_UID, REPORT_DATE_STR]
		if len(parts) == 2 {
			driverUID, _ := strconv.ParseInt(parts[0], 10, 64)
			bh.handleOwnerViewDriverSettlementsForDate(chatID, user, driverUID, parts[1], originalMessageID)
		}
	case fmt.Sprintf("%s_edit_settlement", constants.CALLBACK_PREFIX_OWNER_FINANCIALS): // parts: [SETTLEMENT_ID]
		if len(parts) == 1 {
			settleID, _ := strconv.ParseInt(parts[0], 10, 64)
			bh.handleOwnerEditSettlementStart(chatID, user, settleID, "unknown", 0, originalMessageID)
		}
	case fmt.Sprintf("%s_edit_field", constants.CALLBACK_PREFIX_OWNER_FINANCIALS): // parts: [SETTLEMENT_ID, FIELD_KEY]
		if len(parts) == 2 {
			settleID, _ := strconv.ParseInt(parts[0], 10, 64)
			bh.handleOwnerEditSettlementFieldPrompt(chatID, user, settleID, parts[1], originalMessageID)
		}
	case fmt.Sprintf("%s_save_edited_settlement", constants.CALLBACK_PREFIX_OWNER_FINANCIALS): // parts: [SETTLEMENT_ID]
		if len(parts) == 1 {
			settleID, _ := strconv.ParseInt(parts[0], 10, 64)
			bh.handleOwnerSaveAllSettlementChanges(chatID, user, settleID, originalMessageID)
		}

	// --- НОВОЕ: Управление кассой Владельца ---
	case constants.CALLBACK_PREFIX_OWNER_CASH_MANAGEMENT_MAIN:
		bh.SendOwnerCashManagementMenu(chatID, user, originalMessageID)
	case constants.CALLBACK_PREFIX_OWNER_CASH_ACTUAL_LIST: // parts: [PAGE]
		page := 0
		if len(parts) == 1 {
			page, _ = strconv.Atoi(parts[0])
		}
		bh.SendOwnerActualDebtsList(chatID, user, originalMessageID, page)
	case constants.CALLBACK_PREFIX_OWNER_CASH_SETTLED_LIST: // parts: [PAGE]
		page := 0
		if len(parts) == 1 {
			page, _ = strconv.Atoi(parts[0])
		}
		bh.SendOwnerSettledPaymentsList(chatID, user, originalMessageID, page)
	case constants.CALLBACK_PREFIX_OWNER_CASH_MARK_PAID: // parts: [SETTLEMENT_ID]
		if len(parts) == 1 {
			settlementID, err := strconv.ParseInt(parts[0], 10, 64)
			if err == nil {
				errDb := db.MarkSettlementAsPaidToOwner(settlementID)
				if errDb != nil {
					log.Printf("CALLBACK_ADMIN: Ошибка пометки отчета #%d как оплаченного: %v", settlementID, errDb)
					bh.sendErrorMessageHelper(chatID, originalMessageID, "❌ Ошибка при отметке оплаты.")
				} else {
					tempData := bh.Deps.SessionManager.GetTempDriverSettlement(chatID)
					bh.SendOwnerDriverIndividualSettlementsList(chatID, user, originalMessageID, tempData.DriverUserIDForBackNav, tempData.ViewTypeForBackNav, tempData.PageForBackNav)
				}
			} else {
				log.Printf("CALLBACK_ADMIN: Неверный ID отчета для пометки как оплаченный: %s", parts[0])
				bh.sendErrorMessageHelper(chatID, originalMessageID, "❌ Ошибка: неверный ID отчета.")
			}
		}
	case constants.CALLBACK_PREFIX_OWNER_CASH_MARK_UNPAID: // parts: [SETTLEMENT_ID]
		if len(parts) == 1 {
			settlementID, err := strconv.ParseInt(parts[0], 10, 64)
			if err == nil {
				errDb := db.MarkSettlementAsUnpaidToOwner(settlementID)
				if errDb != nil {
					log.Printf("CALLBACK_ADMIN: Ошибка пометки отчета #%d как НЕ оплаченного: %v", settlementID, errDb)
					bh.sendErrorMessageHelper(chatID, originalMessageID, "❌ Ошибка при отмене отметки оплаты.")
				} else {
					tempData := bh.Deps.SessionManager.GetTempDriverSettlement(chatID)
					bh.SendOwnerDriverIndividualSettlementsList(chatID, user, originalMessageID, tempData.DriverUserIDForBackNav, tempData.ViewTypeForBackNav, tempData.PageForBackNav)
				}
			} else {
				log.Printf("CALLBACK_ADMIN: Неверный ID отчета для пометки как НЕ оплаченный: %s", parts[0])
				bh.sendErrorMessageHelper(chatID, originalMessageID, "❌ Ошибка: неверный ID отчета.")
			}
		}
	case constants.CALLBACK_PREFIX_OWNER_VIEW_DRIVER_SETTLEMENTS: // parts: [DRIVER_USER_ID, VIEW_TYPE, PAGE]
		if len(parts) == 3 {
			driverUserID, errDriverID := strconv.ParseInt(parts[0], 10, 64)
			viewType := parts[1]
			page, errPage := strconv.Atoi(parts[2])
			if errDriverID == nil && errPage == nil && (viewType == constants.VIEW_TYPE_ACTUAL_SETTLEMENTS || viewType == constants.VIEW_TYPE_SETTLED_SETTLEMENTS) {
				bh.SendOwnerDriverIndividualSettlementsList(chatID, user, originalMessageID, driverUserID, viewType, page)
			} else {
				log.Printf("CALLBACK_ADMIN: Неверный формат для просмотра отчетов водителя: %v", parts)
				bh.sendErrorMessageHelper(chatID, originalMessageID, "❌ Ошибка: неверные параметры для просмотра отчетов водителя.")
			}
		} else {
			log.Printf("CALLBACK_ADMIN: Некорректный формат для просмотра отчетов водителя: %v", parts)
			bh.sendErrorMessageHelper(chatID, originalMessageID, "❌ Ошибка: неверный формат команды.")
		}
	case constants.CALLBACK_PREFIX_OWNER_EDIT_SETTLEMENT:
		if len(parts) >= 1 {
			settlementID, err := strconv.ParseInt(parts[0], 10, 64)
			if err == nil {
				if len(parts) == 4 && parts[1] != "field" && parts[1] != "save" {
					driverIDForCtx, errDriverID := strconv.ParseInt(parts[1], 10, 64)
					viewTypeForCtx := parts[2]
					pageForCtx, errPage := strconv.Atoi(parts[3])
					if errDriverID == nil && errPage == nil {
						tempSettleData := bh.Deps.SessionManager.GetTempDriverSettlement(chatID)
						tempSettleData.DriverUserIDForBackNav = driverIDForCtx
						tempSettleData.ViewTypeForBackNav = viewTypeForCtx
						tempSettleData.PageForBackNav = pageForCtx
						bh.Deps.SessionManager.UpdateTempDriverSettlement(chatID, tempSettleData)
						bh.handleOwnerEditSettlementStart(chatID, user, settlementID, viewTypeForCtx, pageForCtx, originalMessageID)
					} else {
						log.Printf("CALLBACK_ADMIN: Ошибка парсинга параметров для %s_START_WITH_CONTEXT: %v", currentCommand, parts)
						bh.sendErrorMessageHelper(chatID, originalMessageID, "❌ Ошибка параметров редактирования.")
					}
				} else if len(parts) == 3 && parts[1] == "field" {
					fieldName := parts[2]
					bh.handleOwnerEditSettlementFieldPrompt(chatID, user, settlementID, fieldName, originalMessageID)
				} else if len(parts) == 2 && parts[1] == "save" {
					bh.handleOwnerSaveAllSettlementChanges(chatID, user, settlementID, originalMessageID)
				} else if len(parts) == 1 {
					settlementForMenu, errGet := db.GetDriverSettlementByID(settlementID)
					if errGet == nil {
						bh.SendOwnerEditSettlementFieldSelectMenu(chatID, settlementForMenu, originalMessageID)
					} else {
						bh.sendErrorMessageHelper(chatID, originalMessageID, "Ошибка загрузки отчета для редактирования.")
					}
				} else {
					log.Printf("CALLBACK_ADMIN: Некорректный формат для %s: %v", currentCommand, parts)
					bh.sendErrorMessageHelper(chatID, originalMessageID, "❌ Ошибка формата команды редактирования отчета.")
				}
			} else {
				log.Printf("CALLBACK_ADMIN: Неверный ID отчета для %s: %s", currentCommand, parts[0])
				bh.sendErrorMessageHelper(chatID, originalMessageID, "❌ Ошибка: неверный ID отчета.")
			}
		} else {
			log.Printf("CALLBACK_ADMIN: Некорректный формат для %s: ID отчета отсутствует.", currentCommand)
			bh.sendErrorMessageHelper(chatID, originalMessageID, "❌ Ошибка: ID отчета не указан.")
		}
	case constants.CALLBACK_PREFIX_OWNER_CASH_MARK_SALARY_PAID:
		if len(parts) == 1 {
			settlementID, err := strconv.ParseInt(parts[0], 10, 64)
			if err == nil {
				bh.handleOwnerMarkSalaryPaid(chatID, user, settlementID, originalMessageID)
			} else {
				bh.sendErrorMessageHelper(chatID, originalMessageID, "Ошибка: неверный ID отчета для выплаты ЗП.")
			}
		}
	case constants.CALLBACK_PREFIX_OWNER_CASH_MARK_SALARY_UNPAID:
		if len(parts) == 1 {
			settlementID, err := strconv.ParseInt(parts[0], 10, 64)
			if err == nil {
				bh.handleOwnerMarkSalaryUnpaid(chatID, user, settlementID, originalMessageID)
			} else {
				bh.sendErrorMessageHelper(chatID, originalMessageID, "Ошибка: неверный ID отчета для отмены выплаты ЗП.")
			}
		}
	case constants.CALLBACK_PREFIX_OWNER_MARK_ALL_SALARY_PAID:
		if len(parts) == 2 {
			driverUID, errUID := strconv.ParseInt(parts[0], 10, 64)
			viewType := parts[1]
			if errUID == nil && (viewType == constants.VIEW_TYPE_ACTUAL_SETTLEMENTS || viewType == constants.VIEW_TYPE_SETTLED_SETTLEMENTS) {
				bh.handleOwnerMarkAllSalaryPaid(chatID, user, driverUID, viewType, originalMessageID)
			} else {
				bh.sendErrorMessageHelper(chatID, originalMessageID, "Ошибка: неверные параметры для массовой выплаты ЗП.")
			}
		}
	case constants.CALLBACK_PREFIX_OWNER_MARK_ALL_MONEY_DEPOSITED:
		if len(parts) == 2 {
			driverUID, errUID := strconv.ParseInt(parts[0], 10, 64)
			viewType := parts[1]
			if errUID == nil && viewType == constants.VIEW_TYPE_ACTUAL_SETTLEMENTS {
				bh.handleOwnerMarkAllMoneyDeposited(chatID, user, driverUID, viewType, originalMessageID)
			} else {
				bh.sendErrorMessageHelper(chatID, originalMessageID, "Ошибка: неверные параметры для массового внесения денег.")
			}
		}

	// --- Выплаты сотрудникам (Владелец) ---
	case constants.CALLBACK_PREFIX_OWNER_STAFF_PAYOUT:
		bh.SendOwnerPayoutsMainMenu(chatID, user, originalMessageID)
	case fmt.Sprintf("%s_page", constants.CALLBACK_PREFIX_OWNER_STAFF_PAYOUT): // parts: [PAGE_NUM]
		if len(parts) == 1 {
			page, _ := strconv.Atoi(parts[0])
			bh.SendOwnerStaffListForPayout(chatID, user, originalMessageID, page)
		}
	case fmt.Sprintf("%s_select", constants.CALLBACK_PREFIX_OWNER_STAFF_PAYOUT): // parts: [TARGET_USER_ID, AMOUNT_OWED_STR]
		if len(parts) == 2 {
			targetUserID, _ := strconv.ParseInt(parts[0], 10, 64)
			amountOwed, _ := strconv.ParseFloat(parts[1], 64)
			bh.SendOwnerConfirmPayoutToStaff(chatID, user, targetUserID, amountOwed, originalMessageID)
		}
	case fmt.Sprintf("%s_confirm", constants.CALLBACK_PREFIX_OWNER_STAFF_PAYOUT): // parts: [TARGET_USER_ID, AMOUNT_TO_PAY_STR]
		if len(parts) == 2 {
			targetUserID, _ := strconv.ParseInt(parts[0], 10, 64)
			amountToPay, _ := strconv.ParseFloat(parts[1], 64)
			bh.handleOwnerDoStaffPayout(chatID, user, targetUserID, amountToPay, originalMessageID)
		}
	default:
		foundSpecificHandlerInSwitch = false
		log.Printf("[CALLBACK_ADMIN] ОШИБКА: Неизвестная команда '%s' передана в dispatchAdminCallbacks (внутренний switch). Parts: %v, Data: '%s', ChatID=%d", currentCommand, parts, data, chatID)
		sentMsg, errHelper = bh.sendErrorMessageHelper(chatID, originalMessageID, "Неизвестная административная команда (внутр).")
	}

	if errHelper == nil && sentMsg.MessageID != 0 {
		newMenuMessageID = sentMsg.MessageID
	} else if foundSpecificHandlerInSwitch {
		currentHandlerState := bh.Deps.SessionManager.GetState(chatID)
		isDriverOrOwnerFinancialContext := strings.HasPrefix(currentHandlerState, "driver_report_") ||
			currentHandlerState == constants.STATE_OWNER_FINANCIAL_EDIT_FIELD ||
			currentHandlerState == constants.STATE_OWNER_FINANCIAL_EDIT_RECORD ||
			strings.HasPrefix(currentHandlerState, "owner_cash_") ||
			strings.HasPrefix(currentCommand, constants.CALLBACK_PREFIX_DRIVER_REPORT_OVERALL_MENU) ||
			strings.HasPrefix(currentCommand, constants.CALLBACK_PREFIX_OWNER_FINANCIALS) ||
			utils.IsCommandInCategory(currentCommand, ownerCashManagementCommands)

		if isDriverOrOwnerFinancialContext {
			sessionMsgID := bh.Deps.SessionManager.GetTempDriverSettlement(chatID).CurrentMessageID
			if sessionMsgID == 0 && utils.IsCommandInCategory(currentCommand, ownerCashManagementCommands) {
				sessionMsgID = bh.Deps.SessionManager.GetTempOrder(chatID).CurrentMessageID
			}
			if sessionMsgID != 0 {
				newMenuMessageID = sessionMsgID
			}
		} else {
			newMenuMessageID = bh.Deps.SessionManager.GetTempOrder(chatID).CurrentMessageID
		}
	}

	log.Printf("[CALLBACK_ADMIN] Диспетчер административных коллбэков завершен. Команда='%s', ChatID=%d, ID нового/текущего меню: %d", currentCommand, chatID, newMenuMessageID)
	return newMenuMessageID
}

// handleStaffAddRoleFinal (пример существующей функции, которую нужно проверить на использование tempData и корректный возврат к меню)
func (bh *BotHandler) handleStaffAddRoleFinal(adminChatID int64, roleKey string, messageIDToEdit int) {
	log.Printf("BotHandler.handleStaffAddRoleFinal: AdminChatID=%d, RoleKey=%s, MessageID=%d", adminChatID, roleKey, messageIDToEdit)
	bh.Deps.SessionManager.SetState(adminChatID, constants.STATE_STAFF_MENU) // Возвращаем в главное меню штата

	tempData := bh.Deps.SessionManager.GetTempOrder(adminChatID) // Данные для нового сотрудника хранятся в TempOrder
	staffName := tempData.Name
	staffSurname := tempData.Description                                                             // Использовали Description для фамилии
	staffNickname := sql.NullString{String: tempData.Subcategory, Valid: tempData.Subcategory != ""} // Subcategory для ника
	staffPhone := sql.NullString{String: tempData.Phone, Valid: tempData.Phone != ""}
	staffChatID := tempData.BlockTargetChatID // Использовали BlockTargetChatID для chatID нового сотрудника
	staffCardNumberStr := tempData.Payment    // Использовали Payment для номера карты

	if staffName == "" || staffSurname == "" || staffChatID == 0 {
		log.Printf("handleStaffAddRoleFinal: Недостаточно данных в сессии. AdminChatID: %d", adminChatID)
		bh.sendErrorMessageHelper(adminChatID, messageIDToEdit, "❌ Ошибка: не все данные для добавления собраны.")
		bh.SendStaffMenu(adminChatID, messageIDToEdit) // Возврат в меню штата
		return
	}

	var staffCardNumber sql.NullString
	if staffCardNumberStr != "" && staffCardNumberStr != "-" {
		re := regexp.MustCompile(`^[0-9]{16,19}$`)
		if !re.MatchString(staffCardNumberStr) {
			log.Printf("handleStaffAddRoleFinal: Неверный формат карты '%s'. AdminChatID: %d", staffCardNumberStr, adminChatID)
			bh.SendStaffAddPrompt(adminChatID, constants.STATE_STAFF_ADD_CARD_NUMBER, "❌ Неверный формат карты. Введите 16-19 цифр или '-' для пропуска:", "staff_add_prompt_chatid", messageIDToEdit)
			return
		}
		staffCardNumber = sql.NullString{String: staffCardNumberStr, Valid: true}
	}

	existingUser, errUserDB := db.GetUserByChatID(staffChatID)
	if errUserDB == nil && existingUser.ID != 0 {
		log.Printf("handleStaffAddRoleFinal: Пользователь с ChatID %d существует, данные будут обновлены.", staffChatID)
	} else if errUserDB != nil && errUserDB != sql.ErrNoRows {
		log.Printf("handleStaffAddRoleFinal: Ошибка проверки ChatID %d: %v", staffChatID, errUserDB)
		bh.sendErrorMessageHelper(adminChatID, messageIDToEdit, "❌ Ошибка проверки данных.")
		return
	}

	errDb := db.AddStaff(staffChatID, roleKey, staffName, staffSurname, staffNickname, staffPhone, staffCardNumber)
	if errDb != nil {
		log.Printf("handleStaffAddRoleFinal: Ошибка добавления/обновления сотрудника %d в БД: %v", staffChatID, errDb)
		bh.sendErrorMessageHelper(adminChatID, messageIDToEdit, "❌ Ошибка сохранения данных сотрудника.")
		return
	}

	bh.Deps.SessionManager.ClearTempOrder(adminChatID)
	confirmationText := fmt.Sprintf("✅ Сотрудник *%s %s* (ChatID: `%d`) успешно добавлен/обновлен с ролью *%s*!",
		utils.EscapeTelegramMarkdown(staffName), utils.EscapeTelegramMarkdown(staffSurname),
		staffChatID, utils.EscapeTelegramMarkdown(utils.GetRoleDisplayName(roleKey)))
	if staffCardNumber.Valid {
		confirmationText += fmt.Sprintf("\nКарта: `****%s`", utils.EscapeTelegramMarkdown(staffCardNumber.String[len(staffCardNumber.String)-4:]))
	}
	bh.SendStaffActionConfirmation(adminChatID, confirmationText, messageIDToEdit, staffChatID)
}

func (bh *BotHandler) handleStaffEditRoleFinal(adminChatID int64, targetChatID int64, newRoleKey string, messageIDToEdit int) {
	log.Printf("BotHandler.handleStaffEditRoleFinal: AdminChatID=%d, TargetChatID=%d, NewRole=%s, MessageID=%d", adminChatID, targetChatID, newRoleKey, messageIDToEdit)
	bh.Deps.SessionManager.SetState(adminChatID, constants.STATE_STAFF_INFO)

	adminUser, okAdmin := bh.getUserFromDB(adminChatID)
	if !okAdmin {
		bh.sendErrorMessageHelper(adminChatID, messageIDToEdit, "Ошибка получения данных администратора.")
		return
	}
	targetUser, okTarget := bh.getUserFromDB(targetChatID)
	if !okTarget {
		bh.sendErrorMessageHelper(adminChatID, messageIDToEdit, "Ошибка получения данных сотрудника.")
		return
	}

	if targetUser.Role == constants.ROLE_OWNER && adminUser.Role != constants.ROLE_OWNER {
		bh.sendErrorMessageHelper(adminChatID, messageIDToEdit, "🚫 Только другой Владелец может изменить роль Владельца.")
		bh.SendStaffInfo(adminChatID, targetChatID, messageIDToEdit)
		return
	}
	if targetUser.Role == constants.ROLE_OWNER && newRoleKey != constants.ROLE_OWNER && adminUser.Role == constants.ROLE_OWNER && adminChatID == targetChatID {
		bh.sendErrorMessageHelper(adminChatID, messageIDToEdit, "🚫 Владелец не может сам себя лишить роли Владельца.")
		bh.SendStaffInfo(adminChatID, targetChatID, messageIDToEdit)
		return
	}

	errDb := db.UpdateUserRole(targetChatID, newRoleKey)
	if errDb != nil {
		log.Printf("handleStaffEditRoleFinal: Ошибка обновления роли для сотрудника %d: %v", targetChatID, errDb)
		bh.sendErrorMessageHelper(adminChatID, messageIDToEdit, "❌ Ошибка изменения роли сотрудника.")
		return
	}

	confirmationText := fmt.Sprintf("✅ Роль сотрудника *%s %s* (ChatID: `%d`) успешно изменена на *%s*!",
		utils.EscapeTelegramMarkdown(targetUser.FirstName),
		utils.EscapeTelegramMarkdown(targetUser.LastName),
		targetChatID,
		utils.EscapeTelegramMarkdown(utils.GetRoleDisplayName(newRoleKey)),
	)
	updatedMsg, _ := bh.sendInfoMessage(adminChatID, messageIDToEdit, confirmationText, fmt.Sprintf("staff_info_%d", targetChatID))
	messageIDForNextMenu := messageIDToEdit
	if updatedMsg.MessageID != 0 {
		messageIDForNextMenu = updatedMsg.MessageID
	}
	bh.SendStaffInfo(adminChatID, targetChatID, messageIDForNextMenu)
	bh.Deps.SessionManager.SetState(adminChatID, constants.STATE_STAFF_INFO)
}

func (bh *BotHandler) handleStaffUnblockConfirm(adminChatID int64, targetChatID int64, messageIDToEdit int) {
	log.Printf("BotHandler.handleStaffUnblockConfirm: AdminChatID=%d, TargetChatID=%d, MessageID=%d", adminChatID, targetChatID, messageIDToEdit)

	targetUser, ok := bh.getUserFromDB(targetChatID)
	if !ok {
		bh.sendErrorMessageHelper(adminChatID, messageIDToEdit, "Ошибка получения данных сотрудника.")
		bh.SendStaffListMenu(adminChatID, messageIDToEdit)
		return
	}
	if !targetUser.IsBlocked {
		bh.sendInfoMessage(adminChatID, messageIDToEdit, "ℹ️ Сотрудник уже разблокирован.", fmt.Sprintf("staff_info_%d", targetChatID))
		return
	}

	errDb := db.UnblockUser(targetChatID)
	if errDb != nil {
		log.Printf("handleStaffUnblockConfirm: Ошибка разблокировки сотрудника %d: %v", targetChatID, errDb)
		bh.sendErrorMessageHelper(adminChatID, messageIDToEdit, "❌ Ошибка разблокировки сотрудника.")
		return
	}

	bh.sendMessage(targetChatID, "🔓 Администратор разблокировал ваш аккаунт.")
	confirmationText := fmt.Sprintf("✅ Сотрудник *%s %s* (ChatID: `%d`) успешно разблокирован!",
		utils.EscapeTelegramMarkdown(targetUser.FirstName),
		utils.EscapeTelegramMarkdown(targetUser.LastName),
		targetChatID,
	)
	updatedMsg, _ := bh.sendInfoMessage(adminChatID, messageIDToEdit, confirmationText, fmt.Sprintf("staff_info_%d", targetChatID))
	messageIDForNextMenu := messageIDToEdit
	if updatedMsg.MessageID != 0 {
		messageIDForNextMenu = updatedMsg.MessageID
	}
	bh.SendStaffInfo(adminChatID, targetChatID, messageIDForNextMenu)
	bh.Deps.SessionManager.SetState(adminChatID, constants.STATE_STAFF_INFO)
}

func (bh *BotHandler) handleStaffDeleteConfirm(adminChatID int64, targetChatID int64, messageIDToEdit int) {
	log.Printf("BotHandler.handleStaffDeleteConfirm: AdminChatID=%d, TargetChatID=%d, MessageID=%d", adminChatID, targetChatID, messageIDToEdit)

	targetUser, ok := bh.getUserFromDB(targetChatID)
	if !ok {
		bh.sendErrorMessageHelper(adminChatID, messageIDToEdit, "Ошибка получения данных сотрудника.")
		bh.SendStaffListMenu(adminChatID, messageIDToEdit)
		return
	}

	if targetUser.Role == constants.ROLE_OWNER {
		bh.sendErrorMessageHelper(adminChatID, messageIDToEdit, "🚫 Нельзя удалить Владельца таким способом.")
		bh.SendStaffInfo(adminChatID, targetChatID, messageIDToEdit)
		return
	}
	if targetUser.Role == constants.ROLE_USER {
		bh.sendInfoMessage(adminChatID, messageIDToEdit, "ℹ️ Этот пользователь уже не является сотрудником.", fmt.Sprintf("staff_info_%d", targetChatID))
		return
	}

	errDb := db.DeleteStaff(targetChatID)
	if errDb != nil {
		log.Printf("handleStaffDeleteConfirm: Ошибка 'удаления' сотрудника %d: %v", targetChatID, errDb)
		bh.sendErrorMessageHelper(adminChatID, messageIDToEdit, "❌ Ошибка при 'удалении' сотрудника (смене роли).")
		return
	}

	bh.sendMessage(targetChatID, "Ваша роль сотрудника была изменена. Вы теперь обычный пользователь.")
	confirmationText := fmt.Sprintf("🗑️ Сотрудник *%s %s* (ChatID: `%d`) 'удален' (роль изменена на *%s*).",
		utils.EscapeTelegramMarkdown(targetUser.FirstName),
		utils.EscapeTelegramMarkdown(targetUser.LastName),
		targetChatID,
		utils.EscapeTelegramMarkdown(utils.GetRoleDisplayName(constants.ROLE_USER)),
	)
	updatedMsg, _ := bh.sendInfoMessage(adminChatID, messageIDToEdit, confirmationText, "staff_list_menu")
	messageIDForNextMenu := messageIDToEdit
	if updatedMsg.MessageID != 0 {
		messageIDForNextMenu = updatedMsg.MessageID
	}
	bh.SendStaffListMenu(adminChatID, messageIDForNextMenu)
	bh.Deps.SessionManager.SetState(adminChatID, constants.STATE_STAFF_MENU)
}

func (bh *BotHandler) handleStatsGetPeriod(periodKey string, data string, chatID int64, originalMessageID int) {
	log.Printf("[STATS_GET_HANDLER] Запрос статистики за период: '%s'. ChatID=%d", periodKey, chatID)
	var startDate, endDate time.Time
	now := time.Now()
	loc := now.Location()
	todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc)
	periodDescription := ""

	switch periodKey {
	case "today":
		startDate = todayStart
		endDate = todayStart.Add(24*time.Hour - 1*time.Nanosecond)
		periodDescription = "Сегодня"
	case "yesterday":
		startDate = todayStart.AddDate(0, 0, -1)
		endDate = todayStart.Add(-1 * time.Nanosecond)
		periodDescription = "Вчера"
	case "current_week":
		weekday := now.Weekday()
		daysToSubstract := int(weekday - time.Monday)
		if weekday == time.Sunday {
			daysToSubstract = 6
		}
		startDate = todayStart.AddDate(0, 0, -daysToSubstract)
		endDate = startDate.AddDate(0, 0, 7).Add(-1 * time.Nanosecond)
		periodDescription = "Текущая неделя"
	case "current_month":
		startDate = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, loc)
		endDate = startDate.AddDate(0, 1, 0).Add(-1 * time.Nanosecond)
		periodDescription = "Текущий месяц"
	case "last_week":
		weekday := now.Weekday()
		daysToSubstractCurrentWeekStart := int(weekday - time.Monday)
		if weekday == time.Sunday {
			daysToSubstractCurrentWeekStart = 6
		}
		currentWeekStart := todayStart.AddDate(0, 0, -daysToSubstractCurrentWeekStart)
		endDate = currentWeekStart.Add(-1 * time.Nanosecond)
		startDate = currentWeekStart.AddDate(0, 0, -7)
		periodDescription = "Прошлая неделя"
	case "last_month":
		firstDayOfCurrentMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, loc)
		endDate = firstDayOfCurrentMonth.Add(-1 * time.Nanosecond)
		startDate = time.Date(endDate.Year(), endDate.Month(), 1, 0, 0, 0, 0, loc)
		periodDescription = "Прошлый месяц"
	default:
		log.Printf("[STATS_GET_HANDLER] Ошибка: неизвестный ключ периода '%s'. ChatID=%d", periodKey, chatID)
		_, _ = bh.sendErrorMessageHelper(chatID, originalMessageID, "Неизвестный период для статистики.")
		bh.SendBasicStatsPeriodsMenu(chatID, originalMessageID)
		return
	}
	log.Printf("[STATS_GET_HANDLER] Расчетный период: %s - %s для ключа '%s'. ChatID=%d", startDate.Format("2006-01-02 15:04:05"), endDate.Format("2006-01-02 15:04:05"), periodKey, chatID)

	stats, err := db.GetStats(startDate, endDate)
	if err != nil {
		log.Printf("[STATS_GET_HANDLER] Ошибка БД при получении статистики за период '%s': %v. ChatID=%d", periodKey, err, chatID)
		_, _ = bh.sendErrorMessageHelper(chatID, originalMessageID, "Ошибка получения статистики из базы данных.")
		return
	}
	bh.DisplayStats(chatID, originalMessageID, stats, periodDescription)
}
func (bh *BotHandler) handleStatsSelectDay(chatID int64, user models.User, context string, year int, month time.Month, day int, originalMessageID int) {
	selectedDate := time.Date(year, month, day, 0, 0, 0, 0, time.Local)
	tempData := bh.Deps.SessionManager.GetTempOrder(chatID)

	if context == "custom_date" {
		startDate := selectedDate
		endDate := selectedDate.Add(24*time.Hour - 1*time.Nanosecond)
		stats, err := db.GetStats(startDate, endDate)
		if err != nil {
			log.Printf("[CALLBACK_ADMIN] Ошибка БД (custom_date) для %s: %v. ChatID=%d", selectedDate.Format("02.01.06"), err, chatID)
			_, _ = bh.sendErrorMessageHelper(chatID, originalMessageID, "Ошибка получения статистики.")
			return
		}
		bh.DisplayStats(chatID, originalMessageID, stats, selectedDate.Format("02.01.2006"))
		bh.Deps.SessionManager.ClearTempOrder(chatID)
	} else if context == "period_start" {
		tempData.Date = selectedDate.Format("2006-01-02") // Сохраняем дату начала в сессию
		bh.Deps.SessionManager.UpdateTempOrder(chatID, tempData)
		log.Printf("[CALLBACK_ADMIN] Начальная дата периода '%s' сохранена. Запрос конечной даты. ChatID=%d", tempData.Date, chatID)
		// Отправляем меню выбора месяца для конечной даты
		bh.SendMonthSelectionMenu(chatID, originalMessageID, year, "period_end")
	} else if context == "period_end" {
		startDateStr := tempData.Date
		if startDateStr == "" {
			log.Printf("[CALLBACK_ADMIN] Ошибка: начальная дата периода не найдена в сессии. ChatID=%d", chatID)
			_, _ = bh.sendErrorMessageHelper(chatID, originalMessageID, "Ошибка: начальная дата периода не найдена.")
			bh.SendStatsMenu(chatID, originalMessageID)
			return
		}
		startDate, errStart := time.ParseInLocation("2006-01-02", startDateStr, time.Local)
		if errStart != nil {
			log.Printf("[CALLBACK_ADMIN] Ошибка парсинга начальной даты периода '%s': %v. ChatID=%d", startDateStr, errStart, chatID)
			_, _ = bh.sendErrorMessageHelper(chatID, originalMessageID, "Ошибка парсинга начальной даты периода.")
			bh.SendStatsMenu(chatID, originalMessageID)
			return
		}
		// Конечная дата включает весь день
		endDate := selectedDate.Add(24*time.Hour - 1*time.Nanosecond)
		if endDate.Before(startDate) {
			log.Printf("[CALLBACK_ADMIN] Ошибка: конечная дата (%s) раньше начальной (%s). ChatID=%d", endDate.Format("02.01.06"), startDate.Format("02.01.06"), chatID)
			_, _ = bh.sendErrorMessageHelper(chatID, originalMessageID, "Конечная дата не может быть раньше начальной.")
			// Повторно запрашиваем конечную дату
			bh.SendMonthSelectionMenu(chatID, originalMessageID, endDate.Year(), "period_end")
			return
		}
		log.Printf("[CALLBACK_ADMIN] Статистика за период: %s - %s. ChatID=%d", startDate.Format("02.01.06"), selectedDate.Format("02.01.06"), chatID)
		stats, err := db.GetStats(startDate, endDate)
		if err != nil {
			log.Printf("[CALLBACK_ADMIN] Ошибка БД (custom_period) для %s - %s: %v. ChatID=%d", startDate.Format("02.01.06"), selectedDate.Format("02.01.06"), err, chatID)
			_, _ = bh.sendErrorMessageHelper(chatID, originalMessageID, "Ошибка получения статистики.")
			return
		}
		bh.DisplayStats(chatID, originalMessageID, stats, fmt.Sprintf("%s - %s", startDate.Format("02.01.2006"), selectedDate.Format("02.01.2006")))
		bh.Deps.SessionManager.ClearTempOrder(chatID)
	}
}

func (bh *BotHandler) handleStatsYearNavigation(statsContext string, yearStr string, data string, chatID int64, originalMessageID int) {
	year, errYear := strconv.Atoi(yearStr)
	if errYear != nil {
		log.Printf("[STATS_YEAR_NAV_HANDLER] Ошибка: неверный формат года '%s' для навигации. ChatID=%d", yearStr, chatID)
		_, _ = bh.sendErrorMessageHelper(chatID, originalMessageID, "Неверный формат года для навигации.")
		bh.SendStatsMenu(chatID, originalMessageID)
		return
	}
	log.Printf("[STATS_YEAR_NAV_HANDLER] Навигация по годам: Контекст=%s, Год=%d. ChatID=%d", statsContext, year, chatID)
	// Просто перерисовываем меню выбора месяца для нового года
	bh.SendMonthSelectionMenu(chatID, originalMessageID, year, statsContext)
}
func (bh *BotHandler) handleExcelGenerate(chatID int64, user models.User, reportType string, originalMessageID int) {
	log.Printf("[EXCEL_HANDLER] Генерация Excel отчета типа: '%s'. ChatID=%d", reportType, chatID)

	generatingMsg, errGenMsg := bh.sendOrEditMessageHelper(chatID, originalMessageID, fmt.Sprintf("⏳ Генерирую отчет '%s'... Это может занять некоторое время.", reportType), nil, "")
	if errGenMsg != nil {
		log.Printf("[EXCEL_HANDLER] Ошибка отправки сообщения 'Генерирую отчет...': %v. ChatID=%d", errGenMsg, chatID)
		return
	}
	messageIDToDeleteAfterGeneration := generatingMsg.MessageID

	switch reportType {
	case "orders":
		bh.generateAndSendOrdersExcel(chatID, messageIDToDeleteAfterGeneration)
	case "referrals":
		bh.generateAndSendReferralsExcel(chatID, messageIDToDeleteAfterGeneration)
	case "salaries":
		bh.generateAndSendSalariesExcel(chatID, messageIDToDeleteAfterGeneration)
	default:
		log.Printf("[EXCEL_HANDLER] Ошибка: неизвестный тип Excel отчета '%s'. ChatID=%d", reportType, chatID)
		if messageIDToDeleteAfterGeneration != 0 {
			bh.deleteMessageHelper(chatID, messageIDToDeleteAfterGeneration)
		}
		bh.sendErrorMessageHelper(chatID, 0, "Неизвестный тип Excel отчета.")
		bh.SendExcelMenu(chatID, 0)
		return
	}
}

func (bh *BotHandler) handleBlockUserFinal(operatorChatID int64, operatorUser models.User, targetUserChatID int64, originalMessageID int) {
	log.Printf("[BLOCK_USER_FINAL] Финальная блокировка пользователя ChatID=%d оператором ChatID=%d.", targetUserChatID, operatorChatID)
	var sentMsg tgbotapi.Message
	var errHelper error

	tempData := bh.Deps.SessionManager.GetTempOrder(operatorChatID)
	reason := tempData.BlockReason

	if reason == "" {
		log.Printf("[BLOCK_USER_FINAL] Ошибка: причина блокировки для пользователя ChatID=%d не найдена в сессии оператора ChatID=%d.", targetUserChatID, operatorChatID)
		sentMsg, errHelper = bh.sendErrorMessageHelper(operatorChatID, originalMessageID, "Причина блокировки не была указана. Пожалуйста, начните процесс блокировки заново.")
		currentMsgIDForBlockMenu := originalMessageID
		if errHelper == nil && sentMsg.MessageID != 0 {
			currentMsgIDForBlockMenu = sentMsg.MessageID
		}
		bh.SendBlockUserMenu(operatorChatID, currentMsgIDForBlockMenu)
		return
	}

	targetUser, errTarget := db.GetUserByChatID(targetUserChatID)
	if errTarget != nil {
		log.Printf("[BLOCK_USER_FINAL] Ошибка получения данных пользователя ChatID=%d для блокировки: %v", targetUserChatID, errTarget)
		_, _ = bh.sendErrorMessageHelper(operatorChatID, originalMessageID, "Не удалось найти пользователя для блокировки.")
		return
	}
	if targetUser.Role != constants.ROLE_USER {
		log.Printf("[BLOCK_USER_FINAL] Попытка заблокировать не обычного пользователя (Роль: %s, ChatID: %d) через это меню.", targetUser.Role, targetUserChatID)
		sentMsg, errHelper = bh.sendErrorMessageHelper(operatorChatID, originalMessageID, "Этого пользователя нельзя заблокировать через данное меню (он не является обычным пользователем). Используйте управление штатом для сотрудников.")
		currentMsgIDForBlockMenu := originalMessageID
		if errHelper == nil && sentMsg.MessageID != 0 {
			currentMsgIDForBlockMenu = sentMsg.MessageID
		}
		bh.SendBlockUserMenu(operatorChatID, currentMsgIDForBlockMenu)
		return
	}

	errDb := db.BlockUser(targetUserChatID, reason)
	if errDb != nil {
		log.Printf("[BLOCK_USER_FINAL] Ошибка БД при блокировке пользователя ChatID=%d: %v", targetUserChatID, errDb)
		_, _ = bh.sendErrorMessageHelper(operatorChatID, originalMessageID, "❌ Ошибка блокировки пользователя.")
		return
	}

	log.Printf("[BLOCK_USER_FINAL] Пользователь ChatID=%d успешно заблокирован оператором ChatID=%d. Причина: %s. Активные заказы пользователя также были отменены.", targetUserChatID, operatorChatID, reason)
	bh.sendMessage(targetUserChatID, fmt.Sprintf("🚫 Вы были заблокированы администратором. Причина: %s. Ваши активные заказы были отменены.", reason))

	finalConfirmationText := "✅ Пользователь успешно заблокирован. Все его активные заказы были автоматически отменены."
	bh.sendInfoMessage(operatorChatID, originalMessageID, finalConfirmationText, "block_user_menu")
	bh.Deps.SessionManager.ClearTempOrder(operatorChatID)
	bh.Deps.SessionManager.ClearState(operatorChatID)
}

func (bh *BotHandler) handleUnblockUserFinal(operatorChatID int64, operatorUser models.User, targetUserChatID int64, originalMessageID int) {
	log.Printf("[UNBLOCK_USER_FINAL] Финальная разблокировка пользователя ChatID=%d оператором ChatID=%d.", targetUserChatID, operatorChatID)

	errDb := db.UnblockUser(targetUserChatID)
	if errDb != nil {
		log.Printf("[UNBLOCK_USER_FINAL] Ошибка БД при разблокировке пользователя ChatID=%d: %v", targetUserChatID, errDb)
		_, _ = bh.sendErrorMessageHelper(operatorChatID, originalMessageID, "❌ Ошибка разблокировки пользователя.")
		return
	}
	log.Printf("[UNBLOCK_USER_FINAL] Пользователь ChatID=%d успешно разблокирован оператором ChatID=%d.", targetUserChatID, operatorChatID)
	bh.sendMessage(targetUserChatID, "🔓 Вы были разблокированы администратором.")
	bh.sendInfoMessage(operatorChatID, originalMessageID, "✅ Пользователь успешно разблокирован.", "block_user_menu")
	bh.Deps.SessionManager.ClearState(operatorChatID)
}

func (bh *BotHandler) sendAccessDenied(chatID int64, originalMessageID int) (tgbotapi.Message, error) {
	log.Printf("[ACCESS_DENIED] Отказ в доступе для ChatID=%d, OriginalMsgID=%d", chatID, originalMessageID)
	sentMsg, err := bh.sendOrEditMessageHelper(chatID, originalMessageID, constants.AccessDeniedMessage, nil, "")
	if err != nil {
		log.Printf("[ACCESS_DENIED] Ошибка отправки/редактирования сообщения об отказе в доступе: %v. ChatID=%d", err, chatID)
		if originalMessageID != 0 {
			newSentMsg, newErr := bh.sendMessage(chatID, constants.AccessDeniedMessage)
			if newErr != nil {
				log.Printf("[ACCESS_DENIED] КРИТИЧЕСКАЯ ОШИБКА: Не удалось отправить новое сообщение об отказе в доступе. ChatID=%d: %v", chatID, newErr)
				return tgbotapi.Message{}, newErr
			}
			return newSentMsg, nil
		}
		return tgbotapi.Message{}, err
	}
	return sentMsg, nil
}

func (bh *BotHandler) handleOwnerDoStaffPayout(ownerChatID int64, ownerUser models.User, targetUserID int64, amountToPay float64, originalMessageID int) {
	log.Printf("handleOwnerDoStaffPayout: Владелец %d (UserID: %d) выплачивает %.0f сотруднику UserID %d", ownerChatID, ownerUser.ID, amountToPay, targetUserID)

	if ownerUser.Role != constants.ROLE_OWNER {
		bh.sendAccessDenied(ownerChatID, originalMessageID)
		return
	}
	if amountToPay <= 0 {
		bh.sendErrorMessageHelper(ownerChatID, originalMessageID, "❌ Сумма выплаты должна быть больше нуля.")
		bh.SendOwnerStaffListForPayout(ownerChatID, ownerUser, originalMessageID, 0)
		return
	}

	targetStaff, errStaff := db.GetUserByID(int(targetUserID))
	if errStaff != nil {
		log.Printf("handleOwnerDoStaffPayout: Ошибка получения данных сотрудника UserID %d: %v", targetUserID, errStaff)
		bh.sendErrorMessageHelper(ownerChatID, originalMessageID, "❌ Ошибка: не удалось найти данные сотрудника для выплаты.")
		return
	}

	payout := models.Payout{
		UserID:       targetUserID,
		Amount:       amountToPay,
		PayoutDate:   time.Now(),
		Comment:      fmt.Sprintf("Общая выплата от Владельца (ID: %d) сотруднику %s (ID: %d)", ownerUser.ID, utils.GetUserDisplayName(targetStaff), targetUserID),
		MadeByUserID: ownerUser.ID,
	}

	payoutID, errPayout := db.AddPayout(payout)
	if errPayout != nil {
		log.Printf("handleOwnerDoStaffPayout: Ошибка создания записи о выплате для сотрудника UserID %d: %v", targetUserID, errPayout)
		bh.sendErrorMessageHelper(ownerChatID, originalMessageID, "❌ Ошибка при регистрации выплаты.")
		return
	}

	log.Printf("Выплата #%d на сумму %.0f сотруднику UserID %d успешно зарегистрирована Владельцем UserID %d.", payoutID, amountToPay, targetUserID, ownerUser.ID)
	bh.sendMessage(targetStaff.ChatID, fmt.Sprintf("💸 Вам была произведена выплата на сумму %.0f ₽. Детали можно посмотреть в разделе 'Моя зарплата'.", amountToPay))
	updatedMsg, errInfo := bh.sendInfoMessage(ownerChatID, originalMessageID,
		fmt.Sprintf("✅ Выплата %.0f ₽ сотруднику %s успешно зарегистрирована.", amountToPay, utils.GetUserDisplayName(targetStaff)),
		fmt.Sprintf("%s_page_0", constants.CALLBACK_PREFIX_OWNER_STAFF_PAYOUT))

	messageIDForListUpdate := originalMessageID
	if errInfo == nil && updatedMsg.MessageID != 0 {
		messageIDForListUpdate = updatedMsg.MessageID
	}
	bh.SendOwnerStaffListForPayout(ownerChatID, ownerUser, messageIDForListUpdate, 0)
}

func (bh *BotHandler) handleOwnerMarkSalaryPaid(chatID int64, user models.User, settlementID int64, originalMessageID int) {
	log.Printf("handleOwnerMarkSalaryPaid: Владелец %d помечает ЗП по отчету #%d как выплаченную. OriginalMsgID: %d", chatID, settlementID, originalMessageID)
	errDb := db.MarkDriverSalaryAsPaid(settlementID)
	if errDb != nil {
		log.Printf("handleOwnerMarkSalaryPaid: Ошибка пометки ЗП по отчету #%d как выплаченной: %v", settlementID, errDb)
		bh.sendErrorMessageHelper(chatID, originalMessageID, "❌ Ошибка при отметке выплаты ЗП.")
		return
	}
	log.Printf("ЗП по отчету #%d помечена как выплаченная. Проверка и обновление связанных заказов завершены (если были).", settlementID)
	tempData := bh.Deps.SessionManager.GetTempDriverSettlement(chatID)
	bh.SendOwnerDriverIndividualSettlementsList(chatID, user, originalMessageID, tempData.DriverUserIDForBackNav, tempData.ViewTypeForBackNav, tempData.PageForBackNav)
}

func (bh *BotHandler) handleOwnerMarkSalaryUnpaid(chatID int64, user models.User, settlementID int64, originalMessageID int) {
	errDb := db.MarkDriverSalaryAsUnpaid(settlementID)
	if errDb != nil {
		log.Printf("CALLBACK_ADMIN: Ошибка снятия пометки ЗП по отчету #%d: %v", settlementID, errDb)
		bh.sendErrorMessageHelper(chatID, originalMessageID, "❌ Ошибка при отмене отметки выплаты ЗП.")
		return
	}
	settlement, errGet := db.GetDriverSettlementByID(settlementID)
	if errGet != nil {
		log.Printf("CALLBACK_ADMIN: Ошибка получения отчета #%d для обновления вида: %v", settlementID, errGet)
		bh.SendOwnerCashManagementMenu(chatID, user, originalMessageID)
		return
	}
	tempData := bh.Deps.SessionManager.GetTempDriverSettlement(chatID)
	bh.SendOwnerDriverIndividualSettlementsList(chatID, user, originalMessageID, settlement.DriverUserID, tempData.ViewTypeForBackNav, tempData.PageForBackNav)
}

func (bh *BotHandler) handleOwnerMarkAllSalaryPaid(chatID int64, owner models.User, driverUserID int64, viewType string, originalMessageID int) {
	log.Printf("handleOwnerMarkAllSalaryPaid: Владелец %d, Водитель %d, Тип списка %s", chatID, driverUserID, viewType)
	var reportsToUpdate []models.DriverSettlementWithDriverName
	page := 0
	processedSettlementIDs := make(map[int64]bool)

	for {
		settlements, total, err := db.GetDriverSettlementsForOwnerView(driverUserID, viewType, page, 100)
		if err != nil {
			log.Printf("Ошибка получения отчетов для 'ЗП за все' (водитель %d, тип %s, стр %d): %v", driverUserID, viewType, page, err)
			bh.sendErrorMessageHelper(chatID, originalMessageID, "Ошибка получения списка отчетов для массовой отметки.")
			return
		}
		if len(settlements) == 0 {
			break
		}
		for _, s := range settlements {
			if !s.DriverSalaryPaidAt.Valid && !processedSettlementIDs[s.ID] {
				reportsToUpdate = append(reportsToUpdate, s)
				processedSettlementIDs[s.ID] = true
			}
		}
		if page*100+len(settlements) >= total {
			break
		}
		page++
		if page > 20 {
			log.Printf("Превышен лимит страниц при выборке отчетов для 'ЗП за все'. Водитель %d", driverUserID)
			break
		}
	}

	if len(reportsToUpdate) == 0 {
		bh.sendInfoMessage(chatID, originalMessageID, "Нет отчетов в текущем списке, по которым нужно отметить выплату ЗП.", fmt.Sprintf("%s_%d_%s_0", constants.CALLBACK_PREFIX_OWNER_VIEW_DRIVER_SETTLEMENTS, driverUserID, viewType))
		return
	}

	successCount := 0
	errorCount := 0
	for _, report := range reportsToUpdate {
		err := db.MarkDriverSalaryAsPaid(report.ID)
		if err != nil {
			log.Printf("Ошибка отметки ЗП по отчету #%d: %v", report.ID, err)
			errorCount++
		} else {
			successCount++
		}
	}

	resultMessage := fmt.Sprintf("Обработка завершена:\nУспешно отмечено ЗП по %d отчетам.", successCount)
	if errorCount > 0 {
		resultMessage += fmt.Sprintf("\nОшибок при отметке: %d.", errorCount)
	}
	bh.sendInfoMessage(chatID, originalMessageID, resultMessage, fmt.Sprintf("%s_%d_%s_0", constants.CALLBACK_PREFIX_OWNER_VIEW_DRIVER_SETTLEMENTS, driverUserID, viewType))
	tempData := bh.Deps.SessionManager.GetTempDriverSettlement(chatID)
	bh.SendOwnerDriverIndividualSettlementsList(chatID, owner, originalMessageID, driverUserID, tempData.ViewTypeForBackNav, tempData.PageForBackNav)
}

func (bh *BotHandler) handleOwnerMarkAllMoneyDeposited(chatID int64, owner models.User, driverUserID int64, viewType string, originalMessageID int) {
	log.Printf("handleOwnerMarkAllMoneyDeposited: Владелец %d, Водитель %d, Тип списка %s", chatID, driverUserID, viewType)

	if viewType != constants.VIEW_TYPE_ACTUAL_SETTLEMENTS {
		bh.sendErrorMessageHelper(chatID, originalMessageID, "Эта операция доступна только для списка актуальных (не внесенных) отчетов.")
		return
	}
	var reportsToUpdate []models.DriverSettlementWithDriverName
	page := 0
	processedSettlementIDs := make(map[int64]bool)

	for {
		settlements, total, err := db.GetDriverSettlementsForOwnerView(driverUserID, viewType, page, 100)
		if err != nil {
			log.Printf("Ошибка получения отчетов для 'Деньги внесены за все' (водитель %d, тип %s, стр %d): %v", driverUserID, viewType, page, err)
			bh.sendErrorMessageHelper(chatID, originalMessageID, "Ошибка получения списка отчетов для массовой отметки.")
			return
		}
		if len(settlements) == 0 {
			break
		}
		for _, s := range settlements {
			if !s.PaidToOwnerAt.Valid && !processedSettlementIDs[s.ID] {
				reportsToUpdate = append(reportsToUpdate, s)
				processedSettlementIDs[s.ID] = true
			}
		}
		if page*100+len(settlements) >= total {
			break
		}
		page++
		if page > 20 {
			log.Printf("Превышен лимит страниц при выборке отчетов для 'Деньги внесены за все'. Водитель %d", driverUserID)
			break
		}
	}

	if len(reportsToUpdate) == 0 {
		bh.sendInfoMessage(chatID, originalMessageID, "Нет отчетов в текущем списке, по которым нужно отметить внесение денег.", fmt.Sprintf("%s_%d_%s_0", constants.CALLBACK_PREFIX_OWNER_VIEW_DRIVER_SETTLEMENTS, driverUserID, viewType))
		return
	}
	successCount := 0
	errorCount := 0
	for _, report := range reportsToUpdate {
		err := db.MarkSettlementAsPaidToOwner(report.ID)
		if err != nil {
			log.Printf("Ошибка отметки 'Деньги внесены' по отчету #%d: %v", report.ID, err)
			errorCount++
		} else {
			successCount++
		}
	}
	resultMessage := fmt.Sprintf("Обработка завершена:\nУспешно отмечено 'Деньги внесены' по %d отчетам.", successCount)
	if errorCount > 0 {
		resultMessage += fmt.Sprintf("\nОшибок при отметке: %d.", errorCount)
	}
	bh.sendInfoMessage(chatID, originalMessageID, resultMessage, fmt.Sprintf("%s_%d_%s_0", constants.CALLBACK_PREFIX_OWNER_VIEW_DRIVER_SETTLEMENTS, driverUserID, viewType))
	tempData := bh.Deps.SessionManager.GetTempDriverSettlement(chatID)
	bh.SendOwnerDriverIndividualSettlementsList(chatID, owner, originalMessageID, driverUserID, tempData.ViewTypeForBackNav, tempData.PageForBackNav)
}

// SendOperatorReviewSettlementMenu показывает оператору отчет для утверждения/отклонения.
func (bh *BotHandler) SendOperatorReviewSettlementMenu(operatorChatID int64, operatorUser models.User, settlementID int64, messageIDToEdit int) {
	log.Printf("SendOperatorReviewSettlementMenu: Оператор %d просматривает отчет #%d для решения.", operatorChatID, settlementID)
	bh.Deps.SessionManager.SetState(operatorChatID, constants.STATE_OPERATOR_REVIEW_SETTLEMENT)

	// Используем существующую функцию для отображения деталей, но добавляем новые кнопки
	settlement, err := db.GetDriverSettlementByID(settlementID)
	if err != nil {
		bh.sendErrorMessageHelper(operatorChatID, messageIDToEdit, "❌ Ошибка загрузки отчета для проверки.")
		return
	}

	driver, _ := db.GetUserByID(int(settlement.DriverUserID))
	driverDisplayName := utils.GetUserDisplayName(driver)

	var reportDetails strings.Builder
	reportDetails.WriteString(fmt.Sprintf("🧾 *Проверка Отчета Водителя: %s*\n", utils.EscapeTelegramMarkdown(driverDisplayName)))
	reportDetails.WriteString(fmt.Sprintf("🆔 Отчета: *%d* от %s\n\n", settlement.ID, settlement.SettlementTimestamp.Format("02.01.06 15:04")))
	reportDetails.WriteString(fmt.Sprintf("📦 *Покрытые заказы (%d шт.):*\n", settlement.CoveredOrdersCount))
	if len(settlement.CoveredOrderIDs) > 0 {
		var orderLinks []string
		for _, orderID := range settlement.CoveredOrderIDs {
			// Ссылка для оператора для просмотра заказа
			orderLinks = append(orderLinks, fmt.Sprintf("[#%d](tg://btn/%s%d)", orderID, "view_order_ops_", orderID))
		}
		reportDetails.WriteString(strings.Join(orderLinks, ", ") + "\n")
	} else {
		reportDetails.WriteString("_ID заказов не указаны_\n")
	}
	reportDetails.WriteString(fmt.Sprintf("\n💰 Общая выручка по заказам: *%.0f ₽*\n", settlement.CoveredOrdersRevenue))

	reportDetails.WriteString("\n*Расходы:*\n")
	reportDetails.WriteString(fmt.Sprintf("  ⛽️ Топливо: *%.0f ₽*\n", settlement.FuelExpense))

	if len(settlement.OtherExpenses) > 0 {
		reportDetails.WriteString("  🛠️ *Прочие расходы:*\n")
		for _, oe := range settlement.OtherExpenses {
			reportDetails.WriteString(fmt.Sprintf("    - %s: *%.0f ₽*\n", utils.EscapeTelegramMarkdown(oe.Description), oe.Amount))
		}
	} else {
		reportDetails.WriteString("  🛠️ Прочие расходы: *0 ₽*\n")
	}

	if len(settlement.LoaderPayments) > 0 {
		reportDetails.WriteString("\n👷 *Зарплаты грузчикам:*\n")
		totalLoaderSalary := 0.0
		for _, lp := range settlement.LoaderPayments {
			reportDetails.WriteString(fmt.Sprintf("  - %s: *%.0f ₽*\n", utils.EscapeTelegramMarkdown(lp.LoaderIdentifier), lp.Amount))
			totalLoaderSalary += lp.Amount
		}
		reportDetails.WriteString(fmt.Sprintf("  Итого грузчикам: *%.0f ₽*\n", totalLoaderSalary))
	}

	reportDetails.WriteString("\n*Итоги:*\n")
	reportDetails.WriteString(fmt.Sprintf("  💸 Расчетная ЗП водителя: *%.0f ₽*\n", settlement.DriverCalculatedSalary))
	reportDetails.WriteString(fmt.Sprintf("  ➡️ Сумма к сдаче в кассу: *%.0f ₽*\n\n", settlement.AmountToCashier))
	reportDetails.WriteString("Пожалуйста, проверьте данные и примите решение.")

	var keyboard tgbotapi.InlineKeyboardMarkup
	var rows [][]tgbotapi.InlineKeyboardButton

	// Кнопки действий только если отчет в статусе 'pending'
	if settlement.Status == constants.SETTLEMENT_STATUS_PENDING {
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("✅ Утвердить", fmt.Sprintf("%s_%d", constants.CALLBACK_PREFIX_OPERATOR_APPROVE_SETTLEMENT, settlement.ID)),
			tgbotapi.NewInlineKeyboardButtonData("❌ Отклонить", fmt.Sprintf("%s_%d", constants.CALLBACK_PREFIX_OPERATOR_REJECT_SETTLEMENT, settlement.ID)),
		))
	} else {
		reportDetails.WriteString(fmt.Sprintf("\n\n*Статус: %s*", settlement.Status))
		if settlement.AdminComment.Valid {
			reportDetails.WriteString(fmt.Sprintf("\n*Комментарий: %s*", utils.EscapeTelegramMarkdown(settlement.AdminComment.String)))
		}
	}

	rows = append(rows, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("🏢 Главное меню", "back_to_main"),
	))
	keyboard.InlineKeyboard = rows

	bh.sendOrEditMessageHelper(operatorChatID, messageIDToEdit, reportDetails.String(), &keyboard, tgbotapi.ModeMarkdown)
}

// handleOperatorApproveSettlement обрабатывает утверждение отчета.
func (bh *BotHandler) handleOperatorApproveSettlement(operatorChatID int64, operatorUser models.User, settlementID int64, messageIDToEdit int) {
	err := db.UpdateDriverSettlementStatus(settlementID, constants.SETTLEMENT_STATUS_APPROVED, sql.NullString{})
	if err != nil {
		bh.sendErrorMessageHelper(operatorChatID, messageIDToEdit, "❌ Ошибка утверждения отчета.")
		return
	}

	settlement, _ := db.GetDriverSettlementByID(settlementID)
	// Уведомляем водителя
	driver, _ := db.GetUserByID(int(settlement.DriverUserID))
	driverMessage := fmt.Sprintf("✅ Ваш отчет #%d был утвержден оператором %s.", settlement.ID, utils.GetUserDisplayName(operatorUser))
	bh.sendMessage(driver.ChatID, driverMessage)

	bh.sendInfoMessage(operatorChatID, messageIDToEdit, fmt.Sprintf("✅ Отчет #%d успешно утвержден.", settlementID), "back_to_main")
	bh.Deps.SessionManager.ClearState(operatorChatID)
}

// handleOperatorRejectSettlement обрабатывает нажатие кнопки "Отклонить".
func (bh *BotHandler) handleOperatorRejectSettlement(operatorChatID int64, operatorUser models.User, settlementID int64, messageIDToEdit int) {
	bh.Deps.SessionManager.SetState(operatorChatID, constants.STATE_OPERATOR_REJECT_REASON_INPUT)
	tempData := bh.Deps.SessionManager.GetTempDriverSettlement(operatorChatID)
	tempData.EditingSettlementID = settlementID
	bh.Deps.SessionManager.UpdateTempDriverSettlement(operatorChatID, tempData)

	promptText := fmt.Sprintf("📝 Укажите причину отклонения отчета #%d:", settlementID)
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("⬅️ Назад к отчету", fmt.Sprintf("%s_%d", constants.CALLBACK_PREFIX_OPERATOR_VIEW_DRIVER_SETTLEMENT, settlementID)),
		),
	)
	bh.sendOrEditMessageHelper(operatorChatID, messageIDToEdit, promptText, &keyboard, "")
}

// handleOperatorFinalizeRejection завершает процесс отклонения после ввода причины.
func (bh *BotHandler) handleOperatorFinalizeRejection(operatorChatID int64, operatorUser models.User, settlementID int64, reason string, messageIDToEdit int) {
	err := db.UpdateDriverSettlementStatus(settlementID, constants.SETTLEMENT_STATUS_REJECTED, sql.NullString{String: reason, Valid: true})
	if err != nil {
		bh.sendErrorMessageHelper(operatorChatID, messageIDToEdit, "❌ Ошибка отклонения отчета.")
		return
	}

	settlement, _ := db.GetDriverSettlementByID(settlementID)
	driver, _ := db.GetUserByID(int(settlement.DriverUserID))
	// Уведомляем водителя
	driverMessage := fmt.Sprintf("❌ Ваш отчет #%d был отклонен оператором %s.\nПричина: %s\n\nПожалуйста, создайте новый, исправленный отчет.",
		settlement.ID, utils.GetUserDisplayName(operatorUser), reason)
	bh.sendMessage(driver.ChatID, driverMessage)

	bh.sendInfoMessage(operatorChatID, messageIDToEdit, fmt.Sprintf("❌ Отчет #%d отклонен. Водитель уведомлен.", settlementID), "back_to_main")
	bh.Deps.SessionManager.ClearState(operatorChatID)
	bh.Deps.SessionManager.ClearTempDriverSettlement(operatorChatID)
}

// SendOperatorViewDriverSettlementDetails отправляет оператору детали отчета водителя.
func (bh *BotHandler) SendOperatorViewDriverSettlementDetails(operatorChatID int64, operatorUser models.User, settlementID int64, messageIDToEdit int) {
	log.Printf("SendOperatorViewDriverSettlementDetails: Оператор %d (UserID: %d) просматривает отчет #%d", operatorChatID, operatorUser.ID, settlementID)

	settlement, err := db.GetDriverSettlementByID(settlementID)
	if err != nil {
		log.Printf("SendOperatorViewDriverSettlementDetails: Ошибка получения отчета #%d: %v", settlementID, err)
		bh.sendErrorMessageHelper(operatorChatID, messageIDToEdit, "❌ Не удалось загрузить данные отчета.")
		return
	}

	// --- НАЧАЛО ИЗМЕНЕНИЯ: перенаправление на новое меню ---
	// Если отчет ожидает решения, показываем меню с кнопками
	if settlement.Status == constants.SETTLEMENT_STATUS_PENDING {
		bh.SendOperatorReviewSettlementMenu(operatorChatID, operatorUser, settlementID, messageIDToEdit)
		return // Важно выйти, чтобы не выполнять остальной код функции
	}
	// --- КОНЕЦ ИЗМЕНЕНИЯ ---

	// Остальной код выполняется, если статус НЕ pending (т.е. просто просмотр)
	driver, errDriver := db.GetUserByID(int(settlement.DriverUserID))
	driverDisplayName := fmt.Sprintf("Водитель ID %d", settlement.DriverUserID)
	if errDriver == nil {
		driverDisplayName = utils.GetUserDisplayName(driver)
	}

	var reportDetails strings.Builder
	reportDetails.WriteString(fmt.Sprintf("🧾 *Отчет Водителя: %s*\n", utils.EscapeTelegramMarkdown(driverDisplayName)))
	reportDetails.WriteString(fmt.Sprintf("🆔 Отчета: *%d*\n", settlement.ID))
	reportDetails.WriteString(fmt.Sprintf("📅 Дата отчета: *%s*\n", settlement.ReportDate.Format("02.01.2006")))
	reportDetails.WriteString(fmt.Sprintf("⏰ Время создания: *%s*\n\n", settlement.SettlementTimestamp.Format("02.01.06 15:04")))

	reportDetails.WriteString(fmt.Sprintf("📦 *Покрытые заказы (%d шт.):*\n", settlement.CoveredOrdersCount))
	if len(settlement.CoveredOrderIDs) > 0 {
		var orderLinks []string
		for _, orderID := range settlement.CoveredOrderIDs {
			orderLinks = append(orderLinks, fmt.Sprintf("[#%d](tg://btn/%s_%d)", orderID, "view_order_ops", orderID))
		}
		reportDetails.WriteString(strings.Join(orderLinks, ", ") + "\n")
	} else {
		reportDetails.WriteString("_ID заказов не указаны_\n")
	}
	reportDetails.WriteString(fmt.Sprintf("\n💰 Общая выручка по заказам: *%.0f ₽*\n", settlement.CoveredOrdersRevenue))

	reportDetails.WriteString("\n*Расходы:*\n")
	reportDetails.WriteString(fmt.Sprintf("  ⛽️ Топливо: *%.0f ₽*\n", settlement.FuelExpense))

	// Отображение прочих расходов
	if len(settlement.OtherExpenses) > 0 {
		reportDetails.WriteString("  🛠️ *Прочие расходы:*\n")
		for _, oe := range settlement.OtherExpenses {
			reportDetails.WriteString(fmt.Sprintf("    - %s: *%.0f ₽*\n", utils.EscapeTelegramMarkdown(oe.Description), oe.Amount))
		}
	} else {
		reportDetails.WriteString("  🛠️ Прочие расходы: *0 ₽*\n")
	}

	if len(settlement.LoaderPayments) > 0 {
		reportDetails.WriteString("\n👷 *Зарплаты грузчикам:*\n")
		totalLoaderSalary := 0.0
		for _, lp := range settlement.LoaderPayments {
			reportDetails.WriteString(fmt.Sprintf("  - %s: *%.0f ₽*\n", utils.EscapeTelegramMarkdown(lp.LoaderIdentifier), lp.Amount))
			totalLoaderSalary += lp.Amount
		}
		reportDetails.WriteString(fmt.Sprintf("  Итого грузчикам: *%.0f ₽*\n", totalLoaderSalary))
	}

	reportDetails.WriteString("\n*Итоги:*\n")
	reportDetails.WriteString(fmt.Sprintf("  💸 Расчетная ЗП водителя: *%.0f ₽*\n", settlement.DriverCalculatedSalary))
	reportDetails.WriteString(fmt.Sprintf("  ➡️ Сумма к сдаче в кассу: *%.0f ₽*\n", settlement.AmountToCashier))

	var statusMoney, statusSalary string
	if settlement.PaidToOwnerAt.Valid {
		statusMoney = fmt.Sprintf("✅ Деньги внесены (%s)", settlement.PaidToOwnerAt.Time.Format("02.01.06 15:04"))
	} else {
		statusMoney = "❌ Деньги НЕ внесены"
	}
	if settlement.DriverSalaryPaidAt.Valid {
		statusSalary = fmt.Sprintf("✅ ЗП выплачена (%s)", settlement.DriverSalaryPaidAt.Time.Format("02.01.06 15:04"))
	} else {
		statusSalary = "❌ ЗП НЕ выплачена"
	}
	reportDetails.WriteString(fmt.Sprintf("\n*Статус расчета:*\n  %s\n  %s\n", statusMoney, statusSalary))

	// Добавляем статус самого отчета
	reportDetails.WriteString(fmt.Sprintf("\n*Статус отчета: %s*", settlement.Status))
	if settlement.AdminComment.Valid {
		reportDetails.WriteString(fmt.Sprintf("\n*Комментарий: %s*", utils.EscapeTelegramMarkdown(settlement.AdminComment.String)))
	}

	var rows [][]tgbotapi.InlineKeyboardButton // <--- ИЗМЕНЕНИЕ ЗДЕСЬ
	if utils.IsRoleOrHigher(operatorUser.Role, constants.ROLE_OWNER) {
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("✏️ Редактировать этот отчет", fmt.Sprintf("%s_%d_%d_%s_%d", constants.CALLBACK_PREFIX_OWNER_EDIT_SETTLEMENT, settlement.ID, settlement.DriverUserID, constants.VIEW_TYPE_ACTUAL_SETTLEMENTS, 0)),
		))
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("💰 К упр. денежными средствами", constants.CALLBACK_PREFIX_OWNER_CASH_MANAGEMENT_MAIN),
		))
	}

	rows = append(rows, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("🏢 Главное меню", "back_to_main"),
	))
	keyboard := tgbotapi.NewInlineKeyboardMarkup(rows...)

	sentMsg, err := bh.sendOrEditMessageHelper(operatorChatID, messageIDToEdit, reportDetails.String(), &keyboard, tgbotapi.ModeMarkdown)
	if err != nil {
		log.Printf("SendOperatorViewDriverSettlementDetails: Ошибка отправки/редактирования деталей отчета #%d оператору %d: %v", settlementID, operatorChatID, err)
	} else {
		if sentMsg.MessageID != 0 && messageIDToEdit == 0 {
			log.Printf("SendOperatorViewDriverSettlementDetails: Отправлено новое сообщение %d оператору %d с деталями отчета #%d", sentMsg.MessageID, operatorChatID, settlementID)
		} else if sentMsg.MessageID != 0 && messageIDToEdit != 0 {
			log.Printf("SendOperatorViewDriverSettlementDetails: Отредактировано сообщение %d оператору %d с деталями отчета #%d", sentMsg.MessageID, operatorChatID, settlementID)
		}
	}
}
