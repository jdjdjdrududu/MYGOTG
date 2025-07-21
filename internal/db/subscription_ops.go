package db

import (
	"database/sql"
	"fmt"
	"log"
	// "Original/internal/models" // Если у вас будет модель Subscription / If you have a Subscription model
)

// AddSubscription добавляет подписку для пользователя.
// userChatID - chat_id пользователя, service - название сервиса.
// AddSubscription adds a subscription for a user.
// userChatID - user's chat_id, service - service name.
func AddSubscription(userChatID int64, service string) error {
	var userID int
	err := DB.QueryRow("SELECT id FROM users WHERE chat_id = $1", userChatID).Scan(&userID)
	if err != nil {
		if err == sql.ErrNoRows {
			// --- MODIFICATION FOR POINT 8 ---
			log.Printf("AddSubscription: Пользователь с chat_id %d не найден. Невозможно добавить подписку на сервис '%s'.", userChatID, service) // More detailed log
			// --- END MODIFICATION FOR POINT 8 ---
			return fmt.Errorf("пользователь с chat_id %d не найден для добавления подписки", userChatID)
		}
		log.Printf("AddSubscription: ошибка получения user_id для chat_id %d: %v", userChatID, err)
		return err
	}

	query := `
        INSERT INTO subscriptions (user_id, service, created_at, updated_at)
        VALUES ($1, $2, NOW(), NOW())
        ON CONFLICT (user_id, service) DO NOTHING` // Пример обработки конфликта, если подписка уже есть / Example of conflict handling if subscription already exists
	// --- MODIFICATION FOR POINT 8 ---
	result, errExec := DB.Exec(query, userID, service) // Changed to get result
	if errExec != nil {
		log.Printf("AddSubscription: ошибка DB.Exec при добавлении подписки на '%s' для user_id %d (chat_id %d): %v", service, userID, userChatID, errExec) // More detailed log
		return errExec
	}
	rowsAffected, _ := result.RowsAffected()
	if rowsAffected > 0 {
		log.Printf("Подписка на '%s' для user_id %d (chat_id %d) успешно добавлена.", service, userID, userChatID)
	} else {
		// This case means ON CONFLICT DO NOTHING was triggered, or some other non-error condition where no rows were inserted.
		log.Printf("Подписка на '%s' для user_id %d (chat_id %d) уже существовала или не была добавлена по другой причине (0 строк затронуто).", service, userID, userChatID)
	}
	// --- END MODIFICATION FOR POINT 8 ---
	return nil
}

// HasSubscription проверяет, есть ли у пользователя подписка на сервис.
// userChatID - chat_id пользователя, service - название сервиса.
// HasSubscription checks if a user has a subscription to a service.
// userChatID - user's chat_id, service - service name.
func HasSubscription(userChatID int64, service string) (bool, error) {
	var userID int
	err := DB.QueryRow("SELECT id FROM users WHERE chat_id = $1", userChatID).Scan(&userID)
	if err != nil {
		if err == sql.ErrNoRows { // Если пользователя нет, то и подписки нет / If user doesn't exist, no subscription either
			return false, nil
		}
		log.Printf("HasSubscription: ошибка получения user_id для chat_id %d: %v", userChatID, err)
		return false, err
	}

	var exists bool
	query := `SELECT EXISTS(SELECT 1 FROM subscriptions WHERE user_id=$1 AND service=$2)`
	err = DB.QueryRow(query, userID, service).Scan(&exists)
	if err != nil {
		log.Printf("HasSubscription: ошибка проверки подписки на '%s' для user_id %d: %v", service, userID, err)
		return false, err
	}
	return exists, nil
}

// RemoveSubscription удаляет подписку пользователя на сервис.
// userChatID - chat_id пользователя, service - название сервиса.
// RemoveSubscription removes a user's subscription to a service.
// userChatID - user's chat_id, service - service name.
func RemoveSubscription(userChatID int64, service string) error {
	var userID int
	err := DB.QueryRow("SELECT id FROM users WHERE chat_id = $1", userChatID).Scan(&userID)
	if err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("пользователь с chat_id %d не найден для удаления подписки", userChatID)
		}
		log.Printf("RemoveSubscription: ошибка получения user_id для chat_id %d: %v", userChatID, err)
		return err
	}

	result, err := DB.Exec("DELETE FROM subscriptions WHERE user_id=$1 AND service=$2", userID, service)
	if err != nil {
		log.Printf("RemoveSubscription: ошибка удаления подписки на '%s' для user_id %d: %v", service, userID, err)
		return err
	}
	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		log.Printf("Подписка на '%s' для user_id %d не найдена для удаления.", service, userID)
		// Можно вернуть ошибку fmt.Errorf(...) если это считается проблемой
		// Can return fmt.Errorf(...) if this is considered a problem
	} else {
		log.Printf("Подписка на '%s' для user_id %d успешно удалена.", service, userID)
	}
	return nil
}
