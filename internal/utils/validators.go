package utils

import (
	"fmt"
	"log"
	"regexp"
	"strconv"
	"strings"
	"time"

	"Original/internal/constants" // Используем Original как имя модуля
)

// localPhoneRegex (не экспортируется) используется внутри ValidatePhoneNumber.
// localPhoneRegex (not exported) is used inside ValidatePhoneNumber.
var localPhoneRegex = regexp.MustCompile(`^(?:\+7|8|7)?(\d{3})(\d{3})(\d{2})(\d{2})$`)

// ValidatePhoneNumber проверяет и нормализует номер телефона.
// Возвращает номер в формате +7XXXXXXXXXX или ошибку.
// ValidatePhoneNumber checks and normalizes a phone number.
// Returns the number in +7XXXXXXXXXX format or an error.
func ValidatePhoneNumber(phone string) (string, error) {
	phone = strings.ReplaceAll(phone, "\\", "") // Удаляем возможные экранирующие слеши / Remove possible escape slashes
	phone = strings.TrimSpace(phone)

	// Удаляем все нечисловые символы, кроме начального '+'
	// Remove all non-numeric characters except for the initial '+'
	digitsOnly := regexp.MustCompile(`[^\d+]`).ReplaceAllString(phone, "")

	if strings.HasPrefix(digitsOnly, "+") {
		if strings.HasPrefix(digitsOnly, "+7") && len(digitsOnly) == 12 { // +7XXXXXXXXXX
			if regexp.MustCompile(`^\+7\d{10}$`).MatchString(digitsOnly) {
				return digitsOnly, nil
			}
			return "", fmt.Errorf("номер должен быть в формате +7XXXXXXXXXX")
		}
		// Другие международные форматы пока не поддерживаем так строго
		// Other international formats are not strictly supported yet
		return "", fmt.Errorf("поддерживаются только российские номера в формате +7XXXXXXXXXX или 8XXXXXXXXXX")
	}

	// Если не начинается с '+', предполагаем российский номер
	// If not starting with '+', assume a Russian number
	digitsOnly = regexp.MustCompile(`[^\d]`).ReplaceAllString(phone, "")

	if len(digitsOnly) == 11 && (digitsOnly[0] == '8' || digitsOnly[0] == '7') { // 8XXXXXXXXXX или 7XXXXXXXXXX
		normalized := "+7" + digitsOnly[1:]
		if regexp.MustCompile(`^\+7\d{10}$`).MatchString(normalized) {
			return normalized, nil
		}
		return "", fmt.Errorf("неверный формат номера (ожидалось 11 цифр, начиная с 7 или 8)")
	}
	if len(digitsOnly) == 10 { // XXXXXXXXXX
		normalized := "+7" + digitsOnly
		if regexp.MustCompile(`^\+7\d{10}$`).MatchString(normalized) {
			return normalized, nil
		}
		return "", fmt.Errorf("неверный формат номера (ожидалось 10 цифр)")
	}

	return "", fmt.Errorf("неверный формат номера телефона, укажите в формате +7XXXXXXXXXX или 8XXXXXXXXXX")
}

// ValidateDate проверяет и парсит строку с датой.
// Поддерживает форматы "2 January 2006" (английский) и "ДД МЕСЯЦ ГГГГ" (русский).
// Возвращает time.Time и ошибку.
// ValidateDate checks and parses a date string.
// Supports "2 January 2006" (English) and "DD MONTH YYYY" (Russian) formats.
// Returns time.Time and an error.
func ValidateDate(dateStr string) (time.Time, error) {
	dateStr = strings.Replace(dateStr, "_", " ", -1) // На случай если в callback_data используется _ вместо пробела
	dateStr = strings.TrimSpace(dateStr)

	if dateStr == "" {
		return time.Time{}, fmt.Errorf("строка даты пуста")
	}

	var parsedDate time.Time
	var err error

	// Форматы для парсинга, в порядке приоритета
	dateFormatsToTry := []string{
		"2006-01-02",          // YYYY-MM-DD (основной формат из БД)
		"2 January 2006",      // D Month YYYY (английский, полное название месяца)
		"02.01.2006",          // DD.MM.YYYY (распространенный русский формат)
		time.RFC3339,          // "2006-01-02T15:04:05Z07:00" (на случай если приходит полный timestamp)
		"2006-01-02 15:04:05", // Формат timestamp без таймзоны
	}

	for _, format := range dateFormatsToTry {
		parsedDate, err = time.ParseInLocation(format, dateStr, time.Local) // Используем time.Local для парсинга
		if err == nil {
			log.Printf("ValidateDate: дата '%s' успешно распознана форматом '%s' -> %s", dateStr, format, parsedDate.Format("2006-01-02"))
			return parsedDate, nil
		}
	}

	// Если стандартные форматы не подошли, пробуем формат с русскими месяцами "ДД МЕСЯЦ ГГГГ"
	// (например, "17 мая 2025" или "3 сентября 2024")
	parts := strings.Fields(dateStr)
	if len(parts) == 3 {
		day, errDay := strconv.Atoi(parts[0])
		if errDay == nil && day >= 1 && day <= 31 {
			monthStrRu := strings.ToLower(parts[1])
			var month time.Month
			foundMonth := false

			for m, name := range constants.MonthMap { // constants.MonthMap должен содержать русские названия месяцев в нижнем регистре или быть адаптирован
				if name == monthStrRu { // Предполагаем, что constants.MonthMap содержит {"января": time.January, "февраля": time.February, ...}
					month = m
					foundMonth = true
					break
				}
			}
			// Дополнительная попытка сопоставить по первым 3-4 буквам, если полные названия не совпали
			if !foundMonth {
				for m, name := range constants.MonthMap {
					if (len(monthStrRu) >= 3 && strings.HasPrefix(name, monthStrRu[:3])) || (len(monthStrRu) >= 4 && strings.HasPrefix(name, monthStrRu[:4])) {
						month = m
						foundMonth = true
						log.Printf("ValidateDate: русский месяц '%s' сопоставлен с '%s' по частичному совпадению.", monthStrRu, name)
						break
					}
				}
			}

			if foundMonth {
				year, errYear := strconv.Atoi(parts[2])
				if errYear == nil {
					currentYear := time.Now().Year()
					// Проверка на разумность года (например, текущий +/- 5 лет)
					if year >= currentYear-5 && year <= currentYear+5 {
						finalDate := time.Date(year, month, day, 0, 0, 0, 0, time.Local)
						log.Printf("ValidateDate: дата '%s' успешно распознана (русский формат) -> %s", dateStr, finalDate.Format("2006-01-02"))
						return finalDate, nil
					} else {
						log.Printf("ValidateDate: год %d для русского формата ('%s') выходит за разумные пределы (%d-%d).", year, dateStr, currentYear-5, currentYear+5)
					}
				} else {
					// log.Printf("ValidateDate: ошибка парсинга года '%s' для русского формата ('%s'): %v", parts[2], dateStr, errYear)
				}
			} else {
				// log.Printf("ValidateDate: не удалось найти русский месяц для '%s' в строке '%s'", parts[1], dateStr)
			}
		} else if errDay != nil {
			// log.Printf("ValidateDate: ошибка парсинга дня '%s' для русского формата ('%s'): %v", parts[0], dateStr, errDay)
		}
	}

	log.Printf("ValidateDate: не удалось распознать формат даты: '%s'.", dateStr)
	return time.Time{}, fmt.Errorf("некорректный формат даты: '%s'. Поддерживаемые форматы: ГГГГ-ММ-ДД, Д МесяцПолностью Год (англ.), ДД.ММ.ГГГГ, Д месяцПоРусски Год", dateStr)
}

// ValidateLocation проверяет корректность координат широты и долготы.
// ValidateLocation checks the correctness of latitude and longitude coordinates.
func ValidateLocation(latitude, longitude float64) error {
	if latitude < -90 || latitude > 90 {
		return fmt.Errorf("широта должна быть в диапазоне [-90, 90], получено: %.6f", latitude)
	}
	if longitude < -180 || longitude > 180 {
		return fmt.Errorf("долгота должна быть в диапазоне [-180, 180], получено: %.6f", longitude)
	}
	return nil
}

// IsOperatorOrHigher проверяет, является ли роль оператором или выше.
// IsOperatorOrHigher checks if the role is operator or higher.
func IsOperatorOrHigher(role string) bool {
	return role == constants.ROLE_OPERATOR ||
		role == constants.ROLE_MAINOPERATOR ||
		role == constants.ROLE_OWNER
}

// IsRoleOrHigher проверяет, соответствует ли роль пользователя минимально требуемой роли.
// Иерархия ролей: User < Loader/Driver < Operator < MainOperator < Owner
// IsRoleOrHigher checks if the user's role meets the minimum required role.
// Role hierarchy: User < Loader/Driver < Operator < MainOperator < Owner
func IsRoleOrHigher(userRole string, requiredRole string) bool {
	roleHierarchy := map[string]int{
		constants.ROLE_USER:         0,
		constants.ROLE_LOADER:       1,
		constants.ROLE_DRIVER:       1, // Водитель и грузчик на одном уровне в этой иерархии / Driver and loader are at the same level in this hierarchy
		constants.ROLE_OPERATOR:     2,
		constants.ROLE_MAINOPERATOR: 3,
		constants.ROLE_OWNER:        4,
	}

	userLevel, okUser := roleHierarchy[userRole]
	requiredLevel, okRequired := roleHierarchy[requiredRole]

	if !okUser || !okRequired {
		log.Printf("IsRoleOrHigher: неизвестная роль при сравнении: userRole='%s', requiredRole='%s'", userRole, requiredRole)
		return false // Если одна из ролей неизвестна, считаем, что доступ запрещен / If one of the roles is unknown, access is considered denied
	}
	return userLevel >= requiredLevel
}

// ValidateCardNumber проверяет базовый формат номера карты (длина и только цифры).
// Можно расширить для проверки по алгоритму Луна.
// ValidateCardNumber checks the basic format of a card number (length and digits only).
// Can be extended for Luhn algorithm validation.
func ValidateCardNumber(cardNumber string) error {
	cardNumber = strings.ReplaceAll(cardNumber, " ", "") // Удаляем пробелы / Remove spaces
	if len(cardNumber) < 16 || len(cardNumber) > 19 {
		return fmt.Errorf("номер карты должен содержать от 16 до 19 цифр")
	}
	if !regexp.MustCompile(`^[0-9]+$`).MatchString(cardNumber) {
		return fmt.Errorf("номер карты должен содержать только цифры")
	}
	// TODO: Добавить проверку по алгоритму Луна для большей точности, если необходимо.
	// TODO: Add Luhn algorithm check for more accuracy if needed.
	// Например, можно найти готовую реализацию или написать свою.
	// For example, find a ready-made implementation or write your own.
	return nil
}

// IsCommandInCategory проверяет, принадлежит ли команда одной из категорий.
// IsCommandInCategory checks if a command belongs to one of the categories.
func IsCommandInCategory(command string, categoryCommands []string) bool {
	for _, cmdPrefix := range categoryCommands {
		if strings.HasPrefix(command, cmdPrefix) {
			return true
		}
	}
	return false
}
