package models

type Reset_Password_Model struct {
	Token    string `json:"token"`
	Password string `json:"password"`
}
