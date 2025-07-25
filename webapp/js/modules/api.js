/**
 * API Module
 * Handles all API communication with the backend
 */

class APIModule {
    constructor(app) {
        this.app = app;
        
        // Get API base URL from config with proper fallback
        const config = window.APP_CONFIG;
        if (config && config.API_BASE && config.API_PREFIX) {
            this.baseURL = config.API_BASE + config.API_PREFIX;
        } else {
            this.baseURL = window.APP_CONFIG?.API_BASE_URL || '/api';
        }
        
        if (this.baseURL.endsWith('/')) {
            this.baseURL = this.baseURL.slice(0, -1);
        }
        
        // Bind methods
        this.request = this.request.bind(this);
        this.handleResponse = this.handleResponse.bind(this);
        this.handleError = this.handleError.bind(this);
        
        console.log('✅ API Module initialized with baseURL:', this.baseURL);
    }

    /**
     * Make an API request
     */
    async request(endpoint, options = {}) {
        try {
            // Store endpoint for fallback data
            this.lastEndpoint = endpoint;

            // Clean up the endpoint
            const cleanEndpoint = endpoint.replace(/^\/+|\/+$/g, '');
            const url = `${this.baseURL}/${cleanEndpoint}`;
            
            // Get current initData or use fallback
            const initData = this.app.tg?.initData;
            let authHeader = 'fallback-development-mode';
            
            if (initData && initData !== 'fallback_init_data') {
                authHeader = initData;
            } else if (window.APP_CONFIG?.AUTH_FALLBACK_ENABLED) {
                authHeader = 'fallback-development-mode';
            } else {
                throw new Error('No Telegram auth data available');
            }
            
            // Default options
            const defaultOptions = {
                headers: {
                    'Content-Type': 'application/json',
                    'X-Telegram-Auth': authHeader
                },
                credentials: 'include'
            };
            
            // Merge options
            const fetchOptions = {
                ...defaultOptions,
                ...options,
                headers: {
                    ...defaultOptions.headers,
                    ...options.headers
                }
            };
            
            // Add body if present
            if (options.body) {
                if (options.body instanceof FormData) {
                    delete fetchOptions.headers['Content-Type'];
                } else {
                    fetchOptions.body = JSON.stringify(options.body);
                }
            }
            
            // Make request with timeout
            const controller = new AbortController();
            const timeout = setTimeout(() => {
                controller.abort();
            }, window.APP_CONFIG?.API_TIMEOUT || 15000);

            fetchOptions.signal = controller.signal;
            
            try {
                const response = await fetch(url, fetchOptions);
                clearTimeout(timeout);
                return await this.handleResponse(response);
            } catch (error) {
                clearTimeout(timeout);
                if (error.name === 'AbortError') {
                    throw new Error('Request timeout');
                }
                throw error;
            }
            
        } catch (error) {
            return this.handleError(error);
        }
    }

    /**
     * Handle API response
     */
    async handleResponse(response) {
        const contentType = response.headers.get('content-type');
        const isJson = contentType && contentType.includes('application/json');
        
        // Parse response
        let data;
        try {
            data = isJson ? await response.json() : await response.text();
        } catch (e) {
            console.error('Failed to parse response:', e);
            throw new Error('Failed to parse server response');
        }
        
        // Check for errors
        if (!response.ok) {
            // Create more descriptive error message
            let errorMessage = 'Unknown error';
            if (data.error) {
                errorMessage = data.error;
            } else if (data.message) {
                errorMessage = data.message;
            } else if (typeof data === 'string' && data.trim()) {
                errorMessage = `HTTP ${response.status}: ${response.statusText}`;
            } else {
                errorMessage = `HTTP ${response.status}: ${response.statusText}`;
            }
            
            const error = new Error(errorMessage);
            error.status = response.status;
            error.response = data;
            
            // Log detailed error in debug mode
            if (window.APP_CONFIG.DEBUG) {
                console.error('API Error Details:', {
                    status: response.status,
                    statusText: response.statusText,
                    data: data,
                    headers: Object.fromEntries([...response.headers])
                });
            }
            
            throw error;
        }
        
        return data;
    }

    /**
     * Handle API error
     */
    handleError(error) {
        // Log error
        if (window.APP_CONFIG.DEBUG) {
            console.error('❌ API Error:', error);
            console.error('Stack trace:', error.stack);
        }
        
        // Check if it's an authentication error
        if (error.status === 401 || error.status === 403) {
            console.error('Authentication failed:', error);
            
            // Use fallback if enabled
            if (window.APP_CONFIG.AUTH_FALLBACK_ENABLED) {
                console.warn('⚠️ Using emergency fallback authentication');
                return this.getFallbackData();
            }
            
            throw new Error('Authentication failed: ' + (error.message || 'Unknown error'));
        }
        
        // Use fallback for 404 errors when in development/test mode
        if (error.status === 404 && window.APP_CONFIG.AUTH_FALLBACK_ENABLED) {
            console.warn('⚠️ Using emergency fallback authentication (404 error)');
            return this.getFallbackData();
        }
        
        // Use fallback if enabled for other errors in dev mode
        if (window.APP_CONFIG.AUTH_FALLBACK_ENABLED && window.APP_CONFIG.DEBUG) {
            console.warn('⚠️ Using emergency fallback authentication');
            return this.getFallbackData();
        }
        
        throw error;
    }

    /**
     * Get fallback data for development
     */
    getFallbackData() {
        const testData = {
            orders: [
                {
                    ID: 1,
                    Status: 'new',
                    ContactName: 'Тестовый Клиент',
                    ContactPhone: '79781234567',
                    ServiceAddress: 'ул. Тестовая, д. 1',
                    FinalCost: 1500,
                    CreatedAt: new Date().toISOString()
                },
                {
                    ID: 2,
                    Status: 'in_progress',
                    ContactName: 'Другой Клиент',
                    ContactPhone: '79787654321',
                    ServiceAddress: 'ул. Ленина, д. 10',
                    FinalCost: 2500,
                    CreatedAt: new Date(Date.now() - 86400000).toISOString()
                }
            ],
            clients: [
                {
                    ID: 1,
                    FirstName: 'Иван',
                    LastName: 'Петров',
                    Username: 'ivan_petrov',
                    Phone: '79781234567',
                    Role: 'user',
                    IsBlocked: false
                },
                {
                    ID: 2,
                    FirstName: 'Мария',
                    LastName: 'Иванова',
                    Username: 'maria_ivanova',
                    Phone: '79787654321',
                    Role: 'user',
                    IsBlocked: false
                }
            ],
            profile: {
                ID: 1263060321,
                Username: 'Demontaj_Crimea',
                FirstName: 'Оператор',
                LastName: 'Сервис-Крым',
                Role: 'operator',
                Phone: '79781234567',
                IsActive: true
            },
            dashboard: {
                activeOrders: 5,
                completedOrders: 15,
                totalOrders: 20
            }
        };

        // Return appropriate test data based on the endpoint
        const endpoint = this.lastEndpoint?.toLowerCase() || '';
        
        if (endpoint.includes('orders')) {
            return {
                status: 'success',
                data: testData.orders
            };
        } else if (endpoint.includes('clients')) {
            return {
                status: 'success',
                data: testData.clients
            };
        } else if (endpoint.includes('profile') || endpoint.includes('user')) {
            return {
                status: 'success',
                data: testData.profile
            };
        } else if (endpoint.includes('dashboard') || endpoint.includes('stats')) {
            return {
                status: 'success',
                data: testData.dashboard
            };
        }

        // Default fallback
        return {
            status: 'success',
            data: []
        };
    }

    /**
     * Fetch user profile
     */
    async fetchUserProfile() {
        try {
            // Get user ID from Telegram WebApp
            const webAppUser = window.Telegram?.WebApp?.initDataUnsafe?.user;
            if (!webAppUser) {
                throw new Error('No user data in Telegram WebApp');
            }

            // Make API request
            const response = await this.request('user/profile', {
                headers: {
                    'X-Telegram-Auth': window.Telegram.WebApp.initData
                }
            });
            
            return response;
            
        } catch (error) {
            if (window.APP_CONFIG.AUTH_FALLBACK_ENABLED) {
                return this.getFallbackData().user;
            }
            throw error;
        }
    }

    /**
     * Fetch orders
     */
    async fetchOrders(filters = {}) {
        try {
            const queryParams = new URLSearchParams(filters).toString();
            
            // Check user role to determine correct endpoint
            const user = this.app?.state?.user;
            let endpoint;
            
            if (user && ['operator', 'admin', 'owner'].includes(user.Role)) {
                // For operators/admins, use admin endpoint
                endpoint = `admin/orders${queryParams ? '?' + queryParams : ''}`;
            } else {
                // For regular users, use user endpoint
                endpoint = `user/orders${queryParams ? '?' + queryParams : ''}`;
            }
            
            return this.request(endpoint);
        } catch (error) {
            console.error('Failed to fetch orders:', error);
            throw error;
        }
    }

    /**
     * Fetch clients
     */
    async fetchClients(filters = {}) {
        try {
            const queryParams = new URLSearchParams(filters).toString();
            
            // Clients endpoint is only for operators/admins
            const user = this.app?.state?.user;
            if (!user || !['operator', 'admin', 'owner'].includes(user.Role)) {
                throw new Error('Access denied: Clients data is only available for operators and admins');
            }
            
            const endpoint = `admin/clients${queryParams ? '?' + queryParams : ''}`;
            return this.request(endpoint);
        } catch (error) {
            console.error('Failed to fetch clients:', error);
            throw error;
        }
    }

    /**
     * Fetch analytics
     */
    async fetchAnalytics(period = 'month') {
        return this.request(`analytics?period=${period}`);
    }

    /**
     * Fetch statistics (alias for analytics with default period)
     */
    async fetchStats() {
        try {
            const user = this.app?.state?.user;
            if (!user || !['operator', 'admin', 'owner'].includes(user.Role)) {
                throw new Error('Access denied: Statistics data is only available for operators and admins');
            }
            
            return await this.fetchAnalytics('month');
        } catch (error) {
            console.error('Failed to fetch stats:', error);
            throw error;
        }
    }

    /**
     * Create order
     */
    async createOrder(orderData) {
        return this.request('orders', {
            method: 'POST',
            body: orderData
        });
    }

    /**
     * Update order
     */
    async updateOrder(orderId, orderData) {
        return this.request(`orders/${orderId}`, {
            method: 'PUT',
            body: orderData
        });
    }

    /**
     * Delete order
     */
    async deleteOrder(orderId) {
        return this.request(`orders/${orderId}`, {
            method: 'DELETE'
        });
    }

    /**
     * Upload file
     */
    async uploadFile(file, type = 'image') {
        const formData = new FormData();
        formData.append('file', file);
        formData.append('type', type);
        
        return this.request('upload-media', {
            method: 'POST',
            body: formData,
            headers: {
                // Remove Content-Type to let browser set it with boundary
                'Content-Type': undefined
            }
        });
    }
}

// Export module
window.APIModule = APIModule; 