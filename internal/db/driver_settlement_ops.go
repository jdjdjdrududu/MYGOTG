package db

import (
	"Original/internal/constants" // Убедитесь, что этот импорт есть
	"Original/internal/models"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/lib/pq"
)

// --- НАЧАЛО НОВОЙ ФУНКЦИИ ---
// UpdateDriverSettlementStatus обновляет статус и комментарий отчета.
func UpdateDriverSettlementStatus(settlementID int64, status string, comment sql.NullString) error {
	query := `UPDATE driver_settlements SET status = $1, admin_comment = $2, updated_at = NOW() WHERE id = $3`
	result, err := DB.Exec(query, status, comment, settlementID)
	if err != nil {
		log.Printf("UpdateDriverSettlementStatus: ошибка обновления статуса для отчета #%d: %v", settlementID, err)
		return err
	}
	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("отчет #%d не найден для обновления статуса", settlementID)
	}
	log.Printf("Статус отчета #%d обновлен на '%s'.", settlementID, status)
	return nil
}

// --- КОНЕЦ НОВОЙ ФУНКЦИИ ---

// MarkSettlementAsPaidToOwnerInTx sets the paid_to_owner_at timestamp for a settlement within a transaction.
func MarkSettlementAsPaidToOwnerInTx(tx *sql.Tx, settlementID int64) error {
	query := `UPDATE driver_settlements SET paid_to_owner_at = NOW(), updated_at = NOW() WHERE id = $1 AND paid_to_owner_at IS NULL`
	result, err := tx.Exec(query, settlementID)
	if err != nil {
		log.Printf("MarkSettlementAsPaidToOwnerInTx: ошибка обновления отчета #%d: %v", settlementID, err)
		return err
	}
	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		var existsAndPaid bool
		checkQuery := `SELECT EXISTS(SELECT 1 FROM driver_settlements WHERE id = $1 AND paid_to_owner_at IS NOT NULL)`
		errCheck := tx.QueryRow(checkQuery, settlementID).Scan(&existsAndPaid)
		if errCheck == nil && existsAndPaid {
			log.Printf("MarkSettlementAsPaidToOwnerInTx: Отчет #%d уже был помечен как оплаченный (в транзакции).", settlementID)
			return nil
		}
		return fmt.Errorf("отчет #%d не найден или уже помечен как оплаченный (в транзакции)", settlementID)
	}
	log.Printf("Отчет #%d помечен как оплаченный владельцу (в транзакции).", settlementID)
	return nil
}

// MarkDriverSalaryAsPaidInTx устанавливает время выплаты ЗП водителю по отчету в рамках транзакции.
func MarkDriverSalaryAsPaidInTx(tx *sql.Tx, settlementID int64) error {
	query := `UPDATE driver_settlements SET driver_salary_paid_at = NOW(), updated_at = NOW() WHERE id = $1 AND driver_salary_paid_at IS NULL`
	result, err := tx.Exec(query, settlementID)
	if err != nil {
		log.Printf("MarkDriverSalaryAsPaidInTx: ошибка обновления отчета #%d: %v", settlementID, err)
		return err
	}
	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		var existsAndPaid bool
		checkQuery := `SELECT EXISTS(SELECT 1 FROM driver_settlements WHERE id = $1 AND driver_salary_paid_at IS NOT NULL)`
		errCheck := tx.QueryRow(checkQuery, settlementID).Scan(&existsAndPaid)
		if errCheck == nil && existsAndPaid {
			log.Printf("MarkDriverSalaryAsPaidInTx: ЗП по отчету #%d уже была помечена как выплаченная (в транзакции).", settlementID)
			return nil
		}
		return fmt.Errorf("отчет #%d не найден или ЗП уже помечена как выплаченная (в транзакции)", settlementID)
	}
	log.Printf("ЗП по отчету #%d помечена как выплаченная водителю (в транзакции).", settlementID)
	return nil
}

// CheckAndSettleOrdersForSettlement проверяет условия по отчету и обновляет статусы заказов.
func CheckAndSettleOrdersForSettlement(tx *sql.Tx, settlementID int64) error {
	var paidToOwnerAt sql.NullTime
	var driverSalaryPaidAt sql.NullTime
	var coveredOrderIDs pq.Int64Array

	query := `SELECT paid_to_owner_at, driver_salary_paid_at, covered_order_ids FROM driver_settlements WHERE id = $1`
	err := tx.QueryRow(query, settlementID).Scan(&paidToOwnerAt, &driverSalaryPaidAt, &coveredOrderIDs)
	if err != nil {
		log.Printf("CheckAndSettleOrdersForSettlement: ошибка получения данных отчета #%d: %v", settlementID, err)
		return fmt.Errorf("ошибка получения данных отчета #%d: %w", settlementID, err)
	}

	if paidToOwnerAt.Valid && driverSalaryPaidAt.Valid {
		log.Printf("CheckAndSettleOrdersForSettlement: Оба условия для отчета #%d выполнены. Проверка заказов (цель статуса: %s): %v", settlementID, constants.STATUS_CALCULATED, coveredOrderIDs)
		if len(coveredOrderIDs) > 0 {
			for _, orderID := range coveredOrderIDs {
				currentStatus, errStatus := GetOrderStatusInTx(tx, orderID)
				if errStatus != nil {
					log.Printf("CheckAndSettleOrdersForSettlement: ошибка получения статуса заказа #%d для отчета #%d: %v", orderID, settlementID, errStatus)
					continue
				}
				if currentStatus == constants.STATUS_COMPLETED {
					errUpdate := UpdateOrderStatusInTx(tx, orderID, constants.STATUS_CALCULATED)
					if errUpdate != nil {
						log.Printf("CheckAndSettleOrdersForSettlement: ошибка обновления статуса заказа #%d из '%s' на '%s': %v", orderID, currentStatus, constants.STATUS_CALCULATED, errUpdate)
					} else {
						log.Printf("CheckAndSettleOrdersForSettlement: Статус заказа #%d (был '%s') успешно обновлен на '%s' по отчету #%d.", orderID, currentStatus, constants.STATUS_CALCULATED, settlementID)
					}
				} else if currentStatus == constants.STATUS_CALCULATED {
					log.Printf("CheckAndSettleOrdersForSettlement: Заказ #%d (отчет #%d) уже в статусе '%s'. Обновление статуса не требуется.", orderID, settlementID, currentStatus)
				} else {
					log.Printf("CheckAndSettleOrdersForSettlement: Заказ #%d (отчет #%d) не в статусе '%s' (текущий статус: '%s'), обновление до '%s' не производится этим механизмом.", orderID, settlementID, constants.STATUS_COMPLETED, currentStatus, constants.STATUS_CALCULATED)
				}
			}
		}
	} else {
		log.Printf("CheckAndSettleOrdersForSettlement: Для отчета #%d условия для перевода заказов в '%s' не выполнены (paid_to_owner_at: %v, driver_salary_paid_at: %v). Обновление заказов не требуется.", settlementID, constants.STATUS_CALCULATED, paidToOwnerAt.Valid, driverSalaryPaidAt.Valid)
	}
	return nil
}

// MarkSettlementAsPaidToOwner sets the paid_to_owner_at timestamp.
func MarkSettlementAsPaidToOwner(settlementID int64) error {
	tx, err := DB.Begin()
	if err != nil {
		log.Printf("MarkSettlementAsPaidToOwner: ошибка начала транзакции: %v", err)
		return err
	}
	var opErr error
	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			panic(p)
		} else if opErr != nil {
			tx.Rollback()
		} else {
			opErr = tx.Commit()
			if opErr != nil {
				log.Printf("MarkSettlementAsPaidToOwner: ошибка коммита транзакции: %v", opErr)
			}
		}
	}()

	opErr = MarkSettlementAsPaidToOwnerInTx(tx, settlementID)
	if opErr != nil {
		return opErr
	}
	opErr = CheckAndSettleOrdersForSettlement(tx, settlementID)
	if opErr != nil {
		return opErr
	}
	return opErr
}

// MarkDriverSalaryAsPaid устанавливает время выплаты ЗП водителю.
func MarkDriverSalaryAsPaid(settlementID int64) error {
	tx, err := DB.Begin()
	if err != nil {
		log.Printf("MarkDriverSalaryAsPaid: ошибка начала транзакции: %v", err)
		return err
	}
	var opErr error
	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			panic(p)
		} else if opErr != nil {
			tx.Rollback()
		} else {
			opErr = tx.Commit()
			if opErr != nil {
				log.Printf("MarkDriverSalaryAsPaid: ошибка коммита транзакции: %v", opErr)
			}
		}
	}()

	opErr = MarkDriverSalaryAsPaidInTx(tx, settlementID)
	if opErr != nil {
		return opErr
	}
	opErr = CheckAndSettleOrdersForSettlement(tx, settlementID)
	if opErr != nil {
		return opErr
	}
	return opErr
}

// AddDriverSettlement добавляет новый отчет водителя.
func AddDriverSettlement(settlement models.DriverSettlement) (int64, error) {
	tx, err := DB.Begin()
	if err != nil {
		log.Printf("AddDriverSettlement: ошибка начала транзакции: %v", err)
		return 0, err
	}
	var opErr error
	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			panic(p)
		} else if opErr != nil {
			tx.Rollback()
			log.Printf("AddDriverSettlement: откат транзакции из-за ошибки: %v", opErr)
		} else {
			opErr = tx.Commit()
			if opErr != nil {
				log.Printf("AddDriverSettlement: ошибка коммита транзакции: %v", opErr)
			}
		}
	}()

	loaderPaymentsJSONBytes, errMarshal := json.Marshal(settlement.LoaderPayments)
	if errMarshal != nil {
		opErr = fmt.Errorf("ошибка маршалинга loader_payments: %w", errMarshal)
		log.Printf("AddDriverSettlement: %v", opErr)
		return 0, opErr
	}
	loaderPaymentsJSON := sql.NullString{String: string(loaderPaymentsJSONBytes), Valid: len(settlement.LoaderPayments) > 0}

	otherExpensesJSONBytes, errMarshalOther := json.Marshal(settlement.OtherExpenses)
	if errMarshalOther != nil {
		opErr = fmt.Errorf("ошибка маршалинга other_expenses: %w", errMarshalOther)
		log.Printf("AddDriverSettlement: %v", opErr)
		return 0, opErr
	}
	otherExpensesJSON := sql.NullString{String: string(otherExpensesJSONBytes), Valid: len(settlement.OtherExpenses) > 0}

	settlement.ReportDate = time.Date(
		settlement.SettlementTimestamp.Year(),
		settlement.SettlementTimestamp.Month(),
		settlement.SettlementTimestamp.Day(),
		0, 0, 0, 0, settlement.SettlementTimestamp.Location(),
	)
	if settlement.CreatedAt.IsZero() {
		settlement.CreatedAt = time.Now()
	}
	if settlement.UpdatedAt.IsZero() {
		settlement.UpdatedAt = time.Now()
	}

	query := `
        INSERT INTO driver_settlements (
            driver_user_id, report_date, settlement_timestamp,
            covered_orders_revenue, fuel_expense, other_expenses_json, loader_payments_json,
            driver_calculated_salary, amount_to_cashier, covered_orders_count,
            created_at, updated_at, covered_order_ids, paid_to_owner_at, driver_salary_paid_at,
            status, admin_comment
        ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, NULL, NULL, $14, NULL)
        RETURNING id`
	var id int64
	opErr = tx.QueryRow(query,
		settlement.DriverUserID, settlement.ReportDate, settlement.SettlementTimestamp,
		settlement.CoveredOrdersRevenue, settlement.FuelExpense, otherExpensesJSON, loaderPaymentsJSON,
		settlement.DriverCalculatedSalary, settlement.AmountToCashier, settlement.CoveredOrdersCount,
		settlement.CreatedAt, settlement.UpdatedAt, pq.Array(settlement.CoveredOrderIDs),
		constants.SETTLEMENT_STATUS_PENDING,
	).Scan(&id)

	if opErr != nil {
		log.Printf("AddDriverSettlement: ошибка добавления отчета для водителя %d: %v", settlement.DriverUserID, opErr)
		return 0, opErr
	}

	opErr = MarkOrdersAsSettled(tx, settlement.CoveredOrderIDs)
	if opErr != nil {
		log.Printf("AddDriverSettlement: ошибка MarkOrdersAsSettled: %v", opErr)
		return 0, opErr
	}

	return id, opErr
}

// GetDriverSettlementsByDriverAndDate получает все отчеты водителя за указанную дату.
func GetDriverSettlementsByDriverAndDate(driverUserID int64, reportDate time.Time) ([]models.DriverSettlement, error) {
	query := `
		SELECT id, driver_user_id, report_date, settlement_timestamp,
		       covered_orders_revenue, fuel_expense, other_expenses_json, loader_payments_json,
		       driver_calculated_salary, amount_to_cashier, covered_orders_count,
		       created_at, updated_at, covered_order_ids, paid_to_owner_at, driver_salary_paid_at
		FROM driver_settlements
		WHERE driver_user_id = $1 AND report_date = $2
		ORDER BY settlement_timestamp ASC`

	rows, err := DB.Query(query, driverUserID, reportDate.Format("2006-01-02"))
	if err != nil {
		log.Printf("GetDriverSettlementsByDriverAndDate: ошибка получения отчетов для водителя %d за %s: %v", driverUserID, reportDate.Format("2006-01-02"), err)
		return nil, err
	}
	defer rows.Close()

	var settlements []models.DriverSettlement
	for rows.Next() {
		var s models.DriverSettlement
		var loaderPaymentsJSON sql.NullString
		var otherExpensesJSON sql.NullString
		var coveredOrderIDs pq.Int64Array

		errScan := rows.Scan(
			&s.ID, &s.DriverUserID, &s.ReportDate, &s.SettlementTimestamp,
			&s.CoveredOrdersRevenue, &s.FuelExpense, &otherExpensesJSON, &loaderPaymentsJSON,
			&s.DriverCalculatedSalary, &s.AmountToCashier, &s.CoveredOrdersCount,
			&s.CreatedAt, &s.UpdatedAt, &coveredOrderIDs, &s.PaidToOwnerAt, &s.DriverSalaryPaidAt,
		)
		if errScan != nil {
			log.Printf("GetDriverSettlementsByDriverAndDate: ошибка сканирования отчета: %v", errScan)
			continue
		}
		s.CoveredOrderIDs = []int64(coveredOrderIDs)

		if loaderPaymentsJSON.Valid && loaderPaymentsJSON.String != "" && loaderPaymentsJSON.String != "null" {
			if errUnmarshal := json.Unmarshal([]byte(loaderPaymentsJSON.String), &s.LoaderPayments); errUnmarshal != nil {
				log.Printf("GetDriverSettlementsByDriverAndDate: ошибка демаршалинга loader_payments для отчета #%d: %v. JSON: %s", s.ID, errUnmarshal, loaderPaymentsJSON.String)
				s.LoaderPayments = []models.LoaderPaymentDetail{}
			}
		} else {
			s.LoaderPayments = []models.LoaderPaymentDetail{}
		}

		if otherExpensesJSON.Valid && otherExpensesJSON.String != "" && otherExpensesJSON.String != "null" {
			if errUnmarshal := json.Unmarshal([]byte(otherExpensesJSON.String), &s.OtherExpenses); errUnmarshal != nil {
				log.Printf("GetDriverSettlementsByDriverAndDate: ошибка демаршалинга other_expenses для отчета #%d: %v. JSON: %s", s.ID, errUnmarshal, otherExpensesJSON.String)
				s.OtherExpenses = []models.OtherExpenseDetail{}
			}
		} else {
			s.OtherExpenses = []models.OtherExpenseDetail{}
		}
		settlements = append(settlements, s)
	}
	return settlements, rows.Err()
}

// GetAggregatedCashierRecordsForOwner (старая логика) - без изменений, т.к. не затрагивает other_expense.
func GetAggregatedCashierRecordsForOwner(targetDate time.Time) ([]models.OwnerCashierRecord, error) {
	query := `
        SELECT
            ds.driver_user_id,
            u.first_name || ' ' || COALESCE(u.last_name, '') AS driver_name,
            ds.report_date,
            SUM(ds.amount_to_cashier) AS total_amount_due,
            array_agg(ds.id ORDER BY ds.settlement_timestamp) AS contributing_sett_ids,
            MAX(ds.updated_at) AS last_updated_at
        FROM driver_settlements ds
        JOIN users u ON ds.driver_user_id = u.id
        WHERE ds.report_date = $1
        GROUP BY ds.driver_user_id, driver_name, ds.report_date
        ORDER BY driver_name, ds.report_date;
    `
	rows, err := DB.Query(query, targetDate.Format("2006-01-02"))
	if err != nil {
		log.Printf("GetAggregatedCashierRecordsForOwner: ошибка получения агрегированных данных за %s: %v", targetDate.Format("2006-01-02"), err)
		return nil, err
	}
	defer rows.Close()

	var records []models.OwnerCashierRecord
	for rows.Next() {
		var r models.OwnerCashierRecord
		var contributingSettIDs pq.Int64Array
		errScan := rows.Scan(
			&r.DriverUserID,
			&r.DriverName,
			&r.ReportDate,
			&r.TotalAmountDue,
			&contributingSettIDs,
			&r.LastUpdatedAt,
		)
		if errScan != nil {
			log.Printf("GetAggregatedCashierRecordsForOwner: ошибка сканирования записи: %v", errScan)
			continue
		}
		r.ContributingSettIDs = []int64(contributingSettIDs)
		records = append(records, r)
	}
	return records, rows.Err()
}

// GetDriverSettlementByID получает один отчет водителя по ID.
func GetDriverSettlementByID(settlementID int64) (models.DriverSettlement, error) {
	var s models.DriverSettlement
	var loaderPaymentsJSON sql.NullString
	var otherExpensesJSON sql.NullString
	var coveredOrderIDs pq.Int64Array

	query := `
		SELECT id, driver_user_id, report_date, settlement_timestamp,
		       covered_orders_revenue, fuel_expense, other_expenses_json, loader_payments_json,
		       driver_calculated_salary, amount_to_cashier, covered_orders_count,
		       created_at, updated_at, covered_order_ids, paid_to_owner_at, driver_salary_paid_at,
		       status, admin_comment
		FROM driver_settlements
		WHERE id = $1`

	err := DB.QueryRow(query, settlementID).Scan(
		&s.ID, &s.DriverUserID, &s.ReportDate, &s.SettlementTimestamp,
		&s.CoveredOrdersRevenue, &s.FuelExpense, &otherExpensesJSON, &loaderPaymentsJSON,
		&s.DriverCalculatedSalary, &s.AmountToCashier, &s.CoveredOrdersCount,
		&s.CreatedAt, &s.UpdatedAt, &coveredOrderIDs, &s.PaidToOwnerAt, &s.DriverSalaryPaidAt,
		&s.Status, &s.AdminComment,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return s, fmt.Errorf("отчет водителя с ID %d не найден", settlementID)
		}
		log.Printf("GetDriverSettlementByID: ошибка получения отчета #%d: %v", settlementID, err)
		return s, err
	}

	s.CoveredOrderIDs = []int64(coveredOrderIDs)

	if loaderPaymentsJSON.Valid && loaderPaymentsJSON.String != "" && loaderPaymentsJSON.String != "null" {
		if errUnmarshal := json.Unmarshal([]byte(loaderPaymentsJSON.String), &s.LoaderPayments); errUnmarshal != nil {
			log.Printf("GetDriverSettlementByID: ошибка демаршалинга loader_payments для отчета #%d: %v. JSON: %s", s.ID, errUnmarshal, loaderPaymentsJSON.String)
			s.LoaderPayments = []models.LoaderPaymentDetail{}
		}
	} else {
		s.LoaderPayments = []models.LoaderPaymentDetail{}
	}

	if otherExpensesJSON.Valid && otherExpensesJSON.String != "" && otherExpensesJSON.String != "null" {
		if errUnmarshal := json.Unmarshal([]byte(otherExpensesJSON.String), &s.OtherExpenses); errUnmarshal != nil {
			log.Printf("GetDriverSettlementByID: ошибка демаршалинга other_expenses для отчета #%d: %v. JSON: %s", s.ID, errUnmarshal, otherExpensesJSON.String)
			s.OtherExpenses = []models.OtherExpenseDetail{}
		}
	} else {
		s.OtherExpenses = []models.OtherExpenseDetail{}
	}
	return s, nil
}

// UpdateDriverSettlement обновляет существующий отчет водителя.
func UpdateDriverSettlement(settlement models.DriverSettlement) error {
	if settlement.ID == 0 {
		return fmt.Errorf("ID отчета не может быть 0 для обновления")
	}
	loaderPaymentsJSONBytes, errMarshalL := json.Marshal(settlement.LoaderPayments)
	if errMarshalL != nil {
		return fmt.Errorf("ошибка подготовки данных о зарплатах грузчиков: %w", errMarshalL)
	}
	loaderPaymentsJSON := sql.NullString{String: string(loaderPaymentsJSONBytes), Valid: len(settlement.LoaderPayments) > 0}

	otherExpensesJSONBytes, errMarshalO := json.Marshal(settlement.OtherExpenses)
	if errMarshalO != nil {
		return fmt.Errorf("ошибка подготовки данных о прочих расходах: %w", errMarshalO)
	}
	otherExpensesJSON := sql.NullString{String: string(otherExpensesJSONBytes), Valid: len(settlement.OtherExpenses) > 0}

	if settlement.SettlementTimestamp.IsZero() {
		log.Printf("UpdateDriverSettlement: ВНИМАНИЕ! SettlementTimestamp не установлен для отчета #%d. ReportDate может быть неверным.", settlement.ID)
		if settlement.ReportDate.IsZero() {
			return fmt.Errorf("SettlementTimestamp или ReportDate должны быть установлены для обновления отчета #%d", settlement.ID)
		}
	} else {
		settlement.ReportDate = time.Date(
			settlement.SettlementTimestamp.Year(),
			settlement.SettlementTimestamp.Month(),
			settlement.SettlementTimestamp.Day(),
			0, 0, 0, 0, settlement.SettlementTimestamp.Location(),
		)
	}
	settlement.UpdatedAt = time.Now()

	driverSharePercentage := 0.35 // ЗАМЕНИТЬ НА РЕАЛЬНОЕ ЗНАЧЕНИЕ ИЗ CONFIG
	netForDriver := settlement.CoveredOrdersRevenue - settlement.FuelExpense

	totalOtherExpenses := 0.0
	for _, oe := range settlement.OtherExpenses {
		totalOtherExpenses += oe.Amount
	}
	netForDriver -= totalOtherExpenses

	totalLoaderSalary := 0.0
	for _, lp := range settlement.LoaderPayments {
		totalLoaderSalary += lp.Amount
	}
	netForDriver -= totalLoaderSalary
	settlement.DriverCalculatedSalary = netForDriver * driverSharePercentage
	settlement.AmountToCashier = netForDriver - settlement.DriverCalculatedSalary

	query := `
		UPDATE driver_settlements SET
			driver_user_id = $1,
			report_date = $2,
			settlement_timestamp = $3,
			covered_orders_revenue = $4,
			fuel_expense = $5,
			other_expenses_json = $6, 
			loader_payments_json = $7,
			driver_calculated_salary = $8,
			amount_to_cashier = $9,
			covered_orders_count = $10,
			covered_order_ids = $11,
			paid_to_owner_at = $12,
			driver_salary_paid_at = $13,
			updated_at = $14
		WHERE id = $15`

	result, err := DB.Exec(query,
		settlement.DriverUserID,
		settlement.ReportDate,
		settlement.SettlementTimestamp,
		settlement.CoveredOrdersRevenue,
		settlement.FuelExpense,
		otherExpensesJSON,
		loaderPaymentsJSON,
		settlement.DriverCalculatedSalary,
		settlement.AmountToCashier,
		settlement.CoveredOrdersCount,
		pq.Array(settlement.CoveredOrderIDs),
		settlement.PaidToOwnerAt,
		settlement.DriverSalaryPaidAt,
		settlement.UpdatedAt,
		settlement.ID,
	)
	if err != nil {
		log.Printf("UpdateDriverSettlement: ошибка обновления отчета #%d: %v", settlement.ID, err)
		return err
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("отчет #%d не найден для обновления", settlement.ID)
	}

	log.Printf("Отчет водителя #%d успешно обновлен. ReportDate: %s, AmountToCashier: %.0f", settlement.ID, settlement.ReportDate.Format("2006-01-02"), settlement.AmountToCashier)
	return nil
}

// GetAggregatedDriverSettlements получает агрегированные данные по отчетам водителей.
func GetAggregatedDriverSettlements(viewType string) ([]models.AggregatedDriverSettlementInfo, int, error) {
	var totalDrivers int
	var rows *sql.Rows
	var err error

	countQueryBase := `SELECT COUNT(DISTINCT driver_user_id) FROM driver_settlements ds WHERE ds.amount_to_cashier > 0 `
	queryBase := `
		SELECT
		    ds.driver_user_id,
		    u.first_name AS driver_first_name,
		    u.last_name AS driver_last_name,
		    u.nickname AS driver_nickname,
		    SUM(ds.amount_to_cashier) AS total_amount_to_cashier,
		    COUNT(ds.id) AS total_reports_count
		FROM driver_settlements ds
		JOIN users u ON ds.driver_user_id = u.id
		WHERE ds.amount_to_cashier > 0 `

	var orderByClause string
	var whereConditions string

	if viewType == "actual" {
		whereConditions = "AND (ds.paid_to_owner_at IS NULL OR ds.driver_salary_paid_at IS NULL) "
		orderByClause = "ORDER BY MIN(ds.settlement_timestamp) ASC, driver_first_name, driver_last_name"
	} else {
		whereConditions = "AND (ds.paid_to_owner_at IS NOT NULL AND ds.driver_salary_paid_at IS NOT NULL) "
		orderByClause = "ORDER BY MAX(CASE WHEN ds.paid_to_owner_at IS NOT NULL THEN ds.paid_to_owner_at ELSE ds.driver_salary_paid_at END) DESC, driver_first_name, driver_last_name"
	}

	finalCountQuery := countQueryBase + whereConditions
	err = DB.QueryRow(finalCountQuery).Scan(&totalDrivers)
	if err != nil {
		log.Printf("GetAggregatedDriverSettlements: ошибка подсчета водителей (viewType: %s): %v", viewType, err)
		return nil, 0, err
	}

	fullQuery := queryBase + whereConditions + "GROUP BY ds.driver_user_id, u.first_name, u.last_name, u.nickname " + orderByClause
	rows, err = DB.Query(fullQuery)
	if err != nil {
		log.Printf("GetAggregatedDriverSettlements: ошибка получения агрегированных данных (viewType: %s): %v", viewType, err)
		return nil, 0, err
	}
	defer rows.Close()

	var aggregatedInfos []models.AggregatedDriverSettlementInfo
	for rows.Next() {
		var aggInfo models.AggregatedDriverSettlementInfo
		errScan := rows.Scan(
			&aggInfo.DriverUserID,
			&aggInfo.DriverFirstName,
			&aggInfo.DriverLastName,
			&aggInfo.DriverNickname,
			&aggInfo.TotalAmountToCashier,
			&aggInfo.TotalReportsCount,
		)
		if errScan != nil {
			log.Printf("GetAggregatedDriverSettlements: ошибка сканирования агрегированного отчета (viewType: %s): %v", viewType, errScan)
			continue
		}
		aggregatedInfos = append(aggregatedInfos, aggInfo)
	}
	return aggregatedInfos, totalDrivers, rows.Err()
}

// GetActualDebts and GetSettledDebts (без изменений)
func GetActualDebts(page int, perPage int) ([]models.AggregatedDriverSettlementInfo, int, error) {
	allAggregated, totalDrivers, err := GetAggregatedDriverSettlements("actual")
	if err != nil {
		return nil, 0, err
	}
	start := page * perPage
	end := start + perPage
	var paginatedData []models.AggregatedDriverSettlementInfo
	if start >= len(allAggregated) {
		paginatedData = []models.AggregatedDriverSettlementInfo{}
	} else if end > len(allAggregated) {
		paginatedData = allAggregated[start:]
	} else {
		paginatedData = allAggregated[start:end]
	}
	return paginatedData, totalDrivers, nil
}

func GetSettledDebts(page int, perPage int) ([]models.AggregatedDriverSettlementInfo, int, error) {
	allAggregated, totalDrivers, err := GetAggregatedDriverSettlements("settled")
	if err != nil {
		return nil, 0, err
	}
	start := page * perPage
	end := start + perPage
	var paginatedData []models.AggregatedDriverSettlementInfo
	if start >= len(allAggregated) {
		paginatedData = []models.AggregatedDriverSettlementInfo{}
	} else if end > len(allAggregated) {
		paginatedData = allAggregated[start:]
	} else {
		paginatedData = allAggregated[start:end]
	}
	return paginatedData, totalDrivers, nil
}

// GetDriverSettlementsForOwnerView - отображает список индивидуальных отчетов водителя.
func GetDriverSettlementsForOwnerView(driverUserID int64, viewType string, page int, perPage int) ([]models.DriverSettlementWithDriverName, int, error) {
	offset := page * perPage
	var totalRecords int
	var countQuery string
	var queryParams []interface{}

	baseQuery := `
		SELECT
		    ds.id, ds.driver_user_id, ds.report_date, ds.settlement_timestamp,
		    ds.covered_orders_revenue, ds.fuel_expense, ds.other_expenses_json, ds.loader_payments_json,
		    ds.driver_calculated_salary, ds.amount_to_cashier, ds.covered_orders_count,
		    ds.created_at, ds.updated_at, ds.covered_order_ids, ds.paid_to_owner_at, ds.driver_salary_paid_at,
		    u.first_name AS driver_first_name, u.last_name AS driver_last_name, u.nickname AS driver_nickname
		FROM driver_settlements ds
		JOIN users u ON ds.driver_user_id = u.id
		WHERE ds.driver_user_id = $1 AND ds.amount_to_cashier > 0 `

	baseCountQuery := `SELECT COUNT(*) FROM driver_settlements ds WHERE ds.driver_user_id = $1 AND ds.amount_to_cashier > 0 `
	queryParams = append(queryParams, driverUserID)

	var whereConditions string
	if viewType == "actual" {
		whereConditions = "AND (ds.paid_to_owner_at IS NULL OR ds.driver_salary_paid_at IS NULL) "
	} else {
		whereConditions = "AND (ds.paid_to_owner_at IS NOT NULL AND ds.driver_salary_paid_at IS NOT NULL) "
	}

	countQuery = baseCountQuery + whereConditions
	err := DB.QueryRow(countQuery, driverUserID).Scan(&totalRecords)
	if err != nil {
		log.Printf("GetDriverSettlementsForOwnerView: ошибка подсчета записей для водителя %d (viewType: %s): %v", driverUserID, viewType, err)
		return nil, 0, err
	}

	query := baseQuery + whereConditions + "ORDER BY ds.settlement_timestamp DESC LIMIT $2 OFFSET $3"
	queryParams = append(queryParams, perPage, offset)

	rows, err := DB.Query(query, queryParams...)
	if err != nil {
		log.Printf("GetDriverSettlementsForOwnerView: ошибка получения отчетов для водителя %d (viewType: %s): %v", driverUserID, viewType, err)
		return nil, 0, err
	}
	defer rows.Close()

	var settlements []models.DriverSettlementWithDriverName
	for rows.Next() {
		var s models.DriverSettlementWithDriverName
		var loaderPaymentsJSON sql.NullString
		var otherExpensesJSON sql.NullString
		var coveredOrderIDs pq.Int64Array

		errScan := rows.Scan(
			&s.ID, &s.DriverUserID, &s.ReportDate, &s.SettlementTimestamp,
			&s.CoveredOrdersRevenue, &s.FuelExpense, &otherExpensesJSON, &loaderPaymentsJSON,
			&s.DriverCalculatedSalary, &s.AmountToCashier, &s.CoveredOrdersCount,
			&s.CreatedAt, &s.UpdatedAt, &coveredOrderIDs, &s.PaidToOwnerAt, &s.DriverSalaryPaidAt,
			&s.DriverFirstName, &s.DriverLastName, &s.DriverNickname,
		)
		if errScan != nil {
			log.Printf("GetDriverSettlementsForOwnerView: ошибка сканирования отчета: %v", errScan)
			continue
		}
		s.CoveredOrderIDs = []int64(coveredOrderIDs)

		if loaderPaymentsJSON.Valid && loaderPaymentsJSON.String != "" && loaderPaymentsJSON.String != "null" {
			if errUnmarshal := json.Unmarshal([]byte(loaderPaymentsJSON.String), &s.LoaderPayments); errUnmarshal != nil {
				log.Printf("GetDriverSettlementsForOwnerView: ошибка демаршалинга loader_payments для отчета #%d: %v. JSON: %s", s.ID, errUnmarshal, loaderPaymentsJSON.String)
				s.LoaderPayments = []models.LoaderPaymentDetail{}
			}
		} else {
			s.LoaderPayments = []models.LoaderPaymentDetail{}
		}

		if otherExpensesJSON.Valid && otherExpensesJSON.String != "" && otherExpensesJSON.String != "null" {
			if errUnmarshal := json.Unmarshal([]byte(otherExpensesJSON.String), &s.OtherExpenses); errUnmarshal != nil {
				log.Printf("GetDriverSettlementsForOwnerView: ошибка демаршалинга other_expenses для отчета #%d: %v. JSON: %s", s.ID, errUnmarshal, otherExpensesJSON.String)
				s.OtherExpenses = []models.OtherExpenseDetail{}
			}
		} else {
			s.OtherExpenses = []models.OtherExpenseDetail{}
		}

		settlements = append(settlements, s)
	}
	return settlements, totalRecords, rows.Err()
}

// MarkSettlementAsUnpaidToOwner снимает отметку о внесении денег.
func MarkSettlementAsUnpaidToOwner(settlementID int64) error {
	tx, err := DB.Begin()
	if err != nil {
		log.Printf("MarkSettlementAsUnpaidToOwner: ошибка начала транзакции: %v", err)
		return err
	}
	var opErr error
	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			panic(p)
		} else if opErr != nil {
			tx.Rollback()
		} else {
			opErr = tx.Commit()
			if opErr != nil {
				log.Printf("MarkSettlementAsUnpaidToOwner: ошибка коммита транзакции: %v", opErr)
			}
		}
	}()

	query := `UPDATE driver_settlements SET paid_to_owner_at = NULL, updated_at = NOW() WHERE id = $1 AND paid_to_owner_at IS NOT NULL`
	result, opErr := tx.Exec(query, settlementID)
	if opErr != nil {
		log.Printf("MarkSettlementAsUnpaidToOwner: ошибка обновления отчета #%d: %v", settlementID, opErr)
		return opErr
	}
	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		var existsAndUnpaid bool
		checkQuery := `SELECT EXISTS(SELECT 1 FROM driver_settlements WHERE id = $1 AND paid_to_owner_at IS NULL)`
		errCheck := tx.QueryRow(checkQuery, settlementID).Scan(&existsAndUnpaid)
		if errCheck == nil && existsAndUnpaid {
			log.Printf("MarkSettlementAsUnpaidToOwner: Отчет #%d уже был помечен как 'деньги не внесены'.", settlementID)
			return nil
		}
		opErr = fmt.Errorf("отчет #%d не найден или 'деньги не внесены' уже было отмечено", settlementID)
		return opErr
	}

	var coveredOrderIDs pq.Int64Array
	errFetchOrders := tx.QueryRow("SELECT covered_order_ids FROM driver_settlements WHERE id = $1", settlementID).Scan(&coveredOrderIDs)
	if errFetchOrders != nil {
		log.Printf("MarkSettlementAsUnpaidToOwner: Не удалось получить ID заказов для отчета #%d: %v", settlementID, errFetchOrders)
	} else {
		if len(coveredOrderIDs) > 0 {
			revertQuery := `UPDATE orders SET status = $1, updated_at = NOW() WHERE id = ANY($2::bigint[]) AND status = $3`
			_, errRevert := tx.Exec(revertQuery, constants.STATUS_COMPLETED, coveredOrderIDs, constants.STATUS_CALCULATED)
			if errRevert != nil {
				log.Printf("MarkSettlementAsUnpaidToOwner: Ошибка при попытке вернуть заказы %v из CALCULATED в COMPLETED для отчета #%d: %v", coveredOrderIDs, settlementID, errRevert)
			} else {
				log.Printf("MarkSettlementAsUnpaidToOwner: Заказы %v, связанные с отчетом #%d, проверены и при необходимости возвращены в статус COMPLETED из CALCULATED.", coveredOrderIDs, settlementID)
			}
		}
	}

	log.Printf("Отметка 'деньги внесены' для отчета #%d снята.", settlementID)
	return opErr
}

// MarkDriverSalaryAsUnpaid снимает отметку о выплате ЗП водителю.
func MarkDriverSalaryAsUnpaid(settlementID int64) error {
	tx, err := DB.Begin()
	if err != nil {
		log.Printf("MarkDriverSalaryAsUnpaid: ошибка начала транзакции: %v", err)
		return err
	}
	var opErr error
	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			panic(p)
		} else if opErr != nil {
			tx.Rollback()
		} else {
			opErr = tx.Commit()
			if opErr != nil {
				log.Printf("MarkDriverSalaryAsUnpaid: ошибка коммита транзакции: %v", opErr)
			}
		}
	}()

	query := `UPDATE driver_settlements SET driver_salary_paid_at = NULL, updated_at = NOW() WHERE id = $1 AND driver_salary_paid_at IS NOT NULL`
	result, opErr := tx.Exec(query, settlementID)
	if opErr != nil {
		log.Printf("MarkDriverSalaryAsUnpaid: ошибка обновления отчета #%d: %v", settlementID, opErr)
		return opErr
	}
	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		var existsAndUnpaid bool
		checkQuery := `SELECT EXISTS(SELECT 1 FROM driver_settlements WHERE id = $1 AND driver_salary_paid_at IS NULL)`
		errCheck := tx.QueryRow(checkQuery, settlementID).Scan(&existsAndUnpaid)
		if errCheck == nil && existsAndUnpaid {
			log.Printf("MarkDriverSalaryAsUnpaid: ЗП по отчету #%d уже была помечена как 'не выплачена'.", settlementID)
			return nil
		}
		opErr = fmt.Errorf("отчет #%d не найден или ЗП уже помечена как 'не выплачена'", settlementID)
		return opErr
	}

	var coveredOrderIDs pq.Int64Array
	errFetchOrders := tx.QueryRow("SELECT covered_order_ids FROM driver_settlements WHERE id = $1", settlementID).Scan(&coveredOrderIDs)
	if errFetchOrders != nil {
		log.Printf("MarkDriverSalaryAsUnpaid: Не удалось получить ID заказов для отчета #%d: %v", settlementID, errFetchOrders)
	} else {
		if len(coveredOrderIDs) > 0 {
			revertQuery := `UPDATE orders SET status = $1, updated_at = NOW() WHERE id = ANY($2::bigint[]) AND status = $3`
			_, errRevert := tx.Exec(revertQuery, constants.STATUS_COMPLETED, coveredOrderIDs, constants.STATUS_CALCULATED)
			if errRevert != nil {
				log.Printf("MarkDriverSalaryAsUnpaid: Ошибка при попытке вернуть заказы %v из CALCULATED в COMPLETED для отчета #%d: %v", coveredOrderIDs, settlementID, errRevert)
			} else {
				log.Printf("MarkDriverSalaryAsUnpaid: Заказы %v, связанные с отчетом #%d, проверены и при необходимости возвращены в статус COMPLETED из CALCULATED.", coveredOrderIDs, settlementID)
			}
		}
	}

	log.Printf("Отметка 'ЗП выплачена' для отчета #%d снята.", settlementID)
	return opErr
}
