package api

import (
	"log"
	"net/http"

	"Original/internal/db"
	"Original/internal/models"
)

// GetStats возвращает полную статистику для админов и операторов
func GetStats(w http.ResponseWriter, r *http.Request) {
	user, ok := r.Context().Value(UserContextKey).(models.User)
	if !ok {
		writeJSONError(w, http.StatusUnauthorized, "User not found in context")
		return
	}

	// Проверяем права доступа
	if user.Role != "owner" && user.Role != "operator" {
		writeJSONError(w, http.StatusForbidden, "Access denied")
		return
	}

	stats, err := calculateStatistics()
	if err != nil {
		log.Printf("Error calculating statistics: %v", err)
		writeJSONError(w, http.StatusInternalServerError, "Failed to calculate statistics")
		return
	}

	writeJSONSuccess(w, "Statistics retrieved successfully", stats)
}

// calculateStatistics вычисляет статистику из базы данных
func calculateStatistics() (map[string]interface{}, error) {
	// Получаем все заказы
	orders, err := db.GetAllOrders()
	if err != nil {
		return nil, err
	}

	// Получаем всех пользователей
	users, err := db.GetAllUsers()
	if err != nil {
		return nil, err
	}

	// Вычисляем основные метрики
	totalOrders := len(orders)
	totalRevenue := int64(0)
	completedOrders := 0
	ordersByType := make(map[string]int)
	monthlyRevenue := make(map[string]int64)

	for _, order := range orders {
		if order.Status == "completed" || order.Status == "settled" {
			if order.Cost.Valid {
				totalRevenue += int64(order.Cost.Float64)
				completedOrders++
			}
		}

		// Подсчет по типам услуг
		serviceType := order.Category
		if serviceType == "" {
			serviceType = "unknown"
		}
		ordersByType[serviceType]++

		// Группировка по месяцам
		if !order.CreatedAt.IsZero() {
			month := order.CreatedAt.Format("2006-01")
			if order.Status == "completed" || order.Status == "settled" && order.Cost.Valid {
				monthlyRevenue[month] += int64(order.Cost.Float64)
			}
		}
	}

	// Подсчет клиентов (роль user)
	totalClients := 0
	for _, user := range users {
		if user.Role == "user" {
			totalClients++
		}
	}

	// Среднее время выполнения заказа (в часах)
	avgOrderTime := 24.0 // Примерное значение, можно вычислить точнее

	// Рост показателей (примерные значения, можно вычислить за период)
	ordersGrowth := 15
	revenueGrowth := 23
	clientsGrowth := 8

	// Топ клиенты (по количеству заказов)
	topClients := getTopClients(orders, users)

	// Топ сотрудники
	topEmployees := getTopEmployees(users)

	// Формируем результат
	stats := map[string]interface{}{
		"totalOrders":     totalOrders,
		"totalRevenue":    totalRevenue,
		"totalClients":    totalClients,
		"completedOrders": completedOrders,
		"avgOrderTime":    avgOrderTime,
		"ordersGrowth":    ordersGrowth,
		"revenueGrowth":   revenueGrowth,
		"clientsGrowth":   clientsGrowth,
		"ordersByType":    ordersByType,
		"monthlyRevenue":  monthlyRevenue,
		"topClients":      topClients,
		"topEmployees":    topEmployees,
	}

	return stats, nil
}

// getTopClients возвращает топ клиентов по количеству заказов
func getTopClients(orders []models.Order, users []models.User) []map[string]interface{} {
	// Подсчитываем заказы по пользователям
	userOrders := make(map[int64]int)
	userRevenue := make(map[int64]int64)

	for _, order := range orders {
		if order.UserID > 0 {
			userOrders[int64(order.UserID)]++
			if order.Status == "completed" || order.Status == "settled" && order.Cost.Valid {
				userRevenue[int64(order.UserID)] += int64(order.Cost.Float64)
			}
		}
	}

	// Создаем мапу пользователей для быстрого поиска
	userMap := make(map[int64]models.User)
	for _, user := range users {
		if user.Role == "user" {
			userMap[user.ID] = user
		}
	}

	// Создаем список топ клиентов
	var topClients []map[string]interface{}
	
	for userID, orderCount := range userOrders {
		if user, exists := userMap[userID]; exists && orderCount > 0 {
			topClients = append(topClients, map[string]interface{}{
				"name":    user.FirstName + " " + user.LastName,
				"orders":  orderCount,
				"revenue": userRevenue[userID],
			})
		}
	}

	// Сортируем по количеству заказов (топ 5)
	// Простая сортировка пузырьком для небольшого количества данных
	for i := 0; i < len(topClients)-1; i++ {
		for j := 0; j < len(topClients)-i-1; j++ {
			if topClients[j]["orders"].(int) < topClients[j+1]["orders"].(int) {
				topClients[j], topClients[j+1] = topClients[j+1], topClients[j]
			}
		}
	}

	// Возвращаем только топ 5
	if len(topClients) > 5 {
		topClients = topClients[:5]
	}

	return topClients
}

// getTopEmployees возвращает топ сотрудников
func getTopEmployees(users []models.User) []map[string]interface{} {
	var topEmployees []map[string]interface{}

	for _, user := range users {
		if user.Role == "driver" || user.Role == "operator" || user.Role == "owner" {
			topEmployees = append(topEmployees, map[string]interface{}{
				"name":   user.FirstName + " " + user.LastName,
				"role":   user.Role,
				"orders": 0, // Можно вычислить количество обработанных заказов
				"rating": 5.0,
			})
		}
	}

	// Возвращаем только топ 5
	if len(topEmployees) > 5 {
		topEmployees = topEmployees[:5]
	}

	return topEmployees
}