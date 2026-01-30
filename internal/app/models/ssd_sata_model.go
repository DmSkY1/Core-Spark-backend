package models

type Ssd_Sata_Model struct {
	ID               int     `json:"id"`
	Photo            string  `json:"photo"`
	Manufacturer     string  `json:"manufacturer"`
	Model            string  `json:"model"`
	Storage_Capacity int     `json:"storage_capacity"`
	Reading_Speed    int     `json:"reading_speed"`
	Write_Speed      int     `json:"write_speed"`
	Rewrite_Resource int     `json:"rewrite_resource"`
	Price            float32 `json:"price"`
}

type Ssd_Sata_Config_Model struct {
	ID          int `json:"id"`
	Id_Ssd_Sata int `json:"id_ssd_sata"`
	Quantity    int `json:"quantity"`
}
