package models

import "time"

type Token_Verification_Model struct {
	ID         int
	Expires_At time.Time
}
