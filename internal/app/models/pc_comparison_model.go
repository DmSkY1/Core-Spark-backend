package models

type PC_model struct {
	Processor      Processor_Comparison_Model
	Motherboard    Motherboard_Comparison_Model
	GPU            Video_Card_Comparison_Model
	RAM            Ram_Slot
	SSD_M2         SSD_M2_Slot
	SSD_SATA       SSD_SATA_Slot
	HDD            HDD_Slot
	Power_Unit     Power_Unit_Comparison_Model
	Frame          Frame_Comparison_Model
	Cooling_System Cooling_System_Comparison_Models
}

type Ram_Slot struct {
	Module   Ram_Comparison_Model
	Quantity int
}
type SSD_M2_Slot struct {
	Module   Ssd_M2_Comparison_Model
	Quantity *int
}
type SSD_SATA_Slot struct {
	Module   Ssd_Sata_Comparison_Model
	Quantity *int
}
type HDD_Slot struct {
	Module   Hdd_Comparison_Model
	Quantity *int
}

type Comparison_Request_Model struct {
	ID []int `json:"id"`
}
