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
	"github.com/vmihailenco/msgpack/v5"
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

	// TODO Инициализировать хеширование каталога, компонентов конфигуратора для оптимизированной работы
	// Синхронизация сессий, сохранение сессии в редис, и изменение сразу во всех(кторые привязадны к пользователю)

	emailSender := email.NewSMTPSender(
		os.Getenv("SMTP_HOST"),
		587,
		os.Getenv("EMAIL"),
		os.Getenv("PASSWORD_MAIL"),
	)

	_repo := repository.NewRepository(db)

	components, err := _repo.Components()
	if err != err {
		logger.Log.Warn(err.Error())
	}
	data, err := msgpack.Marshal(components) // TODO Убрать весь этот ужас в функцим в хендлере, и сделать асинхронами
	if err != nil {
		logger.Log.Warn(err.Error())
	}
	if err := redisDB.Set(context.Background(), "configurator:components", data, 0).Err(); err != nil {
		logger.Log.Warn(err.Error())
	}

	// TODO инициализировать загрузку компонентов в кеш здесь, желательно асинхронно
	_service := service.NewService(_repo, *emailSender)
	_handler := handler.NewHandler(_service, redisDB)

	r := chi.NewRouter()
	r.Use(_handler.RateLimiterMiddleware(100, 130))
	r.Use(middleware.Logger)                                    // TODO заменить в будущем на другой логер
	r.Use(_handler.SessionCheckMiddleware)                      // Для проверки сессии на работоспособность
	r.Post("/register", _handler.Register)                      // Регистариция пользователя
	r.Post("/login", _handler.Login)                            // авторизация пользователя
	r.Post("/forgot_password", _handler.RequestPasswordReset)   // смена пароля, отправка письма на почту
	r.Get("/api/catalog", _handler.Catalog)                     // получение товаров для каталога
	r.Get("/api/catalog/search", _handler.SearchCatalog)        // поиск в каталоге
	r.Get("/api/user/profile", _handler.GetProfile)             // получение профиля пользователя
	r.Post("/api/user/upload_avatar", _handler.UploadAvatar)    // загрузка аватара пользователя
	r.Get("/api/components", _handler.Components)               // получение всех компонентов для конфигуратора
	r.Post("/api/cart/add", _handler.AddCart)                   // добавляет товар в корзину
	r.Post("/api/cart/update", _handler.UpdateCartItemQuantity) // увеличивает или уменьшает количество товара в корзине
	r.Post("/api/cart/remove", _handler.RemoveFromCart)         // удаляет товар из корзины/api/comparison/get_pc
	r.Get("/api/cart/items", _handler.Cart_Items)               // получение всех товаров из корзины пользователя
	r.Post("/api/cart/config/add", _handler.AddConfigToCart)    // добавление кастомной сборки в корзину
	r.Get("/api/verify_token", _handler.IsTokenValid)           // проверка токена для смены пароля
	r.Post("/api/reset_password", _handler.ResetPassword)       // запрос на смену пароля
	r.Post("/api/comparison/get_pc", _handler.GetComponentsPC)
	r.Post("/api/user/update/phone", _handler.UpdatePhoneNumber)
	r.Get("/api/check_auth", _handler.CheckAuth)

	if err := http.ListenAndServe(":3000", r); err != nil {
		logger.Log.Warn(err.Error())
		panic(err)
	}
}
