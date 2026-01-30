package models

type Response_For_Guests_Model struct {
	Id               int     `json:"id"`
	Photo            string  `json:"photo"`
	Category         string  `json:"category"`
	Name             string  `json:"name"`
	Manufacturer     string  `json:"manufacturer"`
	Product_Line     string  `json:"product_line"`
	GPU_Manufacturer *string `json:"gpu_manufacturer"`
	Series           *string `json:"series"`
	Total_Ram_GB     int     `json:"total_ram_gb"`
	Price            float32 `json:"price"`
}

type Response_For_AuthUser_Model struct {
	Id               int     `json:"id"`
	Photo            string  `json:"photo"`
	Category         string  `json:"category"`
	Name             string  `json:"name"`
	Manufacturer     string  `json:"manufacturer"`
	Product_Line     string  `json:"product_line"`
	GPU_Manufacturer *string `json:"gpu_manufacturer"`
	Series           *string `json:"series"`
	Total_Ram_GB     int     `json:"total_ram_gb"`
	In_Cart          bool    `json:"in_cart"`
	Price            float32 `json:"price"`
}
