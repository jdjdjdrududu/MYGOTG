package models

import (
	"database/sql"
	"time"
)

// Referral represents a referral in the system.
type Referral struct {
	ID              int64 // Primary key
	InviterID       int64 // User.ID of the inviter
	InviteeID       int64 // User.ID of the invitee
	OrderID         int   // Order.ID for which the referral bonus is applied
	Amount          float64
	CreatedAt       time.Time
	Name            string        // Name of the referred user (invitee) for display
	PaidOut         bool          // True if this specific referral bonus has been paid out
	PayoutRequestID sql.NullInt64 `db:"payout_request_id"` // ID запроса на выплату, если этот бонус в него включен
	// UpdatedAt time.Time // from DB schema
}

// ReferralPayoutRequest представляет запрос на выплату реферальных бонусов.
type ReferralPayoutRequest struct {
	ID             int64          `db:"id"`
	UserChatID     int64          `db:"user_chat_id"` // ChatID пользователя, запрашивающего выплату
	Amount         float64        `db:"amount"`       // Общая сумма к выплате
	Status         string         `db:"status"`       // e.g., "pending", "approved", "rejected", "completed"
	RequestedAt    time.Time      `db:"requested_at"`
	ReferralIDs    []int64        // Массив ID рефералов (из таблицы referrals), включенных в эту выплату
	AdminComment   sql.NullString `db:"admin_comment"`   // Комментарий администратора (например, причина отклонения)
	ProcessedAt    sql.NullTime   `db:"processed_at"`    // Дата обработки запроса
	PaymentMethod  sql.NullString `db:"payment_method"`  // Способ выплаты, если нужно
	PaymentDetails sql.NullString `db:"payment_details"` // Реквизиты для выплаты
}
