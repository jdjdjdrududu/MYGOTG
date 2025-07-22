package api

import (
	"Original/internal/config"
	"Original/internal/constants"
	"Original/internal/handlers"

	"github.com/go-chi/chi/v5"
)

// ApiDependencies содержит зависимости для обработчиков API.
type ApiDependencies struct {
	Config    *config.Config
	SecretKey string
	Bot       *handlers.BotHandler
}

// SetupRoutes настраивает все маршруты для API.
func SetupRoutes(r *chi.Mux, deps ApiDependencies) {
	// Middleware уже добавлен в main.go, не добавляем его здесь

	r.Group(func(r chi.Router) {
		r.Use(ConfigMiddleware(deps.Config))
		r.Get("/api/client-config", GetClientConfig)
	})

	// ВРЕМЕННЫЙ ТЕСТОВЫЙ МАРШРУТ для отладки (БЕЗ АУТЕНТИФИКАЦИИ)
	r.Get("/api/test/profile", GetTestUserProfile)

	// Этот маршрут должен быть публичным, но с проверкой доступа внутри обработчика
	// Используем MediaProxyHandler вместо ServeMediaHandler для безопасной отдачи файлов
	r.Get("/api/media/{filename}", MediaProxyHandler)

	r.Group(func(r chi.Router) {
		r.Use(AuthMiddleware(deps.SecretKey))
		r.Use(BotMiddleware(deps.Bot))

		// === Маршрут для загрузки файлов (остается защищенным) ===
		r.Post("/api/upload-media", UploadMediaHandler)

		// --- Маршруты для обычных пользователей ---
		r.Get("/api/user/profile", GetUserProfile)
		r.Get("/api/user/orders", GetOrders)
		// --- НАЧАЛО ИЗМЕНЕНИЯ ---
		// Связываем маршрут пользователя с правильным обработчиком CreateUserOrder.
		r.Post("/api/user/create-order", CreateUserOrder)
		// --- КОНЕЦ ИЗМЕНЕНИЯ ---
		r.Get("/api/user/order/{id}", GetOrderDetails)
		r.Post("/api/user/order/{id}/action", HandleUserOrderAction)

		// --- Маршруты для админов/операторов ---
		r.Route("/api/admin", func(r chi.Router) {
			r.Use(RoleMiddleware(constants.ROLE_OPERATOR))

			r.Get("/orders", GetOrders)
			r.Get("/clients", GetClients)
			// Маршрут оператора остается связанным с CreateOrder, который не отправляет уведомления.
			r.Post("/create-order", CreateOrder)
			r.Get("/client/{id}", GetClientDetails)
			r.Get("/order/{id}", GetOrderDetails)
			r.Post("/order/{id}/action", HandleAdminOrderAction)
			r.Post("/order/{id}/update-field", UpdateOrderFieldHandler)
			r.Post("/order/{id}/add-media", AddOrderMedia)
			r.Post("/settlement/{id}/status", UpdateSettlementStatus)
		})

		// --- Маршруты для водителя ---
		r.Route("/api/driver", func(r chi.Router) {
			r.Use(RoleMiddleware(constants.ROLE_DRIVER))
			r.Post("/start-report", StartDriverReport)
			r.Post("/order/{id}/action", HandleDriverOrderAction)
		})
	})
}
