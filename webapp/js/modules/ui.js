/**
 * UI Module
 * Handles all UI-related functionality
 */

class UIModule {
    constructor(app) {
        if (!app) {
            throw new Error('App instance is required for UI Module');
        }

        // Initialize properties
        this.app = app;
        this.currentPanel = null;
        this.panels = new Map();
        this.toastContainer = null;
        this.performanceMode = false;
        this.lazyLoadObserver = null;

        // Create methods before binding
        this.showPanel = async (panelId) => {
            try {
                if (this.currentPanel) {
                    await this.hidePanel(this.currentPanel);
                }
                const panel = await this.getOrCreatePanel(panelId);
                panel.classList.remove('hidden');
                panel.classList.add('active');
                this.updateNavigation(panelId);
                this.currentPanel = panelId;
            } catch (error) {
                console.error('Failed to show panel:', error);
                this.showError('Failed to load panel');
            }
        };

        this.hidePanel = async (panelId) => {
            const panel = document.getElementById(`${panelId}-panel`);
            if (panel) {
                panel.classList.remove('active');
                panel.classList.add('hidden');
            }
        };

        this.showError = (message) => {
            this.showToast('error', message, 5000);
        };

        this.showNotification = (message, type = 'info', duration = 5000) => {
            const notification = document.createElement('div');
            notification.className = `notification ${type}`;
            notification.textContent = message;
            document.body.appendChild(notification);
            setTimeout(() => notification.remove(), duration);
        };

        this.showToast = (type, message, duration = 3000) => {
            const toast = document.createElement('div');
            toast.className = `toast toast-${type}`;
            const icon = this.getToastIcon(type);
            toast.innerHTML = `
                <div class="toast-content">
                    <i class="fas ${icon}"></i>
                    <span>${message}</span>
                </div>
            `;
            this.toastContainer.appendChild(toast);
            setTimeout(() => toast.classList.add('show'), 10);
            setTimeout(() => {
                toast.classList.remove('show');
                setTimeout(() => toast.remove(), 300);
            }, duration);
        };

        this.handleNavigation = (event) => {
            const button = event.target.closest('button');
            if (!button) return;
            const panelId = button.dataset.panel;
            if (panelId) {
                this.showPanel(panelId);
            }
        };

        // Initialize toast container
        this.initToastContainer();

        // Initialize performance optimizations
        this.initPerformanceOptimizations();

        console.log('✅ UI Module initialized');
    }

    /**
     * Setup UI module
     */
    async setup() {
        try {
            console.log('🔄 Setting up UI module...');
            
            // Убедимся что пользователь загружен
            if (!this.app.state.user) {
                console.log('⚠️ User not loaded yet, waiting...');
                await this.app.loadUser();
            }
            
            // Setup navigation
            this.setupNavigation();
            
            // Show initial panel
            await this.showPanel('orders');
            
            console.log('✅ UI setup complete');
            return true;
        } catch (error) {
            console.error('❌ Failed to setup UI:', error);
            throw error;
        }
    }

    /**
     * Setup navigation
     */
    setupNavigation() {
        const nav = document.getElementById('app-navigation');
        if (!nav) {
            console.warn('⚠️ Navigation element not found');
            return;
        }

        const user = this.app?.state?.user;
        
        // Base navigation items that everyone can see
        const navItems = [
            { id: 'orders', icon: 'fas fa-list', label: 'Заказы' },
            { id: 'profile', icon: 'fas fa-user', label: 'Профиль' }
        ];
        
        // Add clients panel only for operators, admins, and owners
        if (user && ['operator', 'admin', 'owner'].includes(user.Role)) {
            navItems.splice(1, 0, { id: 'clients', icon: 'fas fa-users', label: 'Клиенты' });
        }

        // Add statistics panel only for operators, admins, and owners (not regular users)
        if (user && ['operator', 'admin', 'owner', 'main_operator'].includes(user.Role)) {
            navItems.splice(-1, 0, { id: 'stats', icon: 'fas fa-chart-line', label: 'Статистика' });
        }

        // Create BOTTOM navigation
        nav.innerHTML = `
            <div id="bottom-nav" class="bottom-nav">
                ${navItems.map(item => `
                    <button class="nav-item ${item.id === 'orders' ? 'active' : ''}" data-panel="${item.id}">
                        <i class="${item.icon}"></i>
                        <span>${item.label}</span>
                    </button>
                `).join('')}
            </div>
        `;

        // Add click handlers
        const navItemElements = nav.querySelectorAll('.nav-item');
        navItemElements.forEach(item => {
            item.addEventListener('click', () => {
                const panelId = item.dataset.panel;
                if (panelId) {
                    this.showPanel(panelId);
                    navItemElements.forEach(i => i.classList.remove('active'));
                    item.classList.add('active');
                }
            });
        });

        console.log('✅ Navigation setup complete');
    }

    /**
     * Initialize toast container
     */
    initToastContainer() {
        this.toastContainer = document.createElement('div');
        this.toastContainer.className = 'toast-container';
        document.body.appendChild(this.toastContainer);
    }

    /**
     * Initialize performance optimizations
     */
    initPerformanceOptimizations() {
        // Detect low-end devices
        this.detectPerformanceMode();
        
        // Initialize lazy loading
        this.initLazyLoading();
        
        // Setup performance monitoring
        this.setupPerformanceMonitoring();
        
        console.log('✅ Performance optimizations initialized');
    }

    /**
     * Detect if performance mode should be enabled
     */
    detectPerformanceMode() {
        const isLowEnd = () => {
            // Check hardware concurrency (CPU cores)
            if (navigator.hardwareConcurrency && navigator.hardwareConcurrency <= 2) {
                return true;
            }
            
            // Check device memory (if available)
            if (navigator.deviceMemory && navigator.deviceMemory <= 2) {
                return true;
            }
            
            // Check connection speed
            if (navigator.connection) {
                const slowConnections = ['slow-2g', '2g'];
                if (slowConnections.includes(navigator.connection.effectiveType)) {
                    return true;
                }
            }
            
            // Check for old/slow browsers
            const userAgent = navigator.userAgent.toLowerCase();
            if (userAgent.includes('android') && userAgent.includes('chrome/') && 
                parseInt(userAgent.match(/chrome\/(\d+)/)?.[1] || 0) < 80) {
                return true;
            }
            
            return false;
        };

        this.performanceMode = isLowEnd();
        
        if (this.performanceMode) {
            document.body.classList.add('performance-mode');
            console.log('🚀 Performance mode enabled for low-end device');
        }
    }

    /**
     * Initialize lazy loading for images and content
     */
    initLazyLoading() {
        if ('IntersectionObserver' in window) {
            this.lazyLoadObserver = new IntersectionObserver((entries) => {
                entries.forEach(entry => {
                    if (entry.isIntersecting) {
                        const element = entry.target;
                        this.loadLazyContent(element);
                        this.lazyLoadObserver.unobserve(element);
                    }
                });
            }, {
                rootMargin: '50px'
            });
        }
    }

    /**
     * Load lazy content for an element
     */
    loadLazyContent(element) {
        if (element.dataset.lazyType === 'order-card') {
            // Replace placeholder with actual order card
            const orderData = JSON.parse(element.dataset.orderData);
            element.outerHTML = this.renderOrderCard(orderData);
        }
    }

    /**
     * Setup performance monitoring
     */
    setupPerformanceMonitoring() {
        // Monitor frame rate
        let lastTime = performance.now();
        let frameCount = 0;
        let fps = 60;

        const measureFPS = (currentTime) => {
            frameCount++;
            if (currentTime - lastTime >= 1000) {
                fps = Math.round((frameCount * 1000) / (currentTime - lastTime));
                frameCount = 0;
                lastTime = currentTime;
                
                // Enable performance mode if FPS is consistently low
                if (fps < 30 && !this.performanceMode) {
                    this.enablePerformanceMode();
                }
            }
            requestAnimationFrame(measureFPS);
        };

        if (!this.performanceMode) {
            requestAnimationFrame(measureFPS);
        }
    }

    /**
     * Enable performance mode dynamically
     */
    enablePerformanceMode() {
        this.performanceMode = true;
        document.body.classList.add('performance-mode');
        this.showToast('info', 'Включен режим оптимизации производительности', 3000);
        console.log('🚀 Performance mode enabled due to low FPS');
    }

    /**
     * Render orders with lazy loading for better performance
     */
    renderOrdersWithLazyLoading(container, orders) {
        const batchSize = 5;
        const initialOrders = orders.slice(0, batchSize);
        const remainingOrders = orders.slice(batchSize);

        // Render initial batch immediately
        container.innerHTML = initialOrders.map(order => this.renderOrderCard(order)).join('');

        // Create placeholders for remaining orders
        if (remainingOrders.length > 0) {
            const placeholders = remainingOrders.map((order, index) => `
                <div class="lazy-placeholder" 
                     data-lazy-type="order-card" 
                     data-order-data='${JSON.stringify(order).replace(/'/g, "&apos;")}'>
                    <i class="fas fa-spinner fa-spin"></i>
                    Загрузка заказа ${batchSize + index + 1}...
                </div>
            `).join('');

            container.insertAdjacentHTML('beforeend', placeholders);

            // Observe placeholders for lazy loading
            if (this.lazyLoadObserver) {
                const lazyElements = container.querySelectorAll('.lazy-placeholder');
                lazyElements.forEach(element => {
                    this.lazyLoadObserver.observe(element);
                });
            }
        }
    }

    /**
     * Show toast notification
     */
    showToast(type, message, duration = 3000) {
        const toast = document.createElement('div');
        toast.className = `toast toast-${type}`;
        
        const icon = this.getToastIcon(type);
        toast.innerHTML = `
            <div class="toast-content">
                <i class="fas ${icon}"></i>
                <span>${message}</span>
            </div>
        `;
        
        this.toastContainer.appendChild(toast);
        
        // Trigger animation
        setTimeout(() => toast.classList.add('show'), 10);
        
        // Auto remove
        setTimeout(() => {
            toast.classList.remove('show');
            setTimeout(() => toast.remove(), 300);
        }, duration);
    }

    /**
     * Get toast icon based on type
     */
    getToastIcon(type) {
        const icons = {
            'success': 'fa-check-circle',
            'error': 'fa-times-circle',
            'warning': 'fa-exclamation-triangle',
            'info': 'fa-info-circle'
        };
        return icons[type] || icons.info;
    }

    /**
     * Show a panel
     */
    async showPanel(panelId) {
        try {
            // Hide current panel
            if (this.currentPanel) {
                await this.hidePanel(this.currentPanel);
            }
            
            // Get or create panel
            const panel = await this.getOrCreatePanel(panelId);
            
            // Show new panel
            panel.classList.remove('hidden');
            panel.classList.add('active');
            
            // Update navigation
            this.updateNavigation(panelId);
            
            // Update state
            this.currentPanel = panelId;
            
        } catch (error) {
            console.error('Failed to show panel:', error);
            this.showError('Failed to load panel');
        }
    }

    /**
     * Hide a panel
     */
    async hidePanel(panelId) {
        const panel = document.getElementById(`${panelId}-panel`);
        if (panel) {
            panel.classList.remove('active');
            panel.classList.add('hidden');
        }
    }

    /**
     * Get or create a panel
     */
    async getOrCreatePanel(panelId) {
        let panel = document.getElementById(`${panelId}-panel`);
        
        if (!panel) {
            panel = await this.createPanel(panelId);
        }
        
        return panel;
    }

    /**
     * Create a new panel
     */
    async createPanel(panelId) {
        const panel = document.createElement('div');
        panel.id = `${panelId}-panel`;
        panel.className = 'panel hidden';
        
        // Add panel content
        panel.innerHTML = await this.getPanelContent(panelId);
        
        // Add to DOM
        const contentPanels = document.getElementById('content-panels');
        if (contentPanels) {
            contentPanels.appendChild(panel);
        } else {
            document.getElementById('main-content').appendChild(panel);
        }
        
        // Initialize panel components
        await this.initializePanelComponents(panel, panelId);
        
        return panel;
    }

    /**
     * Get panel content
     */
    async getPanelContent(panelId) {
        // Check user permissions before creating panels
        const user = this.app?.state?.user;
        
        switch (panelId) {
            case 'orders':
                return this.getOrdersContent();
            case 'clients':
                // Check if user has permission to view clients
                if (!user || !['operator', 'admin', 'owner'].includes(user.Role)) {
                    return `
                        <div class="error-state">
                            <i class="fas fa-lock"></i>
                            <h3>Доступ запрещен</h3>
                            <p>У вас нет прав для просмотра списка клиентов</p>
                        </div>
                    `;
                }
                return this.getClientsContent();
            case 'stats':
                // Check if user has permission to view statistics
                if (!user || !['operator', 'admin', 'owner', 'main_operator'].includes(user.Role)) {
                    return `
                        <div class="error-state">
                            <i class="fas fa-lock"></i>
                            <h3>Доступ запрещен</h3>
                            <p>У вас нет прав для просмотра статистики</p>
                        </div>
                    `;
                }
                return this.getStatsContent();
            case 'profile':
                return this.getProfileContent();
            default:
                throw new Error(`Unknown panel: ${panelId}`);
        }
    }

    /**
     * Initialize panel components
     */
    async initializePanelComponents(panel, panelId) {
        switch (panelId) {
            case 'orders':
                await this.initializeOrders(panel);
                break;
            case 'clients':
                await this.initializeClients(panel);
                break;
            case 'stats':
                await this.initializeStats(panel);
                break;
            case 'profile':
                await this.initializeProfile(panel);
                break;
        }
    }

    /**
     * Update navigation
     */
    updateNavigation(panelId) {
        // Update bottom navigation
        document.querySelectorAll('#bottom-nav button').forEach(button => {
            button.classList.toggle('active', button.dataset.panel === panelId);
        });
    }

    /**
     * Show error message
     */
    showError(message, duration = 5000) {
        // Check if we're in development mode and API is down
        if (window.APP_CONFIG?.AUTH_FALLBACK_ENABLED && message.includes('Failed to load')) {
            const friendlyMessage = 'Сервер временно недоступен. Показываем тестовые данные.';
            this.showToast('warning', friendlyMessage, duration);
        } else {
            this.showToast('error', message, duration);
        }
    }

    /**
     * Show notification
     */
    showNotification(message, type = 'info', duration = 5000) {
        const notification = document.createElement('div');
        notification.className = `notification ${type}`;
        notification.textContent = message;
        
        document.body.appendChild(notification);
        
        // Remove after duration
        setTimeout(() => {
            notification.remove();
        }, duration);
    }



    /**
     * Get orders content
     */
    getOrdersContent() {
        const user = this.app?.state?.user;
        const isStaff = user && ['operator', 'admin', 'owner'].includes(user.Role);
        
        return `
            <div class="orders">
                <div class="page-header">
                    <h1>${isStaff ? 'Центр управления' : 'Мои заказы'}</h1>
                    <p id="orders-subtitle">${isStaff ? 'Система управления заказами' : 'Ваши заказы и их статус'}</p>
                </div>
                
                <div class="filters">
                    <div class="search-input-wrapper">
                        <i class="fas fa-search"></i>
                        <input type="text" id="orders-search" class="search-input" placeholder="Поиск заказов...">
                    </div>
                    ${isStaff ? `
                    <select id="order-status-filter" class="filter-select">
                        <option value="">Все статусы</option>
                        <option value="new">Новые</option>
                        <option value="in_progress">В работе</option>
                        <option value="completed">Выполненные</option>
                        <option value="canceled">Отменённые</option>
                    </select>
                    ` : ''}
                </div>
                <div id="orders-list" class="orders-list"></div>
                
                <!-- Floating Action Button для создания заказа -->
                <button class="fab" id="create-order-fab" title="Создать заказ">
                    <i class="fas fa-plus"></i>
                </button>
            </div>
        `;
    }

    /**
     * Get clients content
     */
    getClientsContent() {
        return `
            <div class="clients">
                <div class="filters">
                    <div class="search-input-wrapper">
                        <i class="fas fa-search"></i>
                        <input type="text" id="clients-search" class="search-input" placeholder="Поиск клиентов...">
                    </div>
                </div>
                <div id="clients-list" class="clients-list"></div>
            </div>
        `;
    }

    /**
     * Get profile content
     */
    getProfileContent() {
        const user = this.app.state.user;
        if (!user) return '<div class="error-state">Пользователь не найден</div>';
        
        return `
            <div class="profile">
                <div class="profile-header">
                    <div class="avatar">${user.FirstName?.[0] || '?'}</div>
                    <h2>${user.FirstName} ${user.LastName || ''}</h2>
                    <div class="role-badge ${user.Role}">${this.getRoleText(user.Role)}</div>
                </div>
                <div class="profile-info">
                    <div class="info-row">
                        <span class="label">ID:</span>
                        <span class="value">${user.ID}</span>
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
                        <span class="value">${this.formatPhone(user.Phone)}</span>
                    </div>
                    ` : ''}
                </div>
            </div>
        `;
    }

    /**
     * Get statistics content
     */
    getStatsContent() {
        return `
            <div class="stats">
                <div class="page-header">
                    <h1>Аналитика бизнеса</h1>
                    <p>Полная статистика и отчеты</p>
                </div>
                
                <!-- Основные метрики -->
                <div class="stats-overview" id="stats-overview">
                    <div class="stat-card" data-animate>
                        <div class="stat-icon">
                            <i class="fas fa-shopping-cart"></i>
                        </div>
                        <div class="stat-content">
                            <h3>Всего заказов</h3>
                            <div class="stat-value">-</div>
                        </div>
                    </div>
                    <div class="stat-card" data-animate>
                        <div class="stat-icon">
                            <i class="fas fa-ruble-sign"></i>
                        </div>
                        <div class="stat-content">
                            <h3>Общий доход</h3>
                            <div class="stat-value">-</div>
                        </div>
                    </div>
                    <div class="stat-card" data-animate>
                        <div class="stat-icon">
                            <i class="fas fa-users"></i>
                        </div>
                        <div class="stat-content">
                            <h3>Активных клиентов</h3>
                            <div class="stat-value">-</div>
                        </div>
                    </div>
                </div>
                
                <!-- Детальные отчеты -->
                <div class="stats-details" id="stats-details">
                    <div class="loading-state">Загрузка статистики...</div>
                </div>
            </div>
        `;
    }

    /**
     * Initialize orders
     */
    async initializeOrders(panel) {
        try {
            // Setup search
            const searchInput = panel.querySelector('#orders-search');
            if (searchInput) {
                const debounceTime = this.performanceMode ? 500 : 300;
                searchInput.addEventListener('input', this.debounce(() => {
                    this.loadOrders();
                }, debounceTime));
            }
            
            // Setup status filter
            const statusFilter = panel.querySelector('#order-status-filter');
            if (statusFilter) {
                statusFilter.addEventListener('change', () => {
                    this.loadOrders();
                });
            }
            
            // Setup FAB button
            const createOrderFab = panel.querySelector('#create-order-fab');
            if (createOrderFab) {
                createOrderFab.addEventListener('click', () => {
                    this.openCreateOrderModal();
                });
            }

            // Load initial orders
            await this.loadOrders();
            
        } catch (error) {
            console.error('Failed to initialize orders:', error);
            this.showError('Failed to load orders');
        }
    }

    /**
     * Initialize clients
     */
    async initializeClients(panel) {
        try {
            // Setup search
            const searchInput = panel.querySelector('#clients-search');
            if (searchInput) {
                const debounceTime = this.performanceMode ? 500 : 300;
                searchInput.addEventListener('input', this.debounce(() => {
                    this.loadClients();
                }, debounceTime));
            }
            
            // Load initial clients
            await this.loadClients();
            
        } catch (error) {
            console.error('Failed to initialize clients:', error);
            this.showError('Failed to load clients');
        }
    }

    /**
     * Initialize profile
     */
    async initializeProfile(panel) {
        // Profile is static, no initialization needed
    }

    /**
     * Initialize statistics
     */
    async initializeStats(panel) {
        try {
            // Load initial statistics
            await this.loadStats();
        } catch (error) {
            console.error('Failed to initialize stats:', error);
            this.showError('Failed to load statistics');
        }
    }

    /**
     * Load orders
     */
    async loadOrders() {
        try {
            const searchQuery = document.querySelector('#orders-search')?.value || '';
            const statusFilter = document.querySelector('#order-status-filter')?.value || '';
            
            const response = await this.app.modules.get('api').fetchOrders({
                search: searchQuery,
                status: statusFilter
            });
            
            const ordersList = document.querySelector('#orders-list');
            if (ordersList && response.data) {
                if (response.data.length) {
                    if (this.performanceMode && response.data.length > 10) {
                        // Use lazy loading for large lists in performance mode
                        this.renderOrdersWithLazyLoading(ordersList, response.data);
                    } else {
                        // Normal rendering
                        ordersList.innerHTML = response.data.map(order => this.renderOrderCard(order)).join('');
                    }
                } else {
                    // Show different empty states based on user role
                    const user = this.app?.state?.user;
                    const isStaff = user && ['operator', 'admin', 'owner'].includes(user.Role);
                    
                    if (isStaff) {
                        ordersList.innerHTML = '<div class="empty-state">Заказов не найдено</div>';
                    } else {
                        // For regular users, show "create first order" encouragement
                        ordersList.innerHTML = `
                            <div class="empty-state-user">
                                <div class="empty-icon">
                                    <i class="fas fa-plus-circle"></i>
                                </div>
                                <h3>Создайте первый заказ</h3>
                                <p>У вас пока нет заказов. Создайте свой первый заказ, чтобы начать пользоваться сервисом!</p>
                                <button class="btn btn-primary" id="create-first-order-btn">
                                    <i class="fas fa-plus"></i>
                                    Создать первый заказ
                                </button>
                            </div>
                        `;
                        
                        // Add event listener to the create first order button
                        setTimeout(() => {
                            const createFirstOrderBtn = document.querySelector('#create-first-order-btn');
                            if (createFirstOrderBtn) {
                                createFirstOrderBtn.addEventListener('click', () => {
                                    this.openCreateOrderModal();
                                });
                            }
                        }, 100);
                    }
                }
            }
            
        } catch (error) {
            console.error('Failed to load orders:', error);
            this.showError('Failed to load orders');
        }
    }

    /**
     * Load clients
     */
    async loadClients() {
        try {
            const searchQuery = document.querySelector('#clients-search')?.value || '';
            
            const response = await this.app.modules.get('api').fetchClients({
                search: searchQuery
            });
            
            const clientsList = document.querySelector('#clients-list');
            if (clientsList && response.data) {
                clientsList.innerHTML = response.data.length ?
                    response.data.map(client => this.renderClientCard(client)).join('') :
                    '<div class="empty-state">Клиентов не найдено</div>';
            }
            
        } catch (error) {
            console.error('Failed to load clients:', error);
            this.showError('Failed to load clients');
        }
    }

    /**
     * Load statistics
     */
    async loadStats() {
        try {
            const response = await this.app.modules.get('api').fetchStats();
            
            // Update stat cards with click handlers
            const statCards = document.querySelectorAll('#stats-overview .stat-card');
            if (statCards && response.data) {
                const stats = response.data;
                
                if (statCards[0]) {
                    const totalOrdersValue = statCards[0].querySelector('.stat-value');
                    if (totalOrdersValue) totalOrdersValue.textContent = stats.totalOrders || '0';
                    // Add click handler for orders details
                    statCards[0].addEventListener('click', () => this.showOrdersBreakdown(stats));
                    statCards[0].style.cursor = 'pointer';
                    statCards[0].title = 'Нажмите для просмотра детальной статистики заказов';
                }
                
                if (statCards[1]) {
                    const totalRevenueValue = statCards[1].querySelector('.stat-value');
                    if (totalRevenueValue) totalRevenueValue.textContent = this.formatCurrency(stats.totalRevenue || 0);
                    // Add click handler for revenue details
                    statCards[1].addEventListener('click', () => this.showRevenueBreakdown(stats));
                    statCards[1].style.cursor = 'pointer';
                    statCards[1].title = 'Нажмите для просмотра детальной статистики доходов';
                }
                
                if (statCards[2]) {
                    const activeClientsValue = statCards[2].querySelector('.stat-value');
                    if (activeClientsValue) activeClientsValue.textContent = stats.activeClients || '0';
                    // Add click handler for clients details
                    statCards[2].addEventListener('click', () => this.showClientsBreakdown(stats));
                    statCards[2].style.cursor = 'pointer';
                    statCards[2].title = 'Нажмите для просмотра детальной статистики клиентов';
                }
            }
            
            // Update details section
            const detailsSection = document.querySelector('#stats-details');
            if (detailsSection && response.data) {
                detailsSection.innerHTML = `
                    <div class="stats-charts">
                        <div class="chart-card">
                            <h3>Динамика заказов</h3>
                            <div class="chart-placeholder">График будет добавлен позже</div>
                        </div>
                        <div class="chart-card">
                            <h3>Доходы по месяцам</h3>
                            <div class="chart-placeholder">График будет добавлен позже</div>
                        </div>
                    </div>
                `;
            }
            
        } catch (error) {
            console.error('Failed to load stats:', error);
            const detailsSection = document.querySelector('#stats-details');
            if (detailsSection) {
                detailsSection.innerHTML = '<div class="error-state">Не удалось загрузить статистику</div>';
            }
        }
    }

    /**
     * Render order card
     */
    renderOrderCard(order) {
        const user = this.app?.state?.user;
        const isUserOrder = user && order.UserChatID === user.ChatID;
        const needsConfirmation = order.Status === 'awaiting_confirmation' && isUserOrder;
        
        return `
            <div class="card order-card ${needsConfirmation ? 'needs-confirmation' : ''}" data-order-id="${order.ID}">
                <div class="card-header">
                    <div class="card-title">${order.Name || `Заказ #${order.ID}`}</div>
                    <div class="status-badge ${order.Status}">${this.getOrderStatusText(order.Status)}</div>
                </div>
                <div class="card-body">
                    <div class="order-info">
                        <div><strong>Категория:</strong> ${order.Category || 'Не указана'}</div>
                        <div><strong>Телефон:</strong> ${this.formatPhone(order.Phone)}</div>
                        <div><strong>Адрес:</strong> ${order.Address || 'Не указан'}</div>
                        <div><strong>Дата:</strong> ${order.Date || 'Не указана'}</div>
                        ${order.Cost ? `<div><strong>Стоимость:</strong> ${this.formatCurrency(order.Cost)}</div>` : ''}
                        <div><strong>Создан:</strong> ${this.formatDate(order.CreatedAt)}</div>
                    </div>
                    
                    ${needsConfirmation ? `
                    <div class="order-actions">
                        <div class="confirmation-notice">
                            <i class="fas fa-clock"></i>
                            <span>Оператор оценил ваш заказ. Подтвердите стоимость для продолжения работы.</span>
                        </div>
                        <div class="action-buttons">
                            <button class="btn btn-success btn-sm" onclick="handleOrderAction(${order.ID}, 'accept_cost')">
                                <i class="fas fa-check"></i>
                                Согласиться (${this.formatCurrency(order.Cost)})
                            </button>
                            <button class="btn btn-danger btn-sm" onclick="handleOrderAction(${order.ID}, 'reject_cost')">
                                <i class="fas fa-times"></i>
                                Отклонить
                            </button>
                        </div>
                    </div>
                    ` : ''}
                </div>
            </div>
        `;
    }

    /**
     * Render client card
     */
    renderClientCard(client) {
        return `
            <div class="card client-card" data-client-id="${client.ID}">
                <div class="card-header">
                    <div class="card-title">${client.FirstName} ${client.LastName || ''}</div>
                    <div class="status-badge ${client.IsBlocked ? 'blocked' : 'active'}">
                        ${client.IsBlocked ? 'Заблокирован' : 'Активен'}
                    </div>
                </div>
                <div class="card-body">
                    <div class="client-info">
                        <div><strong>Телефон:</strong> ${this.formatPhone(client.Phone)}</div>
                        <div><strong>Username:</strong> ${client.Username ? '@' + client.Username : 'Не указан'}</div>
                        <div><strong>Роль:</strong> ${this.getRoleText(client.Role)}</div>
                    </div>
                </div>
            </div>
        `;
    }



    /**
     * Format currency
     */
    formatCurrency(amount) {
        if (!amount) return 'Не указана';
        return new Intl.NumberFormat('ru-RU', {
            style: 'currency',
            currency: 'RUB'
        }).format(amount);
    }

    /**
     * Format date
     */
    formatDate(date) {
        if (!date) return 'Не указана';
        return new Date(date).toLocaleString('ru-RU', {
            day: '2-digit',
            month: '2-digit',
            year: 'numeric',
            hour: '2-digit',
            minute: '2-digit'
        });
    }

    /**
     * Get order status text
     */
    getOrderStatusText(status) {
        const statuses = {
            'new': 'Новый',
            'awaiting_confirmation': 'Ожидает подтверждения',
            'in_progress': 'В работе',
            'completed': 'Выполнен',
            'canceled': 'Отменён',
            'awaiting_payment': 'Ожидает оплаты'
        };
        return statuses[status] || status;
    }



    /**
     * Debounce utility
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
    }

    /**
     * Get role text
     */
    getRoleText(role) {
        const roles = {
            'owner': 'Владелец',
            'admin': 'Администратор',
            'operator': 'Оператор',
            'user': 'Пользователь'
        };
        return roles[role] || role;
    }

    /**
     * Format phone number
     */
    formatPhone(phone) {
        if (!phone || phone === null || phone === undefined) return 'Не указан';
        
        // Добавляем обработку объектов
        if (typeof phone === 'object' && phone !== null) {
            console.warn('Phone is an object:', phone);
            // Пытаемся извлечь значение из объекта
            if (phone.String) {
                phone = phone.String;
            } else if (phone.value) {
                phone = phone.value;
            } else if (phone.Valid !== undefined && !phone.Valid) {
                return 'Не указан';
            } else {
                // Если это объект, но мы не знаем структуру
                console.error('Unknown phone object structure:', phone);
                return 'Не указан';
            }
        }
        
        // Ensure phone is a string before using string methods
        const phoneStr = typeof phone === 'string' ? phone : String(phone);
        // Simple phone formatting
        const cleaned = phoneStr.replace(/\D/g, '');
        if (cleaned.length === 11 && cleaned.startsWith('7')) {
            return `+7 (${cleaned.slice(1, 4)}) ${cleaned.slice(4, 7)}-${cleaned.slice(7, 9)}-${cleaned.slice(9)}`;
        }
        return phoneStr;
    }

    /**
     * Open create order modal
     */
    openCreateOrderModal() {
        const user = this.app?.state?.user;
        const isOwner = user && user.Role === 'owner';
        
        // Create modal HTML
        const modalHtml = `
            <div class="modal-overlay" id="create-order-modal">
                <div class="modal-container">
                    <div class="modal-header">
                        <h2>Создать заказ</h2>
                        <button class="modal-close" id="close-create-order-modal">
                            <i class="fas fa-times"></i>
                        </button>
                    </div>
                    <form class="modal-body" id="create-order-form">
                        <div class="form-group">
                            <label for="order-category">Категория*</label>
                            <select id="order-category" name="category" required>
                                <option value="">Выберите категорию</option>
                                <option value="repair">Ремонт</option>
                                <option value="construction">Строительство</option>
                                <option value="cleaning">Уборка</option>
                                <option value="delivery">Доставка</option>
                                <option value="other">Другое</option>
                            </select>
                        </div>
                        
                        <div class="form-group">
                            <label for="order-subcategory">Подкатегория</label>
                            <input type="text" id="order-subcategory" name="subcategory" placeholder="Например: сантехника, электрика">
                        </div>
                        
                        <div class="form-group">
                            <label for="order-name">Название заказа*</label>
                            <input type="text" id="order-name" name="name" required placeholder="Краткое описание работы">
                        </div>
                        
                        <div class="form-group">
                            <label for="order-description">Описание</label>
                            <textarea id="order-description" name="description" rows="4" placeholder="Подробное описание работы"></textarea>
                        </div>
                        
                        <div class="form-row">
                            <div class="form-group">
                                <label for="order-date">Дата*</label>
                                <input type="date" id="order-date" name="date" required>
                            </div>
                            <div class="form-group">
                                <label for="order-time">Время</label>
                                <input type="time" id="order-time" name="time">
                            </div>
                        </div>
                        
                        <div class="form-group">
                            <label for="order-phone">Телефон*</label>
                            <input type="tel" id="order-phone" name="phone" required placeholder="+7 (999) 123-45-67">
                        </div>
                        
                        <div class="form-group">
                            <label for="order-address">Адрес*</label>
                            <input type="text" id="order-address" name="address" required placeholder="Адрес выполнения работ">
                        </div>
                        
                        ${isOwner ? `
                        <div class="form-row owner-only">
                            <div class="form-group">
                                <label for="order-status">Статус</label>
                                <select id="order-status" name="status">
                                    <option value="new">Новый</option>
                                    <option value="in_progress">В работе</option>
                                    <option value="completed">Завершенный</option>
                                    <option value="canceled">Отмененный</option>
                                </select>
                            </div>
                            <div class="form-group">
                                <label for="order-cost">Стоимость (₽)</label>
                                <input type="number" id="order-cost" name="cost" min="0" step="0.01" placeholder="0.00">
                            </div>
                        </div>
                        ` : ''}
                        
                        <div class="form-group">
                            <label for="order-payment">Способ оплаты</label>
                            <select id="order-payment" name="payment">
                                <option value="cash">Наличные</option>
                                <option value="card">Банковская карта</option>
                                <option value="transfer">Перевод</option>
                                <option value="not_specified">Не указано</option>
                            </select>
                        </div>
                    </form>
                    <div class="modal-footer">
                        <button type="button" class="btn btn-secondary" id="cancel-create-order">Отмена</button>
                        <button type="submit" class="btn btn-primary" id="submit-create-order">Создать заказ</button>
                    </div>
                </div>
            </div>
        `;
        
        // Add modal to DOM
        document.body.insertAdjacentHTML('beforeend', modalHtml);
        
        // Set default date to today
        const dateInput = document.getElementById('order-date');
        if (dateInput) {
            const today = new Date().toISOString().split('T')[0];
            dateInput.value = today;
        }
        
        // Set default phone if user has one
        const phoneInput = document.getElementById('order-phone');
        if (phoneInput && user?.Phone) {
            phoneInput.value = user.Phone;
        }
        
        // Add event listeners
        this.setupCreateOrderModalListeners();
    }

    /**
     * Setup event listeners for create order modal
     */
    setupCreateOrderModalListeners() {
        const modal = document.getElementById('create-order-modal');
        const closeBtn = document.getElementById('close-create-order-modal');
        const cancelBtn = document.getElementById('cancel-create-order');
        const submitBtn = document.getElementById('submit-create-order');
        const form = document.getElementById('create-order-form');

        // Close modal handlers
        const closeModal = () => {
            if (modal) {
                modal.remove();
            }
        };

        if (closeBtn) closeBtn.addEventListener('click', closeModal);
        if (cancelBtn) cancelBtn.addEventListener('click', closeModal);
        
        // Close on overlay click
        if (modal) {
            modal.addEventListener('click', (e) => {
                if (e.target === modal) {
                    closeModal();
                }
            });
        }

        // Form submission
        if (submitBtn && form) {
            submitBtn.addEventListener('click', (e) => {
                e.preventDefault();
                this.handleCreateOrderSubmit(form);
            });
        }

        // Escape key to close
        const handleEscape = (e) => {
            if (e.key === 'Escape') {
                closeModal();
                document.removeEventListener('keydown', handleEscape);
            }
        };
        document.addEventListener('keydown', handleEscape);
    }

    /**
     * Handle create order form submission
     */
    async handleCreateOrderSubmit(form) {
        try {
            const formData = new FormData(form);
            const user = this.app?.state?.user;
            
            // Validate required fields
            const requiredFields = ['category', 'name', 'date', 'phone', 'address'];
            for (const field of requiredFields) {
                if (!formData.get(field)) {
                    this.showError(`Поле "${this.getFieldLabel(field)}" обязательно для заполнения`);
                    return;
                }
            }

            // Build order object
            const orderData = {
                Category: formData.get('category'),
                Subcategory: formData.get('subcategory') || '',
                Name: formData.get('name'),
                Description: formData.get('description') || '',
                Date: formData.get('date'),
                Time: formData.get('time') || '',
                Phone: formData.get('phone'),
                Address: formData.get('address'),
                Payment: formData.get('payment') || 'not_specified',
                Photos: [],
                Videos: []
            };

            // For owners, include status and cost
            if (user && user.Role === 'owner') {
                orderData.Status = formData.get('status') || 'new';
                const cost = formData.get('cost');
                if (cost && !isNaN(cost)) {
                    orderData.Cost = parseFloat(cost);
                }
            }

            // Show loading state
            const submitBtn = document.getElementById('submit-create-order');
            if (submitBtn) {
                submitBtn.disabled = true;
                submitBtn.innerHTML = '<i class="fas fa-spinner fa-spin"></i> Создание...';
            }

            // Determine endpoint based on user role
            const isStaff = user && ['operator', 'admin', 'owner'].includes(user.Role);
            const endpoint = isStaff ? 'admin/create-order' : 'user/create-order';

            // Submit order
            const response = await this.app.modules.get('api').request(endpoint, {
                method: 'POST',
                body: orderData
            });

            // Close modal and show success
            const modal = document.getElementById('create-order-modal');
            if (modal) modal.remove();

            this.showToast('success', 'Заказ успешно создан!', 3000);
            
            // Refresh orders list
            await this.loadOrders();

        } catch (error) {
            console.error('Failed to create order:', error);
            this.showError('Не удалось создать заказ: ' + (error.message || 'Неизвестная ошибка'));
            
            // Reset submit button
            const submitBtn = document.getElementById('submit-create-order');
            if (submitBtn) {
                submitBtn.disabled = false;
                submitBtn.innerHTML = 'Создать заказ';
            }
        }
    }

    /**
     * Get field label for validation messages
     */
    getFieldLabel(field) {
        const labels = {
            'category': 'Категория',
            'name': 'Название заказа',
            'date': 'Дата',
            'phone': 'Телефон',
            'address': 'Адрес'
        };
        return labels[field] || field;
    }

    /**
     * Handle order action (accept/reject cost)
     */
    async handleOrderAction(orderID, action) {
        try {
            let requestBody = { action };
            
            // For reject_cost, we need to ask for a reason
            if (action === 'reject_cost') {
                const reason = prompt('Укажите причину отклонения стоимости:');
                if (!reason) {
                    return; // User canceled
                }
                requestBody.reason = reason;
            }

            // Show loading state
            const card = document.querySelector(`[data-order-id="${orderID}"]`);
            if (card) {
                card.style.opacity = '0.6';
                card.style.pointerEvents = 'none';
            }

            // Make API request
            const response = await this.app.modules.get('api').request(`user/order/${orderID}/action`, {
                method: 'POST',
                body: requestBody
            });

            // Show success message
            const message = action === 'accept_cost' ? 'Стоимость принята!' : 'Стоимость отклонена, заказ отменен';
            this.showToast('success', message, 3000);

            // Refresh orders list
            await this.loadOrders();

        } catch (error) {
            console.error('Failed to handle order action:', error);
            this.showError('Не удалось выполнить действие: ' + (error.message || 'Неизвестная ошибка'));
            
            // Reset card state
            const card = document.querySelector(`[data-order-id="${orderID}"]`);
            if (card) {
                card.style.opacity = '1';
                card.style.pointerEvents = 'auto';
            }
        }
    }

    /**
     * Show detailed orders breakdown
     */
    showOrdersBreakdown(stats) {
        const modalHtml = `
            <div class="modal-overlay" id="orders-breakdown-modal">
                <div class="modal-container">
                    <div class="modal-header">
                        <h2>Детальная статистика заказов</h2>
                        <button class="modal-close" onclick="document.getElementById('orders-breakdown-modal').remove()">
                            <i class="fas fa-times"></i>
                        </button>
                    </div>
                    <div class="modal-body">
                        <div class="stats-breakdown">
                            <div class="breakdown-section">
                                <h3>По статусам</h3>
                                <div class="breakdown-grid">
                                    <div class="breakdown-item">
                                        <span class="label">Новые заказы:</span>
                                        <span class="value">${stats.ordersByStatus?.new || 0}</span>
                                    </div>
                                    <div class="breakdown-item">
                                        <span class="label">В работе:</span>
                                        <span class="value">${stats.ordersByStatus?.in_progress || 0}</span>
                                    </div>
                                    <div class="breakdown-item">
                                        <span class="label">Завершенные:</span>
                                        <span class="value">${stats.ordersByStatus?.completed || 0}</span>
                                    </div>
                                    <div class="breakdown-item">
                                        <span class="label">Отмененные:</span>
                                        <span class="value">${stats.ordersByStatus?.canceled || 0}</span>
                                    </div>
                                </div>
                            </div>
                            
                            <div class="breakdown-section">
                                <h3>По категориям</h3>
                                <div class="breakdown-grid">
                                    ${stats.ordersByCategory ? Object.entries(stats.ordersByCategory).map(([category, count]) => `
                                        <div class="breakdown-item">
                                            <span class="label">${this.getCategoryName(category)}:</span>
                                            <span class="value">${count}</span>
                                        </div>
                                    `).join('') : '<div class="no-data">Нет данных</div>'}
                                </div>
                            </div>
                            
                            <div class="breakdown-section">
                                <h3>Средние показатели</h3>
                                <div class="breakdown-grid">
                                    <div class="breakdown-item">
                                        <span class="label">Средняя стоимость:</span>
                                        <span class="value">${this.formatCurrency(stats.averageOrderValue || 0)}</span>
                                    </div>
                                    <div class="breakdown-item">
                                        <span class="label">Заказов в день:</span>
                                        <span class="value">${Math.round((stats.totalOrders || 0) / 30)}</span>
                                    </div>
                                </div>
                            </div>
                        </div>
                    </div>
                </div>
            </div>
        `;
        
        document.body.insertAdjacentHTML('beforeend', modalHtml);
    }

    /**
     * Show detailed revenue breakdown
     */
    showRevenueBreakdown(stats) {
        const modalHtml = `
            <div class="modal-overlay" id="revenue-breakdown-modal">
                <div class="modal-container">
                    <div class="modal-header">
                        <h2>Детальная статистика доходов</h2>
                        <button class="modal-close" onclick="document.getElementById('revenue-breakdown-modal').remove()">
                            <i class="fas fa-times"></i>
                        </button>
                    </div>
                    <div class="modal-body">
                        <div class="stats-breakdown">
                            <div class="breakdown-section">
                                <h3>По периодам</h3>
                                <div class="breakdown-grid">
                                    <div class="breakdown-item">
                                        <span class="label">Сегодня:</span>
                                        <span class="value">${this.formatCurrency(stats.revenueToday || 0)}</span>
                                    </div>
                                    <div class="breakdown-item">
                                        <span class="label">Эта неделя:</span>
                                        <span class="value">${this.formatCurrency(stats.revenueThisWeek || 0)}</span>
                                    </div>
                                    <div class="breakdown-item">
                                        <span class="label">Этот месяц:</span>
                                        <span class="value">${this.formatCurrency(stats.revenueThisMonth || 0)}</span>
                                    </div>
                                    <div class="breakdown-item">
                                        <span class="label">Весь период:</span>
                                        <span class="value">${this.formatCurrency(stats.totalRevenue || 0)}</span>
                                    </div>
                                </div>
                            </div>
                            
                            <div class="breakdown-section">
                                <h3>По категориям услуг</h3>
                                <div class="breakdown-grid">
                                    ${stats.revenueByCategory ? Object.entries(stats.revenueByCategory).map(([category, revenue]) => `
                                        <div class="breakdown-item">
                                            <span class="label">${this.getCategoryName(category)}:</span>
                                            <span class="value">${this.formatCurrency(revenue)}</span>
                                        </div>
                                    `).join('') : '<div class="no-data">Нет данных</div>'}
                                </div>
                            </div>
                            
                            <div class="breakdown-section">
                                <h3>Прогнозы</h3>
                                <div class="breakdown-grid">
                                    <div class="breakdown-item">
                                        <span class="label">Прогноз на месяц:</span>
                                        <span class="value">${this.formatCurrency((stats.totalRevenue || 0) * 1.1)}</span>
                                    </div>
                                    <div class="breakdown-item">
                                        <span class="label">Средний чек:</span>
                                        <span class="value">${this.formatCurrency(stats.averageOrderValue || 0)}</span>
                                    </div>
                                </div>
                            </div>
                        </div>
                    </div>
                </div>
            </div>
        `;
        
        document.body.insertAdjacentHTML('beforeend', modalHtml);
    }

    /**
     * Show detailed clients breakdown
     */
    showClientsBreakdown(stats) {
        const modalHtml = `
            <div class="modal-overlay" id="clients-breakdown-modal">
                <div class="modal-container">
                    <div class="modal-header">
                        <h2>Детальная статистика клиентов</h2>
                        <button class="modal-close" onclick="document.getElementById('clients-breakdown-modal').remove()">
                            <i class="fas fa-times"></i>
                        </button>
                    </div>
                    <div class="modal-body">
                        <div class="stats-breakdown">
                            <div class="breakdown-section">
                                <h3>Активность клиентов</h3>
                                <div class="breakdown-grid">
                                    <div class="breakdown-item">
                                        <span class="label">Всего клиентов:</span>
                                        <span class="value">${stats.totalClients || 0}</span>
                                    </div>
                                    <div class="breakdown-item">
                                        <span class="label">Активных:</span>
                                        <span class="value">${stats.activeClients || 0}</span>
                                    </div>
                                    <div class="breakdown-item">
                                        <span class="label">Новых за месяц:</span>
                                        <span class="value">${stats.newClientsThisMonth || 0}</span>
                                    </div>
                                    <div class="breakdown-item">
                                        <span class="label">Повторных заказов:</span>
                                        <span class="value">${stats.repeatClients || 0}</span>
                                    </div>
                                </div>
                            </div>
                            
                            <div class="breakdown-section">
                                <h3>ТОП клиенты</h3>
                                <div class="top-clients">
                                    ${stats.topClients ? stats.topClients.map((client, index) => `
                                        <div class="top-client-item">
                                            <span class="rank">#${index + 1}</span>
                                            <span class="name">${client.name}</span>
                                            <span class="orders">${client.orders} заказов</span>
                                            <span class="spent">${this.formatCurrency(client.totalSpent)}</span>
                                        </div>
                                    `).join('') : '<div class="no-data">Нет данных</div>'}
                                </div>
                            </div>
                            
                            <div class="breakdown-section">
                                <h3>Лояльность</h3>
                                <div class="breakdown-grid">
                                    <div class="breakdown-item">
                                        <span class="label">Средний LTV:</span>
                                        <span class="value">${this.formatCurrency(stats.averageClientLTV || 0)}</span>
                                    </div>
                                    <div class="breakdown-item">
                                        <span class="label">Retention rate:</span>
                                        <span class="value">${Math.round((stats.retentionRate || 0) * 100)}%</span>
                                    </div>
                                </div>
                            </div>
                        </div>
                    </div>
                </div>
            </div>
        `;
        
        document.body.insertAdjacentHTML('beforeend', modalHtml);
    }

    /**
     * Get category display name
     */
    getCategoryName(category) {
        const categories = {
            'repair': 'Ремонт',
            'construction': 'Строительство',
            'cleaning': 'Уборка',
            'delivery': 'Доставка',
            'other': 'Другое'
        };
        return categories[category] || category;
    }
}

// Global function for order actions (used in onclick)
window.handleOrderAction = function(orderID, action) {
    // Try to find the app instance first (new architecture)
    const app = window.telegramApp || window.app;
    if (app && app.modules && app.modules.get('ui')) {
        app.modules.get('ui').handleOrderAction(orderID, action);
    } else {
        console.error('Could not find order action handler');
    }
};

// Export module
window.UIModule = UIModule; 