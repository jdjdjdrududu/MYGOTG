package models

// LoaderSalaryDetail stores details about a loader's salary for a specific order.
type LoaderSalaryDetail struct {
	Amount       float64 `json:"amount"`
	PaidByDriver bool    `json:"paid_by_driver"`
	Comment      string  `json:"comment,omitempty"` // Add this line
}

// Expense represents an expense related to an order.
type Expense struct {
	ID             int64 // Primary key for expense
	OrderID        int   // Foreign key to Order.ID
	DriverID       int64 // Foreign key to User.ID (Driver's User ID, not ChatID)
	Fuel           float64
	Other          float64
	LoaderSalaries map[string]LoaderSalaryDetail `json:"loader_salaries"` // JSONB in DB. Key: loader's User.ID (as string), Value: LoaderSalaryDetail
	Revenue        float64                       // Total revenue from the client for this order
	DriverShare    float64                       // Driver's calculated share from this order
	// CreatedAt time.Time // from DB schema, can be added if needed in the struct
}
