/**
 * @fileoverview Operator Panel Module with real-time updates and performance optimizations
 * @version 2.0
 */

class OperatorPanelModule {
    constructor(app) {
        this.app = app;
        this.currentStatus = 'active';
        this.currentPage = 1;
        this.itemsPerPage = 20;
        this.searchQuery = '';
        this.orders = new Map();
        this.filteredOrders = [];
        
        // Real-time updates
        this.pollInterval = null;
        this.lastUpdate = null;
        
        // Performance tracking
        this.renderTimes = [];
        this.isLoading = false;
        
        // Filters and sorting
        this.filters = {
            status: 'active',
            dateRange: null,
            costRange: null,
            category: null
        };
        
        this.sortConfig = {
            field: 'CreatedAt',
            direction: 'desc'
        };

        this.setupEventListeners();
        this.setupRealTimeUpdates();
    }

    /**
     * Initialize the operator panel
     */
    async initialize() {
        console.log('📋 Initializing Operator Panel');
        
        try {
            // Setup UI
            await this.setupUI();
            
            // Load initial data
            await this.loadOrders();
            
            // Start real-time updates
            this.startRealTimeUpdates();
            
            console.log('✅ Operator Panel initialized successfully');
            
        } catch (error) {
            console.error('❌ Failed to initialize Operator Panel:', error);
            this.app.handleError(error);
        }
    }

    /**
     * Setup the operator panel UI
     */
    async setupUI() {
        const ui = this.app.modules.get('ui');
        if (!ui) throw new Error('UI module not available');

        // Create status tabs
        this.createStatusTabs();
        
        // Create search and filters
        this.createSearchAndFilters();
        
        // Initialize virtualized list
        this.initializeOrdersList();
        
        // Setup FAB for creating orders
        this.setupCreateOrderFAB();
    }

    /**
     * Create status tabs for order filtering
     */
    createStatusTabs() {
        const statusTabs = document.getElementById('status-tabs');
        if (!statusTabs) return;

        const statuses = [
            { key: 'active', label: 'Активные', color: '#007aff' },
            { key: 'new', label: 'Новые', color: '#ff9500' },
            { key: 'in_progress', label: 'В работе', color: '#ff9500' },
            { key: 'awaiting_confirmation', label: 'Ожидают', color: '#ff3b30' },
            { key: 'completed', label: 'Завершённые', color: '#28a745' },
            { key: 'canceled', label: 'Отменённые', color: '#8e8e93' }
        ];

        statusTabs.innerHTML = statuses.map(status => `
            <div class="tab ${status.key === this.currentStatus ? 'active' : ''}" 
                 data-status="${status.key}"
                 data-analytics="status_tab_${status.key}">
                <span class="tab-label">${status.label}</span>
                <span class="tab-count" id="count-${status.key}">0</span>
            </div>
        `).join('');

        // Add tab switching functionality
        statusTabs.addEventListener('click', (e) => {
            const tab = e.target.closest('.tab');
            if (tab) {
                this.switchStatus(tab.dataset.status);
            }
        });
    }

    /**
     * Create search and filter controls
     */
    createSearchAndFilters() {
        const searchContainer = document.querySelector('.search-container');
        if (!searchContainer) return;

        searchContainer.innerHTML = `
            <div class="search-bar">
                <input type="search" 
                       id="orders-search" 
                       placeholder="Поиск по ID, имени, адресу, телефону..." 
                       class="search-input"
                       autocomplete="off">
                <button class="search-clear-btn hidden" id="clear-search">✕</button>
            </div>
            
            <div class="filters-row">
                <select id="category-filter" class="filter-select">
                    <option value="">Все категории</option>
                    <option value="waste_removal">Вывоз мусора</option>
                    <option value="demolition">Демонтаж</option>
                </select>
                
                <select id="sort-select" class="filter-select">
                    <option value="CreatedAt:desc">Сначала новые</option>
                    <option value="CreatedAt:asc">Сначала старые</option>
                    <option value="Cost:desc">По убыванию стоимости</option>
                    <option value="Cost:asc">По возрастанию стоимости</option>
                </select>
                
                <button class="btn btn-secondary" id="advanced-filters">
                    <span class="icon">🔍</span>
                    Фильтры
                </button>
            </div>
        `;

        this.setupSearchFunctionality();
        this.setupFilterFunctionality();
    }

    /**
     * Setup search functionality with debouncing
     */
    setupSearchFunctionality() {
        const searchInput = document.getElementById('orders-search');
        const clearButton = document.getElementById('clear-search');
        
        if (!searchInput) return;

        let searchTimeout;
        
        searchInput.addEventListener('input', (e) => {
            const value = e.target.value.trim();
            
            // Show/hide clear button
            clearButton.classList.toggle('hidden', !value);
            
            // Debounced search
            clearTimeout(searchTimeout);
            searchTimeout = setTimeout(() => {
                this.performSearch(value);
            }, 300);
        });

        // Clear search
        if (clearButton) {
            clearButton.addEventListener('click', () => {
                searchInput.value = '';
                clearButton.classList.add('hidden');
                this.performSearch('');
            });
        }
    }

    /**
     * Setup filter functionality
     */
    setupFilterFunctionality() {
        const categoryFilter = document.getElementById('category-filter');
        const sortSelect = document.getElementById('sort-select');
        const advancedFiltersBtn = document.getElementById('advanced-filters');

        if (categoryFilter) {
            categoryFilter.addEventListener('change', (e) => {
                this.filters.category = e.target.value || null;
                this.applyFilters();
            });
        }

        if (sortSelect) {
            sortSelect.addEventListener('change', (e) => {
                const [field, direction] = e.target.value.split(':');
                this.sortConfig = { field, direction };
                this.applySorting();
            });
        }

        if (advancedFiltersBtn) {
            advancedFiltersBtn.addEventListener('click', () => {
                this.showAdvancedFilters();
            });
        }
    }

    /**
     * Initialize orders list with virtualization
     */
    initializeOrdersList() {
        const ui = this.app.modules.get('ui');
        if (!ui) return;

        this.virtualList = ui.initializeVirtualizedList(
            'orders-list',
            [],
            this.createOrderRenderer()
        );

        // Setup list event delegation
        const listContainer = document.getElementById('orders-list');
        if (listContainer) {
            listContainer.addEventListener('click', (e) => {
                this.handleOrderListClick(e);
            });
        }
    }

    /**
     * Create order renderer for virtualized list
     */
    createOrderRenderer() {
        return (order, index) => {
            const statusConfig = this.getStatusConfig(order.Status);
            const costDisplay = order.Cost?.Valid ? 
                `${order.Cost.Float64.toLocaleString('ru-RU')} ₽` : 
                'Не указано';

            return `
                <div class="order-card" 
                     data-order-id="${order.ID}"
                     data-analytics="order_card_clicked">
                    <div class="card-header">
                        <div class="order-meta">
                            <span class="order-id">№${order.ID}</span>
                            <span class="order-date">${this.formatDateTime(order.CreatedAt)}</span>
                        </div>
                        <span class="status-badge ${order.Status}" 
                              style="background: ${statusConfig.color}">
                            ${statusConfig.label}
                        </span>
                    </div>
                    
                    <div class="card-body">
                        <div class="order-info">
                            <div class="info-row primary">
                                <span class="icon">👤</span>
                                <div class="info-content">
                                    <strong>${order.Name}</strong>
                                    ${order.Phone ? `<small>${this.formatPhone(order.Phone)}</small>` : ''}
                                </div>
                            </div>
                            
                            <div class="info-row">
                                <span class="icon">📍</span>
                                <span class="info-text">${order.Address}</span>
                            </div>
                            
                            <div class="info-row">
                                <span class="icon">🏷️</span>
                                <span class="info-text">${this.getCategoryLabel(order.Category)} → ${order.Subcategory}</span>
                            </div>
                            
                            ${order.Description ? `
                                <div class="info-row description">
                                    <span class="icon">📝</span>
                                    <span class="info-text">${this.truncateText(order.Description, 80)}</span>
                                </div>
                            ` : ''}
                        </div>
                    </div>
                    
                    <div class="card-footer">
                        <div class="order-cost">
                            <span class="cost-amount">${costDisplay}</span>
                            ${order.Payment ? `<small>${this.getPaymentLabel(order.Payment)}</small>` : ''}
                        </div>
                        
                        <div class="action-buttons">
                            ${this.generateActionButtons(order)}
                        </div>
                    </div>
                    
                    ${this.shouldShowUrgentIndicator(order) ? `
                        <div class="urgent-indicator">
                            <span class="icon">⚡</span>
                            Срочно
                        </div>
                    ` : ''}
                </div>
            `;
        };
    }

    /**
     * Load orders from API
     */
    async loadOrders(showLoader = true) {
        if (this.isLoading) return;
        
        this.isLoading = true;
        const startTime = performance.now();
        
        try {
            if (showLoader) {
                this.showLoadingState();
            }

            const api = this.app.modules.get('api');
            const response = await api.fetchOrders(
                this.currentStatus,
                this.currentPage,
                this.itemsPerPage
            );

            // Update orders data
            this.updateOrdersData(response);
            
            // Apply current filters and sorting
            this.applyFiltersAndSorting();
            
            // Update UI
            this.renderOrders();
            this.updateStatusCounts();
            
            // Track performance
            const renderTime = performance.now() - startTime;
            this.renderTimes.push(renderTime);
            console.log(`📊 Orders loaded and rendered in ${renderTime.toFixed(2)}ms`);
            
            this.lastUpdate = Date.now();
            
        } catch (error) {
            console.error('❌ Failed to load orders:', error);
            this.showErrorState(error.message);
            this.app.handleError(error);
        } finally {
            this.isLoading = false;
            this.hideLoadingState();
        }
    }

    /**
     * Update orders data from API response
     */
    updateOrdersData(response) {
        // Clear existing orders for this status
        for (const [id, order] of this.orders.entries()) {
            if (order.Status === this.currentStatus) {
                this.orders.delete(id);
            }
        }

        // Add new orders
        if (response.orders && Array.isArray(response.orders)) {
            response.orders.forEach(order => {
                this.orders.set(order.ID, order);
            });
        }

        // Update pagination info
        this.totalItems = response.total || 0;
        this.totalPages = Math.ceil(this.totalItems / this.itemsPerPage);
    }

    /**
     * Switch order status tab
     */
    async switchStatus(status) {
        if (status === this.currentStatus) return;

        console.log(`🔄 Switching to status: ${status}`);
        
        // Update UI
        document.querySelectorAll('.tab').forEach(tab => {
            tab.classList.toggle('active', tab.dataset.status === status);
        });

        // Update state
        this.currentStatus = status;
        this.currentPage = 1;
        this.searchQuery = '';
        
        // Clear search
        const searchInput = document.getElementById('orders-search');
        if (searchInput) {
            searchInput.value = '';
        }

        // Load new data
        await this.loadOrders();
    }

    /**
     * Perform search with highlighting
     */
    async performSearch(query) {
        this.searchQuery = query.toLowerCase();
        
        if (query.length === 0) {
            // Reset to show all orders
            this.applyFiltersAndSorting();
            return;
        }

        // Filter orders locally first for instant feedback
        this.applyFiltersAndSorting();
        
        // If we have few results, perform server search
        if (this.filteredOrders.length < 5 && query.length > 2) {
            await this.performServerSearch(query);
        }
    }

    /**
     * Perform server-side search
     */
    async performServerSearch(query) {
        try {
            const api = this.app.modules.get('api');
            const response = await api.searchOrders(query, this.currentStatus);
            
            if (response.orders) {
                // Update orders with search results
                response.orders.forEach(order => {
                    this.orders.set(order.ID, order);
                });
                
                this.applyFiltersAndSorting();
            }
        } catch (error) {
            console.warn('Server search failed, using local search only:', error);
        }
    }

    /**
     * Apply filters and sorting
     */
    applyFiltersAndSorting() {
        let filtered = Array.from(this.orders.values())
            .filter(order => order.Status === this.currentStatus);

        // Apply search filter
        if (this.searchQuery) {
            filtered = filtered.filter(order => 
                this.matchesSearch(order, this.searchQuery)
            );
        }

        // Apply category filter
        if (this.filters.category) {
            filtered = filtered.filter(order => 
                order.Category === this.filters.category
            );
        }

        // Apply date range filter
        if (this.filters.dateRange) {
            filtered = filtered.filter(order => 
                this.isInDateRange(order.CreatedAt, this.filters.dateRange)
            );
        }

        // Apply cost range filter
        if (this.filters.costRange) {
            filtered = filtered.filter(order => 
                this.isInCostRange(order.Cost, this.filters.costRange)
            );
        }

        // Apply sorting
        filtered.sort((a, b) => this.compareOrders(a, b));

        this.filteredOrders = filtered;
        this.renderOrders();
    }

    /**
     * Check if order matches search query
     */
    matchesSearch(order, query) {
        const searchFields = [
            order.ID?.toString(),
            order.Name,
            order.Phone,
            order.Address,
            order.Description,
            order.Category,
            order.Subcategory
        ];

        return searchFields.some(field => 
            field && field.toString().toLowerCase().includes(query)
        );
    }

    /**
     * Compare orders for sorting
     */
    compareOrders(a, b) {
        const { field, direction } = this.sortConfig;
        let valueA = a[field];
        let valueB = b[field];

        // Handle special cases
        if (field === 'Cost') {
            valueA = a.Cost?.Valid ? a.Cost.Float64 : 0;
            valueB = b.Cost?.Valid ? b.Cost.Float64 : 0;
        }

        // Handle date fields
        if (field.includes('At') || field.includes('Date')) {
            valueA = new Date(valueA).getTime();
            valueB = new Date(valueB).getTime();
        }

        // Compare values
        let result = 0;
        if (valueA < valueB) result = -1;
        else if (valueA > valueB) result = 1;

        return direction === 'desc' ? -result : result;
    }

    /**
     * Render orders in the virtualized list
     */
    renderOrders() {
        const ui = this.app.modules.get('ui');
        if (ui && this.virtualList) {
            ui.updateVirtualizedList('orders-list', this.filteredOrders);
        }

        // Update results count
        this.updateResultsCount();
    }

    /**
     * Update results count display
     */
    updateResultsCount() {
        const countElement = document.getElementById(`count-${this.currentStatus}`);
        if (countElement) {
            countElement.textContent = this.filteredOrders.length;
        }

        // Update search results indicator
        if (this.searchQuery) {
            this.showSearchResults(this.filteredOrders.length);
        }
    }

    /**
     * Handle clicks in the orders list
     */
    handleOrderListClick(event) {
        const orderCard = event.target.closest('.order-card');
        const actionButton = event.target.closest('.action-btn');
        
        if (actionButton) {
            event.stopPropagation();
            this.handleActionButton(actionButton);
        } else if (orderCard) {
            const orderId = parseInt(orderCard.dataset.orderId);
            this.openOrderDetails(orderId);
        }
    }

    /**
     * Handle action button clicks
     */
    handleActionButton(button) {
        const action = button.dataset.action;
        const orderId = parseInt(button.closest('.order-card').dataset.orderId);
        
        switch (action) {
            case 'set_cost':
                this.showSetCostModal(orderId);
                break;
            case 'complete':
                this.completeOrder(orderId);
                break;
            case 'cancel':
                this.showCancelOrderModal(orderId);
                break;
            case 'assign':
                this.showAssignModal(orderId);
                break;
            default:
                console.warn('Unknown action:', action);
        }
    }

    /**
     * Setup real-time updates
     */
    setupRealTimeUpdates() {
        // Check for updates every 30 seconds
        this.pollInterval = setInterval(() => {
            if (!document.hidden && !this.isLoading) {
                this.checkForUpdates();
            }
        }, 30000);

        // Check when tab becomes visible
        document.addEventListener('visibilitychange', () => {
            if (!document.hidden && this.lastUpdate && Date.now() - this.lastUpdate > 60000) {
                this.checkForUpdates();
            }
        });
    }

    /**
     * Check for updates from server
     */
    async checkForUpdates() {
        try {
            const api = this.app.modules.get('api');
            const response = await api.getOrderUpdates(this.lastUpdate, this.currentStatus);

            if (response.hasUpdates && response.orders?.length > 0) {
                console.log(`🔄 Received ${response.orders.length} order updates`);
                
                // Update local data
                response.orders.forEach(order => {
                    this.orders.set(order.ID, order);
                });

                // Re-apply filters and render
                this.applyFiltersAndSorting();
                
                // Show notification for new orders
                if (response.newOrdersCount > 0) {
                    this.app.showToast('info', `Получено ${response.newOrdersCount} новых заказов`);
                }
            }
        } catch (error) {
            console.warn('Failed to check for updates:', error);
        }
    }

    /**
     * Start real-time updates
     */
    startRealTimeUpdates() {
        if (this.pollInterval) {
            clearInterval(this.pollInterval);
        }
        this.setupRealTimeUpdates();
    }

    /**
     * Stop real-time updates
     */
    stopRealTimeUpdates() {
        if (this.pollInterval) {
            clearInterval(this.pollInterval);
            this.pollInterval = null;
        }
    }

    /**
     * Utility methods
     */
    getStatusConfig(status) {
        const configs = {
            'new': { label: 'Новый', color: '#ff9500' },
            'awaiting_confirmation': { label: 'Ожидает', color: '#ff3b30' },
            'in_progress': { label: 'В работе', color: '#007aff' },
            'completed': { label: 'Завершён', color: '#28a745' },
            'canceled': { label: 'Отменён', color: '#8e8e93' }
        };
        return configs[status] || { label: status, color: '#8e8e93' };
    }

    getCategoryLabel(category) {
        const labels = {
            'waste_removal': 'Вывоз мусора',
            'demolition': 'Демонтаж'
        };
        return labels[category] || category;
    }

    formatDateTime(dateString) {
        return new Date(dateString).toLocaleString('ru-RU', {
            day: '2-digit',
            month: '2-digit',
            hour: '2-digit',
            minute: '2-digit'
        });
    }

    formatPhone(phone) {
        // Format phone number: +7 (978) 900-02-30
        const cleaned = phone.replace(/\D/g, '');
        if (cleaned.length === 11 && cleaned.startsWith('7')) {
            return `+${cleaned[0]} (${cleaned.slice(1, 4)}) ${cleaned.slice(4, 7)}-${cleaned.slice(7, 9)}-${cleaned.slice(9)}`;
        }
        return phone;
    }

    truncateText(text, maxLength) {
        if (text.length <= maxLength) return text;
        return text.substr(0, maxLength) + '...';
    }

    shouldShowUrgentIndicator(order) {
        // Show urgent indicator for orders older than 2 hours without response
        if (order.Status !== 'new') return false;
        
        const orderTime = new Date(order.CreatedAt).getTime();
        const now = Date.now();
        const twoHours = 2 * 60 * 60 * 1000;
        
        return (now - orderTime) > twoHours;
    }

    generateActionButtons(order) {
        const buttons = [];
        const userRole = this.app.state.user?.Role;
        
        if (!['operator', 'main_operator', 'owner'].includes(userRole)) {
            return '';
        }

        switch (order.Status) {
            case 'new':
                buttons.push(`<button class="action-btn btn-warning" data-action="set_cost">💰 Оценить</button>`);
                buttons.push(`<button class="action-btn btn-danger" data-action="cancel">❌ Отменить</button>`);
                break;
            case 'awaiting_confirmation':
                buttons.push(`<button class="action-btn btn-primary" data-action="set_cost">✏️ Изменить цену</button>`);
                buttons.push(`<button class="action-btn btn-danger" data-action="cancel">❌ Отменить</button>`);
                break;
            case 'in_progress':
                buttons.push(`<button class="action-btn btn-success" data-action="complete">✅ Завершить</button>`);
                if (!order.Cost?.Valid) {
                    buttons.push(`<button class="action-btn btn-warning" data-action="set_cost">💰 Оценить</button>`);
                }
                break;
        }

        return buttons.join('');
    }

    // Loading states
    showLoadingState() {
        const ui = this.app.modules.get('ui');
        if (ui) {
            ui.showSkeletonLoading('orders-list', 5);
            ui.showProgressBar(true);
        }
    }

    hideLoadingState() {
        const ui = this.app.modules.get('ui');
        if (ui) {
            ui.hideSkeletonLoading('orders-list');
            ui.showProgressBar(false);
        }
    }

    showErrorState(message) {
        const container = document.getElementById('orders-list');
        if (container) {
            container.innerHTML = `
                <div class="error-state">
                    <h3>⚠️ Ошибка загрузки</h3>
                    <p>${message}</p>
                    <button class="btn btn-primary" onclick="this.loadOrders()">Попробовать снова</button>
                </div>
            `;
        }
    }

    /**
     * Initialize for specific user role
     */
    initializeForRole(role) {
        console.log(`🎯 Initializing Operator Panel for role: ${role}`);
        
        // Store role for customization
        this.userRole = role;
        
        // Initialize with default orders view
        this.initialize();
        
        // Customize UI based on role
        this.customizeForRole(role);
    }

    /**
     * Customize panel based on user role
     */
    customizeForRole(role) {
        console.log(`🎨 Customizing Operator Panel for: ${role}`);
        
        // Role-specific customizations can be added here
        switch (role) {
            case 'user':
                // Users see limited functionality
                this.hideOperatorOnlyFeatures();
                break;
            case 'operator':
            case 'main_operator':
            case 'owner':
                // Operators see full functionality
                this.showOperatorFeatures();
                break;
            case 'driver':
                // Drivers see execution-focused features
                this.customizeForDriver();
                break;
        }
    }

    /**
     * Hide features for users
     */
    hideOperatorOnlyFeatures() {
        // Implementation for user-specific view
        console.log('🔒 Hiding operator-only features');
    }

    /**
     * Show operator features
     */
    showOperatorFeatures() {
        // Implementation for operator view
        console.log('🔓 Showing operator features');
    }

    /**
     * Customize for driver role
     */
    customizeForDriver() {
        // Implementation for driver-specific view
        console.log('🚛 Customizing for driver');
    }

    /**
     * Cleanup when module is destroyed
     */
    destroy() {
        this.stopRealTimeUpdates();
        this.orders.clear();
        this.filteredOrders = [];
        
        // Remove event listeners
        document.removeEventListener('visibilitychange', this.handleVisibilityChange);
        
        console.log('🗑️ Operator Panel destroyed');
    }
}

// Делаем OperatorPanelModule доступным глобально
window.OperatorPanelModule = OperatorPanelModule; 