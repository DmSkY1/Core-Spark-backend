package models

type Frame_Model struct {
	ID                    int     `json:"id"`
	Photo                 string  `json:"photo"`
	Manufacturer          string  `json:"manufacturer"`
	Model                 string  `json:"model"`
	Supports_Mini_Itx     bool    `json:"supports_mini_itx"`
	Supports_Micro_Atx    bool    `json:"supports_micro_atx"`
	Supports_Atx          bool    `json:"supports_atx"`
	Supports_E_Atx        bool    `json:"supports_e_atx"`
	Liquid_Cooling_System bool    `json:"liquid_cooling_system"`
	Fans_Included         bool    `json:"fans_included"`
	Maximum_Length_GPU    int     `json:"maximum_length_gpu"`
	Maximum_Cooler_Height int     `json:"maximum_cooler_height"`
	Type_Size             string  `json:"type_size"`
	Price                 float32 `json:"price"`
}
