package models

import "time"

type Profile_Model struct {
	Name           string    `json:"name"`
	Surname        string    `json:"surname"`
	Email          string    `json:"email"`
	Phone          *string   `json:"phone"`
	Avatar         *string   `json:"avater"`
	Role           string    `json:"role"`
	Created_at     time.Time `json:"created_at"`
	Pick_up_point  *string   `json:"pick_up_point"`
	CartItemsCount int       `json:"cart_items_count"`
}
