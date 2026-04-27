package repository

import (
	"context"
	"errors"
	"fmt"
	"gobackend/internal/app/models"
	"gobackend/pkg/logger"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
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
	LoginUser(login_data *models.Login_Model) (uuid.UUID, int, error)
	AddAvatar(avatar_path string, id int) error
	LoadCatalogGuest(page, limit, id int, order string, category, ram, gpu, cpu []string, price string) ([]models.Response_For_Guests_Model, error)
	LoadCatalogAuthUser(page, limit, id int, order string, category, ram, gpu, cpu []string, price string) ([]models.Response_For_AuthUser_Model, error)
	Components() (*models.Components, error)
	GetUserByEmail(email string) (*models.User, error)
	RequestResetPassword(id int, token string) error
	ResetPasswordRepository(password string, id int) error
	TokenVerification(token string) (*models.Token_Verification_Model, error)
	CheckSession(session string) (*models.Session_Check_Model, error)
	DeleteSession(session string) error
	UpdateExpiresAtSession(session string) error
	AddCart(user_id int, config_id int) error
	GetUserProfile(id int) (*models.Profile_Model, error)
	RemoveFromCart(user_id, config_id int) error
	UpdateCartItemQuantity(user_id, config_id, num int) error
	CartItems(user_id int) ([]models.Cart_Item, error)
	SearchItemsGuest(ram, gpu, cpu, category []string, price, search_string string, page, limit int, order string) ([]models.Response_For_Guests_Model, error)
	SearchItemsAuthUser(ram, gpu, cpu, category []string, price, search_string string, id, page, limit int, order string) ([]models.Response_For_AuthUser_Model, error)
	AddCustomConfigToCart(id int, config models.User_Config_Model) (err error)
	GettingPCForComparison(pc_id []int, user_id int) (*[]models.PC_model, error)
	AddPhoneUser(id int, number_phone string) error
	LogOutProfile(id int, session string) error
	GetAllPickUpPoints(id int) (*[]models.PickUpPoint_Model, error)
	SavePickUpPointUser(user_id, pick_up_point_id int) error
	GetAccountDashboard(id int) (*models.AccountDashboard, error)
	GetAllOrders(id int) (*[]models.Order, error)
	GetInfoOrder(id int, order_code string) (*[]models.Order_Items, error)
	AddOrder(id, pick_up_point_id int, order_code string) (err error)
	ChangeUserData(id int, name, surname string, phone string) error
}

func NewRepository(db *pgxpool.Pool) Repo {
	return &repository_struct{db: db}
}

var (
	UserExist = errors.New("this user already exists")
)

func (r *repository_struct) CreateUser(register_data *models.Register_Model) (err error) {
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

func (r *repository_struct) LoginUser(login_data *models.Login_Model) (uuid.UUID, int, error) {
	user := new(models.User)

	if err := r.db.QueryRow(context.Background(), `SELECT id, email, password FROM users WHERE email = $1`, login_data.Email).Scan(
		&user.ID, &user.Email, &user.Password,
	); err != nil {
		logger.Log.Info(err.Error())
		return uuid.Nil, 0, err
	}

	if correct_password := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(login_data.Password)); correct_password != nil {
		logger.Log.Info(correct_password.Error())
		return uuid.Nil, 0, correct_password
	}
	session_uuid := uuid.New()
	if _, err := r.db.Exec(
		context.Background(),
		`INSERT INTO sessions (uuid, id_user, user_agent, created_at, expires_at, is_active, device_id) 
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (id_user, device_id) DO UPDATE SET
			uuid = EXCLUDED.uuid,             
    		user_agent = EXCLUDED.user_agent, 
    		created_at = EXCLUDED.created_at,  
    		expires_at = EXCLUDED.expires_at, 
    		is_active = TRUE; `,
		session_uuid,
		user.ID,
		login_data.User_Agent,
		time.Now().UTC(),
		time.Now().Add(time.Hour*720).UTC(),
		true,
		login_data.Device_Id,
	); err != nil {
		return uuid.Nil, 0, err
	}

	return session_uuid, user.ID, nil
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

func ParseRange(price string) (int, int) {
	part := strings.Split(price, "-")
	if len(part) != 2 {
		return 0, 0
	}
	minVal, err := strconv.Atoi(strings.TrimSpace(part[0]))
	if err != nil {
		return 0, 0
	}

	maxVal, err := strconv.Atoi(strings.TrimSpace(part[1]))
	if err != nil {
		return 0, 0
	}
	return minVal, maxVal
}

func buildFilterCondition(start_arg_num int, ram, gpu, cpu, category []string, price string) (args_num int, args []interface{}, wcondition []string) {
	args_num = start_arg_num
	wcondition = []string{}
	args = []interface{}{}
	if len(category) != 0 {
		placehold := make([]string, len(category))
		for i := range category {
			placehold[i] = fmt.Sprintf("$%d", args_num)
			args = append(args, category[i])
			args_num++
		}
		wcondition = append(wcondition, "(c.category IN ("+strings.Join(placehold, ", ")+"))")
	}
	if len(ram) != 0 {
		placehold := make([]string, len(ram))
		for i := range ram {
			placehold[i] = fmt.Sprintf("$%d", args_num)
			args = append(args, ram[i])
			args_num++
		}
		wcondition = append(wcondition, "(((rm.volume_one_module * rm.number_modules) * rc.quantity) IN ("+strings.Join(placehold, ", ")+"))")
	}
	if len(gpu) != 0 {
		placehold := make([]string, 0, len(gpu))
		for i := range gpu {
			placehold = append(placehold, fmt.Sprintf("v.series ILIKE $%d", args_num))
			args = append(args, "%"+gpu[i]+"%")
			args_num++
		}
		wcondition = append(wcondition, "("+strings.Join(placehold, " OR ")+")")
	}
	if len(cpu) != 0 {
		placehold := make([]string, 0, len(cpu))
		for i := range cpu {
			placehold = append(placehold, fmt.Sprintf("p.product_line ILIKE $%d", args_num))
			args = append(args, "%"+cpu[i]+"%")
			args_num++
		}
		wcondition = append(wcondition, "("+strings.Join(placehold, " OR ")+")")
	}

	if len(price) != 0 {
		min, max := ParseRange(price)
		wcondition = append(wcondition, fmt.Sprintf("c.price >= $%d", args_num))
		args = append(args, min)
		args_num++
		wcondition = append(wcondition, fmt.Sprintf("c.price <= $%d", args_num))
		args = append(args, max)
		args_num++
	}

	return args_num, args, wcondition
}

func (r *repository_struct) LoadCatalogAuthUser(page, limit, id int, order string, category, ram, gpu, cpu []string, price string) ([]models.Response_For_AuthUser_Model, error) { // Загрузка каталога для авторизованного пользователя

	args_num, args, wcondition := buildFilterCondition(1, ram, gpu, cpu, category, price) // определение основных объектов

	sql := `SELECT c.id, c.photo, c.category, c.name, p.manufacturer, p.product_line,  
		v.gpu_manufacturer, v.series, ((rm.volume_one_module * rm.number_modules) * rc.quantity) AS total_ram_gb, c.price,
		CASE WHEN cp.id IS NOT NULL THEN true ELSE false END AS in_cart,
		COUNT(*) OVER() AS total_count,
		c.article AS article
		FROM config_pc c
		LEFT JOIN processor p ON c.id_processor = p.id
		LEFT JOIN video_card v ON c.id_video_card = v.id
		LEFT JOIN ram_config rc ON c.id_pc_ram_config = rc.id
		LEFT JOIN ram rm ON rc.id_ram = rm.id
	`
	args = append(args, id) // добавлеине id в аргументы, для корректного формирования SQL скрипта
	sql += fmt.Sprintf(`
	LEFT JOIN cart ct on ct.id_user = $%d
	LEFT JOIN cart_pc cp ON cp.id_cart = ct.id AND cp.id_config = c.id 
	`, args_num)
	args_num++

	if len(wcondition) > 0 { // объединение всех условий в одно целое, и добавление в скрипт
		sql += " \nWHERE c.is_catalog = true AND " + strings.Join(wcondition, " AND ")
	}

	sortDir := "ASC" // значение по умолчанию, будет по возрастанию цены
	orderInt, err := strconv.Atoi(order)
	if err != nil {
		orderInt = 0
	}
	if orderInt == 1 {
		sortDir = "DESC"
	}
	sql += fmt.Sprintf(" \nORDER BY c.price %s", sortDir)
	total := (page - 1) * limit
	sql += fmt.Sprintf(" \nLIMIT $%d OFFSET $%d", args_num, args_num+1)
	args = append(args, limit, total)

	var items []models.Response_For_AuthUser_Model
	rows, err := r.db.Query(
		context.Background(), sql, args...)
	if err != nil {
		logger.Log.Error(err.Error())
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
			&item.Total_count,
			&item.Article,
		)
		if err != nil {
			return nil, err
		}

		items = append(items, item)
	}
	return items, nil
}

func (r *repository_struct) LogOutProfile(id int, session string) error {
	_, err := r.db.Exec(context.Background(), `UPDATE sessions SET is_active = false WHERE id_user = $1 AND uuid = $2`, id, session)
	if err != nil {
		return err
	}
	return nil
}

func (r *repository_struct) LoadCatalogGuest(page, limit, id int, order string, category, ram, gpu, cpu []string, price string) ([]models.Response_For_Guests_Model, error) { // Загрузка каталога для гостя

	// TODO Сократить код, вынести большинство повторяющегося кода в отдельные функции
	args_num, args, wcondition := buildFilterCondition(1, ram, gpu, cpu, category, price)

	sql := `SELECT c.id, c.photo, c.category, c.name, p.manufacturer, p.product_line,  
		v.gpu_manufacturer, v.series, ((rm.volume_one_module * rm.number_modules) * rc.quantity) AS total_ram_gb, c.price,
		COUNT(*) OVER() AS total_count,
		c.article AS article
		FROM config_pc c
		LEFT JOIN processor p ON c.id_processor = p.id
		LEFT JOIN video_card v ON c.id_video_card = v.id
		LEFT JOIN ram_config rc ON c.id_pc_ram_config = rc.id
		LEFT JOIN ram rm ON rc.id_ram = rm.id`

	if len(wcondition) > 0 {
		sql += " \nWHERE c.is_catalog = true AND " + strings.Join(wcondition, " AND ")
	}

	sortDir := "DESC"
	orderInt, err := strconv.Atoi(order)
	if err != nil {
		orderInt = 1
	}
	if orderInt == 0 || orderInt == 1 {
		sortDir = "ASC"
	}
	sql += fmt.Sprintf(" \nORDER BY c.price %s", sortDir)
	total := (page - 1) * limit
	sql += fmt.Sprintf(" \nLIMIT $%d OFFSET $%d", args_num, args_num+1)
	args = append(args, limit, total)

	var items []models.Response_For_Guests_Model
	rows, err := r.db.Query(
		context.Background(), sql, args...)
	if err != nil {
		logger.Log.Error(err.Error())
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
			&item.Total_count,
			&item.Article,
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

func (r *repository_struct) GetAllPickUpPoints(id int) (*[]models.PickUpPoint_Model, error) {
	var items []models.PickUpPoint_Model
	req, err := r.db.Query(
		context.Background(), `
		SELECT
			p.id,
			p.name,
			p.address,
			p.opening_hours,
			(SELECT pick_up_point_id FROM users WHERE id = $1) AS default_point
		FROM pick_up_point p
		ORDER BY p.id ASC`,
		id,
	)
	if err != nil {
		return nil, err
	}
	for req.Next() {
		var item models.PickUpPoint_Model
		err := req.Scan(
			&item.ID,
			&item.Name,
			&item.Address,
			&item.OpeningHours,
			&item.DefaultPoint,
		)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}

	return &items, nil
}

func (r *repository_struct) SavePickUpPointUser(user_id, pick_up_point_id int) error {
	_, err := r.db.Exec(context.Background(), `UPDATE users SET pick_up_point_id = $1 WHERE id = $2`, pick_up_point_id, user_id)
	if err != nil {
		return err
	}
	return nil
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

func (r *repository_struct) ChangeUserData(id int, name, surname string, phone string) error {
	_, err := r.db.Exec(context.Background(), "UPDATE users SET name = $1, surname = $2, telephone = $3 WHERE id = $4", name, surname, phone, id)
	if err != nil {
		return err
	}
	return nil
}

func (r *repository_struct) ChangePasswordProfile(id int, new_password, old_password string) error { // Смена пароля из профиля

	var get_old_password string
	if err := r.db.QueryRow(
		context.Background(),
		`SELECT password FROM users WHERE id = $1`,
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
	return nil
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

func (r *repository_struct) SearchItemsAuthUser(ram, gpu, cpu, category []string, price, search_string string, id, page, limit int, order string) ([]models.Response_For_AuthUser_Model, error) {

	// TODO Сократить код, вынести большинство повторяющегося кода в отдельные функции

	var items []models.Response_For_AuthUser_Model
	search_string = "%" + search_string + "%"
	args_num, args, wcondition := buildFilterCondition(1, ram, gpu, cpu, category, price)
	sql := `SELECT c.id, c.photo, c.category, c.name, p.manufacturer, p.product_line,  
		v.gpu_manufacturer, v.series, ((rm.volume_one_module * rm.number_modules) * rc.quantity) AS total_ram_gb, c.price,
		CASE WHEN cp.id IS NOT NULL THEN true ELSE false END AS in_cart,
		COUNT(*) OVER() AS total_count,
		c.article AS article
		FROM config_pc c
		LEFT JOIN processor p ON c.id_processor = p.id
		LEFT JOIN video_card v ON c.id_video_card = v.id
		LEFT JOIN ram_config rc ON c.id_pc_ram_config = rc.id
		LEFT JOIN ram rm ON rc.id_ram = rm.id
	`
	args = append(args, id) // добавлеине id в аргументы, для корректного формирования SQL скрипта
	sql += fmt.Sprintf(`
		LEFT JOIN cart ct on ct.id_user = $%d
		LEFT JOIN cart_pc cp ON cp.id_cart = ct.id AND cp.id_config = c.id 
		`,
		args_num,
	)
	args_num++

	args = append(args, search_string)
	args = append(args, search_string)
	wcondition = append(wcondition, fmt.Sprintf(`(c.name ILIKE $%d OR article ILIKE $%d)`, args_num, args_num+1))
	args_num += 2

	if len(wcondition) > 0 { // объединение всех условий в одно целое, и добавление в скрипт
		sql += " \nWHERE c.is_catalog = true AND " + strings.Join(wcondition, " AND ")
	}

	sortDir := "ASC" // значение по умолчанию, будет по возрастанию цены
	orderInt, err := strconv.Atoi(order)
	if err != nil {
		orderInt = 0
	}
	if orderInt == 1 {
		sortDir = "DESC"
	}
	sql += fmt.Sprintf(" \nORDER BY c.price %s", sortDir)
	total := (page - 1) * limit
	sql += fmt.Sprintf(" \nLIMIT $%d OFFSET $%d", args_num, args_num+1)
	args = append(args, limit, total)

	rows, err := r.db.Query(context.Background(), sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

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
			&item.Total_count,
			&item.Article,
		)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, nil
}

func (r *repository_struct) SearchItemsGuest(ram, gpu, cpu, category []string, price, search_string string, page, limit int, order string) ([]models.Response_For_Guests_Model, error) {

	// TODO Сократить код, вынести большинство повторяющегося кода в отдельные функции

	var items []models.Response_For_Guests_Model
	search_string = "%" + search_string + "%"
	args_num, args, wcondition := buildFilterCondition(1, ram, gpu, cpu, category, price)
	sql := `SELECT c.id, c.photo, c.category, c.name, p.manufacturer, p.product_line,  
		v.gpu_manufacturer, v.series, ((rm.volume_one_module * rm.number_modules) * rc.quantity) AS total_ram_gb, c.price,
		COUNT(*) OVER() AS total_count,
		c.article AS article
		FROM config_pc c
		LEFT JOIN processor p ON c.id_processor = p.id
		LEFT JOIN video_card v ON c.id_video_card = v.id
		LEFT JOIN ram_config rc ON c.id_pc_ram_config = rc.id
		LEFT JOIN ram rm ON rc.id_ram = rm.id`
	args = append(args, search_string)
	args = append(args, search_string)
	wcondition = append(wcondition, fmt.Sprintf(`(c.name ILIKE $%d OR article ILIKE $%d)`, args_num, args_num+1))
	args_num += 2

	if len(wcondition) > 0 { // объединение всех условий в одно целое, и добавление в скрипт
		sql += " \nWHERE c.is_catalog = true AND " + strings.Join(wcondition, " AND ")
	}

	sortDir := "ASC" // значение по умолчанию, будет по возрастанию цены
	orderInt, err := strconv.Atoi(order)
	if err != nil {
		orderInt = 0
	}
	if orderInt == 1 {
		sortDir = "DESC"
	}
	sql += fmt.Sprintf(" \nORDER BY c.price %s", sortDir)
	total := (page - 1) * limit
	sql += fmt.Sprintf(" \nLIMIT $%d OFFSET $%d", args_num, args_num+1)
	args = append(args, limit, total)

	fmt.Println(sql)
	rows, err := r.db.Query(context.Background(), sql, args...)
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
			&item.Total_count,
			&item.Article,
		)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, nil
}

func (r *repository_struct) GetPCByID(ctx context.Context, id int, sql string, user_id int) (models.PC_model, error) {
	var pc_components models.PC_model
	err := r.db.QueryRow(ctx, sql, id, user_id).Scan(
		&pc_components.ID_Config,
		&pc_components.Name,
		&pc_components.Photo,
		&pc_components.Price,
		&pc_components.Processor.Manufacturer,
		&pc_components.Processor.Product_Line,
		&pc_components.Processor.Model,
		&pc_components.Processor.Socket,
		&pc_components.Processor.Architecture,
		&pc_components.Processor.Number_Cores,
		&pc_components.Processor.Number_Threads,
		&pc_components.Processor.Frequency,
		&pc_components.Processor.TDP,
		&pc_components.Processor.Max_TDP,
		&pc_components.Processor.Ram_Standart,
		&pc_components.Processor.Integrated_Graphics_Core,
		&pc_components.Motherboard.Name,
		&pc_components.Motherboard.Manufacturer,
		&pc_components.Motherboard.Chipset,
		&pc_components.Motherboard.Ram_Type,
		&pc_components.Motherboard.Max_Ram,
		&pc_components.Motherboard.Socket,
		&pc_components.Motherboard.PCIE_x16_Port,
		&pc_components.Motherboard.PCIE_x1_Port,
		&pc_components.Motherboard.Wifi,
		&pc_components.Motherboard.Audio_Codec,
		&pc_components.Motherboard.Form_Factor,
		&pc_components.Motherboard.Ram_Slots,
		&pc_components.Motherboard.M2_Slots,
		&pc_components.Motherboard.Sata_Slots,
		&pc_components.GPU.Manufacturer,
		&pc_components.GPU.GPU_Manufacturer,
		&pc_components.GPU.Series,
		&pc_components.GPU.PCIE,
		&pc_components.GPU.Video_Memory_Capacity,
		&pc_components.GPU.HDMI,
		&pc_components.GPU.DisplayPort,
		&pc_components.GPU.Memory_Type,
		&pc_components.GPU.GPU_Frequency,
		&pc_components.GPU.Bandwidth,
		&pc_components.GPU.Video_Memory_Frequency,
		&pc_components.GPU.Consumption,
		&pc_components.GPU.Memory_Bus,
		&pc_components.RAM.Module.Name,
		&pc_components.RAM.Module.Brand,
		&pc_components.RAM.Module.Volume_One_Module,
		&pc_components.RAM.Module.Memory_Type,
		&pc_components.RAM.Module.Frequency,
		&pc_components.RAM.Module.Number_Modules,
		&pc_components.RAM.Quantity,
		&pc_components.SSD_M2.Module.Manufacturer,
		&pc_components.SSD_M2.Module.Model,
		&pc_components.SSD_M2.Module.PCIE,
		&pc_components.SSD_M2.Module.Storage_Capacity,
		&pc_components.SSD_M2.Module.Reading_Speed,
		&pc_components.SSD_M2.Module.Write_Speed,
		&pc_components.SSD_M2.Module.Rewrite_Resource,
		&pc_components.SSD_M2.Quantity,
		&pc_components.SSD_SATA.Module.Manufacturer,
		&pc_components.SSD_SATA.Module.Model,
		&pc_components.SSD_SATA.Module.Storage_Capacity,
		&pc_components.SSD_SATA.Module.Reading_Speed,
		&pc_components.SSD_SATA.Module.Write_Speed,
		&pc_components.SSD_SATA.Module.Rewrite_Resource,
		&pc_components.SSD_SATA.Quantity,
		&pc_components.HDD.Module.Manufacturer,
		&pc_components.HDD.Module.Form_Factor,
		&pc_components.HDD.Module.Model,
		&pc_components.HDD.Module.Storage_Capacity,
		&pc_components.HDD.Module.Rotation_Speed,
		&pc_components.HDD.Quantity,
		&pc_components.Power_Unit.Manufacturer,
		&pc_components.Power_Unit.Model,
		&pc_components.Power_Unit.Power,
		&pc_components.Power_Unit.Has_Ocp,
		&pc_components.Power_Unit.Has_Ovp,
		&pc_components.Power_Unit.Has_Uvp,
		&pc_components.Power_Unit.Has_Scp,
		&pc_components.Power_Unit.Has_Opp,
		&pc_components.Power_Unit.Fan_Size,
		&pc_components.Power_Unit.Form_Factor,
		&pc_components.Frame.Manufacturer,
		&pc_components.Frame.Model,
		&pc_components.Frame.Supports_Mini_Itx,
		&pc_components.Frame.Supports_Micro_Atx,
		&pc_components.Frame.Supports_Atx,
		&pc_components.Frame.Supports_E_Atx,
		&pc_components.Frame.Liquid_Cooling_System,
		&pc_components.Frame.Fans_Included,
		&pc_components.Frame.Maximum_Length_GPU,
		&pc_components.Frame.Maximum_Cooler_Height,
		&pc_components.Frame.Type_Size,
		&pc_components.Cooling_System.Manufacturer,
		&pc_components.Cooling_System.Model,
		&pc_components.Cooling_System.Type,
		&pc_components.Cooling_System.Sockets,
		&pc_components.Cooling_System.Dissipated_Power,
		&pc_components.In_Cart,
	)
	if err != nil {
		fmt.Println(err)
		return models.PC_model{}, err
	}
	return pc_components, nil
}

func (r *repository_struct) GettingPCForComparison(pc_id []int, user_id int) (*[]models.PC_model, error) {

	g, ctx := errgroup.WithContext(context.Background()) // Группа ошибок, для отлавливания их в горутинах
	pc_comparison := make([]models.PC_model, len(pc_id)) // потоко безопасен для горутин
	sql := `
		SELECT cp.id, cp.name, cp.photo, cp.price,
			proc.manufacturer, proc.product_line, proc.model,
			proc.socket, proc.architecture, proc.number_cores,
			proc.number_threads, proc.frequency, proc.tdp,
			proc.max_tdp, proc.ram_standart, proc.integrated_graphics_core,
			m.name, m.manufacturer, m.chipset,
			m.ram_type, m.max_ram, m.socket,
			m.pcie_x16_port, m.pcie_x1_port, m.wifi,
			m.audio_codec, m.form_factor, m.ram_slots,
			m.m2_slots, m.sata_slots, v.manufacturer,
			v.gpu_manufacturer, v.series,
			v.pcie, v.video_memory_capacity, v.hdmi,
			v.displayport, v.memory_type, v.gpu_frequency,
			v.bandwidth, v.video_memory_frequency, v.consumption,
			v.memory_bus, r.name, r.brand,
			r.volume_one_module, r.memory_type, r.frequency,
			r.number_modules, rm.quantity, ssdm2.manufacturer,
			ssdm2.model, ssdm2.pcie, ssdm2.storage_capacity,
			ssdm2.reading_speed, ssdm2.write_speed, ssdm2.rewrite_resource,
			ssdm2c.quantity, ssdsata.manufacturer, ssdsata.model,
			ssdsata.storage_capacity, ssdsata.reading_speed, ssdsata.write_speed,
			ssdsata.rewrite_resource, ssdsatac.quantity, h.manufacturer,
			h.form_factor, h.model, h.storage_capacity, h.rotation_speed,
			hc.quantity, pw.manufacturer, pw.model,
			pw.power, pw.has_ocp, pw.has_ovp,
			pw.has_uvp, pw.has_scp, pw.has_opp,
			pw.fan_size, pw.form_factor, f.manufacturer,
			f.model, f.supports_mini_itx, f.supports_micro_atx,
			f.supports_atx, f.supports_e_atx, f.liquid_cooling_system,
			f.fans_included, f.maximum_length_gpu, f.maximum_cooler_height,
			f.type_size, cs.manufacturer, cs.model,
			cs.type, cs.sockets, cs.dissipated_power,
			CASE
				WHEN cpc.id_config IS NOT NULL THEN TRUE ELSE FALSE
			END AS in_cart
		FROM config_pc cp
		JOIN processor proc ON cp.id_processor = proc.id
		JOIN motherboard m ON cp.id_motherboard = m.id
		LEFT JOIN video_card v ON cp.id_video_card = v.id
		JOIN ram_config rm ON cp.id_pc_ram_config = rm.id
		JOIN ram r ON rm.id_ram = r.id
		LEFT JOIN ssd_m2_config ssdm2c ON cp.id_ssd_m2_config = ssdm2c.id
		LEFT JOIN ssd_m2 ssdm2 ON ssdm2c.id_ssd_m2 = ssdm2.id
		LEFT JOIN ssd_sata_config ssdsatac ON cp.id_ssd_config = ssdsatac.id
		LEFT JOIN ssd_sata ssdsata ON ssdsatac.id_ssd_sata = ssdsata.id
		LEFT JOIN hdd_config hc ON cp.id_hdd_config = hc.id
		LEFT JOIN hdd h ON hc.id_hdd = h.id
		JOIN power_unit pw ON cp.id_power_unit = pw.id
		JOIN frame f ON cp.id_frame = f.id
		JOIN cooling_system cs ON cp.id_cooling_system = cs.id
		LEFT JOIN cart c ON c.id_user = $2
		LEFT JOIN cart_pc cpc ON cpc.id_cart = c.id AND cpc.id_config = cp.id
		WHERE cp.id = $1
	`

	for index, value := range pc_id { // Перебор входного массива
		g.Go(func() error {
			pc, err := r.GetPCByID(ctx, value, sql, user_id) // Вызов йункции для получения характеристик
			if err != nil {
				return err
			}
			pc_comparison[index] = pc // добавление в массив по индексу
			return nil
		})
	}

	// Отлов ошибки из горутин
	if err := g.Wait(); err != nil {
		return nil, err
	}

	return &pc_comparison, nil
}

func (r *repository_struct) AddPhoneUser(id int, number_phone string) error {
	_, err := r.db.Exec(context.Background(), `UPDATE users SET telephone = $1 WHERE id = $2 `, number_phone, id)
	if err != nil {
		return err
	}
	return nil
}

func (r *repository_struct) AddCustomConfigToCart(id int, config models.User_Config_Model) (err error) {

	ctx := context.Background()

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}

	defer func() {
		if err != nil {
			tx.Rollback(ctx)
		}
	}()

	var cartID int
	var customConfigID int
	var ramId int
	var hddID pgtype.Int4
	var ssdSataId pgtype.Int4
	var ssdM2Id pgtype.Int4

	err = tx.QueryRow(
		ctx,
		`SELECT id 
		FROM cart
		WHERE id_user = $1`,
		id,
	).Scan(&cartID)
	if err != nil {
		logger.Log.Error("Error in cart transaction:", zap.Error(err))
		return err
	}

	err = tx.QueryRow(
		ctx,
		`INSERT INTO ram_config (id_ram, quantity)
		VALUES ($1, $2)
		ON CONFLICT (id_ram, quantity) 
		DO UPDATE SET quantity = ram_config.quantity
		RETURNING id`,
		config.Ram.ID,
		config.Ram.Count,
	).Scan(&ramId)

	if err != nil {
		logger.Log.Error("Error in ram_config transaction:", zap.Error(err))
		return err
	}

	if config.HDD.ID != 0 && config.HDD.Count != 0 {
		err = tx.QueryRow(
			ctx,
			`INSERT INTO hdd_config (id_hdd, quantity)
			VALUES ($1, $2)
			ON CONFLICT (id_hdd, quantity)
			DO UPDATE SET quantity = hdd_config.quantity
			RETURNING id`,
			config.HDD.ID,
			config.HDD.Count,
		).Scan(&hddID)

		if err != nil {
			logger.Log.Error("Error in hdd_config transaction:", zap.Error(err))
			return err
		}
	}

	if config.SSD_M2.ID != 0 && config.SSD_M2.Count != 0 {
		err = tx.QueryRow(
			ctx,
			`INSERT INTO ssd_m2_config (id_ssd_m2, quantity) 
			VALUES ($1, $2)
			ON CONFLICT (id_ssd_m2, quantity)
			DO UPDATE SET quantity = ssd_m2_config.quantity
			RETURNING id`,
			config.SSD_M2.ID,
			config.SSD_M2.Count,
		).Scan(&ssdM2Id)

		if err != nil {
			logger.Log.Error("Error in ssd_m2_config transaction:", zap.Error(err))
			return err
		}
	}

	if config.SSD_Sata.ID != 0 && config.SSD_Sata.Count != 0 {
		err = tx.QueryRow(
			ctx,
			`INSERT INTO ssd_sata_config (id_ssd_sata, quantity)
			VALUES ($1, $2)
			ON CONFLICT (id_ssd_sata, quantity)
			DO UPDATE SET quantity = ssd_sata_config.quantity
			RETURNING id`,
			config.SSD_Sata.ID,
			config.SSD_Sata.Count,
		).Scan(&ssdSataId)

		if err != nil {
			logger.Log.Error("Error in ssd_sata_config transaction:", zap.Error(err))
			return err
		}
	}
	// TODO исправить суммирование компонентов, он выдает неправильную сумму
	err = tx.QueryRow(
		ctx,
		`WITH 
			frame_data AS (SELECT photo, price AS frame_price FROM frame WHERE id = $9),
			total_price AS (
				SELECT
					(SELECT price FROM processor WHERE id = $1) + 
					(SELECT price FROM motherboard WHERE id = $2) +
					(SELECT price FROM video_card WHERE id = $4) +
					(SELECT price FROM power_unit WHERE id = $5) +
					(SELECT frame_price FROM frame_data) +
					(SELECT price FROM cooling_system WHERE id = $10) +
					COALESCE((SELECT r.price * rc.quantity FROM ram r JOIN ram_config rc ON r.id = rc.id_ram WHERE rc.id = $3), 0) +
					COALESCE((SELECT h.price * hc.quantity FROM hdd h JOIN hdd_config hc ON h.id = hc.id_hdd WHERE hc.id = $8), 0) +
					COALESCE((SELECT s.price * sc.quantity FROM ssd_m2 s JOIN ssd_m2_config sc ON s.id = sc.id_ssd_m2 WHERE sc.id = $6), 0) +
					COALESCE((SELECT s.price * sc.quantity FROM ssd_sata s JOIN ssd_sata_config sc ON s.id = sc.id_ssd_sata WHERE sc.id = $7), 0) 
					AS total
			) 
		INSERT INTO config_pc (
			id_processor,
			id_motherboard,
			id_pc_ram_config,
			id_video_card,
			id_power_unit,
			id_ssd_m2_config,
			id_ssd_config,
			id_hdd_config,
			id_frame,
			id_cooling_system,
			name,
			photo,
			price,
			category,
			is_catalog,
			article,
			short_description,
			product_description
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, (SELECT photo FROM frame_data), (SELECT total+15000 FROM total_price), $12, $13, $14, $15, $16)
		RETURNING id
		`,
		config.Cpu_ID,
		config.Motherboard_ID,
		ramId,
		config.GPU_ID,
		config.Power_Unit_ID,
		ssdM2Id,
		ssdSataId,
		hddID,
		config.Frame_ID,
		config.Cooling_System_ID,
		"Кастомный ПК",
		"custom",
		false,
		nil,
		nil,
		nil,
	).Scan(&customConfigID)

	if err != nil {
		logger.Log.Error("Error in config_pc transaction:", zap.Error(err))
		return err
	}

	_, err = tx.Exec(
		ctx,
		`INSERT INTO cart_pc (id_cart, id_config, quantity)
		VALUES ($1, $2, 1)
		ON CONFLICT (id_cart, id_config)
		DO UPDATE SET 
			quantity = cart_pc.quantity + 1;`,
		cartID,
		customConfigID,
	)

	if err != nil {
		logger.Log.Error("Error in add config to cart_pc transaction:", zap.Error(err))
		return err
	}

	err = tx.Commit(ctx)
	if err != nil {
		return err
	}

	return nil
}

func (r *repository_struct) GetAllOrders(id int) (*[]models.Order, error) {
	var items []models.Order
	rows, err := r.db.Query(
		context.Background(),
		`SELECT o.id, o.order_code, s.name, o.date, o.sum FROM "order" o
		JOIN status_order s ON o.id_status = s.id
		WHERE id_user = $1
		ORDER BY date`,
		id,
	)
	if err != nil {
		return nil, err
	}

	for rows.Next() {
		var item models.Order
		err := rows.Scan(
			&item.ID,
			&item.Order_code,
			&item.Status,
			&item.Date,
			&item.Sum,
		)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return &items, nil
}

func (r *repository_struct) AddOrder(id, pick_up_point_id int, order_code string) (err error) {
	var order_id int
	ctx := context.Background()
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}

	defer func() {
		if err != nil {
			_ = tx.Rollback(ctx)
		}
	}()

	err = tx.QueryRow(
		ctx,
		`WITH cartid AS (
			SELECT id FROM cart WHERE id_user = $1
		),
		cart_total AS (
    		SELECT COALESCE(SUM(con.price * c.quantity), 0) as total_sum
    		FROM cart_pc c
    		JOIN config_pc con ON c.id_config = con.id
    		WHERE c.id_cart = (SELECT id FROM cartid)
		)
		INSERT INTO "order" (order_code, id_status, id_user, date, sum, pick_up_point_id)
		VALUES ($2, $3, $1, NOW(), (SELECT total_sum FROM cart_total), $4) RETURNING id;`,
		id,
		order_code,
		1,
		pick_up_point_id,
	).Scan(&order_id)

	_, err = tx.Exec(
		ctx,
		`INSERT INTO order_items (id_order, id_config, quantity, price)
		SELECT $1,
		c.id_config,
		c.quantity,
		con.price
		FROM cart_pc c
		JOIN config_pc con ON c.id_config = con.id
		WHERE c.id_cart = (SELECT id FROM cart WHERE id_user = $2)
		`,
		order_id,
		id,
	)

	err = tx.Commit(ctx)
	if err != nil {
		return err
	}

	return nil

}

func (r *repository_struct) GetInfoOrder(id int, order_code string) (*[]models.Order_Items, error) {
	var result []models.Order_Items
	rows, err := r.db.Query(
		context.Background(),
		`SELECT c.photo, c.name, c.price, oi.quantity, o.sum, o.date FROM order_items oi
		JOIN "order" o ON oi.id_order = o.id
		JOIN config_pc c ON oi.id_config = c.id
		JOIN users u ON o.id_user = u.id
		WHERE u.id = $1 AND o.order_code = $2`,
		id, order_code,
	)
	if err != nil {
		return nil, err
	}

	for rows.Next() {
		var item models.Order_Items
		err := rows.Scan(
			&item.Photo,
			&item.Name,
			&item.Price,
			&item.Quantity,
			&item.Sum,
			&item.Date,
		)
		if err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	return &result, nil
}

func (r *repository_struct) GetAccountDashboard(id int) (*models.AccountDashboard, error) {
	var result models.AccountDashboard
	var items []models.Order

	g, ctx := errgroup.WithContext(context.Background())

	g.Go(func() error {
		rows, err := r.db.Query(
			context.Background(),
			`SELECT o.id, o.order_code, s.name, o.date, o.sum FROM "order" o
		JOIN status_order s ON o.id_status = s.id
		WHERE id_user = $1
		ORDER BY date LIMIT 5;`,
			id,
		)
		if err != nil {
			return err
		}

		for rows.Next() {
			var item models.Order
			err := rows.Scan(
				&item.ID,
				&item.Order_code,
				&item.Status,
				&item.Date,
				&item.Sum,
			)
			if err != nil {
				return err
			}
			items = append(items, item)
		}
		result.Last_Orders = items
		return nil
	})

	g.Go(func() error {
		err := r.db.QueryRow(
			ctx,
			`SELECT 
    	COUNT(o.id) as total_orders,
    	u.created_at as registration_date,
    	COALESCE(SUM(o.sum), 0) as total_spent
		FROM users u
		LEFT JOIN "order" o ON o.id_user = u.id
		WHERE u.id = $1
		GROUP BY u.id, u.created_at;
		`,
			id,
		).Scan(
			&result.Total_Orders,
			&result.RegisterdAt,
			&result.Total_Spent,
		)
		if err != nil {
			return err
		}
		return nil
	})

	if err := g.Wait(); err != nil {
		return nil, err
	}

	return &result, nil
}

func (r *repository_struct) UpdateCartItemQuantity(user_id, config_id, num int) (err error) {
	var cart_id int
	ctx := context.Background()
	tx, err := r.db.Begin(ctx)
	if err != nil {
		fmt.Println(err)
		return err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback(ctx)
		}
	}()

	err = tx.QueryRow(ctx, `SELECT id FROM cart WHERE id_user = $1`, user_id).Scan(&cart_id)
	if err != nil {
		fmt.Println(err)
		return err
	}
	_, err = tx.Exec(ctx, `SAVEPOINT update_quantity`)
	if err != nil {
		return err
	}
	_, err = tx.Exec(
		ctx,
		`UPDATE cart_pc
		SET quantity = quantity + $1
		WHERE id_cart = $2 AND id_config = $3`,
		num,
		cart_id,
		config_id,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23514" { // Проверка на ошибку, в бд стоит constraint на количество товара, не моежт быть 0 и меньше
			_, err = tx.Exec(ctx, `ROLLBACK TO SAVEPOINT update_quantity`)
			if err != nil {
				return err
			}
			_, err = tx.Exec(
				ctx,
				`DELETE FROM cart_pc
				WHERE id_cart = $1 AND id_config = $2`,
				cart_id,
				config_id,
			)
			if err != nil {
				fmt.Println(err)
				return err
			}
		} else {
			return err
		}
	}

	err = tx.Commit(ctx)
	if err != nil {
		return err
	}

	return nil
}

func (r *repository_struct) AddCart(user_id int, config_id int) (err error) {
	var cart_id int
	ctx := context.Background()

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback(ctx)
		}
	}()
	err = tx.QueryRow(ctx, `SELECT id FROM cart WHERE id_user = $1`, user_id).Scan(&cart_id)
	if err != nil {
		return err
	}

	_, err = tx.Exec(
		ctx,
		`INSERT INTO cart_pc (id_cart, id_config, quantity)
		VALUES ($1, $2, 1)
		ON CONFLICT (id_cart, id_config) 
		DO UPDATE SET quantity = cart_pc.quantity + 1`,
		cart_id,
		config_id,
	)

	tx.Commit(ctx)
	return nil
}

func (r *repository_struct) CartItems(user_id int) ([]models.Cart_Item, error) {
	var items []models.Cart_Item

	rows, err := r.db.Query(
		context.Background(),
		`SELECT cp.id AS cart_item_id, cp.id_config, cpc.name, cpc.photo, cpc.article, cp.quantity, cpc.price
		FROM cart_pc cp
		JOIN cart c ON cp.id_cart = c.id
		JOIN config_pc cpc ON cp.id_config = cpc.id
		WHERE c.id_user = $1
		ORDER BY cp.id;`, user_id,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var item models.Cart_Item
		err := rows.Scan(
			&item.Cart_item_id,
			&item.ID_config,
			&item.Name,
			&item.Photo,
			&item.Article,
			&item.Quantity,
			&item.Price,
		)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}

	return items, nil
}

func (r *repository_struct) RemoveFromCart(user_id, config_id int) error {
	_, err := r.db.Exec(
		context.Background(),
		`DELETE FROM cart_pc
		USING cart
		WHERE cart_pc.id_cart = cart.id
		AND cart.id_user = $1
		AND cart_pc.id_config = $2`,
		user_id,
		config_id,
	)
	if err != nil {
		return err
	}
	return nil
}

func (r *repository_struct) DeleteCartItem(id_cart string, id_item string) error {
	return nil
}

//func (r *repository_struct) GetCartItems(id int)

func (r *repository_struct) GetUserProfile(id int) (*models.Profile_Model, error) {
	user := new(models.Profile_Model)
	err := r.db.QueryRow(
		context.Background(),
		`SELECT 
    	u.name,
    	u.surname,
		u.email,
		u.telephone,
		u.avatar,
		u.role,
		u.created_at,
		u.pick_up_point_id,
    	(SELECT COUNT(*)
     		   FROM cart c
     		   JOIN cart_pc cp ON c.id = cp.id_cart
     		   WHERE c.id_user = u.id) AS cart_items_count
		FROM users u
		WHERE u.id = $1;`,
		id).Scan(
		&user.Name,
		&user.Surname,
		&user.Email,
		&user.Phone,
		&user.Avatar,
		&user.Role,
		&user.Created_at,
		&user.Pick_up_point_ID,
		&user.CartItemsCount,
	)
	if err != nil {
		logger.Log.Error(err.Error())
		return nil, err
	}
	return user, nil
}
