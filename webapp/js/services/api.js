/**
 * API Service Module
 * Handles all API requests with proper error handling and response formatting
 */

class APIService {
    constructor() {
        // Ensure config is available
        if (!window.APP_CONFIG) {
            console.error('❌ APP_CONFIG not found! APIService requires configuration.');
            return;
        }

        this.baseURL = window.APP_CONFIG.API_BASE;
        this.apiPrefix = window.APP_CONFIG.API_PREFIX;
        this.timeout = window.APP_CONFIG.API_TIMEOUT;
        
        // Bind methods
        this.apiRequest = this.apiRequest.bind(this);
        this.handleResponse = this.handleResponse.bind(this);
        this.handleError = this.handleError.bind(this);

        console.log('✅ APIService initialized');
    }

    /**
     * Make an API request
     * @param {string} endpoint - API endpoint
     * @param {Object} options - Request options
     * @returns {Promise} - API response
     */
    async apiRequest(endpoint, options = {}) {
        const defaultOptions = {
            headers: {
                'Content-Type': 'application/json',
                [window.APP_CONFIG.AUTH_HEADER]: this.getAuthToken()
            },
            timeout: this.timeout
        };

        // Remove leading slash if present
        const cleanEndpoint = endpoint.startsWith('/') ? endpoint.substring(1) : endpoint;
        
        // Construct URL with API prefix
        const url = `${this.baseURL}${this.apiPrefix}/${cleanEndpoint}`;

        if (window.APP_CONFIG.DEBUG) {
            console.log(`🌐 API Request: ${url}`);
        }

        const fetchOptions = {
            ...defaultOptions,
            ...options,
            headers: {
                ...defaultOptions.headers,
                ...options.headers
            }
        };

        try {
            const controller = new AbortController();
            const timeoutId = setTimeout(() => controller.abort(), this.timeout);

            const response = await fetch(url, {
                ...fetchOptions,
                signal: controller.signal
            });

            clearTimeout(timeoutId);
            return await this.handleResponse(response);
        } catch (error) {
            return this.handleError(error);
        }
    }

    /**
     * Handle API response
     * @param {Response} response - Fetch response object
     * @returns {Promise} - Parsed response data
     */
    async handleResponse(response) {
        let data;
        let text = '';
        
        try {
            text = await response.text();
            data = text ? JSON.parse(text) : null;
        } catch (error) {
            if (window.APP_CONFIG.DEBUG) {
                console.error('Failed to parse response:', text);
            }
            throw {
                status: response.status,
                message: `Failed to parse JSON response: ${error.message}`,
                original: error,
                responseText: text
            };
        }

        if (!response.ok) {
            throw {
                status: response.status,
                message: data?.message || response.statusText || 'API Error',
                data: data
            };
        }

        return data;
    }

    /**
     * Handle API error
     * @param {Error} error - Error object
     * @throws {Error} - Enhanced error object
     */
    handleError(error) {
        if (error.name === 'AbortError') {
            throw {
                status: 408,
                message: 'Request timeout',
                original: error
            };
        }

        // If error is already formatted, just rethrow it
        if (error.status && error.message) {
            throw error;
        }

        throw {
            status: 500,
            message: error.message || 'Unknown error',
            original: error
        };
    }

    /**
     * Get auth token from storage
     * @returns {string|null} - Auth token
     */
    getAuthToken() {
        if (window.Telegram?.WebApp?.initData) {
            return window.Telegram.WebApp.initData;
        }
        return localStorage.getItem('auth_token');
    }

    // API Endpoints

    /**
     * User endpoints
     */
    async getUserProfile() {
        return this.apiRequest('/user/profile');
    }

    async updateUserProfile(data) {
        return this.apiRequest('/user/profile', {
            method: 'PUT',
            body: JSON.stringify(data)
        });
    }

    /**
     * Order endpoints
     */
    async getOrders(filters = {}) {
        const params = new URLSearchParams(filters);
        return this.apiRequest(`/orders?${params}`);
    }

    async createOrder(data) {
        return this.apiRequest('/orders', {
            method: 'POST',
            body: JSON.stringify(data)
        });
    }

    async updateOrder(orderId, data) {
        return this.apiRequest(`/orders/${orderId}`, {
            method: 'PUT',
            body: JSON.stringify(data)
        });
    }

    async deleteOrder(orderId) {
        return this.apiRequest(`/orders/${orderId}`, {
            method: 'DELETE'
        });
    }

    /**
     * Client endpoints
     */
    async getClients(filters = {}) {
        const params = new URLSearchParams(filters);
        return this.apiRequest(`/clients?${params}`);
    }

    async createClient(data) {
        return this.apiRequest('/clients', {
            method: 'POST',
            body: JSON.stringify(data)
        });
    }

    /**
     * Media endpoints
     */
    async uploadMedia(file, type = 'image') {
        const formData = new FormData();
        formData.append('file', file);
        formData.append('type', type);

        return this.apiRequest('/upload-media', {
            method: 'POST',
            headers: {
                // Remove Content-Type to let browser set it with boundary
                'Content-Type': undefined
            },
            body: formData
        });
    }

    async deleteMedia(mediaId) {
        return this.apiRequest(`/media/${mediaId}`, {
            method: 'DELETE'
        });
    }

    /**
     * Analytics endpoints
     */
    async getAnalytics(params = {}) {
        const queryParams = new URLSearchParams(params);
        return this.apiRequest(`/analytics?${queryParams}`);
    }
}

// Create and export a single instance
if (!window.APIService) {
    window.APIService = new APIService();
}

// For backwards compatibility and direct script usage
if (typeof module !== 'undefined' && module.exports) {
    module.exports = window.APIService;
} 
window.APIService = new APIService(); 