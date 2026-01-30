package email

import (
	"bytes"
	"path/filepath"
	"runtime"
	"text/template"

	"gopkg.in/gomail.v2"
)

type SMPTSender struct {
	host     string
	port     int
	user     string
	password string
}

func NewSMTPSender(host string, port int, user string, password string) *SMPTSender {
	return &SMPTSender{host, port, user, password}
}

func (s *SMPTSender) SendPasswordReset(to, resetLink string) error {
	_, filename, _, _ := runtime.Caller(0)
	dir := filepath.Dir(filename)
	templatePath := filepath.Join(dir, "templates", "reset_password.html")
	tmpl := template.Must(template.ParseFiles(templatePath))

	var body bytes.Buffer
	tmpl.Execute(&body, struct{ ResetLink string }{resetLink})
	message := gomail.NewMessage()
	message.SetHeader("From", s.user)
	message.SetHeader("To", to)
	message.SetHeader("Subject", "Восстановление пароля — Core-Spark")
	message.SetBody("text/html", body.String())

	dailer := gomail.NewDialer(s.host, s.port, s.user, s.password)
	return dailer.DialAndSend(message)
}
