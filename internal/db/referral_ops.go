package db

import (
	"Original/internal/constants"
	"Original/internal/models" // Используем Original как имя модуля / Use Original as module name
	"database/sql"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/lib/pq" // Для работы с массивами PostgreSQL / For working with PostgreSQL arrays
)

// AddReferral добавляет новую реферальную запись.
// inviterChatID и inviteeChatID - это chat_id пользователей.
// AddReferral adds a new referral record.
// inviterChatID and inviteeChatID are user chat_ids.
func AddReferral(inviterChatID, inviteeChatID int64, orderID int, amount float64) (int64, error) {
	var inviterUserID, inviteeUserID int

	// Получаем ID пользователя-пригласителя / Get inviter's user ID
	err := DB.QueryRow("SELECT id FROM users WHERE chat_id = $1", inviterChatID).Scan(&inviterUserID)
	if err != nil {
		log.Printf("AddReferral: не удалось найти inviter_id для chat_id %d: %v", inviterChatID, err)
		return 0, fmt.Errorf("пригласивший пользователь (chat_id %d) не найден: %w", inviterChatID, err)
	}
	// Получаем ID приглашенного пользователя / Get invitee's user ID
	err = DB.QueryRow("SELECT id FROM users WHERE chat_id = $1", inviteeChatID).Scan(&inviteeUserID)
	if err != nil {
		log.Printf("AddReferral: не удалось найти invitee_id для chat_id %d: %v", inviteeChatID, err)
		return 0, fmt.Errorf("приглашенный пользователь (chat_id %d) не найден: %w", inviteeChatID, err)
	}

	var id int64
	// paid_out по умолчанию FALSE, payout_request_id по умолчанию NULL при создании
	// paid_out defaults to FALSE, payout_request_id defaults to NULL on creation
	query := `
        INSERT INTO referrals (inviter_id, invitee_id, order_id, amount, created_at, updated_at, paid_out, payout_request_id)
        VALUES ($1, $2, $3, $4, NOW(), NOW(), FALSE, NULL) 
        RETURNING id`
	err = DB.QueryRow(query, inviterUserID, inviteeUserID, orderID, amount).Scan(&id)
	if err != nil {
		log.Printf("AddReferral: ошибка добавления реферала (inviter_id %d, invitee_id %d, order_id %d): %v", inviterUserID, inviteeUserID, orderID, err)
		return 0, err
	}
	log.Printf("Реферал #%d успешно добавлен.", id)
	return id, nil
}

// GetReferralsByInviterChatID извлекает всех рефералов, приглашенных пользователем.
// inviterChatID - chat_id пользователя, который приглашал.
// GetReferralsByInviterChatID retrieves all referrals invited by a user.
// inviterChatID - chat_id of the inviting user.
func GetReferralsByInviterChatID(inviterChatID int64) ([]models.Referral, error) {
	var inviterUserID int
	err := DB.QueryRow("SELECT id FROM users WHERE chat_id = $1", inviterChatID).Scan(&inviterUserID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("пользователь-пригласитель с chat_id %d не найден", inviterChatID)
		}
		log.Printf("GetReferralsByInviterChatID: ошибка получения user_id для inviter_chat_id %d: %v", inviterChatID, err)
		return nil, err
	}

	rows, err := DB.Query(`
        SELECT r.id, r.inviter_id, r.invitee_id, r.order_id, r.amount, r.created_at, r.paid_out, r.payout_request_id,
               u_invitee.first_name, u_invitee.last_name -- Имя приглашенного / Invitee's name
        FROM referrals r
        JOIN users u_invitee ON r.invitee_id = u_invitee.id
        WHERE r.inviter_id = $1
        ORDER BY r.created_at DESC`, inviterUserID)
	if err != nil {
		log.Printf("GetReferralsByInviterChatID: ошибка получения рефералов для inviter_id %d: %v", inviterUserID, err)
		return nil, err
	}
	defer rows.Close()

	var referrals []models.Referral
	for rows.Next() {
		var r models.Referral
		var inviteeFirstName, inviteeLastName sql.NullString // Используем sql.NullString для имен / Use sql.NullString for names
		errScan := rows.Scan(
			&r.ID,
			&r.InviterID, // Это users.id / This is users.id
			&r.InviteeID, // Это users.id / This is users.id
			&r.OrderID,
			&r.Amount,
			&r.CreatedAt,
			&r.PaidOut,
			&r.PayoutRequestID, // sql.NullInt64
			&inviteeFirstName,
			&inviteeLastName,
		)
		if errScan != nil {
			log.Printf("GetReferralsByInviterChatID: ошибка сканирования реферала для inviter_id %d: %v", inviterUserID, errScan)
			continue
		}
		nameParts := []string{}
		if inviteeFirstName.Valid && inviteeFirstName.String != "" {
			nameParts = append(nameParts, inviteeFirstName.String)
		}
		if inviteeLastName.Valid && inviteeLastName.String != "" {
			nameParts = append(nameParts, inviteeLastName.String)
		}
		r.Name = strings.Join(nameParts, " ")
		if r.Name == "" {
			r.Name = fmt.Sprintf("Пользователь ID %d", r.InviteeID) // Fallback
		}

		referrals = append(referrals, r)
	}
	if err = rows.Err(); err != nil {
		log.Printf("GetReferralsByInviterChatID: ошибка после итерации по строкам для inviter_id %d: %v", inviterUserID, err)
		return nil, err
	}
	return referrals, nil
}

// HasReferralBonus проверяет, получал ли пользователь реферальный бонус (как приглашенный).
// inviteeChatID - chat_id пользователя, которого могли пригласить.
// HasReferralBonus checks if a user has received a referral bonus (as an invitee).
// inviteeChatID - chat_id of the user who might have been invited.
func HasReferralBonus(inviteeChatID int64) (bool, error) {
	var inviteeUserID int
	err := DB.QueryRow("SELECT id FROM users WHERE chat_id = $1", inviteeChatID).Scan(&inviteeUserID)
	if err != nil {
		if err == sql.ErrNoRows { // Пользователь не найден, значит и бонуса нет / User not found, so no bonus
			return false, nil
		}
		log.Printf("HasReferralBonus: ошибка получения user_id для invitee_chat_id %d: %v", inviteeChatID, err)
		return false, err
	}

	var count int
	err = DB.QueryRow("SELECT COUNT(*) FROM referrals WHERE invitee_id=$1", inviteeUserID).Scan(&count)
	if err != nil {
		log.Printf("HasReferralBonus: ошибка проверки реферального бонуса для invitee_id %d: %v", inviteeUserID, err)
		return false, err
	}
	return count > 0, nil
}

// GetReferralsForExcel получает данные для Excel-отчета по рефералам.
// GetReferralsForExcel retrieves data for an Excel report on referrals.
func GetReferralsForExcel() (*sql.Rows, error) {
	// Выбираем рефералов за текущую дату / Select referrals for the current date
	query := `
        SELECT u_inviter.first_name, u_inviter.last_name,
               u_invitee.first_name, u_invitee.last_name,
               r.amount, r.created_at, r.paid_out
        FROM referrals r
        JOIN users u_inviter ON r.inviter_id = u_inviter.id
        JOIN users u_invitee ON r.invitee_id = u_invitee.id
        WHERE date_trunc('day', r.created_at) = date_trunc('day', CURRENT_TIMESTAMP)
        ORDER BY r.created_at DESC`
	rows, err := DB.Query(query)
	if err != nil {
		log.Printf("GetReferralsForExcel: ошибка получения данных для Excel: %v", err)
		return nil, err
	}
	return rows, nil
}

// GetReferralByID извлекает данные реферала по его ID.
// Также проверяет, что запрашивающий пользователь (currentUserChatID) является пригласившим.
// GetReferralByID retrieves referral data by its ID.
// Also checks that the requesting user (currentUserChatID) is the inviter.
func GetReferralByID(referralID int64, currentUserChatID int64) (models.Referral, error) {
	var r models.Referral
	var inviterUserID int // ID пригласившего из таблицы users / Inviter's ID from users table

	// Получаем ID текущего пользователя (который делает запрос)
	// Get ID of the current user (who is making the request)
	err := DB.QueryRow("SELECT id FROM users WHERE chat_id = $1", currentUserChatID).Scan(&inviterUserID)
	if err != nil {
		if err == sql.ErrNoRows {
			return r, fmt.Errorf("текущий пользователь (chat_id %d) не найден", currentUserChatID)
		}
		log.Printf("GetReferralByID: ошибка получения user_id для currentUserChatID %d: %v", currentUserChatID, err)
		return r, err
	}

	var inviteeFirstName, inviteeLastName sql.NullString
	query := `
        SELECT r.id, r.inviter_id, r.invitee_id, r.order_id, r.amount, r.created_at, r.paid_out, r.payout_request_id,
               u_invitee.first_name, u_invitee.last_name
        FROM referrals r
        JOIN users u_invitee ON r.invitee_id = u_invitee.id
        WHERE r.id = $1 AND r.inviter_id = $2` // Проверяем, что текущий пользователь - пригласивший / Check that current user is the inviter

	err = DB.QueryRow(query, referralID, inviterUserID).Scan(
		&r.ID,
		&r.InviterID,
		&r.InviteeID,
		&r.OrderID,
		&r.Amount,
		&r.CreatedAt,
		&r.PaidOut,
		&r.PayoutRequestID,
		&inviteeFirstName,
		&inviteeLastName,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			log.Printf("GetReferralByID: реферал #%d не найден для пользователя (user_id %d) или пользователь не является пригласившим.", referralID, inviterUserID)
			return r, fmt.Errorf("реферал не найден или у вас нет доступа к его деталям")
		}
		log.Printf("GetReferralByID: ошибка получения реферала #%d для пользователя (user_id %d): %v", referralID, inviterUserID, err)
		return r, err
	}
	nameParts := []string{}
	if inviteeFirstName.Valid {
		nameParts = append(nameParts, inviteeFirstName.String)
	}
	if inviteeLastName.Valid {
		nameParts = append(nameParts, inviteeLastName.String)
	}
	r.Name = strings.Join(nameParts, " ")
	if r.Name == "" {
		r.Name = fmt.Sprintf("Пользователь ID %d", r.InviteeID)
	}

	return r, nil
}

// CreateReferralPayoutRequest создает новый запрос на выплату реферальных бонусов.
// CreateReferralPayoutRequest creates a new referral bonus payout request.
func CreateReferralPayoutRequest(request models.ReferralPayoutRequest) (int64, error) {
	var requestID int64
	tx, err := DB.Begin()
	if err != nil {
		log.Printf("CreateReferralPayoutRequest: ошибка начала транзакции: %v", err)
		return 0, err
	}
	defer tx.Rollback() // Гарантирует откат, если Commit не был вызван / Ensures rollback if Commit was not called

	// 1. Вставляем сам запрос на выплату / 1. Insert the payout request itself
	queryRequest := `
		INSERT INTO referral_payout_requests (user_chat_id, amount, status, requested_at, admin_comment, processed_at, payment_method, payment_details)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id`
	err = tx.QueryRow(queryRequest,
		request.UserChatID,
		request.Amount,
		request.Status,         // Должен быть constants.PAYOUT_REQUEST_STATUS_PENDING / Should be constants.PAYOUT_REQUEST_STATUS_PENDING
		request.RequestedAt,    // Обычно time.Now() при создании / Usually time.Now() on creation
		request.AdminComment,   // Обычно NULL при создании / Usually NULL on creation
		request.ProcessedAt,    // Обычно NULL при создании / Usually NULL on creation
		request.PaymentMethod,  // Может быть NULL / Can be NULL
		request.PaymentDetails, // Может быть NULL / Can be NULL
	).Scan(&requestID)

	if err != nil {
		log.Printf("CreateReferralPayoutRequest: ошибка создания запроса на выплату: %v", err)
		return 0, err
	}

	// 2. Обновляем referral.payout_request_id для всех включенных рефералов
	// 2. Update referral.payout_request_id for all included referrals
	if len(request.ReferralIDs) > 0 {
		stmtUpdateReferral, errPrepare := tx.Prepare(`UPDATE referrals SET payout_request_id = $1, updated_at = NOW() WHERE id = ANY($2) AND payout_request_id IS NULL AND paid_out = FALSE`)
		if errPrepare != nil {
			log.Printf("CreateReferralPayoutRequest: ошибка подготовки обновления рефералов: %v", errPrepare)
			return 0, errPrepare
		}
		defer stmtUpdateReferral.Close()

		res, errExec := stmtUpdateReferral.Exec(requestID, pq.Array(request.ReferralIDs))
		if errExec != nil {
			log.Printf("CreateReferralPayoutRequest: ошибка обновления payout_request_id у рефералов: %v", errExec)
			return 0, errExec
		}
		rowsAffected, _ := res.RowsAffected()
		log.Printf("CreateReferralPayoutRequest: обновлено %d рефералов для запроса #%d.", rowsAffected, requestID)
		if rowsAffected != int64(len(request.ReferralIDs)) {
			log.Printf("CreateReferralPayoutRequest: ВНИМАНИЕ! Ожидалось обновление %d рефералов, но обновлено %d. Возможно, некоторые уже были в запросе или выплачены.", len(request.ReferralIDs), rowsAffected)
			// Это не обязательно ошибка, но стоит залогировать / Not necessarily an error, but worth logging
		}
	} else {
		log.Printf("CreateReferralPayoutRequest: Запрос на выплату #%d создан без ID рефералов.", requestID)
	}

	if err = tx.Commit(); err != nil {
		log.Printf("CreateReferralPayoutRequest: ошибка коммита транзакции: %v", err)
		return 0, err
	}

	log.Printf("Запрос на выплату реферальных бонусов #%d успешно создан для пользователя %d на сумму %.0f.", requestID, request.UserChatID, request.Amount)
	return requestID, nil
}

// MarkReferralsAsPaidOutByRequestID помечает рефералов как выплаченные по ID запроса на выплату.
// MarkReferralsAsPaidOutByRequestID marks referrals as paid out by payout request ID.
func MarkReferralsAsPaidOutByRequestID(payoutRequestID int64, paid bool) error {
	// Обновляем только те рефералы, которые связаны с этим запросом
	// Update only referrals associated with this request
	_, err := DB.Exec(
		"UPDATE referrals SET paid_out = $1, updated_at = NOW() WHERE payout_request_id = $2",
		paid, payoutRequestID,
	)
	if err != nil {
		log.Printf("MarkReferralsAsPaidOutByRequestID: ошибка обновления статуса paid_out для payout_request_id %d: %v", payoutRequestID, err)
		return err
	}
	log.Printf("Статус paid_out для рефералов с payout_request_id %d обновлен на %v.", payoutRequestID, paid)
	return nil
}

// GetReferralPayoutRequestByID получает запрос на выплату по ID.
// GetReferralPayoutRequestByID retrieves a payout request by ID.
func GetReferralPayoutRequestByID(requestID int64) (models.ReferralPayoutRequest, error) {
	var req models.ReferralPayoutRequest
	// var referralIDs pq.Int64Array // Для чтения массива из БД (если бы он хранился в JSON) / For reading array from DB (if stored in JSON)

	query := `SELECT id, user_chat_id, amount, status, requested_at, admin_comment, processed_at, payment_method, payment_details
              FROM referral_payout_requests WHERE id = $1`
	err := DB.QueryRow(query, requestID).Scan(
		&req.ID, &req.UserChatID, &req.Amount, &req.Status, &req.RequestedAt,
		&req.AdminComment, &req.ProcessedAt, &req.PaymentMethod, &req.PaymentDetails,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return req, fmt.Errorf("запрос на выплату с ID %d не найден", requestID)
		}
		log.Printf("GetReferralPayoutRequestByID: ошибка получения запроса на выплату #%d: %v", requestID, err)
		return req, err
	}

	// Загрузка связанных ReferralIDs из таблицы referrals
	// Load associated ReferralIDs from the referrals table
	rows, err := DB.Query("SELECT id FROM referrals WHERE payout_request_id = $1", requestID)
	if err != nil {
		log.Printf("GetReferralPayoutRequestByID: ошибка получения ID рефералов для запроса #%d: %v", requestID, err)
		// Можно вернуть ошибку или продолжить без ID рефералов, если это допустимо
		// Can return an error or continue without referral IDs if permissible
	} else {
		defer rows.Close()
		for rows.Next() {
			var refID int64
			if errScan := rows.Scan(&refID); errScan == nil {
				req.ReferralIDs = append(req.ReferralIDs, refID)
			}
		}
		if err = rows.Err(); err != nil {
			log.Printf("GetReferralPayoutRequestByID: ошибка после итерации по ID рефералов для запроса #%d: %v", requestID, err)
		}
	}
	return req, nil
}

// UpdateReferralPayoutRequestStatusAndComment обновляет статус и комментарий администратора для запроса на выплату.
// UpdateReferralPayoutRequestStatusAndComment updates the status and admin comment for a payout request.
func UpdateReferralPayoutRequestStatusAndComment(requestID int64, newStatus string, adminComment sql.NullString) error {
	query := `UPDATE referral_payout_requests SET status = $1, admin_comment = $2, processed_at = $3, payment_details = $4 WHERE id = $5`
	var processedAt sql.NullTime
	var paymentDetailsForUpdate sql.NullString // Для обновления деталей платежа, если нужно / For updating payment details if needed

	// Устанавливаем processed_at, если статус конечный
	// Set processed_at if status is final
	if newStatus == constants.PAYOUT_REQUEST_STATUS_APPROVED ||
		newStatus == constants.PAYOUT_REQUEST_STATUS_REJECTED ||
		newStatus == constants.PAYOUT_REQUEST_STATUS_COMPLETED {
		processedAt = sql.NullTime{Time: time.Now(), Valid: true}
	}

	// Если статус "approved", можно добавить детали платежа (например, "Переведено на карту XXXX")
	// If status is "approved", payment details can be added (e.g., "Transferred to card XXXX")
	if newStatus == constants.PAYOUT_REQUEST_STATUS_APPROVED {
		// Здесь можно получить карту пользователя из user_chat_id запроса, если нужно
		// User's card can be retrieved from request's user_chat_id if needed
		// paymentDetailsForUpdate = sql.NullString{String: "Одобрено, ожидает выплаты", Valid: true}
	}
	if newStatus == constants.PAYOUT_REQUEST_STATUS_COMPLETED {
		paymentDetailsForUpdate = sql.NullString{String: "Выплачено", Valid: true}
	}

	result, err := DB.Exec(query, newStatus, adminComment, processedAt, paymentDetailsForUpdate, requestID)
	if err != nil {
		log.Printf("UpdateReferralPayoutRequestStatus: ошибка обновления статуса для запроса #%d: %v", requestID, err)
		return err
	}
	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("запрос на выплату #%d не найден для обновления статуса", requestID)
	}

	// Если статус "completed", помечаем связанные рефералы как выплаченные
	// If status is "completed", mark associated referrals as paid out
	if newStatus == constants.PAYOUT_REQUEST_STATUS_COMPLETED {
		errPaid := MarkReferralsAsPaidOutByRequestID(requestID, true)
		if errPaid != nil {
			// Логируем ошибку, но основной статус запроса уже обновлен
			// Log error, but main request status is already updated
			log.Printf("UpdateReferralPayoutRequestStatus: ошибка маркировки рефералов как выплаченных для запроса #%d: %v", requestID, errPaid)
		}
	} else if newStatus == constants.PAYOUT_REQUEST_STATUS_REJECTED || newStatus == constants.PAYOUT_REQUEST_STATUS_PENDING {
		// Если отклонен или вернули в pending, снимаем пометку о выплате с рефералов
		// If rejected or returned to pending, unmark referrals as paid out
		errPaid := MarkReferralsAsPaidOutByRequestID(requestID, false)
		if errPaid != nil {
			log.Printf("UpdateReferralPayoutRequestStatus: ошибка снятия маркировки выплаты для рефералов запроса #%d: %v", requestID, errPaid)
		}
	}

	log.Printf("Статус запроса на выплату #%d обновлен на %s.", requestID, newStatus)
	return nil
}
