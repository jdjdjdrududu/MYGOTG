package db

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"strconv" // Для преобразования int64 в string для ключа карты / For converting int64 to string for map key
	"time"    // Для создания Payout записи / For creating Payout record

	"Original/internal/constants" // <-- УБЕДИТЕСЬ, ЧТО ЭТОТ ИМПОРТ ЕСТЬ (для constants.STATUS_COMPLETED и т.д.)
	"Original/internal/models"
)

// AddExpense добавляет новую запись о расходах или обновляет существующую по order_id.
// Убедитесь, что expense.DriverID это users.id водителя.
// expense.LoaderSalaries теперь map[string]models.LoaderSalaryDetail
// AddExpense adds a new expense record or updates an existing one by order_id.
// Ensure expense.DriverID is the driver's users.id.
// expense.LoaderSalaries is now map[string]models.LoaderSalaryDetail
func AddExpense(expense models.Expense) (int64, error) {
	tx, errTx := DB.Begin()
	if errTx != nil {
		log.Printf("AddExpense: ошибка начала транзакции: %v", errTx)
		return 0, errTx
	}
	// Используем defer с именованной переменной ошибки для корректного отката
	var err error
	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			panic(p) // Re-panic after Rollback
		} else if err != nil {
			log.Printf("AddExpense: откат транзакции из-за ошибки: %v", err)
			tx.Rollback()
		} else {
			err = tx.Commit()
			if err != nil {
				log.Printf("AddExpense: ошибка коммита транзакции: %v", err)
			}
		}
	}()

	loaderSalariesJSON, errMarshal := json.Marshal(expense.LoaderSalaries)
	if errMarshal != nil {
		log.Printf("AddExpense: ошибка маршалинга loader_salaries для orderID %d: %v", expense.OrderID, errMarshal)
		err = fmt.Errorf("ошибка подготовки данных о зарплатах грузчиков: %w", errMarshal)
		return 0, err
	}

	var id int64
	query := `
        INSERT INTO expenses (order_id, driver_id, fuel, other, loader_salaries, revenue, driver_share, created_at, updated_at)
        VALUES ($1, $2, $3, $4, $5, $6, $7, NOW(), NOW())
        ON CONFLICT (order_id) DO UPDATE SET
            driver_id = EXCLUDED.driver_id,
            fuel = EXCLUDED.fuel,
            other = EXCLUDED.other,
            loader_salaries = EXCLUDED.loader_salaries,
            revenue = EXCLUDED.revenue,
            driver_share = EXCLUDED.driver_share,
            updated_at = NOW()
        RETURNING id`

	err = tx.QueryRow(query,
		expense.OrderID,
		expense.DriverID,
		expense.Fuel,
		expense.Other,
		loaderSalariesJSON,
		expense.Revenue,
		expense.DriverShare,
	).Scan(&id)

	if err != nil {
		log.Printf("AddExpense: ошибка добавления/обновления расхода для orderID %d: %v", expense.OrderID, err)
		return 0, err
	}
	expense.ID = id
	log.Printf("Расход #%d для заказа #%d успешно добавлен/обновлен в транзакции.", id, expense.OrderID)

	// ---> НАЧАЛО ИСПОЛЬЗОВАНИЯ GetOrderStatusInTx <---
	currentOrderStatus, errStatus := GetOrderStatusInTx(tx, int64(expense.OrderID))
	if errStatus != nil {
		log.Printf("AddExpense: не удалось получить текущий статус заказа #%d для обновления на CALCULATED: %v", expense.OrderID, errStatus)
		err = fmt.Errorf("ошибка получения статуса заказа #%d: %w", expense.OrderID, errStatus)
		return 0, err
	}

	if currentOrderStatus == constants.STATUS_COMPLETED {
		errUpdate := UpdateOrderStatusInTx(tx, int64(expense.OrderID), constants.STATUS_CALCULATED)
		if errUpdate != nil {
			log.Printf("AddExpense: ошибка обновления статуса заказа #%d на CALCULATED: %v", expense.OrderID, errUpdate)
			err = fmt.Errorf("ошибка обновления статуса заказа #%d на CALCULATED: %w", expense.OrderID, errUpdate)
			return 0, err
		}
		log.Printf("Статус заказа #%d автоматически обновлен на '%s' после добавления/обновления расходов.", expense.OrderID, constants.STATUS_CALCULATED)
	}
	// ---> КОНЕЦ ИСПОЛЬЗОВАНИЯ GetOrderStatusInTx <---

	return id, nil // Если err nil, defer вызовет Commit. В противном случае - Rollback.
}

// GetExpenseByOrderID извлекает запись о расходах для конкретного заказа.
// GetExpenseByOrderID retrieves the expense record for a specific order.
func GetExpenseByOrderID(orderID int) (models.Expense, error) {
	var e models.Expense
	var loaderSalariesJSON []byte
	var createdAt sql.NullTime // Для created_at, если оно есть в таблице и нужно
	var updatedAt sql.NullTime // Для updated_at

	// Убедимся, что выбираем ID расхода, если он есть в таблице
	// Ensure we select the expense ID if it's in the table
	err := DB.QueryRow(`
        SELECT id, order_id, driver_id, fuel, other, loader_salaries, revenue, driver_share, created_at, updated_at
        FROM expenses
        WHERE order_id = $1`, orderID).Scan(
		&e.ID,
		&e.OrderID,
		&e.DriverID,
		&e.Fuel,
		&e.Other,
		&loaderSalariesJSON,
		&e.Revenue,
		&e.DriverShare,
		&createdAt, // Сканируем created_at / Scan created_at
		&updatedAt, // Сканируем updated_at / Scan updated_at
	)

	if err != nil {
		if err == sql.ErrNoRows {
			// log.Printf("GetExpenseByOrderID: запись о расходах для заказа #%d не найдена.", orderID) // Commented out to reduce log spam
			// Возвращаем "пустой" расход и ошибку ErrNoRows, чтобы вызывающий код мог это обработать
			// Return an "empty" expense and ErrNoRows error so the calling code can handle it
			return models.Expense{OrderID: orderID, LoaderSalaries: make(map[string]models.LoaderSalaryDetail)}, err
		}
		log.Printf("GetExpenseByOrderID: ошибка получения расхода для заказа #%d: %v", orderID, err)
		return e, err
	}

	// if createdAt.Valid { // Если нужно использовать created_at в модели
	// 	e.CreatedAt = createdAt.Time
	// }
	// if updatedAt.Valid { // Если нужно использовать updated_at в модели
	//  e.UpdatedAt = updatedAt.Time
	// }

	if len(loaderSalariesJSON) > 0 && string(loaderSalariesJSON) != "null" { // Проверяем, что JSON не пустой и не "null" / Check that JSON is not empty and not "null"
		if errUnmarshal := json.Unmarshal(loaderSalariesJSON, &e.LoaderSalaries); errUnmarshal != nil {
			log.Printf("GetExpenseByOrderID: ошибка демаршалинга loader_salaries для expense_id %d (order_id %d): %v. JSON: %s", e.ID, orderID, errUnmarshal, string(loaderSalariesJSON))
			// Если демаршалинг не удался, инициализируем пустой картой, чтобы избежать паники при доступе
			// If unmarshalling fails, initialize with an empty map to avoid panic on access
			e.LoaderSalaries = make(map[string]models.LoaderSalaryDetail)
			// Можно также вернуть ошибку, если это критично / Can also return an error if critical
			// return e, fmt.Errorf("ошибка данных зарплат грузчиков: %w", errUnmarshal)
		}
	} else {
		// Если JSON пустой или NULL в БД, инициализируем пустой картой
		// If JSON is empty or NULL in DB, initialize with an empty map
		e.LoaderSalaries = make(map[string]models.LoaderSalaryDetail)
	}

	return e, nil
}

// UpdateExpenseInTx обновляет существующую запись о расходах по ее ID в рамках транзакции.
// UpdateExpenseInTx updates an existing expense record by its ID within a transaction.
func UpdateExpenseInTx(tx *sql.Tx, expense models.Expense) error {
	if expense.ID == 0 {
		return fmt.Errorf("невозможно обновить расход: ID расхода не указан (равен 0)")
	}
	loaderSalariesJSON, err := json.Marshal(expense.LoaderSalaries)
	if err != nil {
		log.Printf("UpdateExpenseInTx: ошибка маршалинга loader_salaries для expense_id %d: %v", expense.ID, err)
		return fmt.Errorf("ошибка подготовки данных о зарплатах грузчиков: %w", err)
	}

	query := `
        UPDATE expenses
        SET order_id=$1, driver_id=$2, fuel=$3, other=$4, loader_salaries=$5, revenue=$6, driver_share=$7, updated_at=NOW()
        WHERE id=$8`
	result, err := tx.Exec(query,
		expense.OrderID,
		expense.DriverID,
		expense.Fuel,
		expense.Other,
		loaderSalariesJSON,
		expense.Revenue,
		expense.DriverShare,
		expense.ID,
	)
	if err != nil {
		log.Printf("UpdateExpenseInTx: ошибка обновления расхода #%d: %v", expense.ID, err)
		return err
	}
	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("расход с ID %d не найден для обновления в транзакции", expense.ID)
	}
	log.Printf("Расход #%d успешно обновлен в транзакции.", expense.ID)
	return nil
}

// MarkLoaderSalaryAsPaidByDriver отмечает зарплату грузчика как выплаченную водителем по заказу.
// Создает запись в payouts.
// MarkLoaderSalaryAsPaidByDriver marks a loader's salary as paid by the driver for an order.
// Creates a record in payouts.
func MarkLoaderSalaryAsPaidByDriver(orderID int, loaderUserID int64, driverUserID int64) error {
	tx, err := DB.Begin()
	if err != nil {
		log.Printf("MarkLoaderSalaryAsPaidByDriver: ошибка начала транзакции: %v", err)
		return err
	}
	defer tx.Rollback() // Гарантирует откат, если Commit не был вызван / Ensures rollback if Commit was not called

	// 1. Получаем текущую запись расходов для заказа
	// 1. Get the current expense record for the order
	expense, err := GetExpenseByOrderID(orderID) // Эта функция не использует транзакцию, но для чтения это обычно нормально
	if err != nil {
		if err == sql.ErrNoRows {
			log.Printf("MarkLoaderSalaryAsPaidByDriver: не найдены расходы для заказа #%d, невозможно отметить выплату.", orderID)
			return fmt.Errorf("расходы для заказа #%d не найдены", orderID)
		}
		log.Printf("MarkLoaderSalaryAsPaidByDriver: ошибка получения расходов для заказа #%d: %v", orderID, err)
		return err
	}

	// Ключ в карте LoaderSalaries - это UserID грузчика в виде строки
	// The key in the LoaderSalaries map is the loader's UserID as a string
	loaderUserIDStr := strconv.FormatInt(loaderUserID, 10)
	salaryDetail, ok := expense.LoaderSalaries[loaderUserIDStr]
	if !ok || salaryDetail.Amount <= 0 { // Изменено: <= 0, так как 0 тоже невалидная ЗП для выплаты / Changed: <= 0, as 0 is also an invalid salary for payout
		log.Printf("MarkLoaderSalaryAsPaidByDriver: ЗП для грузчика ID %s по заказу #%d не найдена или равна нулю.", loaderUserIDStr, orderID)
		return fmt.Errorf("зарплата для грузчика %s по заказу #%d не найдена или не установлена", loaderUserIDStr, orderID)
	}

	if salaryDetail.PaidByDriver {
		log.Printf("MarkLoaderSalaryAsPaidByDriver: ЗП для грузчика ID %s по заказу #%d уже была отмечена как выплаченная водителем.", loaderUserIDStr, orderID)
		return fmt.Errorf("эта зарплата уже выплачена вами") // Можно не считать ошибкой, а просто информировать / Can be considered informational rather than an error
	}

	// 2. Обновляем деталь зарплаты
	// 2. Update salary detail
	salaryDetail.PaidByDriver = true
	expense.LoaderSalaries[loaderUserIDStr] = salaryDetail

	// Используем UpdateExpenseInTx для обновления в рамках текущей транзакции
	errUpdate := UpdateExpenseInTx(tx, expense)
	if errUpdate != nil {
		log.Printf("MarkLoaderSalaryAsPaidByDriver: ошибка обновления loader_salaries через UpdateExpenseInTx для expense_id %d: %v", expense.ID, errUpdate)
		return errUpdate // Откат произойдет через defer
	}

	// 4. Создаем запись в таблице payouts
	// 4. Create a record in the payouts table
	payout := models.Payout{
		UserID:       loaderUserID, // Кому выплатили (ID грузчика) / To whom it was paid (loader's ID)
		Amount:       salaryDetail.Amount,
		PayoutDate:   time.Now(),
		OrderID:      int64(orderID), // Привязываем к заказу / Link to the order
		Comment:      fmt.Sprintf("Выплата водителем (ID: %d) грузчику (ID: %d) по заказу #%d", driverUserID, loaderUserID, orderID),
		MadeByUserID: driverUserID, // Кто выплатил (ID водителя) / Who paid (driver's ID)
	}
	_, err = addPayoutWithinTx(tx, payout) // Используем addPayoutWithinTx
	if err != nil {
		log.Printf("MarkLoaderSalaryAsPaidByDriver: ошибка создания записи о выплате для грузчика %d по заказу #%d: %v", loaderUserID, orderID, err)
		return err // tx.Rollback() будет вызван defer / tx.Rollback() will be called by defer
	}

	if err = tx.Commit(); err != nil {
		log.Printf("MarkLoaderSalaryAsPaidByDriver: ошибка коммита транзакции: %v", err)
		return err
	}

	log.Printf("Зарплата для грузчика ID %s по заказу #%d успешно отмечена как выплаченная водителем ID %d.", loaderUserIDStr, orderID, driverUserID)
	return nil
}

// DeleteExpense удаляет запись о расходах по ее ID.
// DeleteExpense deletes an expense record by its ID.
func DeleteExpense(expenseID int) error {
	result, err := DB.Exec("DELETE FROM expenses WHERE id=$1", expenseID)
	if err != nil {
		log.Printf("DeleteExpense: ошибка удаления расхода #%d: %v", expenseID, err)
		return err
	}
	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("расход с ID %d не найден для удаления", expenseID)
	}
	log.Printf("Расход #%d успешно удален.", expenseID)
	return nil
}

// GetSalariesForExcel получает данные о зарплатах (доли водителей и ЗП грузчиков из расходов)
// для Excel отчета, рассчитанные или установленные сегодня.
// GetSalariesForExcel retrieves salary data (driver shares and loader salaries from expenses)
// for an Excel report, calculated or set today.
func GetSalariesForExcel() (*sql.Rows, error) {
	query := `
(
    -- Доли водителей, где расчет (updated_at в expenses) был сегодня
    -- Driver shares where calculation (updated_at in expenses) was today
    SELECT
        u.first_name,
        u.last_name,
        u.nickname,
        u.role,
        e.driver_share AS salary_amount,
        o.id AS order_id,
        o.date AS order_date,
        e.updated_at AS calculation_or_payout_date, -- Используем updated_at из expenses как дату расчета
        'driver_share' AS salary_type,
        u.card_number AS staff_card_number -- Добавлено поле для номера карты сотрудника
    FROM expenses e
    JOIN users u ON e.driver_id = u.id
    JOIN orders o ON e.order_id = o.id
    WHERE
        u.role = 'driver' AND
        e.driver_share > 0 AND
        date_trunc('day', e.updated_at) = date_trunc('day', CURRENT_TIMESTAMP)
)
UNION ALL
(
    -- Зарплаты грузчиков, где ЗП была установлена/изменена (updated_at в expenses) сегодня
    -- Loader salaries where salary was set/changed (updated_at in expenses) today
    SELECT
        u.first_name,
        u.last_name,
        u.nickname,
        u.role,
        (ls.value ->> 'amount')::float AS salary_amount,
        o.id AS order_id,
        o.date AS order_date,
        e.updated_at AS calculation_or_payout_date, -- Используем updated_at из expenses
        'loader_salary' AS salary_type,
        u.card_number AS staff_card_number -- Добавлено поле для номера карты сотрудника
    FROM expenses e
    JOIN orders o ON e.order_id = o.id,
    LATERAL jsonb_each(e.loader_salaries) ls -- Используем LATERAL jsonb_each для разбора JSONB
    JOIN users u ON u.id = (ls.key)::bigint -- Ключ в loader_salaries - это user_id грузчика
    WHERE
        u.role = 'loader' AND
        (ls.value ->> 'amount')::float > 0 AND
        date_trunc('day', e.updated_at) = date_trunc('day', CURRENT_TIMESTAMP)
)
ORDER BY calculation_or_payout_date DESC;
`
	rows, err := DB.Query(query)
	if err != nil {
		log.Printf("GetSalariesForExcel: ошибка получения данных о зарплатах: %v", err)
		return nil, err
	}
	log.Println("GetSalariesForExcel: Данные для отчета по зарплатам успешно получены.")
	return rows, nil
}
