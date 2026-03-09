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

type Cart_Item struct {
	Cart_item_id int     `json:"cart_item_id"`
	ID_config    int     `json:"id_config"`
	Name         string  `json:"name"`
	Photo        string  `json:"photo"`
	Article      string  `json:"article"`
	Quantity     int     `json:"quantity"`
	Price        float32 `json:"price"`
}

type Universal_Model_Cart struct {
	ID_Config int `json:"id_config"`
}

type Update_Cart_Items_Quantity struct {
	ID_config int `json:"id_config"`
	Num       int `json:"num"`
}
