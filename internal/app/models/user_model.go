package models

import (
	"time"
)

type User struct {
	ID          int       `json:"id"`
	Name        string    `json:"name"`
	Surname     string    `json:"surname"`
	Email       string    `json:"email"`
	Telephone   *string   `json:"telephone"`
	Password    string    `json:"password"`
	Avatar      *string   `json:"avatar"`
	Role        string    `json:"role"`
	Created_at  time.Time `json:"created_at"`
	PickUpPoint *string   `json:"pick_up_point"`
}
