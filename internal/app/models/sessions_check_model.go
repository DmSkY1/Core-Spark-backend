package models

import (
	"time"

	"github.com/google/uuid"
)

type Session_Check_Model struct {
	UUID       uuid.UUID `json:"uuid"`
	User_id    int       `json:"user_id"`
	Created_at time.Time `json:"created_at"`
	Expires_at time.Time `json:"expires_at"`
	Is_active  bool      `json:"is_active"`
}
