package models

type Cart struct {
	ID      int `json:"id"`
	ID_User int `json:"id_user"`
}

type Cart_PC struct {
	ID        int `json:"id"`
	ID_Config int `json:"id_config"`
	ID_Cart   int `json:"id_cart"`
	Quantity  int `json:"quantity"`
}
