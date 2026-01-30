package models

type Components struct {
	CPU            []Processor_Model       `json:"cpu"`
	MotherBoard    []Motherboard_Model     `json:"motherboard"`
	GPU            []Video_Card_Model      `json:"gpu"`
	RAM            []Ram_Model             `json:"ram"`
	SSD_SATA       []Ssd_Sata_Model        `json:"ssd_sata"`
	SSD_M2         []Ssd_M2_Model          `json:"ssd_m2"`
	HDD            []Hdd_Model             `json:"hdd"`
	PowerUnit      []Power_Unit_Model      `json:"power_unit"`
	Frame          []Frame_Model           `json:"frame"`
	Cooling_System []Cooling_System_Models `json:"cooling_system"`
}
