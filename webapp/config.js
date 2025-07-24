/**
 * Сервис-Крым Web App Configuration
 * Modern configuration system with environment support
 */

// Determine environment
const ENV = {
    isDev: window.location.hostname === 'localhost' || window.location.hostname === '127.0.0.1',
    isProd: window.location.hostname === 'xn----ctbinlmxece7i.xn--p1ai',
    isTest: window.location.hostname.includes('test') || window.location.hostname.includes('dev')
};

// Base configuration
window.APP_CONFIG = {
    // Application Info
    APP_NAME: 'Сервис-Крым',
    VERSION: '3.0',
    BUILD_DATE: new Date().toISOString(),
    
    // Environment
    ENV: ENV.isDev ? 'development' : ENV.isTest ? 'testing' : 'production',
    DEBUG: false,
    
    // API Configuration - Use absolute URL to avoid webapp path issues
    API_BASE: ENV.isDev ? `http://${window.location.hostname}:8080` :
             ENV.isTest ? 'https://test-api.xn----ctbinlmxece7i.xn--p1ai' :
             (import.meta.env.VITE_APP_API_BASE || ''), // Use environment variable for production
    API_PREFIX: '/api',
    API_TIMEOUT: 30000, // 30 seconds
    
    // Authentication
    AUTH_HEADER: 'X-Telegram-Auth',
    AUTH_FALLBACK_ENABLED: import.meta.env.VITE_APP_AUTH_FALLBACK_ENABLED === 'true', // Use env var
    
    // Push Notifications
    VAPID_PUBLIC_KEY: 'BLBz6HxRFXnGGC1VsK0DFZzwGhqC3kH3v7ZV6W7zP6RUjDVNQFHVh-q_HxF5D5yO_YKxqvzO1eVTXYyXqhLHYqU',
    
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
    
    // Features
    FEATURES: {
        PUSH_NOTIFICATIONS: true,
        OFFLINE_MODE: true,
        DRAG_AND_DROP: true,
        VOICE_MESSAGES: true,
        GEOLOCATION: true,
        CAMERA_UPLOAD: true
    }
};

// Environment-specific overrides
if (ENV.isDev) {
    console.log('🛠️ Development mode enabled');
    Object.assign(window.APP_CONFIG, {
        REFRESH_INTERVAL: 5000, // 5 seconds in dev
        MOCK_DATA: true
    });
} else if (ENV.isTest) {
    console.log('🧪 Testing mode enabled');
    Object.assign(window.APP_CONFIG, {
        REFRESH_INTERVAL: 10000 // 10 seconds in test
    });
} else {
    console.log('🚀 Production mode enabled');
}

// Validate configuration
const requiredFields = [
    'APP_NAME',
    'VERSION',
    'API_BASE',
    'API_PREFIX',
    'AUTH_HEADER'
];

const missingFields = requiredFields.filter(field => !window.APP_CONFIG[field]);
if (missingFields.length > 0) {
    throw new Error(`Missing required configuration fields: ${missingFields.join(', ')}`);
}

// Freeze configuration in production
if (!ENV.isDev) {
    Object.freeze(window.APP_CONFIG);
}

console.log('📋 Сервис-Крым Configuration loaded successfully'); 