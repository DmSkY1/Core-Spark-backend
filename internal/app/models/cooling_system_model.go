package models

type Cooling_System_Models struct {
	ID               int      `json:"id"`
	Photo            string   `json:"photo"`
	Manufacturer     string   `json:"manufacturer"`
	Model            string   `json:"model"`
	Type             string   `json:"type"`
	Sockets          []string `json:"sockets"`
	Dissipated_Power int      `json:"dissipated_power"`
	Price            float32  `json:"price"`
}

type Cooling_System_Comparison_Models struct {
	Manufacturer     string   `json:"manufacturer"`
	Model            string   `json:"model"`
	Type             string   `json:"type"`
	Sockets          []string `json:"sockets"`
	Dissipated_Power int      `json:"dissipated_power"`
}
