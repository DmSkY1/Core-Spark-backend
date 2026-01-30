package models

type Config_PC struct {
	ID                int     `json:"id"`
	ID_Processor      int     `json:"id_processor"`
	ID_Motherboard    int     `json:"id_motherboard"`
	ID_Pc_Ram_Config  int     `json:"id_pc_ram_config"`
	ID_Video_Card     int     `json:"id_video_card"`
	ID_Power_Unit     int     `json:"id_power_unit"`
	ID_Ssd_M2_Config  int     `json:"id_ssd_m2_config"`
	ID_Ssd_Config     int     `json:"id_ssd_config"`
	ID_Hdd_Config     int     `json:"id_hdd_config"`
	ID_Frame          int     `json:"id_frame"`
	ID_Cooling_System int     `json:"id_cooling_system"`
	Name              *string `json:"name"`
	Photo             *string `json:"photo"`
	Description       *string `json:"description"`
	Price             float32 `json:"price"`
	Category          string  `json:"category"`
	Is_Catalog        bool    `json:"is_catalog"`
}
