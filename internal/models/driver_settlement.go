package models

import (
	"database/sql"
	"time"
)

// LoaderPaymentDetail остается без изменений
type LoaderPaymentDetail struct {
	LoaderUserID     int64   `json:"loader_user_id"`    // User.ID грузчика
	LoaderIdentifier string  `json:"loader_identifier"` // Имя или другой идентификатор для отображения
	Amount           float64 `json:"amount"`
}

// НОВАЯ СТРУКТУРА для детализации прочих расходов
type OtherExpenseDetail struct {
	Description string  `json:"description"`
	Amount      float64 `json:"amount"`
}

// DriverSettlement представляет собой отчет водителя о расходах за определенный период/набор заказов.
type DriverSettlement struct {
	ID                     int64                 `json:"id"`
	DriverUserID           int64                 `json:"driver_user_id"`       // User.ID водителя
	SettlementTimestamp    time.Time             `json:"settlement_timestamp"` // Время фактического создания отчета
	CoveredOrdersRevenue   float64               `json:"covered_orders_revenue"`
	FuelExpense            float64               `json:"fuel_expense"`
	OtherExpensesJSON      sql.NullString        `json:"-"`                      // ИЗМЕНЕНО: JSON строка для хранения в БД [{description: "Парковка", amount: 200}, ...]
	OtherExpenses          []OtherExpenseDetail  `json:"other_expenses" db:"-"`  // ИЗМЕНЕНО: Для использования в коде
	LoaderPaymentsJSON     sql.NullString        `json:"-"`                      // JSON строка для хранения в БД [{loader_identifier: "Иван", amount: 1000}, ...]
	LoaderPayments         []LoaderPaymentDetail `json:"loader_payments" db:"-"` // Для использования в коде
	DriverCalculatedSalary float64               `json:"driver_calculated_salary"`
	AmountToCashier        float64               `json:"amount_to_cashier"`
	CoveredOrdersCount     int                   `json:"covered_orders_count"` // Информационно: количество заказов, которое водитель указал
	CreatedAt              time.Time             `json:"created_at"`
	UpdatedAt              time.Time             `json:"updated_at"`

	CoveredOrderIDs []int64   `json:"covered_order_ids" db:"covered_order_ids"`
	ReportDate      time.Time `json:"report_date"`

	PaidToOwnerAt      sql.NullTime `json:"paid_to_owner_at,omitempty" db:"paid_to_owner_at"`
	DriverSalaryPaidAt sql.NullTime `json:"driver_salary_paid_at,omitempty" db:"driver_salary_paid_at"`

	// --- НАЧАЛО ИЗМЕНЕНИЯ ---
	Status       string         `json:"status" db:"status"`
	AdminComment sql.NullString `json:"admin_comment,omitempty" db:"admin_comment"`
	// --- КОНЕЦ ИЗМЕНЕНИЯ ---
}

// OwnerCashierRecord остается без изменений. Его ReportDate будет соответствовать ReportDate из DriverSettlement.
type OwnerCashierRecord struct {
	ID                  int64     `json:"id"`
	DriverUserID        int64     `json:"driver_user_id"`
	DriverName          string    `json:"driver_name"` // Для отображения
	ReportDate          time.Time `json:"report_date"` // Дата, за которую агрегированы данные
	TotalAmountDue      float64   `json:"total_amount_due"`
	ContributingSettIDs []int64   `json:"contributing_sett_ids"` // ID отчетов DriverSettlement, сформировавших эту сумму
	LastUpdatedAt       time.Time `json:"last_updated_at"`       // Время последнего обновления этой записи
}

// НОВАЯ СТРУКТУРА для отображения отчетов с именем водителя
type DriverSettlementWithDriverName struct {
	DriverSettlement
	DriverFirstName sql.NullString `db:"driver_first_name"`
	DriverLastName  sql.NullString `db:"driver_last_name"`
	DriverNickname  sql.NullString `db:"driver_nickname"`
}

// AggregatedDriverSettlementInfo - НОВАЯ СТРУКТУРА для хранения агрегированных данных по отчетам водителей.
type AggregatedDriverSettlementInfo struct {
	DriverUserID         int64          `json:"driver_user_id"`
	DriverFirstName      sql.NullString `json:"driver_first_name" db:"driver_first_name"`
	DriverLastName       sql.NullString `json:"driver_last_name" db:"driver_last_name"`
	DriverNickname       sql.NullString `json:"driver_nickname" db:"driver_nickname"`
	TotalAmountToCashier float64        `json:"total_amount_to_cashier" db:"total_amount_to_cashier"`
	TotalReportsCount    int            `json:"total_reports_count" db:"total_reports_count"`
}
