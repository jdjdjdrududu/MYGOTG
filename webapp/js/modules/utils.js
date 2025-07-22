/**
 * @fileoverview Utility Module with performance-optimized functions
 * @version 2.0
 */

class UtilsModule {
    constructor(app) {
        this.app = app;
        
        // Performance utilities
        this.debounceTimers = new Map();
        this.throttleTimers = new Map();
        this.memoizeCache = new Map();
        
        // DOM utilities
        this.observerPool = new Map();
        
        // Date/Time utilities
        this.dateFormatters = new Map();
        this.setupDateFormatters();
    }

    /**
     * Setup commonly used date formatters for better performance
     */
    setupDateFormatters() {
        this.dateFormatters.set('short', new Intl.DateTimeFormat('ru-RU', {
            day: '2-digit',
            month: '2-digit',
            year: '2-digit'
        }));
        
        this.dateFormatters.set('long', new Intl.DateTimeFormat('ru-RU', {
            day: '2-digit',
            month: 'long',
            year: 'numeric'
        }));
        
        this.dateFormatters.set('time', new Intl.DateTimeFormat('ru-RU', {
            hour: '2-digit',
            minute: '2-digit'
        }));
        
        this.dateFormatters.set('datetime', new Intl.DateTimeFormat('ru-RU', {
            day: '2-digit',
            month: '2-digit',
            hour: '2-digit',
            minute: '2-digit'
        }));
    }

    /**
     * Performance Utilities
     */
    
    /**
     * Debounce function execution
     */
    debounce(func, delay, key = 'default') {
        return (...args) => {
            clearTimeout(this.debounceTimers.get(key));
            this.debounceTimers.set(key, setTimeout(() => func.apply(this, args), delay));
        };
    }

    /**
     * Throttle function execution
     */
    throttle(func, limit, key = 'default') {
        return (...args) => {
            if (!this.throttleTimers.get(key)) {
                func.apply(this, args);
                this.throttleTimers.set(key, setTimeout(() => {
                    this.throttleTimers.delete(key);
                }, limit));
            }
        };
    }

    /**
     * Memoize function results
     */
    memoize(func, keyGenerator) {
        return (...args) => {
            const key = keyGenerator ? keyGenerator(...args) : JSON.stringify(args);
            
            if (this.memoizeCache.has(key)) {
                return this.memoizeCache.get(key);
            }
            
            const result = func.apply(this, args);
            this.memoizeCache.set(key, result);
            
            // Clean cache if it gets too large
            if (this.memoizeCache.size > 100) {
                const firstKey = this.memoizeCache.keys().next().value;
                this.memoizeCache.delete(firstKey);
            }
            
            return result;
        };
    }

    /**
     * Batch DOM operations for better performance
     */
    batchDOMOperations(operations) {
        return new Promise((resolve) => {
            requestAnimationFrame(() => {
                const results = operations.map(op => op());
                resolve(results);
            });
        });
    }

    /**
     * Virtual DOM diffing for efficient updates
     */
    updateElementContent(element, newContent) {
        if (element.innerHTML === newContent) {
            return false; // No update needed
        }
        
        element.innerHTML = newContent;
        return true;
    }

    /**
     * DOM Utilities
     */
    
    /**
     * Safely query DOM element with error handling
     */
    safeQuerySelector(selector, context = document) {
        try {
            return context.querySelector(selector);
        } catch (error) {
            console.warn(`Invalid selector: ${selector}`, error);
            return null;
        }
    }

    /**
     * Create element with attributes and content
     */
    createElement(tag, attributes = {}, content = '') {
        const element = document.createElement(tag);
        
        Object.entries(attributes).forEach(([key, value]) => {
            if (key === 'className') {
                element.className = value;
            } else if (key === 'style' && typeof value === 'object') {
                Object.assign(element.style, value);
            } else if (key.startsWith('data-')) {
                element.setAttribute(key, value);
            } else {
                element[key] = value;
            }
        });
        
        if (content) {
            if (typeof content === 'string') {
                element.innerHTML = content;
            } else if (content instanceof Node) {
                element.appendChild(content);
            }
        }
        
        return element;
    }

    /**
     * Add event listener with automatic cleanup
     */
    addEventListenerWithCleanup(element, event, handler, options) {
        element.addEventListener(event, handler, options);
        
        return () => {
            element.removeEventListener(event, handler, options);
        };
    }

    /**
     * Animate element with CSS transitions
     */
    animateElement(element, properties, duration = 300) {
        return new Promise((resolve) => {
            const originalTransition = element.style.transition;
            element.style.transition = `all ${duration}ms ease-out`;
            
            Object.assign(element.style, properties);
            
            const handleTransitionEnd = () => {
                element.style.transition = originalTransition;
                element.removeEventListener('transitionend', handleTransitionEnd);
                resolve();
            };
            
            element.addEventListener('transitionend', handleTransitionEnd);
            
            // Fallback timeout
            setTimeout(resolve, duration + 50);
        });
    }

    /**
     * Data Processing Utilities
     */
    
    /**
     * Deep clone object with performance optimizations
     */
    deepClone(obj) {
        if (obj === null || typeof obj !== 'object') {
            return obj;
        }
        
        if (obj instanceof Date) {
            return new Date(obj.getTime());
        }
        
        if (obj instanceof Array) {
            return obj.map(item => this.deepClone(item));
        }
        
        if (typeof obj === 'object') {
            const cloned = {};
            Object.keys(obj).forEach(key => {
                cloned[key] = this.deepClone(obj[key]);
            });
            return cloned;
        }
        
        return obj;
    }

    /**
     * Merge objects deeply
     */
    deepMerge(target, ...sources) {
        if (!sources.length) return target;
        const source = sources.shift();

        if (this.isObject(target) && this.isObject(source)) {
            for (const key in source) {
                if (this.isObject(source[key])) {
                    if (!target[key]) Object.assign(target, { [key]: {} });
                    this.deepMerge(target[key], source[key]);
                } else {
                    Object.assign(target, { [key]: source[key] });
                }
            }
        }

        return this.deepMerge(target, ...sources);
    }

    /**
     * Check if value is object
     */
    isObject(item) {
        return item && typeof item === 'object' && !Array.isArray(item);
    }

    /**
     * Format file size to human readable
     */
    formatFileSize(bytes) {
        const sizes = ['Bytes', 'KB', 'MB', 'GB'];
        if (bytes === 0) return '0 Bytes';
        const i = Math.floor(Math.log(bytes) / Math.log(1024));
        return Math.round(bytes / Math.pow(1024, i) * 100) / 100 + ' ' + sizes[i];
    }

    /**
     * String Utilities
     */
    
    /**
     * Truncate text with ellipsis
     */
    truncateText(text, maxLength, suffix = '...') {
        if (text.length <= maxLength) return text;
        return text.substr(0, maxLength - suffix.length) + suffix;
    }

    /**
     * Escape HTML to prevent XSS
     */
    escapeHTML(text) {
        const div = document.createElement('div');
        div.textContent = text;
        return div.innerHTML;
    }

    /**
     * Remove HTML tags from string
     */
    stripHTML(html) {
        const div = document.createElement('div');
        div.innerHTML = html;
        return div.textContent || div.innerText || '';
    }

    /**
     * Generate random string
     */
    generateRandomString(length = 10, charset = 'ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789') {
        let result = '';
        for (let i = 0; i < length; i++) {
            result += charset.charAt(Math.floor(Math.random() * charset.length));
        }
        return result;
    }

    /**
     * Date & Time Utilities
     */
    
    /**
     * Format date using pre-configured formatters
     */
    formatDate(date, format = 'short') {
        if (!(date instanceof Date)) {
            date = new Date(date);
        }
        
        const formatter = this.dateFormatters.get(format);
        if (formatter) {
            return formatter.format(date);
        }
        
        // Fallback to default formatting
        return date.toLocaleDateString('ru-RU');
    }

    /**
     * Get relative time (e.g., "2 hours ago")
     */
    getRelativeTime(date) {
        if (!(date instanceof Date)) {
            date = new Date(date);
        }
        
        const now = new Date();
        const diffMs = now.getTime() - date.getTime();
        const diffMins = Math.floor(diffMs / 60000);
        const diffHours = Math.floor(diffMins / 60);
        const diffDays = Math.floor(diffHours / 24);
        
        if (diffMins < 1) return 'только что';
        if (diffMins < 60) return `${diffMins} мин. назад`;
        if (diffHours < 24) return `${diffHours} ч. назад`;
        if (diffDays < 7) return `${diffDays} дн. назад`;
        
        return this.formatDate(date, 'short');
    }

    /**
     * Check if date is today
     */
    isToday(date) {
        if (!(date instanceof Date)) {
            date = new Date(date);
        }
        
        const today = new Date();
        return date.toDateString() === today.toDateString();
    }

    /**
     * Validation Utilities
     */
    
    /**
     * Validate email address
     */
    isValidEmail(email) {
        const emailRegex = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;
        return emailRegex.test(email);
    }

    /**
     * Validate phone number (Russian format)
     */
    isValidPhone(phone) {
        const phoneRegex = /^(\+7|7|8)?[\s\-]?\(?[489][0-9]{2}\)?[\s\-]?[0-9]{3}[\s\-]?[0-9]{2}[\s\-]?[0-9]{2}$/;
        return phoneRegex.test(phone.replace(/\s/g, ''));
    }

    /**
     * Format phone number to standard format
     */
    formatPhone(phone) {
        const cleaned = phone.replace(/\D/g, '');
        
        if (cleaned.length === 11 && (cleaned.startsWith('7') || cleaned.startsWith('8'))) {
            const normalized = '7' + cleaned.slice(1);
            return `+7 (${normalized.slice(1, 4)}) ${normalized.slice(4, 7)}-${normalized.slice(7, 9)}-${normalized.slice(9, 11)}`;
        }
        
        return phone;
    }

    /**
     * URL Utilities
     */
    
    /**
     * Parse URL parameters
     */
    parseURLParams(url = window.location.href) {
        const urlObj = new URL(url);
        const params = {};
        
        urlObj.searchParams.forEach((value, key) => {
            params[key] = value;
        });
        
        return params;
    }

    /**
     * Build URL with parameters
     */
    buildURL(baseUrl, params = {}) {
        const url = new URL(baseUrl);
        
        Object.entries(params).forEach(([key, value]) => {
            if (value !== null && value !== undefined) {
                url.searchParams.set(key, value);
            }
        });
        
        return url.toString();
    }

    /**
     * Local Storage Utilities with error handling
     */
    
    /**
     * Safe localStorage set
     */
    setLocalStorage(key, value, ttl = null) {
        try {
            const item = {
                value,
                timestamp: Date.now(),
                ttl
            };
            localStorage.setItem(key, JSON.stringify(item));
            return true;
        } catch (error) {
            console.warn('Failed to set localStorage:', error);
            return false;
        }
    }

    /**
     * Safe localStorage get with TTL support
     */
    getLocalStorage(key, defaultValue = null) {
        try {
            const itemStr = localStorage.getItem(key);
            if (!itemStr) return defaultValue;
            
            const item = JSON.parse(itemStr);
            
            // Check TTL
            if (item.ttl && Date.now() - item.timestamp > item.ttl) {
                localStorage.removeItem(key);
                return defaultValue;
            }
            
            return item.value;
        } catch (error) {
            console.warn('Failed to get localStorage:', error);
            return defaultValue;
        }
    }

    /**
     * Performance measurement utilities
     */
    
    /**
     * Measure function execution time
     */
    measureTime(func, name = 'Function') {
        return (...args) => {
            const start = performance.now();
            const result = func.apply(this, args);
            const end = performance.now();
            
            console.log(`⏱️ ${name} execution time: ${(end - start).toFixed(2)}ms`);
            
            return result;
        };
    }

    /**
     * Create performance mark
     */
    markPerformance(name) {
        performance.mark(name);
    }

    /**
     * Measure performance between marks
     */
    measurePerformance(name, startMark, endMark) {
        performance.measure(name, startMark, endMark);
        const measure = performance.getEntriesByName(name)[0];
        console.log(`📊 ${name}: ${measure.duration.toFixed(2)}ms`);
        return measure.duration;
    }

    /**
     * Error handling utilities
     */
    
    /**
     * Safe async function execution
     */
    async safeAsync(asyncFunc, fallback = null) {
        try {
            return await asyncFunc();
        } catch (error) {
            console.error('Async function failed:', error);
            return fallback;
        }
    }

    /**
     * Retry async function with exponential backoff
     */
    async retryAsync(asyncFunc, maxRetries = 3, delay = 1000) {
        for (let i = 0; i < maxRetries; i++) {
            try {
                return await asyncFunc();
            } catch (error) {
                if (i === maxRetries - 1) throw error;
                
                const waitTime = delay * Math.pow(2, i);
                console.log(`Retrying in ${waitTime}ms... (attempt ${i + 1}/${maxRetries})`);
                await this.delay(waitTime);
            }
        }
    }

    /**
     * Delay execution
     */
    delay(ms) {
        return new Promise(resolve => setTimeout(resolve, ms));
    }

    /**
     * Cleanup utilities
     */
    
    /**
     * Clear all caches and timers
     */
    cleanup() {
        // Clear debounce timers
        this.debounceTimers.forEach(timer => clearTimeout(timer));
        this.debounceTimers.clear();
        
        // Clear throttle timers
        this.throttleTimers.forEach(timer => clearTimeout(timer));
        this.throttleTimers.clear();
        
        // Clear memoize cache
        this.memoizeCache.clear();
        
        // Clear observer pool
        this.observerPool.forEach(observer => observer.disconnect());
        this.observerPool.clear();
        
        console.log('🧹 Utils module cleaned up');
    }

    /**
     * Development utilities
     */
    
    /**
     * Create mock data for testing
     */
    createMockOrder(id = 1) {
        return {
            ID: id,
            Name: `Клиент ${id}`,
            Phone: `+7 (978) 900-${String(id).padStart(4, '0')}`,
            Address: `ул. Тестовая, д. ${id}`,
            Category: 'waste_removal',
            Subcategory: 'Строительный мусор',
            Status: ['new', 'in_progress', 'completed'][id % 3],
            Cost: { Valid: true, Float64: 1000 + id * 100 },
            CreatedAt: new Date(Date.now() - id * 3600000).toISOString(),
            Description: `Описание заказа ${id}`
        };
    }

    /**
     * Log with timestamp and context
     */
    log(message, context = 'Utils', level = 'info') {
        const timestamp = new Date().toISOString();
        const emoji = {
            info: 'ℹ️',
            warn: '⚠️',
            error: '❌',
            success: '✅',
            debug: '🐛'
        };
        
        console[level](`${emoji[level]} [${timestamp}] [${context}] ${message}`);
    }
}

// Делаем UtilsModule доступным глобально
window.UtilsModule = UtilsModule; 