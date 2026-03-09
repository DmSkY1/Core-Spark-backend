package models

type Components struct {
	CPU            []Processor_Model       `json:"cpu" msgpack:"cpu"`
	MotherBoard    []Motherboard_Model     `json:"motherboard" msgpack:"motherboard"`
	GPU            []Video_Card_Model      `json:"gpu" msgpack:"gpu"`
	RAM            []Ram_Model             `json:"ram" msgpack:"ram"`
	SSD_SATA       []Ssd_Sata_Model        `json:"ssd_sata" msgpack:"ssd_sata"`
	SSD_M2         []Ssd_M2_Model          `json:"ssd_m2" msgpack:"ssd_m2"`
	HDD            []Hdd_Model             `json:"hdd" msgpack:"hdd"`
	PowerUnit      []Power_Unit_Model      `json:"power_unit" msgpack:"power_unit"`
	Frame          []Frame_Model           `json:"frame" msgpack:"frame"`
	Cooling_System []Cooling_System_Models `json:"cooling_system" msgpack:"cooling_system"`
}
