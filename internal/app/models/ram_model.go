package models

type Ram_Model struct {
	ID                int     `json:"id"`
	Name              string  `json:"name"`
	Photo             string  `json:"photo"`
	Brand             string  `json:"brand"`
	Volume_One_Module int     `json:"volume_one_module"` // объем одного модуля
	Memory_Type       string  `json:"memory_type"`       // тип оперативной памяти
	Frequency         int     `json:"frequency"`         // частота оперативной памяти
	Number_Modules    int     `json:"number_modules"`    // количество модулей в комплекте
	Price             float32 `json:"price"`
}

type Ram_Config struct {
	ID       int `json:"id"`
	Id_Ram   int `json:"id_ram"`
	Quantity int `json:"quantity"`
}
