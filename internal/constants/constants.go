package constants

import (
	"fmt"
	"time"
)

// Order Creation and Management States
// Состояния создания и управления заказами
const (
	STATE_IDLE                   = "idle"
	STATE_ORDER_CATEGORY         = "order_category"
	STATE_ORDER_SUBCATEGORY      = "order_subcategory"
	STATE_ORDER_DESCRIPTION      = "order_description"
	STATE_ORDER_NAME             = "order_name"
	STATE_ORDER_DATE             = "order_date"
	STATE_ORDER_TIME             = "order_time"
	STATE_ORDER_MINUTE_SELECTION = "order_minute_selection"
	STATE_ORDER_PHONE            = "order_phone"
	STATE_ORDER_ADDRESS          = "order_address"
	STATE_ORDER_ADDRESS_LOCATION = "order_address_location"
	STATE_ORDER_ADDRESS_CONFIRM  = "order_address_confirm"
	STATE_ORDER_VOLUME           = "order_volume"  // Может быть не используется, но есть в списке
	STATE_ORDER_TONNAGE          = "order_tonnage" // Может быть не используется, но есть в списке
	STATE_ORDER_PHOTO            = "order_photo"
	STATE_ORDER_PAYMENT          = "order_payment"
	STATE_ORDER_CONFIRM          = "order_confirm" // Общее состояние подтверждения, может быть адаптировано
	STATE_ORDER_EDIT             = "order_edit"
	STATE_ORDER_NAME_CONFIRM     = "order_name_confirm" // Кажется, это для подтверждения конкретного поля, не всего заказа
	STATE_OPERATOR_SELECT_CLIENT = "operator_select_client"

	STATE_DRIVER_CREATE_ORDER_FLOW = "driver_create_order_flow" // Новое состояние для создания заказа водителем

	// States for operator's extended order creation flow
	// Состояния для расширенного потока создания заказа оператором
	STATE_OP_CREATE_ORDER_FLOW          = "op_create_order_flow"          // Общее состояние потока создания заказа оператором
	STATE_OP_ORDER_CONFIRMATION_OPTIONS = "op_order_confirmation_options" // Меню с опциями после создания чернового заказа оператором
	STATE_OP_ORDER_COST_INPUT           = "op_order_cost_input"           // Ввод стоимости заказа оператором
	STATE_OP_ORDER_ASSIGN_EXEC_MENU     = "op_order_assign_exec_menu"     // Меню назначения исполнителей оператором
	STATE_OP_ORDER_FINAL_CONFIRM        = "op_order_final_confirm"        // Финальное подтверждение заказа оператором (после стоимости и исполнителей)

	// DEPRECATED or to be reviewed for operator flow
	STATE_OP_ORDER_SET_COST_IMMEDIATE    = "op_order_set_cost_imm"    // Был для немедленной установки стоимости, возможно, заменён STATE_OP_ORDER_COST_INPUT
	STATE_OP_ORDER_ASSIGN_EXEC_IMMEDIATE = "op_order_assign_exec_imm" // Был для немедленного назначения, возможно, заменён STATE_OP_ORDER_ASSIGN_EXEC_MENU
)

// Communication and Info States
// Состояния связи и информации
const (
	STATE_CONTACT_METHOD      = "contact_method"
	STATE_CHAT_MESSAGE_INPUT  = "chat_message_input"
	STATE_PHONE_OPTIONS       = "phone_options"
	STATE_PHONE_AWAIT_INPUT   = "phone_await_input"
	STATE_PHONE_REQUEST       = "phone_request"
	STATE_OPERATOR_VIEW_CHATS = "operator_view_chats"
)

// Referral Program States
// Состояния реферальной программы
const (
	STATE_INVITE_FRIEND           = "invite_friend"
	STATE_REFERRAL_LINK           = "referral_link"
	STATE_REFERRAL_QR             = "referral_qr"
	STATE_MY_REFERRALS            = "my_referrals"
	STATE_REFERRAL_PAYOUT_CONFIRM = "referral_payout_confirm"
)

// Staff Management States
// Состояния управления персоналом
const (
	STATE_STAFF_MENU             = "staff_menu"
	STATE_STAFF_LIST             = "staff_list"
	STATE_STAFF_INFO             = "staff_info"
	STATE_STAFF_ADD_NAME         = "staff_add_name"
	STATE_STAFF_ADD_SURNAME      = "staff_add_surname"
	STATE_STAFF_ADD_NICKNAME     = "staff_add_nickname"
	STATE_STAFF_ADD_PHONE        = "staff_add_phone"
	STATE_STAFF_ADD_CHATID       = "staff_add_chatid"
	STATE_STAFF_ADD_CARD_NUMBER  = "staff_add_card_number"
	STATE_STAFF_ADD_ROLE         = "staff_add_role"
	STATE_STAFF_EDIT             = "staff_edit"
	STATE_STAFF_EDIT_NAME        = "staff_edit_name"
	STATE_STAFF_EDIT_SURNAME     = "staff_edit_surname"
	STATE_STAFF_EDIT_NICKNAME    = "staff_edit_nickname"
	STATE_STAFF_EDIT_PHONE       = "staff_edit_phone"
	STATE_STAFF_EDIT_CARD_NUMBER = "staff_edit_card_number"
	STATE_STAFF_EDIT_ROLE        = "staff_edit_role"
	STATE_STAFF_BLOCK_REASON     = "staff_block_reason"
)

// Statistics States
// Состояния статистики
const (
	STATE_STATS_MENU       = "stats_menu"
	STATE_STATS_DATE       = "stats_date"
	STATE_STATS_MONTH      = "stats_month"
	STATE_STATS_DAY        = "stats_day"
	STATE_STATS_PERIOD     = "stats_period"
	STATE_STATS_PERIOD_END = "stats_period_end"
)

// User Blocking States
// Состояния блокировки пользователей
const (
	STATE_BLOCK_USER_MENU           = "block_user_menu"
	STATE_BLOCK_USER_SELECT         = "block_user_select"
	STATE_BLOCK_USER_CONFIRM_INFO   = "block_user_confirm_info"
	STATE_BLOCK_REASON              = "block_reason"
	STATE_UNBLOCK_USER_SELECT       = "unblock_user_select"
	STATE_UNBLOCK_USER_CONFIRM_INFO = "unblock_user_confirm_info"
)

// Admin/Operator Action States (related to orders, costs, etc.)
// Состояния действий администратора/оператора (связанные с заказами, стоимостью и т.д.)
const (
	STATE_COST_INPUT             = "cost_input" // Общее состояние для ввода стоимости (может использоваться и оператором для любого заказа)
	STATE_CANCEL_REASON          = "cancel_reason"
	STATE_ORDER_FINAL_COST_INPUT = "order_final_cost_input" // Для изменения финальной стоимости уже завершенного заказа
)

// Salary, Expenses, and Payout States (New Section)
// Состояния зарплат, расходов и выплат (Новый раздел)
const (
	STATE_MY_SALARY_MENU     = "my_salary_menu"
	STATE_VIEW_SALARY_OWED   = "view_salary_owed"
	STATE_VIEW_SALARY_EARNED = "view_salary_earned"

	// Driver Inline Report States

	STATE_DRIVER_REPORT_OVERALL_MENU                    = "driver_report_overall_menu"
	STATE_DRIVER_REPORT_INPUT_FUEL                      = "driver_report_input_fuel"
	STATE_DRIVER_REPORT_OTHER_EXPENSES_MENU             = "driver_report_other_expenses_menu"
	STATE_DRIVER_REPORT_INPUT_OTHER_EXPENSE_DESCRIPTION = "driver_report_input_other_expense_description"
	STATE_DRIVER_REPORT_INPUT_OTHER_EXPENSE_AMOUNT      = "driver_report_input_other_expense_amount"
	STATE_DRIVER_REPORT_CONFIRM_ADD_OTHER_EXPENSE       = "driver_report_confirm_add_other_expense"
	STATE_DRIVER_REPORT_EDIT_OTHER_EXPENSE_DESCRIPTION  = "driver_report_edit_other_expense_description"
	STATE_DRIVER_REPORT_EDIT_OTHER_EXPENSE_AMOUNT       = "driver_report_edit_other_expense_amount"
	STATE_DRIVER_REPORT_CONFIRM_DELETE_OTHER_EXPENSE    = "driver_report_confirm_delete_other_expense"
	STATE_DRIVER_REPORT_LOADERS_MENU                    = "driver_report_loaders_menu"
	STATE_DRIVER_REPORT_INPUT_LOADER_NAME               = "driver_report_input_loader_name"
	STATE_DRIVER_REPORT_INPUT_LOADER_SALARY             = "driver_report_input_loader_salary"
	STATE_DRIVER_REPORT_EDIT_LOADER_SALARY              = "driver_report_edit_loader_salary"
	STATE_DRIVER_REPORT_CONFIRM_DELETE_LOADER           = "driver_report_confirm_delete_loader"

	// Owner Payouts States
	STATE_OWNER_STAFF_PAYOUTS_MENU      = "owner_staff_payouts_menu"
	STATE_OWNER_SELECT_STAFF_FOR_PAYOUT = "owner_select_staff_for_payout"
	STATE_OWNER_CONFIRM_STAFF_PAYOUT    = "owner_confirm_staff_payout"

	// Owner Cash Management & Financial States
	STATE_OWNER_FINANCIAL_MAIN                    = "owner_financial_main" // Старый, для фин.отчетов по датам
	STATE_OWNER_FINANCIAL_VIEW_RECORD             = "owner_financial_view_record"
	STATE_OWNER_FINANCIAL_EDIT_RECORD             = "owner_financial_edit_record" // Для редактирования полей старого отчета
	STATE_OWNER_FINANCIAL_EDIT_FIELD              = "owner_financial_edit_field"  // Для ввода значения поля старого отчета
	STATE_OWNER_CASH_MANAGEMENT_MENU              = "owner_cash_management_menu"
	STATE_OWNER_CASH_ACTUAL_LIST                  = "owner_cash_actual_list"
	STATE_OWNER_CASH_SETTLED_LIST                 = "owner_cash_settled_list"
	STATE_OWNER_CASH_VIEW_DRIVER_SETTLEMENTS      = "owner_cash_view_driver_settlements"
	STATE_OWNER_CASH_EDIT_SETTLEMENT_FIELD        = "owner_cash_edit_settlement_field"
	STATE_OWNER_CASH_EDIT_SETTLEMENT_FIELD_SELECT = "owner_cash_edit_settlement_field_select"
	STATE_OPERATOR_REVIEW_SETTLEMENT              = "operator_review_settlement"
	STATE_OPERATOR_REJECT_REASON_INPUT            = "operator_reject_reason_input"
)
const (
	SETTLEMENT_STATUS_PENDING  = "pending"
	SETTLEMENT_STATUS_APPROVED = "approved"
	SETTLEMENT_STATUS_REJECTED = "rejected"
)
const (
	CALLBACK_PREFIX_OPERATOR_APPROVE_SETTLEMENT = "op_approve_set"
	CALLBACK_PREFIX_OPERATOR_REJECT_SETTLEMENT  = "op_reject_set"
)

// General Text Messages
// Общие текстовые сообщения
const (
	AccessDeniedMessage = "❌ У вас нет прав доступа для этого действия."
	InvisibleMessage    = "⌨️" // Используется для удаления ReplyKeyboard
)

// Order Categories, Statuses, User Roles
// Категории заказов, статусы, роли пользователей
const (
	CAT_WASTE      = "waste_removal"
	CAT_DEMOLITION = "demolition"
	CAT_MATERIALS  = "construction_materials"
	CAT_OTHER      = "other"
)
const (
	STATUS_NEW                   = "new"                   // Заказ создан клиентом или оператором (без стоимости/исполнителей)
	STATUS_AWAITING_COST         = "awaiting_cost"         // Оператор должен установить стоимость (устарел, если стоимость сразу в new)
	STATUS_AWAITING_CONFIRMATION = "awaiting_confirmation" // Клиент должен подтвердить стоимость
	STATUS_AWAITING_PAYMENT      = "awaiting_payment"      // Ожидание оплаты от клиента
	STATUS_INPROGRESS            = "in_progress"           // Заказ в работе (стоимость подтверждена или не требовалась, исполнители могут быть назначены)
	STATUS_COMPLETED             = "completed"             // Заказ физически выполнен исполнителями
	STATUS_CANCELED              = "canceled"
	STATUS_DRAFT                 = "draft"      // Черновик заказа, еще не подтвержден клиентом
	STATUS_CALCULATED            = "calculated" // Финансы (расходы, ЗП грузчиков, доля водителя) рассчитаны
	STATUS_SETTLED               = "settled"    // Все выплаты по этому заказу произведены (деньги сданы, ЗП водителю выплачена)
	// STATUS_AWAITING_CASH         = "awaiting_cash"      // Устарело, логика перенесена в DriverSettlement.PaidToOwnerAt
)
const (
	ROLE_USER         = "user"
	ROLE_OPERATOR     = "operator"
	ROLE_MAINOPERATOR = "main_operator"
	ROLE_DRIVER       = "driver"
	ROLE_LOADER       = "loader"
	ROLE_OWNER        = "owner"
)

// Media Limits
// Ограничения на медиа
const (
	MAX_PHOTOS = 30
	MAX_VIDEOS = 30
)

// Pagination
// Пагинация
const (
	OrdersPerPage      = 10
	StaffPerPage       = 10
	PayoutsPerPage     = 10
	CashRecordsPerPage = 10
)

// Payout Request Statuses
// Статусы запросов на выплату
const (
	PAYOUT_REQUEST_STATUS_PENDING   = "pending"
	PAYOUT_REQUEST_STATUS_APPROVED  = "approved"
	PAYOUT_REQUEST_STATUS_REJECTED  = "rejected"
	PAYOUT_REQUEST_STATUS_COMPLETED = "completed"
)

// Callback Data Prefixes
// Префиксы данных обратного вызова
const (
	// --- НАЧАЛО ИЗМЕНЕНИЯ ---
	CALLBACK_CONTINUE_IN_BOT = "continue_in_bot" // Для кнопки "Продолжить в боте"
	// --- КОНЕЦ ИЗМЕНЕНИЯ ---

	CALLBACK_PREFIX_SELECT_HOUR = "select_hour" // например, select_hour_09
	CALLBACK_PREFIX_MY_SALARY   = "my_salary"

	CALLBACK_PREFIX_OWNER_LOADER_PAY     = "own_loadpay" // Кажется, устарело
	CALLBACK_PREFIX_OWNER_STAFF_PAYOUT   = "own_staffpay"
	CALLBACK_PREFIX_MARK_ORDER_DONE      = "mark_ord_done"
	CALLBACK_PREFIX_ORDER_SET_FINAL_COST = "ord_set_final_cst" // Для установки финальной стоимости УЖЕ ЗАВЕРШЕННОГО заказа
	CALLBACK_PREFIX_ORDER_RESUME         = "ord_resume"
	CALLBACK_PREFIX_PAY_ORDER            = "pay_order"

	CALLBACK_PREFIX_DRIVER_SETTLEMENT = "drv_settle" // Запускает инлайн-отчет водителя
	CALLBACK_PREFIX_OWNER_FINANCIALS  = "own_fin"    // Старый, для фин.отчетов по датам (DEPRECATED)

	// Callback prefixes for Owner's Cash Management
	CALLBACK_PREFIX_OWNER_CASH_MANAGEMENT_MAIN     = "own_cash_main"
	CALLBACK_PREFIX_OWNER_CASH_ACTUAL_LIST         = "own_cash_act_list"
	CALLBACK_PREFIX_OWNER_CASH_SETTLED_LIST        = "own_cash_set_list"
	CALLBACK_PREFIX_OWNER_CASH_MARK_PAID           = "own_cash_mark_paid"
	CALLBACK_PREFIX_OWNER_CASH_MARK_UNPAID         = "own_cash_mark_unpaid"
	CALLBACK_PREFIX_OWNER_VIEW_DRIVER_SETTLEMENTS  = "own_cash_view_drv_sets"
	CALLBACK_PREFIX_OWNER_EDIT_SETTLEMENT          = "own_cash_edit_set"
	CALLBACK_PREFIX_OWNER_CASH_MARK_SALARY_PAID    = "own_cash_mark_sal_paid"
	CALLBACK_PREFIX_OWNER_CASH_MARK_SALARY_UNPAID  = "own_cash_mark_sal_unpaid"
	CALLBACK_PREFIX_OWNER_MARK_ALL_SALARY_PAID     = "own_mark_all_sal_paid"
	CALLBACK_PREFIX_OWNER_MARK_ALL_MONEY_DEPOSITED = "own_mark_all_mon_dep"

	CALLBACK_PREFIX_DRIVER_CREATE_ORDER = "drv_create_new_order"

	// Callback prefixes for Operator's order creation flow
	CALLBACK_PREFIX_OP_CREATE_NEW_ORDER            = "op_create_new_order"    // Button in operator's main menu
	CALLBACK_PREFIX_OP_CONFIRM_ORDER_SIMPLE_CREATE = "op_confirm_simple"      // Operator confirms: create order (status NEW)
	CALLBACK_PREFIX_OP_CONFIRM_ORDER_SET_COST      = "op_confirm_set_cost"    // Operator confirms: create order AND set cost (then status IN_PROGRESS)
	CALLBACK_PREFIX_OP_CONFIRM_ORDER_ASSIGN_EXEC   = "op_confirm_assign_exec" // Operator confirms: create order AND assign executors
	CALLBACK_PREFIX_OP_SKIP_COST                   = "op_skip_cost"           // Operator skips cost input during creation
	CALLBACK_PREFIX_OP_SKIP_ASSIGN_EXEC            = "op_skip_assign_exec"    // Operator skips executor assignment during creation
	CALLBACK_PREFIX_OP_FINALIZE_ORDER_CREATION     = "op_finalize_creation"   // Operator finalizes order creation (after cost/exec or skipping them)
	CALLBACK_PREFIX_OP_EDIT_ORDER_COST             = "op_edit_ord_cost"       // Button "Стоимость" in operator's order edit menu
	CALLBACK_PREFIX_OP_EDIT_ORDER_EXECS            = "op_edit_ord_execs"      // Button "Исполнители" in operator's order edit menu

	CALLBACK_PREFIX_EXECUTOR_NOTIFIED               = "exec_notified"
	CALLBACK_PREFIX_OPERATOR_VIEW_DRIVER_SETTLEMENT = "op_view_drv_set"
)

// Keys for order list callbacks and display (for operator)
// Ключи для callback'ов и отображения списков заказов (для оператора)
const (
	ORDER_LIST_KEY_NEW                   = "new_list"
	ORDER_LIST_KEY_AWAITING_CONFIRMATION = "await_conf_list"
	ORDER_LIST_KEY_IN_PROGRESS           = "in_prog_list"
	ORDER_LIST_KEY_COMPLETED             = "completed_list"
	ORDER_LIST_KEY_CANCELED              = "canceled_list"
	// ORDER_LIST_KEY_CALCULATED не был здесь, но используется в callback_admin_handlers
)

var MonthMap = map[time.Month]string{
	time.January:   "января",
	time.February:  "февраля",
	time.March:     "марта",
	time.April:     "апреля",
	time.May:       "мая",
	time.June:      "июня",
	time.July:      "июля",
	time.August:    "августа",
	time.September: "сентября",
	time.October:   "октября",
	time.November:  "ноября",
	time.December:  "декабря",
}
var CategoryDisplayMap = map[string]string{
	CAT_WASTE:      "Вывоз мусора",
	CAT_DEMOLITION: "Демонтаж",
	CAT_MATERIALS:  "Стройматериалы",
	CAT_OTHER:      "Другое",
}
var CategoryEmojiMap = map[string]string{
	CAT_WASTE:      "🗑️",
	CAT_DEMOLITION: "🛠️",
	CAT_MATERIALS:  "🧱",
	CAT_OTHER:      "❓",
}
var StatusDisplayMap = map[string]string{
	STATUS_NEW:                   "Новые",
	STATUS_AWAITING_COST:         "Ожидание стоимости", // Может быть устаревшим
	STATUS_AWAITING_CONFIRMATION: "Ожидаем клиента",
	STATUS_AWAITING_PAYMENT:      "Ожидание оплаты",
	STATUS_INPROGRESS:            "В работе",
	STATUS_COMPLETED:             "Завершён",
	STATUS_CANCELED:              "Отменён",
	STATUS_DRAFT:                 "Черновик",
	STATUS_CALCULATED:            "Рассчитан",
	STATUS_SETTLED:               "Закрыт (оплачен)",
}
var StatusEmojiMap = map[string]string{
	STATUS_NEW:                   "🆕",
	STATUS_AWAITING_COST:         "💰",
	STATUS_AWAITING_CONFIRMATION: "⏳",
	STATUS_AWAITING_PAYMENT:      "💳",
	STATUS_INPROGRESS:            "🚚",
	STATUS_COMPLETED:             "✅",
	STATUS_CANCELED:              "❌",
	STATUS_DRAFT:                 "📝",
	STATUS_CALCULATED:            "🧮",
	STATUS_SETTLED:               "💯",
}

var WasteSubcategoryMap = map[string]string{
	"construct":   "🏗️ Строительный мусор",
	"household":   "🗑️ Бытовой мусор",
	"metal":       "⚙️ Металл",
	"junk":        "📦 Хлам",
	"greenery":    "🌳 Ветки, деревья, трава",
	"tires":       "🚗 Старые покрышки",
	"other_waste": "❓ Другое",
}
var DemolitionSubcategoryMap = map[string]string{
	"walls":      "🧱 Демонтаж стен",
	"partitions": "🏠 Демонтаж перегородок",
	"floors":     "🪚 Демонтаж полов",
	"ceilings":   "🏛️ Демонтаж потолков",
	"plumbing":   "🚽 Демонтаж сантехники",
	"tiles":      "🚿 Демонтаж плитки",
	"other_demo": "❓ Другое",
}
var WasteSubcategoryShortMap = map[string]string{ // Используется в GetDisplaySubcategory
	"construct":   "🏗️ Строймусор",
	"household":   "🗑️ Бытовой",
	"metal":       "⚙️ Металл",
	"junk":        "📦 Хлам",
	"greenery":    "🌳 Ветки",
	"tires":       "🚗 Покрышки",
	"other_waste": "❓ Другое",
}
var DemolitionSubcategoryShortMap = map[string]string{ // Используется в GetDisplaySubcategory
	"walls":      "🧱 Стены",
	"partitions": "🏠 Перегородки",
	"floors":     "🪚 Полы",
	"ceilings":   "🏛️ Потолки",
	"plumbing":   "🚽 Сантехника",
	"tiles":      "🚿 Плитка",
	"other_demo": "❓ Другое",
}

var WasteSubcategoryReverseMap = make(map[string]string)
var DemolitionSubcategoryReverseMap = make(map[string]string)

var OrderListCallbackMap = map[string]string{
	ORDER_LIST_KEY_NEW:                   fmt.Sprintf("operator_orders_%s_0", STATUS_NEW), // operator_orders_new_0
	ORDER_LIST_KEY_AWAITING_CONFIRMATION: fmt.Sprintf("operator_orders_%s_0", STATUS_AWAITING_CONFIRMATION),
	ORDER_LIST_KEY_IN_PROGRESS:           fmt.Sprintf("operator_orders_%s_0", STATUS_INPROGRESS),
	ORDER_LIST_KEY_COMPLETED:             fmt.Sprintf("operator_orders_%s_0", STATUS_COMPLETED),
	ORDER_LIST_KEY_CANCELED:              fmt.Sprintf("operator_orders_%s_0", STATUS_CANCELED),
	// Для STATUS_CALCULATED используется прямой коллбэк в menu_handlers_order_view.go
}

var OrderListDisplayMap = map[string]string{
	ORDER_LIST_KEY_NEW:                   "🆕 Новые",
	ORDER_LIST_KEY_AWAITING_CONFIRMATION: "⏳ Ждём клиента",
	ORDER_LIST_KEY_IN_PROGRESS:           "🚚 В работе",
	ORDER_LIST_KEY_COMPLETED:             "✅ Завершенные",
	ORDER_LIST_KEY_CANCELED:              "❌ Отмененные",
}

func init() {
	for k, v := range WasteSubcategoryMap {
		WasteSubcategoryReverseMap[v] = k
	}
	for k, v := range DemolitionSubcategoryMap {
		DemolitionSubcategoryReverseMap[v] = k
	}
}

// Driver Inline Report Prefixes and States
// (Остаются без изменений, если не пересекаются с новыми)
const (
	CALLBACK_PREFIX_DRIVER_REPORT_OVERALL_MENU = "drv_rpt_main"
	CALLBACK_PREFIX_DRIVER_REPORT_SET_FUEL     = "drv_rpt_fuel"
	// CALLBACK_PREFIX_DRIVER_REPORT_SET_OTHER (DEPRECATED, replaced by OTHER_EXPENSES_MENU)
	CALLBACK_PREFIX_DRIVER_REPORT_LOADERS_MENU                         = "drv_rpt_load_menu"
	CALLBACK_PREFIX_DRIVER_REPORT_ADD_LOADER_PROMPT                    = "drv_rpt_add_load_p"
	CALLBACK_PREFIX_DRIVER_REPORT_EDIT_LOADER_PROMPT                   = "drv_rpt_edit_load_p"
	CALLBACK_PREFIX_DRIVER_REPORT_DELETE_LOADER_CONFIRM                = "drv_rpt_del_load_c"
	CALLBACK_PREFIX_DRIVER_REPORT_SAVE_FINAL                           = "drv_rpt_save"
	CALLBACK_PREFIX_DRIVER_REPORT_CANCEL_ALL                           = "drv_rpt_cancel"
	CALLBACK_PREFIX_DRIVER_REPORT_OTHER_EXPENSES_MENU                  = "drv_rpt_oth_menu"
	CALLBACK_PREFIX_DRIVER_REPORT_ADD_OTHER_EXPENSE_DESCRIPTION_PROMPT = "drv_rpt_add_oth_desc_p"
	CALLBACK_PREFIX_DRIVER_REPORT_EDIT_OTHER_EXPENSE_PROMPT            = "drv_rpt_edit_oth_p"
	CALLBACK_PREFIX_DRIVER_REPORT_DELETE_OTHER_EXPENSE_CONFIRM         = "drv_rpt_del_oth_c"
	CALLBACK_PREFIX_DRIVER_REPORT_DELETE_OTHER_EXPENSE_SHOW_CONFIRM    = "drv_rpt_del_oth_sc"
)

// View Types for Driver Settlements (used in callbacks)
const (
	VIEW_TYPE_ACTUAL_SETTLEMENTS  = "actual"
	VIEW_TYPE_SETTLED_SETTLEMENTS = "settled"
)
