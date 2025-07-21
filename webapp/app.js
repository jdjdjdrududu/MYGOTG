// app.js - Оптимизированный и структурированный код приложения

/**
 * @section Конфигурация
 * В этом разделе определяются статические конфигурации приложения.
 */

// Проверка состояния сети и обработка офлайн-режима
let isOnline = navigator.onLine;
window.addEventListener('online', () => {
    isOnline = true;
    document.getElementById('error-message').classList.add('hidden');
    document.getElementById('error-message').textContent = '';
    // Обновляем данные, если они были загружены в офлайн-режиме
    if (App && App.state && App.state.lastActivePanel) {
        App.ui.showPanel(App.state.lastActivePanel);
    }
});

window.addEventListener('offline', () => {
    isOnline = false;
    document.getElementById('error-message').classList.remove('hidden');
    document.getElementById('error-message').textContent = 'Нет подключения к интернету. Некоторые функции могут быть недоступны.';
});
const lottieOrderIcons = {
    waste_removal: "lottie/мусор.json", // Анимация для вывоза мусора
    demolition: "lottie/демонтаж.json"  // Анимация для демонтажа
};

const App = {
    // Конфигурация статусов заказов для карусели оператора
    orderStatuses: [
        { key: 'active', label: 'Активные' },
        { key: 'new', label: 'Новые' },
        { key: 'awaiting_confirmation', label: 'Ожидают' },
        { key: 'in_progress', label: 'В работе' },
        { key: 'completed', label: 'Завершённые' },
        { key: 'canceled', label: 'Отменённые' }
    ],

    // Карта для Lottie-анимаций, используемых в различных элементах UI
    lottieIconMap: {
        new: 'lottie/новый.json',
        awaiting: 'lottie/ждать.json',
        awaiting_confirmation: 'lottie/ждать.json',
        in_progress: 'lottie/вработе.json',
        completed: 'lottie/ок.json',
        calculated: 'lottie/рубль.json',
        canceled: 'lottie/отмена.json',
        active: 'lottie/сейчас.json',
        empty_box: 'lottie/пусто.json',
        error_sign: 'lottie/error.json',
        create_order: 'lottie/плюс.json',
        contact_operator: 'lottie/поддержка.json',
        my_orders: 'lottie/мои_заказы.json',
        plus_icon: 'lottie/плюс.json',
        search_icon: 'lottie/лупа.json'
    },

    // Категории и подкатегории для форм создания заказа
    categoriesConfig: {
        waste_removal: {
            displayName: 'Вывоз мусора',
            subcategories: [
                { key: 'construct', label: 'Строительный' },
                { key: 'household', label: 'Бытовой' },
                { key: 'metal', label: 'Металл' },
                { key: 'junk', label: 'Хлам' },
                { key: 'greenery', label: 'Ветки, деревья, трава' },
                { key: 'tires', label: 'Старые покрышки' },
                { key: 'other_waste', label: 'Другое' }
            ]
        },
        demolition: {
            displayName: 'Демонтаж',
            subcategories: [
                { key: 'walls', label: 'Стены' },
                { key: 'partitions', label: 'Перегородки' },
                { key: 'floors', label: 'Полы' },
                { key: 'ceilings', label: 'Потолки' },
                { key: 'plumbing', label: 'Сантехника' },
                { key: 'tiles', label: 'Плитка' },
                { key: 'other_demo', label: 'Другое' }
            ]
        },
        construction_materials: { displayName: 'Стройматериалы', subcategories: [] },
        other: { displayName: 'Другое', subcategories: [] }
    },

    // Отображаемые названия для полей, статусов и ролей
    displayNamesMap: {
        fields: {
            'category': 'Категория', 'subcategory': 'Подкатегория', 'description': 'Описание',
            'name': 'Имя клиента', 'phone': 'Телефон', 'address': 'Адрес',
            'date': 'Дата', 'time': 'Время', 'payment': 'Оплата', 'cost': 'Стоимость',
            'media': 'Фото/Видео'
        },
        statuses: {
            'new': 'Новый', 'awaiting_cost': 'Ожидание стоимости', 'awaiting_confirmation': 'Ожидаем клиента',
            'awaiting_payment': 'Ожидание оплаты', 'in_progress': 'В работе', 'completed': 'Завершён',
            'canceled': 'Отменён', 'draft': 'Черновик', 'calculated': 'Рассчитан', 'settled': 'Закрыт (оплачен)'
        },
        roles: {
            'user': 'Пользователь', 'operator': 'Операторы', 'main_operator': 'Главные операторы',
            'driver': 'Водители', 'loader': 'Грузчики', 'owner': 'Владелец'
        }
    },

    /**
     * @section Состояние приложения
     */
    state: {
        tg: window.Telegram.WebApp,
        user: null,
        orders: {},
        clients: [],
        staff: {},
        currentPanel: 'orders-panel',
        apiBaseUrl: 'https://xn----ctbinlmxece7i.xn--p1ai',
        selectedClient: null,
        selectedOrder: null,
        currentStaffRole: null,
        telegramBotUsername: null,
        tabsSwiper: null,
        contentSwiper: null,
        loadedStatuses: new Set(),
        userOrders: [],
        selectedMediaFiles: {
            operator: [],
            user: []
        },
        operatorMediaSwiper: null,
        userMediaSwiper: null,
        fullscreenSwiper: null
    },

    /**
     * @section Инициализация
     */
    init() {
        this.state.tg.ready();
        this.state.tg.expand();
        this.state.tg.BackButton.hide();

        if (!this.state.tg.initData) {
            this.ui.showError("Ошибка: не удалось получить данные авторизации от Telegram.");
            return;
        }

        this.api.fetchUserProfile().then(user => {
            this.state.user = user;
            this.ui.setupUIForRole(user.Role);

            if (['operator', 'main_operator', 'owner'].includes(user.Role)) {
                this.ui.setupOrderCarousel();
                this.initSwiper();
            }
        }).catch(error => {
            this.ui.showError(`Не удалось загрузить ваш профиль. ${error.message}`);
        });

        this.bindEventListeners();
    },

    initSwiper() {
        const tabsSwiper = new Swiper('#status-tabs-swiper', {
            slidesPerView: 'auto',
            spaceBetween: 10,
            freeMode: true,
            watchSlidesVisibility: true,
            watchSlidesProgress: true,
        });

        const contentSwiper = new Swiper('#orders-content-swiper', {
            spaceBetween: 20,
            thumbs: {
                swiper: tabsSwiper,
            },
            on: {
                slideChange: () => {
                    const newIndex = contentSwiper.activeIndex;
                    const activeStatus = this.orderStatuses[newIndex].key;

                    document.querySelectorAll('#status-tabs-swiper .swiper-slide').forEach((tab, index) => {
                        tab.classList.toggle('active-tab', index === newIndex);
                    });
                    tabsSwiper.slideTo(newIndex);

                    if (!this.state.loadedStatuses.has(activeStatus)) {
                        this.handlers.handleFetchOrders(activeStatus);
                    }
                }
            }
        });

        document.querySelectorAll('#status-tabs-swiper .swiper-slide').forEach((tab, index) => {
            tab.addEventListener('click', () => {
                contentSwiper.slideTo(index);
            });
        });

        this.state.tabsSwiper = tabsSwiper;
        this.state.contentSwiper = contentSwiper;
    },

    bindEventListeners() {
        document.getElementById('fab-create-order')?.addEventListener('click', () => {
            // Очищаем временное состояние клиента, так как это создание нового заказа "с нуля"
            this.state.clientForNewOrder = null;
            document.getElementById('create-order-form').reset(); // Сбрасываем форму
            this.ui.updateSubcategoriesForm('category-select', 'subcategory-select');
            this.ui.showPanel('order-creation-panel', 'forward', () => {
                this.handlers.handleFormOpen('operator');
            });
        });

        document.getElementById('create-order-from-empty-state')?.addEventListener('click', () => {
            this.ui.setupUserOrderForm();
            this.ui.showPanel('user-order-creation-panel', 'forward', () => {
                this.handlers.handleFormOpen('user');
            });
        });

        document.getElementById('user-create-order-form')?.addEventListener('submit', this.handlers.handleUserCreateOrderSubmit.bind(this.handlers));
        document.getElementById('user-category-select')?.addEventListener('change', () => this.ui.updateSubcategoriesForm('user-category-select', 'user-subcategory-select'));
        document.getElementById('open-operator-chat-btn')?.addEventListener('click', () => this.handlers.handleContactOperator())

        document.querySelectorAll('.take-photo').forEach(button => {
            button.addEventListener('click', (e) => {
                const container = e.currentTarget.closest('.media-buttons-container');
                this.handlers.triggerFileInput(container.dataset.formType, 'environment');
            });
        });

        document.querySelectorAll('.choose-gallery').forEach(button => {
            button.addEventListener('click', (e) => {
                const container = e.currentTarget.closest('.media-buttons-container');
                this.handlers.triggerFileInput(container.dataset.formType, null);
            });
        });

        document.getElementById('fab-create-staff')?.addEventListener('click', () => this.ui.showAddStaffModal());
        document.getElementById('close-staff-modal')?.addEventListener('click', () => this.ui.hideAddStaffModal());
        document.getElementById('add-staff-modal-overlay')?.addEventListener('click', (e) => {
            if (e.target.id === 'add-staff-modal-overlay') this.ui.hideAddStaffModal();
        });
        document.getElementById('add-staff-form')?.addEventListener('submit', this.handlers.handleCreateStaffSubmit.bind(this.handlers));
        document.querySelectorAll('.role-category-btn').forEach(btn => {
            btn.addEventListener('click', this.handlers.handleRoleClick.bind(this.handlers));
        });

        document.getElementById('create-order-form')?.addEventListener('submit', this.handlers.handleOperatorCreateOrderSubmit.bind(this.handlers));
        document.getElementById('category-select')?.addEventListener('change', () => this.ui.updateSubcategoriesForm('category-select', 'subcategory-select'));

        document.getElementById('create-order-for-client-btn')?.addEventListener('click', () => this.handlers.handleCreateOrderForClient());
        document.getElementById('view-client-chats-btn')?.addEventListener('click', () => this.handlers.handleViewClientChats());
        document.getElementById('block-client-btn')?.addEventListener('click', () => this.handlers.handleBlockClient());
        document.getElementById('unblock-client-btn')?.addEventListener('click', () => this.handlers.handleUnblockClient());

        document.getElementById('close-fullscreen-media')?.addEventListener('click', () => this.ui.closeFullScreenMedia());

        document.querySelectorAll('.back-button').forEach(btn => {
            btn.addEventListener('click', (e) => {
                const targetPanel = e.currentTarget.dataset.targetPanel;
                this.ui.showPanel(targetPanel, 'backward');
            });
        });

        document.getElementById('orders-search-input')?.addEventListener('input', (e) => {
            const query = e.target.value;
            const currentContentSlide = document.querySelector('#orders-content-swiper .swiper-slide-active');
            if (!currentContentSlide) return;

            const listId = currentContentSlide.querySelector('.content-list').id;
            const statusKey = listId.replace('orders-list-', '');
            const originalData = App.state.orders[statusKey] || [];

            this.ui.filterList(query, originalData, listId, this.ui.renderOrders.bind(this.ui));
        });

        document.getElementById('clients-search-input')?.addEventListener('input', (e) => {
            this.ui.filterList(e.target.value, this.state.clients, 'clients-list', this.ui.renderClients.bind(this.ui));
        });

        document.querySelector('.order-detail-actions')?.addEventListener('click', this.handlers.handleOrderActionClick.bind(this.handlers));

        this.setupGestures();
    },

    setupGestures() {
        const orderDetailPanel = document.getElementById('order-detail-panel');
        if (orderDetailPanel) {
            const orderHammer = new Hammer(orderDetailPanel);
            orderHammer.on('swiperight', (ev) => {
                if (ev.target.closest('#order-detail-media-gallery')) { return; }
                const backToPanel = App.state.user.Role === 'user' ? 'user-panel' : 'orders-panel';
                this.ui.showPanel(backToPanel, 'backward');
            });
        }

        const staffListPanel = document.getElementById('staff-list-panel');
        if (staffListPanel) {
            const staffHammer = new Hammer(staffListPanel);
            staffHammer.on('swipeleft', (ev) => {
                const card = ev.target.closest('.staff-card');
                if (!card) return;
                const chatId = card.dataset.chatId;
                if(chatId && chatId !== "0") { App.state.tg.openTelegramLink(`tg://user?id=${chatId}`); }
                else { App.ui.showError("У этого пользователя не задан ID в Telegram."); }
            });
        }
        this.ui.setupDraggableModal('add-staff-modal-window', 'isStaffModalDragging', this.ui.hideAddStaffModal.bind(this.ui));
    },

    /**
     * @section API-методы
     */
    api: {
        async _fetch(endpoint, options = {}) {
            // Проверяем состояние сети
            if (!navigator.onLine) {
                App.ui.showError('Нет подключения к интернету. Пожалуйста, проверьте соединение.');
                return Promise.reject(new Error('Нет подключения к интернету'));
            }
            
            // Не показываем глобальный лоадер для загрузки файлов, т.к. там свой индикатор
            if (!options.body || !(options.body instanceof FormData)) {
                App.ui.showLoader(true);
            }

            const defaultHeaders = { 'X-Telegram-Auth': App.state.tg.initData };

            const fetchOptions = {
                method: 'GET',
                headers: defaultHeaders,
                ...options
            };

            if (!(options.body instanceof FormData) && typeof options.body === 'object' && options.body !== null) {
                fetchOptions.headers['Content-Type'] = 'application/json';
                fetchOptions.body = JSON.stringify(options.body);
            }

            // Добавляем таймаут для запроса
            const controller = new AbortController();
            const timeoutId = setTimeout(() => controller.abort(), 15000); // 15 секунд таймаут
            
            fetchOptions.signal = controller.signal;

            try {
                const response = await fetch(`${App.state.apiBaseUrl}${endpoint}`, fetchOptions);
                clearTimeout(timeoutId);
                
                if (!response.ok) {
                    const errorData = await response.json().catch(() => ({ message: response.statusText, data: null }));
                    const err = new Error(errorData.message || 'Ошибка сервера');
                    err.data = errorData.data;
                    throw err;
                }
                
                const text = await response.text();
                const jsonData = text ? JSON.parse(text) : {};

                // Проверяем статус в ответе от нашего API
                if (jsonData.status === 'error') {
                    throw new Error(jsonData.message);
                }

                // Сохраняем в localStorage для оффлайн-режима
                try {
                    if (!endpoint.includes('/api/upload-media')) {
                        const cacheKey = `cache_${endpoint.replace(/[^a-zA-Z0-9]/g, '_')}`;
                        localStorage.setItem(cacheKey, JSON.stringify({
                            data: jsonData,
                            timestamp: Date.now()
                        }));
                    }
                } catch (e) {
                    console.warn('Не удалось сохранить данные в кэш:', e);
                }

                // Для загрузки файлов возвращаем всё тело, для остальных - поле data
                if (endpoint.includes('/api/upload-media')) {
                    return jsonData;
                }

                return jsonData.data || jsonData;

            } catch (error) {
                // Специальная обработка ошибок
                if (error.name === 'AbortError') {
                    App.ui.showError('Превышено время ожидания ответа от сервера. Пожалуйста, попробуйте позже.');
                } else if (error.message === 'Failed to fetch' && !navigator.onLine) {
                    App.ui.showError('Нет подключения к интернету. Пожалуйста, проверьте соединение.');
                    
                    // Попытка получить данные из кэша
                    try {
                        const cacheKey = `cache_${endpoint.replace(/[^a-zA-Z0-9]/g, '_')}`;
                        const cachedData = localStorage.getItem(cacheKey);
                        
                        if (cachedData) {
                            const { data, timestamp } = JSON.parse(cachedData);
                            // Используем кэш, только если он не старше 1 часа
                            if (Date.now() - timestamp < 3600000) {
                                console.log('Используем кэшированные данные для:', endpoint);
                                return data;
                            }
                        }
                    } catch (e) {
                        console.warn('Не удалось получить данные из кэша:', e);
                    }
                }
                
                throw error;
            } finally {
                if (!options.body || !(options.body instanceof FormData)) {
                    App.ui.showLoader(false);
                }
            }
        },

        fetchUserProfile() { return this._fetch('/api/user/profile'); },
        fetchOrders(statusKey = 'active') { return this._fetch(`/api/admin/orders?status=${statusKey}`); },
        fetchOrderDetails(orderId) { return this._fetch(`/api/admin/order/${orderId}`); },
        fetchClients() { return this._fetch('/api/admin/clients'); },
        fetchClientDetails(clientId) { return this._fetch(`/api/admin/client/${clientId}`); },
        createOrderForOperator(payload) { return this._fetch('/api/admin/create-order', { method: 'POST', body: payload }); },
        createOrderForUser(payload) { return this._fetch('/api/user/create-order', { method: 'POST', body: payload }); },
        fetchStaff(role) { return this._fetch(`/api/admin/clients?role=${role}`); },
        addStaff(payload) { return this._fetch('/api/admin/staff/add', { method: 'POST', body: payload }); },
        fetchUserOrders() { return this._fetch('/api/user/orders'); },
        fetchUserOrderDetails(orderId) { return this._fetch(`/api/user/order/${orderId}`); },
        adminOrderAction(orderId, payload) { return this._fetch(`/api/admin/order/${orderId}/action`, { method: 'POST', body: payload }); },
        driverOrderAction(orderId, payload) { return this._fetch(`/api/driver/order/${orderId}/action`, { method: 'POST', body: payload }); },
        userOrderAction(orderId, payload) { return this._fetch(`/api/user/order/${orderId}/action`, { method: 'POST', body: payload }); },
        updateSettlementStatus(settlementId, payload) { return this._fetch(`/api/admin/settlement/${settlementId}/status`, { method: 'POST', body: payload }); },
        updateOrderField(orderId, payload) { return this._fetch(`/api/admin/order/${orderId}/update-field`, { method: 'POST', body: payload }); },
        addOrderMedia(orderId, payload) { return this._fetch(`/api/admin/order/${orderId}/add-media`, { method: 'POST', body: payload }); },
    },

    /**
     * @section UI-методы
     */
    ui: {
        renderLottieIcon(container, animationPath) {
            if (!container || !animationPath) return;
            container.innerHTML = '';
            try {
                lottie.loadAnimation({
                    container: container, renderer: 'svg', loop: true, autoplay: true, path: animationPath
                });
            } catch (error) { console.error(`Lottie error for path ${animationPath}:`, error); container.textContent = '⚠️'; }
        },

        initMediaSwiper(formType) {
            const selector = `#${formType}-media-swiper`;
            if (App.state[`${formType}MediaSwiper`] && !App.state[`${formType}MediaSwiper`].destroyed) {
                App.state[`${formType}MediaSwiper`].destroy(true, true);
            }
            const swiperInstance = new Swiper(selector, {
                slidesPerView: 'auto', spaceBetween: 12, freeMode: true,
                pagination: { el: `${selector} .swiper-pagination`, clickable: true, },
                navigation: { nextEl: null, prevEl: null, },
                observer: false, observeParents: false, observeSlideChildren: false,
            });
            App.state[`${formType}MediaSwiper`] = swiperInstance;
            return swiperInstance;
        },

        showAddStaffModal() {
            this.showModal('add-staff-modal-overlay', () => {
                document.getElementById('add-staff-form').reset();
                const title = document.getElementById('add-staff-modal-title');
                const roleName = App.ui.getRoleDisplayName(App.state.currentStaffRole);
                title.textContent = `Добавить в "${roleName}"`;
            });
        },
        hideAddStaffModal() { this.hideModal('add-staff-modal-overlay'); },
        showModal(overlayId, beforeShowCallback) {
            const overlay = document.getElementById(overlayId);
            if (beforeShowCallback) beforeShowCallback();
            overlay.classList.remove('hidden');
            setTimeout(() => { overlay.classList.add('visible'); }, 10);
        },
        hideModal(overlayId) {
            const overlay = document.getElementById(overlayId);
            const modalWindow = overlay.querySelector('.modal-window');
            overlay.classList.remove('visible');
            const onTransitionEnd = () => {
                overlay.classList.add('hidden');
                modalWindow.style.transform = '';
                overlay.removeEventListener('transitionend', onTransitionEnd);
            };
            overlay.addEventListener('transitionend', onTransitionEnd);
        },
        setupDraggableModal(modalWindowId, stateKey, hideFunction) {
            const modalWindow = document.getElementById(modalWindowId);
            if (!modalWindow) return;
            const modalHammer = new Hammer(modalWindow, { recognizers: [[Hammer.Pan, { direction: Hammer.DIRECTION_VERTICAL }]] });
            modalHammer.on('panstart', () => { App.state[stateKey] = true; modalWindow.classList.add('no-transition'); });
            modalHammer.on('panmove', (ev) => {
                if (!App.state[stateKey]) return;
                const delta = ev.deltaY > 0 ? ev.deltaY : 0;
                modalWindow.style.transform = `translateY(${delta}px)`;
            });
            modalHammer.on('panend', (ev) => {
                if (!App.state[stateKey]) return;
                modalWindow.classList.remove('no-transition');
                const threshold = modalWindow.offsetHeight * 0.4;
                if (ev.deltaY > threshold) { hideFunction(); }
                else { modalWindow.style.transform = 'translateY(0px)'; }
                App.state[stateKey] = false;
            });
        },

        showPanel(panelId, direction = 'none', onVisibleCallback = null) {
            const newPanel = document.getElementById(panelId);
            const oldPanel = document.querySelector('.panel.visible');
            if (!newPanel || (oldPanel && oldPanel.id === panelId)) return;

            const createOrderFab = document.getElementById('fab-create-order');
            const createStaffFab = document.getElementById('fab-create-staff');

            if (['orders-panel'].includes(panelId)) {
                if (App.state.user.Role !== 'user') { createOrderFab?.classList.remove('hidden'); }
                else { createOrderFab?.classList.add('hidden'); }
                createStaffFab?.classList.add('hidden');
            } else if (panelId.startsWith('staff-')) {
                createOrderFab?.classList.add('hidden');
                createStaffFab?.classList.remove('hidden');
            } else {
                createOrderFab?.classList.add('hidden');
                createStaffFab?.classList.add('hidden');
            }

            document.body.style.overflow = 'hidden';
            if (direction === 'none') {
                oldPanel?.classList.remove('visible');
                newPanel.classList.add('visible', 'no-transition');
                requestAnimationFrame(() => {
                    newPanel.classList.remove('no-transition');
                    if (onVisibleCallback) { onVisibleCallback(); }
                });
                App.state.currentPanel = panelId;
                document.body.style.overflow = '';
                return;
            }

            if (oldPanel) {
                const oldPanelAnim = direction === 'forward' ? 'slide-out-to-left' : 'slide-out-to-right';
                oldPanel.classList.add(oldPanelAnim);
                oldPanel.addEventListener('animationend', () => { oldPanel.classList.remove('visible', oldPanelAnim); }, { once: true });
            }

            const newPanelAnim = direction === 'forward' ? 'slide-in-right' : 'slide-in-left';
            newPanel.classList.add('visible', newPanelAnim);

            const onAnimationEnd = () => {
                newPanel.classList.remove(newPanelAnim);
                document.body.style.overflow = '';
                if (onVisibleCallback) { onVisibleCallback(); }
            };
            newPanel.addEventListener('animationend', onAnimationEnd, { once: true });

            App.state.currentPanel = panelId;
        },
        showLoader(show = true) { document.getElementById('top-progress-bar')?.classList.toggle('hidden', !show); },
        showError(message) {
            const errorDiv = document.getElementById('error-message');
            if (errorDiv) {
                errorDiv.textContent = message;
                errorDiv.classList.remove('hidden');
                setTimeout(() => errorDiv.classList.add('hidden'), 5000);
            }
        },

        renderSkeleton(containerId, count = 5) {
            const container = document.getElementById(containerId);
            if (!container) return;
            const template = document.getElementById('skeleton-card-template');
            container.innerHTML = '';
            for (let i = 0; i < count; i++) { container.appendChild(template.content.cloneNode(true)); }
        },
        renderEmptyState(containerId, { iconKey, title, message, showCreateButton = false, createButtonId = null }) {
            const container = document.getElementById(containerId);
            if (!container) return;
            container.innerHTML = '';
            const template = document.getElementById('empty-state-template');
            const emptyState = template.content.cloneNode(true);
            const iconContainer = emptyState.querySelector('.empty-state-icon');
            const animationPath = App.lottieIconMap[iconKey] || App.lottieIconMap['empty_box'];
            this.renderLottieIcon(iconContainer, animationPath);
            emptyState.querySelector('.empty-state-title').textContent = title;
            emptyState.querySelector('.empty-state-message').textContent = message;
            const createButton = emptyState.querySelector('.empty-state-button');
            if (createButton) {
                if (showCreateButton && createButtonId) {
                    createButton.id = createButtonId;
                    createButton.classList.remove('hidden');
                } else { createButton.classList.add('hidden'); }
            }
            container.appendChild(emptyState);
        },

        setupUIForRole(role) {
            let actions = [];
            switch (role) {
                case 'user':
                    actions = [
                        { label: 'Создать заказ', icon: 'fa-plus', panel: 'user-order-creation-panel', handler: () => { App.ui.setupUserOrderForm(); App.handlers.handleFormOpen('user'); }},
                        { label: 'Мои заказы', icon: 'fa-box', panel: 'user-panel', handler: () => App.handlers.handleFetchUserOrders() },
                        { label: 'Связь', icon: 'fa-headset', panel: 'contact-operator-panel', handler: () => App.handlers.handleContactOperatorPage() }
                    ];
                    this.showPanel('user-panel', 'none');
                    App.handlers.handleFetchUserOrders();
                    break;
                case 'driver':
                    this.showPanel('driver-panel', 'none');
                    actions = [
                        { label: 'Мои заказы', icon: 'fa-box', panel: 'orders-panel', handler: () => App.handlers.handleFetchOrders('in_progress') },
                        { label: 'Статистика', icon: 'fa-chart-simple', panel: 'driver-panel', handler: () => {} }
                    ];
                    break;
                case 'operator': case 'main_operator': case 'owner':
                    this.showPanel('orders-panel', 'none');
                    App.handlers.handleFetchOrders('active');
                    actions = [
                        { label: 'Заказы', icon: 'fa-box', panel: 'orders-panel', handler: () => App.handlers.handleFetchOrders('active') },
                        { label: 'Клиенты', icon: 'fa-users', panel: 'clients-panel', handler: () => App.handlers.handleFetchClients() },
                        { label: 'Штат', icon: 'fa-user-group', panel: 'staff-hub-panel', handler: () => App.handlers.handleShowStaffHub() }
                    ];
                    break;
                default: this.showError("Неизвестная роль."); return;
            }
            this.renderRibbon(actions);
        },

        renderRibbon(actions) {
            const ribbonContainer = document.getElementById('bottom-ribbon-menu');
            if (!ribbonContainer) return;
            ribbonContainer.innerHTML = '';
            ribbonContainer.classList.remove('hidden');
            actions.forEach(item => {
                const button = document.createElement('button');
                button.className = 'ribbon-button';
                const icon = document.createElement('i');
                icon.className = `fa-solid ${item.icon}`;
                button.appendChild(icon);
                const labelSpan = document.createElement('span');
                labelSpan.textContent = item.label;
                button.appendChild(labelSpan);
                button.addEventListener('click', () => {
                    this.showPanel(item.panel, 'none');
                    if (item.handler) { item.handler(); }
                });
                ribbonContainer.appendChild(button);
            });
            if (actions.length > 0) { ribbonContainer.querySelector('.ribbon-button')?.classList.add('active'); }
            ribbonContainer.addEventListener('click', (e) => {
                const clickedButton = e.target.closest('.ribbon-button');
                if (clickedButton) {
                    ribbonContainer.querySelectorAll('.ribbon-button').forEach(btn => btn.classList.remove('active'));
                    clickedButton.classList.add('active');
                }
            });
        },

        setupOrderCarousel() {
            const tabsWrapper = document.getElementById('status-tabs-wrapper');
            const contentWrapper = document.getElementById('orders-content-wrapper');
            if (!tabsWrapper || !contentWrapper) return;
            tabsWrapper.innerHTML = '';
            contentWrapper.innerHTML = '';
            App.orderStatuses.forEach((status, index) => {
                const tab = document.createElement('div');
                tab.className = 'swiper-slide';
                if (index === 0) tab.classList.add('active-tab');
                tab.dataset.key = status.key;
                const iconContainer = document.createElement('div');
                iconContainer.className = 'status-tab-icon';
                const labelSpan = document.createElement('span');
                labelSpan.textContent = status.label;
                tab.appendChild(iconContainer);
                tab.appendChild(labelSpan);
                tabsWrapper.appendChild(tab);
                const animationPath = App.lottieIconMap[status.key];
                if (animationPath) { this.renderLottieIcon(iconContainer, animationPath); }
                const slide = document.createElement('div');
                slide.className = 'swiper-slide';
                slide.dataset.key = status.key;
                slide.innerHTML = `<ul id="orders-list-${status.key}" class="content-list"></ul>`;
                contentWrapper.appendChild(slide);
            });
        },

        renderOrders(orders, containerId) {
            const container = document.getElementById(containerId);
            if (!container) { console.error(`Контейнер с ID ${containerId} не найден.`); return; }
            if (containerId === 'user-orders-list') {
                const emptyStateContainer = document.getElementById('user-orders-empty-state');
                if (emptyStateContainer) {
                    if (!orders || orders.length === 0) {
                        container.classList.add('hidden');
                        emptyStateContainer.classList.remove('hidden');
                        const iconContainer = document.getElementById('user-orders-empty-icon');
                        if (iconContainer && !iconContainer.hasChildNodes()) {
                            this.renderLottieIcon(iconContainer, App.lottieIconMap['empty_box']);
                        }
                        return;
                    } else {
                        container.classList.remove('hidden');
                        emptyStateContainer.classList.add('hidden');
                    }
                }
            }
            const template = document.getElementById('order-card-template');
            container.innerHTML = '';
            if (!orders || orders.length === 0) {
                this.renderEmptyState(containerId, { iconKey: 'empty_box', title: 'Заказов нет', message: 'В этой категории пока нет заказов.' });
                return;
            }
            orders.forEach(order => {
                const card = template.content.cloneNode(true);
                const orderCard = card.querySelector('.order-card');
                const statusBadge = card.querySelector('.order-status-badge');
                const orderDate = order.Date ? new Date(order.Date).toLocaleDateString('ru-RU', { day: '2-digit', month: 'short' }) : 'не указ.';
                const orderCost = order.Cost?.Valid ? `${order.Cost.Float64.toFixed(0)} ₽` : 'не оценено';
                const statusText = this.getStatusDisplayName(order.Status);
                orderCard.dataset.status = order.Status;
                statusBadge.dataset.status = order.Status;
                card.querySelector('.order-id').textContent = `Заказ №${order.ID}`;
                statusBadge.textContent = statusText;
                const [clientIcon, addressIcon, dateIcon, costIcon] = card.querySelectorAll('.icon-placeholder');
                if (containerId === 'user-orders-list') {
                    clientIcon.className = 'fa-solid fa-box-archive';
                    card.querySelector('.order-client').textContent = this.getCategoryDisplayName(order.Category);
                } else {
                    clientIcon.className = 'fa-solid fa-user';
                    card.querySelector('.order-client').textContent = order.Name || 'Клиент не указан';
                }
                addressIcon.className = 'fa-solid fa-location-dot';
                card.querySelector('.order-address').textContent = order.Address || 'Адрес не указан';
                dateIcon.className = 'fa-solid fa-calendar-days';
                card.querySelector('.order-date').textContent = `${orderDate}, ${order.Time || 'скоро'}`;
                costIcon.className = 'fa-solid fa-coins';
                card.querySelector('.order-cost').textContent = orderCost;
                const typeAnimContainer = card.querySelector('.order-type-anim');
                if (typeAnimContainer) {
                    const lottiePath = lottieOrderIcons[order.Category];
                    if (lottiePath) { this.renderLottieIcon(typeAnimContainer, lottiePath); }
                    else { typeAnimContainer.innerHTML = `<span class="animated-emoji">${{ demolition: "🛠" }[order.Category] || "❓"}</span>`; }
                }
                orderCard.addEventListener('click', () => App.handlers.handleShowOrderDetails(order.ID));
                container.appendChild(card);
            });
        },
        renderStaff(staffList, containerId) {
            const container = document.getElementById(containerId);
            if (!container) return;
            container.innerHTML = '';
            const template = document.getElementById('staff-card-template');
            if (!staffList || staffList.length === 0) {
                this.renderEmptyState(containerId, { iconKey: 'empty_box', title: 'Сотрудники не найдены', message: `В этой группе пока нет сотрудников.` });
                return;
            }
            staffList.forEach(staff => {
                const card = template.content.cloneNode(true);
                const staffCard = card.querySelector('.staff-card');
                staffCard.dataset.chatId = staff.ChatID;
                const staffName = `${staff.FirstName || ''} ${staff.LastName || ''}`.trim();
                card.querySelector('.staff-name').textContent = staffName || `User ID: ${staff.ID}`;
                card.querySelector('.staff-phone').textContent = staff.Phone?.String || 'Телефон не указан';
                container.appendChild(card);
            });
        },
        renderClients(clients) {
            const container = document.getElementById('clients-list');
            const template = document.getElementById('client-card-template');
            container.innerHTML = '';
            if (!clients || clients.length === 0) {
                this.renderEmptyState('clients-list', { iconKey: 'empty_box', title: 'Клиенты не найдены', message: 'Список клиентов пуст.' });
                return;
            }
            clients.forEach(client => {
                const card = template.content.cloneNode(true);
                const clientCard = card.querySelector('.client-card');
                clientCard.dataset.clientId = client.ID;
                const clientName = `${client.FirstName || ''} ${client.LastName || ''}`.trim() || `User ID: ${client.ChatID}`;
                card.querySelector('.client-name').textContent = clientName;
                card.querySelector('.client-info').textContent = `ID в Telegram: ${client.ChatID} | Тел: ${client.Phone?.String || 'не указан'}`;
                clientCard.addEventListener('click', () => App.handlers.handleShowClientDetails(client.ID));
                container.appendChild(card);
            });
        },
        renderClientDetails(detailsData) {
            const contentDiv = document.getElementById('client-details-content');
            if (!contentDiv) return;
            contentDiv.innerHTML = '';
            const clientData = detailsData.User;
            const clientNameElem = document.getElementById('client-detail-name');
            if (clientNameElem) clientNameElem.textContent = `${clientData.FirstName || ''} ${clientData.LastName || ''}`.trim() || `Клиент ID: ${clientData.ID}`;
            contentDiv.innerHTML = `
                <dl class="client-details-grid">
                    <dt>ID в системе</dt><dd>${clientData.ID}</dd>
                    <dt>ID в Telegram</dt><dd>${clientData.ChatID}</dd>
                    <dt>Username</dt><dd>${clientData.Nickname?.String ? `@${clientData.Nickname.String}` : '—'}</dd>
                    <dt>Телефон</dt><dd>${clientData.Phone?.String || '—'}</dd>
                    <dt>Роль</dt><dd>${clientData.Role}</dd>
                    <dt>Всего заказов</dt><dd>${detailsData.order_count ?? 0}</dd>
                    <dt>Регистрация</dt><dd>${new Date(clientData.CreatedAt).toLocaleDateString('ru-RU')}</dd>
                </dl>
            `;
            const blockBtn = document.getElementById('block-client-btn');
            const unblockBtn = document.getElementById('unblock-client-btn');
            if (blockBtn) blockBtn.classList.toggle('hidden', clientData.IsBlocked);
            if (unblockBtn) unblockBtn.classList.toggle('hidden', !clientData.IsBlocked);
        },
        renderClientOrders(orders) {
            const container = document.getElementById('client-orders-list');
            const template = document.getElementById('order-card-template');
            if (!container || !template) return;
            container.innerHTML = '';
            if (!orders || orders.length === 0) {
                this.renderEmptyState('client-orders-list', { iconKey: 'empty_box', title: 'Нет заказов', message: 'Этот клиент пока не сделал заказов.' });
                return;
            }
            orders.slice(0, 5).forEach(order => {
                const card = template.content.cloneNode(true);
                const orderCard = card.querySelector('.order-card');
                const statusBadge = card.querySelector('.order-status-badge');
                const orderDate = order.Date ? new Date(order.Date).toLocaleDateString('ru-RU', { day: '2-digit', month: 'short' }) : 'не указ.';
                const orderCost = order.Cost?.Valid ? `${order.Cost.Float64.toFixed(0)} ₽` : 'не оценено';
                const statusText = App.ui.getStatusDisplayName(order.Status);
                orderCard.dataset.status = order.Status;
                statusBadge.dataset.status = order.Status;
                card.querySelector('.order-id').textContent = `Заказ №${order.ID}`;
                statusBadge.textContent = statusText;
                const [clientIcon, addressIcon, dateIcon, costIcon] = card.querySelectorAll('.icon-placeholder');
                clientIcon.className = 'fa-solid fa-box-archive';
                card.querySelector('.order-client').textContent = App.ui.getCategoryDisplayName(order.Category);
                addressIcon.className = 'fa-solid fa-location-dot';
                card.querySelector('.order-address').textContent = order.Address || 'Адрес не указан';
                dateIcon.className = 'fa-solid fa-calendar-days';
                card.querySelector('.order-date').textContent = `${orderDate}, ${order.Time || 'скоро'}`;
                costIcon.className = 'fa-solid fa-coins';
                card.querySelector('.order-cost').textContent = orderCost;
                orderCard.addEventListener('click', (e) => {
                    e.stopPropagation();
                    App.handlers.handleShowOrderDetails(order.ID);
                });
                container.appendChild(card);
            });
        },
        renderOrderDetails(orderData) {
            const titleElem = document.getElementById('order-detail-title');
            if (titleElem) titleElem.textContent = `Заказ №${orderData.ID}`;
            const statusBadge = document.getElementById('order-detail-status-badge');
            if (statusBadge) {
                statusBadge.textContent = this.getStatusDisplayName(orderData.Status);
                statusBadge.dataset.status = orderData.Status;
            }

            document.getElementById('order-detail-category').textContent = this.getCategoryDisplayName(orderData.Category) || '—';
            document.getElementById('order-detail-subcategory').textContent = this.getSubcategoryDisplayName(orderData.Category, orderData.Subcategory) || '—';
            document.getElementById('order-detail-description').textContent = orderData.Description || '—';
            document.getElementById('order-detail-name').textContent = orderData.Name || '—';
            document.getElementById('order-detail-phone').textContent = orderData.Phone || '—';
            document.getElementById('order-detail-address').textContent = orderData.Address || '—';
            document.getElementById('order-detail-date').textContent = orderData.Date ? new Date(orderData.Date).toLocaleDateString('ru-RU') : '—';
            document.getElementById('order-detail-time').textContent = orderData.Time || '—';
            document.getElementById('order-detail-payment').textContent = orderData.Payment === 'now' ? 'Сразу (со скидкой 5%)' : 'По выполнению';
            document.getElementById('order-detail-cost').textContent = (orderData.Cost?.Valid) ? `${orderData.Cost.Float64.toFixed(0)} ₽` : 'не установлена';

            const editableFields = document.querySelectorAll('#order-details-content dd[data-field]');

            editableFields.forEach(fieldElement => {
                const fieldName = fieldElement.dataset.field;
                const clonedElement = fieldElement.cloneNode(true);
                fieldElement.parentNode.replaceChild(clonedElement, fieldElement);
                const propertyName = fieldName.charAt(0).toUpperCase() + fieldName.slice(1);
                let rawValue = fieldName === 'cost' ? (orderData.Cost?.Valid ? orderData.Cost.Float64.toString() : '') : orderData[propertyName];
                clonedElement.classList.add('editable');
                clonedElement.addEventListener('click', () => App.handlers.handleEditField(fieldName, orderData.ID, rawValue));
            });

            const lottieContainer = document.getElementById('lottie-order-anim');
            lottieContainer.innerHTML = '';
            const lottiePath = lottieOrderIcons[orderData.Category] || lottieOrderIcons["waste_removal"];
            if (lottiePath) { this.renderLottieIcon(lottieContainer, lottiePath); }

            const mediaGalleryContainer = document.getElementById('order-detail-media-gallery');
            const mediaTitle = document.getElementById('order-detail-media-title');
            const addMediaBtn = document.getElementById('add-media-btn');
            if (mediaGalleryContainer) mediaGalleryContainer.innerHTML = '';

            const allMedia = (orderData.Photos || []).map(url => ({type: 'photo', url})).concat(
                (orderData.Videos || []).map(url => ({type: 'video', url}))
            );

            if (addMediaBtn) {
                // === ИСПРАВЛЕНИЕ #1: Логика видимости кнопки ===
                const isOperator = ['operator', 'main_operator', 'owner'].includes(App.state.user?.Role);
                const isOwner = App.state.user?.ChatID === orderData.UserChatID;
                const canAddMedia = isOperator || isOwner;

                addMediaBtn.classList.toggle('hidden', !canAddMedia);
                addMediaBtn.textContent = 'Добавить фото/видео';
                addMediaBtn.onclick = () => App.handlers.triggerAddMediaToOrder();
            }

            if (allMedia.length > 0) {
                mediaTitle?.classList.remove('hidden');
                this.renderMediaGallery(allMedia, 'order-detail-media-gallery');
            } else {
                mediaTitle?.classList.add('hidden');
            }

            const executorsList = document.getElementById('order-detail-executors-list');
            const executorsTitle = document.getElementById('order-detail-executors-title');
            if (executorsList) executorsList.innerHTML = '';

            if (orderData.Executors?.length > 0) {
                executorsTitle?.classList.remove('hidden');
                const templateExecutor = document.getElementById('executor-card-template');
                orderData.Executors.forEach(executor => {
                    const card = templateExecutor.content.cloneNode(true);
                    const firstName = executor.FirstName?.Valid ? executor.FirstName.String : '';
                    const lastName = executor.LastName?.Valid ? executor.LastName.String : '';
                    const nickname = executor.Nickname?.Valid ? `(@${executor.Nickname.String})` : '';
                    const executorName = `${firstName} ${lastName}`.trim() || `ID: ${executor.UserID}`;
                    card.querySelector('.executor-name').textContent = `${executorName} ${nickname}`.trim();
                    card.querySelector('.executor-role').textContent = this.getRoleDisplayName(executor.Role);
                    card.querySelector('.executor-status').textContent = executor.IsNotified ? '🟢 Уведомлен' : '⭕️ Не уведомлен';
                    executorsList?.appendChild(card);
                });
            } else { executorsTitle?.classList.add('hidden'); }

            const actionsContainer = document.querySelector('.order-detail-actions');
            if (actionsContainer) {
                actionsContainer.innerHTML = '';
                this.renderOrderActionButtons(orderData, actionsContainer);
            }
        },

        renderMediaGallery(mediaArray, containerId) {
            const container = document.getElementById(containerId);
            if (!container) return;
            container.innerHTML = '';
            const template = document.getElementById('media-item-template').content.querySelector('.media-item');

            // Данные от бэкенда приходят в верном формате (например, "/api/media/filename.jpg").
            // Этот код просто создает полный URL, объединяя домен и этот путь.
            const fullUrlMediaArray = mediaArray
                .filter(m => m.url && typeof m.url === 'string') // Отсеиваем пустые/некорректные данные
                .map(m => ({
                    type: m.type,
                    // new URL() корректно обработает путь, который начинается со слэша "/"
                    url: new URL(m.url, App.state.apiBaseUrl).href
                }));

            fullUrlMediaArray.forEach((media, index) => {
                const item = template.cloneNode(true);
                const img = item.querySelector('.media-thumbnail[alt="Предпросмотр медиа"]');
                const video = item.querySelector('.media-thumbnail[playsinline]');
                const icon = item.querySelector('.media-type-icon');

                // Используем готовый полный URL для миниатюр
                if (media.type === 'photo') {
                    if (img) {
                        img.src = media.url;
                        img.classList.remove('hidden');
                    }
                    if (video) video.classList.add('hidden');
                    if (icon) icon.className = 'media-type-icon fa-solid fa-image';
                } else if (media.type === 'video') {
                    if (video) {
                        video.src = media.url;
                        video.classList.remove('hidden');
                    }
                    if (img) img.classList.add('hidden');
                    if (icon) icon.className = 'media-type-icon fa-solid fa-video';
                }

                item.addEventListener('click', () => {
                    // Передаем в полноэкранную галерею уже готовый массив
                    App.ui.openFullScreenMedia(fullUrlMediaArray, index);
                });
                container.appendChild(item);
            });
        },




        renderMediaPreviews(formType) {
            let swiper = App.state[`${formType}MediaSwiper`];
            const mediaArray = App.state.selectedMediaFiles[formType];
            const swiperContainer = document.getElementById(`${formType}-media-swiper`);
            if (!swiperContainer) return;

            if (!swiper || swiper.destroyed) {
                swiper = App.ui.initMediaSwiper(formType);
                if (!swiper) return;
            }
            swiper.removeAllSlides();

            const template = document.getElementById('media-item-template');
            if (!template) return;

            mediaArray.forEach((mediaWrapper, index) => {
                const slideFragment = template.content.cloneNode(true);
                const mediaItemDiv = slideFragment.querySelector('.media-item');
                const img = slideFragment.querySelector('img.media-thumbnail');
                const video = slideFragment.querySelector('video.media-thumbnail');

                if (!mediaItemDiv || !img || !video) return;

                const fileUrl = URL.createObjectURL(mediaWrapper.file);
                const fileType = mediaWrapper.file.type.startsWith('image') ? 'photo' : 'video';

                // Управляем видимостью элементов
                if (fileType === 'photo') {
                    img.src = fileUrl;
                    img.style.display = 'block';
                    video.style.display = 'none';
                } else {
                    video.src = fileUrl;
                    video.style.display = 'block';
                    img.style.display = 'none';
                    // Пытаемся запустить беззвучное превью видео
                    video.play().catch(e => console.warn("Autoplay для превью видео был заблокирован браузером."));
                }

                const deleteButton = document.createElement('button');
                deleteButton.className = 'delete-media-button';
                deleteButton.innerHTML = '&times;';
                deleteButton.title = 'Удалить файл';
                deleteButton.addEventListener('click', (e) => {
                    e.stopPropagation();
                    App.handlers.handleDeleteMedia(mediaWrapper.id, formType);
                });
                mediaItemDiv.appendChild(deleteButton);

                mediaItemDiv.addEventListener('click', () => {
                    const galleryItems = mediaArray.map(item => ({
                        type: item.file.type.startsWith('image') ? 'photo' : 'video',
                        url: URL.createObjectURL(item.file)
                    }));
                    App.ui.openFullScreenMedia(galleryItems, index);
                });

                swiper.appendSlide(slideFragment);
            });

            swiperContainer.classList.toggle('swiper-initialized', mediaArray.length > 0);
            const paginationEl = swiper.pagination.el;
            if (paginationEl) {
                paginationEl.style.display = mediaArray.length > 1 ? 'block' : 'none';
            }
            swiper.update();
        },

        initFullscreenSwiper(mediaItems, initialIndex) {
            if (App.state.fullscreenSwiper) {
                App.state.fullscreenSwiper.destroy(true, true);
                App.state.fullscreenSwiper = null;
            }

            const wrapper = document.getElementById('fullscreen-swiper-wrapper');
            if (!wrapper) return;
            wrapper.innerHTML = '';

            const template = document.getElementById('fullscreen-slide-template');
            if (!template) return;

            mediaItems.forEach(item => {
                const slide = template.content.cloneNode(true);
                const img = slide.querySelector('img');
                const video = slide.querySelector('video');

                if (item.type === 'photo') {
                    img.src = item.url;
                    img.style.display = 'block';
                    video.style.display = 'none';
                    // Для фото отключаем увеличение по двойному тапу, чтобы не мешало
                    slide.querySelector('.swiper-zoom-container').addEventListener('dblclick', (e) => e.stopPropagation());
                } else if (item.type === 'video') {
                    video.src = item.url;
                    video.style.display = 'block';
                    img.style.display = 'none';
                }
                wrapper.appendChild(slide);
            });

            App.state.fullscreenSwiper = new Swiper('#fullscreen-swiper', {
                initialSlide: initialIndex,
                slidesPerView: 1,
                spaceBetween: 20,
                centeredSlides: true,
                zoom: {
                    maxRatio: 3,
                    minRatio: 1,
                    toggle: true, // Включаем зум по двойному клику/тапу
                },
                pagination: {
                    el: '.fullscreen-pagination',
                    type: 'fraction',
                },
                keyboard: { enabled: true },
                navigation: false,
                scrollbar: false,
                on: {
                    slideChange: function () {
                        // При переключении слайда останавливаем все видео
                        wrapper.querySelectorAll('video').forEach(v => {
                            if (!v.paused) {
                                v.pause();
                            }
                        });
                    },
                    zoomChange: function(swiper, scale) {
                        // Когда пользователь увеличивает фото, отключаем возможность свайпать слайдер
                        swiper.allowTouchMove = (scale === 1);
                    }
                }
            });
        },

        openFullScreenMedia(mediaArray, clickedIndex) {
            const overlay = document.getElementById('fullscreen-media-overlay');
            if (!overlay) return;

            // Инициализируем Swiper со всеми медиафайлами
            App.ui.initFullscreenSwiper(mediaArray, clickedIndex);

            overlay.classList.remove('hidden');
            document.body.style.overflow = 'hidden';
            const swiperContainer = document.getElementById('fullscreen-swiper');

            // Уничтожаем старый обработчик Hammer, если он был
            if (swiperContainer.hammer) {
                swiperContainer.hammer.destroy();
            }

            const hammer = new Hammer(swiperContainer);
            hammer.get('pan').set({ direction: Hammer.DIRECTION_VERTICAL, threshold: 0 });

            hammer.on('panstart', () => {
                if (App.state.fullscreenSwiper && App.state.fullscreenSwiper.zoom.scale !== 1) {
                    hammer.stop(true);
                    return;
                }
                swiperContainer.style.transition = 'none';
            });

            hammer.on('panmove', (ev) => {
                // Блокируем свайп для закрытия, если пользователь управляет видео (например, скроллит по таймлайну)
                const isVideoControl = ev.target.nodeName === 'VIDEO' || ev.target.closest('video');
                if (isVideoControl && ev.pointerType !== 'touch') return;

                if (App.state.fullscreenSwiper && App.state.fullscreenSwiper.zoom.scale !== 1) return;

                if (ev.deltaY > 0) {
                    swiperContainer.style.transform = `translateY(${ev.deltaY}px)`;
                    const opacity = 1 - (ev.deltaY / (window.innerHeight / 1.5));
                    overlay.style.backgroundColor = `rgba(0, 0, 0, ${Math.max(0.1, opacity)})`;
                }
            });

            hammer.on('panend', (ev) => {
                if (App.state.fullscreenSwiper && App.state.fullscreenSwiper.zoom.scale !== 1) return;
                swiperContainer.style.transition = 'transform 0.3s ease-out';
                const threshold = window.innerHeight * 0.25;
                if (ev.deltaY > threshold) {
                    App.ui.closeFullScreenMedia();
                } else {
                    swiperContainer.style.transform = 'translateY(0px)';
                    overlay.style.backgroundColor = 'rgba(0, 0, 0, 0.9)';
                }
            });
            swiperContainer.hammer = hammer;
        },


        closeFullScreenMedia() {
            const overlay = document.getElementById('fullscreen-media-overlay');
            if (!overlay || overlay.classList.contains('hidden')) return;

            overlay.classList.add('is-closing');

            const onTransitionEnd = () => {
                overlay.classList.add('hidden');
                overlay.classList.remove('is-closing');
                overlay.style.backgroundColor = '';

                // Останавливаем и очищаем все видео
                overlay.querySelectorAll('video').forEach(v => {
                    v.pause();
                    v.src = "";
                    v.removeAttribute('src');
                });

                const swiperContainer = document.getElementById('fullscreen-swiper');
                if (swiperContainer) { swiperContainer.style.transform = ''; }
                if (App.state.fullscreenSwiper) {
                    App.state.fullscreenSwiper.destroy(true, true);
                    App.state.fullscreenSwiper = null;
                }
                if (swiperContainer && swiperContainer.hammer) {
                    swiperContainer.hammer.destroy();
                    swiperContainer.hammer = null;
                }
                document.body.style.overflow = '';
                overlay.removeEventListener('transitionend', onTransitionEnd);
            };

            overlay.addEventListener('transitionend', onTransitionEnd, { once: true });
            // Запасной вариант, если событие transitionend не сработает
            setTimeout(() => {
                if (!overlay.classList.contains('hidden')) {
                    onTransitionEnd();
                }
            }, 350);
        },

        populateClientSelectForm() {
            const select = document.getElementById('client-select');
            if (!select) return;
            select.innerHTML = '<option value="" disabled selected>Выберите из списка...</option>';
            App.state.clients.forEach(client => {
                const option = document.createElement('option');
                option.value = client.ID;
                option.textContent = `${client.FirstName || ''} ${client.LastName || ''} (ID: ${client.ChatID})`.trim();
                option.dataset.chatId = client.ChatID;
                option.dataset.phone = client.Phone?.String || '';
                option.dataset.name = client.FirstName || '';
                option.dataset.isBlocked = client.IsBlocked ? 'true' : 'false';
                select.appendChild(option);
            });
        },
        updateSubcategoriesForm(categorySelectId, subcategorySelectId) {
            const category = document.getElementById(categorySelectId)?.value;
            const subcategorySelect = document.getElementById(subcategorySelectId);
            if (!subcategorySelect) return;
            subcategorySelect.innerHTML = '<option value="" disabled selected></option>';
            const subcategories = App.categoriesConfig[category]?.subcategories || [];
            subcategories.forEach(sub => {
                const option = document.createElement('option');
                option.value = sub.key;
                option.textContent = sub.label;
                subcategorySelect.appendChild(option);
            });
            if (subcategorySelect.options.length > 0) { subcategorySelect.removeAttribute('disabled'); }
            else { subcategorySelect.setAttribute('disabled', 'true'); }
        },
        setupUserOrderForm() {
            const form = document.getElementById('user-create-order-form');
            if (!form) return;
            form.reset();
            if (App.state.user) {
                form.querySelector('#user-order-name').value = App.state.user.FirstName || '';
                form.querySelector('#user-order-phone').value = App.state.user.Phone?.String || '';
            }
            this.updateSubcategoriesForm('user-category-select', 'user-subcategory-select');
        },

        filterList(query, dataList, containerId, renderFunction) {
            const lowerCaseQuery = query.toLowerCase().trim();
            if (!lowerCaseQuery) { renderFunction(dataList, containerId); return; }
            const filteredData = dataList.filter(item => {
                const name = `${item.FirstName || item.Name || ''} ${item.LastName || ''}`.toLowerCase();
                const phone = item.Phone?.String?.toLowerCase() || '';
                const id = item.ID?.toString() || '';
                const chatId = item.ChatID?.toString() || '';
                const address = item.Address?.toLowerCase() || '';
                const category = item.Category?.toLowerCase() || '';
                return name.includes(lowerCaseQuery) || phone.includes(lowerCaseQuery) || id.includes(lowerCaseQuery) ||
                    chatId.includes(lowerCaseQuery) || address.includes(lowerCaseQuery) || category.includes(lowerCaseQuery);
            });
            renderFunction(filteredData, containerId);
        },
        getFieldDisplayName(fieldName) { return App.displayNamesMap.fields[fieldName] || fieldName; },
        getCategoryDisplayName(key) { return App.categoriesConfig[key]?.displayName || key; },
        getSubcategoryDisplayName(category, subcategory) {
            const cat = App.categoriesConfig[category];
            if (!cat || !cat.subcategories) return subcategory || '—';
            const sub = cat.subcategories.find(s => s.key === subcategory);
            return sub ? sub.label : subcategory || '—';
        },
        getStatusDisplayName(key) { return App.displayNamesMap.statuses[key] || key; },
        getRoleDisplayName(key) { return App.displayNamesMap.roles[key] || key; },

        renderOrderActionButtons(orderData, container) {
            const userRole = App.state.user?.Role;
            const isOperatorOrHigher = ['operator', 'main_operator', 'owner'].includes(userRole);
            const isAssignedDriver = orderData.Executors?.some(exec => exec.UserID === App.state.user?.ID && exec.Role === 'driver');
            const isCurrentUserOrder = App.state.user?.ChatID === orderData.UserChatID;

            let buttons = [];
            const createButton = (text, action, className = 'button-primary', orderId = orderData.ID) => ({ text, action, className, orderId });

            if (isOperatorOrHigher) {
                switch (orderData.Status) {
                    case 'new': case 'awaiting_cost':
                        buttons.push(createButton('💰 Установить стоимость', 'set_cost', 'button-warning'));
                        buttons.push(createButton('❌ Отменить заказ', 'cancel', 'button-danger'));
                        break;
                    case 'awaiting_confirmation':
                        buttons.push(createButton('✏️ Изменить стоимость', 'set_cost', 'button-primary'));
                        buttons.push(createButton('❌ Отменить заказ', 'cancel', 'button-danger'));
                        break;
                    case 'in_progress':
                        if (!orderData.Cost?.Valid || orderData.Cost.Float64 === 0) {
                            buttons.push(createButton('💰 Установить стоимость', 'set_cost', 'button-warning'));
                        }
                        buttons.push(createButton('✅ Заказ выполнен', 'complete', 'button-success'));
                        buttons.push(createButton('❌ Отменить заказ', 'cancel', 'button-danger'));
                        break;
                    case 'completed':
                        buttons.push(createButton('💲 Изменить итог. стоимость', 'set_final_cost', 'button-primary'));
                        break;
                    case 'canceled':
                        buttons.push(createButton('🔄 Возобновить', 'resume', 'button-success'));
                        break;
                }
                if (orderData.Status !== 'canceled') {
                    buttons.push(createButton('👷 Исполнители', 'edit_executors', 'button-primary'));
                    buttons.push(createButton('🚫 Блокировать клиента', 'block_user', 'button-danger'));
                }
            }
            if (isCurrentUserOrder && userRole === 'user') {
                switch (orderData.Status) {
                    case 'awaiting_confirmation':
                        buttons.push(createButton(`✅ Да, согласен (${orderData.Cost.Float64.toFixed(0)} ₽)`, 'accept_cost', 'button-success'));
                        buttons.push(createButton('❌ Отказаться от стоимости', 'reject_cost', 'button-danger'));
                        break;
                    case 'awaiting_payment':
                        buttons.push(createButton('💳 Оплатить заказ', 'pay_order', 'button-success'));
                        break;
                    case 'draft': case 'new': case 'awaiting_cost':
                        buttons.push(createButton('❌ Отменить мой заказ', 'cancel_by_user', 'button-danger'));
                        break;
                }
            }
            if (isAssignedDriver) {
                if (orderData.Status === 'in_progress') {
                    buttons.push(createButton('✅ Заказ выполнен', 'complete_by_driver', 'button-success'));
                }
                if (orderData.Status === 'completed') {
                    buttons.push(createButton('💲 Изменить итог. стоимость', 'set_final_cost', 'button-primary'));
                }
            }
            buttons.forEach(btn => {
                const buttonElement = document.createElement('button');
                buttonElement.textContent = btn.text;
                buttonElement.className = btn.className;
                buttonElement.dataset.action = btn.action;
                buttonElement.dataset.orderId = btn.orderId;
                container?.appendChild(buttonElement);
            });
        },

        showSelectPrompt(title, options, currentValue, onSave) {
            const overlay = document.createElement('div');
            overlay.className = 'modal-overlay visible';
            overlay.style.alignItems = 'center';
            overlay.style.zIndex = '2100';

            const modal = document.createElement('div');
            modal.className = 'modal-window';
            modal.style.borderRadius = 'var(--radius-l)';
            modal.style.maxWidth = '400px';
            modal.style.transform = 'translateY(0)';
            modal.style.position = 'relative';
            modal.innerHTML = `
                <button class="close-modal-btn">&times;</button>
                <h2 style="font-size: 1.3em; margin-bottom: 20px;">${title}</h2>
                <div class="form-group">
                    <select id="dynamic-edit-select" class="dynamic-edit-select" style="padding: 14px 16px;">
                        ${options.map(opt => `<option value="${opt.key}" ${opt.key === currentValue ? 'selected' : ''}>${opt.label}</option>`).join('')}
                    </select>
                </div>
                <div class="form-actions" style="position: static; padding: 20px 0 0 0; background: none; border: none;">
                     <button class="button-primary save-btn">Сохранить</button>
                </div>
            `;

            const closeModal = () => overlay.remove();
            modal.querySelector('.close-modal-btn').onclick = closeModal;
            overlay.onclick = (e) => { if (e.target === overlay) closeModal(); };

            modal.querySelector('.save-btn').onclick = () => {
                const select = modal.querySelector('#dynamic-edit-select');
                onSave(select.value);
                closeModal();
            };

            overlay.appendChild(modal);
            document.body.appendChild(overlay);
        }
    },

    /**
     * @section Обработчики событий
     */
    handlers: {
        sendTgCallback(callbackData) { App.state.tg.sendData(callbackData); },

        handleFormOpen(formType) {
            App.state.selectedMediaFiles[formType] = [];
            const oldSwiper = App.state[`${formType}MediaSwiper`];
            if (oldSwiper && !oldSwiper.destroyed) { oldSwiper.destroy(true, true); }
            App.state[`${formType}MediaSwiper`] = null;
            const swiperWrapper = document.querySelector(`#${formType}-media-swiper .swiper-wrapper`);
            if (swiperWrapper) { swiperWrapper.innerHTML = ''; }
            const swiperContainer = document.getElementById(`${formType}-media-swiper`);
            if (swiperContainer) { swiperContainer.classList.remove('swiper-initialized'); }
            const paginationEl = document.querySelector(`#${formType}-media-swiper .swiper-pagination`);
            if (paginationEl) { paginationEl.style.display = 'none'; }
        },
        resetAndDestroyMediaState(formType) {
            const swiper = App.state[`${formType}MediaSwiper`];
            if (swiper && !swiper.destroyed) { swiper.destroy(true, true); }
            App.state[`${formType}MediaSwiper`] = null;
            App.state.selectedMediaFiles[formType].forEach(item => URL.revokeObjectURL(URL.createObjectURL(item.file)));
            App.state.selectedMediaFiles[formType] = [];
            const swiperContainer = document.getElementById(`${formType}-media-swiper`);
            if (swiperContainer) {
                swiperContainer.querySelector('.swiper-wrapper').innerHTML = '';
                swiperContainer.querySelector('.swiper-pagination').innerHTML = '';
                swiperContainer.classList.remove('swiper-initialized');
            }
        },

        triggerFileInput(formType, captureMode = null) {
            const fileInput = document.createElement('input');
            fileInput.type = 'file';
            fileInput.multiple = true;
            fileInput.accept = 'image/*,video/*';
            if (captureMode) {
                fileInput.capture = captureMode;
                fileInput.multiple = false;
                fileInput.accept = 'image/*';
            }
            fileInput.style.position = 'absolute';
            fileInput.style.left = '-9999px';
            fileInput.style.top = '-9999px';
            fileInput.style.opacity = '0';
            fileInput.addEventListener('change', (event) => {
                this.handleFileSelection(event, formType);
                document.body.removeChild(fileInput);
            }, { once: true });
            document.body.appendChild(fileInput);
            fileInput.click();
        },
        handleFileSelection(event, formType) {
            if (!formType || !event.target.files || !event.target.files.length) return;
            const mediaState = App.state.selectedMediaFiles[formType];
            for (const file of event.target.files) {
                if (!file.type.startsWith('image/') && !file.type.startsWith('video/')) { continue; }
                mediaState.push({ id: Date.now() + Math.random(), file: file });
            }
            App.ui.renderMediaPreviews(formType);
        },
        handleDeleteMedia(id, formType) {
            const mediaState = App.state.selectedMediaFiles[formType];
            const index = mediaState.findIndex(item => item.id === id);
            if (index > -1) {
                URL.revokeObjectURL(URL.createObjectURL(mediaState[index].file));
                mediaState.splice(index, 1);
                App.ui.renderMediaPreviews(formType);
            }
        },

        handleFetchOrders(statusKey = 'active') {
            const containerId = `orders-list-${statusKey}`;
            App.ui.renderSkeleton(containerId, 5);
            App.api.fetchOrders(statusKey)
                .then(orders => {
                    App.state.orders[statusKey] = orders || [];
                    App.state.loadedStatuses.add(statusKey);
                    App.ui.renderOrders(App.state.orders[statusKey], containerId);
                })
                .catch(error => App.ui.renderEmptyState(containerId, { iconKey: 'error_sign', title: 'Ошибка загрузки', message: error.message }));
        },

        async _uploadMediaFiles(filesToUpload) {
            if (!filesToUpload || filesToUpload.length === 0) {
                return { photos: [], videos: [] };
            }

            const uploadUrl = '/api/upload-media'; // Относительный путь

            const uploadPromises = filesToUpload.map(mediaWrapper => {
                const formData = new FormData();
                formData.append('media', mediaWrapper.file);

                return App.api._fetch(uploadUrl, {
                    method: 'POST',
                    body: formData
                }).catch(error => {
                    error.fileName = mediaWrapper.file.name;
                    throw error;
                });
            });

            const results = await Promise.all(uploadPromises);

            const uploadedMedia = { photos: [], videos: [] };
            results.forEach(result => {
                if (result.data.type === 'photo') {
                    uploadedMedia.photos.push(result.data.file_id);
                } else if (result.data.type === 'video') {
                    uploadedMedia.videos.push(result.data.file_id);
                }
            });

            return uploadedMedia;
        },

        async handleOperatorCreateOrderSubmit(event) {
            event.preventDefault();
            const form = event.target;
            const tg = App.state.tg;

            const submitButton = form.querySelector('button[type="submit"]');
            submitButton.classList.add('button-disabled');
            submitButton.textContent = 'Загрузка файлов...';

            try {
                // Проверяем на блокировку, только если заказ создается для существующего клиента
                if (App.state.clientForNewOrder && App.state.clientForNewOrder.IsBlocked) {
                    throw new Error('Вы не можете создать заказ для заблокированного клиента.');
                }

                const filesToUpload = App.state.selectedMediaFiles['operator'];
                const uploadedMedia = await this._uploadMediaFiles(filesToUpload);

                submitButton.textContent = 'Создание заказа...';

                const costInput = form.elements['cost'].value;
                const costValue = parseFloat(costInput);
                const costPayload = {
                    Float64: !isNaN(costValue) ? costValue : 0,
                    Valid: !isNaN(costValue) && costInput.trim() !== ''
                };

                const orderPayload = {
                    // Если клиент был передан со страницы деталей - используем его ID, иначе отправляем 0
                    UserID: App.state.clientForNewOrder ? App.state.clientForNewOrder.ID : 0,
                    UserChatID: App.state.clientForNewOrder ? App.state.clientForNewOrder.ChatID : 0,
                    Name: form.elements['name'].value,
                    Phone: form.elements['phone'].value,
                    Category: form.elements['category'].value,
                    Subcategory: form.elements['subcategory'].value,
                    Address: form.elements['address'].value,
                    Date: form.elements['date'].value,
                    Time: form.elements['time'].value,
                    Description: form.elements['description'].value,
                    Status: 'new',
                    Payment: 'later',
                    Cost: costPayload,
                    Photos: uploadedMedia.photos,
                    Videos: uploadedMedia.videos
                };

                const data = await App.api.createOrderForOperator(orderPayload);

                tg.showPopup({ title: 'Успех!', message: `Заказ №${data.order_id} успешно создан.`, buttons: [{ type: 'ok', text: 'Отлично' }] });
                form.reset();
                App.ui.showPanel('orders-panel', 'backward');
                this.resetAndDestroyMediaState('operator');
                App.state.loadedStatuses.clear();
                App.state.orders = {};
                this.handleFetchOrders('active');
                App.state.contentSwiper?.slideTo(0);

            } catch (error) {
                let errorMessage = error.fileName ? `Ошибка загрузки файла ${error.fileName}.` : error.message;
                tg.showAlert(`Ошибка: ${errorMessage}`);
            } finally {
                // Очищаем временное состояние клиента после попытки отправки
                App.state.clientForNewOrder = null;
                submitButton.classList.remove('button-disabled');
                submitButton.textContent = 'Создать заказ';
            }
        },

        handleShowOrderDetails(orderId) {
            App.ui.showPanel('order-detail-panel', 'forward');
            document.getElementById('order-detail-panel').scrollTop = 0;
            const backButton = document.querySelector('#order-detail-panel .back-button');
            if (backButton) { backButton.dataset.targetPanel = App.state.user.Role === 'user' ? 'user-panel' : 'orders-panel'; }
            document.getElementById('order-detail-executors-list').innerHTML = '';
            document.getElementById('order-detail-media-gallery').innerHTML = '';
            document.getElementById('order-detail-executors-title')?.classList.add('hidden');
            document.getElementById('order-detail-media-title')?.classList.add('hidden');
            document.getElementById('add-media-btn')?.classList.add('hidden');
            const userRole = App.state.user?.Role;
            let apiCall;
            if (['operator', 'main_operator', 'owner', 'driver'].includes(userRole)) { apiCall = App.api.fetchOrderDetails(orderId); }
            else if (userRole === 'user') { apiCall = App.api.fetchUserOrderDetails(orderId); }
            else { App.ui.showError('Неизвестная роль пользователя.'); return; }
            apiCall.then(orderData => {
                App.state.selectedOrder = orderData;
                App.ui.renderOrderDetails(orderData);
            }).catch(error => {
                App.ui.showError(`Не удалось загрузить детали заказа: ${error.message}`);
                document.getElementById('order-details-content').innerHTML = `<p style="color: red;">${error.message}</p>`;
            });
        },

        async handleEditField(fieldName, orderId, currentValue) {
            // Проверяем права доступа
            const isOperatorOrHigher = ['operator', 'main_operator', 'owner'].includes(App.state.user?.Role);
            if (!isOperatorOrHigher) {
                App.state.tg.showAlert('Редактирование полей доступно только оператору.');
                return;
            }

            const title = `Изменить "${App.ui.getFieldDisplayName(fieldName)}"`;

            // Функция-обертка для вызова API и обновления UI
            const processUpdate = async (valueToSave) => {
                // Если пользователь нажал "отмена" в промпте или не изменил значение
                if (valueToSave === null || valueToSave === currentValue) return;

                try {
                    // Вызываем новый метод API для обновления поля
                    await App.api.updateOrderField(orderId, { field: fieldName, value: valueToSave });
                    App.state.tg.showPopup({ title: "Успех!", message: "Поле успешно обновлено." });
                    // Обновляем детали заказа, чтобы показать новое значение
                    this.handleShowOrderDetails(orderId);
                } catch (error) {
                    App.state.tg.showAlert(`Ошибка обновления: ${error.message}`);
                }
            };

            switch (fieldName) {
                case 'description':
                case 'name':
                case 'phone':
                case 'address':
                    // ИЗМЕНЕНИЕ: Используем стандартный prompt вместо tg.showPrompt
                    const newValue = prompt(title, currentValue || '');
                    processUpdate(newValue); // processUpdate уже содержит проверку на null
                    break;
                case 'date':
                    // ИЗМЕНЕНИЕ: Используем стандартный prompt и добавляем валидацию
                    const newDate = prompt(`${title}\n(в формате ГГГГ-ММ-ДД)`, currentValue || new Date().toISOString().split('T')[0]);
                    if (newDate !== null) { // Если пользователь не нажал "Отмена"
                        if (/^\d{4}-\d{2}-\d{2}$/.test(newDate)) {
                            processUpdate(newDate);
                        } else {
                            App.state.tg.showAlert('Неверный формат даты. Используйте ГГГГ-ММ-ДД.');
                        }
                    }
                    break;
                case 'time':
                    // ИЗМЕНЕНИЕ: Используем стандартный prompt и добавляем валидацию
                    const newTime = prompt(`${title}\n(в формате ЧЧ:ММ)`, currentValue || '10:00');
                    if (newTime !== null) { // Если пользователь не нажал "Отмена"
                        if (/^\d{2}:\d{2}$/.test(newTime)) {
                            processUpdate(newTime);
                        } else {
                            App.state.tg.showAlert('Неверный формат времени. Используйте ЧЧ:ММ.');
                        }
                    }
                    break;
                case 'category': {
                    const options = Object.keys(App.categoriesConfig).map(key => ({
                        key: key,
                        label: App.categoriesConfig[key].displayName
                    }));
                    App.ui.showSelectPrompt(title, options, currentValue, processUpdate);
                    break;
                }
                case 'subcategory': {
                    const currentCategory = App.state.selectedOrder?.Category;
                    if (!currentCategory || !App.categoriesConfig[currentCategory]?.subcategories) {
                        App.state.tg.showAlert('Сначала выберите категорию заказа.');
                        return;
                    }
                    const options = App.categoriesConfig[currentCategory].subcategories;
                    App.ui.showSelectPrompt(title, options, currentValue, processUpdate);
                    break;
                }
                case 'edit_executors':
                case 'block_user':
                case 'media':
                case 'pay_order':
                    App.state.tg.showConfirm(`Это действие перенаправит вас в бот. Продолжить?`, (confirmed) => {
                        if (confirmed) {
                            // Формируем callback_data для отправки в бот
                            this.sendTgCallback(`edit_field_${fieldName}_${orderId}`);
                            App.state.tg.close();
                        }
                    });
                    break;
                default:
                    App.state.tg.showAlert(`Редактирование поля "${fieldName}" в приложении пока не поддерживается.`);
                    break;
            }
        },

        handleOrderActionClick(event) {
            const button = event.target.closest('button');
            if (!button || !button.dataset.action) return;

            const action = button.dataset.action;
            const orderId = button.dataset.orderId;
            const userRole = App.state.user.Role;
            let payload = { action: action };

            const performApiCall = (apiPromise) => {
                // В новой версии здесь был showPopup, но showAlert тоже будет работать
                apiPromise.then(response => {
                    App.state.tg.showAlert(response.message || 'Действие выполнено успешно!');
                    this.handleShowOrderDetails(orderId);
                }).catch(error => {
                    App.state.tg.showAlert(`Ошибка: ${error.message}`);
                });
            };

            let apiCallFunction;
            if (action === 'complete_by_driver') {
                payload.action = 'complete';
                apiCallFunction = (p) => App.api.driverOrderAction(orderId, p);
            } else if (['operator', 'main_operator', 'owner'].includes(userRole)) {
                apiCallFunction = (p) => App.api.adminOrderAction(orderId, p);
            } else {
                // В старой версии здесь не было userOrderAction, но лучше его оставить, если он используется
                apiCallFunction = (p) => App.api.userOrderAction(orderId, p);
            }

            if (action === 'set_cost' || action === 'set_final_cost') {
                // === ИСПОЛЬЗУЕМ СТАРЫЙ РАБОЧИЙ `prompt` ===
                const currentCost = App.state.selectedOrder?.Cost?.Float64 || 0;
                const newCostInput = prompt('Введите новую стоимость:', currentCost.toFixed(0));

                if (newCostInput === null) return; // Пользователь нажал "Отмена"

                const newCostValue = parseFloat(newCostInput);
                // Проверка, что значение не является NaN, кроме случая, когда строка пустая (для сброса)
                if (isNaN(newCostValue) && newCostInput.trim() !== '') {
                    return App.state.tg.showAlert('Введено некорректное значение.');
                }

                payload.cost = newCostValue;
                performApiCall(apiCallFunction(payload));

            } else if (action === 'cancel' || action === 'reject_cost' || action === 'cancel_by_user') {
                // === ИСПОЛЬЗУЕМ СТАРЫЙ РАБОЧИЙ `prompt` ===
                const reason = prompt('Укажите причину:');
                if (reason === null) return; // Пользователь нажал "Отмена"
                if (reason.trim() === '') {
                    return App.state.tg.showAlert('Причина отмены не может быть пустой.');
                }
                payload.reason = reason;
                performApiCall(apiCallFunction(payload));

            } else if (action === 'edit_executors' || action === 'block_user' || action === 'pay_order') {
                // Это делегирование другим обработчикам, оно корректно
                this.handleEditField(action, orderId, null);
            } else {
                // Для действий без ввода (например, "Возобновить" или "Заказ выполнен")
                performApiCall(apiCallFunction(payload));
            }
        },

        handleFetchClients() {
            if (App.state.clients.length > 0) { App.ui.renderClients(App.state.clients); App.ui.populateClientSelectForm(); return; }
            App.ui.renderSkeleton('clients-list', 8);
            App.api.fetchClients()
                .then(data => {
                    const clientList = Array.isArray(data) ? data : (data.clients || []);
                    App.state.clients = clientList;
                    App.ui.renderClients(App.state.clients);
                    App.ui.populateClientSelectForm();
                })
                .catch(error => App.ui.renderEmptyState('clients-list', { iconKey: 'error_sign', title: 'Ошибка загрузки', message: error.message }));
        },
        handleShowClientDetails(clientId) {
            App.ui.showPanel('client-detail-panel', 'forward');
            document.getElementById('client-detail-panel').scrollTop = 0;
            App.ui.renderSkeleton('client-details-content', 1);
            App.ui.renderSkeleton('client-orders-list', 3);
            App.api.fetchClientDetails(clientId)
                .then(data => {
                    App.state.selectedClient = data.User;
                    App.ui.renderClientDetails(data);
                    App.ui.renderClientOrders(data.Orders || []);
                })
                .catch(error => {
                    document.getElementById('client-details-content').innerHTML = `<p style="color: red;">${error.message}</p>`;
                    App.ui.renderEmptyState('client-orders-list', { iconKey: 'empty_box', title: 'Нет заказов', message: 'Не удалось загрузить заказы клиента.' });
                });
        },
        handleCreateOrderForClient() {
            if (!App.state.selectedClient) { App.ui.showError('Выберите клиента.'); return; }
            if (App.state.selectedClient.IsBlocked) { App.state.tg.showAlert('Вы не можете создать заказ для заблокированного клиента.'); return; }
            App.ui.showPanel('order-creation-panel', 'forward');
            document.getElementById('client-select').value = App.state.selectedClient.ID;
            document.getElementById('order-name').value = App.state.selectedClient.FirstName || '';
            document.getElementById('order-phone').value = App.state.selectedClient.Phone?.String || '';
            App.ui.updateSubcategoriesForm('category-select', 'subcategory-select');
            this.handleFormOpen('operator');
        },
        handleViewClientChats() { if (!App.state.selectedClient) { App.ui.showError('Выберите клиента.'); return; } this.sendTgCallback(`view_chat_history_${App.state.selectedClient.ChatID}`); App.state.tg.close(); },
        handleBlockClient() { if (!App.state.selectedClient) { App.ui.showError('Выберите клиента.'); return; } App.state.tg.showConfirm(`Вы уверены?`, (c) => { if (c) { this.sendTgCallback(`block_user_reason_prompt_${App.state.selectedClient.ChatID}`); App.state.tg.close(); } }); },
        handleUnblockClient() { if (!App.state.selectedClient) { App.ui.showError('Выберите клиента.'); return; } App.state.tg.showConfirm(`Вы уверены?`, (c) => { if (c) { this.sendTgCallback(`unblock_user_final_${App.state.selectedClient.ChatID}`); App.state.tg.close(); } }); },
        fillClientData() {
            const select = document.getElementById('client-select');
            const selectedOption = select?.options[select.selectedIndex];
            if (!selectedOption || !selectedOption.value) return;
            document.getElementById('order-name').value = selectedOption.dataset.name || '';
            document.getElementById('order-phone').value = selectedOption.dataset.phone || '';
        },

        handleShowStaffHub() {},
        handleRoleClick(event) {
            const button = event.currentTarget;
            const role = button.dataset.role;
            App.state.currentStaffRole = role;
            document.getElementById('staff-list-title').textContent = App.ui.getRoleDisplayName(role);
            App.ui.showPanel('staff-list-panel', 'forward');
            this.handleFetchStaff(role);
        },
        handleFetchStaff(role) {
            const containerId = 'staff-list';
            if (App.state.staff[role]) { App.ui.renderStaff(App.state.staff[role], containerId); return; }
            App.ui.renderSkeleton(containerId, 8);
            App.api.fetchStaff(role)
                .then(data => {
                    const staffList = Array.isArray(data) ? data : (data.clients || []);
                    App.state.staff[role] = staffList;
                    App.ui.renderStaff(App.state.staff[role], containerId);
                })
                .catch(error => App.ui.renderEmptyState(containerId, { iconKey: 'error_sign', title: 'Ошибка загрузки', message: error.message }));
        },
        handleCreateStaffSubmit(event) {
            event.preventDefault();
            const form = event.target;
            const formData = new FormData(form);
            const staffPayload = { FirstName: formData.get('FirstName'), LastName: formData.get('LastName'), Phone: formData.get('Phone'), CardNumber: formData.get('CardNumber'), Role: App.state.currentStaffRole };
            if (!staffPayload.FirstName || !staffPayload.LastName || !staffPayload.Phone) { App.state.tg.showAlert('Заполните Имя, Фамилию и Телефон.'); return; }
            App.api.addStaff(staffPayload)
                .then(() => {
                    App.state.tg.showPopup({ title: 'Успех!', message: `Сотрудник ${staffPayload.FirstName} добавлен.` });
                    form.reset();
                    App.ui.hideAddStaffModal();
                    delete App.state.staff[App.state.currentStaffRole];
                    this.handleFetchStaff(App.state.currentStaffRole);
                })
                .catch(error => App.state.tg.showAlert(`Ошибка: ${error.message}`));
        },

        handleFetchUserOrders() {
            const containerId = 'user-orders-list';
            App.ui.renderSkeleton(containerId, 3);
            App.api.fetchUserOrders()
                .then(orders => {
                    App.state.userOrders = orders || [];
                    App.ui.renderOrders(App.state.userOrders, containerId);
                })
                .catch(error => App.ui.renderEmptyState(containerId, { iconKey: 'error_sign', title: 'Не удалось загрузить заказы', message: error.message, showCreateButton: true, createButtonId: 'create-order-from-empty-state' }));
        },
        async handleUserCreateOrderSubmit(event) {
            event.preventDefault();
            const form = event.target;
            const tg = App.state.tg;

            const submitButton = form.querySelector('button[type="submit"]');
            submitButton.classList.add('button-disabled');
            submitButton.textContent = 'Загрузка файлов...';

            try {
                const requiredFields = ['name', 'phone', 'address', 'date', 'time', 'category'];
                for (const fieldName of requiredFields) {
                    if (!form.elements[fieldName].value) {
                        throw new Error(`Пожалуйста, заполните поле "${App.ui.getFieldDisplayName(fieldName)}".`);
                    }
                }

                const filesToUpload = App.state.selectedMediaFiles['user'];
                const uploadedMedia = await this._uploadMediaFiles(filesToUpload);

                submitButton.textContent = 'Отправка заявки...';

                const orderPayload = {
                    UserID: App.state.user.ID,
                    UserChatID: App.state.user.ChatID,
                    Name: form.elements['name'].value,
                    Phone: form.elements['phone'].value,
                    Category: form.elements['category'].value,
                    Subcategory: form.elements['subcategory'].value,
                    Address: form.elements['address'].value,
                    Date: form.elements['date'].value,
                    Time: form.elements['time'].value,
                    Description: form.elements['description'].value,
                    Status: 'new',
                    Payment: 'later',
                    Photos: uploadedMedia.photos,
                    Videos: uploadedMedia.videos
                };

                const data = await App.api.createOrderForUser(orderPayload);

                tg.showPopup({ title: 'Заявка отправлена!', message: `Ваш заказ №${data.order_id} создан.`, buttons: [{ type: 'ok', text: 'Отлично' }] });
                form.reset();
                App.ui.showPanel('user-panel', 'backward');
                this.handleFetchUserOrders();
                this.resetAndDestroyMediaState('user');

            } catch (error) {
                let errorMessage = error.fileName ? `Ошибка загрузки файла ${error.fileName}.` : error.message;
                tg.showAlert(`Ошибка: ${errorMessage}`);
            } finally {
                submitButton.classList.remove('button-disabled');
                submitButton.textContent = 'Отправить заявку';
            }
        },

        handleContactOperatorPage() {
            const lottieContainer = document.getElementById('contact-operator-lottie');
            if (lottieContainer) { App.ui.renderLottieIcon(lottieContainer, App.lottieIconMap['contact_operator']); }
        },
        handleContactOperator() { App.state.tg.openTelegramLink('https://t.me/bogatiri_crimea?start=contact_operator'); },

        triggerAddMediaToOrder() {
            const fileInput = document.createElement('input');
            fileInput.type = 'file';
            fileInput.multiple = true;
            fileInput.accept = 'image/*,video/*';
            fileInput.style.display = 'none';

            fileInput.addEventListener('change', (event) => {
                this.handleAddNewMediaToOrder(event);
                document.body.removeChild(fileInput);
            }, { once: true });

            document.body.appendChild(fileInput);
            fileInput.click();
        },

        async handleAddNewMediaToOrder(event) {
            const files = event.target.files;
            if (!files || files.length === 0) return;

            const orderId = App.state.selectedOrder.ID;
            if (!orderId) {
                App.ui.showError("Не удалось определить ID заказа.");
                return;
            }

            App.ui.showLoader(true);

            try {
                const filesToUpload = Array.from(files).map(file => ({ file }));
                const newMedia = await this._uploadMediaFiles(filesToUpload);

                if (newMedia.photos.length === 0 && newMedia.videos.length === 0) {
                    throw new Error("Не удалось загрузить выбранные файлы.");
                }

                const response = await App.api.addOrderMedia(orderId, newMedia);

                App.state.tg.showPopup({title: "Успех!", message: response.message || "Медиа файлы успешно добавлены!"});
                this.handleShowOrderDetails(orderId);
            } catch (error) {
                App.state.tg.showAlert(`Ошибка добавления файлов: ${error.message}`);
            } finally {
                App.ui.showLoader(false);
            }
        },
    }
};

/**
 * @section Запуск приложения и инициализация сторонних библиотек
 * Код, который выполняется после загрузки DOM.
 */
document.addEventListener("DOMContentLoaded", () => {
    App.init();
    App.ui.renderLottieIcon(document.getElementById('fab-plus-lottie'), App.lottieIconMap.plus_icon);
    App.ui.renderLottieIcon(document.getElementById('fab-plus-staff-lottie'), App.lottieIconMap.plus_icon);
    App.ui.renderLottieIcon(document.getElementById('lottie-search-icon-orders'), App.lottieIconMap.search_icon);
    App.ui.renderLottieIcon(document.getElementById('lottie-search-icon-clients'), App.lottieIconMap.search_icon);
});