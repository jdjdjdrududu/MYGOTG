package db

import (
	"database/sql"
	"fmt"
	"log"

	"github.com/lib/pq" // Для pq.Array, если используется где-то еще в этом файле

	"Original/internal/constants"
	"Original/internal/models"
	"Original/internal/utils" // Для utils.EncryptCardNumber и utils.DecryptCardNumber
)

// RegisterUser регистрирует нового пользователя или обновляет существующего, если необходимо.
// RegisterUser registers a new user or updates an existing one if necessary.
func RegisterUser(chatID int64, firstName, lastName string) (models.User, error) {
	var user models.User
	var exists bool
	err := DB.QueryRow("SELECT EXISTS(SELECT 1 FROM users WHERE chat_id=$1)", chatID).Scan(&exists)
	if err != nil {
		log.Printf("RegisterUser: ошибка проверки существования пользователя chatID %d: %v", chatID, err)
		return user, err
	}

	if !exists {
		// card_number по умолчанию NULL при новой регистрации
		// card_number defaults to NULL for new registrations
		_, err = DB.Exec(`
            INSERT INTO users (chat_id, role, first_name, last_name, nickname, phone, card_number, main_menu_message_id, created_at, updated_at)
            VALUES ($1, $2, $3, $4, $5, $6, NULL, $7, NOW(), NOW())`, // card_number is NULL
			chatID, constants.ROLE_USER, firstName, lastName, sql.NullString{}, sql.NullString{}, 0)
		if err != nil {
			log.Printf("RegisterUser: ошибка вставки нового пользователя chatID %d: %v", chatID, err)
			return user, err
		}
		log.Printf("Зарегистрирован новый пользователь с chatID %d", chatID)
	}

	// После регистрации или если пользователь уже существует, получаем его данные
	// After registration or if the user already exists, retrieve their data
	return GetUserByChatID(chatID)
}

// GetUserByChatID извлекает пользователя по его chat_id.
// Возвращает models.User и ошибку, если пользователь не найден или произошла другая ошибка.
// GetUserByChatID retrieves a user by their chat_id.
// Returns models.User and an error if the user is not found or another error occurs.
func GetUserByChatID(chatID int64) (models.User, error) {
	var u models.User
	var encryptedCardNumber sql.NullString // Для чтения зашифрованного номера карты / For reading the encrypted card number

	err := DB.QueryRow(`
        SELECT id, chat_id, role, first_name, last_name, nickname, phone, card_number, 
               is_blocked, block_reason, block_date, COALESCE(main_menu_message_id, 0)
        FROM users WHERE chat_id=$1`, chatID).Scan(
		&u.ID, &u.ChatID, &u.Role, &u.FirstName, &u.LastName, &u.Nickname, &u.Phone, &encryptedCardNumber,
		&u.IsBlocked, &u.BlockReason, &u.BlockDate, &u.MainMenuMessageID)

	if err != nil {
		if err == sql.ErrNoRows {
			// log.Printf("GetUserByChatID: пользователь с chatID %d не найден в БД.", chatID) // Commented out to reduce log spam
			return u, err // Возвращаем ошибку sql.ErrNoRows, чтобы вызывающий код мог ее обработать / Return sql.ErrNoRows error so the calling code can handle it
		}
		log.Printf("GetUserByChatID: ошибка получения пользователя chatID %d: %v", chatID, err)
		return u, err
	}

	if encryptedCardNumber.Valid && encryptedCardNumber.String != "" {
		decryptedCard, errDecrypt := utils.DecryptCardNumber(encryptedCardNumber.String)
		if errDecrypt != nil {
			log.Printf("GetUserByChatID: ошибка дешифрования номера карты для chatID %d: %v. Будет возвращено пустое значение.", chatID, errDecrypt)
			u.CardNumber = sql.NullString{String: "", Valid: false}
		} else {
			u.CardNumber = sql.NullString{String: decryptedCard, Valid: true}
		}
	} else {
		u.CardNumber = sql.NullString{String: "", Valid: false} // Если в БД NULL или пусто / If NULL or empty in DB
	}

	// log.Printf("GetUserByChatID: пользователь chatID %d найден, роль: %s", chatID, u.Role) // Commented out to reduce log spam
	return u, nil
}

// UpdateUserRole обновляет роль пользователя.
// UpdateUserRole updates the user's role.
func UpdateUserRole(chatID int64, role string) error {
	_, err := DB.Exec("UPDATE users SET role=$1, updated_at=NOW() WHERE chat_id=$2", role, chatID)
	if err != nil {
		log.Printf("UpdateUserRole: ошибка обновления роли для chatID %d на %s: %v", chatID, role, err)
		return err
	}
	log.Printf("Роль пользователя chatID %d обновлена на %s", chatID, role)
	return nil
}

// UpdateUserMainMenuMessageID обновляет main_menu_message_id для пользователя.
// UpdateUserMainMenuMessageID updates the main_menu_message_id for the user.
func UpdateUserMainMenuMessageID(chatID int64, messageID int) error {
	_, err := DB.Exec("UPDATE users SET main_menu_message_id=$1 WHERE chat_id=$2", messageID, chatID)
	if err != nil {
		log.Printf("UpdateUserMainMenuMessageID: Ошибка сохранения main_menu_message_id %d для chatID %d: %v", messageID, chatID, err)
		return err
	}
	return nil
}

// ResetUserMainMenuMessageID сбрасывает main_menu_message_id для пользователя.
// ResetUserMainMenuMessageID resets the main_menu_message_id for the user.
func ResetUserMainMenuMessageID(chatID int64) error {
	_, err := DB.Exec("UPDATE users SET main_menu_message_id=0 WHERE chat_id=$1", chatID)
	if err != nil {
		log.Printf("ResetUserMainMenuMessageID: Ошибка сброса main_menu_message_id для chatID %d: %v", chatID, err)
		return err
	}
	return nil
}

// GetUserMainMenuMessageID получает main_menu_message_id для пользователя.
// GetUserMainMenuMessageID retrieves the main_menu_message_id for the user.
func GetUserMainMenuMessageID(chatID int64) (int, error) {
	var mainMenuMessageID sql.NullInt64 // Используем NullInt64 для корректной обработки NULL из БД / Use NullInt64 for correct handling of NULL from DB
	err := DB.QueryRow("SELECT main_menu_message_id FROM users WHERE chat_id=$1", chatID).Scan(&mainMenuMessageID)
	if err != nil {
		if err == sql.ErrNoRows {
			return 0, nil // Пользователь может не существовать или ID не установлен / User may not exist or ID not set
		}
		log.Printf("GetUserMainMenuMessageID: Ошибка получения main_menu_message_id для chatID %d: %v", chatID, err)
		return 0, err
	}
	if mainMenuMessageID.Valid {
		return int(mainMenuMessageID.Int64), nil
	}
	return 0, nil // Если NULL, возвращаем 0 / If NULL, return 0
}

// BlockUser блокирует пользователя с указанной причиной.
// BlockUser blocks a user with the specified reason.
func BlockUser(chatID int64, reason string) error {
	tx, err := DB.Begin()
	if err != nil {
		log.Printf("BlockUser: ошибка начала транзакции: %v", err)
		return err
	}
	// Используем defer с именованной переменной ошибки для корректного отката
	var opErr error
	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			panic(p) // Re-panic after Rollback
		} else if opErr != nil {
			tx.Rollback()
		} else {
			opErr = tx.Commit()
			if opErr != nil {
				log.Printf("BlockUser: ошибка коммита транзакции: %v", opErr)
			}
		}
	}()

	var userID int64
	err = tx.QueryRow("SELECT id FROM users WHERE chat_id = $1", chatID).Scan(&userID)
	if err != nil {
		if err == sql.ErrNoRows {
			opErr = fmt.Errorf("пользователь с chat_id %d не найден для блокировки", chatID)
			log.Printf("BlockUser: %v", opErr)
			return opErr
		}
		opErr = fmt.Errorf("ошибка получения userID для chat_id %d: %w", chatID, err)
		log.Printf("BlockUser: %v", opErr)
		return opErr
	}

	_, opErr = tx.Exec(`
        UPDATE users
        SET is_blocked=TRUE, block_reason=$1, block_date=NOW(), updated_at=NOW()
        WHERE id=$2`, reason, userID)
	if opErr != nil {
		log.Printf("BlockUser: ошибка блокировки пользователя ID %d (ChatID %d): %v", userID, chatID, opErr)
		return opErr
	}
	log.Printf("Пользователь ID %d (ChatID %d) заблокирован. Причина: %s", userID, chatID, reason)

	// --- НАЧАЛО ИЗМЕНЕНИЯ: Отмена активных заказов пользователя ---
	cancelReason := fmt.Sprintf("Пользователь (ID: %d, ChatID: %d) был заблокирован. Причина: %s", userID, chatID, reason)
	opErr = CancelUserActiveOrdersInTx(tx, userID, cancelReason)
	if opErr != nil {
		// Ошибка уже залогирована в CancelUserActiveOrdersInTx
		// Возвращаем ошибку, чтобы транзакция откатилась
		return opErr
	}
	// --- КОНЕЦ ИЗМЕНЕНИЯ ---

	return opErr // Если opErr == nil, defer вызовет Commit
}

// UnblockUser разблокирует пользователя.
// UnblockUser unblocks a user.
func UnblockUser(chatID int64) error {
	_, err := DB.Exec(`
        UPDATE users
        SET is_blocked=FALSE, block_reason=NULL, block_date=NULL, updated_at=NOW()
        WHERE chat_id=$1`, chatID)
	if err != nil {
		log.Printf("UnblockUser: ошибка разблокировки пользователя %d: %v", chatID, err)
		return err
	}
	log.Printf("Пользователь %d разблокирован.", chatID)
	return nil
}

// UpdateUserPhone обновляет номер телефона пользователя.
// UpdateUserPhone updates the user's phone number.
func UpdateUserPhone(chatID int64, phone string) error {
	_, err := DB.Exec("UPDATE users SET phone=$1, updated_at=NOW() WHERE chat_id=$2", phone, chatID)
	if err != nil {
		log.Printf("UpdateUserPhone: ошибка сохранения номера телефона %s для chatID %d: %v", phone, chatID, err)
		return err
	}
	log.Printf("Номер телефона для chatID %d обновлен на %s", chatID, phone)
	return nil
}

// CheckPhoneNumberExists проверяет, используется ли номер телефона другим пользователем.
// Возвращает chat_id существующего пользователя или 0, если номер не найден или ошибка.
// CheckPhoneNumberExists checks if a phone number is used by another user.
// Returns the chat_id of the existing user or 0 if the number is not found or an error occurs.
func CheckPhoneNumberExists(phone string, excludeChatID int64) (int64, error) {
	var existingChatID int64
	err := DB.QueryRow("SELECT chat_id FROM users WHERE phone=$1 AND chat_id != $2", phone, excludeChatID).Scan(&existingChatID)
	if err != nil {
		if err == sql.ErrNoRows {
			return 0, nil // Номер не используется другим пользователем / Number is not used by another user
		}
		log.Printf("CheckPhoneNumberExists: ошибка проверки номера телефона %s (исключая %d): %v", phone, excludeChatID, err)
		return 0, err
	}
	return existingChatID, nil // Номер используется existingChatID / Number is used by existingChatID
}

// GetUserByID извлекает пользователя по его ID.
// GetUserByID retrieves a user by their ID.
func GetUserByID(userID int) (models.User, error) {
	var u models.User
	var encryptedCardNumber sql.NullString

	err := DB.QueryRow(`
        SELECT id, chat_id, role, first_name, last_name, nickname, phone, card_number,
               is_blocked, block_reason, block_date, COALESCE(main_menu_message_id, 0)
        FROM users WHERE id=$1`, userID).Scan(
		&u.ID, &u.ChatID, &u.Role, &u.FirstName, &u.LastName, &u.Nickname, &u.Phone, &encryptedCardNumber,
		&u.IsBlocked, &u.BlockReason, &u.BlockDate, &u.MainMenuMessageID)

	if err != nil {
		log.Printf("GetUserByID: ошибка получения пользователя ID %d: %v", userID, err)
		return u, err
	}

	if encryptedCardNumber.Valid && encryptedCardNumber.String != "" {
		decryptedCard, errDecrypt := utils.DecryptCardNumber(encryptedCardNumber.String)
		if errDecrypt != nil {
			log.Printf("GetUserByID: ошибка дешифрования номера карты для userID %d: %v. Будет возвращено пустое значение.", userID, errDecrypt)
			u.CardNumber = sql.NullString{String: "", Valid: false}
		} else {
			u.CardNumber = sql.NullString{String: decryptedCard, Valid: true}
		}
	} else {
		u.CardNumber = sql.NullString{String: "", Valid: false}
	}

	return u, nil
}

// GetOperatorForContact извлекает данные оператора для связи.
// GetOperatorForContact retrieves operator data for contact.
func GetOperatorForContact() (name string, phone string, err error) {
	err = DB.QueryRow(`
        SELECT first_name, phone FROM users
        WHERE role IN ($1, $2, $3) AND is_blocked=FALSE
        ORDER BY updated_at DESC LIMIT 1`,
		constants.ROLE_OPERATOR, constants.ROLE_MAINOPERATOR, constants.ROLE_OWNER).
		Scan(&name, &phone)
	if err != nil {
		if err == sql.ErrNoRows {
			log.Println("GetOperatorForContact: подходящий оператор не найден, используются значения по умолчанию.")
			return "Евгений", "+79789597077", nil // Default values
		}
		log.Printf("GetOperatorForContact: не удалось найти подходящего оператора: %v", err)
		return "", "", err
	}
	return name, phone, nil
}

// GetStaffListByRole извлекает список сотрудников по указанной роли.
// GetStaffListByRole retrieves a list of staff members by the specified role.
func GetStaffListByRole(role string) ([]models.User, error) {
	query := `SELECT id, chat_id, first_name, last_name, nickname, phone, role, card_number FROM users WHERE role=$1 AND is_blocked=FALSE ORDER BY first_name`
	rows, err := DB.Query(query, role)
	if err != nil {
		log.Printf("GetStaffListByRole: ошибка получения списка сотрудников для роли %s: %v", role, err)
		return nil, err
	}
	defer rows.Close()

	var staff []models.User
	for rows.Next() {
		var u models.User
		var encryptedCardNumber sql.NullString
		errScan := rows.Scan(&u.ID, &u.ChatID, &u.FirstName, &u.LastName, &u.Nickname, &u.Phone, &u.Role, &encryptedCardNumber)
		if errScan != nil {
			log.Printf("GetStaffListByRole: ошибка чтения сотрудника: %v", errScan)
			continue
		}

		if encryptedCardNumber.Valid && encryptedCardNumber.String != "" {
			decryptedCard, errDecrypt := utils.DecryptCardNumber(encryptedCardNumber.String)
			if errDecrypt != nil {
				log.Printf("GetStaffListByRole: ошибка дешифрования номера карты для chatID %d: %v.", u.ChatID, errDecrypt)
				u.CardNumber = sql.NullString{String: "", Valid: false}
			} else {
				u.CardNumber = sql.NullString{String: decryptedCard, Valid: true}
			}
		} else {
			u.CardNumber = sql.NullString{String: "", Valid: false}
		}
		staff = append(staff, u)
	}
	if err = rows.Err(); err != nil {
		log.Printf("GetStaffListByRole: ошибка после итерации по строкам: %v", err)
		return nil, err
	}
	return staff, nil
}

// AddStaff добавляет нового сотрудника.
// AddStaff adds a new staff member.
func AddStaff(chatID int64, role, firstName, lastName string, nickname sql.NullString, phone sql.NullString, cardNumber sql.NullString) error {
	var encryptedCardNumber sql.NullString
	if cardNumber.Valid && cardNumber.String != "" {
		encryptedVal, errEncrypt := utils.EncryptCardNumber(cardNumber.String)
		if errEncrypt != nil {
			log.Printf("AddStaff: ошибка шифрования номера карты для chatID %d: %v", chatID, errEncrypt)
			return fmt.Errorf("ошибка шифрования номера карты: %w", errEncrypt)
		}
		encryptedCardNumber = sql.NullString{String: encryptedVal, Valid: true}
	} else {
		encryptedCardNumber = sql.NullString{Valid: false} // Будет NULL в БД / Will be NULL in DB
	}

	_, err := DB.Exec(`
        INSERT INTO users (chat_id, role, first_name, last_name, nickname, phone, card_number, created_at, updated_at)
        VALUES ($1, $2, $3, $4, $5, $6, $7, NOW(), NOW())
        ON CONFLICT (chat_id) DO UPDATE SET
        role = EXCLUDED.role,
        first_name = EXCLUDED.first_name,
        last_name = EXCLUDED.last_name,
        nickname = EXCLUDED.nickname,
        phone = EXCLUDED.phone,
        card_number = EXCLUDED.card_number,
        updated_at = NOW(),
        is_blocked = FALSE, -- При добавлении/обновлении через AddStaff, сотрудник не должен быть заблокирован
        block_reason = NULL,
        block_date = NULL`,
		chatID, role, firstName, lastName, nickname, phone, encryptedCardNumber)
	if err != nil {
		log.Printf("AddStaff: ошибка добавления/обновления сотрудника с chat_id %d: %v", chatID, err)
		return err
	}
	log.Printf("Сотрудник с chat_id %d успешно добавлен/обновлен.", chatID)
	return nil
}

// DeleteStaff (мягкое удаление) переводит сотрудника в роль 'user'.
// DeleteStaff (soft delete) changes the staff member's role to 'user'.
func DeleteStaff(targetChatID int64) error {
	_, err := DB.Exec("UPDATE users SET role=$1, is_blocked=FALSE, block_reason=NULL, block_date=NULL, updated_at=NOW() WHERE chat_id=$2", constants.ROLE_USER, targetChatID)
	if err != nil {
		log.Printf("DeleteStaff: ошибка 'удаления' сотрудника (смена роли на user) %d: %v", targetChatID, err)
		return err
	}
	log.Printf("Сотрудник %d 'удален' (роль изменена на user).", targetChatID)
	return nil
}

// UpdateStaffField обновляет указанное поле для сотрудника.
// UpdateStaffField updates the specified field for a staff member.
func UpdateStaffField(targetChatID int64, field string, value interface{}) error {
	var valueToStore interface{} = value

	if field == "card_number" {
		if strVal, ok := value.(string); ok && strVal != "" {
			encryptedVal, errEncrypt := utils.EncryptCardNumber(strVal)
			if errEncrypt != nil {
				log.Printf("UpdateStaffField: ошибка шифрования номера карты для сотрудника %d: %v", targetChatID, errEncrypt)
				return fmt.Errorf("ошибка шифрования номера карты: %w", errEncrypt)
			}
			valueToStore = encryptedVal
		} else if strVal, ok := value.(string); ok && strVal == "" { // Если передана пустая строка, ставим NULL / If an empty string is passed, set NULL
			valueToStore = sql.NullString{Valid: false}
		} else if value == nil { // Если передано nil, ставим NULL / If nil is passed, set NULL
			valueToStore = sql.NullString{Valid: false}
		}
	}

	query := fmt.Sprintf("UPDATE users SET %s=$1, updated_at=NOW() WHERE chat_id=$2", field)
	_, err := DB.Exec(query, valueToStore, targetChatID)
	if err != nil {
		log.Printf("UpdateStaffField: ошибка обновления поля '%s' для сотрудника %d: %v", field, targetChatID, err)
		return err
	}
	log.Printf("Поле '%s' для сотрудника %d обновлено.", field, targetChatID)
	return nil
}

// GetUsersForBlocking получает список пользователей (роль 'user', не заблокированных) для отображения в меню блокировки.
// GetUsersForBlocking retrieves a list of users (role 'user', not blocked) to display in the blocking menu.
func GetUsersForBlocking() ([]models.User, error) {
	rows, err := DB.Query(`
        SELECT u.chat_id, u.first_name, u.last_name, u.nickname
        FROM users u
        LEFT JOIN (
            SELECT user_id, MAX(created_at) as last_message_time
            FROM chat_messages
            WHERE is_from_user = TRUE
            GROUP BY user_id
        ) cm ON u.id = cm.user_id
        WHERE u.is_blocked = FALSE AND u.role = $1
        ORDER BY cm.last_message_time DESC NULLS LAST, u.created_at DESC
        LIMIT 10`, constants.ROLE_USER) // Consider pagination if list can be long
	if err != nil {
		log.Printf("GetUsersForBlocking: ошибка получения списка пользователей: %v", err)
		return nil, err
	}
	defer rows.Close()

	var users []models.User
	for rows.Next() {
		var u models.User
		err := rows.Scan(&u.ChatID, &u.FirstName, &u.LastName, &u.Nickname)
		if err != nil {
			log.Printf("GetUsersForBlocking: ошибка чтения пользователя: %v", err)
			continue
		}
		users = append(users, u)
	}
	if err = rows.Err(); err != nil {
		log.Printf("GetUsersForBlocking: ошибка после итерации по строкам: %v", err)
		return nil, err
	}
	return users, nil
}

// GetBlockedUsers получает список заблокированных пользователей.
// GetBlockedUsers retrieves a list of blocked users.
func GetBlockedUsers() ([]models.User, error) {
	rows, err := DB.Query(`
        SELECT chat_id, first_name, last_name, nickname, block_date, block_reason
        FROM users
        WHERE is_blocked = TRUE
        ORDER BY block_date DESC NULLS LAST
        LIMIT 10`) // Consider pagination
	if err != nil {
		log.Printf("GetBlockedUsers: ошибка получения списка заблокированных пользователей: %v", err)
		return nil, err
	}
	defer rows.Close()

	var users []models.User
	for rows.Next() {
		var u models.User
		err := rows.Scan(&u.ChatID, &u.FirstName, &u.LastName, &u.Nickname, &u.BlockDate, &u.BlockReason)
		if err != nil {
			log.Printf("GetBlockedUsers: ошибка чтения заблокированного пользователя: %v", err)
			continue
		}
		users = append(users, u)
	}
	if err = rows.Err(); err != nil {
		log.Printf("GetBlockedUsers: ошибка после итерации по строкам: %v", err)
		return nil, err
	}
	return users, nil
}

// UserExists проверяет существование пользователя по chat_id.
// UserExists checks if a user exists by chat_id.
func UserExists(chatID int64) (bool, error) {
	var exists bool
	err := DB.QueryRow("SELECT EXISTS(SELECT 1 FROM users WHERE chat_id=$1)", chatID).Scan(&exists)
	if err != nil {
		log.Printf("UserExists: ошибка проверки существования пользователя chatID %d: %v", chatID, err)
		return false, err
	}
	return exists, nil
}

// UpdateUserField обновляет указанное поле для пользователя по chatID.
// Отличается от UpdateStaffField тем, что здесь chatID - это основной идентификатор.
// UpdateUserField updates the specified field for a user by chatID.
// Differs from UpdateStaffField in that chatID is the primary identifier here.
func UpdateUserField(chatID int64, field string, value interface{}) error {
	var valueToStore interface{} = value

	if field == "card_number" {
		if strVal, ok := value.(string); ok && strVal != "" {
			encryptedVal, errEncrypt := utils.EncryptCardNumber(strVal)
			if errEncrypt != nil {
				log.Printf("UpdateUserField: ошибка шифрования номера карты для chatID %d: %v", chatID, errEncrypt)
				return fmt.Errorf("ошибка шифрования номера карты: %w", errEncrypt)
			}
			valueToStore = encryptedVal
		} else if strVal, ok := value.(string); ok && strVal == "" {
			valueToStore = sql.NullString{Valid: false}
		} else if value == nil {
			valueToStore = sql.NullString{Valid: false}
		}
	} else if field == "first_name" || field == "last_name" || field == "phone" || field == "nickname" {
		if strVal, ok := value.(string); ok && strVal == "" {
			if field == "nickname" || field == "phone" { // Эти поля могут быть NULL / These fields can be NULL
				valueToStore = sql.NullString{Valid: false}
			}
			// Для FirstName и LastName пустая строка останется пустой строкой
			// For FirstName and LastName, an empty string will remain an empty string
		}
	}

	query := fmt.Sprintf("UPDATE users SET %s=$1, updated_at=NOW() WHERE chat_id=$2", field)
	_, err := DB.Exec(query, valueToStore, chatID)
	if err != nil {
		log.Printf("UpdateUserField: ошибка обновления поля '%s' для chatID %d: %v", field, chatID, err)
		return err
	}
	log.Printf("UpdateUserField: Поле '%s' для chatID %d обновлено.", field, chatID)
	return nil
}

// GetUsersByRole извлекает пользователей по указанным ролям.
// GetUsersByRole retrieves users by the specified roles.
func GetUsersByRole(roles ...string) ([]models.User, error) {
	if len(roles) == 0 {
		return nil, fmt.Errorf("необходимо указать хотя бы одну роль")
	}
	query := `
        SELECT id, chat_id, role, first_name, last_name, nickname, phone, card_number,
               is_blocked, block_reason, block_date, COALESCE(main_menu_message_id, 0)
        FROM users
        WHERE role = ANY($1) AND is_blocked = FALSE -- Добавлено AND is_blocked = FALSE, чтобы не получать заблокированных сотрудников
        ORDER BY first_name, last_name`
	rows, err := DB.Query(query, pq.Array(roles))
	if err != nil {
		log.Printf("GetUsersByRole: ошибка: %v", err)
		return nil, err
	}
	defer rows.Close()

	var users []models.User
	for rows.Next() {
		var u models.User
		var encryptedCardNumber sql.NullString
		if errScan := rows.Scan(
			&u.ID, &u.ChatID, &u.Role, &u.FirstName, &u.LastName, &u.Nickname, &u.Phone, &encryptedCardNumber,
			&u.IsBlocked, &u.BlockReason, &u.BlockDate, &u.MainMenuMessageID); errScan != nil {
			log.Printf("GetUsersByRole: ошибка сканирования: %v", errScan)
			continue
		}

		if encryptedCardNumber.Valid && encryptedCardNumber.String != "" {
			decryptedCard, errDecrypt := utils.DecryptCardNumber(encryptedCardNumber.String)
			if errDecrypt != nil {
				log.Printf("GetUsersByRole: ошибка дешифрования номера карты для chatID %d: %v.", u.ChatID, errDecrypt)
				u.CardNumber = sql.NullString{String: "", Valid: false}
			} else {
				u.CardNumber = sql.NullString{String: decryptedCard, Valid: true}
			}
		} else {
			u.CardNumber = sql.NullString{String: "", Valid: false}
		}
		users = append(users, u)
	}
	return users, rows.Err()
}

// === НАЧАЛО: ИСПРАВЛЕННАЯ ВЕРСИЯ ФУНКЦИИ GetOrdersByUserID ===

// GetOrdersByUserID извлекает все заказы для указанного ID пользователя.
func GetOrdersByUserID(userID int64) ([]models.Order, error) {
	query := `
		SELECT
			o.id, o.user_id, o.user_chat_id, o.category, o.subcategory, o.description,
			o.name, o.phone, o.address, o.date, o.time, o.payment, o.cost,
			o.status, o.reason, o.created_at, o.updated_at, o.photos, o.videos
		FROM orders o
		WHERE o.user_id = $1
		ORDER BY o.created_at DESC
	`
	rows, err := DB.Query(query, userID)
	if err != nil {
		log.Printf("GetOrdersByUserID: ошибка получения заказов для userID %d: %v", userID, err)
		return nil, err
	}
	defer rows.Close()

	var orders []models.Order
	for rows.Next() {
		var order models.Order
		// Для photos и videos нужны pq.Array
		var photos pq.StringArray
		var videos pq.StringArray

		// ИСПРАВЛЕНИЕ: Добавлено поле Reason в сканирование
		errScan := rows.Scan(
			&order.ID, &order.UserID, &order.UserChatID, &order.Category, &order.Subcategory, &order.Description,
			&order.Name, &order.Phone, &order.Address, &order.Date, &order.Time, &order.Payment, &order.Cost,
			&order.Status, &order.Reason, &order.CreatedAt, &order.UpdatedAt, &photos, &videos,
		)
		if errScan != nil {
			log.Printf("GetOrdersByUserID: ошибка сканирования заказа: %v", errScan)
			continue
		}
		order.Photos = photos
		order.Videos = videos
		orders = append(orders, order)
	}

	if err = rows.Err(); err != nil {
		log.Printf("GetOrdersByUserID: ошибка после итерации по строкам: %v", err)
		return nil, err
	}

	return orders, nil
}


// GetAllUsers возвращает всех пользователей из базы данных
func GetAllUsers() ([]models.User, error) {
	query := `
	SELECT 
		id, first_name, last_name, nickname, phone, role, 
		chat_id, is_blocked
	FROM users
	ORDER BY id DESC
	`

	rows, err := DB.Query(query)
	if err != nil {
		return nil, fmt.Errorf("ошибка выполнения запроса GetAllUsers: %v", err)
	}
	defer rows.Close()

	var users []models.User
	for rows.Next() {
		var user models.User

		err := rows.Scan(
			&user.ID, &user.FirstName, &user.LastName, &user.Nickname,
			&user.Phone, &user.Role, &user.ChatID, &user.IsBlocked,
		)
		if err != nil {
			return nil, fmt.Errorf("ошибка сканирования строки в GetAllUsers: %v", err)
		}


		users = append(users, user)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("ошибка итерации строк в GetAllUsers: %v", err)
	}

	return users, nil
}

// DeleteUser удаляет пользователя из базы данных
// DeleteUser removes a user from the database
func DeleteUser(userID int64) error {
	// Сначала проверим, есть ли у пользователя заказы
	var orderCount int
	err := DB.QueryRow("SELECT COUNT(*) FROM orders WHERE user_id = $1", userID).Scan(&orderCount)
	if err != nil {
		log.Printf("DeleteUser: ошибка проверки заказов для userID %d: %v", userID, err)
		return err
	}
	
	if orderCount > 0 {
		log.Printf("DeleteUser: пользователь %d имеет %d заказов, удаление невозможно", userID, orderCount)
		return fmt.Errorf("невозможно удалить пользователя с заказами")
	}
	
	_, err = DB.Exec("DELETE FROM users WHERE id = $1", userID)
	if err != nil {
		log.Printf("DeleteUser: ошибка удаления пользователя %d: %v", userID, err)
		return err
	}
	
	log.Printf("Пользователь %d успешно удален", userID)
	return nil
}
