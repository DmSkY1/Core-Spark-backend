package models

type Power_Unit_Model struct {
	ID           int     `json:"id"`
	Photo        string  `json:"photo"`
	Manufacturer string  `json:"manufacturer"`
	Model        string  `json:"model"`
	Power        int     `json:"power"`
	Has_Ocp      bool    `json:"has_ocp"`
	Has_Ovp      bool    `json:"has_ovp"`
	Has_Uvp      bool    `json:"has_uvp"`
	Has_Scp      bool    `json:"has_scp"`
	Has_Opp      bool    `json:"has_opp"`
	Fan_Size     int     `json:"fan_size"`
	Form_Factor  string  `json:"form_factor"`
	Price        float32 `json:"price"`
}
