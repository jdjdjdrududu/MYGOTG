package models

import (
	"database/sql"
	"time"
)

type Order struct {
	ID                      int64
	UserID                  int // ВОЗВРАЩАЕНО: sql.NullInt64 -> int
	UserChatID              int64
	Category                string
	Subcategory             string
	Name                    string
	Photos                  []string
	Videos                  []string
	Date                    string
	Time                    string
	Phone                   string
	Address                 string
	Description             string
	Status                  string
	Cost                    sql.NullFloat64
	Payment                 string
	Latitude                float64
	Longitude               float64
	Reason                  sql.NullString
	MediaMessageIDs         []int
	MediaMessageIDsMap      map[string]bool
	CurrentMessageID        int
	LocationPromptMessageID int
	MessageSent             bool
	BlockTargetChatID       int64
	CreatedAt               time.Time
	UpdatedAt               time.Time
	IsDriverSettled         bool `json:"is_driver_settled" db:"is_driver_settled"`
}
