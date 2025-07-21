package models

import "time"

// Payout represents a payout transaction to a user (driver or loader).
type Payout struct {
	ID           int64     `json:"id"`
	UserID       int64     `json:"user_id"`            // Foreign key to User.ID (the recipient of the payout)
	Amount       float64   `json:"amount"`             // The amount paid out
	PayoutDate   time.Time `json:"payout_date"`        // Date and time of the payout
	OrderID      int64     `json:"order_id,omitempty"` // Optional: Order.ID if this payout is related to a specific order (e.g., driver paying loader for an order)
	Comment      string    `json:"comment,omitempty"`  // Optional: A comment for the payout (e.g., "Payment by driver for order #123", "Monthly salary payout")
	MadeByUserID int64     `json:"made_by_user_id"`    // User.ID of the person who made the payout (e.g., driver's User.ID or owner's User.ID)
	CreatedAt    time.Time `json:"created_at"`         // Timestamp of when the payout record was created
}
