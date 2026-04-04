package handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"gobackend/internal/app/models"
	"gobackend/internal/app/repository"
	"gobackend/internal/app/service"
	"gobackend/pkg/logger"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/vmihailenco/msgpack/v5"
	"go.uber.org/zap"
	"golang.org/x/time/rate"
)

type handler_struct struct {
	serv service.Serv
	red  *redis.Client
}

func NewHandler(serv service.Serv, red *redis.Client) *handler_struct {
	return &handler_struct{serv: serv, red: red}
}

func (h *handler_struct) UploadAvatar(w http.ResponseWriter, r *http.Request) {

	userid, ok := r.Context().Value("user_id").(int)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 700*1024) // Ограничитель веса тела запроса, не пропускает большие запросы (установлен ограничитель в 700кб)

	if err := r.ParseMultipartForm(700 * 1024); err != nil { // Парсим форму, и задаем ограничиьель
		logger.Log.Info(err.Error())
		if strings.Contains(err.Error(), "request body too large") {
			w.WriteHeader(http.StatusRequestEntityTooLarge)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"code":   http.StatusRequestEntityTooLarge,
				"status": "request body too large (max 700 KB)",
			})
			return
		} else {

			http.Error(w, "invalid request format", http.StatusBadRequest)
			return
		}
	}

	files := r.MultipartForm.File["photo"] // Берем значения из мапы по ключу -> получаем массив указателей

	if err := h.serv.AvatarCheck(files, userid); err != nil {

		http.Error(w, "", http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"code":    http.StatusBadRequest,
			"message": err.Error(),
		})
		return
	}
}

func (h *handler_struct) SearchCatalog(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	order := query.Get("order")
	price := query.Get("price")
	category := query["category"]
	ram := query["ram"]
	gpu := query["gpu"]
	cpu := query["cpu"]
	search_string := query.Get("search")
	page := query.Get("page")
	userid, ok := r.Context().Value("user_id").(int)
	if !ok {
		items, err := h.serv.SearchGuestService(
			normalizeToStringSlice(ram),
			normalizeToStringSlice(gpu),
			normalizeToStringSlice(cpu),
			normalizeToStringSlice(category),
			price,
			search_string,
			page,
			"9",
			order,
		) // Жеская передача 9, заключается в особой ненадобности регулировать лимит пользователем, поэтому задано фиксированное число
		if err != nil {
			fmt.Println(err)
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(items); err != nil {
			return
		}
	} else {
		items, err := h.serv.SearchAuthUserService(
			normalizeToStringSlice(ram),
			normalizeToStringSlice(gpu),
			normalizeToStringSlice(cpu),
			normalizeToStringSlice(category),
			price,
			search_string,
			userid,
			page,
			"9",
			order,
		)
		if err != nil {
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(items); err != nil {
			return
		}
	}
}

func (h *handler_struct) Catalog(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()        // получение query параметров
	order := query.Get("order")   // параметр для cортировки
	price := query.Get("price")   // параметр для цены 100-1000 (мин-макс)
	category := query["category"] // параметр для категории
	ram := query["ram"]           // параметр для оперативной памяти
	gpu := query["gpu"]           // параметр для ведокарты
	cpu := query["cpu"]           // параметр для процесосора
	page := query.Get("page")     // номер страницы
	// limit := query.Get("limit") // лимит карточек товаров
	userid, ok := r.Context().Value("user_id").(int)
	if !ok {
		items, err := h.serv.CatalogCheckGuest(
			page,
			"9",
			price,
			userid,
			order,
			normalizeToStringSlice(category),
			normalizeToStringSlice(ram),
			normalizeToStringSlice(gpu),
			normalizeToStringSlice(cpu),
		) // Жеская передача 9, заключается в особой ненадобности регулировать лимит пользователем, поэтому задано фиксированное число
		if err != nil {
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(items); err != nil {
			return
		}
	} else {
		items, err := h.serv.CatalogCheckAuthUser(
			page,
			"9",
			price,
			userid,
			order,
			normalizeToStringSlice(category),
			normalizeToStringSlice(ram),
			normalizeToStringSlice(gpu),
			normalizeToStringSlice(cpu),
		)
		if err != nil {
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(items); err != nil {
			return
		}
	}
}

func (h *handler_struct) AddCart(w http.ResponseWriter, r *http.Request) {
	user_id, ok := r.Context().Value("user_id").(int)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
	}
	var add_cartModel models.Universal_Model_Cart
	json.NewDecoder(r.Body).Decode(&add_cartModel)

	err := h.serv.AddCartService(user_id, add_cartModel.ID_Config)
	if err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (h *handler_struct) RemoveFromCart(w http.ResponseWriter, r *http.Request) {
	user_id, ok := r.Context().Value("user_id").(int)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
	}

	var remove_model models.Universal_Model_Cart
	json.NewDecoder(r.Body).Decode(&remove_model)

	err := h.serv.RemoveFromCartService(user_id, remove_model.ID_Config)
	if err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (h *handler_struct) Cart_Items(w http.ResponseWriter, r *http.Request) {
	user_id, ok := r.Context().Value("user_id").(int)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
	}

	req, err := h.serv.CartItemsService(user_id)
	if err != nil {
		logger.Log.Error("An error occurred while retrieving items from the cart:", zap.Error(err))
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(req)
}

func (h *handler_struct) UpdateCartItemQuantity(w http.ResponseWriter, r *http.Request) {
	var res models.Update_Cart_Items_Quantity
	json.NewDecoder(r.Body).Decode(&res)

	user_id, ok := r.Context().Value("user_id").(int)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
	}

	if err := h.serv.UpdateCartItemQuantityService(user_id, res.ID_config, res.Num); err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (h *handler_struct) AddConfigToCart(w http.ResponseWriter, r *http.Request) {
	var config models.User_Config_Model
	user_id, ok := r.Context().Value("user_id").(int)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
	}
	json.NewDecoder(r.Body).Decode(&config)

	if err := h.serv.AddCustomConfigToCartService(user_id, config); err != nil {
		//logger.Log.Info(err.Error())
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (h *handler_struct) GetProfile(w http.ResponseWriter, r *http.Request) {
	user_id, ok := r.Context().Value("user_id").(int)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	user_profile, err := h.serv.GetUserProfileService(user_id)
	if err != nil {
		logger.Log.Error(err.Error())
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(user_profile)
}

func (h *handler_struct) Cart(w http.ResponseWriter, r *http.Request) {
	// TODO доделать
}

func (h *handler_struct) RequestPasswordReset(w http.ResponseWriter, r *http.Request) {
	type Email struct {
		Email string `json:"email"`
	}
	user_mail := new(Email)
	json.NewDecoder(r.Body).Decode(user_mail)

	go func() {
		err := h.serv.ReqPasswordReset(user_mail.Email)
		if err != nil {
			http.Error(w, "failed to process request", http.StatusInternalServerError)
			return
		}
	}()
	w.WriteHeader(http.StatusOK)
}

func (h *handler_struct) Components(w http.ResponseWriter, r *http.Request) {
	data, err := h.red.Get(context.Background(), "configurator:components").Bytes()
	if err != nil {
		logger.Log.Error(err.Error())
	}
	var items models.Components

	if err := msgpack.Unmarshal(data, &items); err != nil {
		logger.Log.Error(err.Error())
	}

	// TODO дописать в случае, если в редис ничего нет, то обращаться в бд
	//items, err := h.serv.GetComponents()
	//if err != nil {
	//	fmt.Println(err)
	//	return
	//}
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(items)
}

func (h *handler_struct) Register(w http.ResponseWriter, r *http.Request) {

	userRegister := new(models.Register_Model)
	if err := json.NewDecoder(r.Body).Decode(userRegister); err != nil {
		return
	}
	defer r.Body.Close()

	if err := h.serv.RegisterUser(userRegister); err != nil {
		fmt.Println(err)
		if errors.Is(err, repository.UserExist) {
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"status": err.Error(),
				"code":   http.StatusUnauthorized,
			})
			return
		}
		http.Error(w, "failed to process request", http.StatusInternalServerError)
		return
	}

	w.Write([]byte("успех"))

}

func (h *handler_struct) GetComponentsPC(w http.ResponseWriter, r *http.Request) {
	var id models.Comparison_Request_Model

	if err := json.NewDecoder(r.Body).Decode(&id); err != nil {
		http.Error(w, "Incorrect IDs", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	pc, err := h.serv.GettingPCForComparisonService(id.ID)
	if err != nil {
		http.Error(w, "Incorrect data", http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(pc)
}

func (h *handler_struct) Login(w http.ResponseWriter, r *http.Request) {
	userLogin := new(models.Login_Model)

	if err := json.NewDecoder(r.Body).Decode(userLogin); err != nil {
		return
	}
	defer r.Body.Close()

	userSession_uuid, user_id, err := h.serv.LoginUser(userLogin)
	if err != nil {
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     "session_id",
		Value:    userSession_uuid.String(),
		Path:     "/",
		HttpOnly: true,
		Secure:   false, // TODO в финале изменить на true
		Expires:  time.Now().Add(1 * time.Hour),
		MaxAge:   3600,
	})
	data, err := msgpack.Marshal(userSession_uuid)
	if err != nil {
		logger.Log.Error(err.Error())
	}
	h.red.Set(context.Background(), fmt.Sprintf("session:%s", data), userSession_uuid, 12*time.Hour)
	h.red.SAdd(context.Background(), fmt.Sprintf("user:%s", strconv.Itoa(user_id)), userSession_uuid)

	w.WriteHeader(200)
}

func (h *handler_struct) IsTokenValid(w http.ResponseWriter, r *http.Request) { // Должен вызываться первым при загрузке страницы, чтобы проверить токен на валидность

	query := r.URL.Query()
	token := query.Get("token")
	if token == "" {
		http.Error(w, "Missing or invalid token", http.StatusBadRequest)
		return
	}

	if _, err := h.serv.TokenVerifier(token); err != nil {
		// TODO Добавить логирование
		http.Error(w, "Missing or invalid token", http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]int{"status": http.StatusOK})
}

func (h *handler_struct) ResetPassword(w http.ResponseWriter, r *http.Request) {
	req := new(models.Reset_Password_Model)

	if err := json.NewDecoder(r.Body).Decode(req); err != nil {
		return
	} // Это нужно для дополнительной проверки валидности, на всякий случай
	defer r.Body.Close()

	if req.Token == "" {
		http.Error(w, "Missing or invalid token", http.StatusBadRequest)
		return
	}
	id, err := h.serv.TokenVerifier(req.Token)
	if err != nil {
		http.Error(w, "Token is not valid", http.StatusBadRequest)
		return
	}
	if err := h.serv.ResetPasswordService(id, req.Password); err != nil {
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (h *handler_struct) RateLimiterMiddleware(rps int, burst int) func(http.Handler) http.Handler {
	limiters := make(map[string]*rate.Limiter)
	mu := sync.Mutex{}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := r.RemoteAddr
			if forwarded := r.Header.Get("X-Real-IP"); forwarded != "" {
				ip = forwarded
			}
			mu.Lock()
			limiter, exist := limiters[ip]
			if !exist {
				limiter = rate.NewLimiter(rate.Limit(rps), burst)
				limiters[ip] = limiter
			}
			mu.Unlock()

			if !limiter.Allow() {
				w.Header().Set("Retry-After", "1")
				http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func normalizeToStringSlice(v interface{}) []string { // Для преобразования string в []string
	switch val := v.(type) {
	case string:
		if val == "" {
			return []string{}
		}
		return []string{val}
	case []string:
		return val
	default:
		return []string{}
	}
}

func DelCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     "session_id",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		Expires:  time.Now().Add(-1 * time.Hour),
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	})
}

// TODO Сделать проверку действительности куки, на ее срок жизни

func (h *handler_struct) SessionCheckMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sessionCookie, err := r.Cookie("session_id")
		if err != nil {
			next.ServeHTTP(w, r) // Кука не найдена, возвращаем управление, но только гостевой режим
			return
		}
		var user_id int
		key := fmt.Sprintf("session:%s", sessionCookie.Value)

		redis_value, err := h.red.Get(context.Background(), key).Result()
		if err == redis.Nil { // записи в редис нет, переход к проверке в бд
			//fmt.Printf("failed to set data, error: %s", err.Error())
			req, err := h.serv.VerifySession(sessionCookie.Value) // получение данных из бд
			if err != nil {
				DelCookie(w)
				next.ServeHTTP(w, r)
				return
			}
			user_id = req.User_id
			data, err := json.Marshal(req)
			if err != nil {
				next.ServeHTTP(w, r)
				return
			}

			if err := h.red.Set(context.Background(), key, data, 12*time.Hour).Err(); err != nil { // добавление записи сессии в редис
			}
		} else if err != nil {
			next.ServeHTTP(w, r)
			return
		} else {
			var session_value models.Session_Check_Model

			if err := json.Unmarshal([]byte(redis_value), &session_value); err != nil {
				next.ServeHTTP(w, r)
				return
			}
			if !session_value.Is_active {
				DelCookie(w)
				next.ServeHTTP(w, r)
				return
			}
			if session_value.Expires_at.Before(time.Now()) {
				DelCookie(w)
				if err := h.serv.DelSession(sessionCookie.Value); err != nil {
					// TODO возвращать нормальные ошибки и логировать
				}
				h.red.Del(context.Background(), key).Result()
				next.ServeHTTP(w, r)
				return
			}
			if int(time.Until(session_value.Expires_at.UTC()).Hours()/24) <= 7 { // сессия не должна быть меньше 7 дней, иначе обновление срока жизни
				if err := h.serv.UpadateExpiresSession(sessionCookie.Value); err != nil {

				}
				session_value.Expires_at = time.Now().UTC().Add(720 * time.Hour)
				update_session, err := json.Marshal(session_value)
				if err == nil {
					h.red.Set(context.Background(), key, update_session, 12*time.Hour)
				}
			}
			user_id = session_value.User_id
		}
		ctx := context.WithValue(r.Context(), "user_id", user_id) // Возвращаем id пользователя через контекст
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
