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
	PickUpPoint *int      `json:"pick_up_point"`
}

type ResponseUpdatePhoneModel struct {
	Phone string `json:"phone"`
}

type PickUpPoint_Model struct {
	ID           int    `json:"id"`
	Name         string `json:"name"`
	Address      string `json:"address"`
	OpeningHours string `json:"opening_hours"`
	DefaultPoint *int   `json:"default_point"`
}

type Response_Pick_Up_Point struct {
	ID int `json:"id"`
}

type Response_Change_Password struct {
	Old_Password string `json:"old_password"`
	New_Password string `json:"new_password"`
}

type Response_Change_Data struct {
	Name    string `json:"name"`
	Surname string `json:"surname"`
	Phone   string `json:"phone"`
}
