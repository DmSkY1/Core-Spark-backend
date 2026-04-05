package service

import (
	"errors"
	"fmt"
	"gobackend/internal/app/email"
	"gobackend/internal/app/models"
	"gobackend/internal/app/repository"
	"gobackend/pkg"
	"gobackend/pkg/logger"
	"image"
	_ "image/jpeg"
	_ "image/png"
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
}

const (
	minWidth  = 200
	minHeight = 200
	maxWidth  = 800
	maxHeight = 800
)

type Serv interface {
	RegisterUser(register_data *models.Register_Model) error
	LoginUser(login_data *models.Login_Model) (uuid.UUID, int, error)
	AvatarCheck(files []*multipart.FileHeader, id int) error
	CatalogCheckGuest(pageStr, limitStr, price string, id int, order string, category, ram, gpu, cpu []string) ([]models.Response_For_Guests_Model, error)
	CatalogCheckAuthUser(pageStr, limitStr, price string, id int, order string, category, ram, gpu, cpu []string) ([]models.Response_For_AuthUser_Model, error)
	GetComponents() (*models.Components, error)
	ReqPasswordReset(email string) error
	TokenVerifier(token string) (int, error)
	ResetPasswordService(id int, password string) error
	VerifySession(session string) (*models.Session_Check_Model, error)
	DelSession(session string) error
	UpadateExpiresSession(session string) error
	GetUserProfileService(id int) (*models.Profile_Model, error)
	AddCartService(id, config_id int) error
	UpdateCartItemQuantityService(user_id, config_id, num int) error
	RemoveFromCartService(id, config_id int) error
	CartItemsService(user_id int) ([]models.Cart_Item, error)
	AddCustomConfigToCartService(id int, config models.User_Config_Model) error
	SearchGuestService(ram, gpu, cpu, category []string, price, search_string string, pageStr, limitStr string, order string) ([]models.Response_For_Guests_Model, error)
	SearchAuthUserService(ram, gpu, cpu, category []string, price, search_string string, id int, pageStr, limitStr string, order string) ([]models.Response_For_AuthUser_Model, error)
	GettingPCForComparisonService(pc_id []int, user_id int) (*[]models.PC_model, error)
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

func (s *service_struct) GetUserProfileService(id int) (*models.Profile_Model, error) {
	user, err := s.repo.GetUserProfile(id)
	if err != nil {
		logger.Log.Error(err.Error())
		return nil, err
	}
	return user, nil
}

func (s *service_struct) AvatarCheck(files []*multipart.FileHeader, id int) error {

	if len(files) == 0 {
		return errors.New("file not transferred")
	}
	if len(files) > 1 { //
		return errors.New("only one file can be uploaded")
	}

	handler := files[0]         // Берем 1 файл из массива файлов
	file, err := handler.Open() // открываем файл
	if err != nil {
		return errors.New("failed to open file")
	}
	defer file.Close()

	if handler.Size > 700*1024 { // проверка на размер файла, он не должен быть большк 700 кб
		return errors.New("file too large (max 700 KB)")
	}

	ext := filepath.Ext(handler.Filename) // Получние расширения файла,
	if ext == "" {
		return errors.New("file has no extension")
	}

	if !allowedExts[strings.ToLower(ext)] { // проверка на доступыне расширения
		return errors.New("Only PNG, JPEG, JPG are allowed")
	}

	imgCfg, _, err := image.DecodeConfig(file) // читаем конфиг фото (ширина высота)
	if err != nil {
		fmt.Println(err)
		return err
	}

	if _, err := file.Seek(0, 0); err != nil { // возвращаем указатель файла в начало, для полного его прочтения
		logger.Log.Error(err.Error())
		return err
	}

	if imgCfg.Width < minWidth || imgCfg.Height < minHeight { // проверка на мин ширину и длину
		return fmt.Errorf("image too small (minimum %dx%d pixels)", minWidth, minHeight)
	}

	if imgCfg.Width > maxWidth || imgCfg.Height > maxHeight { // проверка на макс ширину и длину
		return fmt.Errorf("image too large (maximum %dx%d pixels)", maxWidth, maxHeight)
	}

	uuid_photo := uuid.New().String() + ext // генерация имени файла

	save_photo, err := os.Create(filepath.Join("/home/dmsky/", uuid_photo)) // создание файла пустышки
	if err != nil {
		return errors.New("error creating file")
	}
	defer save_photo.Close()

	_, err = save_photo.ReadFrom(file) // заполнение файла данными -> становится фотографией
	if err != nil {
		return errors.New("Error saving file")
	}
	if err := s.repo.AddAvatar(filepath.Join("/images/avatar/", uuid_photo), id); err != nil { // запрос на добавление пути к фото для пользователя
		logger.Log.Error(err.Error())
		return fmt.Errorf("internal server error")
	}

	return nil
}

func (s *service_struct) AddCustomConfigToCartService(id int, config models.User_Config_Model) error {
	err := s.repo.AddCustomConfigToCart(id, config)
	if err != nil {
		return err
	}
	return nil
}

func (s *service_struct) GettingPCForComparisonService(pc_id []int, user_id int) (*[]models.PC_model, error) {
	if len(pc_id) == 0 || len(pc_id) > 3 {
		return nil, errors.New("The array with the identifier cannot be empty or greater than 3.")
	}

	for _, value := range pc_id {
		if value <= 0 {
			return nil, errors.New("Incorrect IDs")
		}
	}

	pc, err := s.repo.GettingPCForComparison(pc_id, user_id)
	if err != nil {
		return nil, err
	}

	return pc, nil
}

func (s *service_struct) GetComponents() (*models.Components, error) {
	items, err := s.repo.Components()
	if err != nil {
		return nil, err
	}
	return items, nil
}

func (s *service_struct) SearchAuthUserService(ram, gpu, cpu, category []string, price, search_string string, id int, pageStr, limitStr string, order string) ([]models.Response_For_AuthUser_Model, error) {
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
	req, err := s.repo.SearchItemsAuthUser(ram, gpu, cpu, category, price, search_string, id, page, limit, order)
	if err != nil {
		return nil, err
	}
	return req, nil
}

func (s *service_struct) SearchGuestService(ram, gpu, cpu, category []string, price, search_string string, pageStr, limitStr string, order string) ([]models.Response_For_Guests_Model, error) {
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
	req, err := s.repo.SearchItemsGuest(ram, gpu, cpu, category, price, search_string, page, limit, order)
	if err != nil {
		return nil, err
	}
	return req, nil
}

func (s *service_struct) CatalogCheckGuest(pageStr, limitStr, price string, id int, order string, category, ram, gpu, cpu []string) ([]models.Response_For_Guests_Model, error) {
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

	items, err := s.repo.LoadCatalogGuest(page, limit, id, order, category, ram, gpu, cpu, price)
	if err != nil {
		return nil, err
	}

	return items, nil
}

func (s *service_struct) UpdateCartItemQuantityService(user_id, config_id, num int) error {
	if err := s.repo.UpdateCartItemQuantity(user_id, config_id, num); err != nil {
		return err
	}
	return nil
}

func (s *service_struct) AddCartService(id, config_id int) error {
	err := s.repo.AddCart(id, config_id)
	if err != nil {
		return err
	}
	return nil
}

func (s *service_struct) CartItemsService(user_id int) ([]models.Cart_Item, error) {
	req, err := s.repo.CartItems(user_id)
	if err != nil {
		return nil, err
	}
	return req, nil
}

func (s *service_struct) RemoveFromCartService(id, config_id int) error {
	err := s.repo.RemoveFromCart(id, config_id)
	if err != nil {
		return err
	}
	return nil
}

func (s *service_struct) CatalogCheckAuthUser(pageStr, limitStr, price string, id int, order string, category, ram, gpu, cpu []string) ([]models.Response_For_AuthUser_Model, error) {
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
	var err error
	limit, err = strconv.Atoi(limitStr)
	if err != nil {
		return nil, err
	}

	items, err := s.repo.LoadCatalogAuthUser(page, limit, id, order, category, ram, gpu, cpu, price)
	if err != nil {
		return nil, err
	}

	return items, nil
}

func (s *service_struct) LoginUser(login_data *models.Login_Model) (uuid.UUID, int, error) {
	userSession_uuid, user_id, err := s.repo.LoginUser(login_data)
	if err != nil {
		logger.Log.Info(err.Error())
		return uuid.Nil, 0, err
	}
	return userSession_uuid, user_id, nil
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
	if req.Expires_at.Before(time.Now()) {
		return nil, errors.New("Session is not valid")
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
