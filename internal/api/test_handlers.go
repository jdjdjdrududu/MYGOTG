package api

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
)

// GetTestOrders возвращает тестовые заказы
func GetTestOrders(w http.ResponseWriter, r *http.Request) {
	log.Println("⚠️ ТЕСТОВЫЙ ЗАПРОС: Используются тестовые заказы")

	status := r.URL.Query().Get("status")

	// Создаем тестовые заказы
	orders := []map[string]interface{}{
		{
			"ID":              1001,
			"UserID":          1,
			"UserChatID":      1263060321,
			"Category":        "Демонтаж",
			"Subcategory":     "Квартира",
			"ServiceType":     "Демонтаж квартиры",
			"ClientName":      "Мария Петрова",
			"ClientFirstName": "Мария",
			"ClientPhone":     "+79787654321",
			"Phone":           "+79787654321",
			"Address":         "г. Симферополь, ул. Киевская, д. 15, кв. 42",
			"Description":     "Необходим полный демонтаж старой кухни перед ремонтом. Включает демонтаж кухонного гарнитура, плитки, старой проводки.",
			"Status":          "new",
			"Cost":            15000.0,
			"Price":           15000.0,
			"Payment":         "after",
			"Date":            "2024-07-25",
			"Time":            "10:00",
			"CreatedAt":       time.Now().Add(-2 * time.Hour).Format(time.RFC3339),
			"UpdatedAt":       time.Now().Add(-2 * time.Hour).Format(time.RFC3339),
			"OperatorName":    "Александр Иванов",
			"Photos":          []string{},
			"Videos":          []string{},
		},
		{
			"ID":              1002,
			"UserID":          2,
			"UserChatID":      1234567890,
			"Category":        "Вывоз мусора",
			"Subcategory":     "Строительный",
			"ServiceType":     "Вывоз строительного мусора",
			"ClientName":      "Иван Сидоров",
			"ClientFirstName": "Иван",
			"ClientPhone":     "+79788888888",
			"Phone":           "+79788888888",
			"Address":         "г. Севастополь, ул. Ленина, д. 45",
			"Description":     "После ремонта накопилось около 3 кубов строительного мусора. Нужен вывоз и утилизация.",
			"Status":          "in_progress",
			"Cost":            8000.0,
			"Price":           8000.0,
			"Payment":         "now",
			"Date":            "2024-07-24",
			"Time":            "14:00",
			"CreatedAt":       time.Now().Add(-24 * time.Hour).Format(time.RFC3339),
			"UpdatedAt":       time.Now().Add(-1 * time.Hour).Format(time.RFC3339),
			"OperatorName":    "Александр Иванов",
			"Photos":          []string{"photo1.jpg", "photo2.jpg"},
			"Videos":          []string{},
		},
		{
			"ID":              1003,
			"UserID":          3,
			"UserChatID":      9876543210,
			"Category":        "Доставка материалов",
			"Subcategory":     "Песок",
			"ServiceType":     "Доставка песка",
			"ClientName":      "Андрей Кузнецов",
			"ClientFirstName": "Андрей",
			"ClientPhone":     "+79789999999",
			"Phone":           "+79789999999",
			"Address":         "г. Ялта, ул. Морская, участок 25",
			"Description":     "Нужна доставка 5 тонн речного песка для строительства. Подъезд для грузовика есть.",
			"Status":          "completed",
			"Cost":            12000.0,
			"Price":           12000.0,
			"Payment":         "after",
			"Date":            "2024-07-22",
			"Time":            "09:00",
			"CreatedAt":       time.Now().Add(-72 * time.Hour).Format(time.RFC3339),
			"UpdatedAt":       time.Now().Add(-48 * time.Hour).Format(time.RFC3339),
			"OperatorName":    "Александр Иванов",
			"Photos":          []string{},
			"Videos":          []string{},
		},
		{
			"ID":              1004,
			"UserID":          4,
			"UserChatID":      5555555555,
			"Category":        "Демонтаж",
			"Subcategory":     "Дом",
			"ServiceType":     "Демонтаж дома",
			"ClientName":      "Елена Васильева",
			"ClientFirstName": "Елена",
			"ClientPhone":     "+79787777777",
			"Phone":           "+79787777777",
			"Address":         "г. Алушта, ул. Садовая, д. 8",
			"Description":     "Требуется снос старого дома площадью 80 кв.м. Вывоз мусора включен в стоимость.",
			"Status":          "awaiting_confirmation",
			"Cost":            45000.0,
			"Price":           45000.0,
			"Payment":         "after",
			"Date":            "2024-07-28",
			"Time":            "СРОЧНО",
			"CreatedAt":       time.Now().Add(-30 * time.Minute).Format(time.RFC3339),
			"UpdatedAt":       time.Now().Add(-30 * time.Minute).Format(time.RFC3339),
			"OperatorName":    "Александр Иванов",
			"Photos":          []string{"house1.jpg", "house2.jpg", "house3.jpg"},
			"Videos":          []string{"video1.mp4"},
		},
		{
			"ID":                 1005,
			"UserID":             5,
			"UserChatID":         4444444444,
			"Category":           "Вывоз мусора",
			"Subcategory":        "Бытовой",
			"ServiceType":        "Вывоз бытового мусора",
			"ClientName":         "Николай Смирнов",
			"ClientFirstName":    "Николай",
			"ClientPhone":        "+79786666666",
			"Phone":              "+79786666666",
			"Address":            "г. Феодосия, ул. Крымская, д. 12",
			"Description":        "Накопился крупногабаритный мусор: старая мебель, холодильник, стиральная машина.",
			"Status":             "canceled",
			"Cost":               5000.0,
			"Price":              5000.0,
			"Payment":            "now",
			"Date":               "2024-07-23",
			"Time":               "16:00",
			"CreatedAt":          time.Now().Add(-48 * time.Hour).Format(time.RFC3339),
			"UpdatedAt":          time.Now().Add(-24 * time.Hour).Format(time.RFC3339),
			"OperatorName":       "Александр Иванов",
			"CancellationReason": "Клиент отменил заказ: нашел другого исполнителя",
			"Photos":             []string{},
			"Videos":             []string{},
		},
	}

	// Фильтруем по статусу если указан
	if status != "" && status != "active" {
		filteredOrders := []map[string]interface{}{}
		for _, order := range orders {
			if order["Status"] == status {
				filteredOrders = append(filteredOrders, order)
			}
		}
		orders = filteredOrders
	} else if status == "active" {
		// Для активных показываем new, in_progress, awaiting_confirmation
		filteredOrders := []map[string]interface{}{}
		for _, order := range orders {
			orderStatus := order["Status"].(string)
			if orderStatus == "new" || orderStatus == "in_progress" || orderStatus == "awaiting_confirmation" {
				filteredOrders = append(filteredOrders, order)
			}
		}
		orders = filteredOrders
	}

	// Возвращаем в формате API
	response := map[string]interface{}{
		"status":  "success",
		"message": "Orders retrieved successfully",
		"data":    orders,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetTestClients возвращает тестовых клиентов
func GetTestClients(w http.ResponseWriter, r *http.Request) {
	log.Println("⚠️ ТЕСТОВЫЙ ЗАПРОС: Используются тестовые клиенты")

	// Создаем тестовых клиентов
	clients := []map[string]interface{}{
		{
			"ID":         2,
			"ChatID":     1234567890,
			"Role":       "user",
			"FirstName":  "Иван",
			"LastName":   "Сидоров",
			"Username":   "ivan_sid",
			"Phone":      "+79788888888",
			"IsBlocked":  false,
			"CreatedAt":  "2024-06-15T10:00:00Z",
			"UpdatedAt":  "2024-07-23T14:00:00Z",
			"OrderCount": 15,
			"CardNumber": "5555123456789012",
		},
		{
			"ID":         3,
			"ChatID":     9876543210,
			"Role":       "user",
			"FirstName":  "Андрей",
			"LastName":   "Кузнецов",
			"Username":   "andrey_k",
			"Phone":      "+79789999999",
			"IsBlocked":  false,
			"CreatedAt":  "2024-05-20T10:00:00Z",
			"UpdatedAt":  "2024-07-22T09:00:00Z",
			"OrderCount": 8,
		},
		{
			"ID":         4,
			"ChatID":     5555555555,
			"Role":       "user",
			"FirstName":  "Елена",
			"LastName":   "Васильева",
			"Username":   "",
			"Phone":      "+79787777777",
			"IsBlocked":  false,
			"CreatedAt":  "2024-07-10T10:00:00Z",
			"UpdatedAt":  "2024-07-23T11:00:00Z",
			"OrderCount": 3,
		},
		{
			"ID":          5,
			"ChatID":      4444444444,
			"Role":        "user",
			"FirstName":   "Николай",
			"LastName":    "Смирнов",
			"Username":    "n_smirnov",
			"Phone":       "+79786666666",
			"IsBlocked":   true,
			"BlockReason": "Многократная отмена заказов без уважительной причины",
			"CreatedAt":   "2024-04-01T10:00:00Z",
			"UpdatedAt":   "2024-07-20T10:00:00Z",
			"OrderCount":  12,
		},
		{
			"ID":         6,
			"ChatID":     3333333333,
			"Role":       "driver",
			"FirstName":  "Сергей",
			"LastName":   "Козлов",
			"Username":   "driver_sergey",
			"Phone":      "+79785555555",
			"IsBlocked":  false,
			"CreatedAt":  "2024-03-15T10:00:00Z",
			"UpdatedAt":  "2024-07-23T08:00:00Z",
			"OrderCount": 156,
		},
		{
			"ID":         7,
			"ChatID":     2222222222,
			"Role":       "owner",
			"FirstName":  "Дмитрий",
			"LastName":   "Петров",
			"Username":   "boss_dp",
			"Phone":      "+79784444444",
			"IsBlocked":  false,
			"CreatedAt":  "2024-01-01T10:00:00Z",
			"UpdatedAt":  "2024-07-23T16:00:00Z",
			"OrderCount": 0,
		},
	}

	// Фильтруем по роли если указана
	role := r.URL.Query().Get("role")
	if role != "" {
		filteredClients := []map[string]interface{}{}
		for _, client := range clients {
			if client["Role"] == role {
				filteredClients = append(filteredClients, client)
			}
		}
		clients = filteredClients
	}

	// Возвращаем в формате API
	response := map[string]interface{}{
		"status":  "success",
		"message": "Clients retrieved successfully",
		"data":    clients,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetTestStats возвращает тестовые данные статистики
func GetTestStats(w http.ResponseWriter, r *http.Request) {
	stats := map[string]interface{}{
		"totalOrders":     156,
		"totalRevenue":    2850000,
		"totalClients":    89,
		"avgOrderTime":    4.2,
		"ordersGrowth":    15,
		"revenueGrowth":   23,
		"clientsGrowth":   8,
		"monthlyRevenue": []map[string]interface{}{
			{"month": "Январь", "revenue": 245000},
			{"month": "Февраль", "revenue": 289000},
			{"month": "Март", "revenue": 356000},
			{"month": "Апрель", "revenue": 412000},
			{"month": "Май", "revenue": 387000},
			{"month": "Июнь", "revenue": 445000},
		},
		"ordersByType": map[string]int{
			"демонтаж":           70,
			"вывоз_мусора":      55,
			"демонтаж_и_вывоз":  31,
		},
		"topClients": []map[string]interface{}{
			{"name": "Иван Сидоров", "orders": 15, "revenue": 185000},
			{"name": "Андрей Кузнецов", "orders": 8, "revenue": 92000},
			{"name": "Елена Васильева", "orders": 3, "revenue": 35000},
		},
		"topEmployees": []map[string]interface{}{
			{"name": "Сергей Козлов", "role": "driver", "orders": 156, "rating": 4.9},
			{"name": "Дмитрий Петров", "role": "owner", "orders": 0, "rating": 5.0},
		},
	}

	response := map[string]interface{}{
		"status":  "success",
		"message": "Statistics retrieved successfully",
		"data":    stats,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetTestOrderDetails возвращает детали тестового заказа
func GetTestOrderDetails(w http.ResponseWriter, r *http.Request) {
	orderID := chi.URLParam(r, "id")
	log.Printf("⚠️ ТЕСТОВЫЙ ЗАПРОС: Детали заказа #%s", orderID)

	// Создаем детальную информацию о заказе
	order := map[string]interface{}{
		"ID":              orderID,
		"UserID":          1,
		"UserChatID":      1263060321,
		"Category":        "Демонтаж",
		"Subcategory":     "Квартира",
		"ServiceType":     "Демонтаж квартиры",
		"ClientName":      "Мария Петрова",
		"ClientFirstName": "Мария",
		"ClientPhone":     "+79787654321",
		"Phone":           "+79787654321",
		"Address":         "г. Симферополь, ул. Киевская, д. 15, кв. 42",
		"Description":     "Необходим полный демонтаж старой кухни перед ремонтом. Включает демонтаж кухонного гарнитура, плитки, старой проводки. Вывоз мусора обязателен.",
		"Notes":           "Клиент будет дома с 10:00 до 18:00. Есть грузовой лифт.",
		"Status":          "new",
		"Cost":            15000.0,
		"Price":           15000.0,
		"Payment":         "after",
		"Date":            "2024-07-25",
		"Time":            "10:00",
		"CreatedAt":       time.Now().Add(-2 * time.Hour).Format(time.RFC3339),
		"UpdatedAt":       time.Now().Add(-2 * time.Hour).Format(time.RFC3339),
		"OperatorName":    "Александр Иванов",
		"OperatorID":      1,
		"Photos":          []string{"kitchen1.jpg", "kitchen2.jpg", "kitchen3.jpg"},
		"Videos":          []string{"kitchen_video.mp4"},
		"Latitude":        44.952117,
		"Longitude":       34.102417,
		"Executors": []map[string]interface{}{
			{
				"UserID":    6,
				"Role":      "driver",
				"FirstName": "Сергей",
				"LastName":  "Козлов",
				"Phone":     "+79785555555",
				"Confirmed": false,
			},
		},
	}

	// Возвращаем в формате API
	response := map[string]interface{}{
		"status":  "success",
		"message": "Order details retrieved successfully",
		"data":    order,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
