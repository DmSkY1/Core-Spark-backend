package models

type Motherboard_Model struct {
	ID            int     `json:"id"`
	Name          string  `json:"name"`
	Photo         string  `json:"photo"`
	Manufacturer  string  `json:"manufacturer"`
	Chipset       string  `json:"chipset"`
	Ram_Type      string  `json:"ram_type"`
	Max_Ram       int     `json:"max_ram"`
	Socket        string  `json:"socket"`
	PCIE_x16_Port int     `json:"ipcie_x16_port"`
	PCIE_x1_Port  *int    `json:"pcie_x1_port"`
	Wifi          bool    `json:"wifi"`
	Audio_Codec   string  `json:"audio_codec"`
	Form_Factor   string  `json:"form_factor"`
	Ram_Slots     int     `json:"ram_slots"`
	M2_Slots      int     `json:"m2_slots"`
	Sata_Slots    int     `json:"sata_slots"`
	Price         float32 `json:"price"`
}

type Motherboard_Comparison_Model struct {
	Name          string `json:"name"`
	Manufacturer  string `json:"manufacturer"`
	Chipset       string `json:"chipset"`
	Ram_Type      string `json:"ram_type"`
	Max_Ram       int    `json:"max_ram"`
	Socket        string `json:"socket"`
	PCIE_x16_Port int    `json:"ipcie_x16_port"`
	PCIE_x1_Port  *int   `json:"pcie_x1_port"`
	Wifi          bool   `json:"wifi"`
	Audio_Codec   string `json:"audio_codec"`
	Form_Factor   string `json:"form_factor"`
	Ram_Slots     int    `json:"ram_slots"`
	M2_Slots      int    `json:"m2_slots"`
	Sata_Slots    int    `json:"sata_slots"`
}
