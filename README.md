# 🚀 Сервис-Крым Telegram Bot

## Быстрый запуск

### 1. Настройка конфигурации
Отредактируйте файл `config.env` с вашими данными:
```bash
TELEGRAM_APITOKEN=ваш_токен_бота
BOT_USERNAME=ваш_бот_username
DATABASE_URL=postgres://пользователь:пароль@localhost:5432/база_данных
STORAGE_CHANNEL_ID=-1001234567890
OWNER_CHAT_ID=123456789
```

### 2. Запуск сервера
```bash
chmod +x start.sh
./start.sh
```

Или вручную:
```bash
go build -o main main.go
./main
```

### 3. Проверка работы
- Веб-приложение: http://localhost:8080/webapp/
- API: http://localhost:8080/api/client-config

## 🔧 Решение проблемы INIT_FAILED

Если веб-приложение показывает ошибку "INIT_FAILED", проверьте:

1. **Переменные окружения установлены** - файл `config.env` существует и содержит правильные данные
2. **PostgreSQL запущен** - `systemctl status postgresql`  
3. **Сервер работает** - проверьте логи в `server.log`

### Основные переменные окружения:
- `TELEGRAM_APITOKEN` - токен бота от @BotFather
- `DATABASE_URL` - строка подключения к PostgreSQL
- `STORAGE_CHANNEL_ID` - ID канала для хранения файлов
- `BOT_USERNAME` - имя пользователя бота

## 📁 Структура проекта

```
├── main.go                 # Основной файл сервера
├── config.env              # Конфигурация окружения  
├── start.sh                # Скрипт запуска
├── webapp/                 # Веб-приложение
│   ├── index.html          # Главная страница
│   ├── app.js              # Основной JS код
│   └── style.css           # Стили
└── internal/               # Внутренние модули
    ├── config/             # Конфигурация
    ├── api/                # API endpoints
    └── db/                 # База данных
``` 