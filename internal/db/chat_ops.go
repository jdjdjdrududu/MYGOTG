package db

import (
	"Original/internal/constants"
	"Original/internal/models" // Используем Original как имя модуля / Use Original as module name
	"database/sql"
	"fmt"
	"log"
	// "Original/internal/constants" // Не используется напрямую здесь / Not used directly here
)

// AddChatMessage добавляет новое сообщение в чат.
// userChatID - chat_id клиента, operatorChatID - chat_id оператора (или ID группы).
// AddChatMessage adds a new message to the chat.
// userChatID - client's chat_id, operatorChatID - operator's chat_id (or group ID).
func AddChatMessage(userChatID, operatorChatID int64, message string, isFromUser bool, conversationID string) (int64, error) {
	var userID, operatorUserID sql.NullInt64 // Используем sql.NullInt64 для operator_id, так как он может быть не пользователем / Use sql.NullInt64 for operator_id as it might not be a user

	// Получаем users.id для userChatID (клиента) / Get users.id for userChatID (client)
	err := DB.QueryRow("SELECT id FROM users WHERE chat_id = $1", userChatID).Scan(&userID)
	if err != nil {
		// Если клиент не найден, это может быть проблемой, но для записи сообщения можно продолжить с NULL userID,
		// если схема это позволяет и есть логика для таких случаев.
		// В текущей схеме user_id в chat_messages ссылается на users.id и не может быть NULL.
		// If client not found, this might be an issue, but for message logging, one could proceed with NULL userID
		// if the schema allows and there's logic for such cases.
		// In the current schema, user_id in chat_messages references users.id and cannot be NULL.
		log.Printf("AddChatMessage: не удалось найти user_id для chat_id %d (клиент): %v", userChatID, err)
		return 0, fmt.Errorf("пользователь-клиент (chat_id %d) не найден: %w", userChatID, err)
	}

	// Пытаемся получить users.id для operatorChatID, если это ID пользователя-оператора
	// Try to get users.id for operatorChatID if it's an operator user's ID
	var tempOperatorID int64
	err = DB.QueryRow("SELECT id FROM users WHERE chat_id = $1 AND role != $2", operatorChatID, constants.ROLE_USER).Scan(&tempOperatorID)
	if err == nil {
		operatorUserID = sql.NullInt64{Int64: tempOperatorID, Valid: true}
	} else if err == sql.ErrNoRows {
		// Если operatorChatID не соответствует пользователю-оператору, operator_id будет NULL.
		// Это может быть ID группы или системного чата.
		// If operatorChatID does not correspond to an operator user, operator_id will be NULL.
		// This could be a group ID or system chat.
		log.Printf("AddChatMessage: operatorChatID %d не найден как пользователь-оператор, operator_id будет NULL.", operatorChatID)
		operatorUserID = sql.NullInt64{Valid: false}
	} else {
		log.Printf("AddChatMessage: ошибка получения operator_id для chat_id %d: %v", operatorChatID, err)
		// Не прерываем, operator_id будет NULL / Do not interrupt, operator_id will be NULL
		operatorUserID = sql.NullInt64{Valid: false}
	}

	var id int64
	query := `
        INSERT INTO chat_messages (user_id, operator_id, message, is_from_user, conversation_id, created_at)
        VALUES ($1, $2, $3, $4, $5, NOW())
        RETURNING id`

	err = DB.QueryRow(query, userID, operatorUserID, message, isFromUser, conversationID).Scan(&id)
	if err != nil {
		log.Printf("AddChatMessage: ошибка добавления сообщения в чат (user_id %v, operator_id %v, conv_id %s): %v", userID, operatorUserID, conversationID, err)
		return 0, err
	}
	log.Printf("Сообщение #%d в чате (conv_id %s) успешно добавлено.", id, conversationID)
	return id, nil
}

// GetChatMessagesByConversationID извлекает все сообщения для указанного ID беседы.
// GetChatMessagesByConversationID retrieves all messages for the specified conversation ID.
func GetChatMessagesByConversationID(conversationID string) ([]models.ChatMessage, error) {
	rows, err := DB.Query(`
        SELECT cm.id, cm.user_id, u.chat_id AS user_chat_id, u.first_name AS user_first_name, 
               cm.operator_id, op.chat_id AS operator_chat_id, op.first_name AS operator_first_name,
               cm.message, cm.is_from_user, cm.created_at
        FROM chat_messages cm
        JOIN users u ON cm.user_id = u.id
        LEFT JOIN users op ON cm.operator_id = op.id -- LEFT JOIN, так как operator_id может быть NULL / LEFT JOIN as operator_id can be NULL
        WHERE cm.conversation_id = $1
        ORDER BY cm.created_at ASC`, conversationID)
	if err != nil {
		log.Printf("GetChatMessagesByConversationID: ошибка получения сообщений для conv_id %s: %v", conversationID, err)
		return nil, err
	}
	defer rows.Close()

	var messages []models.ChatMessage
	for rows.Next() {
		var msg models.ChatMessage
		var userChatID sql.NullInt64         // Для чтения user_chat_id
		var userFirstName sql.NullString     // Для чтения user_first_name
		var operatorChatID sql.NullInt64     // Для чтения operator_chat_id
		var operatorFirstName sql.NullString // Для чтения operator_first_name

		errScan := rows.Scan(
			&msg.ID,
			&msg.UserID, // Это users.id клиента / This is client's users.id
			&userChatID,
			&userFirstName,
			&msg.OperatorID, // Это users.id оператора или NULL / This is operator's users.id or NULL
			&operatorChatID,
			&operatorFirstName,
			&msg.Message,
			&msg.IsFromUser,
			&msg.CreatedAt, // Сканируем в time.Time / Scan into time.Time
		)
		if errScan != nil {
			log.Printf("GetChatMessagesByConversationID: ошибка сканирования сообщения для conv_id %s: %v", conversationID, errScan)
			continue
		}
		// Заполняем дополнительные поля для отображения, если они нужны в модели ChatMessage
		// Populate additional fields for display if needed in ChatMessage model
		// msg.UserChatID = userChatID.Int64 (если поле есть в модели)
		// msg.UserFirstName = userFirstName.String (если поле есть в модели)
		// msg.OperatorChatID = operatorChatID.Int64 (если поле есть в модели)
		// msg.OperatorFirstName = operatorFirstName.String (если поле есть в модели)
		messages = append(messages, msg)
	}
	if err = rows.Err(); err != nil {
		log.Printf("GetChatMessagesByConversationID: ошибка после итерации по строкам для conv_id %s: %v", conversationID, err)
		return nil, err
	}
	return messages, nil
}

// GetActiveClientChats получает список уникальных чатов (пользователей), которые писали сообщения.
// GetActiveClientChats retrieves a list of unique chats (users) who have sent messages.
func GetActiveClientChats() ([]models.User, error) {
	// Запрос выбирает пользователей, от которых были сообщения (is_from_user=TRUE),
	// и сортирует их по времени последнего сообщения.
	// The query selects users from whom messages were sent (is_from_user=TRUE),
	// and sorts them by the time of the last message.
	rows, err := DB.Query(`
        SELECT DISTINCT u.id, u.chat_id, u.first_name, u.last_name, u.nickname, MAX(cm.created_at) as last_message_time
        FROM users u
        JOIN chat_messages cm ON u.id = cm.user_id
        WHERE cm.is_from_user = TRUE
        GROUP BY u.id, u.chat_id, u.first_name, u.last_name, u.nickname
        ORDER BY last_message_time DESC`)
	if err != nil {
		log.Printf("GetActiveClientChats: ошибка получения активных чатов клиентов: %v", err)
		return nil, err
	}
	defer rows.Close()

	var users []models.User
	for rows.Next() {
		var u models.User
		var lastMessageTime sql.NullTime // Это поле не используется в модели User, но нужно для сканирования / This field is not used in User model, but needed for scanning
		errScan := rows.Scan(&u.ID, &u.ChatID, &u.FirstName, &u.LastName, &u.Nickname, &lastMessageTime)
		if errScan != nil {
			log.Printf("GetActiveClientChats: ошибка сканирования пользователя: %v", errScan)
			continue
		}
		users = append(users, u)
	}
	if err = rows.Err(); err != nil {
		log.Printf("GetActiveClientChats: ошибка после итерации по строкам: %v", err)
		return nil, err
	}
	return users, nil
}

// GetUserChatIDFromConversationID получает chat_id пользователя (клиента) из сообщения чата по ID беседы.
// GetUserChatIDFromConversationID retrieves the user's (client's) chat_id from a chat message by conversation ID.
func GetUserChatIDFromConversationID(conversationID string) (int64, error) {
	var userChatID int64
	err := DB.QueryRow(`
        SELECT u.chat_id
        FROM chat_messages cm
        JOIN users u ON cm.user_id = u.id
        WHERE cm.conversation_id = $1 AND cm.is_from_user = TRUE -- Ищем сообщение от пользователя / Look for message from user
        ORDER BY cm.created_at ASC -- Берем самое первое сообщение от пользователя в этой беседе / Take the very first message from the user in this conversation
        LIMIT 1`, conversationID).Scan(&userChatID)
	if err != nil {
		if err == sql.ErrNoRows {
			// Это может означать, что беседа была инициирована оператором, или сообщений еще нет
			// This might mean the conversation was initiated by an operator, or there are no messages yet
			log.Printf("GetUserChatIDFromConversationID: не найдено сообщений от пользователя для беседы ID: %s", conversationID)
			return 0, fmt.Errorf("не найдено сообщений от пользователя для беседы ID: %s", conversationID)
		}
		log.Printf("GetUserChatIDFromConversationID: ошибка получения chat_id пользователя для беседы %s: %v", conversationID, err)
		return 0, err
	}
	return userChatID, nil
}
