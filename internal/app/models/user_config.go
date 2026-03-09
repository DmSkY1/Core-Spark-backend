package models

type User_Config_Model struct {
	Cpu_ID            int                 `json:"cpu_id"`
	Motherboard_ID    int                 `json:"motherboard_id"`
	GPU_ID            int                 `json:"gpu_id"`
	Ram               Ram_For_Config      `json:"ram"`
	SSD_M2            SSD_M2_For_Config   `json:"ssd_m2"`
	SSD_Sata          SSD_Sata_For_Config `json:"ssd_sata"`
	HDD               HDD_For_Config      `json:"hdd"`
	Power_Unit_ID     int                 `json:"power_unit_id"`
	Frame_ID          int                 `json:"frame_id"`
	Cooling_System_ID int                 `json:"cooling_system_id"`
}

type Ram_For_Config struct {
	ID    int `json:"id"`
	Count int `json:"count"`
}

type SSD_M2_For_Config struct {
	ID    int `json:"id"`
	Count int `json:"count"`
}

type SSD_Sata_For_Config struct {
	ID    int `json:"id"`
	Count int `json:"count"`
}

type HDD_For_Config struct {
	ID    int `json:"id"`
	Count int `json:"count"`
}
