package models

import (
	"database/sql"
	// "time" // Раскомментируйте, если будете добавлять CreatedAt/UpdatedAt в структуру User
)

// User represents a user in the system.
type User struct {
	ID                int64
	ChatID            int64
	Role              string
	FirstName         string
	LastName          string
	Nickname          sql.NullString
	Phone             sql.NullString
	CardNumber        sql.NullString // Новое поле для номера карты
	IsBlocked         bool
	BlockReason       sql.NullString
	BlockDate         sql.NullTime
	MainMenuMessageID int
	// CreatedAt and UpdatedAt can be added if you track them,
	// which is good practice. They are present in your DB schema.
	// CreatedAt         time.Time
	// UpdatedAt         time.Time
}

// Stats represents statistical data.
// Note: This was in your main.go, but it's more of a data structure (model)
// used for representing results, so it fits well here or in a dedicated stats_model.go.
type Stats struct {
	TotalOrders      int
	NewOrders        int
	InProgressOrders int
	CompletedOrders  int
	CanceledOrders   int
	WasteOrders      int
	DemolitionOrders int
	MaterialOrders   int
	Revenue          float64
	Expenses         float64
	Profit           float64
	Debts            float64 // Это поле было в вашем коде, но логика для него не была ясна
	NewClients       int
}

const (
	RoleLoader = "loader"
	RoleDriver = "driver"
)
