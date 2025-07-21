package formatters

import (
	"Original/internal/constants"
	"Original/internal/models"
	"Original/internal/utils"
	"fmt"
	"strings"
)

const (
	separator = "─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─"
)

// FormatOrderConfirmationForUser форматирует сообщение для клиента на этапе подтверждения создания заказа.
// На этом этапе еще нет ID заказа, стоимости от оператора и исполнителей.
func FormatOrderConfirmationForUser(orderData models.Order) string {
	var summaryBuilder strings.Builder

	// --- Блок "Ваши данные" ---
	summaryBuilder.WriteString("👤 *ВАШИ ДАННЫЕ:*\n")
	summaryBuilder.WriteString(fmt.Sprintf(" •  Имя: %s\n", utils.EscapeTelegramMarkdown(orderData.Name)))
	summaryBuilder.WriteString(fmt.Sprintf(" •  Телефон: %s\n", utils.EscapeTelegramMarkdown(utils.FormatPhoneNumber(orderData.Phone))))
	summaryBuilder.WriteString("\n")

	// --- Блок "Детали заказа" ---
	displaySubcategory := utils.GetDisplaySubcategory(orderData)
	formattedDate, _ := utils.FormatDateForDisplay(orderData.Date)
	timeStr := orderData.Time
	if timeStr == "" {
		timeStr = "В ближайшее время"
	}
	paymentStr := "По выполнению"
	if orderData.Payment == "now" {
		paymentStr = "Сразу (скидка 5%)"
	}

	summaryBuilder.WriteString("📋 *ДЕТАЛИ ЗАКАЗА:*\n")
	summaryBuilder.WriteString(fmt.Sprintf(" •  Услуга: %s (%s)\n",
		utils.EscapeTelegramMarkdown(constants.CategoryDisplayMap[orderData.Category]),
		utils.EscapeTelegramMarkdown(displaySubcategory)))
	summaryBuilder.WriteString(fmt.Sprintf(" •  Адрес: %s\n", utils.EscapeTelegramMarkdown(orderData.Address)))
	summaryBuilder.WriteString(fmt.Sprintf(" •  Дата и время: %s, %s\n",
		utils.EscapeTelegramMarkdown(formattedDate),
		utils.EscapeTelegramMarkdown(timeStr)))

	if orderData.Description != "" {
		summaryBuilder.WriteString(fmt.Sprintf(" •  Описание: %s\n", utils.EscapeTelegramMarkdown(orderData.Description)))
	}
	if len(orderData.Photos) > 0 || len(orderData.Videos) > 0 {
		var mediaParts []string
		if len(orderData.Photos) > 0 {
			mediaParts = append(mediaParts, fmt.Sprintf("%d фото", len(orderData.Photos)))
		}
		if len(orderData.Videos) > 0 {
			mediaParts = append(mediaParts, fmt.Sprintf("%d видео", len(orderData.Videos)))
		}
		summaryBuilder.WriteString(fmt.Sprintf(" •  Медиа: %s\n", strings.Join(mediaParts, ", ")))
	}
	summaryBuilder.WriteString(fmt.Sprintf(" •  Оплата: %s\n", utils.EscapeTelegramMarkdown(paymentStr)))

	header := "✨ *Пожалуйста, проверьте ваш заказ*"
	footer := "Всё верно? Нажмите \"Подтвердить\", и мы примем заказ в работу. Вы сможете отменить его до того, как оператор назначит стоимость."

	return fmt.Sprintf("%s\n%s\n%s\n%s\n%s",
		header, separator, summaryBuilder.String(), separator, footer)
}

// FormatOrderDetailsForUser форматирует сообщение для клиента при просмотре им существующего заказа.
// Показывает стоимость и исполнителей (в ограниченном виде), если они есть.
func FormatOrderDetailsForUser(order models.Order, assignedExecutors []models.Executor) string {
	var summaryBuilder strings.Builder

	// --- Блок "Статус" ---
	summaryBuilder.WriteString(fmt.Sprintf("⚙️ *Статус:* %s\n\n", utils.EscapeTelegramMarkdown(constants.StatusDisplayMap[order.Status])))

	// --- Блок "Ваши данные" ---
	summaryBuilder.WriteString("👤 *ВАШИ ДАННЫЕ:*\n")
	summaryBuilder.WriteString(fmt.Sprintf(" •  Имя: %s\n", utils.EscapeTelegramMarkdown(order.Name)))
	summaryBuilder.WriteString(fmt.Sprintf(" •  Телефон: %s\n", utils.EscapeTelegramMarkdown(utils.FormatPhoneNumber(order.Phone))))
	summaryBuilder.WriteString("\n")

	// --- Блок "Детали заказа" ---
	displaySubcategory := utils.GetDisplaySubcategory(order)
	formattedDate, _ := utils.FormatDateForDisplay(order.Date)
	timeStr := order.Time
	if timeStr == "" {
		timeStr = "в ближайшее время"
	}

	summaryBuilder.WriteString("📋 *ДЕТАЛИ ЗАКАЗА:*\n")
	summaryBuilder.WriteString(fmt.Sprintf(" •  Услуга: %s (%s)\n",
		utils.EscapeTelegramMarkdown(constants.CategoryDisplayMap[order.Category]),
		utils.EscapeTelegramMarkdown(displaySubcategory)))
	summaryBuilder.WriteString(fmt.Sprintf(" •  Адрес: %s\n", utils.EscapeTelegramMarkdown(order.Address)))
	summaryBuilder.WriteString(fmt.Sprintf(" •  Дата и время: %s, %s\n",
		utils.EscapeTelegramMarkdown(formattedDate),
		utils.EscapeTelegramMarkdown(timeStr)))
	summaryBuilder.WriteString("\n")

	// --- Блок "Финансы" ---
	if order.Cost.Valid && order.Cost.Float64 > 0 {
		paymentStr := "По выполнению"
		if order.Payment == "now" {
			paymentStr = "Сразу (скидка 5%)"
		}
		summaryBuilder.WriteString("💰 *ФИНАНСЫ:*\n")
		summaryBuilder.WriteString(fmt.Sprintf(" •  Стоимость: *%.0f ₽*\n", order.Cost.Float64))
		summaryBuilder.WriteString(fmt.Sprintf(" •  Оплата: %s\n", utils.EscapeTelegramMarkdown(paymentStr)))
		summaryBuilder.WriteString("\n")
	}

	// --- Блок "Исполнители" ---
	if len(assignedExecutors) > 0 {
		summaryBuilder.WriteString("👷 *ИСПОЛНИТЕЛИ:*\n")
		for _, exec := range assignedExecutors {
			if exec.Role == constants.ROLE_DRIVER {
				summaryBuilder.WriteString(fmt.Sprintf(" •  🚚 Водитель: %s\n", utils.EscapeTelegramMarkdown(exec.FirstName.String)))
			} else if exec.Role == constants.ROLE_LOADER {
				summaryBuilder.WriteString(fmt.Sprintf(" •  💪 Грузчик: %s\n", utils.EscapeTelegramMarkdown(exec.FirstName.String)))
			}
		}
		summaryBuilder.WriteString("\n")
	}

	header := fmt.Sprintf("📋 *Ваш Заказ №%d*", order.ID)
	footer := "Вы можете отследить статус заказа в этом меню или связаться с оператором."

	return fmt.Sprintf("%s\n%s\n%s\n%s\n%s",
		header, separator, summaryBuilder.String(), separator, footer)
}

// FormatOrderDetailsForOperator форматирует сообщение с полной информацией о заказе для оператора.
// Универсален для просмотра и финального подтверждения при создании.
func FormatOrderDetailsForOperator(order models.Order, client models.User, assignedExecutors []models.Executor, title string, footer string) string {
	var summaryBuilder strings.Builder

	// --- Блок "Статус" (если есть) ---
	if order.Status != "" {
		summaryBuilder.WriteString(fmt.Sprintf("⚙️ *Статус:* %s\n\n", utils.EscapeTelegramMarkdown(constants.StatusDisplayMap[order.Status])))
	}

	// --- Блок "Клиент" ---
	summaryBuilder.WriteString("👤 *КЛИЕНТ:*\n")
	summaryBuilder.WriteString(fmt.Sprintf(" •  Имя: %s\n", utils.EscapeTelegramMarkdown(client.FirstName)))
	summaryBuilder.WriteString(fmt.Sprintf(" •  Телефон: `%s`\n", utils.EscapeTelegramMarkdown(client.Phone.String)))
	summaryBuilder.WriteString(fmt.Sprintf(" •  ChatID: `%d`\n", client.ChatID))
	summaryBuilder.WriteString("\n")

	// --- Блок "Детали Заказа" ---
	displaySubcategory := utils.GetDisplaySubcategory(order)
	formattedDate, _ := utils.FormatDateForDisplay(order.Date)
	timeStr := order.Time
	if timeStr == "" {
		timeStr = "в ближайшее время"
	}
	summaryBuilder.WriteString("📋 *ДЕТАЛИ ЗАКАЗА:*\n")
	summaryBuilder.WriteString(fmt.Sprintf(" •  Услуга: %s (%s)\n",
		utils.EscapeTelegramMarkdown(constants.CategoryDisplayMap[order.Category]),
		utils.EscapeTelegramMarkdown(displaySubcategory)))
	summaryBuilder.WriteString(fmt.Sprintf(" •  Адрес: %s\n", utils.EscapeTelegramMarkdown(order.Address)))
	summaryBuilder.WriteString(fmt.Sprintf(" •  Дата и время: %s, %s\n",
		utils.EscapeTelegramMarkdown(formattedDate),
		utils.EscapeTelegramMarkdown(timeStr)))
	if order.Description != "" {
		summaryBuilder.WriteString(fmt.Sprintf(" •  Описание: %s\n", utils.EscapeTelegramMarkdown(order.Description)))
	}
	if len(order.Photos) > 0 || len(order.Videos) > 0 {
		summaryBuilder.WriteString(fmt.Sprintf(" •  Медиа: %d фото, %d видео\n", len(order.Photos), len(order.Videos)))
	}
	summaryBuilder.WriteString("\n")

	// --- Блок "Финансы" ---
	paymentStr := "По выполнению"
	if order.Payment == "now" {
		paymentStr = "Сразу (скидка 5%)"
	}
	costDisplay := "не установлена"
	if order.Cost.Valid && order.Cost.Float64 > 0 {
		costDisplay = fmt.Sprintf("%.0f ₽", order.Cost.Float64)
	}
	summaryBuilder.WriteString("💰 *ФИНАНСЫ:*\n")
	summaryBuilder.WriteString(fmt.Sprintf(" •  Стоимость: *%s*\n", utils.EscapeTelegramMarkdown(costDisplay)))
	summaryBuilder.WriteString(fmt.Sprintf(" •  Оплата: %s\n", utils.EscapeTelegramMarkdown(paymentStr)))
	summaryBuilder.WriteString("\n")

	// --- Блок "Исполнители" ---
	summaryBuilder.WriteString("👷 *НАЗНАЧЕННЫЕ ИСПОЛНИТЕЛИ:*\n")
	if len(assignedExecutors) == 0 {
		summaryBuilder.WriteString(" •  _Не назначены_\n")
	} else {
		for _, exec := range assignedExecutors {
			roleEmoji := "❓"
			if exec.Role == constants.ROLE_DRIVER {
				roleEmoji = "🚚"
			} else if exec.Role == constants.ROLE_LOADER {
				roleEmoji = "💪"
			}
			summaryBuilder.WriteString(fmt.Sprintf(" •  %s %s\n", roleEmoji, utils.EscapeTelegramMarkdown(utils.GetUserDisplayName(models.User{
				FirstName: exec.FirstName.String,
				LastName:  exec.LastName.String,
			}))))
		}
	}

	return fmt.Sprintf("%s\n%s\n%s\n%s\n%s",
		title, separator, summaryBuilder.String(), separator, footer)
}

// FormatTaskForExecutor форматирует сообщение с заданием для водителя или грузчика.
// Скрывает финансовую информацию.
func FormatTaskForExecutor(order models.Order, client models.User, brigade []models.Executor) string {
	var summaryBuilder strings.Builder

	// --- Блок "Клиент" ---
	summaryBuilder.WriteString("👤 *КЛИЕНТ (для связи):*\n")
	summaryBuilder.WriteString(fmt.Sprintf(" •  Имя: %s\n", utils.EscapeTelegramMarkdown(order.Name))) // Имя из заказа
	summaryBuilder.WriteString(fmt.Sprintf(" •  Телефон: `%s`\n", utils.EscapeTelegramMarkdown(order.Phone)))
	summaryBuilder.WriteString("\n")

	// --- Блок "Детали заказа" ---
	displaySubcategory := utils.GetDisplaySubcategory(order)
	formattedDate, _ := utils.FormatDateForDisplay(order.Date)
	timeStr := order.Time
	if timeStr == "" {
		timeStr = "в ближайшее время"
	}
	summaryBuilder.WriteString("📋 *ДЕТАЛИ ЗАКАЗА:*\n")
	summaryBuilder.WriteString(fmt.Sprintf(" •  Услуга: %s (%s)\n",
		utils.EscapeTelegramMarkdown(constants.CategoryDisplayMap[order.Category]),
		utils.EscapeTelegramMarkdown(displaySubcategory)))
	summaryBuilder.WriteString(fmt.Sprintf(" •  Адрес: %s\n", utils.EscapeTelegramMarkdown(order.Address)))
	summaryBuilder.WriteString(fmt.Sprintf(" •  Дата и время: %s, %s\n",
		utils.EscapeTelegramMarkdown(formattedDate),
		utils.EscapeTelegramMarkdown(timeStr)))
	if order.Description != "" {
		summaryBuilder.WriteString(fmt.Sprintf(" •  Описание: %s\n", utils.EscapeTelegramMarkdown(order.Description)))
	}
	if len(order.Photos) > 0 || len(order.Videos) > 0 {
		summaryBuilder.WriteString(fmt.Sprintf(" •  Медиа: %d фото, %d видео\n", len(order.Photos), len(order.Videos)))
	}
	summaryBuilder.WriteString("\n")

	// --- Блок "Бригада" ---
	summaryBuilder.WriteString("👷 *ВАША БРИГАДА:*\n")
	if len(brigade) == 0 {
		summaryBuilder.WriteString(" •  _Другие исполнители не назначены_\n")
	} else {
		for _, exec := range brigade {
			roleEmoji := "❓"
			if exec.Role == constants.ROLE_DRIVER {
				roleEmoji = "🚚"
			} else if exec.Role == constants.ROLE_LOADER {
				roleEmoji = "💪"
			}
			summaryBuilder.WriteString(fmt.Sprintf(" •  %s %s\n", roleEmoji, utils.EscapeTelegramMarkdown(utils.GetUserDisplayName(models.User{
				FirstName: exec.FirstName.String,
				LastName:  exec.LastName.String,
			}))))
		}
	}

	header := fmt.Sprintf("🛠️ *Новое Задание по Заказу №%d*", order.ID)
	footer := "Пожалуйста, подтвердите получение уведомления, нажав на кнопку ниже."

	return fmt.Sprintf("%s\n%s\n%s\n%s\n%s",
		header, separator, summaryBuilder.String(), separator, footer)
}
