package main

import (
	"context"
	"gobackend/internal/app/email"
	"gobackend/internal/app/handler"
	"gobackend/internal/app/handler/database"
	"gobackend/internal/app/models"
	"gobackend/internal/app/repository"
	"gobackend/internal/app/service"
	"gobackend/pkg/logger"
	"net/http"
	"os"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func main() {
	defer logger.Log.Sync()
	//logger.InitFileLogger()

	logger.Log.Info("start")
	db := database.DBConnection()
	defer db.Close()

	redisDB, err := database.RedisConnection(context.Background(), models.Redis_Config_Model{
		Addr:            os.Getenv("REDIS_ADDR"),
		Password:        os.Getenv("REDIS_PASSWORD"),
		User:            os.Getenv("REDIS_NAME"),
		DB:              0,
		MaxRetries:      5,
		DialTimeout:     1 * time.Second,
		Timeout:         2 * time.Second,
		PoolSize:        100,
		MinIdleConns:    40,
		PoolTimeout:     30 * time.Second,
		ConnMaxLifetime: 30 * time.Minute,
		ConnMaxIdleTime: 5 * time.Minute,
	})
	if err != nil {
		panic(err)
	}
	emailSender := email.NewSMTPSender(
		os.Getenv("SMTP_HOST"),
		587,
		os.Getenv("EMAIL"),
		os.Getenv("PASSWORD_MAIL"),
	)

	_repo := repository.NewRepository(db)
	_service := service.NewService(_repo, *emailSender)
	_handler := handler.NewHandler(_service, redisDB)

	r := chi.NewRouter()
	r.Use(middleware.Logger)               // TODO заменить в будущем на другой логер
	r.Use(_handler.SessionCheckMiddleware) // Для проверки сессии на работоспособность
	r.Post("/register", _handler.Register)
	r.Post("/login", _handler.Login)
	r.Post("/forgot-password", _handler.RequestPasswordReset)
	r.Get("/api/catalog", _handler.Catalog)
	r.Get("/api/components", _handler.Components)
	r.Get("/api/verify-token", _handler.IsTokenValid)
	r.Post("/api/reset-password", _handler.ResetPassword)
	http.ListenAndServe(":3000", r)
}
