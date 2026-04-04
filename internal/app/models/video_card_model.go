package models

type Video_Card_Model struct {
	ID                     int     `json:"id"`
	Manufacturer           string  `json:"manufacturer"`
	Photo                  string  `json:"photo"`
	GPU_Manufacturer       string  `json:"gpu_manufacturer"`
	Series                 string  `json:"series"`
	Price                  float32 `json:"price"`
	PCIE                   string  `json:"pcie"`
	Video_Memory_Capacity  int     `json:"video_memory_capacity"`
	HDMI                   int     `json:"hdmi"`
	DisplayPort            int     `json:"displayport"`
	Memory_Type            string  `json:"memory_type"`
	GPU_Frequency          int     `json:"gpu_frequency"`
	Bandwidth              int     `json:"bandwidth"`
	Video_Memory_Frequency int     `json:"video_memory_frequency"`
	Consumption            int     `json:"consumption"`
	Memory_Bus             int     `json:"memory_bus"`
}

type Video_Card_Comparison_Model struct {
	Manufacturer           *string `json:"manufacturer"`
	GPU_Manufacturer       *string `json:"gpu_manufacturer"`
	Series                 *string `json:"series"`
	PCIE                   *string `json:"pcie"`
	Video_Memory_Capacity  *int    `json:"video_memory_capacity"`
	HDMI                   *int    `json:"hdmi"`
	DisplayPort            *int    `json:"displayport"`
	Memory_Type            *string `json:"memory_type"`
	GPU_Frequency          *int    `json:"gpu_frequency"`
	Bandwidth              *int    `json:"bandwidth"`
	Video_Memory_Frequency *int    `json:"video_memory_frequency"`
	Consumption            *int    `json:"consumption"`
	Memory_Bus             *int    `json:"memory_bus"`
}
