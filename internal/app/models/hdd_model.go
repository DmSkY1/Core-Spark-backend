package models

type Hdd_Model struct {
	ID               int     `json:"id"`
	Photo            string  `json:"photo"`
	Manufacturer     string  `json:"manufacturer"`
	Form_Factor      string  `json:"form_factor"`
	Model            string  `json:"model"`
	Storage_Capacity int     `json:"storage_capacity"`
	Rotation_Speed   int     `json:"rotation_speed"`
	Price            float32 `json:"price"`
}

type Hdd_Config_Model struct {
	ID       int `json:"id"`
	Id_Hdd   int `json:"id_hdd"`
	Quantity int `json:"quantity"`
}

type Hdd_Comparison_Model struct {
	Manufacturer     *string `json:"manufacturer"`
	Form_Factor      *string `json:"form_factor"`
	Model            *string `json:"model"`
	Storage_Capacity *int    `json:"storage_capacity"`
	Rotation_Speed   *int    `json:"rotation_speed"`
}
