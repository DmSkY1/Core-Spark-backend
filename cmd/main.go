package main

import (
	"gobackend/internal/app/email"
	"gobackend/internal/app/handler"
	"gobackend/internal/app/handler/database"
	"gobackend/internal/app/repository"
	"gobackend/internal/app/service"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func main() {

	db := database.DBConnection()
	defer db.Close()

	emailSender := email.NewSMTPSender(
		os.Getenv("SMTP_HOST"),
		587,
		os.Getenv("EMAIL"),
		os.Getenv("PASSWORD_MAIL"),
	)

	_repo := repository.NewRepository(db)
	_service := service.NewService(_repo, *emailSender)
	_handler := handler.NewHandler(_service)

	r := chi.NewRouter()
	r.Use(middleware.Logger) // TODO заменить в будущем на другой логер 
	// r.Use(middleware, ) Для проверки сессии на работоспособность
	r.Post("/register", _handler.Register)
	r.Post("/login", _handler.Login)
	r.Post("/forgot-password", _handler.RequestPasswordReset)
	r.Get("/api/catalog", _handler.Catalog)
	r.Get("/api/components", _handler.Components)
	r.Get("/api/verify-token", _handler.IsTokenValid)
	r.Post("/api/reset-password", _handler.ResetPassword)
	http.ListenAndServe(":3000", r)
}
