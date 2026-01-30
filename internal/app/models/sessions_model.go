package models

import (
	"time"

	"github.com/google/uuid"
)

type Sessions struct {
	UUID       uuid.UUID `json:"uuid"`
	ID_User    int       `json:"id_user"`
	User_Agent string    `json:"user_agent"`
	Created_At time.Time `json:"created_at"`
	Expires_At time.Time `json:"expires_at"`
	Is_Active  bool      `json:"is_active"`
}
