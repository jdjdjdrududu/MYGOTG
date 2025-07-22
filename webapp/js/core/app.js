/**
 * @fileoverview Modern Telegram Web App Core Module
 * @version 2.0
 * @author Service-Crym Development Team
 */

/**
 * Main Application Class
 * Handles initialization, state management, and module loading
 */
class TelegramWebApp {
    constructor() {
        this.version = '2.0';
        this.modules = new Map();
        this.state = {
            user: null,
            currentPanel: null,
            isInitialized: false,
            loadedModules: new Set(),
            performance: {
                startTime: performance.now(),
                initTime: null,
                navigationTimes: []
            }
        };
        
        this.config = window.AppConfig || {
            apiBaseUrl: 'https://xn----ctbinlmxece7i.xn--p1ai',
            enableAnalytics: true,
            enableServiceWorker: true,
            cacheVersion: '2.0',
            enableVirtualization: true,
            enableLazyLoading: true
        };

        // Bind methods
        this.init = this.init.bind(this);
        this.loadModule = this.loadModule.bind(this);
        this.showPanel = this.showPanel.bind(this);
        this.handleError = this.handleError.bind(this);
    }

    /**
     * Initialize the application
     */
    async init() {
        try {
            console.log(`🚀 Initializing Telegram Web App v${this.version}`);
            
            // Setup Telegram WebApp
            this.setupTelegramWebApp();
            
            // Initialize performance monitoring
            this.setupPerformanceMonitoring();
            
            // Setup error handling
            this.setupErrorHandling();
            
            // Setup network monitoring
            this.setupNetworkMonitoring();
            
            // Load core modules
            await this.loadCoreModules();
            
            // Authenticate user
            const user = await this.authenticateUser();
            this.state.user = user;
            
            // Load role-specific modules
            await this.loadRoleSpecificModules(user.Role);
            
            // Setup UI based on role
            await this.setupUIForRole(user.Role);
            
            // Hide loading screen
            this.hideLoadingScreen();
            
            this.state.isInitialized = true;
            this.state.performance.initTime = performance.now() - this.state.performance.startTime;
            
            console.log(`✅ App initialized in ${this.state.performance.initTime.toFixed(2)}ms`);
            
            // Send analytics
            this.sendAnalytics('app_initialized', {
                initTime: this.state.performance.initTime,
                userRole: user.Role
            });
            
        } catch (error) {
            console.error('❌ Failed to initialize app:', error);
            this.handleError(error);
        }
    }

    /**
     * Setup Telegram WebApp integration
     */
    setupTelegramWebApp() {
        // Детальная диагностика состояния Telegram WebApp
        console.log('🔍 Checking Telegram WebApp availability...');
        console.log('window.Telegram:', window.Telegram);
        console.log('window.telegramWebAppLoaded:', window.telegramWebAppLoaded);
        console.log('window.telegramWebAppError:', window.telegramWebAppError);
        
        if (!window.Telegram) {
            throw new Error('Telegram object is not available. App opened outside Telegram context?');
        }
        
        if (!window.Telegram.WebApp) {
            throw new Error('Telegram.WebApp is not available. Script loading failed?');
        }

        const tg = window.Telegram.WebApp;
        this.tg = tg;
        
        // Проверяем базовые методы WebApp
        if (typeof tg.ready !== 'function') {
            throw new Error('Telegram WebApp methods are not available');
        }
        
        try {
            tg.ready();
            tg.expand();
            tg.BackButton.hide();
            
            console.log('✅ Telegram WebApp initialized successfully');
            console.log('User data available:', !!tg.initData);
            console.log('Color scheme:', tg.colorScheme);
            
            // Setup theme
            this.setupTheme();
            
            // Setup main button if needed
            if (tg.MainButton) {
                tg.MainButton.hide();
            }
        } catch (error) {
            console.error('❌ Error during Telegram WebApp setup:', error);
            throw new Error(`Telegram WebApp setup failed: ${error.message}`);
        }
    }

    /**
     * Setup theme based on Telegram colors
     */
    setupTheme() {
        const tg = this.tg;
        const root = document.documentElement;
        
        if (tg.colorScheme === 'dark') {
            root.style.setProperty('--tg-bg-color', tg.backgroundColor || '#1a1a1a');
            root.style.setProperty('--tg-text-color', '#ffffff');
            root.style.setProperty('--tg-secondary-bg-color', '#2a2a2a');
        } else {
            root.style.setProperty('--tg-bg-color', tg.backgroundColor || '#ffffff');
            root.style.setProperty('--tg-text-color', '#000000');
            root.style.setProperty('--tg-secondary-bg-color', '#f7f7f7');
        }
    }

    /**
     * Setup performance monitoring
     */
    setupPerformanceMonitoring() {
        // Monitor Core Web Vitals
        this.observeWebVitals();
        
        // Monitor resource loading
        this.observeResourceLoading();
        
        // Monitor user interactions
        this.observeUserInteractions();
    }

    /**
     * Setup error handling
     */
    setupErrorHandling() {
        window.addEventListener('error', (event) => {
            this.handleError(event.error, 'global_error');
        });

        window.addEventListener('unhandledrejection', (event) => {
            this.handleError(event.reason, 'unhandled_promise');
        });
    }

    /**
     * Setup network monitoring
     */
    setupNetworkMonitoring() {
        // Monitor online/offline status
        window.addEventListener('online', () => {
            this.showToast('success', 'Соединение восстановлено');
            this.modules.get('api')?.retryFailedRequests();
        });

        window.addEventListener('offline', () => {
            this.showToast('warning', 'Отсутствует подключение к интернету');
        });
    }

    /**
     * Load core modules that are always needed
     */
    async loadCoreModules() {
        // Модули уже загружены через HTML, просто инициализируем их
        try {
            if (window.UtilsModule) {
                this.modules.set('utils', new UtilsModule(this));
                console.log('✅ Utils module initialized');
            }
            
            if (window.APIModule) {
                this.modules.set('api', new APIModule(this));
                console.log('✅ API module initialized');
            }
            
            if (window.UIModule) {
                this.modules.set('ui', new UIModule(this));
                console.log('✅ UI module initialized');
            }
            
            if (window.OperatorPanelModule) {
                this.modules.set('operatorPanel', new OperatorPanelModule(this));
                console.log('✅ Operator Panel module initialized');
            }
            
            console.log('🎯 Core modules initialized successfully');
        } catch (error) {
            console.error('❌ Failed to initialize core modules:', error);
            throw error;
        }
    }

    /**
     * Load role-specific modules
     */
    async loadRoleSpecificModules(role) {
        // For now, all roles use the same operator panel module
        // Additional role-specific functionality will be handled within UI module
        console.log(`📦 Loading role-specific modules for: ${role}`);
        
        // All available modules are already loaded via HTML script tags
        // Just verify they're initialized
        if (!this.modules.has('operatorPanel')) {
            throw new Error('Operator Panel module not available');
        }
        
        console.log(`✅ Role-specific modules ready for: ${role}`);
    }

    /**
     * Dynamically load a module
     */
    async loadModule(name, path) {
        if (this.modules.has(name)) {
            return this.modules.get(name);
        }

        try {
            console.log(`📦 Loading module: ${name}`);
            const moduleStart = performance.now();
            
            const module = await import(path);
            const ModuleClass = module.default;
            const instance = new ModuleClass(this);
            
            this.modules.set(name, instance);
            this.state.loadedModules.add(name);
            
            const loadTime = performance.now() - moduleStart;
            console.log(`✅ Module ${name} loaded in ${loadTime.toFixed(2)}ms`);
            
            return instance;
        } catch (error) {
            console.error(`❌ Failed to load module ${name}:`, error);
            throw error;
        }
    }

    /**
     * Authenticate user via Telegram WebApp
     */
    async authenticateUser() {
        const api = this.modules.get('api');
        if (!api) {
            throw new Error('API module not loaded');
        }

        console.log('🔐 Starting user authentication...');
        console.log('Telegram initData available:', !!this.tg.initData);
        console.log('Telegram initData length:', this.tg.initData ? this.tg.initData.length : 0);
        
        if (!this.tg.initData) {
            // Детальная диагностика проблемы с авторизацией
            console.error('❌ No Telegram auth data available');
            console.log('Telegram WebApp object:', this.tg);
            console.log('Available properties:', Object.keys(this.tg));
            
            throw new Error('No Telegram auth data available. App might be opened outside Telegram or auth data is missing.');
        }

        try {
            console.log('📡 Fetching user profile from API...');
            const user = await api.fetchUserProfile();
            console.log('👤 User authenticated successfully:', user.Role);
            console.log('User ID:', user.ID);
            console.log('Username:', user.Username);
            return user;
        } catch (error) {
            console.error('❌ Authentication failed:', error);
            console.log('Error details:', {
                message: error.message,
                status: error.status,
                response: error.response
            });
            throw new Error(`Authentication failed: ${error.message}`);
        }
    }

    /**
     * Setup UI based on user role
     */
    async setupUIForRole(role) {
        const ui = this.modules.get('ui');
        if (!ui) {
            throw new Error('UI module not loaded');
        }

        await ui.setupForRole(role);
        
        // Setup navigation
        this.setupNavigation(role);
        
        // Load initial panel
        await this.loadInitialPanel(role);
    }

    /**
     * Setup navigation based on role
     */
    setupNavigation(role) {
        const navigationMap = {
            'user': ['orders', 'create-order', 'contact'],
            'operator': ['orders', 'clients', 'create-order'],
            'main_operator': ['orders', 'clients', 'staff', 'create-order'],
            'owner': ['orders', 'clients', 'staff', 'analytics', 'financials'],
            'driver': ['orders', 'statistics']
        };

        const navItems = navigationMap[role] || ['orders'];
        this.modules.get('ui')?.setupNavigation(navItems);
    }

    /**
     * Load initial panel based on role
     */
    async loadInitialPanel(role) {
        const panelMap = {
            'user': 'user-panel',
            'operator': 'operator-panel',
            'main_operator': 'operator-panel',
            'owner': 'operator-panel',
            'driver': 'driver-panel'
        };

        const initialPanel = panelMap[role] || 'operator-panel';
        await this.showPanel(initialPanel);
    }

    /**
     * Show a panel with smooth transition
     */
    async showPanel(panelId, direction = 'forward') {
        const startTime = performance.now();
        
        try {
            const ui = this.modules.get('ui');
            if (!ui) {
                throw new Error('UI module not loaded');
            }

            await ui.showPanel(panelId, direction);
            this.state.currentPanel = panelId;
            
            const transitionTime = performance.now() - startTime;
            this.state.performance.navigationTimes.push(transitionTime);
            
            console.log(`🔄 Panel transition to ${panelId} in ${transitionTime.toFixed(2)}ms`);
            
        } catch (error) {
            console.error(`❌ Failed to show panel ${panelId}:`, error);
            this.handleError(error);
        }
    }

    /**
     * Show toast notification
     */
    showToast(type, message, duration = 3000) {
        const ui = this.modules.get('ui');
        if (ui) {
            ui.showToast(type, message, duration);
        }
    }

    /**
     * Handle errors gracefully
     */
    handleError(error, context = 'unknown') {
        console.error('🔥 Application error:', error);
        
        // Show user-friendly error message
        const message = this.getErrorMessage(error);
        this.showToast('error', message);
        
        // Send error analytics
        this.sendAnalytics('error_occurred', {
            error: error.message,
            context,
            stack: error.stack,
            userAgent: navigator.userAgent,
            timestamp: Date.now()
        });
    }

    /**
     * Get user-friendly error message
     */
    getErrorMessage(error) {
        const errorMessages = {
            'NetworkError': 'Проблема с подключением к интернету',
            'AuthenticationError': 'Ошибка аутентификации. Перезапустите приложение',
            'ValidationError': 'Проверьте правильность заполнения формы',
            'ServerError': 'Ошибка сервера. Попробуйте позже'
        };

        return errorMessages[error.name] || 'Произошла ошибка. Попробуйте еще раз';
    }

    /**
     * Hide loading screen with animation
     */
    hideLoadingScreen() {
        const loadingElement = document.getElementById('app-loading');
        const appContainer = document.getElementById('app-container');
        
        if (loadingElement && appContainer) {
            // Fade out loading
            loadingElement.style.opacity = '0';
            loadingElement.style.transform = 'scale(0.8)';
            
            // Fade in app
            appContainer.style.opacity = '1';
            
            setTimeout(() => {
                loadingElement.style.display = 'none';
            }, 300);
        }
    }

    /**
     * Observe Core Web Vitals
     */
    observeWebVitals() {
        // Largest Contentful Paint
        new PerformanceObserver((entryList) => {
            const entries = entryList.getEntries();
            const lastEntry = entries[entries.length - 1];
            console.log('LCP:', lastEntry.startTime);
        }).observe({ entryTypes: ['largest-contentful-paint'] });

        // First Input Delay
        new PerformanceObserver((entryList) => {
            for (const entry of entryList.getEntries()) {
                console.log('FID:', entry.processingStart - entry.startTime);
            }
        }).observe({ entryTypes: ['first-input'] });

        // Cumulative Layout Shift
        let clsValue = 0;
        new PerformanceObserver((entryList) => {
            for (const entry of entryList.getEntries()) {
                if (!entry.hadRecentInput) {
                    clsValue += entry.value;
                }
            }
            console.log('CLS:', clsValue);
        }).observe({ entryTypes: ['layout-shift'] });
    }

    /**
     * Observe resource loading
     */
    observeResourceLoading() {
        new PerformanceObserver((entryList) => {
            for (const entry of entryList.getEntries()) {
                if (entry.name.includes('.js') || entry.name.includes('.css')) {
                    console.log(`Resource loaded: ${entry.name} in ${entry.duration.toFixed(2)}ms`);
                }
            }
        }).observe({ entryTypes: ['resource'] });
    }

    /**
     * Observe user interactions
     */
    observeUserInteractions() {
        ['click', 'touchstart'].forEach(eventType => {
            document.addEventListener(eventType, (event) => {
                const target = event.target.closest('[data-analytics]');
                if (target) {
                    const action = target.dataset.analytics;
                    this.sendAnalytics('user_interaction', {
                        action,
                        element: target.tagName,
                        timestamp: Date.now()
                    });
                }
            }, { passive: true });
        });
    }

    /**
     * Send analytics data
     */
    sendAnalytics(event, data = {}) {
        if (!this.config.enableAnalytics) return;

        const analyticsData = {
            event,
            data,
            user_id: this.state.user?.ID,
            user_role: this.state.user?.Role,
            session_id: this.generateSessionId(),
            timestamp: Date.now(),
            url: window.location.href,
            user_agent: navigator.userAgent
        };

        // Send to analytics service (could be Google Analytics, custom endpoint, etc.)
        console.log('📊 Analytics:', analyticsData);
    }

    /**
     * Generate session ID
     */
    generateSessionId() {
        if (!this.sessionId) {
            this.sessionId = Date.now().toString(36) + Math.random().toString(36).substr(2);
        }
        return this.sessionId;
    }

    /**
     * Get application statistics
     */
    getStatistics() {
        return {
            version: this.version,
            loadedModules: Array.from(this.state.loadedModules),
            initTime: this.state.performance.initTime,
            averageNavigationTime: this.state.performance.navigationTimes.reduce((a, b) => a + b, 0) / this.state.performance.navigationTimes.length || 0,
            totalNavigations: this.state.performance.navigationTimes.length,
            currentPanel: this.state.currentPanel,
            userRole: this.state.user?.Role
        };
    }

    /**
     * Show specific panel
     */
    async showPanel(panelId) {
        console.log(`🎯 Showing panel: ${panelId}`);
        
        try {
            const ui = this.modules.get('ui');
            if (!ui) {
                throw new Error('UI module not available');
            }

            // Get panel content
            const content = await ui.getPanelContent(panelId);
            
            // Show in dynamic content area
            const dynamicContent = document.getElementById('dynamic-content');
            if (dynamicContent) {
                dynamicContent.innerHTML = content;
            }

            // Update current panel state
            this.state.currentPanel = panelId;
            
            // Initialize panel specific functionality
            const operatorPanel = this.modules.get('operatorPanel');
            if (operatorPanel && panelId === 'operator-panel') {
                await operatorPanel.initialize();
            }

            console.log(`✅ Panel ${panelId} loaded successfully`);

        } catch (error) {
            console.error(`❌ Failed to show panel ${panelId}:`, error);
            this.handleError(error, `showPanel:${panelId}`);
        }
    }


}

// Делаем TelegramWebApp доступным глобально
window.TelegramWebApp = TelegramWebApp;

// УБИРАЕМ АВТОМАТИЧЕСКУЮ ИНИЦИАЛИЗАЦИЮ - ОНА БУДЕТ В INDEX.HTML
// Initialize application when DOM is ready
// if (document.readyState === 'loading') {
//     document.addEventListener('DOMContentLoaded', initializeApp);
// } else {
//     initializeApp();
// }

// async function initializeApp() {
//     try {
//         // Create global app instance
//         window.TelegramApp = new TelegramWebApp();
        
//         // Initialize the app
//         await window.TelegramApp.init();
        
//         // Make modules globally accessible for debugging
//         window.AppModules = window.TelegramApp.modules;
        
//     } catch (error) {
//         console.error('💥 Critical error during app initialization:', error);
        
//         // Show critical error screen
//         document.body.innerHTML = `
//             <div style="
//                 position: fixed;
//                 top: 0;
//                 left: 0;
//                 width: 100%;
//                 height: 100%;
//                 background: #fff;
//                 display: flex;
//                 flex-direction: column;
//                 align-items: center;
//                 justify-content: center;
//                 padding: 20px;
//                 text-align: center;
//                 font-family: -apple-system, BlinkMacSystemFont, sans-serif;
//             ">
//                 <h1 style="color: #dc3545; margin-bottom: 20px;">⚠️ Ошибка загрузки</h1>
//                 <p style="color: #666; margin-bottom: 20px;">
//                     Приложение не может быть загружено. Пожалуйста, перезапустите Telegram и попробуйте снова.
//                 </p>
//                 <button onclick="window.location.reload()" style="
//                     background: #007aff;
//                     color: white;
//                     border: none;
//                     padding: 12px 24px;
//                     border-radius: 8px;
//                     font-size: 16px;
//                     cursor: pointer;
//                 ">
//                     Перезагрузить
//                 </button>
//             </div>
//         `;
//     }
// } 