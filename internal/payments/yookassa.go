package payments

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/google/uuid"
)

// API-адрес YooKassa
const yooKassaAPIEndpoint = "https://api.yookassa.ru/v3/payments"

// Определяем структуры для запроса и ответа прямо здесь.

// --- НАЧАЛО ИЗМЕНЕНИЯ: Добавлены структуры для чека ---

// Receipt представляет структуру фискального чека.
type Receipt struct {
	Customer Customer      `json:"customer"`
	Items    []ReceiptItem `json:"items"`
}

// Customer представляет данные о покупателе.
type Customer struct {
	Phone string `json:"phone,omitempty"`
	Email string `json:"email,omitempty"`
}

// ReceiptItem представляет товарную позицию в чеке.
type ReceiptItem struct {
	Description string `json:"description"`
	Quantity    string `json:"quantity"`
	Amount      Amount `json:"amount"`
	VATCode     int    `json:"vat_code"` // Код ставки НДС. 1 = без НДС.
}

// --- КОНЕЦ ИЗМЕНЕНИЯ ---

// PaymentRequest - структура запроса на создание платежа.
type PaymentRequest struct {
	Amount       Amount          `json:"amount"`
	Confirmation Confirmation    `json:"confirmation"`
	Description  string          `json:"description"`
	Capture      bool            `json:"capture"`
	Metadata     json.RawMessage `json:"metadata,omitempty"`
	Receipt      *Receipt        `json:"receipt,omitempty"` // --- ИЗМЕНЕНИЕ: Добавлено поле для чека ---
}

// Amount - сумма платежа.
type Amount struct {
	Value    string `json:"value"`
	Currency string `json:"currency"`
}

// Confirmation - способ подтверждения платежа.
type Confirmation struct {
	Type      string `json:"type"`
	ReturnURL string `json:"return_url"`
}

// PaymentResponse - структура ответа от API YooKassa.
type PaymentResponse struct {
	ID           string               `json:"id"`
	Status       string               `json:"status"`
	Paid         bool                 `json:"paid"`
	Amount       Amount               `json:"amount"`
	Confirmation ConfirmationResponse `json:"confirmation"`
	CreatedAt    time.Time            `json:"created_at"`
	Description  string               `json:"description"`
	Metadata     json.RawMessage      `json:"metadata"`
	Test         bool                 `json:"test"`
}

// ConfirmationResponse - содержит URL для подтверждения платежа пользователем.
type ConfirmationResponse struct {
	Type            string `json:"type"`
	ConfirmationURL string `json:"confirmation_url"`
}

// CreatePaymentLink - основная функция, которая создает платежную ссылку.
// --- НАЧАЛО ИЗМЕНЕНИЯ: Добавлен параметр clientPhone ---
func CreatePaymentLink(shopID, secretKey string, orderID int64, amountValue float64, currency, description, returnURL, clientPhone string) (string, error) {
	// --- КОНЕЦ ИЗМЕНЕНИЯ ---
	log.Printf("Создание платежной ссылки через прямой API-запрос для заказа #%d", orderID)

	// 1. Формируем тело JSON-запроса.
	metadata, _ := json.Marshal(map[string]string{
		"order_id": fmt.Sprintf("%d", orderID),
	})

	// --- НАЧАЛО ИЗМЕНЕНИЯ: Создаем объект чека ---
	receipt := &Receipt{
		Customer: Customer{
			Phone: clientPhone, // Используем номер телефона клиента
		},
		Items: []ReceiptItem{
			{
				Description: description, // Описание услуги
				Quantity:    "1.00",
				Amount: Amount{
					Value:    fmt.Sprintf("%.2f", amountValue),
					Currency: currency,
				},
				VATCode: 1, // 1 = НДС не облагается
			},
		},
	}
	// --- КОНЕЦ ИЗМЕНЕНИЯ ---

	requestBody := PaymentRequest{
		Amount: Amount{
			Value:    fmt.Sprintf("%.2f", amountValue),
			Currency: currency,
		},
		Confirmation: Confirmation{
			Type:      "redirect",
			ReturnURL: returnURL,
		},
		Description: description,
		Capture:     true,
		Metadata:    metadata,
		Receipt:     receipt, // --- ИЗМЕНЕНИЕ: Добавляем чек в запрос ---
	}

	payload, err := json.Marshal(requestBody)
	if err != nil {
		log.Printf("Ошибка маршалинга запроса к YooKassa: %v", err)
		return "", fmt.Errorf("ошибка подготовки запроса: %w", err)
	}

	// 2. Создаем HTTP-клиент и сам запрос.
	client := &http.Client{Timeout: 15 * time.Second}
	req, err := http.NewRequestWithContext(context.Background(), "POST", yooKassaAPIEndpoint, bytes.NewBuffer(payload))
	if err != nil {
		log.Printf("Ошибка создания HTTP-запроса к YooKassa: %v", err)
		return "", fmt.Errorf("ошибка создания HTTP-запроса: %w", err)
	}

	// 3. Устанавливаем необходимые заголовки.
	req.SetBasicAuth(shopID, secretKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotence-Key", uuid.New().String())

	// 4. Выполняем запрос.
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Ошибка выполнения HTTP-запроса к YooKassa: %v", err)
		return "", fmt.Errorf("ошибка выполнения запроса к API YooKassa: %w", err)
	}
	defer resp.Body.Close()

	// 5. Читаем и обрабатываем ответ.
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Ошибка чтения ответа от API YooKassa: %v", err)
		return "", fmt.Errorf("ошибка чтения ответа API: %w", err)
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		log.Printf("API YooKassa вернул ошибку: статус %d, тело: %s", resp.StatusCode, string(responseBody))
		return "", fmt.Errorf("ошибка API YooKassa, статус: %d", resp.StatusCode)
	}

	var paymentResponse PaymentResponse
	if err := json.Unmarshal(responseBody, &paymentResponse); err != nil {
		log.Printf("Ошибка демаршалинга ответа от API YooKassa: %v", err)
		return "", fmt.Errorf("ошибка обработки ответа API: %w", err)
	}

	// 6. Проверяем и возвращаем ссылку на оплату.
	if paymentResponse.Confirmation.ConfirmationURL == "" {
		log.Println("Критическая ошибка: API YooKassa не вернул ссылку на оплату.")
		return "", fmt.Errorf("API не вернул ссылку на оплату")
	}

	log.Printf("Успешно создан платеж YooKassa ID: %s, статус: %s", paymentResponse.ID, paymentResponse.Status)
	return paymentResponse.Confirmation.ConfirmationURL, nil
}

// YooKassaNotification представляет структуру входящего уведомления от ЮKassa.
type YooKassaNotification struct {
	Type   string          `json:"type"`   // e.g., "notification"
	Event  string          `json:"event"`  // e.g., "payment.succeeded"
	Object PaymentResponse `json:"object"` // Содержит полную информацию о платеже
}
