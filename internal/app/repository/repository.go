package repository

import (
	"context"
	"errors"
	"fmt"
	"gobackend/internal/app/models"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/sync/errgroup"
)

type repository_struct struct {
	db *pgxpool.Pool
}

type Repo interface {
	GetUserById(id int) (*models.User, error)
	ChangePasswordProfile(id int, new_password, old_password string) error
	CreateUser(register_data *models.Register_Model) error
	LoginUser(login_data *models.Login_Model) (uuid.UUID, error)
	AddAvatar(avatar_path string, id int) error
	LoadCatalogGuest(page, limit int) ([]models.Response_For_Guests_Model, error)
	LoadCatalogAuthUser(page, limit, id int) ([]models.Response_For_AuthUser_Model, error)
	Components() (*models.Components, error)
	GetUserByEmail(email string) (*models.User, error)
	RequestResetPassword(id int, token string) error
	ResetPasswordRepository(password string, id int) error
	TokenVerification(token string) (*models.Token_Verification_Model, error)
	CheckSession(session string) (*models.Session_Check_Model, error)
	DeleteSession(session string) error
	UpdateExpiresAtSession(session string) error
}

func NewRepository(db *pgxpool.Pool) Repo {
	return &repository_struct{db: db}
}

var (
	UserExist = errors.New("this user already exists") // TODO добавлять ошибки по мере разработки
)

// addPc - добавление нового пк в каталог для админа
// getPC - получение всех пк и вывод их в каталог
// addCart - добавление товара в корзину
// quantityIncrement - увелечение количетсва еденицы товара в корзине пользователя
// delProductCart - удаление товара из корзину
// addOrder - оформление заказа
// savePickUpPoint - сохранение пункта выдачи
// dataChange - изменение телефона, почты, имени, фамилии пользователя
// filtr - поиск по филтрам
// addPcConfig - добавление конфига пк

func (r *repository_struct) CreateUser(register_data *models.Register_Model) error {
	// можно не проверять наличие почты в бд, если сразу после создания бы выполнить CREATE UNIQUE INDEX IF NOT EXISTS idx_users_email_unique ON users (email);
	//главное тут в коде правильно обработать эту ошибку
	hash_password, err := bcrypt.GenerateFromPassword([]byte(register_data.Password), 12)
	if err != nil {
		return err
	}

	tx, err := r.db.Begin(context.Background()) // создание транзакции, которая гарантирует что создастся и пользователь и корзина , или ничего
	if err != nil {
		return err
	}

	defer func() { // функция которая сработает при ошибке, и откатит транзакцию, т.к она сама этого не сделает
		if err != nil {
			_ = tx.Rollback(context.Background())
		}
	}()

	var userID int
	err = tx.QueryRow( //Создание пользователя
		context.Background(),
		`INSERT INTO users (name, surname, email, password, created_at) 
		VALUES ($1, $2, $3, $4, $5) 
		RETURNING id`,
		register_data.Name,
		register_data.Surname,
		register_data.Email,
		hash_password,
		time.Now().UTC(),
	).Scan(&userID)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" { // unique_violation
			return UserExist
		}
		return err
	}

	_, err = tx.Exec(context.Background(), `INSERT INTO cart (id_user) VALUES ($1)`, userID) // Добавление корзины для пользователя
	if err != nil {
		return err
	}

	err = tx.Commit(context.Background()) // Фиксирует транзакцию и закрывает ее
	if err != nil {
		return err
	}

	return nil
}

func (r *repository_struct) FetchAll(ctx context.Context, query string, scan func(pgx.Rows) error) error {
	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		if err := scan(rows); err != nil {
			return err
		}
	}
	return rows.Err()
}

func (r *repository_struct) Components() (*models.Components, error) {
	components := new(models.Components)
	g, ctx := errgroup.WithContext(context.Background()) // группа которая улавливает ошибки в горутинах, и с помощью контекта отменяет все процессы если есьт хотябы 1 ошибка

	g.Go(func() error { // Получение всех процессоров
		var items []models.Processor_Model
		err := r.FetchAll(ctx, `SELECT * FROM processor`, func(rows pgx.Rows) error {
			var item models.Processor_Model
			err := rows.Scan(
				&item.ID,
				&item.Manufacturer,
				&item.Photo,
				&item.Product_Line,
				&item.Model,
				&item.Socket,
				&item.Architecture,
				&item.Number_Cores,
				&item.Number_Threads,
				&item.Frequency,
				&item.TDP,
				&item.Max_TDP,
				&item.Ram_Standart,
				&item.Integrated_Graphics_Core,
				&item.Price,
			)
			if err == nil {
				items = append(items, item)
			}
			return err
		})
		if err == nil {
			components.CPU = items
		}
		return err
	})

	g.Go(func() error { // Получение вех мат.плат
		var items []models.Motherboard_Model
		err := r.FetchAll(ctx, `SELECT * FROM motherboard`, func(rows pgx.Rows) error {
			var item models.Motherboard_Model
			err := rows.Scan(
				&item.ID,
				&item.Name,
				&item.Photo,
				&item.Manufacturer,
				&item.Chipset,
				&item.Ram_Type,
				&item.Max_Ram,
				&item.Socket,
				&item.PCIE_x16_Port,
				&item.PCIE_x1_Port,
				&item.Wifi,
				&item.Audio_Codec,
				&item.Form_Factor,
				&item.Ram_Slots,
				&item.M2_Slots,
				&item.Sata_Slots,
				&item.Price,
			)
			if err == nil {
				items = append(items, item)
			}
			return err
		})
		if err == nil {
			components.MotherBoard = items
		}
		return err
	})

	g.Go(func() error { // Получение всех видео карт
		var items []models.Video_Card_Model
		err := r.FetchAll(ctx, `SELECT * FROM video_card`, func(rows pgx.Rows) error {
			var item models.Video_Card_Model
			err := rows.Scan(
				&item.ID,
				&item.Manufacturer,
				&item.Photo,
				&item.GPU_Manufacturer,
				&item.Series,
				&item.Price,
				&item.PCIE,
				&item.Video_Memory_Capacity,
				&item.HDMI,
				&item.DisplayPort,
				&item.Memory_Type,
				&item.GPU_Frequency,
				&item.Bandwidth,
				&item.Video_Memory_Frequency,
				&item.Consumption,
				&item.Memory_Bus,
			)
			if err == nil {
				items = append(items, item)
			}
			return err
		})
		if err == nil {
			components.GPU = items
		}
		return err
	})

	g.Go(func() error { // Получение всей оперативки
		var items []models.Ram_Model
		err := r.FetchAll(ctx, `SELECT * FROM ram`, func(rows pgx.Rows) error {
			var item models.Ram_Model
			err := rows.Scan(
				&item.ID,
				&item.Name,
				&item.Photo,
				&item.Brand,
				&item.Volume_One_Module,
				&item.Memory_Type,
				&item.Frequency,
				&item.Number_Modules,
				&item.Price,
			)
			if err == nil {
				items = append(items, item)
			}
			return err
		})
		if err == nil {
			components.RAM = items
		}
		return err
	})

	g.Go(func() error { // Получение всех sata ssd
		var items []models.Ssd_Sata_Model
		err := r.FetchAll(ctx, `SELECT * FROM ssd_sata`, func(rows pgx.Rows) error {
			var item models.Ssd_Sata_Model
			err := rows.Scan(
				&item.ID,
				&item.Photo,
				&item.Manufacturer,
				&item.Model,
				&item.Storage_Capacity,
				&item.Reading_Speed,
				&item.Write_Speed,
				&item.Rewrite_Resource,
				&item.Price,
			)
			if err == nil {
				items = append(items, item)
			}
			return err
		})
		if err == nil {
			components.SSD_SATA = items
		}
		return err
	})

	g.Go(func() error { // Получение всех ssd m2
		var items []models.Ssd_M2_Model
		err := r.FetchAll(ctx, `SELECT * FROM ssd_m2`, func(rows pgx.Rows) error {
			var item models.Ssd_M2_Model
			err := rows.Scan(
				&item.ID,
				&item.Photo,
				&item.Manufacturer,
				&item.Model,
				&item.PCIE,
				&item.Storage_Capacity,
				&item.Reading_Speed,
				&item.Write_Speed,
				&item.Rewrite_Resource,
				&item.Price,
			)
			if err == nil {
				items = append(items, item)
			}
			return err
		})
		if err == nil {
			components.SSD_M2 = items
		}
		return err
	})

	g.Go(func() error { // Получение всех hdd
		var items []models.Hdd_Model
		err := r.FetchAll(ctx, `SELECT * FROM hdd`, func(rows pgx.Rows) error {
			var item models.Hdd_Model
			err := rows.Scan(
				&item.ID,
				&item.Photo,
				&item.Manufacturer,
				&item.Form_Factor,
				&item.Model,
				&item.Storage_Capacity,
				&item.Rotation_Speed,
				&item.Price,
			)
			if err == nil {
				items = append(items, item)
			}
			return err
		})
		if err == nil {
			components.HDD = items
		}
		return err
	})

	g.Go(func() error { // Получение всех блоков питания
		var items []models.Power_Unit_Model
		err := r.FetchAll(ctx, `SELECT * FROM power_unit`, func(rows pgx.Rows) error {
			var item models.Power_Unit_Model
			err := rows.Scan(
				&item.ID,
				&item.Photo,
				&item.Manufacturer,
				&item.Model,
				&item.Power,
				&item.Has_Ocp,
				&item.Has_Ovp,
				&item.Has_Uvp,
				&item.Has_Scp,
				&item.Has_Opp,
				&item.Fan_Size,
				&item.Form_Factor,
				&item.Price,
			)
			if err == nil {
				items = append(items, item)
			}
			return err
		})
		if err == nil {
			components.PowerUnit = items
		}
		return err
	})

	g.Go(func() error { // Получение всех корпусов
		var items []models.Frame_Model
		err := r.FetchAll(ctx, `SELECT * FROM frame`, func(rows pgx.Rows) error {
			var item models.Frame_Model
			err := rows.Scan(
				&item.ID,
				&item.Photo,
				&item.Manufacturer,
				&item.Model,
				&item.Supports_Mini_Itx,
				&item.Supports_Micro_Atx,
				&item.Supports_Atx,
				&item.Supports_E_Atx,
				&item.Liquid_Cooling_System,
				&item.Fans_Included,
				&item.Maximum_Length_GPU,
				&item.Maximum_Cooler_Height,
				&item.Type_Size,
				&item.Price,
			)
			if err == nil {
				items = append(items, item)
			}
			return err
		})
		if err == nil {
			components.Frame = items
		}
		return err
	})

	g.Go(func() error { // Получение всех систем охлождения
		var items []models.Cooling_System_Models
		err := r.FetchAll(ctx, `SELECT * FROM cooling_system`, func(rows pgx.Rows) error {
			var item models.Cooling_System_Models
			err := rows.Scan(
				&item.ID,
				&item.Photo,
				&item.Manufacturer,
				&item.Model,
				&item.Type,
				&item.Sockets,
				&item.Dissipated_Power,
				&item.Price,
			)
			if err == nil {
				items = append(items, item)
			}
			return err
		})
		if err == nil {
			components.Cooling_System = items
		}
		return err
	})

	if err := g.Wait(); err != nil { // При возникновении ошибки в какой либо горутине попадает сюда
		return &models.Components{}, err
	}
	return components, nil
}

func (r *repository_struct) LoginUser(login_data *models.Login_Model) (uuid.UUID, error) {
	user := new(models.User)

	if err := r.db.QueryRow(context.Background(), `SELECT id, email, password FROM users WHERE email = $1`, login_data.Email).Scan(
		&user.ID, &user.Email, &user.Password,
	); err != nil {
		return uuid.Nil, err
	}

	if correct_password := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(login_data.Password)); correct_password != nil {
		return uuid.Nil, correct_password
	}
	session_uuid := uuid.New()
	if _, err := r.db.Exec(
		context.Background(),
		`INSERT INTO sessions (uuid, id_user, user_agent, created_at, expires_at, is_active) VALUES ($1, $2, $3, $4, $5, $6)`,
		session_uuid,
		user.ID,
		login_data.User_Agent,
		time.Now().UTC(),
		time.Now().Add(time.Hour*336).UTC(),
		true,
	); err != nil {
		fmt.Println(err)
		return uuid.Nil, err
	}

	return session_uuid, nil
}

func (r *repository_struct) AddAvatar(avatar_path string, id int) error { // Добавление аватара пользователя
	_, err := r.db.Exec(
		context.Background(),
		`UPDATE users SET avatar = $1 WHERE id = $2`,
		avatar_path, id,
	)
	if err != nil {
		return err
	}
	return nil
}

func (r *repository_struct) LoadCatalogAuthUser(page, limit, id int) ([]models.Response_For_AuthUser_Model, error) { // Загрузка каталога для авторизованного пользователя
	var items []models.Response_For_AuthUser_Model
	total := (page - 1) * limit
	rows, err := r.db.Query(
		context.Background(),
		`SELECT c.id, c.photo, c.category, c.name, p.manufacturer, p.product_line,  
		v.gpu_manufacturer, v.series, ((rm.volume_one_module * rm.number_modules) * rc.quantity) AS total_ram_gb, c.price,
		CASE WHEN cp.id IS NOT NULL THEN true ELSE false END AS in_cart 
		FROM config_pc c
		LEFT JOIN processor p ON c.id_processor = p.id
		LEFT JOIN video_card v ON c.id_video_card = v.id
		LEFT JOIN ram_config rc ON c.id_pc_ram_config = rc.id
		LEFT JOIN ram rm ON rc.id_ram = rm.id
		LEFT JOIN cart ct on ct.id_user = $3
		LEFT JOIN cart_pc cp ON cp.id_cart = ct.id AND cp.id_config = c.id 

		WHERE c.is_catalog = true
		ORDER BY c.id
		LIMIT $1 OFFSET $2`, limit, total, id,
	)
	if err != nil {
		return nil, err
	}

	for rows.Next() {
		var item models.Response_For_AuthUser_Model
		err := rows.Scan(
			&item.Id,
			&item.Photo,
			&item.Category,
			&item.Name,
			&item.Manufacturer,
			&item.Product_Line,
			&item.GPU_Manufacturer,
			&item.Series,
			&item.Total_Ram_GB,
			&item.Price,
			&item.In_Cart,
		)
		if err != nil {
			return nil, err
		}

		items = append(items, item)
	}
	return items, nil
}

func (r *repository_struct) LoadCatalogGuest(page, limit int) ([]models.Response_For_Guests_Model, error) { // Загрузка каталога для гостя

	var items []models.Response_For_Guests_Model
	total := (page - 1) * limit
	rows, err := r.db.Query(
		context.Background(),
		`SELECT c.id, c.photo, c.category, c.name, p.manufacturer, p.product_line,  
		v.gpu_manufacturer, v.series, ((rm.volume_one_module * rm.number_modules) * rc.quantity) AS total_ram_gb, c.price
		FROM config_pc c
		LEFT JOIN processor p ON c.id_processor = p.id
		LEFT JOIN video_card v ON c.id_video_card = v.id
		LEFT JOIN ram_config rc ON c.id_pc_ram_config = rc.id
		LEFT JOIN ram rm ON rc.id_ram = rm.id

		WHERE c.is_catalog = true
		ORDER BY c.id
		LIMIT $1 OFFSET $2`,
		limit,
		total,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var item models.Response_For_Guests_Model
		err := rows.Scan(
			&item.Id,
			&item.Photo,
			&item.Category,
			&item.Name,
			&item.Manufacturer,
			&item.Product_Line,
			&item.GPU_Manufacturer,
			&item.Series,
			&item.Total_Ram_GB,
			&item.Price,
		)

		if err != nil {
			fmt.Println(err)
			return nil, err
		}

		items = append(items, item)
	}

	return items, nil
}

func (r *repository_struct) RequestResetPassword(id int, token string) error { // запрос на сброс пароля
	_, err := r.db.Exec(
		context.Background(), // При запросе создается новая запись в котору. записывается токен и его срок годности, или если запись уже есть, то она просто перезаписывается на новые данные
		`INSERT INTO password_reset_tokens (user_id, token, expires_at, is_active)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (user_id)
		DO UPDATE SET
		token = EXCLUDED.token,
		expires_at = EXCLUDED.expires_at,
		is_active = EXCLUDED.is_active`,
		id, token, time.Now().Add(15*time.Minute).UTC(), true,
	)
	if err != nil {
		fmt.Println(err)
		return err
	}
	return nil
}

func (r *repository_struct) GetUserById(id int) (*models.User, error) { // Получение данных пользователя по id
	user := new(models.User)

	err := r.db.QueryRow(context.Background(),
		`SELECT id, name, surname, email, telephone, password, avatar, created_at, pick_up_point
			FROM users
			WHERE id = $1`, id).Scan(
		&user.ID,
		&user.Name,
		&user.Surname,
		&user.Email,
		&user.Telephone,
		&user.Password,
		&user.Avatar,
		&user.Created_at,
		&user.PickUpPoint,
	)

	if err != nil {
		return nil, err
	}

	return user, nil
}

func (r *repository_struct) GetUserByEmail(email string) (*models.User, error) { // Получение данных пользователя по почте
	user := new(models.User) // Модель для хранения данных пользователя
	err := r.db.QueryRow(context.Background(),
		`SELECT id, name, surname, email, telephone, password, avatar, created_at, pick_up_point
			FROM users
			WHERE email = $1`, email).Scan(
		&user.ID,
		&user.Name,
		&user.Surname,
		&user.Email,
		&user.Telephone,
		&user.Password,
		&user.Avatar,
		&user.Created_at,
		&user.PickUpPoint,
	)

	if err != nil {
		return nil, err
	}

	return user, nil
}

func (r *repository_struct) TokenVerification(token string) (*models.Token_Verification_Model, error) { // Проверка токена сброса пароля на достоверность
	result := new(models.Token_Verification_Model)

	err := r.db.QueryRow(context.Background(), `SELECT id, expires_at FROM password_reset_tokens WHERE token = $1`, token).Scan(&result.ID, &result.Expires_At)
	if err != nil {
		return nil, err
	}

	return result, nil
}

func (r *repository_struct) ResetPasswordRepository(password string, id int) error { // Метод для сброса пароля, который меняет пароль на новый
	_, err := r.db.Exec(context.Background(), `UPDATE users SET password = $1 WHERE id = $2`, password, id)
	if err != nil {
		return err
	}

	return nil
}

func (r *repository_struct) ChangePasswordProfile(id int, new_password, old_password string) error { // Смена пароля из профиля

	var get_old_password string
	if err := r.db.QueryRow(
		context.Background(),
		`SELECT password FROM users where id = $1`,
		id,
	).Scan(&get_old_password); err != nil {
		return err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(get_old_password), []byte(old_password)); err != nil {
		return errors.New("passwords don't match")
	}
	hash_new_password, err := bcrypt.GenerateFromPassword([]byte(new_password), bcrypt.DefaultCost) // TODO перенести это в сервис
	if err != nil {
		return err
	}

	_, err = r.db.Exec(
		context.Background(),
		`UPDATE users SET password = $1 WHERE id = $2`,
		string(hash_new_password), id,
	)
	if err != nil {
		return err
	}
	return nil // TODO этот метод еще не совсем корректный, в будущем надо его доделать
}

func (r *repository_struct) DeleteSession(session string) error {
	uuid_session, err := uuid.Parse(session)
	if err != nil {
		return err
	}
	_, err = r.db.Exec(context.Background(), `DELETE FROM sessions WHERE uuid = $1`, uuid_session)
	if err != nil {
		return err
	}
	return nil
}

func (r *repository_struct) UpdateExpiresAtSession(session string) error {
	uuid_session, err := uuid.Parse(session)
	if err != nil {
		return err
	}
	_, err = r.db.Exec(
		context.Background(),
		`UPDATE sessions SET expires_at = $1 WHERE uuid = $2`,
		time.Now().Add(720*time.Hour).UTC(),
		uuid_session,
	)
	if err != nil {
		return err
	}
	return nil
}

func (r *repository_struct) CheckSession(session string) (*models.Session_Check_Model, error) {
	session_model := new(models.Session_Check_Model)

	err := r.db.QueryRow(
		context.Background(),
		`SELECT uuid, id_user, created_at, expires_at, is_active FROM sessions WHERE uuid = $1`,
		session).Scan(
		&session_model.UUID,
		&session_model.User_id,
		&session_model.Created_at,
		&session_model.Expires_at,
		&session_model.Is_active,
	)
	if err != nil {
		return nil, err // TODO логировать
	}
	return session_model, nil
}
