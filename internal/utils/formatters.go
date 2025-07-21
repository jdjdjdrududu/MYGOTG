// Файл: internal/utils/formatters.go

package utils

import (
	"Original/internal/constants" // Для constants.MonthMap, WasteSubcategoryMap и т.д.
	"Original/internal/models"    // Для models.Order, models.User
	"fmt"
	"log"
	"regexp"
	"strconv" // НУЖНО ДЛЯ НОВОЙ ФУНКЦИИ
	"strings"
	"time" // Для GetRussianMonthName

	"github.com/google/uuid" // Для GenerateUUID
)

// Int64SliceToStringSlice преобразует слайс int64 в слайс string.
func Int64SliceToStringSlice(int64Slice []int64) []string {
	stringSlice := make([]string, len(int64Slice))
	for i, v := range int64Slice {
		stringSlice[i] = strconv.FormatInt(v, 10)
	}
	return stringSlice
}

// FormatDateForDisplay форматирует дату для отображения (например, "25 мая").
// dateStr должен быть в формате "YYYY-MM-DD" или "02 January 2006" или других, поддерживаемых ValidateDate.
func FormatDateForDisplay(dateStr string) (string, error) {
	if dateStr == "" {
		return "не указана", nil
	}
	parsedDate, err := ValidateDate(dateStr)
	if err != nil {
		log.Printf("FormatDateForDisplay: не удалось распарсить дату '%s' через ValidateDate: %v", dateStr, err)
		return dateStr, err
	}
	day := parsedDate.Day()
	monthName := constants.MonthMap[parsedDate.Month()]
	return fmt.Sprintf("%d %s", day, monthName), nil
}

// StripEmoji удаляет эмодзи из строки.
func StripEmoji(text string) string {
	re := regexp.MustCompile(`[\p{So}\p{Sk}]`)
	return strings.TrimSpace(re.ReplaceAllString(text, ""))
}

// FormatPhoneNumber форматирует номер телефона для отображения.
func FormatPhoneNumber(phone string) string {
	re := regexp.MustCompile(`[^\d+]`)
	cleanedPhone := re.ReplaceAllString(phone, "")

	if strings.HasPrefix(cleanedPhone, "+7") && len(cleanedPhone) == 12 {
		return fmt.Sprintf("+7 (%s) %s-%s-%s", cleanedPhone[2:5], cleanedPhone[5:8], cleanedPhone[8:10], cleanedPhone[10:12])
	}
	if len(cleanedPhone) == 11 && (cleanedPhone[0] == '8' || cleanedPhone[0] == '7') {
		return fmt.Sprintf("+7 (%s) %s-%s-%s", cleanedPhone[1:4], cleanedPhone[4:7], cleanedPhone[7:9], cleanedPhone[9:11])
	}
	if len(cleanedPhone) == 10 {
		return fmt.Sprintf("+7 (%s) %s-%s-%s", cleanedPhone[0:3], cleanedPhone[3:6], cleanedPhone[6:8], cleanedPhone[8:10])
	}
	return phone
}

// EscapeTelegramMarkdown экранирует специальные символы для Telegram Markdown (старый стиль).
func EscapeTelegramMarkdown(text string) string {
	var replacer = strings.NewReplacer(
		"_", "\\_", "*", "\\*", "`", "\\`", "[", "\\[",
	)
	return replacer.Replace(text)
}

// EscapeTelegramMarkdownV2 экранирует специальные символы для Telegram MarkdownV2.
func EscapeTelegramMarkdownV2(text string) string {
	var replacer = strings.NewReplacer(
		"_", "\\_", "*", "\\*", "[", "\\[", "]", "\\]", "(", "\\(", ")", "\\)",
		"~", "\\~", "`", "\\`", ">", "\\>", "#", "\\#", "+", "\\+", "-", "\\-",
		"=", "\\=", "|", "\\|", "{", "\\{", "}", "\\}", ".", "\\.", "!", "\\!",
	)
	return replacer.Replace(text)
}

// GetDisplaySubcategory возвращает отображаемое имя подкатегории.
func GetDisplaySubcategory(order models.Order) string {
	if order.Category == constants.CAT_WASTE { //
		if val, ok := constants.WasteSubcategoryMap[order.Subcategory]; ok { //
			return val
		}
		if shortVal, okShort := constants.WasteSubcategoryShortMap[order.Subcategory]; okShort { //
			return shortVal
		}
		log.Printf("GetDisplaySubcategory: не найдено отображение для подкатегории мусора '%s'", order.Subcategory)
		return order.Subcategory
	} else if order.Category == constants.CAT_DEMOLITION { //
		if val, ok := constants.DemolitionSubcategoryMap[order.Subcategory]; ok { //
			return val
		}
		if shortVal, okShort := constants.DemolitionSubcategoryShortMap[order.Subcategory]; okShort { //
			return shortVal
		}
		log.Printf("GetDisplaySubcategory: не найдено отображение для подкатегории демонтажа '%s'", order.Subcategory)
		return order.Subcategory
	}
	return order.Subcategory
}

// GetRoleDisplayName возвращает отображаемое имя роли на русском языке.
func GetRoleDisplayName(roleKey string) string {
	names := map[string]string{
		constants.ROLE_USER:         "🙎 Пользователь",       //
		constants.ROLE_OPERATOR:     "👩‍💻 Оператор",         //
		constants.ROLE_MAINOPERATOR: "👨‍💼 Главный оператор", //
		constants.ROLE_DRIVER:       "🚒 Водитель",           //
		constants.ROLE_LOADER:       "👷 Грузчик",            //
		constants.ROLE_OWNER:        "👑 Владелец",           //
	}
	if name, ok := names[roleKey]; ok {
		return name
	}
	return roleKey
}

// GetBackText возвращает кастомный текст для кнопки "Назад" на основе ключа коллбэка.
func GetBackText(callbackKey string) (string, bool) {
	customBackTexts := map[string]string{
		"manage_orders":                       "🔙 К упр. заказами",
		"staff_menu":                          "🔙 В меню штата",
		"stats_menu":                          "🔙 В меню статистики",
		constants.STATE_STAFF_ADD_NAME:        "🔙 В меню штата",   //
		constants.STATE_STAFF_ADD_SURNAME:     "🔙 К имени",        //
		constants.STATE_STAFF_ADD_NICKNAME:    "🔙 К фамилии",      //
		constants.STATE_STAFF_ADD_PHONE:       "🔙 К позывному",    //
		constants.STATE_STAFF_ADD_CHATID:      "🔙 К телефону",     //
		constants.STATE_STAFF_ADD_CARD_NUMBER: "🔙 К ChatID",       //
		constants.STATE_STAFF_ADD_ROLE:        "🔙 К номеру карты", //
		"back_to_category":                    "⬅️ К Категориям",
		"back_to_subcategory":                 "⬅️ К Подкатегориям",
		// --- НАЧАЛО ИЗМЕНЕНИЯ: Добавляем ключ для "Назад к Описанию" ---
		"back_to_description": "⬅️ К Описанию",
		// --- КОНЕЦ ИЗМЕНЕНИЯ ---
		"back_to_name":    "⬅️ К Имени",
		"back_to_date":    "⬅️ К Дате",
		"back_to_time":    "⬅️ Ко Времени",
		"back_to_phone":   "⬅️ К Телефону",
		"back_to_address": "⬅️ К Адресу",
		"back_to_photo":   "⬅️ К Фото",
		"back_to_payment": "⬅️ К Оплате",
		"back_to_main":    "🏢 Главное меню",
		// Для нового флоу отчета водителя
		constants.CALLBACK_PREFIX_DRIVER_SETTLEMENT + "_back_to_fuel_prompt":          "🔙 К топливу",          //
		constants.CALLBACK_PREFIX_DRIVER_SETTLEMENT + "_back_to_other_prompt":         "🔙 К прочим расходам",  //
		constants.CALLBACK_PREFIX_DRIVER_SETTLEMENT + "_back_to_loaders_count_prompt": "🔙 К кол-ву грузчиков", //
		// Для loader_id и loader_salary может потребоваться динамический текст с индексом
	}
	if text, ok := customBackTexts[callbackKey]; ok {
		return text, true
	}
	if strings.HasPrefix(callbackKey, "staff_info_") {
		return "🔙 К инфо о сотруднике", true
	}
	if strings.HasPrefix(callbackKey, "staff_edit_menu_") {
		return "🔙 К выбору поля", true
	}
	if strings.HasPrefix(callbackKey, constants.CALLBACK_PREFIX_DRIVER_SETTLEMENT+"_back_to_loader_id_prompt") { //
		return "🔙 К ID грузчика", true
	}
	if strings.HasPrefix(callbackKey, constants.CALLBACK_PREFIX_DRIVER_SETTLEMENT+"_back_to_loader_salary_prompt") { //
		return "🔙 К ЗП грузчика", true
	}
	return "⬅️ Назад", false
}

// GetRussianMonthName преобразует английское название месяца в русское.
func GetRussianMonthName(englishMonthName string) string {
	var m time.Month
	switch strings.ToLower(englishMonthName) {
	case "january":
		m = time.January
	case "february":
		m = time.February
	case "march":
		m = time.March
	case "april":
		m = time.April
	case "may":
		m = time.May
	case "june":
		m = time.June
	case "july":
		m = time.July
	case "august":
		m = time.August
	case "september":
		m = time.September
	case "october":
		m = time.October
	case "november":
		m = time.November
	case "december":
		m = time.December
	default:
		return englishMonthName
	}
	if russianName, ok := constants.MonthMap[m]; ok { //
		return russianName
	}
	return englishMonthName
}

// GenerateUUID генерирует новый UUID.
func GenerateUUID() string {
	return uuid.New().String()
}

// GetUserDisplayName формирует отображаемое имя пользователя.
func GetUserDisplayName(user models.User) string {
	nameParts := []string{}
	if user.FirstName != "" {
		nameParts = append(nameParts, user.FirstName)
	}
	if user.LastName != "" {
		nameParts = append(nameParts, user.LastName)
	}
	name := strings.TrimSpace(strings.Join(nameParts, " "))

	if name == "" {
		if user.Nickname.Valid && user.Nickname.String != "" {
			name = user.Nickname.String
		} else {
			name = fmt.Sprintf("User %d", user.ChatID)
		}
	} else {
		if user.Nickname.Valid && user.Nickname.String != "" {
			name = fmt.Sprintf("%s (@%s)", name, user.Nickname.String)
		}
	}
	return name
}
