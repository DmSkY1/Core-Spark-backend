package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"gobackend/internal/app/models"
	"gobackend/internal/app/repository"
	"gobackend/internal/app/service"
	"net/http"
	"strings"
)

type handler_struct struct {
	serv service.Serv
}

func NewHandler(serv service.Serv) *handler_struct {
	return &handler_struct{serv: serv}
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
	userid, ok := r.Context().Value("id").(int)
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
