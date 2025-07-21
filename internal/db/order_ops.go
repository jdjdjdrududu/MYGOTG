package db

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	// "strconv" // Не используется напрямую в этом файле, но может быть в utils
	"strings"
	"time"

	"github.com/lib/pq" // Для работы с массивами PostgreSQL / For working with PostgreSQL arrays

	"Original/internal/constants"
	"Original/internal/models"
	"Original/internal/utils" // Для utils.ValidateDate
)

// CreateInitialOrder создает черновик заказа в базе данных.
// CreateInitialOrder creates a draft order in the database.
func CreateInitialOrder(orderData models.Order) (int64, error) {
	if orderData.Category == "" {
		return 0, errors.New("категория не может быть пустой")
	}
	clientChatID := orderData.UserChatID
	if clientChatID == 0 {
		log.Printf("CreateInitialOrder: Внимание! UserChatID не установлен в orderData. Заказ не может быть создан без клиента.")
		return 0, errors.New("UserChatID (идентификатор клиента) не установлен для заказа")
	}

	tx, err := DB.Begin()
	if err != nil {
		log.Printf("CreateInitialOrder: Ошибка начала транзакции: %v", err)
		return 0, err
	}
	defer tx.Rollback()

	var userID int
	err = tx.QueryRow("SELECT id FROM users WHERE chat_id=$1", clientChatID).Scan(&userID)
	if err != nil {
		if err == sql.ErrNoRows {
			log.Printf("CreateInitialOrder: Клиент с chat_id %d не найден в БД. Невозможно создать заказ.", clientChatID)
			return 0, fmt.Errorf("клиент с chat_id %d не найден", clientChatID)
		}
		log.Printf("CreateInitialOrder: Ошибка получения userID для клиента %d: %v", clientChatID, err)
		return 0, err
	}

	var parsedDate sql.NullTime
	if strings.TrimSpace(orderData.Date) != "" {
		pDate, errDate := time.ParseInLocation("02 January 2006", orderData.Date, time.Local)
		if errDate != nil {
			pDate, errDate = utils.ValidateDate(orderData.Date)
			if errDate != nil {
				log.Printf("CreateInitialOrder: Ошибка валидации/парсинга даты '%s': %v", orderData.Date, errDate)
				return 0, fmt.Errorf("некорректный формат даты: '%s'", orderData.Date)
			}
		}
		parsedDate = sql.NullTime{Time: pDate, Valid: true}
	} else {
		parsedDate = sql.NullTime{Valid: false}
	}

	var timeVal sql.NullString
	if orderData.Time == "" || strings.ToUpper(orderData.Time) == "СРОЧНО" || strings.ToLower(orderData.Time) == "в ближайшее время" || orderData.Time == "❗ СРОЧНО (В БЛИЖАЙШЕЕ ВРЕМЯ) ❗" {
		if strings.ToUpper(orderData.Time) == "СРОЧНО" {
			timeVal = sql.NullString{String: "СРОЧНО", Valid: true}
		} else {
			timeVal = sql.NullString{Valid: false}
		}
	} else {
		timeVal = sql.NullString{String: orderData.Time, Valid: true}
	}

	var id int64
	// ИЗМЕНЕНИЕ: Добавлено поле is_driver_settled (по умолчанию FALSE)
	query := `
        INSERT INTO orders (
            user_id, user_chat_id, category, subcategory, name,
            photos, videos, date, time, phone, address,
            description, status, cost, payment,
            latitude, longitude, created_at, updated_at, is_driver_settled
        )
        VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, NOW(), NOW(), FALSE)
        RETURNING id`

	err = tx.QueryRow(query,
		userID, clientChatID, orderData.Category, orderData.Subcategory, orderData.Name,
		pq.Array(orderData.Photos), pq.Array(orderData.Videos), parsedDate, timeVal,
		orderData.Phone, orderData.Address, orderData.Description, constants.STATUS_DRAFT,
		orderData.Cost, orderData.Payment, orderData.Latitude, orderData.Longitude,
	).Scan(&id)

	if err != nil {
		log.Printf("CreateInitialOrder: Ошибка выполнения INSERT для черновика заказа (клиент %d): %v", clientChatID, err)
		return 0, err
	}

	err = tx.Commit()
	if err != nil {
		log.Printf("CreateInitialOrder: Ошибка фиксации транзакции: %v", err)
		return 0, err
	}

	log.Printf("Черновик заказа #%d успешно создан для клиента chat_id %d.", id, clientChatID)
	return id, nil
}

// GetOrderByID извлекает заказ по его ID.
// Поле Date возвращается в формате "YYYY-MM-DD".
// ИЗМЕНЕНИЕ: Добавлено сканирование поля is_driver_settled.
func GetOrderByID(orderID int) (models.Order, error) {
	var order models.Order
	var dbDate sql.NullTime
	var dbTime sql.NullString
	var dbCreatedAt, dbUpdatedAt sql.NullTime

	err := DB.QueryRow(`
        SELECT o.id, o.user_id, o.user_chat_id, o.category, o.subcategory, o.name,
               o.photos, o.videos, o.date, o.time, o.phone, o.address,
               o.description, o.status, o.reason, o.cost, o.payment,
               o.latitude, o.longitude, o.created_at, o.updated_at, o.is_driver_settled
        FROM orders o
        WHERE o.id = $1`, orderID).Scan(
		&order.ID, &order.UserID, &order.UserChatID, &order.Category, &order.Subcategory, &order.Name,
		pq.Array(&order.Photos), pq.Array(&order.Videos), &dbDate, &dbTime, &order.Phone, &order.Address,
		&order.Description, &order.Status, &order.Reason, &order.Cost, &order.Payment,
		&order.Latitude, &order.Longitude, &dbCreatedAt, &dbUpdatedAt, &order.IsDriverSettled, // Сканируем новое поле
	)

	if err != nil {
		log.Printf("GetOrderByID: ошибка получения заказа #%d: %v", orderID, err)
		return order, err
	}

	if dbDate.Valid {
		order.Date = dbDate.Time.Format("2006-01-02")
	} else {
		order.Date = ""
	}
	if dbTime.Valid {
		order.Time = dbTime.String
	} else {
		order.Time = ""
	}
	if dbCreatedAt.Valid {
		order.CreatedAt = dbCreatedAt.Time
	}
	if dbUpdatedAt.Valid {
		order.UpdatedAt = dbUpdatedAt.Time
	}

	return order, nil
}

// GetOrderStatus получает статус заказа по ID.
func GetOrderStatus(orderID int64) (string, error) {
	var status string
	err := DB.QueryRow("SELECT status FROM orders WHERE id=$1", orderID).Scan(&status)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", fmt.Errorf("заказ с ID %d не найден", orderID)
		}
		log.Printf("GetOrderStatus: ошибка получения статуса заказа #%d: %v", orderID, err)
		return "", err
	}
	return status, nil
}

// GetOrderStatusInTx получает статус заказа по ID в рамках транзакции.
func GetOrderStatusInTx(tx *sql.Tx, orderID int64) (string, error) {
	var status string
	err := tx.QueryRow("SELECT status FROM orders WHERE id=$1 FOR UPDATE", orderID).Scan(&status) // Добавлено FOR UPDATE для предотвращения гонок состояний
	if err != nil {
		if err == sql.ErrNoRows {
			return "", fmt.Errorf("заказ с ID %d не найден в транзакции", orderID)
		}
		// Не логируем здесь подробно, так как это внутренняя функция транзакции
		return "", err
	}
	return status, nil
}

// UpdateOrderStatus обновляет статус заказа.
func UpdateOrderStatus(orderID int64, status string) error {
	_, err := DB.Exec("UPDATE orders SET status=$1, updated_at=NOW() WHERE id=$2", status, orderID)
	if err != nil {
		log.Printf("UpdateOrderStatus: ошибка обновления статуса заказа #%d на %s: %v", orderID, status, err)
		return err
	}
	log.Printf("Статус заказа #%d обновлен на %s", orderID, status)
	return nil
}

// UpdateOrderStatusInTx обновляет статус заказа в рамках транзакции.
func UpdateOrderStatusInTx(tx *sql.Tx, orderID int64, status string) error {
	result, err := tx.Exec("UPDATE orders SET status=$1, updated_at=NOW() WHERE id=$2", status, orderID)
	if err != nil {
		log.Printf("UpdateOrderStatusInTx: ошибка обновления статуса заказа #%d на %s: %v", orderID, status, err)
		return err
	}
	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("заказ с ID %d не найден для обновления статуса в транзакции", orderID)
	}
	log.Printf("Статус заказа #%d обновлен на %s в транзакции.", orderID, status)
	return nil
}

// UpdateOrderCostAndStatus обновляет стоимость и статус заказа.
func UpdateOrderCostAndStatus(orderID int64, cost float64, status string) error {
	_, err := DB.Exec("UPDATE orders SET cost=$1, status=$2, updated_at=NOW() WHERE id=$3", cost, status, orderID)
	if err != nil {
		log.Printf("UpdateOrderCostAndStatus: ошибка обновления стоимости/статуса заказа #%d: %v", orderID, err)
		return err
	}
	log.Printf("Стоимость (%.0f) и статус (%s) заказа #%d обновлены.", cost, status, orderID)
	return nil
}

// UpdateOrderReasonAndStatus обновляет причину отмены и статус заказа.
func UpdateOrderReasonAndStatus(orderID int64, reason string, status string) error {
	_, err := DB.Exec("UPDATE orders SET reason=$1, status=$2, updated_at=NOW() WHERE id=$3", reason, status, orderID)
	if err != nil {
		log.Printf("UpdateOrderReasonAndStatus: ошибка обновления причины/статуса заказа #%d: %v", orderID, err)
		return err
	}
	log.Printf("Причина (%s) и статус (%s) заказа #%d обновлены.", reason, status, orderID)
	return nil
}

// GetOrdersByChatIDAndStatus извлекает заказы пользователя по его user_chat_id и статусу.
// Поле Date возвращается в формате "YYYY-MM-DD".
// ИЗМЕНЕНИЕ: Добавлено сканирование поля is_driver_settled.
func GetOrdersByChatIDAndStatus(userChatID int64, status string, page int) ([]models.Order, error) {
	offset := page * constants.OrdersPerPage
	var rows *sql.Rows
	var err error

	queryParams := []interface{}{userChatID}
	queryBase := `
        SELECT o.id, o.user_id, o.user_chat_id, o.category, o.subcategory, o.name,
               o.photos, o.videos, o.date, o.time, o.phone, o.address,
               o.description, o.status, o.reason, o.cost, o.payment,
               o.latitude, o.longitude, o.created_at, o.updated_at, o.is_driver_settled
        FROM orders o
        WHERE o.user_chat_id = $1`

	if status != "" {
		if status == constants.STATUS_NEW {
			queryBase += " AND o.status IN ($2, $3)"
			queryParams = append(queryParams, constants.STATUS_NEW, constants.STATUS_AWAITING_COST)
		} else {
			queryBase += " AND o.status = $2"
			queryParams = append(queryParams, status)
		}
	}
	queryBase += fmt.Sprintf(" ORDER BY o.created_at DESC LIMIT %d OFFSET $%d", constants.OrdersPerPage, len(queryParams)+1)
	queryParams = append(queryParams, offset)

	rows, err = DB.Query(queryBase, queryParams...)

	if err != nil {
		log.Printf("GetOrdersByChatIDAndStatus: ошибка запроса заказов для chatID %d, статус '%s': %v", userChatID, status, err)
		return nil, err
	}
	defer rows.Close()

	var orders []models.Order
	for rows.Next() {
		var o models.Order
		var dbDate sql.NullTime
		var dbTime sql.NullString
		var dbCreatedAt, dbUpdatedAt sql.NullTime
		errScan := rows.Scan(
			&o.ID, &o.UserID, &o.UserChatID, &o.Category, &o.Subcategory, &o.Name,
			pq.Array(&o.Photos), pq.Array(&o.Videos), &dbDate, &dbTime, &o.Phone, &o.Address,
			&o.Description, &o.Status, &o.Reason, &o.Cost, &o.Payment,
			&o.Latitude, &o.Longitude, &dbCreatedAt, &dbUpdatedAt, &o.IsDriverSettled, // Сканируем новое поле
		)
		if errScan != nil {
			log.Printf("GetOrdersByChatIDAndStatus: ошибка сканирования заказа для chatID %d: %v", userChatID, errScan)
			continue
		}
		if dbDate.Valid {
			o.Date = dbDate.Time.Format("2006-01-02")
		} else {
			o.Date = ""
		}
		if dbTime.Valid {
			o.Time = dbTime.String
		} else {
			o.Time = ""
		}
		if dbCreatedAt.Valid {
			o.CreatedAt = dbCreatedAt.Time
		}
		if dbUpdatedAt.Valid {
			o.UpdatedAt = dbUpdatedAt.Time
		}
		orders = append(orders, o)
	}
	if err = rows.Err(); err != nil {
		log.Printf("GetOrdersByChatIDAndStatus: ошибка после итерации по строкам: %v", err)
		return nil, err
	}
	return orders, nil
}

// GetOrdersByMultipleStatuses извлекает заказы по нескольким статусам для операторов.
// ИЗМЕНЕНИЕ: Добавлено сканирование поля is_driver_settled.
func GetOrdersByMultipleStatuses(statuses []string, page int, orderByField string, orderByDirection string) ([]models.Order, error) {
	offset := page * constants.OrdersPerPage
	sqlOrderByField := orderByField
	fieldForCheck := strings.TrimPrefix(orderByField, "o.")
	if fieldForCheck == "" {
		fieldForCheck = "created_at"
		sqlOrderByField = "o.created_at"
	} else if !strings.HasPrefix(sqlOrderByField, "o.") && (fieldForCheck == "created_at" || fieldForCheck == "updated_at" || fieldForCheck == "date" || fieldForCheck == "id" || fieldForCheck == "cost") {
		sqlOrderByField = "o." + fieldForCheck
	}
	if orderByDirection == "" {
		orderByDirection = "DESC"
	}
	allowedOrderByFields := map[string]bool{"created_at": true, "updated_at": true, "date": true, "id": true, "cost": true}
	if !allowedOrderByFields[fieldForCheck] {
		return nil, fmt.Errorf("недопустимое поле для сортировки: %s", sqlOrderByField)
	}
	upperOrderByDirection := strings.ToUpper(orderByDirection)
	validDirection := false
	directionParts := strings.Fields(upperOrderByDirection)
	if len(directionParts) > 0 {
		mainDir := directionParts[0]
		if mainDir == "ASC" || mainDir == "DESC" {
			if len(directionParts) == 1 {
				validDirection = true
			} else if len(directionParts) == 3 && strings.ToUpper(directionParts[1]) == "NULLS" && (strings.ToUpper(directionParts[2]) == "FIRST" || strings.ToUpper(directionParts[2]) == "LAST") {
				validDirection = true
			}
		}
	}
	if !validDirection {
		return nil, fmt.Errorf("недопустимое направление для сортировки: %s", orderByDirection)
	}
	sqlQuery := fmt.Sprintf(`
        SELECT o.id, o.user_id, o.user_chat_id, o.category, o.subcategory, o.name,
               o.photos, o.videos, o.date, o.time, o.phone, o.address,
               o.description, o.status, o.cost, o.payment, o.latitude, o.longitude,
               o.reason, o.created_at, o.updated_at, o.is_driver_settled
        FROM orders o
        WHERE o.status = ANY($1)
        ORDER BY %s %s
        LIMIT $2 OFFSET $3`, sqlOrderByField, orderByDirection)
	rows, err := DB.Query(sqlQuery, pq.Array(statuses), constants.OrdersPerPage, offset)
	if err != nil {
		return nil, fmt.Errorf("ошибка выполнения SQL запроса: %w", err)
	}
	defer rows.Close()
	var orders []models.Order
	for rows.Next() {
		var o models.Order
		var dbDate sql.NullTime
		var dbTime sql.NullString
		var dbCreatedAt, dbUpdatedAt sql.NullTime
		errScan := rows.Scan(
			&o.ID, &o.UserID, &o.UserChatID, &o.Category, &o.Subcategory, &o.Name,
			pq.Array(&o.Photos), pq.Array(&o.Videos), &dbDate, &dbTime, &o.Phone, &o.Address,
			&o.Description, &o.Status, &o.Cost, &o.Payment, &o.Latitude, &o.Longitude,
			&o.Reason, &dbCreatedAt, &dbUpdatedAt, &o.IsDriverSettled, // Сканируем новое поле
		)
		if errScan != nil {
			log.Printf("GetOrdersByMultipleStatuses: ошибка сканирования строки заказа: %v", errScan)
			continue
		}
		if dbDate.Valid {
			o.Date = dbDate.Time.Format("2006-01-02")
		} else {
			o.Date = ""
		}
		if dbTime.Valid {
			o.Time = dbTime.String
		} else {
			o.Time = ""
		}
		if dbCreatedAt.Valid {
			o.CreatedAt = dbCreatedAt.Time
		}
		if dbUpdatedAt.Valid {
			o.UpdatedAt = dbUpdatedAt.Time
		}
		orders = append(orders, o)
	}
	return orders, rows.Err()
}

// GetActiveOrdersForDisplay извлекает активные заказы.
// ИЗМЕНЕНИЕ: Добавлено сканирование поля is_driver_settled.
func GetActiveOrdersForDisplay(limit int, offset int) ([]models.Order, error) {
	activeStatuses := []string{
		constants.STATUS_NEW, constants.STATUS_AWAITING_COST,
		constants.STATUS_AWAITING_CONFIRMATION, constants.STATUS_INPROGRESS, constants.STATUS_DRAFT,
	}
	query := `
        SELECT id, user_id, user_chat_id, category, subcategory, name, photos, videos, date, time,
               phone, address, description, status, cost, payment, latitude, longitude, reason,
               created_at, updated_at, is_driver_settled
        FROM orders
        WHERE status = ANY($1)
        ORDER BY date ASC NULLS LAST, created_at DESC
        LIMIT $2 OFFSET $3`
	rows, err := DB.Query(query, pq.Array(activeStatuses), limit, offset)
	if err != nil {
		log.Printf("GetActiveOrdersForDisplay: ошибка получения активных заказов: %v", err)
		return nil, err
	}
	defer rows.Close()
	var orders []models.Order
	for rows.Next() {
		var o models.Order
		var dbDate sql.NullTime
		var dbTime sql.NullString
		var dbCreatedAt, dbUpdatedAt sql.NullTime
		errScan := rows.Scan(
			&o.ID, &o.UserID, &o.UserChatID, &o.Category, &o.Subcategory,
			&o.Name, pq.Array(&o.Photos), pq.Array(&o.Videos), &dbDate, &dbTime,
			&o.Phone, &o.Address, &o.Description, &o.Status, &o.Cost, &o.Payment,
			&o.Latitude, &o.Longitude, &o.Reason, &dbCreatedAt, &dbUpdatedAt, &o.IsDriverSettled, // Сканируем новое поле
		)
		if errScan != nil {
			log.Printf("GetActiveOrdersForDisplay: ошибка сканирования заказа: %v", errScan)
			continue
		}
		if dbDate.Valid {
			o.Date = dbDate.Time.Format("2006-01-02")
		} else {
			o.Date = ""
		}
		if dbTime.Valid {
			o.Time = dbTime.String
		} else {
			o.Time = ""
		}
		if dbCreatedAt.Valid {
			o.CreatedAt = dbCreatedAt.Time
		}
		if dbUpdatedAt.Valid {
			o.UpdatedAt = dbUpdatedAt.Time
		}
		orders = append(orders, o)
	}
	return orders, rows.Err()
}

// GetOrdersByExecutorIDAndStatuses извлекает заказы для исполнителя.
// ИЗМЕНЕНИЕ: Добавлено сканирование поля is_driver_settled.
func GetOrdersByExecutorIDAndStatuses(executorUserID int64, executorRole string, statuses []string, page int, perPage int) ([]models.Order, error) {
	offset := page * perPage
	if perPage <= 0 {
		perPage = constants.OrdersPerPage
	}
	var queryWhereStatus string
	queryParams := []interface{}{executorUserID, executorRole}
	paramIndex := 3
	if len(statuses) > 0 {
		queryWhereStatus = fmt.Sprintf("AND o.status = ANY($%d)", paramIndex)
		queryParams = append(queryParams, pq.Array(statuses))
		paramIndex++
	}
	query := fmt.Sprintf(`
        SELECT o.id, o.user_id, o.user_chat_id, o.category, o.subcategory, o.name,
               o.photos, o.videos, o.date, o.time, o.phone, o.address,
               o.description, o.status, o.reason, o.cost, o.payment,
               o.latitude, o.longitude, o.created_at, o.updated_at, o.is_driver_settled
        FROM orders o
        JOIN executors e ON o.id = e.order_id
        WHERE e.user_id = $1 AND e.role = $2 %s
        ORDER BY o.date ASC NULLS LAST, o.created_at DESC
        LIMIT $%d OFFSET $%d`, queryWhereStatus, paramIndex, paramIndex+1)
	queryParams = append(queryParams, perPage, offset)
	rows, err := DB.Query(query, queryParams...)
	if err != nil {
		log.Printf("GetOrdersByExecutorIDAndStatuses: ошибка получения заказов для исполнителя UserID %d (роль %s), статусы %v: %v", executorUserID, executorRole, statuses, err)
		return nil, err
	}
	defer rows.Close()
	var orders []models.Order
	for rows.Next() {
		var o models.Order
		var dbDate sql.NullTime
		var dbTime sql.NullString
		var dbCreatedAt, dbUpdatedAt sql.NullTime
		errScan := rows.Scan(
			&o.ID, &o.UserID, &o.UserChatID, &o.Category, &o.Subcategory, &o.Name,
			pq.Array(&o.Photos), pq.Array(&o.Videos), &dbDate, &dbTime, &o.Phone, &o.Address,
			&o.Description, &o.Status, &o.Reason, &o.Cost, &o.Payment,
			&o.Latitude, &o.Longitude, &dbCreatedAt, &dbUpdatedAt, &o.IsDriverSettled, // Сканируем новое поле
		)
		if errScan != nil {
			log.Printf("GetOrdersByExecutorIDAndStatuses: ошибка сканирования заказа для исполнителя UserID %d: %v", executorUserID, errScan)
			continue
		}
		if dbDate.Valid {
			o.Date = dbDate.Time.Format("2006-01-02")
		} else {
			o.Date = ""
		}
		if dbTime.Valid {
			o.Time = dbTime.String
		} else {
			o.Time = ""
		}
		if dbCreatedAt.Valid {
			o.CreatedAt = dbCreatedAt.Time
		}
		if dbUpdatedAt.Valid {
			o.UpdatedAt = dbUpdatedAt.Time
		}
		orders = append(orders, o)
	}
	return orders, rows.Err()
}

// UpdateOrderField обновляет указанное поле заказа.
func UpdateOrderField(orderID int64, field string, value interface{}) error {
	allowedFields := map[string]bool{
		"category": true, "subcategory": true, "name": true, "date": true, "time": true,
		"phone": true, "address": true, "description": true, "status": true, "cost": true,
		"payment": true, "latitude": true, "longitude": true, "reason": true,
		"is_driver_settled": true, // ИЗМЕНЕНИЕ: разрешаем обновлять это поле
	}
	if !allowedFields[field] {
		return fmt.Errorf("обновление поля '%s' не разрешено через UpdateOrderField", field)
	}
	if field == "date" {
		if dateStr, ok := value.(string); ok && dateStr != "" {
			parsedDate, errDate := time.ParseInLocation("2006-01-02", dateStr, time.Local)
			if errDate != nil {
				parsedDate, errDate = time.ParseInLocation("02 January 2006", dateStr, time.Local)
				if errDate != nil {
					log.Printf("UpdateOrderField: ошибка парсинга даты '%s' для заказа #%d: %v", dateStr, orderID, errDate)
					return fmt.Errorf("некорректный формат даты: %s", dateStr)
				}
			}
			value = parsedDate
		} else if _, ok := value.(time.Time); !ok && value != nil {
			return fmt.Errorf("некорректный тип значения для поля 'date': ожидалась строка, time.Time или nil, получено %T", value)
		} else if value == nil || (ok && dateStr == "") {
			value = sql.NullTime{Valid: false}
		}
	}
	if field == "time" {
		if timeStr, ok := value.(string); ok && (strings.ToUpper(timeStr) == "СРОЧНО" || timeStr == "") {
			if strings.ToUpper(timeStr) == "СРОЧНО" {
				value = sql.NullString{String: "СРОЧНО", Valid: true}
			} else {
				value = sql.NullString{Valid: false}
			}
		} else if !ok && value != nil {
			return fmt.Errorf("некорректный тип значения для поля 'time': ожидалась строка или nil, получено %T", value)
		} else if value == nil {
			value = sql.NullString{Valid: false}
		}
	}
	query := fmt.Sprintf("UPDATE orders SET %s=$1, updated_at=NOW() WHERE id=$2", field)
	_, err := DB.Exec(query, value, orderID)
	if err != nil {
		log.Printf("UpdateOrderField: ошибка обновления поля '%s' для заказа #%d: %v", field, orderID, err)
		return err
	}
	log.Printf("Поле '%s' для заказа #%d обновлено значением: %v.", field, orderID, value)
	return nil
}

// UpdateOrderPhotosAndVideos обновляет списки фото и видео для заказа.
func UpdateOrderPhotosAndVideos(orderID int64, photos []string, videos []string) error {
	_, err := DB.Exec("UPDATE orders SET photos=$1, videos=$2, updated_at=NOW() WHERE id=$3",
		pq.Array(photos), pq.Array(videos), orderID)
	if err != nil {
		log.Printf("UpdateOrderPhotosAndVideos: ошибка обновления фото/видео для заказа #%d: %v", orderID, err)
		return err
	}
	log.Printf("Фото/видео для заказа #%d обновлены.", orderID)
	return nil
}

// GetOrdersForReminder получает заказы, требующие напоминания операторам.
// ИЗМЕНЕНИЕ: Добавлено сканирование поля is_driver_settled.
func GetOrdersForReminder() ([]models.Order, error) {
	rows, err := DB.Query(`
        SELECT o.id, o.user_chat_id, o.created_at, o.name, o.date, o.time, o.is_driver_settled
        FROM orders o
        WHERE o.status = $1 AND o.created_at < NOW() - INTERVAL '30 minutes'`, constants.STATUS_NEW)
	if err != nil {
		log.Printf("GetOrdersForReminder: ошибка запроса заказов для напоминаний: %v", err)
		return nil, err
	}
	defer rows.Close()
	var orders []models.Order
	for rows.Next() {
		var o models.Order
		var dbDate sql.NullTime
		var dbTime sql.NullString
		errScan := rows.Scan(&o.ID, &o.UserChatID, &o.CreatedAt, &o.Name, &dbDate, &dbTime, &o.IsDriverSettled) // Сканируем новое поле
		if errScan != nil {
			log.Printf("GetOrdersForReminder: ошибка сканирования заказа: %v", errScan)
			continue
		}
		if dbDate.Valid {
			o.Date = dbDate.Time.Format("2006-01-02")
		} else {
			o.Date = ""
		}
		if dbTime.Valid {
			o.Time = dbTime.String
		} else {
			o.Time = ""
		}
		orders = append(orders, o)
	}
	return orders, rows.Err()
}

// GetFullOrderDetailsForNotification получает все необходимые данные заказа для уведомления операторов.
// ИЗМЕНЕНИЕ: Добавлено сканирование поля is_driver_settled.
func GetFullOrderDetailsForNotification(orderID int64) (models.Order, error) {
	var order models.Order
	var dbDate sql.NullTime
	var dbTime sql.NullString
	err := DB.QueryRow(`
        SELECT category, subcategory, name, date, time, phone, address, description, latitude, longitude,
               photos, videos, user_chat_id, cost, payment, status, is_driver_settled
        FROM orders WHERE id=$1`, orderID).Scan(
		&order.Category, &order.Subcategory, &order.Name, &dbDate, &dbTime, &order.Phone,
		&order.Address, &order.Description, &order.Latitude, &order.Longitude,
		pq.Array(&order.Photos), pq.Array(&order.Videos), &order.UserChatID, &order.Cost, &order.Payment, &order.Status, &order.IsDriverSettled, // Сканируем новое поле
	)
	if err != nil {
		log.Printf("GetFullOrderDetailsForNotification: ошибка получения данных заказа #%d: %v", orderID, err)
		return order, err
	}
	order.ID = orderID
	if dbDate.Valid {
		order.Date = dbDate.Time.Format("2006-01-02")
	} else {
		order.Date = ""
	}
	if dbTime.Valid {
		order.Time = dbTime.String
	} else {
		order.Time = ""
	}
	return order, nil
}

// GetOrdersCountForUser проверяет, есть ли у пользователя заказы.
func GetOrdersCountForUser(chatID int64) (int, error) {
	var count int
	err := DB.QueryRow("SELECT COUNT(*) FROM orders WHERE user_chat_id=$1", chatID).Scan(&count)
	if err != nil {
		log.Printf("GetOrdersCountForUser: ошибка проверки заказов для chatID %d: %v", chatID, err)
		return 0, err
	}
	return count, nil
}

// GetStats получает статистику за указанный период.
func GetStats(startDate, endDate time.Time) (models.Stats, error) {
	var stats models.Stats
	err := DB.QueryRow(`
        SELECT
            COUNT(o.id) as total_orders,
            COUNT(o.id) FILTER (WHERE o.status = $3) as new_orders,
            COUNT(o.id) FILTER (WHERE o.status = $4) as in_progress_orders,
            COUNT(o.id) FILTER (WHERE o.status IN ($5, $10, $11)) as completed_orders,
            COUNT(o.id) FILTER (WHERE o.status = $6) as canceled_orders,
            COUNT(o.id) FILTER (WHERE o.category = $7) as waste_orders,
            COUNT(o.id) FILTER (WHERE o.category = $8) as demolition_orders,
            COUNT(o.id) FILTER (WHERE o.category = $9) as material_orders,
            COALESCE(SUM(CASE WHEN o.status IN ($5, $10, $11) THEN o.cost ELSE 0 END), 0) as revenue
        FROM orders o
        WHERE o.created_at BETWEEN $1 AND $2
    `, startDate, endDate,
		constants.STATUS_NEW, constants.STATUS_INPROGRESS, constants.STATUS_COMPLETED, constants.STATUS_CANCELED,
		constants.CAT_WASTE, constants.CAT_DEMOLITION, constants.CAT_MATERIALS,
		constants.STATUS_CALCULATED, constants.STATUS_SETTLED,
	).Scan(
		&stats.TotalOrders, &stats.NewOrders, &stats.InProgressOrders, &stats.CompletedOrders,
		&stats.CanceledOrders, &stats.WasteOrders, &stats.DemolitionOrders, &stats.MaterialOrders,
		&stats.Revenue,
	)
	if err != nil && err != sql.ErrNoRows {
		log.Printf("GetStats: ошибка получения статистики по заказам: %v", err)
		return stats, fmt.Errorf("ошибка получения статистики по заказам: %w", err)
	}
	var totalExpenses sql.NullFloat64
	err = DB.QueryRow(`
		SELECT COALESCE(SUM(
			e.fuel + e.other +
			(SELECT COALESCE(SUM((ls.value->>'amount')::float), 0.0) FROM jsonb_each(e.loader_salaries) ls) +
			e.driver_share
		), 0.0)
		FROM expenses e
		JOIN orders o ON e.order_id = o.id
		WHERE o.status IN ($1, $2, $3) AND o.created_at BETWEEN $4 AND $5
	`, constants.STATUS_COMPLETED, constants.STATUS_CALCULATED, constants.STATUS_SETTLED, startDate, endDate).Scan(&totalExpenses)
	if err != nil && err != sql.ErrNoRows {
		log.Printf("GetStats: ошибка получения суммы расходов: %v", err)
		return stats, fmt.Errorf("ошибка получения суммы расходов: %w", err)
	}
	if totalExpenses.Valid {
		stats.Expenses = totalExpenses.Float64
	}

	// --- НАЧАЛО ИЗМЕНЕНИЯ ---
	// Исправлен JOIN: u.id = first_o.user_id и подзапрос o_inner.user_id = u.id
	err = DB.QueryRow(`
        SELECT COUNT(DISTINCT u.id)
        FROM users u
        JOIN orders first_o ON u.id = first_o.user_id
        WHERE first_o.created_at = (SELECT MIN(o_inner.created_at) FROM orders o_inner WHERE o_inner.user_id = u.id)
          AND first_o.created_at BETWEEN $1 AND $2
    `, startDate, endDate).Scan(&stats.NewClients)
	// --- КОНЕЦ ИЗМЕНЕНИЯ ---

	if err != nil && err != sql.ErrNoRows {
		log.Printf("GetStats: ошибка получения новых клиентов: %v", err)
		// Не возвращаем ошибку, чтобы остальная статистика отобразилась
	}
	stats.Profit = stats.Revenue - stats.Expenses
	return stats, nil
}

// GetOrdersForExcel генерирует данные для Excel-отчета по заказам за сегодня.
// ИЗМЕНЕНИЕ: Добавлено поле is_driver_settled в выборку.
func GetOrdersForExcel() (*sql.Rows, error) {
	query := `
        SELECT o.id, u.first_name, u.last_name, u.nickname, o.category, o.subcategory,
               o.date, o.time, o.phone, o.address, o.status, o.cost, o.is_driver_settled
        FROM orders o
        JOIN users u ON o.user_chat_id = u.chat_id
        WHERE date_trunc('day', o.created_at AT TIME ZONE 'UTC') = date_trunc('day', CURRENT_TIMESTAMP AT TIME ZONE 'UTC')
        ORDER BY o.created_at DESC`
	rows, err := DB.Query(query)
	if err != nil {
		log.Printf("GetOrdersForExcel: ошибка получения данных для Excel: %v", err)
		return nil, err
	}
	return rows, nil
}

// UpdateOrderAddress обновляет адрес заказа, включая координаты.
func UpdateOrderAddress(orderID int64, address string, latitude, longitude float64) error {
	_, err := DB.Exec("UPDATE orders SET address=$1, latitude=$2, longitude=$3, updated_at=NOW() WHERE id=$4",
		address, latitude, longitude, orderID)
	if err != nil {
		log.Printf("UpdateOrderAddress: ошибка обновления адреса для заказа #%d: %v", orderID, err)
		return err
	}
	return nil
}

// GetOrderStatusAndClientChatID получает статус заказа и user_chat_id клиента по ID заказа.
func GetOrderStatusAndClientChatID(orderID int64) (string, int64, error) {
	var status string
	var clientChatID int64
	err := DB.QueryRow("SELECT status, user_chat_id FROM orders WHERE id = $1", orderID).Scan(&status, &clientChatID)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", 0, fmt.Errorf("заказ с ID %d не найден", orderID)
		}
		log.Printf("GetOrderStatusAndClientChatID: ошибка получения статуса и chat_id клиента для заказа #%d: %v", orderID, err)
		return "", 0, err
	}
	return status, clientChatID, nil
}

// UpdateOrderStatusAndReason обновляет статус и причину отмены заказа.
func UpdateOrderStatusAndReason(orderID int64, status string, reason sql.NullString) error {
	_, err := DB.Exec("UPDATE orders SET status=$1, reason=$2, updated_at=NOW() WHERE id=$3", status, reason, orderID)
	if err != nil {
		log.Printf("UpdateOrderStatusAndReason: ошибка обновления статуса/причины для заказа #%d: %v", orderID, err)
		return err
	}
	log.Printf("Статус (%s) и причина для заказа #%d обновлены.", status, orderID)
	return nil
}

// НОВАЯ ФУНКЦИЯ: GetUnsettledCompletedOrdersForDriver
// Эта функция извлекает все заказы, которые были выполнены водителем (статус COMPLETED),
// но по которым еще не был произведен расчет (is_driver_settled = FALSE).
func GetUnsettledCompletedOrdersForDriver(driverUserID int64) ([]models.Order, error) {
	query := `
        SELECT o.id, o.user_id, o.user_chat_id, o.category, o.subcategory, o.name,
               o.photos, o.videos, o.date, o.time, o.phone, o.address,
               o.description, o.status, o.reason, o.cost, o.payment,
               o.latitude, o.longitude, o.created_at, o.updated_at, o.is_driver_settled
        FROM orders o
        INNER JOIN executors ex ON o.id = ex.order_id
        WHERE ex.user_id = $1 AND ex.role = $2
          AND o.status = $3 AND o.is_driver_settled = FALSE
        ORDER BY o.created_at ASC` // Сортируем по дате создания, чтобы обеспечить последовательность

	rows, err := DB.Query(query, driverUserID, constants.ROLE_DRIVER, constants.STATUS_COMPLETED)
	if err != nil {
		log.Printf("GetUnsettledCompletedOrdersForDriver: ошибка получения нерассчитанных заказов для водителя UserID %d: %v", driverUserID, err)
		return nil, err
	}
	defer rows.Close()

	var orders []models.Order
	for rows.Next() {
		var o models.Order
		var dbDate sql.NullTime
		var dbTime sql.NullString
		var dbCreatedAt, dbUpdatedAt sql.NullTime // Объявляем здесь, чтобы избежать затенения
		errScan := rows.Scan(
			&o.ID, &o.UserID, &o.UserChatID, &o.Category, &o.Subcategory, &o.Name,
			pq.Array(&o.Photos), pq.Array(&o.Videos), &dbDate, &dbTime, &o.Phone, &o.Address,
			&o.Description, &o.Status, &o.Reason, &o.Cost, &o.Payment,
			&o.Latitude, &o.Longitude, &dbCreatedAt, &dbUpdatedAt, &o.IsDriverSettled,
		)
		if errScan != nil {
			log.Printf("GetUnsettledCompletedOrdersForDriver: ошибка сканирования заказа: %v", errScan)
			continue // Пропускаем этот заказ, но продолжаем с остальными
		}
		if dbDate.Valid {
			o.Date = dbDate.Time.Format("2006-01-02")
		} else {
			o.Date = ""
		}
		if dbTime.Valid {
			o.Time = dbTime.String
		} else {
			o.Time = ""
		}
		if dbCreatedAt.Valid { // Проверяем Valid перед присвоением
			o.CreatedAt = dbCreatedAt.Time
		}
		if dbUpdatedAt.Valid { // Проверяем Valid перед присвоением
			o.UpdatedAt = dbUpdatedAt.Time
		}
		orders = append(orders, o)
	}
	if err = rows.Err(); err != nil { // Проверка ошибок после цикла
		log.Printf("GetUnsettledCompletedOrdersForDriver: ошибка после итерации по строкам: %v", err)
		return nil, err
	}
	return orders, nil
}

// НОВАЯ ФУНКЦИЯ: MarkOrdersAsSettled (вызывается внутри транзакции)
// Помечает заказы как рассчитанные с водителем.
func MarkOrdersAsSettled(tx *sql.Tx, orderIDs []int64) error {
	if len(orderIDs) == 0 {
		return nil // Нечего обновлять
	}
	query := `UPDATE orders SET is_driver_settled = TRUE, updated_at = NOW() WHERE id = ANY($1::bigint[])`
	result, err := tx.Exec(query, pq.Array(orderIDs))
	if err != nil {
		log.Printf("MarkOrdersAsSettled: ошибка обновления статуса is_driver_settled для заказов: %v", err)
		return err
	}
	rowsAffected, _ := result.RowsAffected()
	log.Printf("MarkOrdersAsSettled: %d заказа(ов) помечены как is_driver_settled=TRUE.", rowsAffected)
	if rowsAffected != int64(len(orderIDs)) {
		log.Printf("MarkOrdersAsSettled: ВНИМАНИЕ! Ожидалось обновление %d заказов, но обновлено %d.", len(orderIDs), rowsAffected)
		// Это может быть не критической ошибкой, если некоторые ID были неверны, но стоит залогировать.
	}
	return nil
}

// CancelUserActiveOrdersInTx отменяет все активные заказы пользователя в рамках транзакции.
// Активными считаются заказы, не имеющие статусов COMPLETED, CANCELED, CALCULATED, SETTLED.
func CancelUserActiveOrdersInTx(tx *sql.Tx, userID int64, reason string) error {
	activeStatusesToCancel := []string{
		constants.STATUS_NEW,
		constants.STATUS_AWAITING_COST,
		constants.STATUS_AWAITING_CONFIRMATION,
		constants.STATUS_INPROGRESS,
		constants.STATUS_DRAFT,
	}

	// Сначала получим ID заказов, которые нужно отменить
	var orderIDsToCancel []int64
	querySelect := `
        SELECT id
        FROM orders
        WHERE user_id = $1 AND status = ANY($2)`
	rows, err := tx.Query(querySelect, userID, pq.Array(activeStatusesToCancel))
	if err != nil {
		log.Printf("CancelUserActiveOrdersInTx: ошибка получения списка активных заказов для пользователя ID %d: %v", userID, err)
		return fmt.Errorf("ошибка получения списка активных заказов пользователя: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var orderID int64
		if errScan := rows.Scan(&orderID); errScan != nil {
			log.Printf("CancelUserActiveOrdersInTx: ошибка сканирования ID заказа для пользователя ID %d: %v", userID, errScan)
			// Продолжаем, чтобы попытаться отменить другие заказы
			continue
		}
		orderIDsToCancel = append(orderIDsToCancel, orderID)
	}
	if err = rows.Err(); err != nil {
		log.Printf("CancelUserActiveOrdersInTx: ошибка после итерации по заказам пользователя ID %d: %v", userID, err)
		return fmt.Errorf("ошибка обработки списка активных заказов пользователя: %w", err)
	}

	if len(orderIDsToCancel) == 0 {
		log.Printf("CancelUserActiveOrdersInTx: Нет активных заказов для отмены у пользователя ID %d.", userID)
		return nil // Нет заказов для отмены
	}

	log.Printf("CancelUserActiveOrdersInTx: Будут отменены следующие заказы пользователя ID %d: %v", userID, orderIDsToCancel)

	queryUpdate := `
        UPDATE orders
        SET status = $1, reason = $2, updated_at = NOW()
        WHERE user_id = $3 AND id = ANY($4::bigint[]) AND status = ANY($5)` // Дополнительная проверка статуса на случай гонки

	result, err := tx.Exec(queryUpdate, constants.STATUS_CANCELED, reason, userID, pq.Array(orderIDsToCancel), pq.Array(activeStatusesToCancel))
	if err != nil {
		log.Printf("CancelUserActiveOrdersInTx: ошибка обновления статусов заказов для пользователя ID %d: %v", userID, err)
		return fmt.Errorf("ошибка отмены активных заказов пользователя: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	log.Printf("CancelUserActiveOrdersInTx: %d активных заказа(ов) пользователя ID %d были отменены. Причина: %s", rowsAffected, userID, reason)
	if rowsAffected != int64(len(orderIDsToCancel)) {
		log.Printf("CancelUserActiveOrdersInTx: ВНИМАНИЕ! Ожидалась отмена %d заказов, но отменено %d. Возможна гонка состояний или не все ID были корректны.", len(orderIDsToCancel), rowsAffected)
	}

	return nil
}

// CreateFullOrder создает заказ со всеми данными, полученными из API.
func CreateFullOrder(orderData models.Order) (int64, error) {
	// Проверяем наличие обязательного поля - ID пользователя-клиента
	if orderData.UserID == 0 {
		return 0, errors.New("UserID (идентификатор клиента) не установлен для заказа")
	}

	tx, err := DB.Begin()
	if err != nil {
		log.Printf("CreateFullOrder: Ошибка начала транзакции: %v", err)
		return 0, err
	}
	defer tx.Rollback() // Откат, если Commit не будет вызван

	// Преобразуем строковую дату в тип DATE для БД
	var parsedDate sql.NullTime
	if strings.TrimSpace(orderData.Date) != "" {
		// Ожидаем формат yyyy-MM-DD от HTML-инпута
		pDate, errDate := time.Parse("2006-01-02", orderData.Date)
		if errDate != nil {
			log.Printf("CreateFullOrder: Ошибка парсинга даты '%s': %v", orderData.Date, errDate)
			// Можно вернуть ошибку или продолжить с NULL датой
			parsedDate = sql.NullTime{Valid: false}
		} else {
			parsedDate = sql.NullTime{Time: pDate, Valid: true}
		}
	} else {
		parsedDate = sql.NullTime{Valid: false}
	}

	// Устанавливаем статус по умолчанию, если он не передан
	if orderData.Status == "" {
		orderData.Status = constants.STATUS_NEW
	}

	var id int64
	query := `
        INSERT INTO orders (
            user_id, user_chat_id, category, subcategory, name,
            photos, videos, date, time, phone, address,
            description, status, cost, payment,
            latitude, longitude, created_at, updated_at, is_driver_settled
        )
        VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, NOW(), NOW(), FALSE)
        RETURNING id`

	err = tx.QueryRow(query,
		orderData.UserID, orderData.UserChatID, orderData.Category, orderData.Subcategory, orderData.Name,
		pq.Array(orderData.Photos), pq.Array(orderData.Videos), parsedDate, orderData.Time,
		orderData.Phone, orderData.Address, orderData.Description, orderData.Status,
		orderData.Cost, orderData.Payment, orderData.Latitude, orderData.Longitude,
	).Scan(&id)

	if err != nil {
		log.Printf("CreateFullOrder: Ошибка выполнения INSERT для заказа (клиент ID %d): %v", orderData.UserID, err)
		return 0, err
	}

	err = tx.Commit()
	if err != nil {
		log.Printf("CreateFullOrder: Ошибка фиксации транзакции: %v", err)
		return 0, err
	}

	log.Printf("Заказ #%d успешно создан через API для клиента ID %d.", id, orderData.UserID)
	return id, nil
}

// GetOrderCountForUser возвращает общее количество заказов для указанного пользователя.
func GetOrderCountForUser(userID int64) (int, error) {
	var count int
	query := `SELECT COUNT(*) FROM orders WHERE user_id = $1`
	err := DB.QueryRow(query, userID).Scan(&count)
	if err != nil {
		log.Printf("GetOrderCountForUser: Ошибка подсчета заказов для UserID %d: %v", userID, err)
		return 0, err
	}
	return count, nil
}

// GetOrdersByChatIDAndMultipleStatuses извлекает заказы пользователя по его user_chat_id и списку статусов.
// Поле Date возвращается в формате "YYYY-MM-DD".
func GetOrdersByChatIDAndMultipleStatuses(userChatID int64, statuses []string, page int) ([]models.Order, error) {
	offset := page * constants.OrdersPerPage
	var rows *sql.Rows
	var err error

	// Создаем динамический список плейсхолдеров для IN-оператора
	placeholders := make([]string, len(statuses))
	queryParams := make([]interface{}, len(statuses)+2) // +2 для userChatID и offset
	queryParams[0] = userChatID

	for i, status := range statuses {
		placeholders[i] = fmt.Sprintf("$%d", i+2) // $2, $3, ...
		queryParams[i+1] = status
	}
	queryParams[len(queryParams)-1] = offset // Последний плейсхолдер для OFFSET

	query := fmt.Sprintf(`
        SELECT o.id, o.user_id, o.user_chat_id, o.category, o.subcategory, o.name,
               o.photos, o.videos, o.date, o.time, o.phone, o.address,
               o.description, o.status, o.reason, o.cost, o.payment,
               o.latitude, o.longitude, o.created_at, o.updated_at, o.is_driver_settled
        FROM orders o
        WHERE o.user_chat_id = $1 AND o.status IN (%s)
        ORDER BY o.created_at DESC
        LIMIT %d OFFSET $%d`, strings.Join(placeholders, ","), constants.OrdersPerPage, len(queryParams))

	rows, err = DB.Query(query, queryParams...)
	if err != nil {
		log.Printf("GetOrdersByChatIDAndMultipleStatuses: ошибка запроса заказов для chatID %d, статусы '%v': %v", userChatID, statuses, err)
		return nil, err
	}
	defer rows.Close()

	var orders []models.Order
	for rows.Next() {
		var o models.Order
		var dbDate sql.NullTime
		var dbTime sql.NullString
		var dbCreatedAt, dbUpdatedAt sql.NullTime
		var photos, videos pq.StringArray // Для сканирования массивов

		errScan := rows.Scan(
			&o.ID, &o.UserID, &o.UserChatID, &o.Category, &o.Subcategory, &o.Name,
			&photos, &videos, &dbDate, &dbTime, &o.Phone, &o.Address,
			&o.Description, &o.Status, &o.Reason, &o.Cost, &o.Payment,
			&o.Latitude, &o.Longitude, &dbCreatedAt, &dbUpdatedAt, &o.IsDriverSettled,
		)
		if errScan != nil {
			log.Printf("GetOrdersByChatIDAndMultipleStatuses: ошибка сканирования строки: %v", errScan)
			return nil, errScan
		}

		o.Photos = []string(photos)
		o.Videos = []string(videos)

		if dbDate.Valid {
			o.Date = dbDate.Time.Format("2006-01-02")
		} else {
			o.Date = ""
		}
		if dbTime.Valid {
			o.Time = dbTime.String
		} else {
			o.Time = ""
		}
		if dbCreatedAt.Valid {
			o.CreatedAt = dbCreatedAt.Time
		}
		if dbUpdatedAt.Valid {
			o.UpdatedAt = dbUpdatedAt.Time
		}
		orders = append(orders, o)
	}

	if err = rows.Err(); err != nil {
		log.Printf("GetOrdersByChatIDAndMultipleStatuses: ошибка после итерации по строкам: %v", err)
		return nil, err
	}

	return orders, nil
}
