// Файл: internal/handlers/payment_handler.go
package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"

	"Original/internal/constants"
	"Original/internal/db"
	"Original/internal/payments"
	"Original/internal/utils"
)

// HandleYooKassaNotification обрабатывает входящие вебхуки от ЮKassa.
func (bh *BotHandler) HandleYooKassaNotification(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		log.Printf("[YOOKASSA_HANDLER] Получен не-POST запрос: %s", r.Method)
		http.Error(w, "Only POST method is accepted", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("[YOOKASSA_HANDLER] Ошибка чтения тела запроса: %v", err)
		http.Error(w, "Cannot read request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	log.Printf("[YOOKASSA_HANDLER] Получено уведомление: %s", string(body))

	var notification payments.YooKassaNotification
	if err := json.Unmarshal(body, &notification); err != nil {
		log.Printf("[YOOKASSA_HANDLER] Ошибка демаршалинга JSON: %v", err)
		http.Error(w, "Invalid JSON format", http.StatusBadRequest)
		return
	}

	// Проверяем, что это уведомление об успешном платеже
	if notification.Event == "payment.succeeded" {
		log.Printf("[YOOKASSA_HANDLER] Обработка успешного платежа. PaymentID: %s", notification.Object.ID)

		// Извлекаем ID заказа из метаданных
		var metadata map[string]string
		if err := json.Unmarshal(notification.Object.Metadata, &metadata); err != nil {
			log.Printf("[YOOKASSA_HANDLER] Ошибка парсинга метаданных для PaymentID %s: %v", notification.Object.ID, err)
			// Отвечаем 200 OK, чтобы ЮKassa не повторяла запрос
			w.WriteHeader(http.StatusOK)
			return
		}

		orderIDStr, ok := metadata["order_id"]
		if !ok {
			log.Printf("[YOOKASSA_HANDLER] В метаданных для PaymentID %s отсутствует order_id.", notification.Object.ID)
			w.WriteHeader(http.StatusOK)
			return
		}

		orderID, err := strconv.ParseInt(orderIDStr, 10, 64)
		if err != nil {
			log.Printf("[YOOKASSA_HANDLER] Неверный формат order_id '%s' в метаданных для PaymentID %s.", orderIDStr, notification.Object.ID)
			w.WriteHeader(http.StatusOK)
			return
		}

		// Получаем заказ из БД
		order, err := db.GetOrderByID(int(orderID))
		if err != nil {
			log.Printf("[YOOKASSA_HANDLER] Ошибка получения заказа #%d из БД: %v", orderID, err)
			w.WriteHeader(http.StatusOK)
			return
		}

		// Проверяем, что заказ ожидает оплаты, чтобы избежать повторной обработки
		if order.Status == constants.STATUS_AWAITING_PAYMENT {
			// Обновляем статус заказа
			errUpdate := db.UpdateOrderStatus(orderID, constants.STATUS_INPROGRESS)
			if errUpdate != nil {
				log.Printf("[YOOKASSA_HANDLER] Ошибка обновления статуса заказа #%d на IN_PROGRESS: %v", orderID, errUpdate)
				// Здесь можно было бы вернуть 500, чтобы ЮKassa попробовала снова
				http.Error(w, "Failed to update order status", http.StatusInternalServerError)
				return
			}

			log.Printf("[YOOKASSA_HANDLER] Статус заказа #%d успешно обновлен на 'in_progress'.", orderID)

			// Уведомляем клиента
			clientMsg := fmt.Sprintf("✅ Оплата по заказу №%d прошла успешно! Ваш заказ принят в работу. Скоро с вами свяжутся исполнители.", orderID)
			bh.sendMessage(order.UserChatID, clientMsg)

			// Уведомляем операторов
			client, _ := db.GetUserByChatID(order.UserChatID)
			operatorMsg := fmt.Sprintf(
				"💸 Получена оплата по заказу №%d от клиента %s.\n"+
					"Статус заказа изменен на '%s'. Можно назначать исполнителей, если они еще не назначены.",
				orderID,
				utils.GetUserDisplayName(client),
				constants.StatusDisplayMap[constants.STATUS_INPROGRESS],
			)
			bh.NotifyOperatorsAndGroup(operatorMsg)

		} else {
			log.Printf("[YOOKASSA_HANDLER] Получено уведомление об оплате для заказа #%d, который уже не в статусе 'ожидание оплаты' (текущий статус: %s). Игнорируется.", orderID, order.Status)
		}
	}

	// Отвечаем ЮKassa, что все получили и обработали
	w.WriteHeader(http.StatusOK)
}
