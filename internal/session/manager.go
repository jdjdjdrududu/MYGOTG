package session

import (
	"Original/internal/constants" // Используем Original как имя модуля
	"fmt"
	"log"
	"sync"
	// "Original/internal/models" // TempOrderData уже импортирует models.Order
)

// SessionManager управляет состояниями пользователей и временными данными заказов.
// SessionManager manages user states and temporary order data.
type SessionManager struct {
	userStates     map[int64]string   // Ключ: chatID, Значение: текущее состояние (например, constants.STATE_ORDER_NAME) / Key: chatID, Value: current state (e.g., constants.STATE_ORDER_NAME)
	userStateMutex sync.RWMutex       // Мьютекс для безопасного доступа к userStates и userHistory / Mutex for safe access to userStates and userHistory
	userHistory    map[int64][]string // Ключ: chatID, Значение: слайс строк состояний (история) / Key: chatID, Value: slice of state strings (history)

	tempOrders      map[int64]TempOrderData // Ключ: chatID пользователя, который инициирует сессию / Key: chatID of the user initiating the session
	tempOrdersMutex sync.RWMutex            // Мьютекс для безопасного доступа к tempOrders / Mutex for safe access to tempOrders

	// Кэш для отслеживания сообщений, которые были удалены или для которых была предпринята попытка удаления.
	// Это помогает избежать повторных запросов к API Telegram для удаления уже удаленных сообщений.
	// Cache for tracking messages that were deleted or for which a deletion attempt was made.
	// This helps avoid repeated Telegram API requests to delete already deleted messages.
	deletedMessages      map[int64]map[int]bool  // Ключ1: chatID, Ключ2: messageID, Значение: true если помечено как удаленное / Key1: chatID, Key2: messageID, Value: true if marked as deleted
	deletedMessagesMutex map[int64]*sync.RWMutex // Карта мьютексов, по одному на каждого пользователя для его карты deletedMessages / Map of mutexes, one per user for their deletedMessages map
	// Общий мьютекс для доступа к карте deletedMessagesMutex (для ее инициализации)
	// General mutex for accessing the deletedMessagesMutex map (for its initialization)
	deletedMessagesMapMutex sync.Mutex

	tempDriverSettlements      map[int64]TempDriverSettlementData // Ключ: chatID водителя
	tempDriverSettlementsMutex sync.RWMutex
}

// NewSessionManager создает и возвращает новый экземпляр SessionManager.
// NewSessionManager creates and returns a new instance of SessionManager.
func NewSessionManager() *SessionManager {
	return &SessionManager{
		userStates:            make(map[int64]string),
		userHistory:           make(map[int64][]string),
		tempOrders:            make(map[int64]TempOrderData),
		deletedMessages:       make(map[int64]map[int]bool),
		deletedMessagesMutex:  make(map[int64]*sync.RWMutex),
		tempDriverSettlements: make(map[int64]TempDriverSettlementData),
	}
}

// --- Управление состоянием пользователя (User State) ---
// --- User State Management ---

// GetState возвращает текущее состояние пользователя.
// Если состояние для пользователя не установлено, возвращает STATE_IDLE.
// GetState returns the current user state.
// If no state is set for the user, returns STATE_IDLE.
func (sm *SessionManager) GetState(chatID int64) string {
	sm.userStateMutex.RLock()
	defer sm.userStateMutex.RUnlock()
	state, ok := sm.userStates[chatID]
	if !ok {
		// log.Printf("SessionManager.GetState: Состояние для chatID %d не найдено, возвращено IDLE.", chatID) // Закомментировано для уменьшения спама в логах / Commented out to reduce log spam
		return constants.STATE_IDLE // Состояние по умолчанию / Default state
	}
	// log.Printf("SessionManager.GetState: Состояние для chatID %d: %s.", chatID, state) // Закомментировано / Commented out
	return state
}

// SetState устанавливает новое состояние для пользователя и добавляет его в историю.
// SetState sets a new state for the user and adds it to history.
func (sm *SessionManager) SetState(chatID int64, state string) {
	sm.userStateMutex.Lock()
	defer sm.userStateMutex.Unlock()

	sm.userStates[chatID] = state
	if _, exists := sm.userHistory[chatID]; !exists {
		sm.userHistory[chatID] = []string{}
	}
	// Предотвращаем дублирование последнего состояния в истории, если оно такое же
	// Prevent duplication of the last state in history if it's the same
	historyLen := len(sm.userHistory[chatID])
	if historyLen == 0 || sm.userHistory[chatID][historyLen-1] != state {
		sm.userHistory[chatID] = append(sm.userHistory[chatID], state)
	}
	log.Printf("SessionManager.SetState: Состояние для chatID %d установлено: %s, история: %v", chatID, state, sm.userHistory[chatID])
}

// PopState удаляет последнее состояние из истории и устанавливает предыдущее как текущее.
// Возвращает новое текущее состояние. Если история пуста или содержит одно состояние, устанавливает STATE_IDLE.
// PopState removes the last state from history and sets the previous one as current.
// Returns the new current state. If history is empty or contains one state, sets STATE_IDLE.
func (sm *SessionManager) PopState(chatID int64) string {
	sm.userStateMutex.Lock()
	defer sm.userStateMutex.Unlock()

	history, ok := sm.userHistory[chatID]
	if ok && len(history) > 1 {
		// Удаляем последнее состояние из истории / Remove last state from history
		sm.userHistory[chatID] = history[:len(history)-1]
		// Новое текущее состояние - это теперь последнее в урезанной истории / New current state is now the last in the truncated history
		newState := sm.userHistory[chatID][len(sm.userHistory[chatID])-1]
		sm.userStates[chatID] = newState
		log.Printf("SessionManager.PopState: Для chatID %d новое состояние: %s, история: %v", chatID, newState, sm.userHistory[chatID])
		return newState
	}

	// Если история пуста или содержит только одно состояние (обычно IDLE), возвращаем IDLE
	// If history is empty or contains only one state (usually IDLE), return IDLE
	sm.userStates[chatID] = constants.STATE_IDLE
	sm.userHistory[chatID] = []string{constants.STATE_IDLE} // Сбрасываем историю к IDLE / Reset history to IDLE
	log.Printf("SessionManager.PopState: Для chatID %d история пуста или содержит одно состояние, установлено: %s", chatID, constants.STATE_IDLE)
	return constants.STATE_IDLE
}

// GetHistory возвращает копию истории состояний пользователя.
// Используется в основном для отладки или сложной логики навигации.
// GetHistory returns a copy of the user's state history.
// Mainly used for debugging or complex navigation logic.
func (sm *SessionManager) GetHistory(chatID int64) []string {
	sm.userStateMutex.RLock()
	defer sm.userStateMutex.RUnlock()
	if history, ok := sm.userHistory[chatID]; ok {
		// Возвращаем копию, чтобы избежать модификации оригинального слайса извне
		// Return a copy to avoid modifying the original slice from outside
		historyCopy := make([]string, len(history))
		copy(historyCopy, history)
		return historyCopy
	}
	return []string{} // Возвращаем пустой слайс, если истории нет / Return empty slice if no history
}

// ClearState сбрасывает состояние пользователя к STATE_IDLE и очищает его историю состояний.
// ClearState resets the user's state to STATE_IDLE and clears their state history.
func (sm *SessionManager) ClearState(chatID int64) {
	sm.userStateMutex.Lock()
	defer sm.userStateMutex.Unlock()
	sm.userStates[chatID] = constants.STATE_IDLE
	sm.userHistory[chatID] = []string{constants.STATE_IDLE} // Инициализируем историю с IDLE / Initialize history with IDLE
	log.Printf("SessionManager.ClearState: Состояние и история для chatID %d очищены (установлено в IDLE).", chatID)
}

// --- Управление временными заказами (Temp Orders) ---
// --- Temp Orders Management ---

// GetTempOrder возвращает временные данные заказа для пользователя (по chatID инициатора сессии).
// Если для данного chatID временного заказа нет, создает новый пустой экземпляр TempOrderData.
// GetTempOrder returns temporary order data for the user (by session initiator's chatID).
// If no temporary order exists for this chatID, creates a new empty TempOrderData instance.
func (sm *SessionManager) GetTempOrder(chatID int64) TempOrderData {
	sm.tempOrdersMutex.RLock()
	order, exists := sm.tempOrders[chatID]
	sm.tempOrdersMutex.RUnlock()

	if !exists {
		// log.Printf("SessionManager.GetTempOrder: Временный заказ для chatID %d не найден, создается новый.", chatID) // Закомментировано / Commented out
		// NewTempOrder устанавливает UserChatID равным chatID, переданному в NewTempOrder.
		// NewTempOrder sets UserChatID equal to the chatID passed to NewTempOrder.
		newOrder := NewTempOrder(chatID) // UserChatID будет chatID по умолчанию / UserChatID will be chatID by default
		sm.tempOrdersMutex.Lock()
		sm.tempOrders[chatID] = newOrder
		sm.tempOrdersMutex.Unlock()
		return newOrder
	}
	// log.Printf("SessionManager.GetTempOrder: Временный заказ для chatID %d найден. CurrentMessageID: %d", chatID, order.CurrentMessageID) // Закомментировано / Commented out
	return order
}

// UpdateTempOrder обновляет временные данные заказа для пользователя.
// UpdateTempOrder updates temporary order data for the user.
func (sm *SessionManager) UpdateTempOrder(chatID int64, orderData TempOrderData) {
	sm.tempOrdersMutex.Lock()
	defer sm.tempOrdersMutex.Unlock()
	sm.tempOrders[chatID] = orderData
	// log.Printf("SessionManager.UpdateTempOrder: Временный заказ для chatID %d обновлен. CurrentMessageID: %d, Photos: %d, UserChatID в заказе: %d", chatID, orderData.CurrentMessageID, len(orderData.Photos), orderData.UserChatID) // Закомментировано / Commented out
}

// ClearTempOrder удаляет временные данные заказа для пользователя.
// ClearTempOrder deletes temporary order data for the user.
func (sm *SessionManager) ClearTempOrder(chatID int64) {
	sm.tempOrdersMutex.Lock()
	defer sm.tempOrdersMutex.Unlock()
	delete(sm.tempOrders, chatID)
	log.Printf("SessionManager.ClearTempOrder: Временный заказ для chatID %d удален.", chatID)
}

// --- Управление кэшем удаленных сообщений (Deleted Messages Cache) ---
// --- Deleted Messages Cache Management ---

// getDeletedMessagesMutexForChat получает или создает мьютекс для карты удаленных сообщений конкретного пользователя.
// getDeletedMessagesMutexForChat gets or creates a mutex for a specific user's deleted messages map.
func (sm *SessionManager) getDeletedMessagesMutexForChat(chatID int64) *sync.RWMutex {
	sm.deletedMessagesMapMutex.Lock() // Блокируем доступ к карте мьютексов / Lock access to mutex map
	defer sm.deletedMessagesMapMutex.Unlock()

	userDelMutex, exists := sm.deletedMessagesMutex[chatID]
	if !exists {
		userDelMutex = &sync.RWMutex{}
		sm.deletedMessagesMutex[chatID] = userDelMutex
		// Инициализируем и саму карту deletedMessages для этого chatID, если ее нет
		// Initialize the deletedMessages map itself for this chatID if it doesn't exist
		if _, mapExists := sm.deletedMessages[chatID]; !mapExists {
			sm.deletedMessages[chatID] = make(map[int]bool)
		}
	}
	return userDelMutex
}

// MarkMessageAsDeleted помечает сообщение как удаленное (или что была предпринята попытка удаления).
// MarkMessageAsDeleted marks a message as deleted (or that a deletion attempt was made).
func (sm *SessionManager) MarkMessageAsDeleted(chatID int64, messageID int) {
	if messageID == 0 {
		return
	}
	userDelMutex := sm.getDeletedMessagesMutexForChat(chatID)

	userDelMutex.Lock()
	defer userDelMutex.Unlock()
	sm.deletedMessages[chatID][messageID] = true
	// log.Printf("SessionManager.MarkMessageAsDeleted: Сообщение %d для chatID %d помечено как удаленное.", messageID, chatID) // Закомментировано / Commented out
}

// IsMessageDeleted проверяет, было ли сообщение помечено как удаленное.
// IsMessageDeleted checks if a message has been marked as deleted.
func (sm *SessionManager) IsMessageDeleted(chatID int64, messageID int) bool {
	if messageID == 0 {
		return false
	}
	userDelMutex := sm.getDeletedMessagesMutexForChat(chatID)

	userDelMutex.RLock()
	defer userDelMutex.RUnlock()

	userMessages, mapExists := sm.deletedMessages[chatID]
	if !mapExists {
		return false // Если карты для пользователя нет, сообщение точно не помечено / If no map for user, message is definitely not marked
	}
	isDeleted := userMessages[messageID]
	// if isDeleted { // Закомментировано / Commented out
	// 	log.Printf("SessionManager.IsMessageDeleted: Сообщение %d для chatID %d ранее было помечено как удаленное.", messageID, chatID)
	// }
	return isDeleted
}

// ClearDeletedMessagesCacheForChat очищает кэш удаленных сообщений для конкретного чата.
// Это может быть полезно, например, при старте новой сессии или по команде /start.
// ClearDeletedMessagesCacheForChat clears the deleted messages cache for a specific chat.
// This can be useful, for example, when starting a new session or on /start command.
func (sm *SessionManager) ClearDeletedMessagesCacheForChat(chatID int64) {
	userDelMutex := sm.getDeletedMessagesMutexForChat(chatID)

	userDelMutex.Lock()
	defer userDelMutex.Unlock()
	sm.deletedMessages[chatID] = make(map[int]bool) // Пересоздаем карту для пользователя / Recreate map for user
	log.Printf("SessionManager.ClearDeletedMessagesCacheForChat: Кэш удаленных сообщений для chatID %d очищен.", chatID)
}

// --- Управление временными отчетами водителей (Temp Driver Settlements) ---

// GetTempDriverSettlement возвращает временные данные отчета водителя.
func (sm *SessionManager) GetTempDriverSettlement(chatID int64) TempDriverSettlementData {
	sm.tempDriverSettlementsMutex.RLock()
	reportData, exists := sm.tempDriverSettlements[chatID]
	sm.tempDriverSettlementsMutex.RUnlock()

	if !exists {
		newReportData := NewTempDriverSettlement()
		sm.tempDriverSettlementsMutex.Lock()
		sm.tempDriverSettlements[chatID] = newReportData
		sm.tempDriverSettlementsMutex.Unlock()
		return newReportData
	}
	return reportData
}

// UpdateTempDriverSettlement обновляет временные данные отчета водителя.
func (sm *SessionManager) UpdateTempDriverSettlement(chatID int64, reportData TempDriverSettlementData) {
	sm.tempDriverSettlementsMutex.Lock()
	defer sm.tempDriverSettlementsMutex.Unlock()
	sm.tempDriverSettlements[chatID] = reportData
}

// ClearTempDriverSettlement удаляет временные данные отчета водителя.
func (sm *SessionManager) ClearTempDriverSettlement(chatID int64) {
	sm.tempDriverSettlementsMutex.Lock()
	defer sm.tempDriverSettlementsMutex.Unlock()
	delete(sm.tempDriverSettlements, chatID)
	log.Printf("SessionManager.ClearTempDriverSettlement: Временный отчет для chatID %d удален.", chatID)
}

// AddMediaToTempOrder атомарно добавляет FileID фото или видео во временный заказ.
// Возвращает обновленное количество фото, видео и ошибку, если достигнут лимит или медиа уже существует.
func (sm *SessionManager) AddMediaToTempOrder(chatID int64, fileID string, mediaType string, mediaGroupID string, isInPhotoUploadState bool) (photoCount int, videoCount int, err error) {
	sm.tempOrdersMutex.Lock()
	defer sm.tempOrdersMutex.Unlock()

	orderData, exists := sm.tempOrders[chatID]
	if !exists {
		if mediaGroupID != "" && !isInPhotoUploadState {
			log.Printf("AddMediaToTempOrder: tempOrder не существует для chatID %d (альбом '%s', состояние не фото). Медиа '%s' не добавлено.", chatID, mediaGroupID, fileID)
			return 0, 0, fmt.Errorf("сессия заказа не активна для добавления фото из альбома")
		}
		orderData = NewTempOrder(chatID)
		log.Printf("AddMediaToTempOrder: Временный заказ для chatID %d не найден, создан новый.", chatID)
	}

	// Логика управления ActiveMediaGroupID
	if mediaGroupID != "" { // Это элемент альбома
		if orderData.ActiveMediaGroupID == "" && isInPhotoUploadState {
			// Первый элемент нового альбома, пользователь в состоянии загрузки фото
			orderData.ActiveMediaGroupID = mediaGroupID
			// Очищаем предыдущие фото/видео, так как начался новый альбом
			log.Printf("AddMediaToTempOrder: Новый альбом '%s' начат для chatID %d. Очистка предыдущих фото/видео из сессии.", mediaGroupID, chatID)
			orderData.Photos = []string{}
			orderData.Videos = []string{}
		} else if orderData.ActiveMediaGroupID != "" && orderData.ActiveMediaGroupID != mediaGroupID {
			// Пришел элемент из ДРУГОГО альбома
			if isInPhotoUploadState {
				// Пользователь все еще в состоянии загрузки, но начал слать другой альбом. Начинаем заново.
				log.Printf("AddMediaToTempOrder: Обнаружен новый MediaGroupID '%s' (старый '%s') для chatID %d в состоянии фото. Очистка предыдущих фото и установка нового активного альбома.", mediaGroupID, orderData.ActiveMediaGroupID, chatID)
				orderData.Photos = []string{}
				orderData.Videos = []string{}
				orderData.ActiveMediaGroupID = mediaGroupID
			} else {
				// Пользователь уже не в состоянии загрузки, и это файл из другого альбома (не тот, что был активен). Игнорируем.
				log.Printf("AddMediaToTempOrder: Медиа '%s' из альбома '%s' пришло, когда активен альбом '%s' и состояние не фото. Пропуск.", fileID, mediaGroupID, orderData.ActiveMediaGroupID)
				return len(orderData.Photos), len(orderData.Videos), fmt.Errorf("медиа из другого альбома или сессия фото завершена")
			}
		} else if orderData.ActiveMediaGroupID != "" && orderData.ActiveMediaGroupID == mediaGroupID {
			// Это очередной элемент текущего активного альбома. Ничего дополнительно с ActiveMediaGroupID делать не нужно.
		} else if orderData.ActiveMediaGroupID == "" && mediaGroupID != "" && !isInPhotoUploadState {
			// Элемент альбома, но в сессии нет активного альбома, и пользователь не в состоянии загрузки. Поздний/случайный файл.
			log.Printf("AddMediaToTempOrder: Медиа '%s' из альбома '%s' пришло, когда нет активного альбома и состояние не фото. Пропуск.", fileID, mediaGroupID)
			return len(orderData.Photos), len(orderData.Videos), fmt.Errorf("сессия фото не активна для этого альбома")
		}
	} else { // Это одиночное медиа (не часть альбома)
		// Если пользователь был в процессе загрузки альбома (ActiveMediaGroupID установлен),
		// а потом прислал одиночное фото, это может означать, что он закончил с альбомом.
		// Однако, явное завершение альбома лучше делать по кнопке "Готово".
		// Если просто сбросить ActiveMediaGroupID здесь, то если он отправит еще одно фото из того же альбома,
		// оно будет считаться началом нового.
		// Пока оставляем ActiveMediaGroupID без изменений при одиночных фото.
		// Он будет сброшен кнопками "Готово", "Пропустить", "Сбросить", "Назад".
	}

	// Добавление медиа в срезы
	if mediaType == "photo" {
		// Проверка на дубликат
		for _, pID := range orderData.Photos {
			if pID == fileID {
				return len(orderData.Photos), len(orderData.Videos), fmt.Errorf("фото %s уже добавлено", fileID)
			}
		}
		// Проверка лимита
		if len(orderData.Photos) >= constants.MAX_PHOTOS {
			return len(orderData.Photos), len(orderData.Videos), fmt.Errorf("достигнут лимит фото (%d)", constants.MAX_PHOTOS)
		}
		orderData.Photos = append(orderData.Photos, fileID)
	} else if mediaType == "video" {
		// Проверка на дубликат
		for _, vID := range orderData.Videos {
			if vID == fileID {
				return len(orderData.Photos), len(orderData.Videos), fmt.Errorf("видео %s уже добавлено", fileID)
			}
		}
		// Проверка лимита
		if len(orderData.Videos) >= constants.MAX_VIDEOS {
			return len(orderData.Photos), len(orderData.Videos), fmt.Errorf("достигнут лимит видео (%d)", constants.MAX_VIDEOS)
		}
		orderData.Videos = append(orderData.Videos, fileID)
	} else {
		return len(orderData.Photos), len(orderData.Videos), fmt.Errorf("неизвестный тип медиа: %s", mediaType)
	}

	sm.tempOrders[chatID] = orderData // Записываем измененную структуру обратно в карту
	log.Printf("AddMediaToTempOrder: Медиа '%s' (тип: %s, группа: '%s') добавлено для chatID %d. Фото: %d, Видео: %d. ActiveMediaGroupID: '%s'", fileID, mediaType, mediaGroupID, chatID, len(orderData.Photos), len(orderData.Videos), orderData.ActiveMediaGroupID)
	return len(orderData.Photos), len(orderData.Videos), nil
}
