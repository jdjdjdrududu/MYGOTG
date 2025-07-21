// Файл: internal/db/db.go
package db

import (
	"database/sql"
	"fmt"
	"log"
	"net/url"
	"os"
	"strings" // Добавлен для strings.Contains
	"time"

	_ "github.com/lib/pq" // PostgreSQL driver
)

var DB *sql.DB // Глобальная переменная для хранения подключения к БД

// InitDB инициализирует соединение с базой данных и выполняет миграции.
func InitDB() error {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		return fmt.Errorf("DATABASE_URL не установлена")
	}

	// Parse the DATABASE_URL
	parsedURL, err := url.Parse(dbURL)
	if err != nil {
		return fmt.Errorf("ошибка парсинга DATABASE_URL: %v", err)
	}

	query := parsedURL.Query()
	// Пример: query.Set("sslmode", "require")
	// Если вы используете Yandex Cloud CA.pem, убедитесь, что путь к нему корректен
	// или он находится в ожидаемом месте.
	// query.Set("sslrootcert", "CA.pem") // Раскомментируйте и настройте, если необходимо
	parsedURL.RawQuery = query.Encode()
	finalURL := parsedURL.String()

	// Open the database connection
	DB, err = sql.Open("postgres", finalURL)
	if err != nil {
		return fmt.Errorf("ошибка подключения к базе данных: %v", err)
	}

	// Set connection pool settings
	DB.SetMaxOpenConns(50)
	DB.SetMaxIdleConns(20)
	DB.SetConnMaxLifetime(5 * time.Minute)

	// Verify connection
	if err := DB.Ping(); err != nil {
		return fmt.Errorf("ошибка проверки соединения с базой данных: %v", err)
	}

	log.Println("Успешное подключение к базе данных.")

	// Step 1: Create tables if they don't exist
	tx, err := DB.Begin()
	if err != nil {
		return fmt.Errorf("ошибка начала транзакции для создания таблиц: %v", err)
	}
	// Defer rollback in case of panic or error later in this function block
	// before commit or successful return.
	// Note: if tx.Commit() is successful, this defer will do nothing.
	// If an error occurs and is returned before tx.Commit(), this defer will execute.
	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			panic(p) // Re-panic after Rollback
		} else if err != nil { // err is the named return variable or a local var shadowing it
			log.Printf("Откат транзакции из-за ошибки: %v", err)
			tx.Rollback()
		}
	}()

	createTablesSQL := `
        CREATE TABLE IF NOT EXISTS users (
            id SERIAL PRIMARY KEY,
            chat_id BIGINT UNIQUE,
            role TEXT,
            first_name VARCHAR(100),
            last_name VARCHAR(100),
            nickname VARCHAR(100) UNIQUE,
            phone VARCHAR(20),
            card_number TEXT,
            is_blocked BOOLEAN DEFAULT FALSE,
            block_reason TEXT,
            block_date TIMESTAMP,
            main_menu_message_id INTEGER DEFAULT 0,
            created_at TIMESTAMP,
            updated_at TIMESTAMP
        );
        CREATE TABLE IF NOT EXISTS orders (
            id SERIAL PRIMARY KEY,
            user_id INTEGER REFERENCES users(id),
            user_chat_id BIGINT,
            category TEXT,
            subcategory TEXT,
            name TEXT,
            photos TEXT[],
            videos TEXT[],
            date DATE,
            time TEXT,
            phone TEXT,
            address TEXT,
            description TEXT,
            status TEXT,
            reason TEXT,
            cost FLOAT,
            payment TEXT,
            latitude FLOAT,
            longitude FLOAT,
            created_at TIMESTAMP,
            updated_at TIMESTAMP,
            is_driver_settled BOOLEAN DEFAULT FALSE
        );
        CREATE TABLE IF NOT EXISTS expenses (
            id SERIAL PRIMARY KEY,
            order_id INTEGER REFERENCES orders(id) UNIQUE,
            driver_id INTEGER REFERENCES users(id),
            fuel FLOAT,
            other FLOAT,
            loader_salaries JSONB DEFAULT '{}',
            revenue FLOAT,
            driver_share FLOAT,
            created_at TIMESTAMP,
            updated_at TIMESTAMP
        );
        CREATE TABLE IF NOT EXISTS executors (
            id SERIAL PRIMARY KEY,
            order_id INTEGER REFERENCES orders(id),
            user_id INTEGER REFERENCES users(id),
            role TEXT CHECK (role IN ('driver', 'loader')),
            confirmed BOOLEAN DEFAULT FALSE,
            is_notified BOOLEAN DEFAULT FALSE, -- НОВОЕ ПОЛЕ
            created_at TIMESTAMP,
            updated_at TIMESTAMP,
            UNIQUE (order_id, user_id, role)
        );
        CREATE TABLE IF NOT EXISTS referrals (
            id SERIAL PRIMARY KEY,
            inviter_id INTEGER REFERENCES users(id),
            invitee_id INTEGER REFERENCES users(id),
            order_id INTEGER REFERENCES orders(id),
            amount FLOAT,
            paid_out BOOLEAN DEFAULT FALSE,
            payout_request_id INTEGER,
            created_at TIMESTAMP,
            updated_at TIMESTAMP
        );
        CREATE TABLE IF NOT EXISTS referral_payout_requests (
            id SERIAL PRIMARY KEY,
            user_chat_id BIGINT NOT NULL,
            amount FLOAT NOT NULL,
            status TEXT NOT NULL,
            requested_at TIMESTAMP NOT NULL,
            admin_comment TEXT,
            processed_at TIMESTAMP,
            payment_method TEXT,
            payment_details TEXT
        );
        CREATE TABLE IF NOT EXISTS chat_messages (
            id SERIAL PRIMARY KEY,
            user_id INTEGER REFERENCES users(id),
            operator_id INTEGER REFERENCES users(id),
            message TEXT,
            is_from_user BOOLEAN,
            conversation_id TEXT,
            created_at TIMESTAMP
        );
        CREATE TABLE IF NOT EXISTS reviews (
            id SERIAL PRIMARY KEY,
            user_id INTEGER REFERENCES users(id),
            review TEXT,
            created_at TIMESTAMP
        );
        CREATE TABLE IF NOT EXISTS subscriptions (
            id SERIAL PRIMARY KEY,
            user_id INTEGER REFERENCES users(id),
            service TEXT,
            created_at TIMESTAMP,
            updated_at TIMESTAMP,
            UNIQUE(user_id, service)
        );
        CREATE TABLE IF NOT EXISTS payouts (
            id SERIAL PRIMARY KEY,
            user_id INTEGER REFERENCES users(id) NOT NULL,
            amount FLOAT NOT NULL,
            payout_date TIMESTAMP NOT NULL,
            order_id INTEGER REFERENCES orders(id),
            comment TEXT,
            made_by_user_id INTEGER REFERENCES users(id) NOT NULL,
            created_at TIMESTAMP DEFAULT NOW()
        );
          CREATE TABLE IF NOT EXISTS driver_settlements (
            id SERIAL PRIMARY KEY,
            driver_user_id INTEGER REFERENCES users(id) NOT NULL,
            report_date DATE NOT NULL,
            settlement_timestamp TIMESTAMP WITHOUT TIME ZONE DEFAULT NOW(),
            covered_orders_revenue FLOAT NOT NULL,
            fuel_expense FLOAT NOT NULL,
            other_expenses_json JSONB,
            loader_payments_json JSONB,
            driver_calculated_salary FLOAT NOT NULL,
            amount_to_cashier FLOAT NOT NULL,
            covered_orders_count INTEGER,
            created_at TIMESTAMP WITHOUT TIME ZONE DEFAULT NOW(),
            updated_at TIMESTAMP WITHOUT TIME ZONE DEFAULT NOW(),
            covered_order_ids BIGINT[],
            paid_to_owner_at TIMESTAMP WITH TIME ZONE NULL,
            driver_salary_paid_at TIMESTAMP WITH TIME ZONE NULL,
            -- НАЧАЛО ИЗМЕНЕНИЯ --
            status TEXT DEFAULT 'pending' NOT NULL, -- pending, approved, rejected
            admin_comment TEXT
            -- КОНЕЦ ИЗМЕНЕНИЯ --
        );
        CREATE TABLE IF NOT EXISTS owner_cashier_records (
            id SERIAL PRIMARY KEY,
            driver_user_id INTEGER REFERENCES users(id) NOT NULL,
            report_date DATE NOT NULL,
            total_amount_due FLOAT NOT NULL,
            contributing_sett_ids BIGINT[],
            last_updated_at TIMESTAMP WITHOUT TIME ZONE DEFAULT NOW(),
            UNIQUE (driver_user_id, report_date)
        );
    `
	_, err = tx.Exec(createTablesSQL)
	if err != nil {
		// err is already set, so the defer func will rollback
		return fmt.Errorf("ошибка создания таблиц: %v", err)
	}

	err = tx.Commit() // Commit after table creation
	if err != nil {
		// err is already set, so the defer func will rollback (though commit failed)
		return fmt.Errorf("ошибка фиксации транзакции создания таблиц: %v", err)
	}
	log.Println("Создание таблиц (если не существуют) завершено.")

	// Step 2: Perform schema migrations (e.g., adding new columns to existing tables)
	err = migrateDBSchema()
	if err != nil {
		// No transaction to rollback here, as migrateDBSchema handles its own.
		return fmt.Errorf("ошибка выполнения миграции схемы: %v", err)
	}
	log.Println("Миграция схемы базы данных успешно завершена.")

	// Step 3: Create indexes
	// These can be run in a new transaction or as separate statements.
	// Using DB.Exec for simplicity here, as CREATE INDEX IF NOT EXISTS is idempotent.
	// Run these after migrations to ensure columns exist.
	createIndexesSQL := `
        CREATE INDEX IF NOT EXISTS idx_users_chat_id ON users(chat_id);
        CREATE INDEX IF NOT EXISTS idx_orders_user_id_status ON orders(user_id, status);
        CREATE INDEX IF NOT EXISTS idx_orders_created_at ON orders(created_at);
        CREATE INDEX IF NOT EXISTS idx_orders_is_driver_settled ON orders(is_driver_settled);
        CREATE INDEX IF NOT EXISTS idx_executors_order_id ON executors(order_id);
        CREATE INDEX IF NOT EXISTS idx_executors_user_id ON executors(user_id);
        CREATE INDEX IF NOT EXISTS idx_executors_is_notified ON executors(is_notified); -- НОВЫЙ ИНДЕКС
        CREATE INDEX IF NOT EXISTS idx_referrals_inviter_id ON referrals(inviter_id);
        CREATE INDEX IF NOT EXISTS idx_subscriptions_user_id_service ON subscriptions(user_id, service);
        CREATE INDEX IF NOT EXISTS idx_payouts_user_id ON payouts(user_id);
        CREATE INDEX IF NOT EXISTS idx_expenses_order_id ON expenses(order_id);
        CREATE INDEX IF NOT EXISTS idx_chat_messages_conversation_id ON chat_messages(conversation_id);
        CREATE INDEX IF NOT EXISTS idx_driver_settlements_driver_date ON driver_settlements(driver_user_id, report_date);
        CREATE INDEX IF NOT EXISTS idx_driver_settlements_paid_to_owner_at ON driver_settlements(paid_to_owner_at); 
        CREATE INDEX IF NOT EXISTS idx_driver_settlements_salary_paid_at ON driver_settlements(driver_salary_paid_at); 
        CREATE INDEX IF NOT EXISTS idx_owner_cashier_records_driver_date ON owner_cashier_records(driver_user_id, report_date);
    `
	// We'll execute index creation statements one by one to better isolate potential errors
	indexStatements := strings.Split(strings.TrimSpace(createIndexesSQL), ";")
	for _, stmt := range indexStatements {
		trimmedStmt := strings.TrimSpace(stmt)
		if trimmedStmt == "" {
			continue
		}
		_, errIdx := DB.Exec(trimmedStmt)
		if errIdx != nil {
			// Log the error for the specific index, but continue with others
			log.Printf("Предупреждение: ошибка при создании индекса ('%s'): %v. Проверьте логи.", trimmedStmt, errIdx)
			// Optionally, collect errors and return them, or make it fatal:
			// return fmt.Errorf("ошибка создания индекса '%s': %v", trimmedStmt, errIdx)
		}
	}
	log.Println("Создание индексов (если не существуют) завершено.")

	log.Println("Инициализация базы данных успешно завершена.")
	return nil
}

// migrateDBSchema выполняет необходимые миграции схемы базы данных.
// This function should be idempotent.
func migrateDBSchema() error {
	// --- Существующие миграции ---
	// (Эти ALTER TABLE команды должны выполняться после того, как таблицы точно созданы)
	// (These ALTER TABLE commands should run after tables are definitely created)

	migrations := []struct {
		name string
		sql  string
	}{
		{
			name: "driver_settlements.other_expenses_json",
			sql:  `ALTER TABLE driver_settlements ADD COLUMN IF NOT EXISTS other_expenses_json JSONB; ALTER TABLE driver_settlements DROP COLUMN IF EXISTS other_expense;`,
		},
		{
			name: "users.card_number",
			sql: `ALTER TABLE users
                  ADD COLUMN IF NOT EXISTS card_number TEXT;`,
		},
		{
			name: "users.block_reason_date",
			sql: `ALTER TABLE users
                  ADD COLUMN IF NOT EXISTS block_reason TEXT,
                  ADD COLUMN IF NOT EXISTS block_date TIMESTAMP;`,
		},
		{
			name: "referrals.paid_out_payout_request_id",
			sql: `ALTER TABLE referrals
                  ADD COLUMN IF NOT EXISTS paid_out BOOLEAN DEFAULT FALSE,
                  ADD COLUMN IF NOT EXISTS payout_request_id INTEGER;`,
		},
		// Table referral_payout_requests is created in the main block if not exists
		// Table payouts is created in the main block if not exists
		{
			name: "executors.unique_constraint",
			sql: `DO $$
                  BEGIN
                      IF NOT EXISTS (
                          SELECT 1 FROM pg_constraint
                          WHERE conrelid = 'executors'::regclass
                          AND conname = 'executors_order_user_role_key'
                      ) AND EXISTS (
                          SELECT 1 FROM information_schema.tables
                          WHERE table_name = 'executors'
                      ) THEN
                          ALTER TABLE executors ADD CONSTRAINT executors_order_user_role_key UNIQUE (order_id, user_id, role);
                      END IF;
                  END$$;`,
		},
		{
			name: "expenses.updated_at",
			sql: `ALTER TABLE expenses
                  ADD COLUMN IF NOT EXISTS updated_at TIMESTAMP;`,
		},
		{
			name: "expenses.order_id_unique",
			sql: `DO $$
                  BEGIN
                      IF NOT EXISTS (
                          SELECT 1 FROM pg_constraint
                          WHERE conrelid = 'expenses'::regclass
                          AND conname = 'expenses_order_id_key'
                      ) AND EXISTS (
                          SELECT 1 FROM information_schema.tables
                          WHERE table_name = 'expenses'
                      ) THEN
                          ALTER TABLE expenses ADD CONSTRAINT expenses_order_id_key UNIQUE (order_id);
                      END IF;
                  END$$;`,
		},
		{
			name: "orders.is_driver_settled", // This ensures the column is added if the table existed before this column was in CREATE TABLE
			sql: `ALTER TABLE orders
                  ADD COLUMN IF NOT EXISTS is_driver_settled BOOLEAN DEFAULT FALSE;`,
		},
		{
			name: "driver_settlements.covered_order_ids",
			sql: `ALTER TABLE driver_settlements
                  ADD COLUMN IF NOT EXISTS covered_order_ids BIGINT[];`,
		},

		{
			name: "driver_settlements.paid_to_owner_at",
			sql:  `ALTER TABLE driver_settlements ADD COLUMN IF NOT EXISTS paid_to_owner_at TIMESTAMP WITH TIME ZONE NULL;`,
		},

		{
			name: "driver_settlements.driver_salary_paid_at",
			sql:  `ALTER TABLE driver_settlements ADD COLUMN IF NOT EXISTS driver_salary_paid_at TIMESTAMP WITH TIME ZONE NULL;`,
		},
		// НОВАЯ МИГРАЦИЯ для is_notified
		{
			name: "executors.is_notified",
			sql:  `ALTER TABLE executors ADD COLUMN IF NOT EXISTS is_notified BOOLEAN DEFAULT FALSE;`,
		},
		// --- НАЧАЛО НОВОЙ МИГРАЦИИ ---
		{
			name: "driver_settlements.status_and_comment",
			sql: `
                ALTER TABLE driver_settlements ADD COLUMN IF NOT EXISTS status TEXT DEFAULT 'pending' NOT NULL;
                ALTER TABLE driver_settlements ADD COLUMN IF NOT EXISTS admin_comment TEXT;
            `,
		},
		// --- КОНЕЦ НОВОЙ МИГРАЦИИ ---
	}

	for _, migration := range migrations {
		_, err := DB.Exec(migration.sql)
		if err != nil {
			// Handle specific errors like "already exists" gracefully
			// The DO $$ BEGIN ... END$$ blocks already handle "IF NOT EXISTS" for constraints.
			// For ADD COLUMN IF NOT EXISTS, PostgreSQL handles it.
			// So, we mainly check for unexpected errors.
			if strings.Contains(err.Error(), "already exists") ||
				(migration.name == "executors.unique_constraint" && strings.Contains(err.Error(), "duplicate key value violates unique constraint")) ||
				(migration.name == "expenses.order_id_unique" && strings.Contains(err.Error(), "could not create unique index") && strings.Contains(err.Error(), "contains duplicate values")) {
				log.Printf("INFO: Миграция '%s' пропущена (объект уже существует или данные нарушают его). Детали: %v", migration.name, err)
			} else {
				// For "expenses.order_id_unique", check if table exists if other errors occur
				if migration.name == "expenses.order_id_unique" {
					var tableExists bool
					checkTableErr := DB.QueryRow("SELECT EXISTS (SELECT FROM information_schema.tables WHERE table_name = 'expenses')").Scan(&tableExists)
					if checkTableErr == nil && !tableExists {
						log.Printf("INFO: Таблица expenses не существует, пропуск миграции '%s'.", migration.name)
						continue // Skip this migration if table doesn't exist
					}
				}
				return fmt.Errorf("ошибка миграции схемы ('%s'): %v", migration.name, err)
			}
		} else {
			log.Printf("INFO: Миграция ('%s') успешно применена или объект уже существовал.", migration.name)
		}
	}

	log.Println("Миграция схемы базы данных успешно выполнена (или не требовалась).")
	return nil
}

// CloseDB закрывает соединение с базой данных.
func CloseDB() {
	if DB != nil {
		DB.Close()
		log.Println("Соединение с базой данных закрыто.")
	}
}
