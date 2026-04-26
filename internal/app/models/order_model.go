package models

import (
	"time"
)

type Order struct {
	ID         int       `json:"id"`
	Order_code string    `json:"order_code"`
	Status     string    `json:"id_status"`
	Date       time.Time `json:"date"`
	Sum        float32   `json:"sum"`
}

type Status_Order struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type Order_Items struct {
	Photo    string    `json:"photo"`
	Name     string    `json:"name"`
	Price    float32   `json:"price"`
	Quantity int       `json:"quantity"`
	Sum      float32   `json:"sum"`
	Date     time.Time `json:"date"`
}

type Request_Order_Info struct {
	Order_code string `json:"order_code"`
}

type Pick_Up_Point_Order struct {
	Pick_Up_Point_ID int `json:"pick_up_point_id"`
}

type AccountDashboard struct {
	Total_Orders int       `json:"total_orders"`
	RegisterdAt  time.Time `json:"registered_at"`
	Total_Spent  float32   `json:"total_spent"`
	Last_Orders  []Order   `json:"recent_orders"`
}
