package models

import "time"

type EmailCode struct {
	ID        int
	Email     string
	Purpose   string
	CodeHash  string
	Payload   []byte
	ExpiresAt time.Time
	CreatedAt time.Time
}
