package models

type PC_model struct {
	ID_Config      int
	Name           string
	Photo          string
	Price          float32
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
	In_Cart        bool `json:"in_cart"`
}

type PC_model_By_Product struct {
	ID_Config           int                              `json:"id_config"`
	Name                string                           `json:"name"`
	Photo               string                           `json:"photo"`
	Article             string                           `json:"article"`
	Category            string                           `json:"category"`
	Price               float32                          `json:"price"`
	Short_Description   string                           `json:"short_description"`
	Product_Description string                           `json:"product_description"`
	Processor           Processor_Comparison_Model       `json:"cpu"`
	Motherboard         Motherboard_Comparison_Model     `json:"motherboard"`
	GPU                 Video_Card_Comparison_Model      `json:"gpu"`
	RAM                 Ram_Slot                         `json:"ram"`
	SSD_M2              SSD_M2_Slot                      `json:"ssd_m2"`
	SSD_SATA            SSD_SATA_Slot                    `json:"ssd_sata"`
	HDD                 HDD_Slot                         `json:"hdd"`
	Power_Unit          Power_Unit_Comparison_Model      `json:"power_unit"`
	Frame               Frame_Comparison_Model           `json:"frame"`
	Cooling_System      Cooling_System_Comparison_Models `json:"cooling_system"`
	In_Cart             bool                             `json:"in_cart"`
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
