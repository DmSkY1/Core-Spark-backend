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
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
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

func (h *handler_struct) CheckConfigByArticle(w http.ResponseWriter, r *http.Request) {
	article := chi.URLParam(r, "article")
	req, err := h.serv.CheckConfigByArticleService(article)
	if err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	if !req {
		http.Error(w, "Article not found", http.StatusNotFound)
		return
	}
	w.WriteHeader(http.StatusOK)
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
	search_string := query.Get("search")
	// limit := query.Get("limit") // лимит карточек товаров
	userid, ok := r.Context().Value("user_id").(int)
	if !ok {
		items, err := h.serv.CatalogCheckGuest(
			search_string,
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
			search_string,
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
			fmt.Print(err)
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(items); err != nil {
			return
		}
	}
}

func (h *handler_struct) GetAccountDashboard(w http.ResponseWriter, r *http.Request) {
	user_id, ok := r.Context().Value("user_id").(int)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	req, err := h.serv.GetAccountDashboardService(user_id)
	if err != nil {
		logger.Log.Error(err.Error())
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(200)
	json.NewEncoder(w).Encode(req)
}

func (h *handler_struct) GetAllOrders(w http.ResponseWriter, r *http.Request) {
	user_id, ok := r.Context().Value("user_id").(int)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	req, err := h.serv.GetAllOrdersService(user_id)
	if err != nil {
		logger.Log.Error(err.Error())
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(200)
	json.NewEncoder(w).Encode(req)
}

func (h *handler_struct) ChangeUserData(w http.ResponseWriter, r *http.Request) {
	var user_data models.Response_Change_Data
	json.NewDecoder(r.Body).Decode(&user_data)
	defer r.Body.Close()
	user_id, ok := r.Context().Value("user_id").(int)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	if err := h.serv.ChangeUserDataService(user_id, user_data.Name, user_data.Surname, user_data.Phone); err != nil {
		logger.Log.Error(err.Error())
		http.Error(w, "Unauthorized", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(200)
}

func (h *handler_struct) ChangePasswordProfile(w http.ResponseWriter, r *http.Request) {
	var get_passwords models.Response_Change_Password
	json.NewDecoder(r.Body).Decode(&get_passwords)
	defer r.Body.Close()
	fmt.Println(get_passwords.Old_Password)
	fmt.Println(get_passwords.New_Password)
	user_id, ok := r.Context().Value("user_id").(int)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	if err := h.serv.ChangePasswordProfileService(user_id, get_passwords.Old_Password, get_passwords.New_Password); err != nil {
		logger.Log.Error(err.Error())
		http.Error(w, "Unauthorized", http.StatusUnprocessableEntity)
		return
	}

	w.WriteHeader(200)
}

func (h *handler_struct) GetInfoOrder(w http.ResponseWriter, r *http.Request) {
	var req_info_order models.Request_Order_Info
	user_id, ok := r.Context().Value("user_id").(int)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	json.NewDecoder(r.Body).Decode(&req_info_order)
	defer r.Body.Close()

	req, err := h.serv.GetInfoOrderService(user_id, req_info_order.Order_code)
	if err != nil {
		logger.Log.Error(err.Error())
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(200)
	json.NewEncoder(w).Encode(req)
}

func (h *handler_struct) AddCart(w http.ResponseWriter, r *http.Request) {
	user_id, ok := r.Context().Value("user_id").(int)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	var add_cartModel models.Universal_Model_Cart
	json.NewDecoder(r.Body).Decode(&add_cartModel)
	defer r.Body.Close()

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
	defer r.Body.Close()

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
		return
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

func (h *handler_struct) AddOrder(w http.ResponseWriter, r *http.Request) {
	var pickUpPoint models.Pick_Up_Point_Order
	user_id, ok := r.Context().Value("user_id").(int)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	json.NewDecoder(r.Body).Decode(&pickUpPoint)
	defer r.Body.Close()

	code, err := h.serv.AddOrderService(user_id, pickUpPoint.Pick_Up_Point_ID)
	if err != nil {
		logger.Log.Error(err.Error())
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(map[string]string{
		"order_code": code,
	})
	w.WriteHeader(200)
}

func (h *handler_struct) LogOutUser(w http.ResponseWriter, r *http.Request) {
	// TODO сделать нормальное логирование
	user_id, ok := r.Context().Value("user_id").(int)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	session, err := r.Cookie("session_id")
	if err != nil {
		logger.Log.Error(err.Error())
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	key := fmt.Sprintf("session:%s", session.Value)

	err = h.serv.LogOut(user_id, session.Value)
	if err != nil {
		logger.Log.Error(err.Error())
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	DelCookie(w)

	h.red.Del(context.Background(), key)
	w.WriteHeader(200)
}

func (h *handler_struct) UpdateCartItemQuantity(w http.ResponseWriter, r *http.Request) {
	var res models.Update_Cart_Items_Quantity
	json.NewDecoder(r.Body).Decode(&res)
	defer r.Body.Close()

	user_id, ok := r.Context().Value("user_id").(int)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
	}

	if err := h.serv.UpdateCartItemQuantityService(user_id, res.ID_config, res.Num); err != nil {
		logger.Log.Error(err.Error())
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (h *handler_struct) SavePickUpPointUser(w http.ResponseWriter, r *http.Request) {
	var pick_up_point_id models.Response_Pick_Up_Point
	user_id, ok := r.Context().Value("user_id").(int)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	json.NewDecoder(r.Body).Decode(&pick_up_point_id)
	defer r.Body.Close()

	if err := h.serv.SavePickUpPoint(user_id, pick_up_point_id.ID); err != nil {
		logger.Log.Error(err.Error())
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(200)
}

func (h *handler_struct) GetPickUpPoints(w http.ResponseWriter, r *http.Request) {
	user_id, ok := r.Context().Value("user_id").(int)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	req, err := h.serv.GetPickUpPoints(user_id)
	if err != nil {
		logger.Log.Error(err.Error())
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(200)
	json.NewEncoder(w).Encode(req)
}

func (h *handler_struct) AddConfigToCart(w http.ResponseWriter, r *http.Request) {
	var config models.User_Config_Model
	user_id, ok := r.Context().Value("user_id").(int)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
	}
	json.NewDecoder(r.Body).Decode(&config)
	defer r.Body.Close()

	if err := h.serv.AddCustomConfigToCartService(user_id, config); err != nil {
		logger.Log.Error(err.Error())
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

func (h *handler_struct) RequestPasswordReset(w http.ResponseWriter, r *http.Request) {
	type Email struct {
		Email string `json:"email"`
	}
	user_mail := new(Email)
	json.NewDecoder(r.Body).Decode(user_mail)
	defer r.Body.Close()

	err := h.serv.ReqPasswordReset(user_mail.Email)
	if err != nil {
		logger.Log.Error(err.Error())
		http.Error(w, "failed to process request", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (h *handler_struct) Components(w http.ResponseWriter, r *http.Request) {
	var items *models.Components
	data, err := h.red.Get(context.Background(), "configurator:components").Bytes()
	if err != nil {
		logger.Log.Error(err.Error())
		items, err = h.serv.GetComponents()
		if err != nil {
			logger.Log.Error(err.Error())
			return
		}
	}
	if err := msgpack.Unmarshal(data, &items); err != nil {
		logger.Log.Error(err.Error())
	}

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

	w.WriteHeader(http.StatusOK)

}

func (h *handler_struct) UpdatePhoneNumber(w http.ResponseWriter, r *http.Request) {
	var phone_response models.ResponseUpdatePhoneModel
	json.NewDecoder(r.Body).Decode(&phone_response)
	defer r.Body.Close()
	user_id, ok := r.Context().Value("user_id").(int)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	if err := h.serv.UpdatePhone(user_id, phone_response.Phone); err != nil {
		http.Error(w, "Incorrect data", http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (h *handler_struct) GetProductInfo(w http.ResponseWriter, r *http.Request) {
	user_id, ok := r.Context().Value("user_id").(int)
	if !ok {
		user_id = 0
	}

	article := r.URL.Query().Get("article")

	req, err := h.serv.GetProductInfoService(article, user_id)
	if err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(&req)
	w.WriteHeader(http.StatusOK)
}

func (h *handler_struct) GetComponentsPC(w http.ResponseWriter, r *http.Request) {
	var id models.Comparison_Request_Model

	user_id, ok := r.Context().Value("user_id").(int)
	if !ok {
		user_id = 0
	}

	if err := json.NewDecoder(r.Body).Decode(&id); err != nil {
		http.Error(w, "Incorrect IDs", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	fmt.Println(id.ID)

	pc, err := h.serv.GettingPCForComparisonService(id.ID, user_id)
	if err != nil {
		http.Error(w, "Incorrect data", http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(pc)
}

func (h *handler_struct) CheckAuth(w http.ResponseWriter, r *http.Request) {
	_, ok := r.Context().Value("user_id").(int)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	w.WriteHeader(200)
}

func (h *handler_struct) Login(w http.ResponseWriter, r *http.Request) {
	userLogin := new(models.Login_Model)

	if err := json.NewDecoder(r.Body).Decode(userLogin); err != nil {
		return
	}
	defer r.Body.Close()

	userSession_uuid, _, err := h.serv.LoginUser(userLogin)
	if err != nil {
		http.Error(w, "Incorrect login or password", http.StatusUnauthorized)
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     "session_id",
		Value:    userSession_uuid.String(),
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		Expires:  time.Now().Add(720 * time.Hour),
	})

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
		logger.Log.Error(err.Error())
		http.Error(w, "Token not found", http.StatusInternalServerError)
		return
	} // Это нужно для дополнительной проверки валидности, на всякий случай
	defer r.Body.Close()

	if req.Token == "" {
		http.Error(w, "Missing or invalid token", http.StatusBadRequest)
		return
	}
	if req.Password == "" {
		http.Error(w, "The password field is empty", http.StatusBadRequest)
		return
	}
	id, err := h.serv.TokenVerifier(req.Token)
	if err != nil {
		http.Error(w, "Token is not valid", http.StatusBadRequest)
		return
	}
	if err := h.serv.ResetPasswordService(id, req.Password); err != nil {
		logger.Log.Error(err.Error())
		http.Error(w, "Token is not valid", http.StatusInternalServerError)
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

			if err := h.red.Set(context.Background(), key, data, 12*time.Hour).Err(); err != nil {
				logger.Log.Error(err.Error())
				return
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
			// Проверка валидно ли время жизни сессии. Если нынешнее время до истеения сесии- то все гуд, иначе удаление куки и запимей с сессией
			if session_value.Expires_at.Before(time.Now()) {
				DelCookie(w)
				if err := h.serv.DelSession(sessionCookie.Value); err != nil {
					logger.Log.Error(err.Error())
					return
				}
				h.red.Del(context.Background(), key).Result()
				next.ServeHTTP(w, r)
				return
			}

			user_id = session_value.User_id
		}
		ctx := context.WithValue(r.Context(), "user_id", user_id) // Возвращаем id пользователя через контекст
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
