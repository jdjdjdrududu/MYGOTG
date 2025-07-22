/**
 * Сервис-Крым Service Worker v3.0
 * Modern caching and offline functionality
 */

const CACHE_NAME = 'service-crym-v3.0';
const API_CACHE_NAME = 'service-crym-api-v3.0';
const MEDIA_CACHE_NAME = 'service-crym-media-v3.0';

// Static files to cache
const STATIC_FILES = [
    '/',
    '/index.html',
    '/config.js',
    '/manifest.json',
    'https://telegram.org/js/telegram-web-app.js'
];

// API endpoints to cache with their strategies
const API_CACHE_STRATEGIES = {
    '/api/user/profile': { strategy: 'stale-while-revalidate', maxAge: 300000 }, // 5 min
    '/api/user/orders': { strategy: 'network-first', maxAge: 60000 }, // 1 min  
    '/api/admin/orders': { strategy: 'network-first', maxAge: 30000 }, // 30 sec
    '/api/admin/clients': { strategy: 'stale-while-revalidate', maxAge: 300000 }, // 5 min
    '/api/test/profile': { strategy: 'cache-first', maxAge: 3600000 } // 1 hour (test only)
};

// Performance metrics
let metrics = {
    cacheHits: 0,
    cacheMisses: 0,
    networkRequests: 0,
    offlineRequests: 0,
    totalRequests: 0
};

/**
 * Service Worker Installation
 */
self.addEventListener('install', event => {
    console.log('🔧 Сервис-Крым SW v3.0 installing...');
    
    event.waitUntil(
        Promise.all([
            // Cache static files
            caches.open(CACHE_NAME).then(cache => {
                console.log('📦 Caching static files...');
                return cache.addAll(STATIC_FILES);
            }),
            
            // Initialize API cache
            caches.open(API_CACHE_NAME).then(cache => {
                console.log('📡 API cache initialized');
                return cache;
            }),
            
            // Initialize media cache
            caches.open(MEDIA_CACHE_NAME).then(cache => {
                console.log('🖼️ Media cache initialized');
                return cache;
            })
        ]).then(() => {
            console.log('✅ Service Worker installed successfully');
            // Skip waiting to activate immediately
            return self.skipWaiting();
        }).catch(error => {
            console.error('❌ Service Worker installation failed:', error);
        })
    );
});

/**
 * Service Worker Activation
 */
self.addEventListener('activate', event => {
    console.log('🚀 Сервис-Крым SW v3.0 activating...');
    
    event.waitUntil(
        Promise.all([
            // Clean up old caches
            cleanupOldCaches(),
            
            // Take control of all clients immediately
            self.clients.claim()
        ]).then(() => {
            console.log('✅ Service Worker activated and controlling all clients');
        })
    );
});

/**
 * Fetch Event Handler
 */
self.addEventListener('fetch', event => {
    metrics.totalRequests++;
    
    // Skip non-GET requests and chrome-extension requests
    if (event.request.method !== 'GET' || event.request.url.startsWith('chrome-extension://')) {
        return;
    }
    
    event.respondWith(handleRequest(event.request));
});

/**
 * Handle different types of requests
 */
async function handleRequest(request) {
    const url = new URL(request.url);
    
    try {
        // Static files
        if (isStaticFile(url)) {
            return await cacheFirstStrategy(request, CACHE_NAME);
        }
        
        // API requests
        if (isApiRequest(url)) {
            return await handleApiRequest(request);
        }
        
        // Media files
        if (isMediaRequest(url)) {
            return await staleWhileRevalidateStrategy(request, MEDIA_CACHE_NAME);
        }
        
        // Default: Network first for everything else
        return await networkFirstStrategy(request, CACHE_NAME);
        
    } catch (error) {
        console.error('❌ Request failed:', request.url, error);
        metrics.offlineRequests++;
        return await handleOfflineRequest(request);
    }
}

/**
 * Handle API requests with specific strategies
 */
async function handleApiRequest(request) {
    const url = new URL(request.url);
    const pathname = url.pathname;
    
    // Find matching API cache strategy
    let strategy = null;
    for (const [pattern, config] of Object.entries(API_CACHE_STRATEGIES)) {
        if (pathname.includes(pattern)) {
            strategy = config;
            break;
        }
    }
    
    // Default strategy for unknown API endpoints
    if (!strategy) {
        strategy = { strategy: 'network-first', maxAge: 60000 };
    }
    
    // Apply the appropriate strategy
    switch (strategy.strategy) {
        case 'cache-first':
            return await cacheFirstStrategy(request, API_CACHE_NAME, strategy.maxAge);
        case 'network-first':
            return await networkFirstStrategy(request, API_CACHE_NAME, strategy.maxAge);
        case 'stale-while-revalidate':
            return await staleWhileRevalidateStrategy(request, API_CACHE_NAME, strategy.maxAge);
        default:
            return await networkFirstStrategy(request, API_CACHE_NAME, strategy.maxAge);
    }
}

/**
 * Cache First Strategy
 */
async function cacheFirstStrategy(request, cacheName, maxAge = null) {
    const cache = await caches.open(cacheName);
    const cachedResponse = await cache.match(request);
    
    if (cachedResponse && (!maxAge || !isCacheExpired(cachedResponse, maxAge))) {
        metrics.cacheHits++;
        return cachedResponse;
    }
    
    try {
        metrics.networkRequests++;
        const networkResponse = await fetch(request);
        
        if (networkResponse.ok) {
            // Cache the response
            const responseToCache = networkResponse.clone();
            await cache.put(request, addTimestamp(responseToCache));
        }
        
        return networkResponse;
    } catch (error) {
        metrics.cacheMisses++;
        // Return cached version if available, even if expired
        if (cachedResponse) {
            return cachedResponse;
        }
        throw error;
    }
}

/**
 * Network First Strategy
 */
async function networkFirstStrategy(request, cacheName, maxAge = null) {
    const cache = await caches.open(cacheName);
    
    try {
        metrics.networkRequests++;
        const networkResponse = await fetch(request);
        
        if (networkResponse.ok) {
            // Cache successful responses
            const responseToCache = networkResponse.clone();
            await cache.put(request, addTimestamp(responseToCache));
        }
        
        return networkResponse;
    } catch (error) {
        metrics.cacheMisses++;
        // Try cache as fallback
        const cachedResponse = await cache.match(request);
        
        if (cachedResponse && (!maxAge || !isCacheExpired(cachedResponse, maxAge))) {
            metrics.cacheHits++;
            return cachedResponse;
        }
        
        throw error;
    }
}

/**
 * Stale While Revalidate Strategy
 */
async function staleWhileRevalidateStrategy(request, cacheName, maxAge = null) {
    const cache = await caches.open(cacheName);
    const cachedResponse = await cache.match(request);
    
    // Start fetch in background
    const fetchPromise = fetch(request).then(response => {
        if (response.ok) {
            cache.put(request, addTimestamp(response.clone()));
        }
        return response;
    }).catch(error => {
        console.warn('Background fetch failed:', request.url, error);
    });
    
    // Return cached response immediately if available and not expired
    if (cachedResponse && (!maxAge || !isCacheExpired(cachedResponse, maxAge))) {
        metrics.cacheHits++;
        // Don't await the fetch promise
        fetchPromise;
        return cachedResponse;
    }
    
    // If no cache or expired, wait for network
    metrics.networkRequests++;
    try {
        return await fetchPromise;
    } catch (error) {
        metrics.cacheMisses++;
        // Return stale cache as last resort
        if (cachedResponse) {
            return cachedResponse;
        }
        throw error;
    }
}

/**
 * Handle offline requests
 */
async function handleOfflineRequest(request) {
    const url = new URL(request.url);
    
    // Try to find any cached version
    const caches_names = [CACHE_NAME, API_CACHE_NAME, MEDIA_CACHE_NAME];
    
    for (const cacheName of caches_names) {
        const cache = await caches.open(cacheName);
        const cachedResponse = await cache.match(request);
        if (cachedResponse) {
            return cachedResponse;
        }
    }
    
    // Return offline page for navigation requests
    if (request.mode === 'navigate') {
        const cache = await caches.open(CACHE_NAME);
        const offlinePage = await cache.match('/index.html');
        if (offlinePage) {
            return offlinePage;
        }
    }
    
    // Return offline JSON response for API requests
    if (isApiRequest(url)) {
        return new Response(
            JSON.stringify({
                status: 'error',
                message: 'Нет подключения к интернету',
                data: null,
                offline: true
            }),
            {
                status: 503,
                statusText: 'Service Unavailable',
                headers: {
                    'Content-Type': 'application/json',
                    'Cache-Control': 'no-cache'
                }
            }
        );
    }
    
    // Fallback response
    return new Response('Offline', {
        status: 503,
        statusText: 'Service Unavailable'
    });
}

/**
 * Add timestamp to response headers for cache expiration
 */
function addTimestamp(response) {
    const headers = new Headers(response.headers);
    headers.set('sw-cache-timestamp', Date.now().toString());
    
    return new Response(response.body, {
        status: response.status,
        statusText: response.statusText,
        headers: headers
    });
}

/**
 * Check if cached response is expired
 */
function isCacheExpired(response, maxAge) {
    if (!maxAge) return false;
    
    const timestamp = response.headers.get('sw-cache-timestamp');
    if (!timestamp) return true;
    
    const age = Date.now() - parseInt(timestamp);
    return age > maxAge;
}

/**
 * Clean up old caches
 */
async function cleanupOldCaches() {
    const cacheNames = await caches.keys();
    const currentCaches = [CACHE_NAME, API_CACHE_NAME, MEDIA_CACHE_NAME];
    
    const deletionPromises = cacheNames
        .filter(cacheName => !currentCaches.includes(cacheName))
        .map(cacheName => {
            console.log('🗑️ Deleting old cache:', cacheName);
            return caches.delete(cacheName);
        });
    
    return Promise.all(deletionPromises);
}

/**
 * Helper functions to determine request types
 */
function isStaticFile(url) {
    const pathname = url.pathname;
    return pathname === '/' || 
           pathname.endsWith('.html') || 
           pathname.endsWith('.js') || 
           pathname.endsWith('.css') || 
           pathname.endsWith('.json') ||
           pathname.includes('telegram-web-app.js');
}

function isApiRequest(url) {
    return url.pathname.startsWith('/api/');
}

function isMediaRequest(url) {
    const pathname = url.pathname;
    return pathname.includes('/media/') || 
           pathname.match(/\.(jpg|jpeg|png|gif|webp|svg|mp4|webm|ogg)$/i);
}

/**
 * Background sync for failed requests
 */
self.addEventListener('sync', event => {
    if (event.tag === 'background-sync') {
        event.waitUntil(doBackgroundSync());
    }
});

async function doBackgroundSync() {
    console.log('🔄 Background sync started');
    // Here you could retry failed requests stored in IndexedDB
    // For now, just log that sync happened
}

/**
 * Handle push notifications
 */
self.addEventListener('push', event => {
    if (!event.data) return;
    
    const data = event.data.json();
    const options = {
        body: data.body || 'Новое уведомление',
        icon: '/icons/icon-192x192.png',
        badge: '/icons/icon-72x72.png',
        tag: data.tag || 'general',
        requireInteraction: data.requireInteraction || false,
        actions: data.actions || []
    };
    
    event.waitUntil(
        self.registration.showNotification(data.title || 'Сервис-Крым', options)
    );
});

/**
 * Handle notification clicks
 */
self.addEventListener('notificationclick', event => {
    event.notification.close();
    
    event.waitUntil(
        clients.openWindow(event.notification.data?.url || '/')
    );
});

/**
 * Message handler for communication with main app
 */
self.addEventListener('message', event => {
    const { type, data } = event.data;
    
    switch (type) {
        case 'GET_METRICS':
            event.ports[0].postMessage(metrics);
            break;
            
        case 'CLEAR_CACHE':
            event.waitUntil(clearAllCaches().then(() => {
                event.ports[0].postMessage({ success: true });
            }));
            break;
            
        case 'SKIP_WAITING':
            self.skipWaiting();
            break;
            
        default:
            console.log('Unknown message type:', type);
    }
});

/**
 * Clear all caches
 */
async function clearAllCaches() {
    const cacheNames = await caches.keys();
    const deletionPromises = cacheNames.map(name => caches.delete(name));
    await Promise.all(deletionPromises);
    
    // Reset metrics
    metrics = {
        cacheHits: 0,
        cacheMisses: 0,
        networkRequests: 0,
        offlineRequests: 0,
        totalRequests: 0
    };
    
    console.log('🗑️ All caches cleared');
}

// Log Service Worker startup
console.log('🚀 Сервис-Крым Service Worker v3.0 started'); 