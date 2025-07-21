package models

import (
	"database/sql"
	"encoding/json"
	"time"
)

// NullString - обертка для sql.NullString для правильной обработки JSON.
type NullString struct {
	sql.NullString
}

// MarshalJSON реализует интерфейс json.Marshaler для NullString.
func (ns NullString) MarshalJSON() ([]byte, error) {
	if !ns.Valid {
		return []byte("null"), nil
	}
	return json.Marshal(ns.String)
}

// UnmarshalJSON реализует интерфейс json.Unmarshaler для NullString.
func (ns *NullString) UnmarshalJSON(b []byte) error {
	var s *string
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}
	if s != nil {
		ns.String = *s
		ns.Valid = true
	} else {
		ns.Valid = false
	}
	return nil
}

// NullTime - обертка для sql.NullTime для правильной обработки JSON.
type NullTime struct {
	sql.NullTime
}

// MarshalJSON реализует интерфейс json.Marshaler для NullTime.
func (nt NullTime) MarshalJSON() ([]byte, error) {
	if !nt.Valid {
		return []byte("null"), nil
	}
	return json.Marshal(nt.Time)
}

// UnmarshalJSON реализует интерфейс json.Unmarshaler для NullTime.
func (nt *NullTime) UnmarshalJSON(b []byte) error {
	var t *time.Time
	if err := json.Unmarshal(b, &t); err != nil {
		return err
	}
	if t != nil {
		nt.Time = *t
		nt.Valid = true
	} else {
		nt.Valid = false
	}
	return nil
}
