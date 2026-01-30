package models

import "time"

type Order struct {
	ID         int       `json:"id"`
	Order_code string    `json:"order_code"`
	ID_Product int       `json:"id_product"`
	ID_Status  int       `json:"id_status"`
	ID_User    int       `json:"id_user"`
	Date       time.Time `json:"date"`
	Sum        float32   `json:"sum"`
}

type Status_Order struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type Product_Order struct {
	ID       int     `json:"id"`
	Name     string  `json:"name"`
	Quantity int     `json:"quantity"`
	Sum      float32 `json:"sum"`
}
