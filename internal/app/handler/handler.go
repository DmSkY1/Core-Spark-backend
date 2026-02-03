package handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"gobackend/internal/app/models"
	"gobackend/internal/app/repository"
	"gobackend/internal/app/service"
	"net/http"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

type handler_struct struct {
	serv service.Serv
	red  *redis.Client
}

func NewHandler(serv service.Serv, red *redis.Client) *handler_struct {
	return &handler_struct{serv: serv, red: red}
}

func (h *handler_struct) UploadAvatar(w http.ResponseWriter, r *http.Request) {
	userid, ok := r.Context().Value("id").(int)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 6<<20)

	if err := r.ParseMultipartForm(6 << 20); err != nil {
		if strings.Contains(err.Error(), "request body too large") {
			w.WriteHeader(http.StatusRequestEntityTooLarge)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"code":   http.StatusRequestEntityTooLarge,
				"status": "request body too large (max 5-6 Mb)",
			})
		} else {
			http.Error(w, "invalid request format", http.StatusBadRequest)
		}
	}

	files := r.MultipartForm.File["photo"]

	if err := h.serv.AvatarCheck(files, userid); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

}

func (h *handler_struct) Catalog(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()      // получение query параметров
	page := query.Get("page")   // номер страницы
	limit := query.Get("limit") // лимит карточек товаров
	userid, ok := r.Context().Value("user_id").(int)
	if !ok {
		items, err := h.serv.CatalogCheckGuest(page, limit)
		if err != nil {
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(items); err != nil {
			return
		}

	} else {
		items, err := h.serv.CatalogCheckAuthUser(page, limit, userid)
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
	items, err := h.serv.GetComponents()
	if err != nil {
		fmt.Println(err)
		return
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

	w.Write([]byte("успех"))

}

func (h *handler_struct) Login(w http.ResponseWriter, r *http.Request) {
	userLogin := new(models.Login_Model)

	if err := json.NewDecoder(r.Body).Decode(userLogin); err != nil {
		return
	}
	defer r.Body.Close()

	userSession_uuid, err := h.serv.LoginUser(userLogin)
	if err != nil {
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     "session_id",
		Value:    userSession_uuid.String(),
		Path:     "/",
		HttpOnly: true,
		Secure:   false, // TODO в финале изменить на true
		MaxAge:   3600,
	})
	http.Redirect(w, r, "https://core-spark.space/", 200)
	w.Write([]byte(userSession_uuid.String()))
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
			fmt.Printf("failed to set data, error: %s", err.Error())
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
			if int(time.Until(session_value.Expires_at.UTC()).Hours()/24) <= 7 {
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
