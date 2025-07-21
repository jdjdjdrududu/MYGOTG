package api

import (
	"Original/internal/config"
	"Original/internal/constants"
	"Original/internal/db"
	"Original/internal/handlers"
	"Original/internal/models"
	"Original/internal/utils"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	tgbotapi "github.com/OvyFlash/telegram-bot-api"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// --- Абсолютный путь к хранилищу ---

var (
	mediaStoragePath string
	once             sync.Once
)

// initStoragePath инициализирует абсолютный путь к папке media_storage.
// Папка будет создана в той же директории, где находится исполняемый файл.
func initStoragePath() {
	once.Do(func() {
		executable, err := os.Executable()
		if err != nil {
			log.Fatalf("FATAL: Cannot get executable path: %v", err)
		}
		executableDir := filepath.Dir(executable)
		mediaStoragePath = filepath.Join(executableDir, "media_storage")

		// Создаем директорию, если её нет
		if err := os.MkdirAll(mediaStoragePath, os.ModePerm); err != nil {
			log.Fatalf("FATAL: Cannot create media storage directory at %s: %v", mediaStoragePath, err)
		}
		log.Printf("Media storage initialized at: %s", mediaStoragePath)
	})
}

// jsonResponse - вспомогательная структура для стандартного ответа API
type jsonResponse struct {
	Status  string      `json:"status"` // "success" или "error"
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// ClientDetailsResponse - Структура для детального ответа о клиенте, включая заказы.
type ClientDetailsResponse struct {
	User       models.User    `json:"User"`
	OrderCount int            `json:"order_count"`
	Orders     []models.Order `json:"orders"`
}

// OrderActionRequest - структура для запросов на изменение заказа
type OrderActionRequest struct {
	Action string      `json:"action"`
	Reason string      `json:"reason,omitempty"`
	Cost   json.Number `json:"cost,omitempty"`
}

// SettlementStatusRequest - структура для запросов на изменение статуса отчета
type SettlementStatusRequest struct {
	Status string `json:"status"`
	Reason string `json:"reason,omitempty"`
}

// AddMediaRequest определяет структуру для добавления медиа к заказу.
type AddMediaRequest struct {
	Photos []string `json:"photos"`
	Videos []string `json:"videos"`
}

// UpdateOrderFieldRequest определяет структуру для запроса на обновление поля.
type UpdateOrderFieldRequest struct {
	Field string      `json:"field"`
	Value interface{} `json:"value"`
}

// UploadFileResponse - структура ответа для загруженного файла
type UploadFileResponse struct {
	FileID string `json:"file_id"`
	Type   string `json:"type"`
}

// --- Вспомогательные функции для JSON-ответов ---
func writeJSONError(w http.ResponseWriter, statusCode int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(jsonResponse{Status: "error", Message: message})
}

func writeJSONSuccess(w http.ResponseWriter, message string, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(jsonResponse{Status: "success", Message: message, Data: data})
}

// ServeMediaHandler обслуживает статические медиафайлы, сохраненные локально.
func ServeMediaHandler(w http.ResponseWriter, r *http.Request) {
	initStoragePath() // Убедимся, что путь инициализирован

	filename := chi.URLParam(r, "filename")
	if filename == "" {
		writeJSONError(w, http.StatusBadRequest, "Filename is required")
		return
	}

	if strings.Contains(filename, "..") || strings.Contains(filename, "/") || strings.Contains(filename, "\\") {
		writeJSONError(w, http.StatusBadRequest, "Invalid filename")
		return
	}

	// ИСПОЛЬЗУЕМ АБСОЛЮТНЫЙ ПУТЬ
	filePath := filepath.Join(mediaStoragePath, filename)

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		http.NotFound(w, r)
		return
	}

	http.ServeFile(w, r, filePath)
}

// UploadMediaHandler обрабатывает загрузку одного медиафайла от WebApp,
// сохраняет его локально и возвращает уникальное имя файла.
func UploadMediaHandler(w http.ResponseWriter, r *http.Request) {
	initStoragePath() // Убедимся, что путь и папка готовы

	if err := r.ParseMultipartForm(32 << 20); err != nil { // 32 MB
		writeJSONError(w, http.StatusBadRequest, "Failed to parse multipart form: "+err.Error())
		return
	}

	file, handler, err := r.FormFile("media")
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "Failed to get media file from form: "+err.Error())
		return
	}
	defer file.Close()

	ext := filepath.Ext(handler.Filename)
	uniqueFilename := fmt.Sprintf("%s%s", uuid.New().String(), ext)

	// ИСПОЛЬЗУЕМ АБСОЛЮТНЫЙ ПУТЬ
	destPath := filepath.Join(mediaStoragePath, uniqueFilename)

	destFile, err := os.Create(destPath)
	if err != nil {
		log.Printf("Failed to create destination file at %s: %v", destPath, err)
		writeJSONError(w, http.StatusInternalServerError, "Could not save file")
		return
	}
	defer destFile.Close()

	if _, err := io.Copy(destFile, file); err != nil {
		log.Printf("Failed to copy file content to %s: %v", destPath, err)
		writeJSONError(w, http.StatusInternalServerError, "Could not save file content")
		return
	}

	var fileType string
	contentType := handler.Header.Get("Content-Type")
	if utils.IsVideo(contentType) {
		fileType = "video"
	} else {
		fileType = "photo"
	}

	writeJSONSuccess(w, "File uploaded successfully", UploadFileResponse{
		FileID: uniqueFilename,
		Type:   fileType,
	})
}

// GetUserProfile возвращает профиль пользователя, прошедшего аутентификацию.
func GetUserProfile(w http.ResponseWriter, r *http.Request) {
	user, ok := r.Context().Value(UserContextKey).(models.User)
	if !ok {
		writeJSONError(w, http.StatusInternalServerError, "User context not found")
		return
	}
	user.CardNumber.String = ""
	writeJSONSuccess(w, "Profile retrieved successfully", user)
}

// GetOrders возвращает список заказов в зависимости от GET-параметра status.
func GetOrders(w http.ResponseWriter, r *http.Request) {
	user, ok := r.Context().Value(UserContextKey).(models.User)
	if !ok {
		writeJSONError(w, http.StatusInternalServerError, "User context not found")
		return
	}

	statusKey := r.URL.Query().Get("status")
	if statusKey == "" {
		statusKey = "active"
	}

	var orders []models.Order
	var err error

	if user.Role == constants.ROLE_USER {
		var userStatusesToFetch []string
		switch statusKey {
		case "new":
			userStatusesToFetch = []string{constants.STATUS_NEW, constants.STATUS_AWAITING_COST}
		case "awaiting_confirmation":
			userStatusesToFetch = []string{constants.STATUS_AWAITING_CONFIRMATION, constants.STATUS_AWAITING_PAYMENT}
		case "in_progress":
			userStatusesToFetch = []string{constants.STATUS_INPROGRESS}
		case "completed":
			userStatusesToFetch = []string{constants.STATUS_COMPLETED, constants.STATUS_CALCULATED, constants.STATUS_SETTLED}
		case "canceled":
			userStatusesToFetch = []string{constants.STATUS_CANCELED}
		case "active":
			fallthrough
		default:
			userStatusesToFetch = []string{
				constants.STATUS_NEW,
				constants.STATUS_AWAITING_CONFIRMATION,
				constants.STATUS_INPROGRESS,
				constants.STATUS_AWAITING_COST,
				constants.STATUS_AWAITING_PAYMENT,
			}
		}
		orders, err = db.GetOrdersByChatIDAndMultipleStatuses(user.ChatID, userStatusesToFetch, 0)
	} else {
		var statusesToFetch []string
		orderByField := "o.created_at"
		orderByDirection := "DESC"

		switch statusKey {
		case "new":
			statusesToFetch = []string{constants.STATUS_NEW, constants.STATUS_AWAITING_COST}
		case "awaiting_confirmation":
			statusesToFetch = []string{constants.STATUS_AWAITING_CONFIRMATION, constants.STATUS_AWAITING_PAYMENT}
		case "in_progress":
			statusesToFetch = []string{constants.STATUS_INPROGRESS}
		case "completed":
			statusesToFetch = []string{constants.STATUS_COMPLETED, constants.STATUS_CALCULATED, constants.STATUS_SETTLED}
			orderByField = "o.updated_at"
		case "canceled":
			statusesToFetch = []string{constants.STATUS_CANCELED}
			orderByField = "o.updated_at"
		case "active":
			fallthrough
		default:
			statusesToFetch = []string{
				constants.STATUS_NEW, constants.STATUS_AWAITING_CONFIRMATION, constants.STATUS_INPROGRESS,
				constants.STATUS_AWAITING_COST, constants.STATUS_AWAITING_PAYMENT,
			}
		}
		orders, err = db.GetOrdersByMultipleStatuses(statusesToFetch, 0, orderByField, orderByDirection)
	}

	if err != nil {
		log.Printf("API GetOrders error for user %d, role %s, status '%s': %v", user.ID, user.Role, statusKey, err)
		writeJSONError(w, http.StatusInternalServerError, "Failed to retrieve orders")
		return
	}

	if orders == nil {
		orders = make([]models.Order, 0)
	}

	writeJSONSuccess(w, "Orders retrieved successfully", orders)
}

// GetClients возвращает список пользователей по роли.
func GetClients(w http.ResponseWriter, r *http.Request) {
	role := r.URL.Query().Get("role")
	if role == "" {
		role = constants.ROLE_USER
	}

	clients, err := db.GetUsersByRole(role)
	if err != nil {
		log.Printf("API GetClients error for role '%s': %v", role, err)
		writeJSONError(w, http.StatusInternalServerError, "Failed to retrieve clients")
		return
	}

	if clients == nil {
		clients = make([]models.User, 0)
	}

	writeJSONSuccess(w, "Clients retrieved successfully", clients)
}

// CreateOrder создает новый заказ, записывая ID оператора в поле user_id.
func CreateOrder(w http.ResponseWriter, r *http.Request) {
	operator, ok := r.Context().Value(UserContextKey).(models.User)
	if !ok {
		writeJSONError(w, http.StatusInternalServerError, "Не удалось определить оператора из контекста")
		return
	}

	var orderData models.Order
	if err := json.NewDecoder(r.Body).Decode(&orderData); err != nil {
		writeJSONError(w, http.StatusBadRequest, "Invalid request body: "+err.Error())
		return
	}

	orderData.UserID = int(operator.ID)
	orderData.UserChatID = operator.ChatID
	orderData.Status = "in_progress"

	newOrderID, err := db.CreateFullOrder(orderData)
	if err != nil {
		log.Printf("API CreateOrder: db.CreateFullOrder вернула ошибку: %v", err)
		writeJSONError(w, http.StatusInternalServerError, "Не удалось создать заказ: "+err.Error())
		return
	}

	writeJSONSuccess(w, "Заказ успешно создан!", map[string]int64{"order_id": newOrderID})
}

// GetClientDetails возвращает подробную информацию о клиенте.
func GetClientDetails(w http.ResponseWriter, r *http.Request) {
	clientIDStr := chi.URLParam(r, "id")
	clientID, err := strconv.ParseInt(clientIDStr, 10, 64)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "Invalid client ID")
		return
	}

	user, err := db.GetUserByID(int(clientID))
	if err != nil {
		log.Printf("API GetClientDetails: ошибка получения пользователя %d: %v", clientID, err)
		writeJSONError(w, http.StatusNotFound, "Client not found")
		return
	}

	orderCount, err := db.GetOrderCountForUser(clientID)
	if err != nil {
		log.Printf("API GetClientDetails: не удалось получить кол-во заказов для %d: %v", clientID, err)
	}

	recentOrders, errOrders := db.GetOrdersByUserID(clientID)
	if errOrders != nil {
		log.Printf("API GetClientDetails: не удалось получить последние заказы для клиента %d: %v", clientID, errOrders)
		recentOrders = []models.Order{}
	}

	if recentOrders == nil {
		recentOrders = make([]models.Order, 0)
	}

	if len(recentOrders) > 5 {
		recentOrders = recentOrders[:5]
	}

	response := ClientDetailsResponse{
		User:       user,
		OrderCount: orderCount,
		Orders:     recentOrders,
	}

	writeJSONSuccess(w, "Client details retrieved successfully", response)
}

// GetOrderDetails возвращает полную информацию о заказе, включая готовые ссылки на медиа.
func GetOrderDetails(w http.ResponseWriter, r *http.Request) {
	requestingUser, ok := r.Context().Value(UserContextKey).(models.User)
	if !ok {
		writeJSONError(w, http.StatusInternalServerError, "User context not found")
		return
	}

	orderIDStr := chi.URLParam(r, "id")
	orderID, err := strconv.Atoi(orderIDStr)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "Invalid order ID")
		return
	}

	order, err := db.GetOrderByID(orderID)
	if err != nil {
		log.Printf("API GetOrderDetails: ошибка получения заказа %d: %v", orderID, err)
		writeJSONError(w, http.StatusNotFound, "Order not found")
		return
	}

	isOperator := utils.IsOperatorOrHigher(requestingUser.Role)
	isOwnerOfOrder := requestingUser.ChatID == order.UserChatID
	if !isOperator && !isOwnerOfOrder {
		log.Printf("API GetOrderDetails: отказано в доступе. User %d, Role %s, Order %d, Owner %d",
			requestingUser.ChatID, requestingUser.Role, order.ID, order.UserChatID)
		writeJSONError(w, http.StatusForbidden, "Access Denied")
		return
	}

	getRelativeLink := func(filename string) string {
		if filename == "" {
			return ""
		}
		return fmt.Sprintf("/api/media/%s", filename)
	}

	relativePhotoLinks := make([]string, 0, len(order.Photos))
	for _, photoFilename := range order.Photos {
		if link := getRelativeLink(photoFilename); link != "" {
			relativePhotoLinks = append(relativePhotoLinks, link)
		}
	}
	order.Photos = relativePhotoLinks

	relativeVideoLinks := make([]string, 0, len(order.Videos))
	for _, videoFilename := range order.Videos {
		if link := getRelativeLink(videoFilename); link != "" {
			relativeVideoLinks = append(relativeVideoLinks, link)
		}
	}
	order.Videos = relativeVideoLinks

	executors, err := db.GetExecutorsByOrderID(orderID)
	if err != nil {
		log.Printf("API GetOrderDetails: ошибка получения исполнителей для заказа %d: %v", orderID, err)
	}

	type OrderResponse struct {
		models.Order
		Executors []models.Executor `json:"executors"`
	}
	response := OrderResponse{
		Order:     order,
		Executors: executors,
	}

	writeJSONSuccess(w, "Order details retrieved successfully", response)
}

// HandleAdminOrderAction - единый обработчик действий над заказом для администраторов/операторов.
func HandleAdminOrderAction(w http.ResponseWriter, r *http.Request) {
	user, ok := r.Context().Value(UserContextKey).(models.User)
	if !ok {
		writeJSONError(w, http.StatusInternalServerError, "User context not found")
		return
	}
	bot, ok := r.Context().Value("bot").(*handlers.BotHandler)
	if !ok {
		writeJSONError(w, http.StatusInternalServerError, "Bot context not found")
		return
	}

	orderIDStr := chi.URLParam(r, "id")
	orderID, err := strconv.Atoi(orderIDStr)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "Invalid order ID")
		return
	}

	var req OrderActionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "Invalid request body: "+err.Error())
		return
	}

	order, err := db.GetOrderByID(orderID)
	if err != nil {
		log.Printf("API HandleAdminOrderAction: Order %d not found. Error: %v", orderID, err)
		writeJSONError(w, http.StatusNotFound, "Order not found")
		return
	}

	sendBotMessage := func(chatID int64, text string) {
		msg := tgbotapi.NewMessage(chatID, text)
		msg.ParseMode = tgbotapi.ModeMarkdown
		if _, err := bot.Deps.BotClient.Send(msg); err != nil {
			log.Printf("Failed to send message to chat %d: %v", chatID, err)
		}
	}

	log.Printf("Admin Action: User '%s' (ID: %d) initiated action '%s' for order %d.", utils.GetUserDisplayName(user), user.ID, req.Action, orderID)

	switch req.Action {
	case "set_cost", "set_final_cost":
		cost, err := req.Cost.Float64()
		if err != nil || cost < 0 {
			writeJSONError(w, http.StatusBadRequest, "Invalid cost value")
			return
		}

		if err := db.UpdateOrderField(int64(orderID), "cost", cost); err != nil {
			log.Printf("API HandleAdminOrderAction: Failed to update cost for order %d. Error: %v", orderID, err)
			writeJSONError(w, http.StatusInternalServerError, "Failed to update cost")
			return
		}

		var newStatus, clientMessage string
		if (order.Status == constants.STATUS_NEW || order.Status == constants.STATUS_AWAITING_COST) && req.Action == "set_cost" {
			newStatus = constants.STATUS_AWAITING_CONFIRMATION
			clientMessage = fmt.Sprintf("💰 Установлена стоимость для вашего заказа №%d: *%.0f ₽*.\n\nПожалуйста, подтвердите или отклоните ее в меню 'Мои заказы'.", orderID, cost)
		} else if order.Status == constants.STATUS_COMPLETED && req.Action == "set_final_cost" {
			newStatus = constants.STATUS_CALCULATED
		}

		if newStatus != "" {
			if err := db.UpdateOrderStatus(int64(orderID), newStatus); err != nil {
				log.Printf("API HandleAdminOrderAction: Failed to update status for order %d. Error: %v", orderID, err)
				writeJSONError(w, http.StatusInternalServerError, "Failed to update order status")
				return
			}
			if clientMessage != "" {
				// --- НАЧАЛО ИЗМЕНЕНИЯ ---
				// Создаем сообщение, к которому прикрепим кнопку
				msg := tgbotapi.NewMessage(order.UserChatID, clientMessage)
				msg.ParseMode = tgbotapi.ModeMarkdown

				// URL вашего Web App. Можно вынести в конфиг.
				webAppURL := "https://xn----ctbinlmxece7i.xn--p1ai/webapp/"

				// Создаем кнопку WebApp
				webAppButton := tgbotapi.NewInlineKeyboardButtonWebApp(
					"🌐 Открыть приложение", // Текст кнопки
					tgbotapi.WebAppInfo{URL: webAppURL},
				)

				// Создаем клавиатуру с этой кнопкой
				keyboard := tgbotapi.NewInlineKeyboardMarkup(
					tgbotapi.NewInlineKeyboardRow(webAppButton),
				)

				// Прикрепляем клавиатуру к сообщению
				msg.ReplyMarkup = keyboard

				// Отправляем комплексное сообщение
				if _, err := bot.Deps.BotClient.Send(msg); err != nil {
					log.Printf("Failed to send message with WebApp button to chat %d: %v", order.UserChatID, err)
				}
				// --- КОНЕЦ ИЗМЕНЕНИЯ ---
			}
		}

		log.Printf("Admin Action: User '%s' (ID: %d) set cost for order %d to %.2f", utils.GetUserDisplayName(user), user.ID, orderID, cost)
		writeJSONSuccess(w, "Стоимость успешно обновлена", nil)

	case "complete":
		if order.Status != constants.STATUS_INPROGRESS {
			writeJSONError(w, http.StatusConflict, "Order is not in 'in_progress' status")
			return
		}
		if err := db.UpdateOrderStatus(int64(orderID), constants.STATUS_COMPLETED); err != nil {
			log.Printf("API HandleAdminOrderAction: Failed to complete order %d. Error: %v", orderID, err)
			writeJSONError(w, http.StatusInternalServerError, "Failed to update order status")
			return
		}

		clientMessage := fmt.Sprintf("✅ Ваш заказ №%d выполнен!", orderID)
		sendBotMessage(order.UserChatID, clientMessage)

		log.Printf("Admin action: User '%s' (ID: %d) marked order %d as completed", utils.GetUserDisplayName(user), user.ID, orderID)
		writeJSONSuccess(w, "Заказ отмечен как выполненный", nil)

	case "cancel":
		if req.Reason == "" {
			writeJSONError(w, http.StatusBadRequest, "Cancellation reason is required")
			return
		}
		cancellationReason := sql.NullString{String: req.Reason, Valid: true}
		if err := db.UpdateOrderStatusAndReason(int64(orderID), constants.STATUS_CANCELED, cancellationReason); err != nil {
			log.Printf("API HandleAdminOrderAction: Failed to cancel order %d. Error: %v", orderID, err)
			writeJSONError(w, http.StatusInternalServerError, "Failed to cancel order")
			return
		}
		clientMessage := fmt.Sprintf("❌ Ваш заказ №%d был отменен оператором.\nПричина: %s", orderID, req.Reason)
		sendBotMessage(order.UserChatID, clientMessage)

		log.Printf("Admin action: User '%s' (ID: %d) canceled order %d. Reason: %s", utils.GetUserDisplayName(user), user.ID, orderID, req.Reason)
		writeJSONSuccess(w, "Заказ успешно отменён", nil)

	case "resume":
		if order.Status != constants.STATUS_CANCELED {
			writeJSONError(w, http.StatusConflict, "Order is not in 'canceled' status")
			return
		}
		if err := db.UpdateOrderStatusAndReason(int64(orderID), constants.STATUS_NEW, sql.NullString{Valid: false}); err != nil {
			log.Printf("API HandleAdminOrderAction: Failed to resume order %d. Error: %v", orderID, err)
			writeJSONError(w, http.StatusInternalServerError, "Failed to resume order")
			return
		}
		clientMessage := fmt.Sprintf("✅ Ваш заказ №%d был возобновлен оператором и снова в работе!", orderID)
		sendBotMessage(order.UserChatID, clientMessage)

		log.Printf("Admin action: User '%s' (ID: %d) resumed order %d", utils.GetUserDisplayName(user), user.ID, orderID)
		writeJSONSuccess(w, "Заказ успешно возобновлён", nil)

	default:
		writeJSONError(w, http.StatusBadRequest, "Unknown action: "+req.Action)
	}
}

// UpdateOrderFieldHandler
func UpdateOrderFieldHandler(w http.ResponseWriter, r *http.Request) {
	user, ok := r.Context().Value(UserContextKey).(models.User)
	if !ok {
		writeJSONError(w, http.StatusInternalServerError, "User context not found")
		return
	}
	if !utils.IsOperatorOrHigher(user.Role) {
		writeJSONError(w, http.StatusForbidden, "Access Denied: Operator role required")
		return
	}
	orderIDStr := chi.URLParam(r, "id")
	orderID, err := strconv.ParseInt(orderIDStr, 10, 64)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "Invalid order ID")
		return
	}
	var req UpdateOrderFieldRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "Invalid request body: "+err.Error())
		return
	}
	if req.Field == "" {
		writeJSONError(w, http.StatusBadRequest, "Field name is required")
		return
	}
	if err := db.UpdateOrderField(orderID, req.Field, req.Value); err != nil {
		writeJSONError(w, http.StatusInternalServerError, "Failed to update order field: "+err.Error())
		return
	}
	log.Printf("Admin Action: User '%s' (ID: %d) updated field '%s' for order %d.", utils.GetUserDisplayName(user), user.ID, req.Field, orderID)
	writeJSONSuccess(w, fmt.Sprintf("Поле '%s' для заказа №%d успешно обновлено.", req.Field, orderID), nil)
}

// AddOrderMedia
func AddOrderMedia(w http.ResponseWriter, r *http.Request) {
	user, ok := r.Context().Value(UserContextKey).(models.User)
	if !ok {
		writeJSONError(w, http.StatusInternalServerError, "User context not found")
		return
	}
	orderIDStr := chi.URLParam(r, "id")
	orderID, err := strconv.ParseInt(orderIDStr, 10, 64)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "Invalid order ID")
		return
	}
	var req AddMediaRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "Invalid request body: "+err.Error())
		return
	}
	order, err := db.GetOrderByID(int(orderID))
	if err != nil {
		writeJSONError(w, http.StatusNotFound, "Order not found")
		return
	}

	isOperator := utils.IsOperatorOrHigher(user.Role)
	isOwner := user.ChatID == order.UserChatID
	isAssignedDriver := false
	executors, err := db.GetExecutorsByOrderID(int(orderID))
	if err == nil {
		for _, exec := range executors {
			if exec.UserID == user.ID && exec.Role == constants.ROLE_DRIVER {
				isAssignedDriver = true
				break
			}
		}
	}

	if !isOperator && !isOwner && !isAssignedDriver {
		log.Printf("AddOrderMedia FORBIDDEN: User '%s' (ID: %d, Role: %s) tried to add media to order %d.",
			utils.GetUserDisplayName(user), user.ID, user.Role, orderID)
		writeJSONError(w, http.StatusForbidden, "Access to this resource is denied")
		return
	}

	photoSet := make(map[string]struct{})
	for _, p := range order.Photos {
		if p != "" {
			photoSet[p] = struct{}{}
		}
	}
	for _, p := range req.Photos {
		photoSet[p] = struct{}{}
	}

	videoSet := make(map[string]struct{})
	for _, v := range order.Videos {
		if v != "" {
			videoSet[v] = struct{}{}
		}
	}
	for _, v := range req.Videos {
		videoSet[v] = struct{}{}
	}

	updatedPhotos := make([]string, 0, len(photoSet))
	for p := range photoSet {
		updatedPhotos = append(updatedPhotos, p)
	}
	updatedVideos := make([]string, 0, len(videoSet))
	for v := range videoSet {
		updatedVideos = append(updatedVideos, v)
	}

	dbPhotoPaths := utils.ExtractFilenamesFromUrls(updatedPhotos)
	dbVideoPaths := utils.ExtractFilenamesFromUrls(updatedVideos)

	if err := db.UpdateOrderPhotosAndVideos(orderID, dbPhotoPaths, dbVideoPaths); err != nil {
		writeJSONError(w, http.StatusInternalServerError, "Failed to add media to order: "+err.Error())
		return
	}

	log.Printf("Media Action: User '%s' (ID: %d) added %d photos and %d videos to order %d.", utils.GetUserDisplayName(user), user.ID, len(req.Photos), len(req.Videos), orderID)
	writeJSONSuccess(w, "Медиа файлы успешно добавлены к заказу.", nil)
}

// HandleDriverOrderAction
func HandleDriverOrderAction(w http.ResponseWriter, r *http.Request) {
	driver, ok := r.Context().Value(UserContextKey).(models.User)
	if !ok {
		writeJSONError(w, http.StatusInternalServerError, "User context not found")
		return
	}
	bot, ok := r.Context().Value("bot").(*handlers.BotHandler)
	if !ok {
		writeJSONError(w, http.StatusInternalServerError, "Bot context not found")
		return
	}

	orderIDStr := chi.URLParam(r, "id")
	orderID, err := strconv.Atoi(orderIDStr)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "Invalid order ID")
		return
	}
	var req OrderActionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	executors, _ := db.GetExecutorsByOrderID(orderID)
	isAssignedDriver := false
	for _, exec := range executors {
		if exec.UserID == driver.ID && exec.Role == constants.ROLE_DRIVER {
			isAssignedDriver = true
			break
		}
	}
	if !isAssignedDriver {
		writeJSONError(w, http.StatusForbidden, "You are not assigned to this order")
		return
	}

	switch req.Action {
	case "complete":
		order, err := db.GetOrderByID(orderID)
		if err != nil {
			writeJSONError(w, http.StatusNotFound, "Order not found")
			return
		}
		if order.Status != constants.STATUS_INPROGRESS {
			writeJSONError(w, http.StatusConflict, "Order is not in 'in_progress' status")
			return
		}
		if err := db.UpdateOrderStatus(int64(orderID), constants.STATUS_COMPLETED); err != nil {
			writeJSONError(w, http.StatusInternalServerError, "Failed to update order status")
			return
		}
		msg := tgbotapi.NewMessage(order.UserChatID, fmt.Sprintf("✅ Ваш заказ №%d выполнен!", orderID))
		bot.Deps.BotClient.Send(msg)

		writeJSONSuccess(w, "Заказ отмечен как выполненный", nil)

	default:
		writeJSONError(w, http.StatusBadRequest, "Unknown action")
	}
}

// HandleUserOrderAction
func HandleUserOrderAction(w http.ResponseWriter, r *http.Request) {
	user, ok := r.Context().Value(UserContextKey).(models.User)
	if !ok {
		writeJSONError(w, http.StatusInternalServerError, "User context not found")
		return
	}
	bot, ok := r.Context().Value("bot").(*handlers.BotHandler)
	if !ok {
		writeJSONError(w, http.StatusInternalServerError, "Bot context not found")
		return
	}
	orderIDStr := chi.URLParam(r, "id")
	orderID, err := strconv.Atoi(orderIDStr)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "Invalid order ID")
		return
	}
	var req OrderActionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	order, err := db.GetOrderByID(orderID)
	if err != nil {
		writeJSONError(w, http.StatusNotFound, "Order not found")
		return
	}
	if order.UserChatID != user.ChatID {
		writeJSONError(w, http.StatusForbidden, "Access denied to this order")
		return
	}

	sendBotMessage := func(chatID int64, text string) {
		msg := tgbotapi.NewMessage(chatID, text)
		_, err := bot.Deps.BotClient.Send(msg)
		if err != nil {
			log.Printf("Failed to send message to chat %d: %v", chatID, err)
		}
	}

	switch req.Action {
	case "accept_cost":
		if order.Status != constants.STATUS_AWAITING_CONFIRMATION {
			writeJSONError(w, http.StatusConflict, "Order is not awaiting cost confirmation")
			return
		}
		var newStatus string
		var clientMessageText string
		if order.Payment == "now" {
			newStatus = constants.STATUS_AWAITING_PAYMENT
			clientMessageText = fmt.Sprintf("✅ Стоимость по заказу №%d подтверждена. Теперь вы можете оплатить его в меню 'Мои заказы'.", orderID)
		} else {
			newStatus = constants.STATUS_INPROGRESS
			clientMessageText = fmt.Sprintf("✅ Стоимость по заказу №%d принята. Скоро с вами свяжутся для уточнения деталей.", orderID)
		}

		db.UpdateOrderStatus(int64(orderID), newStatus)
		sendBotMessage(order.UserChatID, clientMessageText)

		writeJSONSuccess(w, "Стоимость принята", nil)

	case "reject_cost":
		if order.Status != constants.STATUS_AWAITING_CONFIRMATION {
			writeJSONError(w, http.StatusConflict, "Order is not awaiting cost confirmation")
			return
		}
		if req.Reason == "" {
			writeJSONError(w, http.StatusBadRequest, "Rejection reason is required")
			return
		}

		db.UpdateOrderStatusAndReason(int64(orderID), constants.STATUS_CANCELED, sql.NullString{String: "Клиент отклонил стоимость: " + req.Reason, Valid: true})
		writeJSONSuccess(w, "Стоимость отклонена, заказ отменён", nil)

	case "cancel_by_user":
		canCancel := order.Status == constants.STATUS_DRAFT || order.Status == constants.STATUS_NEW || order.Status == constants.STATUS_AWAITING_COST
		if !canCancel {
			writeJSONError(w, http.StatusConflict, "This order cannot be canceled at this stage.")
			return
		}
		if req.Reason == "" {
			writeJSONError(w, http.StatusBadRequest, "Cancellation reason is required")
			return
		}

		db.UpdateOrderStatusAndReason(int64(orderID), constants.STATUS_CANCELED, sql.NullString{String: "Отменено клиентом: " + req.Reason, Valid: true})
		writeJSONSuccess(w, "Заказ отменён", nil)

	default:
		writeJSONError(w, http.StatusBadRequest, "Unknown action")
	}
}

// UpdateSettlementStatus
func UpdateSettlementStatus(w http.ResponseWriter, r *http.Request) {
	user, ok := r.Context().Value(UserContextKey).(models.User)
	if !ok {
		writeJSONError(w, http.StatusInternalServerError, "User context not found")
		return
	}
	bot, ok := r.Context().Value("bot").(*handlers.BotHandler)
	if !ok {
		writeJSONError(w, http.StatusInternalServerError, "Bot context not found")
		return
	}
	settlementIDStr := chi.URLParam(r, "id")
	settlementID, err := strconv.ParseInt(settlementIDStr, 10, 64)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "Invalid settlement ID")
		return
	}

	var req SettlementStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	settlement, err := db.GetDriverSettlementByID(settlementID)
	if err != nil {
		writeJSONError(w, http.StatusNotFound, "Settlement not found")
		return
	}
	if settlement.Status != constants.SETTLEMENT_STATUS_PENDING {
		writeJSONError(w, http.StatusConflict, "Settlement is not in pending state")
		return
	}

	var newStatus string
	var comment sql.NullString
	var driverMessageText string

	if req.Status == "approved" {
		newStatus = constants.SETTLEMENT_STATUS_APPROVED
		driverMessageText = fmt.Sprintf("✅ Ваш отчет #%d был утвержден оператором %s.", settlement.ID, utils.GetUserDisplayName(user))
	} else if req.Status == "rejected" {
		if req.Reason == "" {
			writeJSONError(w, http.StatusBadRequest, "Rejection reason is required")
			return
		}
		newStatus = constants.SETTLEMENT_STATUS_REJECTED
		comment = sql.NullString{String: req.Reason, Valid: true}
		driverMessageText = fmt.Sprintf("❌ Ваш отчет #%d был отклонен оператором %s.\nПричина: %s\n\nПожалуйста, создайте новый, исправленный отчет.",
			settlement.ID, utils.GetUserDisplayName(user), req.Reason)
	} else {
		writeJSONError(w, http.StatusBadRequest, "Invalid status")
		return
	}

	if err := db.UpdateDriverSettlementStatus(settlementID, newStatus, comment); err != nil {
		writeJSONError(w, http.StatusInternalServerError, "Failed to update settlement status")
		return
	}

	driver, err := db.GetUserByID(int(settlement.DriverUserID))
	if err == nil {
		msg := tgbotapi.NewMessage(driver.ChatID, driverMessageText)
		bot.Deps.BotClient.Send(msg)
	}

	writeJSONSuccess(w, "Статус отчёта обновлён", nil)
}

// GetClientConfig
func GetClientConfig(w http.ResponseWriter, r *http.Request) {
	cfg, ok := r.Context().Value("config").(*config.Config)
	if !ok {
		writeJSONError(w, http.StatusInternalServerError, "Config not found in context")
		return
	}

	response := map[string]string{
		"telegramBotUsername": cfg.BotUsername,
	}

	writeJSONSuccess(w, "Config retrieved", response)
}

// StartDriverReport
func StartDriverReport(w http.ResponseWriter, r *http.Request) {
	driver, ok := r.Context().Value(UserContextKey).(models.User)
	if !ok {
		writeJSONError(w, http.StatusInternalServerError, "User context not found")
		return
	}

	unsettledOrders, err := db.GetUnsettledCompletedOrdersForDriver(driver.ID)
	if err != nil {
		log.Printf("API StartDriverReport: failed to get unsettled orders for driver %d: %v", driver.ID, err)
		writeJSONError(w, http.StatusInternalServerError, "Failed to get unsettled orders")
		return
	}

	if len(unsettledOrders) == 0 {
		writeJSONSuccess(w, "Нет нерассчитанных заказов.", map[string]int{"unsettled_orders_count": 0})
		return
	}

	writeJSONSuccess(w, "Отчет по заказам инициирован.", map[string]int{"unsettled_orders_count": len(unsettledOrders)})
}
func CreateUserOrder(w http.ResponseWriter, r *http.Request) {
	// Шаг 1: Получаем модель пользователя, который делает запрос, и экземпляр бота.
	user, ok := r.Context().Value(UserContextKey).(models.User)
	if !ok {
		writeJSONError(w, http.StatusInternalServerError, "Не удалось определить пользователя из контекста")
		return
	}
	bot, botOk := r.Context().Value("bot").(*handlers.BotHandler)
	if !botOk {
		log.Printf("CRITICAL: Bot context not found in CreateUserOrder. Cannot send notification.")
	}

	// Шаг 2: Читаем данные заказа из тела запроса.
	var orderData models.Order
	if err := json.NewDecoder(r.Body).Decode(&orderData); err != nil {
		writeJSONError(w, http.StatusBadRequest, "Invalid request body: "+err.Error())
		return
	}

	// Шаг 3: Принудительно устанавливаем ID и ChatID пользователя, создающего заказ.
	// Это гарантирует, что заказ будет привязан к правильному пользователю.
	orderData.UserID = int(user.ID)
	orderData.UserChatID = user.ChatID

	// Шаг 4: Устанавливаем статус "новый", так как заказ от пользователя требует оценки.
	orderData.Status = constants.STATUS_NEW

	// Шаг 5: Вызываем функцию для создания заказа в БД.
	newOrderID, err := db.CreateFullOrder(orderData)
	if err != nil {
		log.Printf("API CreateUserOrder: db.CreateFullOrder вернула ошибку: %v", err)
		writeJSONError(w, http.StatusInternalServerError, "Не удалось создать заказ: "+err.Error())
		return
	}

	// Шаг 6: Если бот доступен, вызываем функцию для отправки уведомлений операторам.
	// Делаем это в отдельной горутине, чтобы не замедлять ответ API.
	if botOk {
		go bot.NotifyOperatorsAboutNewOrder(newOrderID, user.ChatID)
	}

	// Шаг 7: Отправляем успешный ответ.
	writeJSONSuccess(w, "Заказ успешно создан!", map[string]int64{"order_id": newOrderID})
}
