package db

import (
	"Original/internal/constants"
	"Original/internal/models"
	"Original/internal/utils" // Для дешифрования номера карты при необходимости (хотя здесь не используется напрямую)
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"strconv" // Для преобразования UserID грузчика (int64) в строку для ключа JSON
)

// addPayoutWithinTx добавляет запись о выплате в рамках существующей транзакции.
// Это полезно, когда создание выплаты является частью более крупной операции (например, MarkLoaderSalaryAsPaidByDriver).
// addPayoutWithinTx adds a payout record within an existing transaction.
// This is useful when creating a payout is part of a larger operation (e.g., MarkLoaderSalaryAsPaidByDriver).
func addPayoutWithinTx(tx *sql.Tx, payout models.Payout) (int64, error) {
	var id int64
	// Убедимся, что order_id передается как sql.NullInt64, если он может быть 0
	// Ensure order_id is passed as sql.NullInt64 if it can be 0
	var orderIDArg sql.NullInt64
	if payout.OrderID != 0 {
		orderIDArg = sql.NullInt64{Int64: payout.OrderID, Valid: true}
	}

	query := `
        INSERT INTO payouts (user_id, amount, payout_date, order_id, comment, made_by_user_id, created_at)
        VALUES ($1, $2, $3, $4, $5, $6, NOW())
        RETURNING id`
	err := tx.QueryRow(query,
		payout.UserID,
		payout.Amount,
		payout.PayoutDate,
		orderIDArg, // Используем sql.NullInt64 для order_id / Use sql.NullInt64 for order_id
		payout.Comment,
		payout.MadeByUserID,
	).Scan(&id)

	if err != nil {
		log.Printf("addPayoutWithinTx: ошибка добавления выплаты для userID %d: %v", payout.UserID, err)
		return 0, err
	}
	log.Printf("addPayoutWithinTx: выплата #%d на сумму %.0f для userID %d успешно добавлена в транзакции.", id, payout.Amount, payout.UserID)
	return id, nil
}

// AddPayout добавляет новую запись о выплате.
// AddPayout adds a new payout record.
func AddPayout(payout models.Payout) (int64, error) {
	tx, err := DB.Begin()
	if err != nil {
		log.Printf("AddPayout: ошибка начала транзакции: %v", err)
		return 0, err
	}
	defer tx.Rollback() // Rollback if commit is not called

	id, err := addPayoutWithinTx(tx, payout)
	if err != nil {
		return 0, err // Ошибка уже залогирована в addPayoutWithinTx / Error already logged in addPayoutWithinTx
	}

	if err = tx.Commit(); err != nil {
		log.Printf("AddPayout: ошибка коммита транзакции: %v", err)
		return 0, err
	}
	log.Printf("AddPayout: выплата #%d для userID %d успешно зафиксирована в БД.", id, payout.UserID)
	return id, nil
}

// GetPayoutsByUserID извлекает все выплаты, произведенные пользователю.
// GetPayoutsByUserID retrieves all payouts made to a user.
func GetPayoutsByUserID(userID int64) ([]models.Payout, error) {
	rows, err := DB.Query(`
        SELECT id, user_id, amount, payout_date, order_id, comment, made_by_user_id, created_at
        FROM payouts
        WHERE user_id = $1
        ORDER BY payout_date DESC`, userID)
	if err != nil {
		log.Printf("GetPayoutsByUserID: ошибка получения выплат для userID %d: %v", userID, err)
		return nil, err
	}
	defer rows.Close()

	var payouts []models.Payout
	for rows.Next() {
		var p models.Payout
		var orderID sql.NullInt64 // Для чтения order_id, который может быть NULL / For reading order_id, which can be NULL
		errScan := rows.Scan(
			&p.ID,
			&p.UserID,
			&p.Amount,
			&p.PayoutDate,
			&orderID, // Сканируем в sql.NullInt64 / Scan into sql.NullInt64
			&p.Comment,
			&p.MadeByUserID,
			&p.CreatedAt,
		)
		if errScan != nil {
			log.Printf("GetPayoutsByUserID: ошибка сканирования выплаты для userID %d: %v", userID, errScan)
			continue
		}
		if orderID.Valid {
			p.OrderID = orderID.Int64
		}
		payouts = append(payouts, p)
	}
	if err = rows.Err(); err != nil {
		log.Printf("GetPayoutsByUserID: ошибка после итерации по строкам для userID %d: %v", userID, err)
		return nil, err
	}
	return payouts, nil
}

// GetTotalPaidToUser рассчитывает общую сумму, выплаченную пользователю.
// GetTotalPaidToUser calculates the total amount paid to a user.
func GetTotalPaidToUser(userID int64) (float64, error) {
	var totalPaid sql.NullFloat64 // Используем NullFloat64, так как SUM может вернуть NULL, если нет записей / Use NullFloat64 as SUM can return NULL if no records
	err := DB.QueryRow("SELECT SUM(amount) FROM payouts WHERE user_id = $1", userID).Scan(&totalPaid)
	if err != nil {
		// Ошибка sql.ErrNoRows здесь не ожидается для SUM(), но на всякий случай.
		// sql.ErrNoRows is not expected here for SUM(), but just in case.
		if err == sql.ErrNoRows {
			return 0, nil
		}
		log.Printf("GetTotalPaidToUser: ошибка расчета общей выплаченной суммы для userID %d: %v", userID, err)
		return 0, err
	}
	if !totalPaid.Valid { // Если SUM вернул NULL (нет выплат) / If SUM returned NULL (no payouts)
		return 0, nil
	}
	return totalPaid.Float64, nil
}

// GetTotalEarnedForUser рассчитывает общую сумму, заработанную пользователем (водителем или грузчиком).
// 'role' используется для определения, как считать заработок.
// GetTotalEarnedForUser calculates the total amount earned by a user (driver or loader).
// 'role' is used to determine how to calculate earnings.
func GetTotalEarnedForUser(userID int64, role string) (float64, error) {
	var totalEarned float64

	switch role {
	case constants.ROLE_LOADER:
		// Для грузчика: суммируем его зарплаты из всех записей expenses.loader_salaries,
		// где заказ имеет статус CALCULATED или SETTLED (или COMPLETED, если расчет происходит сразу).
		// For loader: sum their salaries from all expenses.loader_salaries records,
		// where the order has status CALCULATED or SETTLED (or COMPLETED, if calculation happens immediately).
		rows, err := DB.Query(`
            SELECT e.loader_salaries
            FROM expenses e
            JOIN orders o ON e.order_id = o.id
            WHERE o.status IN ($1, $2, $3) AND e.loader_salaries IS NOT NULL AND e.loader_salaries::text != '{}'`,
			constants.STATUS_CALCULATED, constants.STATUS_SETTLED, constants.STATUS_COMPLETED) // Добавлен COMPLETED для универсальности
		if err != nil {
			log.Printf("GetTotalEarnedForUser (loader): ошибка получения записей expenses: %v", err)
			return 0, err
		}
		defer rows.Close()

		loaderUserIDStr := strconv.FormatInt(userID, 10)

		for rows.Next() {
			var loaderSalariesJSON []byte
			if errScan := rows.Scan(&loaderSalariesJSON); errScan != nil {
				log.Printf("GetTotalEarnedForUser (loader): ошибка сканирования loader_salaries: %v", errScan)
				continue
			}

			var salariesMap map[string]models.LoaderSalaryDetail
			if errUnmarshal := json.Unmarshal(loaderSalariesJSON, &salariesMap); errUnmarshal != nil {
				log.Printf("GetTotalEarnedForUser (loader): ошибка демаршалинга loader_salaries: %v, JSON: %s", errUnmarshal, string(loaderSalariesJSON))
				continue
			}

			if salaryDetail, ok := salariesMap[loaderUserIDStr]; ok {
				totalEarned += salaryDetail.Amount
			}
		}
		if err = rows.Err(); err != nil {
			log.Printf("GetTotalEarnedForUser (loader): ошибка после итерации по expenses: %v", err)
			return 0, err
		}

	case constants.ROLE_DRIVER:
		// Для водителя: суммируем его driver_share из всех записей expenses,
		// где заказ имеет статус CALCULATED или SETTLED (или COMPLETED).
		// For driver: sum their driver_share from all expenses records,
		// where the order has status CALCULATED or SETTLED (or COMPLETED).
		var sumDriverShare sql.NullFloat64
		err := DB.QueryRow(`
            SELECT SUM(e.driver_share) 
            FROM expenses e
            JOIN orders o ON e.order_id = o.id
            WHERE e.driver_id = $1 AND o.status IN ($2, $3, $4)`,
			userID, constants.STATUS_CALCULATED, constants.STATUS_SETTLED, constants.STATUS_COMPLETED).Scan(&sumDriverShare)
		if err != nil {
			log.Printf("GetTotalEarnedForUser (driver): ошибка расчета driver_share для userID %d: %v", userID, err)
			return 0, err
		}
		if sumDriverShare.Valid {
			totalEarned = sumDriverShare.Float64
		}
	default:
		// Для других ролей (оператор, владелец) "заработок" может рассчитываться иначе или не рассчитываться здесь.
		// For other roles (operator, owner) "earnings" might be calculated differently or not here.
		log.Printf("GetTotalEarnedForUser: роль '%s' для userID %d не предполагает прямого расчета заработка через expenses.", role, userID)
		// Можно вернуть 0 или ошибку, в зависимости от ожидаемого поведения.
		// Can return 0 or an error, depending on expected behavior.
		return 0, nil // fmt.Errorf("невозможно рассчитать заработок для роли: %s", role)
	}

	return totalEarned, nil
}

// GetAmountOwedToUser рассчитывает сумму, которую компания должна пользователю.
// (Общий заработок - Общая сумма выплат)
// GetAmountOwedToUser calculates the amount the company owes to the user.
// (Total earnings - Total payouts)
func GetAmountOwedToUser(userID int64, role string) (float64, error) {
	totalEarned, err := GetTotalEarnedForUser(userID, role)
	if err != nil {
		log.Printf("GetAmountOwedToUser: ошибка получения общего заработка для userID %d, роль %s: %v", userID, role, err)
		return 0, err
	}

	totalPaid, err := GetTotalPaidToUser(userID)
	if err != nil {
		log.Printf("GetAmountOwedToUser: ошибка получения общей выплаченной суммы для userID %d: %v", userID, err)
		return 0, err
	}

	amountOwed := totalEarned - totalPaid
	// Округление до 2 знаков после запятой, если необходимо
	// Round to 2 decimal places if necessary
	// amountOwed = math.Round(amountOwed*100) / 100
	if amountOwed < 0 { // Сумма к выплате не может быть отрицательной
		amountOwed = 0
	}
	return amountOwed, nil
}

// GetCardNumberByUserID извлекает номер карты пользователя по его ID.
// Возвращает дешифрованный номер карты.
// GetCardNumberByUserID retrieves a user's card number by their ID.
// Returns the decrypted card number.
func GetCardNumberByUserID(userID int64) (string, error) {
	var encryptedCardNumber sql.NullString
	err := DB.QueryRow("SELECT card_number FROM users WHERE id = $1", userID).Scan(&encryptedCardNumber)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", fmt.Errorf("пользователь с ID %d не найден", userID)
		}
		log.Printf("GetCardNumberByUserID: ошибка получения номера карты для userID %d: %v", userID, err)
		return "", err
	}

	if !encryptedCardNumber.Valid || encryptedCardNumber.String == "" {
		return "", fmt.Errorf("номер карты не указан для пользователя ID %d", userID)
	}

	decryptedCard, errDecrypt := utils.DecryptCardNumber(encryptedCardNumber.String)
	if errDecrypt != nil {
		log.Printf("GetCardNumberByUserID: ошибка дешифрования номера карты для userID %d: %v", userID, errDecrypt)
		return "", fmt.Errorf("ошибка дешифрования номера карты")
	}
	return decryptedCard, nil
}
