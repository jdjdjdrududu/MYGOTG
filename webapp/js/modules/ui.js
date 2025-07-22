/**
 * @fileoverview Modern UI Module with virtualization and performance optimizations
 * @version 2.0
 */

class UIModule {
    constructor(app) {
        this.app = app;
        this.panels = new Map();
        this.components = new Map();
        this.virtualizedLists = new Map();
        this.animationQueue = [];
        this.intersectionObserver = null;
        this.resizeObserver = null;
        
        // Performance settings
        this.settings = {
            enableVirtualization: true,
            virtualItemHeight: 80,
            virtualBuffer: 5,
            debounceDelay: 300,
            animationDuration: 300
        };

        this.setupObservers();
        this.setupGlobalEventListeners();
    }

    /**
     * Setup performance observers
     */
    setupObservers() {
        // Intersection Observer for lazy loading
        this.intersectionObserver = new IntersectionObserver(
            (entries) => this.handleIntersection(entries),
            {
                rootMargin: '50px',
                threshold: 0.1
            }
        );

        // Resize Observer for responsive updates
        this.resizeObserver = new ResizeObserver(
            (entries) => this.handleResize(entries)
        );
    }

    /**
     * Setup global event listeners
     */
    setupGlobalEventListeners() {
        // Debounced scroll handler
        let scrollTimer;
        document.addEventListener('scroll', () => {
            clearTimeout(scrollTimer);
            scrollTimer = setTimeout(() => {
                this.handleScroll();
            }, this.settings.debounceDelay);
        }, { passive: true });

        // Touch gesture support
        this.setupTouchGestures();
        
        // Keyboard navigation
        this.setupKeyboardNavigation();
    }

    /**
     * Setup for specific user role
     */
    async setupForRole(role) {
        console.log(`🎨 Setting up UI for role: ${role}`);
        
        // Clear existing UI
        this.clearDynamicContent();
        
        // Setup navigation based on role
        this.setupNavigationForRole(role);
        
        // Show main content container
        this.showMainContent();
        
        // Initialize role-specific UI components
        this.initializeForRole(role);
        
        console.log(`✅ UI setup completed for role: ${role}`);
    }

    /**
     * Clear dynamic content
     */
    clearDynamicContent() {
        const dynamicContent = document.getElementById('dynamic-content');
        if (dynamicContent) {
            dynamicContent.innerHTML = '';
        }
    }

    /**
     * Setup navigation for specific role
     */
    setupNavigationForRole(role) {
        const bottomNav = document.getElementById('bottom-nav');
        const ribbonMenu = document.getElementById('bottom-ribbon-menu');
        
        if (!bottomNav || !ribbonMenu) return;

        // Show bottom navigation
        bottomNav.classList.remove('hidden');
        
        // Configure navigation based on role
        const navConfig = this.getNavigationConfig(role);
        this.renderBottomNavigation(ribbonMenu, navConfig);
    }

    /**
     * Get navigation configuration for role
     */
    getNavigationConfig(role) {
        const configs = {
            'user': [
                { id: 'orders', icon: '📋', label: 'Заказы', active: true },
                { id: 'create', icon: '➕', label: 'Создать' },
                { id: 'contact', icon: '💬', label: 'Связаться' }
            ],
            'operator': [
                { id: 'orders', icon: '📋', label: 'Заказы', active: true },
                { id: 'clients', icon: '👥', label: 'Клиенты' },
                { id: 'create', icon: '➕', label: 'Создать' }
            ],
            'main_operator': [
                { id: 'orders', icon: '📋', label: 'Заказы', active: true },
                { id: 'clients', icon: '👥', label: 'Клиенты' },
                { id: 'staff', icon: '🏢', label: 'Персонал' },
                { id: 'create', icon: '➕', label: 'Создать' }
            ],
            'owner': [
                { id: 'orders', icon: '📋', label: 'Заказы', active: true },
                { id: 'clients', icon: '👥', label: 'Клиенты' },
                { id: 'staff', icon: '🏢', label: 'Персонал' },
                { id: 'analytics', icon: '📊', label: 'Аналитика' },
                { id: 'create', icon: '➕', label: 'Создать' }
            ],
            'driver': [
                { id: 'orders', icon: '🚛', label: 'Заказы', active: true },
                { id: 'statistics', icon: '📈', label: 'Статистика' }
            ]
        };

        return configs[role] || configs['operator'];
    }

    /**
     * Render bottom navigation
     */
    renderBottomNavigation(container, navConfig) {
        const navHTML = navConfig.map(item => `
            <div class="nav-item ${item.active ? 'active' : ''}" data-nav="${item.id}">
                <span class="nav-icon">${item.icon}</span>
                <span class="nav-label">${item.label}</span>
            </div>
        `).join('');

        container.innerHTML = `<div class="nav-ribbon">${navHTML}</div>`;
        
        // Add event listeners
        container.addEventListener('click', (e) => {
            const navItem = e.target.closest('.nav-item');
            if (navItem) {
                this.handleNavigationClick(navItem.dataset.nav);
            }
        });
    }

    /**
     * Handle navigation clicks
     */
    async handleNavigationClick(navId) {
        console.log(`🚀 Navigation clicked: ${navId}`);
        
        // Update active state
        document.querySelectorAll('.nav-item').forEach(item => {
            item.classList.remove('active');
        });
        document.querySelector(`[data-nav="${navId}"]`)?.classList.add('active');
        
        const userRole = this.app.state.user?.Role;
        const api = this.app.modules.get('api');
        
        switch(navId) {
            case 'orders':
                await this.handleOrdersNavigation(userRole, api);
                break;
            case 'clients':
                await this.handleClientsNavigation(api);
                break;
            case 'staff':
                await this.handleStaffNavigation(api);
                break;
            case 'create':
                this.handleCreateNavigation(userRole);
                break;
            case 'contact':
                this.handleContactNavigation();
                break;
            case 'statistics':
                this.handleStatisticsNavigation(api);
                break;
            default:
                console.warn(`Unknown navigation item: ${navId}`);
                this.app.showPanel(this.getDestinationPanel(navId));
        }
    }

    /**
     * Get destination panel for navigation item
     */
    getDestinationPanel(navId) {
        const panelMap = {
            'orders': 'operator-panel',
            'clients': 'client-management',
            'staff': 'staff-management',
            'analytics': 'analytics',
            'create': 'order-creation',
            'contact': 'contact',
            'statistics': 'statistics'
        };
        
        return panelMap[navId] || 'operator-panel';
    }

    /**
     * Handle orders navigation based on user role
     */
    async handleOrdersNavigation(userRole, api) {
        try {
            if (userRole === 'user') {
                await this.app.showPanel('user-panel');
                const orders = await api.fetchUserOrders();
                console.log('📦 User orders loaded:', orders?.length);
            } else {
                await this.app.showPanel('operator-panel');
                const orders = await api.fetchOrders('active');
                console.log('📦 Admin orders loaded:', orders?.length);
            }
        } catch (error) {
            console.error('❌ Failed to load orders:', error);
            this.showError('Не удалось загрузить заказы');
        }
    }

    /**
     * Handle clients navigation
     */
    async handleClientsNavigation(api) {
        try {
            await this.app.showPanel('clients-panel');
            const clients = await api.fetchClients();
            console.log('👥 Clients loaded:', clients?.length);
        } catch (error) {
            console.error('❌ Failed to load clients:', error);
            this.showError('Не удалось загрузить клиентов');
        }
    }

    /**
     * Handle staff navigation
     */
    async handleStaffNavigation(api) {
        await this.app.showPanel('staff-panel');
        console.log('👷 Staff panel opened');
    }

    /**
     * Handle create navigation
     */
    handleCreateNavigation(userRole) {
        if (userRole === 'user') {
            this.app.showPanel('user-order-creation');
        } else {
            this.app.showPanel('order-creation');
        }
        console.log('➕ Create panel opened for role:', userRole);
    }

    /**
     * Handle contact navigation
     */
    handleContactNavigation() {
        this.app.showPanel('contact-panel');
        console.log('💬 Contact panel opened');
    }

    /**
     * Handle statistics navigation
     */
    handleStatisticsNavigation(api) {
        this.app.showPanel('statistics-panel');
        console.log('📈 Statistics panel opened');
    }

    /**
     * Show error message
     */
    showError(message) {
        // Create or update error container
        let errorContainer = document.getElementById('error-message');
        if (!errorContainer) {
            errorContainer = document.createElement('div');
            errorContainer.id = 'error-message';
            errorContainer.className = 'error-container';
            document.getElementById('app-container').appendChild(errorContainer);
        }

        errorContainer.textContent = message;
        errorContainer.classList.remove('hidden');

        // Auto-hide after 5 seconds
        setTimeout(() => {
            errorContainer.classList.add('hidden');
        }, 5000);

        console.error('UI Error:', message);
    }

    /**
     * Show main content
     */
    showMainContent() {
        const appContainer = document.getElementById('app-container');
        if (appContainer) {
            appContainer.style.opacity = '1';
        }
    }

    /**
     * Initialize components for specific role
     */
    initializeForRole(role) {
        console.log(`🔧 Initializing components for role: ${role}`);
        
        // Initialize operator panel module for all roles
        const operatorPanel = this.app.modules.get('operatorPanel');
        if (operatorPanel && typeof operatorPanel.initializeForRole === 'function') {
            operatorPanel.initializeForRole(role);
        }
        
        // Setup FAB buttons
        this.setupFABButtons(role);
    }

    /**
     * Setup FAB (Floating Action Buttons) for role
     */
    setupFABButtons(role) {
        const fabContainer = document.getElementById('fab-container');
        if (!fabContainer) return;

        const fabConfig = this.getFABConfig(role);
        if (fabConfig.length > 0) {
            fabContainer.classList.remove('hidden');
            this.renderFABButtons(fabContainer, fabConfig);
        }
    }

    /**
     * Get FAB configuration for role
     */
    getFABConfig(role) {
        const configs = {
            'user': [
                { id: 'create-order', icon: '➕', label: 'Создать заказ' }
            ],
            'operator': [
                { id: 'create-order', icon: '➕', label: 'Создать заказ' },
                { id: 'add-client', icon: '👤', label: 'Добавить клиента' }
            ],
            'main_operator': [
                { id: 'create-order', icon: '➕', label: 'Создать заказ' },
                { id: 'add-client', icon: '👤', label: 'Добавить клиента' },
                { id: 'add-staff', icon: '🏢', label: 'Добавить персонал' }
            ],
            'owner': [
                { id: 'create-order', icon: '➕', label: 'Создать заказ' },
                { id: 'add-client', icon: '👤', label: 'Добавить клиента' },
                { id: 'add-staff', icon: '🏢', label: 'Добавить персонал' }
            ],
            'driver': []
        };

        return configs[role] || [];
    }

    /**
     * Render FAB buttons
     */
    renderFABButtons(container, fabConfig) {
        const fabHTML = fabConfig.map(item => `
            <button class="fab-button" data-action="${item.id}" title="${item.label}">
                <span class="fab-icon">${item.icon}</span>
            </button>
        `).join('');

        container.innerHTML = fabHTML;
        
        // Add event listeners
        container.addEventListener('click', (e) => {
            const fabButton = e.target.closest('.fab-button');
            if (fabButton) {
                this.handleFABClick(fabButton.dataset.action);
            }
        });
    }

    /**
     * Handle FAB button clicks
     */
    handleFABClick(actionId) {
        console.log(`🎯 FAB clicked: ${actionId}`);
        
        switch (actionId) {
            case 'create-order':
                this.app.showPanel('order-creation');
                break;
            case 'add-client':
                this.showAddClientDialog();
                break;
            case 'add-staff':
                this.showAddStaffDialog();
                break;
        }
    }

    /**
     * Show add client dialog
     */
    showAddClientDialog() {
        console.log('📱 Showing add client dialog');
        // Implementation for add client dialog
        this.app.showToast('info', 'Функция добавления клиента в разработке');
    }

    /**
     * Show add staff dialog
     */
    showAddStaffDialog() {
        console.log('🏢 Showing add staff dialog');
        // Implementation for add staff dialog
        this.app.showToast('info', 'Функция добавления персонала в разработке');
    }

    /**
     * Load templates for specific role
     */
    async loadTemplates(role) {
        const templateMap = {
            'user': ['user-panel', 'order-creation', 'order-details'],
            'operator': ['operator-panel', 'order-management', 'client-management'],
            'main_operator': ['operator-panel', 'order-management', 'client-management', 'staff-management'],
            'owner': ['operator-panel', 'order-management', 'client-management', 'staff-management', 'analytics'],
            'driver': ['driver-panel', 'order-execution']
        };

        const templates = templateMap[role] || [];
        
        for (const template of templates) {
            await this.loadTemplate(template);
        }
    }

    /**
     * Load individual template
     */
    async loadTemplate(templateName) {
        try {
            const response = await fetch(`/templates/${templateName}.html`);
            const html = await response.text();
            
            // Create template element
            const template = document.createElement('template');
            template.innerHTML = html;
            
            // Store template
            this.components.set(templateName, template);
            
        } catch (error) {
            console.warn(`⚠️ Could not load template ${templateName}, using fallback`);
            this.createFallbackTemplate(templateName);
        }
    }

    /**
     * Create fallback template if loading fails
     */
    createFallbackTemplate(templateName) {
        const fallbackTemplates = {
            'operator-panel': this.createOperatorPanelTemplate(),
            'user-panel': this.createUserPanelTemplate(),
            'order-list': this.createOrderListTemplate()
        };

        const template = document.createElement('template');
        template.innerHTML = fallbackTemplates[templateName] || '<div>Template not found</div>';
        this.components.set(templateName, template);
    }

    /**
     * Show panel with smooth transition
     */
    async showPanel(panelId, direction = 'forward') {
        const startTime = performance.now();
        
        try {
            // Get or create panel
            let panel = this.panels.get(panelId);
            if (!panel) {
                panel = await this.createPanel(panelId);
            }

            // Hide current panel
            const currentPanel = document.querySelector('.panel.visible');
            if (currentPanel && currentPanel !== panel) {
                await this.hidePanel(currentPanel, direction);
            }

            // Show new panel
            await this.displayPanel(panel, direction);
            
            // Update navigation state
            this.updateNavigationState(panelId);
            
            const transitionTime = performance.now() - startTime;
            console.log(`✨ Panel transition completed in ${transitionTime.toFixed(2)}ms`);
            
        } catch (error) {
            console.error(`❌ Error showing panel ${panelId}:`, error);
            this.app.handleError(error);
        }
    }

    /**
     * Create panel dynamically
     */
    async createPanel(panelId) {
        const container = document.getElementById('dynamic-content');
        if (!container) {
            throw new Error('Dynamic content container not found');
        }

        // Create panel element
        const panel = document.createElement('div');
        panel.id = panelId;
        panel.className = 'panel';
        
        // Load panel content
        const content = await this.getPanelContent(panelId);
        panel.innerHTML = content;
        
        // Append to container
        container.appendChild(panel);
        
        // Initialize panel components
        this.initializePanelComponents(panel);
        
        // Store panel reference
        this.panels.set(panelId, panel);
        
        return panel;
    }

    /**
     * Get panel content for specific panel
     */
    async getPanelContent(panelId) {
        try {
            console.log(`🎯 Getting content for panel: ${panelId}`);
            
            switch (panelId) {
                case 'operator-panel':
                    return this.getOperatorPanelContent();
                    
                case 'orders-panel':
                    return this.getOrdersPanelContent();
                    
                case 'clients-panel':
                    return this.getClientsPanelContent();
                    
                default:
                    console.warn(`Unknown panel: ${panelId}`);
                    return `<div class="panel-content">
                        <h2>Панель ${panelId}</h2>
                        <p>Содержимое панели загружается...</p>
                    </div>`;
            }
        } catch (error) {
            console.error(`❌ Failed to get panel content for ${panelId}:`, error);
            return `<div class="panel-content error">
                <h2>Ошибка загрузки</h2>
                <p>Не удалось загрузить содержимое панели ${panelId}</p>
            </div>`;
        }
    }

    /**
     * Get operator panel content
     */
    getOperatorPanelContent() {
        return `
            <div class="operator-panel">
                <div class="panel-header">
                    <h1>Панель оператора</h1>
                    <div class="panel-controls">
                        <button class="button button-primary" id="refresh-orders">
                            <i class="fas fa-sync-alt"></i> Обновить
                        </button>
                    </div>
                </div>
                
                <div class="status-tabs" id="status-tabs">
                    <div class="tab active" data-status="active">Активные</div>
                    <div class="tab" data-status="new">Новые</div>
                    <div class="tab" data-status="in_progress">В работе</div>
                    <div class="tab" data-status="completed">Завершённые</div>
                </div>
                
                <div class="orders-container" id="orders-container">
                    <div class="loading-skeleton">
                        <div class="skeleton" style="height: 80px; margin-bottom: 16px;"></div>
                        <div class="skeleton" style="height: 80px; margin-bottom: 16px;"></div>
                        <div class="skeleton" style="height: 80px; margin-bottom: 16px;"></div>
                    </div>
                </div>
            </div>
        `;
    }

    /**
     * Get orders panel content
     */
    getOrdersPanelContent() {
        return `
            <div class="orders-panel">
                <div class="panel-header">
                    <h1>Заказы</h1>
                </div>
                <div class="orders-list" id="orders-list">
                    <div class="loading-skeleton">
                        <div class="skeleton" style="height: 60px; margin-bottom: 12px;"></div>
                        <div class="skeleton" style="height: 60px; margin-bottom: 12px;"></div>
                        <div class="skeleton" style="height: 60px; margin-bottom: 12px;"></div>
                    </div>
                </div>
            </div>
        `;
    }

    /**
     * Get clients panel content
     */
    getClientsPanelContent() {
        return `
            <div class="clients-panel">
                <div class="panel-header">
                    <h1>Клиенты</h1>
                </div>
                <div class="clients-list" id="clients-list">
                    <div class="loading-skeleton">
                        <div class="skeleton" style="height: 60px; margin-bottom: 12px;"></div>
                        <div class="skeleton" style="height: 60px; margin-bottom: 12px;"></div>
                        <div class="skeleton" style="height: 60px; margin-bottom: 12px;"></div>
                    </div>
                </div>
            </div>
        `;
    }

    /**
     * Create operator panel with virtualized order list
     */
    createOperatorPanelContent() {
        return `
            <div class="panel-header">
                <h2>Заказы</h2>
                <button class="btn btn-secondary" id="refresh-orders">
                    <span class="icon">🔄</span>
                </button>
            </div>
            
            <div class="status-tabs" id="status-tabs">
                <div class="tab active" data-status="active">Активные</div>
                <div class="tab" data-status="new">Новые</div>
                <div class="tab" data-status="in_progress">В работе</div>
                <div class="tab" data-status="completed">Завершённые</div>
            </div>
            
            <div class="search-container">
                <input type="search" 
                       id="orders-search" 
                       placeholder="Поиск заказов..." 
                       class="search-input">
            </div>
            
            <div class="virtualized-list" 
                 id="orders-list" 
                 data-item-height="80"
                 style="height: calc(100vh - 200px);">
                <div class="list-viewport"></div>
                <div class="scrollbar-track">
                    <div class="scrollbar-thumb"></div>
                </div>
            </div>
        `;
    }

    /**
     * Create user panel content
     */
    createUserPanelContent() {
        return `
            <div class="panel-header">
                <h2>Мои заказы</h2>
            </div>
            
            <div class="user-orders-container">
                <div class="quick-actions">
                    <button class="btn btn-primary" id="create-order-btn">
                        <span class="icon">➕</span>
                        Создать заказ
                    </button>
                    <button class="btn btn-secondary" id="contact-operator-btn">
                        <span class="icon">💬</span>
                        Связаться с оператором
                    </button>
                </div>
                
                <div class="orders-list" id="user-orders-list">
                    <!-- Orders will be loaded here -->
                </div>
            </div>
        `;
    }

    /**
     * Initialize virtualized list
     */
    initializeVirtualizedList(containerId, items = [], renderItem) {
        const container = document.getElementById(containerId);
        if (!container || !this.settings.enableVirtualization) {
            return this.initializeRegularList(containerId, items, renderItem);
        }

        const virtualList = new VirtualizedList(container, {
            items,
            itemHeight: this.settings.virtualItemHeight,
            buffer: this.settings.virtualBuffer,
            renderItem,
            onScroll: (scrollTop) => this.handleVirtualScroll(containerId, scrollTop)
        });

        this.virtualizedLists.set(containerId, virtualList);
        return virtualList;
    }

    /**
     * Update virtualized list items
     */
    updateVirtualizedList(containerId, items) {
        const virtualList = this.virtualizedLists.get(containerId);
        if (virtualList) {
            virtualList.updateItems(items);
        }
    }

    /**
     * Create order card renderer for virtualized list
     */
    createOrderCardRenderer() {
        return (order, index) => {
            const statusColors = {
                'new': '#007aff',
                'in_progress': '#ff9500',
                'completed': '#28a745',
                'canceled': '#dc3545'
            };

            return `
                <div class="card order-card" 
                     data-order-id="${order.ID}"
                     data-analytics="order_clicked">
                    <div class="card-header">
                        <span class="order-id">№${order.ID}</span>
                        <span class="order-status" 
                              style="background: ${statusColors[order.Status] || '#666'}">
                            ${this.getStatusDisplayName(order.Status)}
                        </span>
                    </div>
                    <div class="card-body">
                        <div class="order-info">
                            <div class="info-row">
                                <span class="icon">👤</span>
                                <span class="text">${order.Name}</span>
                            </div>
                            <div class="info-row">
                                <span class="icon">📍</span>
                                <span class="text">${order.Address}</span>
                            </div>
                            <div class="info-row">
                                <span class="icon">📅</span>
                                <span class="text">${this.formatDate(order.Date)}</span>
                            </div>
                        </div>
                    </div>
                    <div class="card-footer">
                        <span class="order-cost">
                            ${order.Cost?.Valid ? order.Cost.Float64 + '₽' : 'Не указано'}
                        </span>
                        <button class="btn btn-primary btn-sm view-order-btn">
                            Открыть
                        </button>
                    </div>
                </div>
            `;
        };
    }

    /**
     * Show toast notification
     */
    showToast(type, message, duration = 3000) {
        const container = document.getElementById('toast-container');
        if (!container) return;

        const toast = document.createElement('div');
        toast.className = `toast ${type}`;
        toast.innerHTML = `
            <div class="toast-content">
                <div class="toast-icon">${this.getToastIcon(type)}</div>
                <div class="toast-message">${message}</div>
                <button class="toast-close">✕</button>
            </div>
        `;

        // Add to container
        container.appendChild(toast);

        // Animate in
        requestAnimationFrame(() => {
            toast.classList.add('show');
        });

        // Auto remove
        const removeToast = () => {
            toast.classList.remove('show');
            setTimeout(() => {
                if (toast.parentNode) {
                    toast.parentNode.removeChild(toast);
                }
            }, 300);
        };

        setTimeout(removeToast, duration);

        // Manual close
        toast.querySelector('.toast-close').addEventListener('click', removeToast);
    }

    /**
     * Show/hide progress bar
     */
    showProgressBar(show = true) {
        const progressBar = document.getElementById('top-progress-bar');
        if (progressBar) {
            if (show) {
                progressBar.classList.remove('hidden');
                progressBar.classList.add('active');
            } else {
                progressBar.classList.remove('active');
                setTimeout(() => {
                    progressBar.classList.add('hidden');
                }, 300);
            }
        }
    }

    /**
     * Show skeleton loading state
     */
    showSkeletonLoading(containerId, count = 5) {
        const container = document.getElementById(containerId);
        if (!container) return;

        const skeletons = Array.from({ length: count }, (_, i) => `
            <div class="skeleton skeleton-card" style="animation-delay: ${i * 100}ms"></div>
        `).join('');

        container.innerHTML = `<div class="skeleton-container">${skeletons}</div>`;
    }

    /**
     * Hide skeleton and show content
     */
    hideSkeletonLoading(containerId) {
        const container = document.getElementById(containerId);
        if (!container) return;

        const skeletonContainer = container.querySelector('.skeleton-container');
        if (skeletonContainer) {
            skeletonContainer.style.opacity = '0';
            setTimeout(() => {
                skeletonContainer.remove();
            }, 300);
        }
    }

    /**
     * Setup navigation
     */
    setupNavigation(role) {
        const navItems = this.getNavigationItems(role);
        const bottomNav = document.getElementById('bottom-nav');
        
        if (bottomNav && navItems.length > 0) {
            bottomNav.innerHTML = navItems.map(item => `
                <button class="nav-item" 
                        data-panel="${item.panel}"
                        data-analytics="nav_${item.panel}">
                    <span class="nav-icon">${item.icon}</span>
                    <span class="nav-label">${item.label}</span>
                </button>
            `).join('');

            bottomNav.classList.remove('hidden');
            this.bindNavigationEvents(bottomNav);
        }
    }

    /**
     * Get navigation items for role
     */
    getNavigationItems(role) {
        const navigationMap = {
            'user': [
                { panel: 'user-panel', icon: '📦', label: 'Заказы' },
                { panel: 'create-order', icon: '➕', label: 'Создать' },
                { panel: 'contact', icon: '💬', label: 'Связь' }
            ],
            'operator': [
                { panel: 'operator-panel', icon: '📋', label: 'Заказы' },
                { panel: 'clients', icon: '👥', label: 'Клиенты' },
                { panel: 'create-order', icon: '➕', label: 'Создать' }
            ],
            'main_operator': [
                { panel: 'operator-panel', icon: '📋', label: 'Заказы' },
                { panel: 'clients', icon: '👥', label: 'Клиенты' },
                { panel: 'staff', icon: '👷', label: 'Штат' },
                { panel: 'analytics', icon: '📊', label: 'Аналитика' }
            ],
            'owner': [
                { panel: 'operator-panel', icon: '📋', label: 'Заказы' },
                { panel: 'clients', icon: '👥', label: 'Клиенты' },
                { panel: 'staff', icon: '👷', label: 'Штат' },
                { panel: 'analytics', icon: '📊', label: 'Аналитика' },
                { panel: 'financials', icon: '💰', label: 'Финансы' }
            ],
            'driver': [
                { panel: 'driver-panel', icon: '🚗', label: 'Мои заказы' },
                { panel: 'statistics', icon: '📈', label: 'Статистика' }
            ]
        };

        return navigationMap[role] || [];
    }

    /**
     * Bind navigation events
     */
    bindNavigationEvents(bottomNav) {
        bottomNav.addEventListener('click', (event) => {
            const navItem = event.target.closest('.nav-item');
            if (navItem) {
                const panelId = navItem.dataset.panel;
                this.app.showPanel(panelId);
                
                // Update active state
                bottomNav.querySelectorAll('.nav-item').forEach(item => {
                    item.classList.remove('active');
                });
                navItem.classList.add('active');
            }
        });
    }

    /**
     * Setup touch gestures
     */
    setupTouchGestures() {
        let startX, startY, startTime;

        document.addEventListener('touchstart', (e) => {
            startX = e.touches[0].clientX;
            startY = e.touches[0].clientY;
            startTime = Date.now();
        }, { passive: true });

        document.addEventListener('touchend', (e) => {
            if (!startX || !startY) return;

            const endX = e.changedTouches[0].clientX;
            const endY = e.changedTouches[0].clientY;
            const endTime = Date.now();

            const deltaX = endX - startX;
            const deltaY = endY - startY;
            const deltaTime = endTime - startTime;

            // Swipe detection
            if (deltaTime < 300 && Math.abs(deltaX) > 50 && Math.abs(deltaY) < 100) {
                if (deltaX > 0) {
                    this.handleSwipeRight();
                } else {
                    this.handleSwipeLeft();
                }
            }

            startX = startY = null;
        }, { passive: true });
    }

    /**
     * Setup keyboard navigation
     */
    setupKeyboardNavigation() {
        document.addEventListener('keydown', (e) => {
            if (e.key === 'Escape') {
                this.handleEscapeKey();
            } else if (e.key === 'Enter') {
                this.handleEnterKey(e);
            }
        });
    }

    /**
     * Handle intersection for lazy loading
     */
    handleIntersection(entries) {
        entries.forEach(entry => {
            if (entry.isIntersecting) {
                const element = entry.target;
                this.lazyLoadElement(element);
                this.intersectionObserver.unobserve(element);
            }
        });
    }

    /**
     * Lazy load element
     */
    lazyLoadElement(element) {
        const src = element.dataset.src;
        const component = element.dataset.component;

        if (src) {
            element.src = src;
        }

        if (component) {
            this.loadLazyComponent(element, component);
        }
    }

    /**
     * Utility methods
     */
    clearDynamicContent() {
        const container = document.getElementById('dynamic-content');
        if (container) {
            container.innerHTML = '';
        }
        this.panels.clear();
        this.virtualizedLists.clear();
    }

    getStatusDisplayName(status) {
        const statusMap = {
            'new': 'Новый',
            'in_progress': 'В работе',
            'completed': 'Завершён',
            'canceled': 'Отменён',
            'awaiting_confirmation': 'Ожидает подтверждения'
        };
        return statusMap[status] || status;
    }

    formatDate(dateString) {
        return new Date(dateString).toLocaleDateString('ru-RU', {
            day: '2-digit',
            month: '2-digit',
            year: '2-digit'
        });
    }

    getToastIcon(type) {
        const icons = {
            'success': '✅',
            'error': '❌',
            'warning': '⚠️',
            'info': 'ℹ️'
        };
        return icons[type] || 'ℹ️';
    }

    // Animation methods
    async hidePanel(panel, direction) {
        const animationClass = direction === 'forward' ? 'slide-out-left' : 'slide-out-right';
        panel.classList.add(animationClass);
        
        await this.waitForAnimation(panel);
        
        panel.classList.remove('visible', animationClass);
    }

    async displayPanel(panel, direction) {
        const animationClass = direction === 'forward' ? 'slide-in-right' : 'slide-in-left';
        
        panel.classList.add('visible', animationClass);
        
        await this.waitForAnimation(panel);
        
        panel.classList.remove(animationClass);
    }

    waitForAnimation(element) {
        return new Promise(resolve => {
            const handleAnimationEnd = () => {
                element.removeEventListener('animationend', handleAnimationEnd);
                resolve();
            };
            element.addEventListener('animationend', handleAnimationEnd);
        });
    }
}

/**
 * Virtualized List implementation for performance
 */
class VirtualizedList {
    constructor(container, options) {
        this.container = container;
        this.viewport = container.querySelector('.list-viewport');
        this.options = {
            itemHeight: 80,
            buffer: 5,
            ...options
        };
        
        this.items = options.items || [];
        this.visibleRange = { start: 0, end: 0 };
        this.scrollTop = 0;
        
        this.setupVirtualization();
        this.render();
    }

    setupVirtualization() {
        // Set up scrolling
        this.container.addEventListener('scroll', (e) => {
            this.scrollTop = e.target.scrollTop;
            this.updateVisibleRange();
            this.render();
            
            if (this.options.onScroll) {
                this.options.onScroll(this.scrollTop);
            }
        });

        // Set container height
        this.updateContainerHeight();
    }

    updateItems(items) {
        this.items = items;
        this.updateContainerHeight();
        this.updateVisibleRange();
        this.render();
    }

    updateContainerHeight() {
        const totalHeight = this.items.length * this.options.itemHeight;
        this.viewport.style.height = totalHeight + 'px';
    }

    updateVisibleRange() {
        const containerHeight = this.container.clientHeight;
        const start = Math.max(0, Math.floor(this.scrollTop / this.options.itemHeight) - this.options.buffer);
        const visibleCount = Math.ceil(containerHeight / this.options.itemHeight);
        const end = Math.min(this.items.length, start + visibleCount + this.options.buffer * 2);
        
        this.visibleRange = { start, end };
    }

    render() {
        const { start, end } = this.visibleRange;
        const visibleItems = this.items.slice(start, end);
        
        const html = visibleItems.map((item, index) => {
            const actualIndex = start + index;
            const top = actualIndex * this.options.itemHeight;
            
            return `
                <div class="virtual-item" 
                     style="position: absolute; top: ${top}px; height: ${this.options.itemHeight}px; width: 100%;">
                    ${this.options.renderItem(item, actualIndex)}
                </div>
            `;
        }).join('');
        
        this.viewport.innerHTML = html;
    }
}

// Делаем UIModule доступным глобально
window.UIModule = UIModule; 