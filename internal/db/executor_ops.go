package db

import (
	"database/sql"
	"fmt"
	"log"
	"strings"

	"Original/internal/constants"
	"Original/internal/models" // Используем Original как имя модуля / Use Original as module name
)

// AssignExecutor назначает исполнителя на заказ.
// executorChatID - это chat_id исполнителя.
// Устанавливает is_notified в FALSE по умолчанию.
func AssignExecutor(orderID int, executorChatID int64, role string) error {
	// Сначала получаем users.id по executorChatID
	var executorUserID int
	err := DB.QueryRow("SELECT id FROM users WHERE chat_id = $1", executorChatID).Scan(&executorUserID)
	if err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("исполнитель с chat_id %d не найден", executorChatID)
		}
		log.Printf("AssignExecutor: ошибка получения user_id для исполнителя chat_id %d: %v", executorChatID, err)
		return err
	}

	// Проверка, занят ли исполнитель
	var orderTime sql.NullString
	var orderDateString string
	err = DB.QueryRow("SELECT time, to_char(date, 'YYYY-MM-DD') FROM orders WHERE id=$1", orderID).Scan(&orderTime, &orderDateString)
	if err != nil {
		log.Printf("AssignExecutor: ошибка получения времени/даты заказа #%d: %v", orderID, err)
		// Не прерываем, но это может повлиять на корректность проверки занятости
	}

	if orderDateString != "" {
		var count int
		checkQuery := `
            SELECT COUNT(*)
            FROM executors e
            JOIN orders o ON o.id = e.order_id
            WHERE e.user_id = $1
              AND o.date = $2::date
              AND (
                    ($3::TEXT IS NULL AND o.time IS NULL) OR 
                    (o.time = $3::TEXT)                    
                  )
              AND o.status = $4 AND o.id != $5`

		var timeArg sql.NullString
		if orderTime.Valid && orderTime.String != "" && strings.ToLower(orderTime.String) != "в ближайшее время" && strings.ToLower(orderTime.String) != "срочно" {
			timeArg = sql.NullString{String: orderTime.String, Valid: true}
		} else {
			timeArg = sql.NullString{Valid: false}
		}

		err = DB.QueryRow(checkQuery, executorUserID, orderDateString, timeArg, constants.STATUS_INPROGRESS, orderID).Scan(&count)
		if err != nil {
			log.Printf("AssignExecutor: ошибка проверки занятости исполнителя user_id %d: %v", executorUserID, err)
			return fmt.Errorf("ошибка проверки занятости: %v", err)
		}
		if count > 0 {
			log.Printf("AssignExecutor: Исполнитель user_id %d занят на %s. Время заказа: %v (передано как: %v)", executorUserID, orderDateString, orderTime, timeArg)
			return fmt.Errorf("исполнитель занят на это время")
		}
	}

	// Назначение исполнителя с is_notified = FALSE
	_, err = DB.Exec(`
        INSERT INTO executors (order_id, user_id, role, is_notified, created_at, updated_at)
        VALUES ($1, $2, $3, FALSE, NOW(), NOW())
        ON CONFLICT (order_id, user_id, role) DO UPDATE SET
        is_notified = FALSE, updated_at = NOW()`, // Сбрасываем is_notified при повторном назначении
		orderID, executorUserID, role)
	if err != nil {
		log.Printf("AssignExecutor: ошибка назначения исполнителя user_id %d на заказ #%d: %v", executorUserID, orderID, err)
		return err
	}
	log.Printf("Исполнитель user_id %d (роль: %s, is_notified: FALSE) назначен на заказ #%d.", executorUserID, role, orderID)
	return nil
}

// ConfirmExecutorConfirmation подтверждает участие исполнителя в заказе.
// executorChatID - chat_id исполнителя.
// Эта функция подтверждает общее участие, а не статус "уведомлен".
func ConfirmExecutorConfirmation(orderID int, executorChatID int64) error {
	var executorUserID int
	err := DB.QueryRow("SELECT id FROM users WHERE chat_id = $1", executorChatID).Scan(&executorUserID)
	if err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("исполнитель с chat_id %d не найден для подтверждения", executorChatID)
		}
		log.Printf("ConfirmExecutorConfirmation: ошибка получения user_id для chat_id %d: %v", executorChatID, err)
		return err
	}

	result, err := DB.Exec(`
        UPDATE executors SET confirmed=TRUE, updated_at=NOW()
        WHERE order_id=$1 AND user_id=$2 AND confirmed=FALSE`,
		orderID, executorUserID)
	if err != nil {
		log.Printf("ConfirmExecutorConfirmation: ошибка подтверждения участия исполнителя user_id %d в заказе #%d: %v", executorUserID, orderID, err)
		return err
	}
	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		log.Printf("ConfirmExecutorConfirmation: не обновлено подтверждение для исполнителя user_id %d, заказ #%d (возможно, уже подтверждено или не назначен).", executorUserID, orderID)
		var exists bool
		_ = DB.QueryRow("SELECT EXISTS(SELECT 1 FROM executors WHERE order_id=$1 AND user_id=$2 AND confirmed=TRUE)", orderID, executorUserID).Scan(&exists)
		if exists {
			return fmt.Errorf("участие уже подтверждено")
		}
		return fmt.Errorf("назначение не найдено или уже подтверждено")

	}
	log.Printf("Участие исполнителя user_id %d в заказе #%d подтверждено.", executorUserID, orderID)
	return nil
}

// GetExecutorsByOrderID извлекает всех исполнителей, назначенных на заказ.
// Возвращает слайс models.Executor, включая поле IsNotified.
func GetExecutorsByOrderID(orderID int) ([]models.Executor, error) {
	rows, err := DB.Query(`
        SELECT e.user_id, e.role, e.confirmed, e.is_notified, u.chat_id, u.first_name, u.last_name, u.nickname
        FROM executors e
        JOIN users u ON e.user_id = u.id
        WHERE e.order_id=$1`, orderID)
	if err != nil {
		log.Printf("GetExecutorsByOrderID: ошибка получения исполнителей для заказа #%d: %v", orderID, err)
		return nil, err
	}
	defer rows.Close()

	var executors []models.Executor
	for rows.Next() {
		var ex models.Executor
		errScan := rows.Scan(&ex.UserID, &ex.Role, &ex.Confirmed, &ex.IsNotified, &ex.ChatID, &ex.FirstName, &ex.LastName, &ex.Nickname)
		if errScan != nil {
			log.Printf("GetExecutorsByOrderID: ошибка сканирования исполнителя для заказа #%d: %v", orderID, errScan)
			continue
		}
		executors = append(executors, ex)
	}
	if err = rows.Err(); err != nil {
		log.Printf("GetExecutorsByOrderID: ошибка после итерации по строкам для заказа #%d: %v", orderID, err)
		return nil, err
	}
	return executors, nil
}

// RemoveExecutor удаляет исполнителя с заказа.
// executorChatID - chat_id исполнителя.
func RemoveExecutor(orderID int, executorChatID int64) error {
	var executorUserID int
	err := DB.QueryRow("SELECT id FROM users WHERE chat_id = $1", executorChatID).Scan(&executorUserID)
	if err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("исполнитель с chat_id %d не найден для удаления с заказа", executorChatID)
		}
		log.Printf("RemoveExecutor: ошибка получения user_id для chat_id %d: %v", executorChatID, err)
		return err
	}

	result, err := DB.Exec("DELETE FROM executors WHERE order_id=$1 AND user_id=$2", orderID, executorUserID)
	if err != nil {
		log.Printf("RemoveExecutor: ошибка удаления исполнителя user_id %d с заказа #%d: %v", executorUserID, orderID, err)
		return err
	}
	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("исполнитель user_id %d не был назначен на заказ #%d или уже удален", executorUserID, orderID)
	}
	log.Printf("Исполнитель user_id %d удален с заказа #%d.", executorUserID, orderID)
	return nil
}

// MarkExecutorAsNotified помечает исполнителя как уведомленного по заказу.
func MarkExecutorAsNotified(orderID int, executorUserID int64) error {
	result, err := DB.Exec(`
        UPDATE executors SET is_notified=TRUE, updated_at=NOW()
        WHERE order_id=$1 AND user_id=$2 AND is_notified=FALSE`,
		orderID, executorUserID)
	if err != nil {
		log.Printf("MarkExecutorAsNotified: ошибка обновления статуса уведомления для исполнителя user_id %d, заказ #%d: %v", executorUserID, orderID, err)
		return err
	}
	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		// Это может означать, что исполнитель не найден, или уже уведомлен
		log.Printf("MarkExecutorAsNotified: статус уведомления не обновлен для исполнителя user_id %d, заказ #%d (возможно, уже уведомлен или не назначен).", executorUserID, orderID)
		// Чтобы не показывать ошибку пользователю, если он нажал кнопку повторно,
		// можно проверить, существует ли запись с is_notified = TRUE.
		var alreadyNotified bool
		checkErr := DB.QueryRow("SELECT EXISTS(SELECT 1 FROM executors WHERE order_id=$1 AND user_id=$2 AND is_notified=TRUE)", orderID, executorUserID).Scan(&alreadyNotified)
		if checkErr == nil && alreadyNotified {
			return nil // Уже уведомлен, не ошибка
		}
		return fmt.Errorf("назначение не найдено или исполнитель уже уведомлен")
	}
	log.Printf("Исполнитель user_id %d по заказу #%d отмечен как уведомленный.", executorUserID, orderID)
	return nil
}
