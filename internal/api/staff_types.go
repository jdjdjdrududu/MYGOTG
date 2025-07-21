package api

// Этот файл будет содержать структуры запросов и ответов для API штата.

// StaffListResponse структура для ответа со списком сотрудников
type StaffListResponse struct {
	Status  string        `json:"status"`
	Message string        `json:"message"`
	Data    StaffListData `json:"data"`
}

// StaffListData содержит список пользователей и общее количество
type StaffListData struct {
	Users []StaffMember `json:"users"`
	Total int           `json:"total"`
}

// StaffMember упрощенная структура пользователя для списка
type StaffMember struct {
	ID        int64  `json:"id"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Nickname  string `json:"nickname,omitempty"`
	Role      string `json:"role"`
	IsBlocked bool   `json:"is_blocked"`
}

// StaffMemberDetailsResponse структура для ответа с деталями сотрудника
type StaffMemberDetailsResponse struct {
	Status  string                 `json:"status"`
	Message string                 `json:"message"`
	Data    StaffMemberFullDetails `json:"data"`
}

// StaffMemberFullDetails полная структура пользователя для деталей
type StaffMemberFullDetails struct {
	ID          int64  `json:"id"`
	ChatID      int64  `json:"chat_id"`
	FirstName   string `json:"first_name"`
	LastName    string `json:"last_name"`
	Nickname    string `json:"nickname,omitempty"`
	Phone       string `json:"phone,omitempty"`
	CardNumber  string `json:"card_number,omitempty"` // Будет дешифрован и очищен для клиента при необходимости
	Role        string `json:"role"`
	IsBlocked   bool   `json:"is_blocked"`
	BlockReason string `json:"block_reason,omitempty"`
	BlockDate   string `json:"block_date,omitempty"` // В формате ISO 8601
}

// UpdateRoleRequest структура для запроса изменения роли
type UpdateRoleRequest struct {
	UserID  int64  `json:"user_id"`
	NewRole string `json:"new_role"`
}

// ToggleBlockStatusRequest структура для запроса блокировки/разблокировки
type ToggleBlockStatusRequest struct {
	UserID    int64  `json:"user_id"`
	IsBlocked bool   `json:"is_blocked"`
	Reason    string `json:"reason,omitempty"`
}

// SearchStaffRequest структура для запроса поиска сотрудников
type SearchStaffRequest struct {
	SearchTerm string `json:"search_term"`
	Limit      int    `json:"limit"`
	Offset     int    `json:"offset"`
}
