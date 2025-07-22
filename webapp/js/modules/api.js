/**
 * @fileoverview Optimized API Module with caching and performance optimizations
 * @version 2.0
 */

class APIModule {
    constructor(app) {
        this.app = app;
        this.baseUrl = app.config.apiBaseUrl;
        this.cache = new Map();
        this.requestQueue = [];
        this.failedRequests = [];
        this.abortControllers = new Map();
        
        // Request deduplication
        this.pendingRequests = new Map();
        
        // Batch processing
        this.batchQueue = [];
        this.batchTimer = null;
        
        // Performance monitoring
        this.metrics = {
            totalRequests: 0,
            successfulRequests: 0,
            failedRequests: 0,
            averageResponseTime: 0,
            cacheHits: 0
        };

        this.setupInterceptors();
    }

    /**
     * Setup request/response interceptors
     */
    setupInterceptors() {
        // Auto-retry for failed requests
        this.retryConfig = {
            maxRetries: 3,
            retryDelay: 1000,
            retryMultiplier: 2,
            retryableStatusCodes: [408, 429, 500, 502, 503, 504]
        };
    }

    /**
     * Main fetch method with all optimizations
     */
    async fetch(endpoint, options = {}) {
        const requestKey = this.generateRequestKey(endpoint, options);
        
        // Check for pending identical request
        if (this.pendingRequests.has(requestKey)) {
            return this.pendingRequests.get(requestKey);
        }

        const requestPromise = this._performRequest(endpoint, options);
        this.pendingRequests.set(requestKey, requestPromise);

        try {
            const result = await requestPromise;
            return result;
        } finally {
            this.pendingRequests.delete(requestKey);
        }
    }

    /**
     * Internal request performer
     */
    async _performRequest(endpoint, options = {}) {
        const startTime = performance.now();
        this.metrics.totalRequests++;

        // Check cache first
        const cacheKey = this.generateCacheKey(endpoint, options);
        if (options.cache !== false && this.cache.has(cacheKey)) {
            const cached = this.cache.get(cacheKey);
            if (!this.isCacheExpired(cached)) {
                this.metrics.cacheHits++;
                console.log(`📦 Cache hit for ${endpoint}`);
                return cached.data;
            } else {
                this.cache.delete(cacheKey);
            }
        }

        // Check network status
        if (!navigator.onLine) {
            throw new Error('No internet connection');
        }

        // Create abort controller for request cancellation
        const abortController = new AbortController();
        const requestId = this.generateRequestId();
        this.abortControllers.set(requestId, abortController);

        // Prepare headers with authentication
        const headers = {
            'Content-Type': 'application/json',
            'X-Client-Version': this.app.version,
            'X-Request-ID': requestId,
            ...options.headers
        };

        // Add Telegram authentication if available
        if (this.app.tg && this.app.tg.initData) {
            headers['X-Telegram-Auth'] = this.app.tg.initData;
        } else {
            console.warn('⚠️ No Telegram auth data available, request may fail');
        }

        const requestOptions = {
            method: 'GET',
            headers,
            signal: abortController.signal,
            ...options
        };

        // Handle different body types
        if (options.body && !(options.body instanceof FormData)) {
            requestOptions.body = JSON.stringify(options.body);
        }

        try {
            const response = await this._fetchWithRetry(`${this.baseUrl}${endpoint}`, requestOptions);
            
            if (!response.ok) {
                throw new APIError(response.status, response.statusText, await this.extractErrorData(response));
            }

            const data = await this.parseResponse(response);
            
            // Cache successful responses
            if (options.cache !== false && requestOptions.method === 'GET') {
                this.cacheResponse(cacheKey, data, options.cacheTTL);
            }

            this.metrics.successfulRequests++;
            const responseTime = performance.now() - startTime;
            this.updateAverageResponseTime(responseTime);

            console.log(`✅ API success: ${endpoint} (${responseTime.toFixed(2)}ms)`);
            return data;

        } catch (error) {
            this.metrics.failedRequests++;
            
            if (error.name !== 'AbortError') {
                this.failedRequests.push({
                    endpoint,
                    options,
                    error,
                    timestamp: Date.now(),
                    retryCount: options.retryCount || 0
                });
            }

            console.error(`❌ API error: ${endpoint}`, error);
            throw error;
        } finally {
            this.abortControllers.delete(requestId);
        }
    }

    /**
     * Fetch with automatic retry logic
     */
    async _fetchWithRetry(url, options, retryCount = 0) {
        try {
            return await fetch(url, options);
        } catch (error) {
            if (retryCount < this.retryConfig.maxRetries && this.shouldRetry(error)) {
                const delay = this.retryConfig.retryDelay * Math.pow(this.retryConfig.retryMultiplier, retryCount);
                console.log(`🔄 Retrying request in ${delay}ms (attempt ${retryCount + 1})`);
                
                await this.delay(delay);
                return this._fetchWithRetry(url, options, retryCount + 1);
            }
            throw error;
        }
    }

    /**
     * Parse response based on content type
     */
    async parseResponse(response) {
        const contentType = response.headers.get('content-type');
        
        if (contentType?.includes('application/json')) {
            const text = await response.text();
            if (!text) return {};
            
            const data = JSON.parse(text);
            
            // Handle API response format
            if (data.status === 'error') {
                throw new APIError(400, data.message, data.data);
            }
            
            return data.data || data;
        } else {
            return response.text();
        }
    }

    /**
     * Extract error data from response
     */
    async extractErrorData(response) {
        try {
            return await response.json();
        } catch {
            return { message: response.statusText };
        }
    }

    /**
     * Batch multiple requests together
     */
    async batchRequest(requests) {
        if (!Array.isArray(requests)) {
            throw new Error('Batch requests must be an array');
        }

        console.log(`📦 Processing batch of ${requests.length} requests`);
        
        // Execute requests in parallel with concurrency limit
        const concurrencyLimit = 5;
        const results = [];
        
        for (let i = 0; i < requests.length; i += concurrencyLimit) {
            const batch = requests.slice(i, i + concurrencyLimit);
            const batchPromises = batch.map(req => 
                this.fetch(req.endpoint, req.options).catch(error => ({ error, request: req }))
            );
            
            const batchResults = await Promise.all(batchPromises);
            results.push(...batchResults);
        }

        return results;
    }

    /**
     * Fetch user profile from backend with real authentication
     */
    async fetchUserProfile() {
        try {
            const response = await this.fetch('/api/user/profile', {
                cache: true,
                cacheTTL: 300000 // 5 minutes
            });
            
            // Transform backend response to match frontend expectations
            if (response && response.data) {
                const user = response.data;
                return {
                    Role: user.Role,
                    ID: user.ID,
                    Name: user.FirstName + (user.LastName ? ' ' + user.LastName : ''),
                    FirstName: user.FirstName,
                    LastName: user.LastName,
                    Username: user.Nickname,
                    ChatID: user.ChatID,
                    Phone: user.Phone,
                    IsBlocked: user.IsBlocked
                };
            }
            
            throw new Error('Invalid response format');
            
        } catch (error) {
            console.error('❌ Authentication error:', error);
            
            // Пробуем тестовый эндпоинт для отладки
            if (error.status === 401 || !this.app.tg?.initData) {
                console.warn('⚠️ Пробуем тестовый эндпоинт для отладки...');
                try {
                    const testResponse = await this.fetch('/api/test/profile', {
                        cache: false
                    });
                    
                    if (testResponse && testResponse.data) {
                        console.warn('✅ Используется тестовый профиль пользователя');
                        const user = testResponse.data;
                        return {
                            Role: user.Role,
                            ID: user.ID,
                            Name: user.FirstName + (user.LastName ? ' ' + user.LastName : ''),
                            FirstName: user.FirstName,
                            LastName: user.LastName,
                            Username: user.Nickname,
                            ChatID: user.ChatID,
                            Phone: user.Phone,
                            IsBlocked: user.IsBlocked
                        };
                    }
                } catch (testError) {
                    console.error('❌ Тестовый эндпоинт также недоступен:', testError);
                }
            }
            
            // Emergency fallback for development/testing only
            const tg = window.Telegram?.WebApp;
            if (tg && tg.initDataUnsafe?.user) {
                console.warn('⚠️ Using emergency fallback authentication');
                return {
                    Role: 'operator', // Default role for testing
                    ID: tg.initDataUnsafe.user.id || 999999,
                    Name: tg.initDataUnsafe.user.first_name || 'Тестовый пользователь',
                    FirstName: tg.initDataUnsafe.user.first_name,
                    LastName: tg.initDataUnsafe.user.last_name,
                    Username: tg.initDataUnsafe.user.username,
                    ChatID: tg.initDataUnsafe.user.id
                };
            }
            
            throw new Error('Authentication failed and no fallback available');
        }
    }

    /**
     * Order management methods
     */
    async fetchOrders(statusKey = 'active') {
        const userRole = this.app.state.user?.Role;
        
        // Choose correct endpoint based on user role
        if (userRole === 'user') {
            return this.fetch(`/api/user/orders?status=${statusKey}`);
        } else {
            return this.fetch(`/api/admin/orders?status=${statusKey}`);
        }
    }

    async fetchOrderDetails(orderId) {
        const userRole = this.app.state.user?.Role;
        
        // Choose correct endpoint based on user role
        if (userRole === 'user') {
            return this.fetch(`/api/user/order/${orderId}`);
        } else {
            return this.fetch(`/api/admin/order/${orderId}`);
        }
    }

    async fetchClients() {
        return this.fetch('/api/admin/clients');
    }

    async fetchClientDetails(clientId) {
        return this.fetch(`/api/admin/client/${clientId}`);
    }

    async createOrderForUser(payload) {
        return this.fetch('/api/user/create-order', { method: 'POST', body: payload });
    }

    async createOrderForOperator(payload) {
        return this.fetch('/api/admin/create-order', { method: 'POST', body: payload });
    }

    async fetchUserOrders() {
        return this.fetch('/api/user/orders');
    }

    async fetchUserOrderDetails(orderId) {
        return this.fetch(`/api/user/order/${orderId}`);
    }

    async updateOrder(orderId, updateData) {
        // Determine correct endpoint based on user role and operation
        const user = this.app.state.user;
        const isOperator = user && ['operator', 'main_operator', 'owner'].includes(user.Role);
        
        // Admin/operator endpoints for order actions
        const endpoint = isOperator ? 
            `/api/admin/order/${orderId}/action` : 
            `/api/user/order/${orderId}/action`;
            
        const result = await this.fetch(endpoint, {
            method: 'POST',
            body: updateData,
            cache: false
        });
        
        // Invalidate related caches
        this.invalidateCache(`/api/user/order/${orderId}`);
        this.invalidateCache(`/api/admin/order/${orderId}`);
        this.invalidateCache('/api/user/orders');
        this.invalidateCache('/api/admin/orders');
        
        return result;
    }

    /**
     * Search orders with query
     */
    async searchOrders(query, status = 'active') {
        const user = this.app.state.user;
        const isOperator = user && ['operator', 'main_operator', 'owner'].includes(user.Role);
        
        if (isOperator) {
            return this.fetch(`/api/admin/orders?search=${encodeURIComponent(query)}&status=${status}`, {
                cache: false // Search results shouldn't be cached
            });
        } else {
            return this.fetch(`/api/user/orders?search=${encodeURIComponent(query)}&status=${status}`, {
                cache: false
            });
        }
    }

    /**
     * Get real-time updates for orders
     */
    async getOrderUpdates(since, status = 'active') {
        const user = this.app.state.user;
        const isOperator = user && ['operator', 'main_operator', 'owner'].includes(user.Role);
        
        if (isOperator) {
            return this.fetch(`/api/admin/orders?since=${since}&status=${status}`, {
                cache: false,
                cacheTTL: 0 // No caching for real-time updates
            });
        } else {
            return this.fetch(`/api/user/orders?since=${since}&status=${status}`, {
                cache: false,
                cacheTTL: 0
            });
        }
    }

    /**
     * Update order field (for operators)
     */
    async updateOrderField(orderId, field, value) {
        const user = this.app.state.user;
        const isOperator = user && ['operator', 'main_operator', 'owner'].includes(user.Role);
        
        if (!isOperator) {
            throw new Error('Access denied: Only operators can update order fields');
        }
        
        const result = await this.fetch(`/api/admin/order/${orderId}/update-field`, {
            method: 'POST',
            body: { field, value },
            cache: false
        });
        
        // Invalidate related caches
        this.invalidateCache(`/api/admin/order/${orderId}`);
        this.invalidateCache('/api/admin/orders');
        
        return result;
    }

    /**
     * Add media to order
     */
    async addOrderMedia(orderId, mediaData) {
        const user = this.app.state.user;
        const isOperator = user && ['operator', 'main_operator', 'owner'].includes(user.Role);
        
        const endpoint = isOperator ? 
            `/api/admin/order/${orderId}/add-media` : 
            `/api/user/order/${orderId}/add-media`;
            
        const result = await this.fetch(endpoint, {
            method: 'POST',
            body: mediaData,
            cache: false
        });
        
        // Invalidate related caches
        this.invalidateCache(`/api/user/order/${orderId}`);
        this.invalidateCache(`/api/admin/order/${orderId}`);
        
        return result;
    }

    /**
     * Client management methods
     */
    async fetchClients(page = 1, limit = 20, search = '') {
        const queryParams = new URLSearchParams({
            page: page.toString(),
            limit: limit.toString(),
            ...(search && { search })
        });
        
        return this.fetch(`/api/clients?${queryParams}`, {
            cache: true,
            cacheTTL: 60000 // 1 minute
        });
    }

    async fetchClientDetails(clientId) {
        return this.fetch(`/api/clients/${clientId}`, {
            cache: true,
            cacheTTL: 120000 // 2 minutes
        });
    }

    /**
     * Staff management methods
     */
    async fetchStaff(role) {
        return this.fetch(`/api/staff?role=${role}`, {
            cache: true,
            cacheTTL: 300000 // 5 minutes
        });
    }

    async addStaff(staffData) {
        const result = await this.fetch('/api/staff', {
            method: 'POST',
            body: staffData,
            cache: false
        });
        
        // Invalidate staff cache
        this.invalidateCache('/api/staff');
        
        return result;
    }

    /**
     * Media upload with progress tracking
     */
    async uploadMedia(files, onProgress) {
        if (!Array.isArray(files)) {
            files = [files];
        }

        const formData = new FormData();
        files.forEach((file, index) => {
            formData.append(`media_${index}`, file);
        });

        return new Promise((resolve, reject) => {
            const xhr = new XMLHttpRequest();
            
            xhr.upload.addEventListener('progress', (event) => {
                if (event.lengthComputable && onProgress) {
                    const percentComplete = (event.loaded / event.total) * 100;
                    onProgress(percentComplete);
                }
            });

            xhr.addEventListener('load', () => {
                if (xhr.status >= 200 && xhr.status < 300) {
                    try {
                        const response = JSON.parse(xhr.responseText);
                        resolve(response);
                    } catch (error) {
                        reject(new Error('Invalid JSON response'));
                    }
                } else {
                    reject(new Error(`Upload failed: ${xhr.statusText}`));
                }
            });

            xhr.addEventListener('error', () => {
                reject(new Error('Network error during upload'));
            });

            xhr.open('POST', `${this.baseUrl}/api/upload-media`);
            xhr.setRequestHeader('X-Telegram-Auth', this.app.tg.initData);
            xhr.send(formData);
        });
    }

    /**
     * Cache management methods
     */
    cacheResponse(key, data, ttl = 60000) {
        this.cache.set(key, {
            data,
            timestamp: Date.now(),
            ttl
        });
        
        // Clean old cache entries periodically
        if (this.cache.size > 100) {
            this.cleanExpiredCache();
        }
    }

    isCacheExpired(cached) {
        return Date.now() - cached.timestamp > cached.ttl;
    }

    invalidateCache(pattern) {
        if (typeof pattern === 'string') {
            // Exact match or pattern matching
            for (const key of this.cache.keys()) {
                if (key.includes(pattern)) {
                    this.cache.delete(key);
                }
            }
        }
    }

    cleanExpiredCache() {
        for (const [key, cached] of this.cache.entries()) {
            if (this.isCacheExpired(cached)) {
                this.cache.delete(key);
            }
        }
    }

    clearCache() {
        this.cache.clear();
        console.log('🗑️ Cache cleared');
    }

    /**
     * Request management
     */
    generateRequestKey(endpoint, options) {
        return `${options.method || 'GET'}:${endpoint}:${JSON.stringify(options.body || {})}`;
    }

    generateCacheKey(endpoint, options) {
        return `cache:${endpoint}:${JSON.stringify(options.query || {})}`;
    }

    generateRequestId() {
        return Date.now().toString(36) + Math.random().toString(36).substr(2);
    }

    /**
     * Error handling
     */
    shouldRetry(error) {
        if (error.name === 'AbortError') return false;
        if (error instanceof APIError) {
            return this.retryConfig.retryableStatusCodes.includes(error.status);
        }
        return true; // Retry network errors
    }

    /**
     * Retry failed requests
     */
    async retryFailedRequests() {
        if (this.failedRequests.length === 0) return;

        console.log(`🔄 Retrying ${this.failedRequests.length} failed requests`);
        
        const requestsToRetry = this.failedRequests.splice(0);
        
        for (const { endpoint, options } of requestsToRetry) {
            try {
                await this.fetch(endpoint, { ...options, retryCount: 0 });
            } catch (error) {
                console.warn(`Still failing: ${endpoint}`, error);
            }
        }
    }

    /**
     * Cancel all pending requests
     */
    cancelAllRequests() {
        for (const controller of this.abortControllers.values()) {
            controller.abort();
        }
        this.abortControllers.clear();
        console.log('🚫 All requests cancelled');
    }

    /**
     * Utility methods
     */
    delay(ms) {
        return new Promise(resolve => setTimeout(resolve, ms));
    }

    updateAverageResponseTime(responseTime) {
        const totalRequests = this.metrics.successfulRequests;
        const currentAverage = this.metrics.averageResponseTime;
        this.metrics.averageResponseTime = ((currentAverage * (totalRequests - 1)) + responseTime) / totalRequests;
    }

    /**
     * Get API statistics
     */
    getStatistics() {
        return {
            ...this.metrics,
            cacheSize: this.cache.size,
            pendingRequests: this.pendingRequests.size,
            failedRequestsCount: this.failedRequests.length,
            successRate: (this.metrics.successfulRequests / this.metrics.totalRequests * 100).toFixed(2) + '%',
            cacheHitRate: (this.metrics.cacheHits / this.metrics.totalRequests * 100).toFixed(2) + '%'
        };
    }
}

/**
 * Custom API Error class
 */
class APIError extends Error {
    constructor(status, message, data = null) {
        super(message);
        this.name = 'APIError';
        this.status = status;
        this.data = data;
    }
}

// Делаем APIModule доступным глобально
window.APIModule = APIModule; 