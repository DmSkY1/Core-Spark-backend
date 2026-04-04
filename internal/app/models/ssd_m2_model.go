package models

type Ssd_M2_Model struct {
	ID               int     `json:"id"`
	Photo            string  `json:"photo"`
	Manufacturer     string  `json:"manufacturer"`
	Model            string  `json:"model"`
	PCIE             string  `json:"pcie"`
	Storage_Capacity int     `json:"storage_capacity"`
	Reading_Speed    int     `json:"reading_speed"`
	Write_Speed      int     `json:"write_speed"`
	Rewrite_Resource int     `json:"rewrite_resource"`
	Price            float32 `json:"price"`
}

type Ssd_M2_Config_Model struct {
	ID        int `json:"id"`
	Id_Ssd_M2 int `json:"id_ssd_m2"`
	Quantity  int `json:"quantity"`
}

type Ssd_M2_Comparison_Model struct {
	Manufacturer     *string `json:"manufacturer"`
	Model            *string `json:"model"`
	PCIE             *string `json:"pcie"`
	Storage_Capacity *int    `json:"storage_capacity"`
	Reading_Speed    *int    `json:"reading_speed"`
	Write_Speed      *int    `json:"write_speed"`
	Rewrite_Resource *int    `json:"rewrite_resource"`
}
