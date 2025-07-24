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
        return `
            <div class="orders">
                <div class="filters">
                    <div class="search-input-wrapper">
                        <i class="fas fa-search"></i>
                        <input type="text" id="orders-search" class="search-input" placeholder="Поиск заказов...">
                    </div>
                    <select id="order-status-filter" class="filter-select">
                        <option value="">Все статусы</option>
                        <option value="new">Новые</option>
                        <option value="in_progress">В работе</option>
                        <option value="completed">Выполненные</option>
                        <option value="canceled">Отменённые</option>
                    </select>
                </div>
                <div id="orders-list" class="orders-list"></div>
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
     * Initialize orders
     */
    async initializeOrders(panel) {
        try {
            // Setup search
            const searchInput = panel.querySelector('#orders-search');
            if (searchInput) {
                searchInput.addEventListener('input', this.debounce(() => {
                    this.loadOrders();
                }, 300));
            }
            
            // Setup status filter
            const statusFilter = panel.querySelector('#order-status-filter');
            if (statusFilter) {
                statusFilter.addEventListener('change', () => {
                    this.loadOrders();
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
                searchInput.addEventListener('input', this.debounce(() => {
                    this.loadClients();
                }, 300));
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
                ordersList.innerHTML = response.data.length ? 
                    response.data.map(order => this.renderOrderCard(order)).join('') :
                    '<div class="empty-state">Заказов не найдено</div>';
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
     * Render order card
     */
    renderOrderCard(order) {
        return `
            <div class="card order-card" data-order-id="${order.ID}">
                <div class="card-header">
                    <div class="card-title">Заказ #${order.ID}</div>
                    <div class="status-badge ${order.Status}">${this.getOrderStatusText(order.Status)}</div>
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
}

// Export module
window.UIModule = UIModule; 