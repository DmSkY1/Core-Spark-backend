package models

type Processor_Model struct {
	ID                       int     `json:"id"`
	Manufacturer             string  `json:"manufacturer"`
	Photo                    string  `json:"photo"`
	Product_Line             string  `json:"product_line"`
	Model                    string  `json:"model"`
	Socket                   string  `json:"socket"`
	Architecture             string  `json:"architecture"`
	Number_Cores             int     `json:"number_cores"`
	Number_Threads           int     `json:"number_threads"`
	Frequency                int     `json:"frequency "`
	TDP                      int     `json:"tdp"`
	Max_TDP                  int     `json:"max_tdp"`
	Ram_Standart             string  `json:"ram_standart"`
	Integrated_Graphics_Core bool    `json:"integrated_graphics_core"`
	Price                    float32 `json:"price"`
}
