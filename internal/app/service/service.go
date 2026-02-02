package service

import (
	"errors"
	"gobackend/internal/app/email"
	"gobackend/internal/app/models"
	"gobackend/internal/app/repository"
	"gobackend/pkg"
	"mime/multipart"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type service_struct struct {
	repo         repository.Repo
	email_sender email.SMPTSender
}

var allowedExts = map[string]bool{
	".png":  true,
	".jpg":  true,
	".jpeg": true,
	".webp": true,
}

type Serv interface {
	RegisterUser(register_data *models.Register_Model) error
	LoginUser(login_data *models.Login_Model) (uuid.UUID, error)
	AvatarCheck(files []*multipart.FileHeader, id int) error
	CatalogCheckGuest(pageStr, limitStr string) ([]models.Response_For_Guests_Model, error)
	CatalogCheckAuthUser(pageStr, limitStr string, id int) ([]models.Response_For_AuthUser_Model, error)
	GetComponents() (*models.Components, error)
	ReqPasswordReset(email string) error
	TokenVerifier(token string) (int, error)
	ResetPasswordService(id int, password string) error
	VerifySession(session string) (*models.Session_Check_Model, error)
	DelSession(session string) error
	UpadateExpiresSession(session string) error
}

func NewService(repo repository.Repo, email_sender email.SMPTSender) Serv {
	return &service_struct{repo: repo, email_sender: email_sender}
}

func (s *service_struct) RegisterUser(register_data *models.Register_Model) error { //  TODO сделать минимальную валидацию почты, имени и фамилии, пароля
	if err := s.repo.CreateUser(register_data); err != nil {
		return err
	}

	return nil
}

func (s *service_struct) AvatarCheck(files []*multipart.FileHeader, id int) error {

	if len(files) == 0 {
		return errors.New("file not transferred")
	}
	if len(files) > 1 {
		return errors.New("only one file can be uploaded")
	}

	handler := files[0]
	file, err := handler.Open()
	if err != nil {
		return errors.New("failed to open file")
	}

	defer file.Close()

	if handler.Size > 5<<20 {
		return errors.New("file too large (max 5 MB)")
	}

	ext := filepath.Ext(handler.Filename)
	if ext == "" {
		return errors.New("file has no extension")
	}

	if !allowedExts[strings.ToLower(ext)] {
		return errors.New("Only PNG, JPEG, JPG, WEBP are allowed")
	}

	uuid_photo := uuid.New().String() + ext

	save_photo, err := os.Create(filepath.Join("/var/www/i.core-spark/images/avatars", uuid_photo)) // TODO доделать и проверить работоспособность
	if err != nil {
		return errors.New("error reading file")
	}

	defer save_photo.Close()

	_, err = save_photo.ReadFrom(file)
	if err != nil {
		return errors.New("Error saving file")
	}

	if err := s.repo.AddAvatar(filepath.Join("/images/avatar/", uuid_photo), id); err != nil {
		return err
	}

	return nil
}

func (s *service_struct) GetComponents() (*models.Components, error) {
	items, err := s.repo.Components()
	if err != nil {
		return nil, err
	}
	return items, nil
}

func (s *service_struct) CatalogCheckGuest(pageStr, limitStr string) ([]models.Response_For_Guests_Model, error) {
	var page, limit int

	if pageStr == "" {
		page = 1
	} else {
		var err error
		page, err = strconv.Atoi(pageStr)
		if err != nil {
			return nil, err
		}
	}

	if limitStr == "" {
		limit = 10
	} else {
		var err error
		limit, err = strconv.Atoi(limitStr)
		if err != nil {
			return nil, err
		}
	}

	items, err := s.repo.LoadCatalogGuest(page, limit)
	if err != nil {
		return nil, err
	}

	return items, nil
}

func (s *service_struct) CatalogCheckAuthUser(pageStr, limitStr string, id int) ([]models.Response_For_AuthUser_Model, error) {
	var page, limit int

	if pageStr == "" {
		page = 1
	} else {
		var err error
		page, err = strconv.Atoi(pageStr)
		if err != nil {
			return nil, err
		}
	}

	if limitStr == "" {
		limit = 10
	} else {
		var err error
		limit, err = strconv.Atoi(limitStr)
		if err != nil {
			return nil, err
		}
	}

	items, err := s.repo.LoadCatalogAuthUser(page, limit, id)
	if err != nil {
		return nil, err
	}

	return items, nil
}

func (s *service_struct) LoginUser(login_data *models.Login_Model) (uuid.UUID, error) {
	userSession_uuid, err := s.repo.LoginUser(login_data)
	if err != nil {
		return uuid.Nil, err
	}
	return userSession_uuid, nil
}

func (s *service_struct) TokenVerifier(token string) (int, error) {
	result, err := s.repo.TokenVerification(token)
	if err != nil {
		return 0, err
	}
	if time.Now().UTC().After(result.Expires_At) {
		return 0, errors.New("xz") // TODO придумать ошибку
	}
	return result.ID, nil
}

func (s *service_struct) ResetPasswordService(id int, password string) error {
	hash_password, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	err = s.repo.ResetPasswordRepository(string(hash_password), id)
	if err != nil {
		return err
	}
	return nil
}

func (s *service_struct) VerifySession(session string) (*models.Session_Check_Model, error) {
	req, err := s.repo.CheckSession(session) // Сделать проверку срока годности токенаЫ
	if err != nil {
		return nil, err
	}
	return req, nil
}

func (s *service_struct) DelSession(session string) error {
	return s.repo.DeleteSession(session)
}

func (s *service_struct) UpadateExpiresSession(session string) error {
	return s.repo.UpdateExpiresAtSession(session)
}

func (s *service_struct) ReqPasswordReset(email string) error { // Запрос на изменение пароля, с отправкой ссылки на почту
	user, err := s.repo.GetUserByEmail(email)
	if err != nil {
		return err
	}
	token, err := pkg.GenerateSecureToken()
	if err != nil {
		return err
	}

	if err := s.repo.RequestResetPassword(user.ID, token); err != nil {
		return err
	}

	resetLink := os.Getenv("APP_URL") + "/reset-password?token=" + token
	err = s.email_sender.SendPasswordReset(email, resetLink)
	if err != nil {
		return err
	}
	return nil
}
