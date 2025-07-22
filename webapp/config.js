/**
 * Сервис-Крым Web App Configuration
 * Modern configuration system with environment support
 */

// Base configuration
window.APP_CONFIG = {
    // Application Info
    APP_NAME: 'Сервис-Крым',
    VERSION: '3.0',
    BUILD_DATE: new Date().toISOString(),
    
    // API Configuration [[memory:3922676]]
    API_BASE: window.location.origin,
    API_VERSION: 'v1',
    API_TIMEOUT: 30000, // 30 seconds
    
    // Authentication
    AUTH_HEADER: 'X-Telegram-Auth',
    AUTH_FALLBACK_ENABLED: true,
    
    // Real-time Updates
    REFRESH_INTERVAL: 30000, // 30 seconds
    REALTIME_ENABLED: true,
    
    // UI Settings
    THEME: 'dark', // 'dark' | 'light' | 'auto'
    ANIMATIONS_ENABLED: true,
    NOTIFICATIONS_TIMEOUT: 5000,
    MODAL_ANIMATION_DURATION: 300,
    
    // Performance
    CACHE_TTL: 300000, // 5 minutes
    LAZY_LOADING: true,
    VIRTUALIZATION: true,
    DEBOUNCE_DELAY: 300,
    
    // Development
    DEBUG: false,
    VERBOSE_LOGGING: false,
    MOCK_DATA: false,
    
    // Features
    FEATURES: {
        PUSH_NOTIFICATIONS: true,
        OFFLINE_MODE: true,
        DRAG_AND_DROP: true,
        VOICE_MESSAGES: true,
        GEOLOCATION: true,
        CAMERA_UPLOAD: true
    },
    
    // Limits
    MAX_FILE_SIZE: 10 * 1024 * 1024, // 10MB
    MAX_FILES_COUNT: 10,
    MAX_MESSAGE_LENGTH: 1000,
    
    // Endpoints
    ENDPOINTS: {
        // User endpoints
        USER_PROFILE: '/api/user/profile',
        USER_ORDERS: '/api/user/orders',
        USER_CREATE_ORDER: '/api/user/create-order',
        USER_ORDER_DETAILS: '/api/user/order',
        USER_ORDER_ACTION: '/api/user/order',
        
        // Admin endpoints
        ADMIN_ORDERS: '/api/admin/orders',
        ADMIN_CLIENTS: '/api/admin/clients',
        ADMIN_CREATE_ORDER: '/api/admin/create-order',
        ADMIN_ORDER_DETAILS: '/api/admin/order',
        ADMIN_ORDER_ACTION: '/api/admin/order',
        ADMIN_UPDATE_FIELD: '/api/admin/order',
        
        // Common endpoints
        UPLOAD_MEDIA: '/api/upload-media',
        MEDIA_PROXY: '/api/media',
        CLIENT_CONFIG: '/api/client-config',
        
        // Test endpoints (development only)
        TEST_PROFILE: '/api/test/profile'
    },
    
    // Error Messages
    MESSAGES: {
        NETWORK_ERROR: 'Ошибка сети. Проверьте подключение к интернету.',
        AUTH_ERROR: 'Ошибка авторизации. Перезапустите приложение.',
        SERVER_ERROR: 'Ошибка сервера. Попробуйте позже.',
        UNKNOWN_ERROR: 'Произошла неизвестная ошибка.',
        LOADING_ERROR: 'Ошибка загрузки данных.',
        SAVE_ERROR: 'Ошибка сохранения данных.',
        PERMISSION_ERROR: 'Недостаточно прав для выполнения операции.'
    },
    
    // Status mappings
    ORDER_STATUSES: {
        'new': { text: 'Новый', class: 'status-new', color: '#ff9500' },
        'awaiting_cost': { text: 'Ожидает оценки', class: 'status-new', color: '#ff9500' },
        'awaiting_confirmation': { text: 'Ожидает подтверждения', class: 'status-warning', color: '#ff9500' },
        'awaiting_payment': { text: 'Ожидает оплаты', class: 'status-warning', color: '#ff9500' },
        'inprogress': { text: 'В работе', class: 'status-inprogress', color: '#007aff' },
        'completed': { text: 'Выполнен', class: 'status-completed', color: '#34c759' },
        'calculated': { text: 'Рассчитан', class: 'status-completed', color: '#34c759' },
        'settled': { text: 'Оплачен', class: 'status-completed', color: '#34c759' },
        'canceled': { text: 'Отменен', class: 'status-canceled', color: '#ff3b30' }
    },
    
    // User roles
    USER_ROLES: {
        'user': { text: 'Клиент', permissions: ['view_own_orders', 'create_order'] },
        'operator': { text: 'Оператор', permissions: ['view_all_orders', 'manage_orders', 'view_clients'] },
        'admin': { text: 'Администратор', permissions: ['view_all_orders', 'manage_orders', 'view_clients', 'manage_users'] },
        'owner': { text: 'Владелец', permissions: ['full_access'] },
        'driver': { text: 'Водитель', permissions: ['view_assigned_orders', 'update_order_status'] }
    }
};

// Environment detection and configuration override
(function initializeConfig() {
    const hostname = window.location.hostname;
    const isLocal = hostname === 'localhost' || hostname === '127.0.0.1';
    const isDev = hostname.includes('dev') || hostname.includes('test');
    
    // Development environment
    if (isLocal) {
        Object.assign(window.APP_CONFIG, {
            DEBUG: true,
            VERBOSE_LOGGING: true,
            MOCK_DATA: false, // Set to true if you want to use mock data
            API_BASE: 'http://localhost:8080',
            REFRESH_INTERVAL: 10000 // Faster refresh in development
        });
        console.log('🔧 Development mode enabled');
    }
    
    // Testing environment
    if (isDev && !isLocal) {
        Object.assign(window.APP_CONFIG, {
            DEBUG: true,
            VERBOSE_LOGGING: true,
            REFRESH_INTERVAL: 15000
        });
        console.log('🧪 Testing mode enabled');
    }
    
    // Production optimizations
    if (!isLocal && !isDev) {
        Object.assign(window.APP_CONFIG, {
            DEBUG: false,
            VERBOSE_LOGGING: false,
            CACHE_TTL: 600000 // 10 minutes in production
        });
        console.log('🚀 Production mode enabled');
    }
    
    // Override with server-provided config if available
    if (window.SERVER_CONFIG) {
        Object.assign(window.APP_CONFIG, window.SERVER_CONFIG);
        console.log('📡 Server configuration applied');
    }
    
    // Telegram WebApp theme detection
    if (window.Telegram?.WebApp) {
        const tg = window.Telegram.WebApp;
        if (tg.colorScheme) {
            window.APP_CONFIG.THEME = tg.colorScheme;
        }
        
        // Apply Telegram theme colors
        if (tg.themeParams) {
            document.documentElement.style.setProperty('--tg-bg-color', tg.themeParams.bg_color || '#1a1a1a');
            document.documentElement.style.setProperty('--tg-text-color', tg.themeParams.text_color || '#ffffff');
            document.documentElement.style.setProperty('--tg-hint-color', tg.themeParams.hint_color || '#999999');
            document.documentElement.style.setProperty('--tg-link-color', tg.themeParams.link_color || '#007aff');
            document.documentElement.style.setProperty('--tg-button-color', tg.themeParams.button_color || '#007aff');
            document.documentElement.style.setProperty('--tg-button-text-color', tg.themeParams.button_text_color || '#ffffff');
        }
    }
    
    // Log final configuration in debug mode
    if (window.APP_CONFIG.DEBUG) {
        console.log('⚙️ Final App Configuration:', window.APP_CONFIG);
    }
})();

// Utility functions for working with configuration
window.AppUtils = {
    /**
     * Get API endpoint URL
     * @param {string} endpoint - Endpoint key from ENDPOINTS config
     * @param {string|number} id - Optional ID to append to endpoint
     * @returns {string} Full API URL
     */
    getApiUrl(endpoint, id = null) {
        const baseUrl = window.APP_CONFIG.API_BASE;
        const endpointPath = window.APP_CONFIG.ENDPOINTS[endpoint];
        
        if (!endpointPath) {
            console.warn(`Unknown endpoint: ${endpoint}`);
            return `${baseUrl}/api/unknown`;
        }
        
        const fullPath = id ? `${endpointPath}/${id}` : endpointPath;
        return `${baseUrl}${fullPath}`;
    },
    
    /**
     * Check if user has permission
     * @param {string} permission - Permission to check
     * @param {object} user - User object
     * @returns {boolean} Has permission
     */
    hasPermission(permission, user) {
        if (!user || !user.Role) return false;
        
        const roleConfig = window.APP_CONFIG.USER_ROLES[user.Role];
        if (!roleConfig) return false;
        
        return roleConfig.permissions.includes(permission) || 
               roleConfig.permissions.includes('full_access');
    },
    
    /**
     * Get order status configuration
     * @param {string} status - Order status
     * @returns {object} Status configuration
     */
    getOrderStatus(status) {
        return window.APP_CONFIG.ORDER_STATUSES[status] || {
            text: status,
            class: 'status-unknown',
            color: '#999999'
        };
    },
    
    /**
     * Get user role configuration
     * @param {string} role - User role
     * @returns {object} Role configuration
     */
    getUserRole(role) {
        return window.APP_CONFIG.USER_ROLES[role] || {
            text: role,
            permissions: []
        };
    },
    
    /**
     * Log message if verbose logging is enabled
     * @param {string} message - Message to log
     * @param {any} data - Optional data to log
     */
    log(message, data = null) {
        if (window.APP_CONFIG.VERBOSE_LOGGING) {
            if (data) {
                console.log(`[ServiceApp] ${message}`, data);
            } else {
                console.log(`[ServiceApp] ${message}`);
            }
        }
    },
    
    /**
     * Format currency value
     * @param {number} value - Value to format
     * @returns {string} Formatted currency
     */
    formatCurrency(value) {
        if (!value || isNaN(value)) return 'Не указано';
        return new Intl.NumberFormat('ru-RU', {
            style: 'currency',
            currency: 'RUB',
            minimumFractionDigits: 0,
            maximumFractionDigits: 0
        }).format(value);
    },
    
    /**
     * Format date
     * @param {string|Date} date - Date to format
     * @returns {string} Formatted date
     */
    formatDate(date) {
        if (!date) return 'Не указано';
        
        const d = new Date(date);
        if (isNaN(d.getTime())) return 'Некорректная дата';
        
        return new Intl.DateTimeFormat('ru-RU', {
            year: 'numeric',
            month: 'short',
            day: 'numeric',
            hour: '2-digit',
            minute: '2-digit'
        }).format(d);
    },
    
    /**
     * Format phone number
     * @param {string} phone - Phone number to format
     * @returns {string} Formatted phone
     */
    formatPhone(phone) {
        if (!phone) return 'Не указан';
        
        // Remove all non-digits
        const digits = phone.replace(/\D/g, '');
        
        // Format as +7 (XXX) XXX-XX-XX for Russian numbers
        if (digits.length === 11 && digits.startsWith('7')) {
            return `+7 (${digits.slice(1, 4)}) ${digits.slice(4, 7)}-${digits.slice(7, 9)}-${digits.slice(9)}`;
        }
        
        return phone;
    },
    
    /**
     * Debounce function to limit the rate of function calls
     * @param {Function} func - Function to debounce
     * @param {number} wait - Wait time in milliseconds
     * @returns {Function} Debounced function
     */
    debounce(func, wait) {
        let timeout;
        return function executedFunction(...args) {
            const later = () => {
                clearTimeout(timeout);
                func(...args);
            };
            clearTimeout(timeout);
            timeout = setTimeout(later, wait);
        };
    },
    
    /**
     * Throttle function to limit the rate of function calls
     * @param {Function} func - Function to throttle
     * @param {number} limit - Time limit in milliseconds
     * @returns {Function} Throttled function
     */
    throttle(func, limit) {
        let inThrottle;
        return function(...args) {
            if (!inThrottle) {
                func.apply(this, args);
                inThrottle = true;
                setTimeout(() => inThrottle = false, limit);
            }
        };
    }
};

// Export configuration for modules
if (typeof module !== 'undefined' && module.exports) {
    module.exports = window.APP_CONFIG;
}

console.log('📋 Сервис-Крым Configuration loaded successfully'); 