package handlers

import (
	"fmt"
	"log"
	"strconv"
	"strings" // Добавлено для strings.Join / Added for strings.Join
	"time"

	"Original/internal/constants"
	"Original/internal/db"
	"Original/internal/models"
	"Original/internal/utils"
	tgbotapi "github.com/OvyFlash/telegram-bot-api"
)

// dispatchInfoCommsCallbacks маршрутизирует коллбэки, связанные с информацией, коммуникацией и реферальной программой.
// currentCommand - это уже определенная основная команда (например, "invite_friend", "referral_link").
// parts - это оставшиеся части callback_data после извлечения currentCommand.
// data - это полная строка callback_data.
// Возвращает ID нового отправленного/отредактированного сообщения или 0.
// dispatchInfoCommsCallbacks routes callbacks related to information, communication, and the referral program.
// currentCommand is the already defined main command (e.g., "invite_friend", "referral_link").
// parts are the remaining parts of callback_data after extracting currentCommand.
// data is the full callback_data string.
// Returns the ID of the new sent/edited message or 0.
func (bh *BotHandler) dispatchInfoCommsCallbacks(currentCommand string, parts []string, data string, chatID int64, user models.User, originalMessageID int) int {
	log.Printf("[CALLBACK_INFO_COMMS] Диспетчер: Команда='%s', Части=%v, Data='%s', ChatID=%d", currentCommand, parts, data, chatID)
	var newMenuMessageID int = originalMessageID
	var sentMsg tgbotapi.Message
	var errHelper error

	switch currentCommand {
	case "invite_friend":
		log.Printf("[CALLBACK_INFO_COMMS] Запрос меню 'Пригласить друга'. ChatID=%d", chatID)
		bh.SendInviteFriendMenu(chatID, originalMessageID)
		newMenuMessageID = bh.Deps.SessionManager.GetTempOrder(chatID).CurrentMessageID
	case "contact_operator":
		log.Printf("[CALLBACK_INFO_COMMS] Запрос меню связи с оператором. ChatID=%d", chatID)
		bh.SendContactOperatorMenu(chatID, user, originalMessageID)
		newMenuMessageID = bh.Deps.SessionManager.GetTempOrder(chatID).CurrentMessageID
	case "contact_chat":
		log.Printf("[CALLBACK_INFO_COMMS] Запрос на начало чата с оператором. ChatID=%d", chatID)
		bh.SendChatMessageInputPrompt(chatID, originalMessageID)
		newMenuMessageID = bh.Deps.SessionManager.GetTempOrder(chatID).CurrentMessageID
	case "contact_phone_options":
		log.Printf("[CALLBACK_INFO_COMMS] Запрос опций связи по телефону. ChatID=%d", chatID)
		bh.SendPhoneOptionsMenu(chatID, originalMessageID)
		newMenuMessageID = bh.Deps.SessionManager.GetTempOrder(chatID).CurrentMessageID
	case "client_chats": // Просмотр оператором списка активных чатов / Operator views list of active chats
		if !utils.IsOperatorOrHigher(user.Role) {
			log.Printf("[CALLBACK_INFO_COMMS] Доступ запрещен к 'client_chats' для ChatID=%d, Роль=%s", chatID, user.Role)
			sentMsg, _ = bh.sendAccessDenied(chatID, originalMessageID)
			if sentMsg.MessageID != 0 {
				newMenuMessageID = sentMsg.MessageID
			}
			return newMenuMessageID
		}
		log.Printf("[CALLBACK_INFO_COMMS] Оператор ChatID=%d запрашивает список чатов с клиентами.", chatID)
		bh.SendClientChatsMenu(chatID, originalMessageID)
		newMenuMessageID = bh.Deps.SessionManager.GetTempOrder(chatID).CurrentMessageID
	case "view_chat_history": // parts: [CLIENT_CHAT_ID]
		if !utils.IsOperatorOrHigher(user.Role) {
			log.Printf("[CALLBACK_INFO_COMMS] Доступ запрещен к 'view_chat_history' для ChatID=%d, Роль=%s", chatID, user.Role)
			sentMsg, _ = bh.sendAccessDenied(chatID, originalMessageID)
			if sentMsg.MessageID != 0 {
				newMenuMessageID = sentMsg.MessageID
			}
			return newMenuMessageID
		}
		if len(parts) == 1 {
			targetClientChatID, err := strconv.ParseInt(parts[0], 10, 64)
			if err == nil {
				log.Printf("[CALLBACK_INFO_COMMS] ЗАГЛУШКА: Оператор ChatID=%d пытается просмотреть историю чата с клиентом ChatID=%d.", chatID, targetClientChatID)
				// TODO: Реализовать SendChatHistoryMenu(operatorChatID, clientChatID, messageIDToEdit)
				// TODO: Implement SendChatHistoryMenu(operatorChatID, clientChatID, messageIDToEdit)
				sentMsg, errHelper = bh.sendInfoMessage(chatID, originalMessageID, fmt.Sprintf("Функция просмотра истории чата с клиентом %d еще не реализована.", targetClientChatID), "client_chats")
				if errHelper == nil && sentMsg.MessageID != 0 {
					newMenuMessageID = sentMsg.MessageID
				}
			} else {
				log.Printf("[CALLBACK_INFO_COMMS] Ошибка парсинга ClientChatID для view_chat_history: '%s'. ChatID=%d", parts[0], chatID)
				sentMsg, errHelper = bh.sendErrorMessageHelper(chatID, originalMessageID, "Ошибка: неверный ID клиента.")
				if errHelper == nil && sentMsg.MessageID != 0 {
					newMenuMessageID = sentMsg.MessageID
				}
			}
		} else {
			log.Printf("[CALLBACK_INFO_COMMS] Некорректный формат для 'view_chat_history': %v. Ожидался ID клиента. ChatID=%d", parts, chatID)
			sentMsg, errHelper = bh.sendErrorMessageHelper(chatID, originalMessageID, "Ошибка: неверный формат команды.")
			if errHelper == nil && sentMsg.MessageID != 0 {
				newMenuMessageID = sentMsg.MessageID
			}
		}
	case "materials_soon":
		log.Printf("[CALLBACK_INFO_COMMS] Запрос информации о стройматериалах (скоро). ChatID=%d", chatID)
		bh.SendMaterialsSoonInfo(chatID, originalMessageID)
		newMenuMessageID = bh.Deps.SessionManager.GetTempOrder(chatID).CurrentMessageID
	case "subscribe_materials_updates":
		log.Printf("[CALLBACK_INFO_COMMS] Запрос подписки на обновления стройматериалов. ChatID=%d", chatID)
		errDb := db.AddSubscription(chatID, "materials") // "materials" - ключ сервиса / "materials" - service key
		if errDb != nil {
			log.Printf("[CALLBACK_INFO_COMMS] Ошибка БД при подписке ChatID=%d на 'materials': %v", chatID, errDb)
			sentMsg, errHelper = bh.sendErrorMessageHelper(chatID, originalMessageID, "❌ Не удалось оформить подписку.")
			if errHelper == nil && sentMsg.MessageID != 0 {
				newMenuMessageID = sentMsg.MessageID
			}
		} else {
			log.Printf("[CALLBACK_INFO_COMMS] Пользователь ChatID=%d успешно подписан на 'materials'.", chatID)
			bh.SendSubscriptionConfirmation(chatID, "Стройматериалы", originalMessageID)
			newMenuMessageID = bh.Deps.SessionManager.GetTempOrder(chatID).CurrentMessageID
		}
	case "referral_link":
		log.Printf("[CALLBACK_INFO_COMMS] Запрос реферальной ссылки. ChatID=%d", chatID)
		bh.SendReferralLink(chatID, originalMessageID)
		newMenuMessageID = bh.Deps.SessionManager.GetTempOrder(chatID).CurrentMessageID
	case "referral_qr":
		log.Printf("[CALLBACK_INFO_COMMS] Запрос QR-кода реферальной ссылки. ChatID=%d", chatID)
		bh.SendReferralQRCode(chatID, originalMessageID)
		// SendReferralQRCode управляет своим CurrentMessageID / SendReferralQRCode manages its own CurrentMessageID
		newMenuMessageID = bh.Deps.SessionManager.GetTempOrder(chatID).CurrentMessageID
	case "referral_my":
		log.Printf("[CALLBACK_INFO_COMMS] Запрос списка 'Мои рефералы'. ChatID=%d", chatID)
		bh.SendMyReferralsMenu(chatID, originalMessageID)
		newMenuMessageID = bh.Deps.SessionManager.GetTempOrder(chatID).CurrentMessageID
	case "referral_details": // parts: [REFERRAL_ID]
		if len(parts) == 1 {
			referralID, err := strconv.ParseInt(parts[0], 10, 64)
			if err == nil {
				log.Printf("[CALLBACK_INFO_COMMS] Запрос деталей реферала #%d. ChatID=%d", referralID, chatID)
				bh.SendReferralDetails(chatID, referralID, originalMessageID)
				newMenuMessageID = bh.Deps.SessionManager.GetTempOrder(chatID).CurrentMessageID
			} else {
				log.Printf("[CALLBACK_INFO_COMMS] Ошибка конвертации ReferralID для 'referral_details': '%s'. ChatID=%d", parts[0], chatID)
				sentMsg, errHelper = bh.sendErrorMessageHelper(chatID, originalMessageID, "Ошибка: неверный ID реферала.")
				if errHelper == nil && sentMsg.MessageID != 0 {
					newMenuMessageID = sentMsg.MessageID
				}
			}
		} else {
			log.Printf("[CALLBACK_INFO_COMMS] Некорректный формат для 'referral_details': %v. Ожидался ID реферала. ChatID=%d", parts, chatID)
			sentMsg, errHelper = bh.sendErrorMessageHelper(chatID, originalMessageID, "Ошибка: неверный формат команды.")
			if errHelper == nil && sentMsg.MessageID != 0 {
				newMenuMessageID = sentMsg.MessageID
			}
		}
	case "request_referral_payout":
		log.Printf("[CALLBACK_INFO_COMMS] Запрос на выплату реферальных бонусов. ChatID=%d", chatID)
		bh.handleRequestReferralPayout(chatID, user, originalMessageID)
		newMenuMessageID = bh.Deps.SessionManager.GetTempOrder(chatID).CurrentMessageID
	case "phone_action_request_call", "phone_action_call_self":
		actionKey := strings.TrimPrefix(currentCommand, "phone_action_")

		if actionKey == "request_call" {
			log.Printf("[CALLBACK_INFO_COMMS] Действие по телефону: 'request_call'. ChatID=%d", chatID)
			bh.SendRequestPhoneNumberPrompt(chatID, originalMessageID)
			newMenuMessageID = bh.Deps.SessionManager.GetTempOrder(chatID).CurrentMessageID
		} else if actionKey == "call_self" {
			log.Printf("[CALLBACK_INFO_COMMS] Действие по телефону: 'call_self'. ChatID=%d", chatID)
			bh.SendOperatorContactInfo(chatID, originalMessageID)
			newMenuMessageID = bh.Deps.SessionManager.GetTempOrder(chatID).CurrentMessageID
		} else {
			log.Printf("[CALLBACK_INFO_COMMS] Неизвестное действие для 'phone_action'. CurrentCommand: %s, Parts: %v, Original Data: %s, ChatID=%d", currentCommand, parts, data, chatID)
			sentMsg, errHelper = bh.sendErrorMessageHelper(chatID, originalMessageID, "Ошибка: неверное телефонное действие.")
			if errHelper == nil && sentMsg.MessageID != 0 {
				newMenuMessageID = sentMsg.MessageID
			}
		}
	default:
		log.Printf("[CALLBACK_INFO_COMMS] ОШИБКА: Неизвестная команда '%s' передана в dispatchInfoCommsCallbacks. Parts: %v, Data: '%s', ChatID=%d", currentCommand, parts, data, chatID)
		sentMsg, errHelper = bh.sendErrorMessageHelper(chatID, originalMessageID, "Неизвестная команда (инфо).")
		if errHelper == nil && sentMsg.MessageID != 0 {
			newMenuMessageID = sentMsg.MessageID
		}
	}
	log.Printf("[CALLBACK_INFO_COMMS] Диспетчер информации/связи завершен. Команда='%s', ChatID=%d, ID нового меню=%d", currentCommand, chatID, newMenuMessageID)
	return newMenuMessageID
}

// handleRequestReferralPayout обрабатывает запрос на выплату реферальных бонусов.
// handleRequestReferralPayout processes a referral bonus payout request.
func (bh *BotHandler) handleRequestReferralPayout(chatID int64, user models.User, originalMessageID int) {
	log.Printf("[REFERRAL_HANDLER] Обработка запроса на выплату реферальных бонусов. ChatID=%d", chatID)
	var sentMsg tgbotapi.Message
	var errHelper error

	referrals, err := db.GetReferralsByInviterChatID(chatID)
	if err != nil {
		log.Printf("[REFERRAL_HANDLER] Ошибка БД при получении рефералов для выплаты ChatID=%d: %v", chatID, err)
		_, _ = bh.sendErrorMessageHelper(chatID, originalMessageID, "Ошибка получения данных о ваших бонусах.")
		bh.SendMyReferralsMenu(chatID, originalMessageID) // Возврат в меню рефералов / Return to referrals menu
		return
	}

	totalUnpaidBonus := 0.0
	var unpaidReferralIDs []int64
	for _, r := range referrals {
		// Бонус доступен к выплате, если он не выплачен И не находится уже в другом запросе на выплату
		// Bonus is available for payout if it's not paid AND not already in another payout request
		if !r.PaidOut && !r.PayoutRequestID.Valid {
			totalUnpaidBonus += r.Amount
			unpaidReferralIDs = append(unpaidReferralIDs, r.ID)
		}
	}

	if totalUnpaidBonus <= 0 {
		log.Printf("[REFERRAL_HANDLER] Сумма невыплаченных и не запрошенных бонусов для ChatID=%d равна нулю или меньше.", chatID)
		// Отправляем сообщение и затем меню рефералов / Send message and then referrals menu
		sentMsg, errHelper = bh.sendOrEditMessageHelper(chatID, originalMessageID, "У вас нет доступных бонусов для запроса выплаты.", nil, "")
		currentMenuID := originalMessageID
		if errHelper == nil && sentMsg.MessageID != 0 {
			currentMenuID = sentMsg.MessageID
		}
		bh.SendMyReferralsMenu(chatID, currentMenuID)
		return
	}

	payoutRequest := models.ReferralPayoutRequest{
		UserChatID:  chatID,
		Amount:      totalUnpaidBonus,
		Status:      constants.PAYOUT_REQUEST_STATUS_PENDING, // Начальный статус / Initial status
		RequestedAt: time.Now(),
		ReferralIDs: unpaidReferralIDs, // ID рефералов, включенных в этот запрос / IDs of referrals included in this request
	}
	requestID, err := db.CreateReferralPayoutRequest(payoutRequest)
	if err != nil {
		log.Printf("[REFERRAL_HANDLER] Ошибка БД при создании запроса на выплату для ChatID=%d: %v", chatID, err)
		_, _ = bh.sendErrorMessageHelper(chatID, originalMessageID, "Не удалось создать запрос на выплату. Попробуйте позже.")
		return
	}

	// Уведомляем администраторов/бухгалтерию о новом запросе
	// Notify administrators/accounting about the new request
	adminMessage := fmt.Sprintf("💸 Новый запрос на выплату реферальных бонусов!\nID Запроса: *%d*\nПользователь: %s (ChatID: `%d`)\nСумма: *%.0f ₽*",
		requestID, utils.GetUserDisplayName(user), chatID, totalUnpaidBonus)

	bh.NotifyAdminsPayoutRequest(adminMessage, requestID) // Реализация этой функции ниже / Implementation of this function below
	bh.SendReferralPayoutConfirmation(chatID, originalMessageID, totalUnpaidBonus, requestID)
}

// NotifyAdminsPayoutRequest уведомляет администраторов и/или бухгалтерию о новом запросе на выплату.
// NotifyAdminsPayoutRequest notifies administrators and/or accounting about a new payout request.
func (bh *BotHandler) NotifyAdminsPayoutRequest(message string, requestID int64) {
	log.Printf("[NOTIFY_ADMINS_PAYOUT] Уведомление о запросе на выплату #%d: %s", requestID, message)

	// Получаем список администраторов (MainOperator, Owner)
	// Get a list of administrators (MainOperator, Owner)
	admins, err := db.GetUsersByRole(constants.ROLE_MAINOPERATOR, constants.ROLE_OWNER)
	if err != nil {
		log.Printf("NotifyAdminsPayoutRequest: ошибка получения списка администраторов: %v", err)
	} else {
		for _, admin := range admins {
			// Можно добавить кнопку для быстрого перехода к управлению этим запросом
			// Can add a button for quick navigation to manage this request
			// keyboard := tgbotapi.NewInlineKeyboardMarkup(
			// 	tgbotapi.NewInlineKeyboardRow(
			// 		tgbotapi.NewInlineKeyboardButtonData("处理请求", fmt.Sprintf("admin_view_payout_request_%d", requestID)),
			// 	),
			// )
			// msg := tgbotapi.NewMessage(admin.ChatID, message)
			// msg.ReplyMarkup = keyboard
			// bh.Deps.BotClient.Send(msg)
			bh.sendMessage(admin.ChatID, message) // Отправляем простое сообщение / Send a simple message
		}
	}

	// Отправляем в специальный чат бухгалтерии, если он настроен
	// Send to a special accounting chat if configured
	if bh.Deps.Config.AccountingChatID != 0 {
		// Аналогично, можно добавить кнопки / Similarly, buttons can be added
		bh.sendMessage(bh.Deps.Config.AccountingChatID, message)
	}

	// Также можно отправить в общую группу, если это релевантно
	// Can also send to the common group if relevant
	if bh.Deps.Config.GroupChatID != 0 && bh.Deps.Config.GroupChatID != bh.Deps.Config.AccountingChatID {
		bh.sendMessage(bh.Deps.Config.GroupChatID, message)
	}
}
