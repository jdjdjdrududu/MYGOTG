package models

import "database/sql"

// Executor represents an assigned executor (driver/loader) for an order.
type Executor struct {
	ID        int64  // Primary key for executor assignment, if you add it to DB schema
	OrderID   int    // Foreign key to Order.ID
	UserID    int64  // User.ID of the executor (это users.id из таблицы users)
	ChatID    int64  // Telegram ChatID of the executor (добавлено)
	Role      string // e.g., "driver", "loader"
	Confirmed bool
	// CreatedAt and UpdatedAt can be added from your DB schema.
	// CreatedAt time.Time
	// UpdatedAt time.Time

	// Поля для отображения информации об исполнителе, если нужно передавать из DB напрямую
	FirstName sql.NullString `db:"first_name"` // Тег для удобства сканирования, если используется sqlx
	Nickname  sql.NullString `db:"nickname"`
	LastName  sql.NullString `db:"last_name"`

	// НОВОЕ ПОЛЕ: Статус уведомления исполнителя
	// NEW FIELD: Executor notification status
	IsNotified bool `db:"is_notified"`
}
