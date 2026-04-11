package models

type Login_Model struct {
	Email      string `json:"email"`
	Password   string `json:"password"`
	User_Agent string `json:"user_agent"`
	Device_Id  string `json:"device_id"`
}
