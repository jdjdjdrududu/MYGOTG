package db

import (
	"database/sql"
	"fmt"
	"log"
	// "Original/internal/models" // Если у вас будет модель Review
)

// AddReview добавляет отзыв от пользователя.
// userChatID - chat_id пользователя, reviewText - текст отзыва.
func AddReview(userChatID int64, reviewText string) (int64, error) {
	var userID int
	err := DB.QueryRow("SELECT id FROM users WHERE chat_id = $1", userChatID).Scan(&userID)
	if err != nil {
		if err == sql.ErrNoRows {
			return 0, fmt.Errorf("пользователь с chat_id %d не найден для добавления отзыва", userChatID)
		}
		log.Printf("AddReview: ошибка получения user_id для chat_id %d: %v", userChatID, err)
		return 0, err
	}

	var id int64
	query := `
        INSERT INTO reviews (user_id, review, created_at)
        VALUES ($1, $2, NOW())
        RETURNING id`
	err = DB.QueryRow(query, userID, reviewText).Scan(&id)
	if err != nil {
		log.Printf("AddReview: ошибка добавления отзыва от user_id %d: %v", userID, err)
		return 0, err
	}
	log.Printf("Отзыв #%d от user_id %d (chat_id %d) успешно добавлен.", id, userID, userChatID)
	return id, nil
}

// GetReviewsByUserID извлекает все отзывы, оставленные пользователем.
// userChatID - chat_id пользователя.
func GetReviewsByUserID(userChatID int64) ([]string, error) { // Пример возврата только текстов отзывов
	var userID int
	err := DB.QueryRow("SELECT id FROM users WHERE chat_id = $1", userChatID).Scan(&userID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("пользователь с chat_id %d не найден", userChatID)
		}
		log.Printf("GetReviewsByUserID: ошибка получения user_id для chat_id %d: %v", userChatID, err)
		return nil, err
	}

	rows, err := DB.Query("SELECT review FROM reviews WHERE user_id=$1 ORDER BY created_at DESC", userID)
	if err != nil {
		log.Printf("GetReviewsByUserID: ошибка получения отзывов для user_id %d: %v", userID, err)
		return nil, err
	}
	defer rows.Close()

	var reviews []string
	for rows.Next() {
		var reviewText string
		if errScan := rows.Scan(&reviewText); errScan != nil {
			log.Printf("GetReviewsByUserID: ошибка сканирования отзыва для user_id %d: %v", userID, errScan)
			continue
		}
		reviews = append(reviews, reviewText)
	}
	if err = rows.Err(); err != nil {
		log.Printf("GetReviewsByUserID: ошибка после итерации по строкам для user_id %d: %v", userID, err)
		return nil, err
	}
	return reviews, nil
}
