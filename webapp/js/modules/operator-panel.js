/**
 * Operator Panel Module
 * Handles operator-specific functionality
 */

class OperatorPanelModule {
    constructor(app) {
        this.app = app;
        this.currentView = 'list';
        this.currentOrder = null;
        
        // Bind methods
        this.initialize = this.initialize.bind(this);
        this.showOrderDetails = this.showOrderDetails.bind(this);
        this.updateOrderStatus = this.updateOrderStatus.bind(this);
    }

    /**
     * Initialize module
     */
    async initialize() {
        try {
            // Load initial data
            await this.loadOrders();
            
            // Setup event listeners
            this.setupEventListeners();
            
            console.log('✅ Operator panel module initialized');
            
        } catch (error) {
            console.error('Failed to initialize operator panel:', error);
            throw error;
        }
    }

    /**
     * Setup event listeners
     */
    setupEventListeners() {
        // Status filter
        const statusFilter = document.getElementById('order-status-filter');
        if (statusFilter) {
            statusFilter.addEventListener('change', (e) => {
                this.loadOrders(e.target.value);
            });
        }
        
        // Search
        const searchInput = document.getElementById('orders-search');
        if (searchInput) {
            searchInput.addEventListener('input', this.debounce((e) => {
                this.loadOrders(undefined, e.target.value);
            }, 300));
        }
        
        // Refresh button
        const refreshButton = document.getElementById('refresh-orders');
        if (refreshButton) {
            refreshButton.addEventListener('click', () => {
                this.loadOrders();
            });
        }
    }

    /**
     * Load orders
     */
    async loadOrders(status = 'active', search = '') {
        try {
            // Use the correct API endpoint based on user role
            const user = this.app.state?.user;
            let endpoint = 'user/orders';
            
            if (user && ['operator', 'admin', 'owner'].includes(user.Role)) {
                endpoint = 'admin/orders';
            }
            
            const params = new URLSearchParams();
            if (status) params.append('status', status);
            if (search) params.append('search', search);
            
            const queryString = params.toString();
            const fullEndpoint = queryString ? `${endpoint}?${queryString}` : endpoint;
            
            const response = await this.app.modules.get('api').request(fullEndpoint);
            
            this.renderOrders(response.data || []);
            
        } catch (error) {
            console.error('Failed to load orders:', error);
            // Use the utils module for notifications if available
            if (this.app.modules.get('utils')) {
                this.app.modules.get('utils').showNotification('Не удалось загрузить заказы', 'error');
            }
        }
    }

    /**
     * Render orders
     */
    renderOrders(orders) {
        const container = document.getElementById('orders-list');
        if (!container) return;
        
        if (!orders.length) {
            container.innerHTML = '<div class="empty-state">Заказов не найдено</div>';
            return;
        }
        
        container.innerHTML = orders.map(order => this.renderOrderCard(order)).join('');
        
        // Add click handlers
        container.querySelectorAll('.order-card').forEach(card => {
            card.addEventListener('click', () => {
                this.showOrderDetails(card.dataset.orderId);
            });
        });
    }

    /**
     * Render order card
     */
    renderOrderCard(order) {
        return `
            <div class="card order-card" data-order-id="${order.ID}">
                <div class="card-header">
                    <div class="card-title">Заказ #${order.ID}</div>
                    <div class="status-badge ${order.Status}">${this.getStatusText(order.Status)}</div>
                </div>
                <div class="card-body">
                    <div class="order-info">
                        <div><strong>Клиент:</strong> ${order.ContactName || 'Не указан'}</div>
                        <div><strong>Телефон:</strong> ${this.formatPhone(order.ContactPhone)}</div>
                        <div><strong>Адрес:</strong> ${order.ServiceAddress || 'Не указан'}</div>
                        <div><strong>Стоимость:</strong> ${this.formatCurrency(order.FinalCost)}</div>
                        <div><strong>Создан:</strong> ${this.formatDate(order.CreatedAt)}</div>
                    </div>
                </div>
            </div>
        `;
    }

    /**
     * Show order details
     */
    async showOrderDetails(orderId) {
        try {
            const response = await this.app.modules.get('api').request(`orders/${orderId}`);
            const order = response.data;
            
            if (!order) {
                throw new Error('Order not found');
            }
            
            this.currentOrder = order;
            this.currentView = 'details';
            
            const container = document.getElementById('orders-list');
            if (!container) return;
            
            container.innerHTML = this.renderOrderDetails(order);
            
            // Add event listeners
            this.setupOrderDetailsEvents(order);
            
        } catch (error) {
            console.error('Failed to load order details:', error);
            this.app.modules.get('ui').showError('Failed to load order details');
        }
    }

    /**
     * Render order details
     */
    renderOrderDetails(order) {
        return `
            <div class="order-details">
                <div class="details-header">
                    <button class="btn btn-icon back-button">
                        <i class="fas fa-arrow-left"></i>
                    </button>
                    <h2>Заказ #${order.ID}</h2>
                </div>
                
                <div class="details-content">
                    <div class="details-section">
                        <h3>Информация о заказе</h3>
                        <div class="info-grid">
                            <div class="info-item">
                                <label>Статус</label>
                                <div class="status-badge ${order.Status}">
                                    ${this.getStatusText(order.Status)}
                                </div>
                            </div>
                            <div class="info-item">
                                <label>Создан</label>
                                <div>${this.formatDate(order.CreatedAt)}</div>
                            </div>
                            <div class="info-item">
                                <label>Стоимость</label>
                                <div>${this.formatCurrency(order.FinalCost)}</div>
                            </div>
                        </div>
                    </div>
                    
                    <div class="details-section">
                        <h3>Контактная информация</h3>
                        <div class="info-grid">
                            <div class="info-item">
                                <label>Имя</label>
                                <div>${order.ContactName || 'Не указано'}</div>
                            </div>
                            <div class="info-item">
                                <label>Телефон</label>
                                <div>${this.formatPhone(order.ContactPhone)}</div>
                            </div>
                            <div class="info-item">
                                <label>Адрес</label>
                                <div>${order.ServiceAddress || 'Не указан'}</div>
                            </div>
                        </div>
                    </div>
                    
                    <div class="details-section">
                        <h3>Управление заказом</h3>
                        <div class="status-actions">
                            ${this.renderStatusActions(order)}
                        </div>
                    </div>
                </div>
            </div>
        `;
    }

    /**
     * Render status actions
     */
    renderStatusActions(order) {
        const actions = {
            'new': [
                { status: 'in_progress', label: 'Взять в работу', class: 'primary' },
                { status: 'canceled', label: 'Отменить', class: 'danger' }
            ],
            'in_progress': [
                { status: 'completed', label: 'Завершить', class: 'success' },
                { status: 'canceled', label: 'Отменить', class: 'danger' }
            ]
        };
        
        const availableActions = actions[order.Status] || [];
        
        return availableActions.map(action => `
            <button class="btn btn-${action.class}" data-status="${action.status}">
                ${action.label}
            </button>
        `).join('');
    }

    /**
     * Setup order details events
     */
    setupOrderDetailsEvents(order) {
        // Back button
        document.querySelector('.back-button')?.addEventListener('click', () => {
            this.currentView = 'list';
            this.loadOrders();
        });
        
        // Status actions
        document.querySelectorAll('.status-actions button').forEach(button => {
            button.addEventListener('click', () => {
                this.updateOrderStatus(order.ID, button.dataset.status);
            });
        });
    }

    /**
     * Update order status
     */
    async updateOrderStatus(orderId, newStatus) {
        try {
            await this.app.modules.get('api').updateOrder(orderId, {
                status: newStatus
            });
            
            this.app.modules.get('ui').showNotification('Статус заказа обновлен', 'success');
            
            // Refresh order details
            await this.showOrderDetails(orderId);
            
        } catch (error) {
            console.error('Failed to update order status:', error);
            this.app.modules.get('ui').showError('Failed to update order status');
        }
    }

    /**
     * Format phone number
     */
    formatPhone(phone) {
        if (!phone) return 'Не указан';
        return phone.replace(/(\d{1})(\d{3})(\d{3})(\d{2})(\d{2})/, '+$1 ($2) $3-$4-$5');
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
     * Get status text
     */
    getStatusText(status) {
        const statuses = {
            'new': 'Новый',
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
}

// Export module
window.OperatorPanelModule = OperatorPanelModule; 