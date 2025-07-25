# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build and Development Commands

### Building and Running
```bash
# Build the application
go build -o main main.go

# Run the application directly
./main

# Or build and run in one command
go run main.go
```

### Configuration Setup
The application requires environment variables set via `config.env` file:
- `TELEGRAM_APITOKEN` - Bot token from @BotFather
- `DATABASE_URL` - PostgreSQL connection string (postgres://user:password@host:port/dbname)
- `STORAGE_CHANNEL_ID` - Telegram channel ID for file storage
- `BOT_USERNAME` - Bot username
- `OWNER_CHAT_ID` - Owner's Telegram chat ID

### Development Server
- Web application runs on: http://localhost:8080/webapp/
- API endpoints available at: http://localhost:8080/api/*
- Port can be overridden with `PORT` environment variable

## Architecture Overview

This is a Telegram bot service with an integrated web application for managing orders, staff, and financial operations. The architecture follows a clean separation of concerns:

### Core Components

**Main Entry Point** (`main.go`):
- Initializes configuration, database, encryption, and Telegram bot
- Sets up HTTP server with Chi router for web API
- Manages concurrent handling of Telegram updates and HTTP requests

**Configuration Layer** (`internal/config/`):
- Centralized configuration management from environment variables
- Parses DATABASE_URL into individual DB connection parameters
- Handles optional parameters with sensible defaults

**Database Layer** (`internal/db/`):
- PostgreSQL database operations for all entities
- Separate files for each business domain: orders, users, expenses, payouts, etc.
- Uses lib/pq PostgreSQL driver

**Telegram Integration** (`internal/telegram_api/`):
- Wrapper around OvyFlash/telegram-bot-api library
- Manages bot client lifecycle and messaging operations
- Handles file uploads to storage channel

**Session Management** (`internal/session/`):
- Manages temporary user states during multi-step operations
- Handles temporary order creation and driver settlement processes

**Business Logic Handlers** (`internal/handlers/`):
- Menu handlers for different user roles (admin, driver, owner)
- Callback handlers for inline keyboard interactions
- Message processing and payment handling
- Organized by functional areas: orders, staff, financials, etc.

**Web API** (`internal/api/`):
- REST API endpoints for web application
- Media proxy for file serving
- Authentication middleware using Telegram authentication
- CORS configuration for web app integration

**Models** (`internal/models/`):
- Data structures for all business entities
- Custom JSON/SQL types for complex data fields
- Database schema representations

### Key Architectural Patterns

1. **Dependency Injection**: Handlers receive dependencies (config, bot client, session manager) through constructor pattern
2. **Concurrent Processing**: Telegram updates and HTTP requests handled in separate goroutines
3. **Stateful Sessions**: User interaction state maintained between message exchanges
4. **File Storage**: Media files stored in Telegram channels and served via web proxy
5. **Dual Interface**: Both Telegram bot commands and web interface for the same operations

### Web Application (`webapp/`)
- Single-page application with modular JavaScript architecture
- Components organized in `js/components/` and `js/modules/`
- API integration through `js/services/api.js`
- Lottie animations for user experience
- Unified CSS with Telegram Web App styling

### Critical Dependencies
- OvyFlash/telegram-bot-api for Telegram bot functionality
- Chi router for HTTP routing with CORS support
- PostgreSQL with lib/pq driver
- UUID generation for unique identifiers
- Excel file generation via xuri/excelize