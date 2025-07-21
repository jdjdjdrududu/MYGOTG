// Файл: internal/session/temp_order.go
// Полное содержимое файла:
package session

import (
	"Original/internal/models" // Используем Original как имя модуля / Use Original as module name
)

// TempOrderData представляет собой временное состояние заказа или другого многошагового процесса в сессии пользователя.
// Она встраивает models.Order (для данных заказа) и добавляет поля, специфичные для сессии.
// TempOrderData represents the temporary state of an order or other multi-step process in a user's session.
// It embeds models.Order (for order data) and adds session-specific fields.
type TempOrderData struct {
	models.Order
	BlockReason               string
	OrderAction               string
	EphemeralMediaMessageIDs  []int
	SelectedHourForMinuteView int
	ActiveMediaGroupID        string // <--- НОВОЕ ПОЛЕ для отслеживания активного альбома
	// Если у вас уже есть мьютекс для других полей TempOrderData, он может также защищать ActiveMediaGroupID.
	// Если нет, и если TempOrderData напрямую модифицируется из разных горутин (что маловероятно, если SessionManager используется правильно),
	// то мьютекс может понадобиться. В данном случае SessionManager синхронизирует доступ к TempOrderData.
	// mediaMutex sync.Mutex // Пример мьютекса, если бы он был нужен внутри TempOrderData
}

// NewTempOrder создает новый экземпляр TempOrderData для указанного chatID.
// UserChatID в TempOrderData.Order будет установлен в chatID.
// NewTempOrder creates a new TempOrderData instance for the specified chatID.
// UserChatID in TempOrderData.Order will be set to chatID.
func NewTempOrder(chatID int64) TempOrderData {
	return TempOrderData{
		Order: models.Order{
			UserChatID:         chatID,
			MediaMessageIDs:    make([]int, 0),
			MediaMessageIDsMap: make(map[string]bool),
		},
		EphemeralMediaMessageIDs:  make([]int, 0),
		SelectedHourForMinuteView: -1,
		ActiveMediaGroupID:        "", // <--- ИНИЦИАЛИЗАЦИЯ
	}
}
