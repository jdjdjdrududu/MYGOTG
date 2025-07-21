// service-worker.js - Кэширование и поддержка офлайн-режима

const CACHE_NAME = 'service-crym-cache-v1';
const ASSETS_TO_CACHE = [
  './',
  './index.html',
  './style.css',
  './app.js',
  './lottie/вработе.json',
  './lottie/демонтаж.json',
  './lottie/ждать.json',
  './lottie/лупа.json',
  './lottie/мусор.json',
  './lottie/новый.json',
  './lottie/ок.json',
  './lottie/отмена.json',
  './lottie/плюс.json',
  './lottie/пусто.json',
  './lottie/рубль.json',
  './lottie/сейчас.json',
  'https://telegram.org/js/telegram-web-app.js',
  'https://cdnjs.cloudflare.com/ajax/libs/hammer.js/2.0.8/hammer.min.js',
  'https://cdnjs.cloudflare.com/ajax/libs/lottie-web/5.12.2/lottie.min.js',
  'https://cdnjs.cloudflare.com/ajax/libs/font-awesome/6.5.2/css/all.min.css',
  'https://cdn.jsdelivr.net/npm/swiper/swiper-bundle.min.css',
  'https://cdn.jsdelivr.net/npm/swiper/swiper-bundle.min.js'
];

// Установка Service Worker и кэширование статических ресурсов
self.addEventListener('install', (event) => {
  event.waitUntil(
    caches.open(CACHE_NAME)
      .then((cache) => {
        console.log('Кэширование статических ресурсов');
        return cache.addAll(ASSETS_TO_CACHE);
      })
      .then(() => self.skipWaiting())
  );
});

// Активация Service Worker и удаление старых кэшей
self.addEventListener('activate', (event) => {
  event.waitUntil(
    caches.keys().then((cacheNames) => {
      return Promise.all(
        cacheNames.map((cacheName) => {
          if (cacheName !== CACHE_NAME) {
            console.log('Удаление старого кэша:', cacheName);
            return caches.delete(cacheName);
          }
        })
      );
    }).then(() => self.clients.claim())
  );
});

// Стратегия кэширования: сначала кэш, затем сеть
self.addEventListener('fetch', (event) => {
  // Пропускаем запросы к API
  if (event.request.url.includes('/api/')) {
    return;
  }

  event.respondWith(
    caches.match(event.request)
      .then((cachedResponse) => {
        // Возвращаем из кэша, если есть
        if (cachedResponse) {
          return cachedResponse;
        }

        // Иначе делаем запрос к сети
        return fetch(event.request)
          .then((response) => {
            // Если ответ не валидный, просто возвращаем его
            if (!response || response.status !== 200 || response.type !== 'basic') {
              return response;
            }

            // Кэшируем новый ответ
            const responseToCache = response.clone();
            caches.open(CACHE_NAME)
              .then((cache) => {
                cache.put(event.request, responseToCache);
              });

            return response;
          })
          .catch(() => {
            // Если нет сети и нет кэша, возвращаем страницу офлайн
            if (event.request.headers.get('accept').includes('text/html')) {
              return caches.match('./offline.html');
            }
          });
      })
  );
});

// Обработка сообщений от клиента
self.addEventListener('message', (event) => {
  if (event.data && event.data.type === 'SKIP_WAITING') {
    self.skipWaiting();
  }
}); 