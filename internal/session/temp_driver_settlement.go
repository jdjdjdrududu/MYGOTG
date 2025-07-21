package session

import (
	"Original/internal/models"
	"database/sql"
	"time"
)

// TempDriverSettlementData хранит временные данные при заполнении отчета водителем.
type TempDriverSettlementData struct {
	CurrentStep            string
	CoveredOrdersRevenue   float64
	FuelExpense            float64
	OtherExpenses          []models.OtherExpenseDetail // Список прочих расходов
	CurrentLoaderIndex     int
	LoadersCount           int
	LoaderPayments         []models.LoaderPaymentDetail
	CoveredOrdersCount     int
	CurrentMessageID       int
	EditingSettlementID    int64
	FieldToEditByOwner     string
	DriverCalculatedSalary float64
	AmountToCashier        float64

	UnsettledOrders []models.Order `json:"-"`
	CoveredOrderIDs []int64

	SettlementCreateTime time.Time
	EditingLoaderIndex   int
	TempLoaderNameInput  string

	// Поля для временного хранения прочих расходов
	TempOtherExpenseDescription string
	EditingOtherExpenseIndex    int // ИНДЕКС для редактирования/удаления "прочего расхода"

	OriginalPaidToOwnerAt  sql.NullTime
	DriverUserIDForBackNav int64
	ViewTypeForBackNav     string
	PageForBackNav         int
}

// NewTempDriverSettlement создает новый экземпляр TempDriverSettlementData.
func NewTempDriverSettlement() TempDriverSettlementData {
	return TempDriverSettlementData{
		LoaderPayments:           make([]models.LoaderPaymentDetail, 0),
		UnsettledOrders:          make([]models.Order, 0),
		CoveredOrderIDs:          make([]int64, 0),
		OtherExpenses:            make([]models.OtherExpenseDetail, 0),
		EditingLoaderIndex:       -1,
		EditingOtherExpenseIndex: -1, // Инициализация индекса для прочих расходов
	}
}

// RecalculateTotals - Вспомогательная функция для пересчета ЗП и суммы к сдаче.
func (td *TempDriverSettlementData) RecalculateTotals(driverSharePercentage float64) {
	netForDriver := td.CoveredOrdersRevenue - td.FuelExpense

	totalOtherExpenses := 0.0
	for _, oe := range td.OtherExpenses {
		totalOtherExpenses += oe.Amount
	}
	netForDriver -= totalOtherExpenses

	totalLoaderSalary := 0.0
	for _, lp := range td.LoaderPayments {
		totalLoaderSalary += lp.Amount
	}
	netForDriver -= totalLoaderSalary

	td.DriverCalculatedSalary = netForDriver * driverSharePercentage
	td.AmountToCashier = netForDriver - td.DriverCalculatedSalary
}
