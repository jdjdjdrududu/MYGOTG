/**
 * Main Application Module
 * Handles core application functionality, state management, and initialization
 */

class TelegramWebApp {
    constructor() {
        console.log('🚀 Initializing Telegram Web App...');
        
        // App state
        this.state = {
            user: null,
            currentPanel: 'orders',
            isInitialized: false,
            isLoading: true,
            refreshInterval: null,
            performance: {
                startTime: performance.now(),
                initTime: null,
                navigationTimes: []
            }
        };

        // Initialize modules map
        this.modules = new Map();

        // Bind methods
        this.init = this.init.bind(this);
        this.loadUser = this.loadUser.bind(this);
        this.showPanel = this.showPanel.bind(this);
        this.handleError = this.handleError.bind(this);
        this.setupEventListeners = this.setupEventListeners.bind(this);
        this.loadInitialData = this.loadInitialData.bind(this);

        // Initialize Telegram WebApp
        this.initTelegram();
    }

    /**
     * Initialize Telegram WebApp
     */
    initTelegram() {
        try {
            if (window.Telegram?.WebApp) {
                this.tg = window.Telegram.WebApp;
                console.log('✅ Using Telegram WebApp');
            } else if (window.APP_CONFIG?.AUTH_FALLBACK_ENABLED) {
                console.log('⚠️ Using fallback mode for development');
                // Create fallback object for development
                this.tg = {
                    initData: 'fallback_init_data',
                    initDataUnsafe: {
                        user: {
                            id: 1263060321,
                            username: 'Demontaj_Crimea',
                            first_name: 'Оператор',
                            last_name: 'Сервис-Крым'
                        }
                    },
                    ready: () => console.log('✅ Fallback WebApp ready'),
                    expand: () => console.log('✅ Fallback WebApp expanded'),
                    close: () => console.log('✅ Fallback WebApp close called')
                };
            } else {
                throw new Error('Telegram WebApp не доступен и fallback отключен');
            }
            
            // Call Telegram ready
            if (this.tg.ready) {
                this.tg.ready();
            }
            if (this.tg.expand) {
                this.tg.expand();
            }
            
        } catch (error) {
            console.error('❌ Failed to initialize Telegram WebApp:', error);
            throw error;
        }
    }

    /**
     * Initialize the application
     */
    async init() {
        try {
            console.log(`🔄 Initializing app v${window.APP_CONFIG.VERSION}...`);
            
            // Load core modules
            await this.loadCoreModules();
            console.log('✅ Core modules loaded');
            
            // Load user data first
            await this.loadUser();
            console.log('✅ User data loaded');
            
            // Setup UI with user data
            await this.setupUI();
            console.log('✅ UI setup complete');
            
            // Setup event listeners
            this.setupEventListeners();
            console.log('✅ Event listeners setup');
            
            // Load initial data for current panel
            await this.loadInitialData();
            console.log('✅ Initial data loaded');
            
            // Start auto-refresh
            this.startAutoRefresh();
            
            // Mark as initialized
            this.state.isInitialized = true;
            this.state.isLoading = false;
            this.state.performance.initTime = performance.now() - this.state.performance.startTime;
            
            // Hide loading screen and show main content
            this.hideLoadingScreen();
            
            console.log(`✅ Application initialized successfully in ${this.state.performance.initTime.toFixed(2)}ms`);
            
        } catch (error) {
            console.error('❌ Failed to initialize app:', error);
            this.handleError(error);
        }
    }

    /**
     * Load core modules that are always needed
     */
    async loadCoreModules() {
        try {
            console.log('🔄 Loading core modules...');
            
            // Wait a bit for all scripts to load
            await new Promise(resolve => setTimeout(resolve, 200));
            
            // Check that module classes are available
            const moduleClasses = {
                'utils': window.UtilsModule,
                'api': window.APIModule,
                'ui': window.UIModule
            };
            
            // Only add operator module if it's available
            if (window.OperatorPanelModule) {
                moduleClasses['operator'] = window.OperatorPanelModule;
            }
            
            // Verify all critical modules are loaded
            const criticalModules = ['utils', 'api', 'ui'];
            const missingCritical = criticalModules.filter(name => !moduleClasses[name]);
            
            if (missingCritical.length > 0) {
                console.error(`❌ Critical modules missing: ${missingCritical.join(', ')}`);
                throw new Error(`Critical modules not loaded: ${missingCritical.join(', ')}`);
            }
            
            // Initialize modules in sequence
            for (const [name, ModuleClass] of Object.entries(moduleClasses)) {
                if (!ModuleClass) {
                    console.warn(`⚠️ Module class ${name} not found. Skipping.`);
                    continue;
                }
                
                try {
                    const module = new ModuleClass(this);
                    this.modules.set(name, module);
                    
                    // Initialize module if it has an initialize method
                    if (typeof module.initialize === 'function') {
                        await module.initialize();
                    }
                    
                    console.log(`✅ ${name} module initialized`);
                } catch (error) {
                    console.error(`❌ Failed to initialize ${name} module:`, error);
                    // Don't throw error for non-critical modules
                    if (criticalModules.includes(name)) {
                        throw error;
                    }
                    console.warn(`⚠️ ${name} module is not critical, continuing...`);
                }
            }
            
            console.log('🎯 All core modules initialized successfully');
            
        } catch (error) {
            console.error('❌ Failed to initialize core modules:', error);
            throw new Error(`Не удалось загрузить основные модули приложения: ${error.message}`);
        }
    }

    /**
     * Load user data
     */
    async loadUser() {
        try {
            console.log('🔄 Loading user data...');
            
            const apiModule = this.modules.get('api');
            if (!apiModule) {
                throw new Error('API module not available');
            }

            try {
                // Try to load real profile first
                const response = await apiModule.request('user/profile');
                const user = response.data || response;
                this.state.user = user;
                console.log('✅ User profile loaded from API:', user);
            } catch (error) {
                console.warn('⚠️ Failed to load user profile from API, trying fallback...');
                
                // Fallback to test profile if enabled
                if (window.APP_CONFIG.AUTH_FALLBACK_ENABLED) {
                    try {
                        const testResponse = await fetch('/api/test/profile');
                        if (testResponse.ok) {
                            const testUser = await testResponse.json();
                            this.state.user = testUser.data || testUser;
                            console.log('✅ Test user profile loaded:', this.state.user);
                        } else {
                            throw new Error('Test profile also failed');
                        }
                    } catch (fallbackError) {
                        console.warn('⚠️ Test profile also failed, using hardcoded fallback');
                        // Use hardcoded fallback
                        this.state.user = {
                            ID: 1,
                            ChatID: 1263060321,
                            FirstName: 'Тестовый',
                            LastName: 'Пользователь',
                            Username: 'test_user',
                            Role: 'user',
                            Phone: '',
                            IsBlocked: false
                        };
                    }
                } else {
                    throw error;
                }
            }
            
        } catch (error) {
            console.error('❌ Failed to load user data:', error);
            throw new Error('Не удалось загрузить данные пользователя');
        }
    }

    /**
     * Setup UI
     */
    async setupUI() {
        try {
            console.log('🔄 Setting up UI...');
            
            // Use UI module for setup
            const uiModule = this.modules.get('ui');
            if (uiModule && typeof uiModule.setup === 'function') {
                await uiModule.setup();
            } else {
                console.warn('⚠️ UI module not available, using fallback');
                // Fallback setup - just show initial panel, UI module will handle navigation
                await this.showPanel(this.state.currentPanel);
            }
            
        } catch (error) {
            console.error('❌ Failed to setup UI:', error);
            throw error;
        }
    }

    /**
     * Update user info display
     */
    updateUserInfo() {
        const user = this.state.user;
        if (!user) return;

        // Find or create profile info element
        let profileInfo = document.getElementById('profile-info');
        if (!profileInfo) {
            const profilePanel = document.getElementById('profile-panel');
            if (profilePanel) {
                profileInfo = document.createElement('div');
                profileInfo.id = 'profile-info';
                profilePanel.appendChild(profileInfo);
            }
        }

        if (profileInfo) {
            const utilsModule = this.modules.get('utils');
            profileInfo.innerHTML = `
                <div class="profile-card">
                    <div class="profile-header">
                        <div class="avatar">${user.FirstName?.[0] || '?'}</div>
                        <h2>${user.FirstName || ''} ${user.LastName || ''}</h2>
                        <div class="role-badge ${user.Role}">${utilsModule ? utilsModule.getRoleText(user.Role) : user.Role}</div>
                    </div>
                    <div class="profile-info">
                        <div class="info-row">
                            <span class="label">ID:</span>
                            <span class="value">${user.ID || 'N/A'}</span>
                        </div>
                        ${user.Username ? `
                        <div class="info-row">
                            <span class="label">Username:</span>
                            <span class="value">@${user.Username}</span>
                        </div>
                        ` : ''}
                        ${user.Phone ? `
                        <div class="info-row">
                            <span class="label">Телефон:</span>
                            <span class="value">${utilsModule ? utilsModule.formatPhone(user.Phone) : (user.Phone || 'Не указан')}</span>
                        </div>
                        ` : ''}
                    </div>
                </div>
            `;
        }
    }

    /**
     * Setup event listeners
     */
    setupEventListeners() {
        // Navigation clicks
        const navElement = document.getElementById('app-navigation');
        if (navElement) {
            navElement.addEventListener('click', (e) => {
                const navItem = e.target.closest('.nav-item');
                if (navItem) {
                    this.showPanel(navItem.dataset.panel);
                }
            });
        }

        // Window focus - refresh data
        window.addEventListener('focus', () => {
            if (this.state.isInitialized && !this.state.isLoading) {
                this.refreshCurrentPanel();
            }
        });

        // Error handling
        window.addEventListener('error', (e) => {
            console.error('Global error:', e.error);
            this.handleError(e.error, 'Неожиданная ошибка');
        });

        window.addEventListener('unhandledrejection', (e) => {
            console.error('Unhandled promise rejection:', e.reason);
            this.handleError(e.reason, 'Ошибка обещания');
        });
    }

    /**
     * Show a panel - delegate to UI module
     */
    async showPanel(panelId) {
        try {
            const uiModule = this.modules.get('ui');
            if (uiModule && typeof uiModule.showPanel === 'function') {
                await uiModule.showPanel(panelId);
                this.state.currentPanel = panelId;
            } else {
                throw new Error('UI module not available');
            }
        } catch (error) {
            console.error(`❌ Failed to show panel ${panelId}:`, error);
            this.handleError(error, `Не удалось загрузить панель: ${panelId}`);
        }
    }

    /**
     * Update navigation active state
     */
    updateNavigation(panelId) {
        const navItems = document.querySelectorAll('.nav-item');
        navItems.forEach(item => {
            item.classList.toggle('active', item.dataset.panel === panelId);
        });
    }

    /**
     * Load panel data
     */
    async loadPanelData(panelId) {
        try {
            switch (panelId) {
                case 'orders':
                    await this.loadOrders();
                    break;
                case 'clients':
                    await this.loadClients();
                    break;
                case 'profile':
                    this.loadProfile();
                    break;
                default:
                    console.warn(`No data loader for panel: ${panelId}`);
            }
        } catch (error) {
            console.warn(`Failed to load data for panel ${panelId}:`, error);
            // Don't throw - allow panel to be shown even if data fails
        }
    }

    /**
     * Load initial data for current panel
     */
    async loadInitialData() {
        try {
            await this.loadPanelData(this.state.currentPanel);
        } catch (error) {
            console.warn('Failed to load initial data:', error);
        }
    }

    /**
     * Load orders
     */
    async loadOrders() {
        try {
            const operatorModule = this.modules.get('operator');
            if (operatorModule) {
                await operatorModule.loadOrders();
            } else {
                console.warn('Operator module not available');
            }
        } catch (error) {
            console.error('Failed to load orders:', error);
        }
    }

    /**
     * Load clients
     */
    async loadClients() {
        try {
            const apiModule = this.modules.get('api');
            const utilsModule = this.modules.get('utils');
            
            // Используем метод fetchClients вместо прямого запроса
            const response = await apiModule.fetchClients();
            const clients = response.data || [];
            
            this.renderClients(clients, utilsModule);
        } catch (error) {
            console.error('Failed to load clients:', error);
            const panel = document.getElementById('clients-panel');
            if (panel) {
                panel.innerHTML = `<div class="error-state">
                    <i class="fas fa-exclamation-triangle"></i>
                    <h3>Ошибка загрузки клиентов</h3>
                    <p>${error.message || 'Не удалось получить данные с сервера.'}</p>
                </div>`;
            }
        }
    }

    /**
     * Render clients
     */
    renderClients(clients, utilsModule) {
        const panel = document.getElementById('clients-panel');
        if (!panel) return;
        
        if (!clients.length) {
            panel.innerHTML = '<div class="empty-state">Клиентов не найдено</div>';
            return;
        }
        
        panel.innerHTML = `
            <div class="clients-list">
                ${clients.map(client => `
                    <div class="card client-card">
                        <div class="card-header">
                            <div class="card-title">${client.FirstName} ${client.LastName}</div>
                            <div class="status-badge ${client.IsBlocked ? 'blocked' : 'active'}">
                                ${client.IsBlocked ? 'Заблокирован' : 'Активен'}
                            </div>
                        </div>
                        <div class="card-body">
                            <div class="client-info">
                                <div><strong>Телефон:</strong> ${utilsModule ? utilsModule.formatPhone(client.Phone) : client.Phone}</div>
                                <div><strong>Username:</strong> ${client.Username || 'Не указан'}</div>
                                <div><strong>Роль:</strong> ${utilsModule ? utilsModule.getRoleText(client.Role) : client.Role}</div>
                            </div>
                        </div>
                    </div>
                `).join('')}
            </div>
        `;
    }

    /**
     * Load profile
     */
    loadProfile() {
        // Profile info is already updated in updateUserInfo()
        console.log('✅ Profile panel loaded');
    }

    /**
     * Start auto-refresh
     */
    startAutoRefresh() {
        if (this.state.refreshInterval) {
            clearInterval(this.state.refreshInterval);
        }
        
        const interval = window.APP_CONFIG.REFRESH_INTERVAL || 30000;
        this.state.refreshInterval = setInterval(() => {
            if (this.state.isInitialized && !this.state.isLoading) {
                this.refreshCurrentPanel();
            }
        }, interval);
        
        console.log(`✅ Auto-refresh started (${interval}ms)`);
    }

    /**
     * Refresh current panel
     */
    async refreshCurrentPanel() {
        if (this.state.isLoading) return;
        
        try {
            this.state.isLoading = true;
            await this.loadPanelData(this.state.currentPanel);
        } catch (error) {
            console.warn('Failed to refresh panel:', error);
        } finally {
            this.state.isLoading = false;
        }
    }

    /**
     * Hide loading screen
     */
    hideLoadingScreen() {
        const loadingScreen = document.getElementById('loading-screen');
        const mainContent = document.getElementById('main-content');
        
        if (loadingScreen) {
            loadingScreen.classList.add('hidden');
        }
        if (mainContent) {
            mainContent.classList.remove('hidden');
        }
        
        console.log('✅ Loading screen hidden');
    }

    /**
     * Handle error
     */
    handleError(error, context = '') {
        console.error(`❌ ${context}:`, error);
        
        // Show error screen
        const loadingScreen = document.getElementById('loading-screen');
        const errorScreen = document.getElementById('error-screen');
        const errorMessage = document.getElementById('error-message');
        
        if (loadingScreen) loadingScreen.classList.add('hidden');
        if (errorScreen) errorScreen.classList.remove('hidden');
        if (errorMessage) {
            errorMessage.textContent = error.message || 'Неизвестная ошибка';
        }
        
        // Add retry functionality
        const retryButton = document.getElementById('retry-button');
        if (retryButton) {
            retryButton.addEventListener('click', () => {
                window.location.reload();
            });
        }
        
        // Log detailed error in debug mode
        if (window.APP_CONFIG.DEBUG) {
            console.debug('Error details:', {
                context,
                error,
                state: this.state,
                timestamp: new Date().toISOString()
            });
        }
    }
}

// Export the class
window.TelegramWebApp = TelegramWebApp; 