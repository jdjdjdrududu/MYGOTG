package main

import (
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	tgbotapi "github.com/OvyFlash/telegram-bot-api"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/joho/godotenv"

	"Original/internal/api"
	"Original/internal/config"
	"Original/internal/db"
	"Original/internal/handlers"
	"Original/internal/session"
	"Original/internal/telegram_api"
	"Original/internal/utils"
)

func main() {
	// --- Блок инициализации ---
	err := godotenv.Load()
	if err != nil {
		log.Println("Предупреждение: не удалось загрузить файл .env. Переменные окружения должны быть установлены иным способом.")
	}

	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Критическая ошибка: не удалось загрузить конфигурацию: %v", err)
	}

	if err := utils.InitEncryptionKey(); err != nil {
		log.Fatalf("Критическая ошибка: не удалось инициализировать ключ шифрования: %v", err)
	}

	if err := db.InitDB(); err != nil {
		log.Fatalf("Критическая ошибка: не удалось инициализировать базу данных: %v", err)
	}
	defer db.CloseDB()

	err = telegram_api.InitBot(cfg.TelegramToken, cfg.AppEnv == "dev", cfg.BotUsername)
	if err != nil {
		log.Fatalf("Критическая ошибка: не удалось инициализировать Telegram бота: %v", err)
	}

	if telegram_api.Client == nil || telegram_api.Client.GetAPI() == nil {
		log.Fatalf("Критическая ошибка: Telegram API клиент не был корректно инициализирован.")
	}
	botAPI := telegram_api.Client.GetAPI()

	sessionManager := session.NewSessionManager()

	handlerDeps := handlers.HandlerDependencies{
		Config:         cfg,
		BotClient:      telegram_api.Client,
		SessionManager: sessionManager,
	}

	botHandler := handlers.NewBotHandler(handlerDeps)

	// --- Настройка роутера и Middleware ---
	apiRouter := chi.NewRouter()

	// ГЛОБАЛЬНЫЕ MIDDLEWARES ДОЛЖНЫ ИДТИ ПЕРЕД api.SetupRoutes
	apiRouter.Use(middleware.Logger)
	apiRouter.Use(middleware.Recoverer)
	apiRouter.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"https://*", "http://*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-Telegram-Auth"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	// Создаем зависимости для API
	apiDeps := api.ApiDependencies{
		Config:    cfg,
		SecretKey: cfg.TelegramToken,
		Bot:       botHandler, // <-- ДОБАВЛЕНО: Передаем botHandler в зависимости API
	}

	// Теперь вызываем SetupRoutes
	api.SetupRoutes(apiRouter, apiDeps)

	// Маршруты для статики и редиректы могут идти здесь, так как они не используют r.Use для глобальных middlewares.
	apiRouter.Get("/", http.RedirectHandler("/webapp/", http.StatusMovedPermanently).ServeHTTP)

	// Обработка запроса иконки, чтобы избежать ошибки 404 в логах
	apiRouter.Get("/favicon.ico", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	// Настройка файлового сервера для WebApp
	workDir, _ := os.Getwd()
	filesDir := http.Dir(filepath.Join(workDir, "webapp"))
	FileServer(apiRouter, "/webapp", filesDir)

	// Запускаем HTTP-сервер в отдельной горутине
	go func() {
		port := os.Getenv("PORT")
		if port == "" {
			port = "8080"
		}
		log.Printf("Запуск HTTP-сервера для WebApp API на порту %s", port)
		if err := http.ListenAndServe(":"+port, apiRouter); err != nil {
			log.Fatalf("КРИТИЧЕСКАЯ ОШИБКА: не удалось запустить HTTP-сервер: %v", err)
		}
	}()

	// Запуск самого бота
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	updates := botAPI.GetUpdatesChan(u)

	log.Println("Бот и API-сервер запущены и готовы к работе...")

	for update := range updates {
		if update.Message != nil {
			log.Printf("[%s] %s", update.Message.From.UserName, update.Message.Text)
			go botHandler.HandleMessage(update)
		} else if update.CallbackQuery != nil {
			log.Printf("Callback от %s: %s", update.CallbackQuery.From.UserName, update.CallbackQuery.Data)
			go botHandler.HandleCallback(update)
		}
	}
}

// FileServer для обслуживания статичных файлов
func FileServer(r chi.Router, path string, root http.FileSystem) {
	if strings.ContainsAny(path, "{}*") {
		panic("FileServer не поддерживает шаблоны URL")
	}

	if path != "/" && path[len(path)-1] != '/' {
		r.Get(path, http.RedirectHandler(path+"/", http.StatusMovedPermanently).ServeHTTP)
		path += "/"
	}
	path += "*"

	r.Get(path, func(w http.ResponseWriter, r *http.Request) {
		rctx := chi.RouteContext(r.Context())
		pathPrefix := strings.TrimSuffix(rctx.RoutePattern(), "/*")
		fs := http.StripPrefix(pathPrefix, http.FileServer(root))
		fs.ServeHTTP(w, r)
	})
}

// GracefulShutdown (без изменений)
func GracefulShutdown(signals chan os.Signal, done chan bool, bot *tgbotapi.BotAPI) {
	// ... ваш код
}
