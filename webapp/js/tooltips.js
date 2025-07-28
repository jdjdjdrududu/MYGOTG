/**
 * Tooltips Initialization Module
 * Adds helpful tooltips to important UI elements
 */

class TooltipsManager {
    constructor() {
        this.tooltips = new Map();
        this.init();
    }

    /**
     * Initialize tooltips
     */
    init() {
        // Wait for DOM and UIComponents to be ready
        if (document.readyState === 'loading') {
            document.addEventListener('DOMContentLoaded', () => this.setupTooltips());
        } else {
            this.setupTooltips();
        }
    }

    /**
     * Setup tooltips for UI elements
     */
    setupTooltips() {
        // Make sure UIComponents is available
        if (!window.UIComponents) {
            console.warn('UIComponents not found, retrying...');
            setTimeout(() => this.setupTooltips(), 100);
            return;
        }

        // Navigation items tooltips
        this.addNavigationTooltips();
        
        // Button tooltips
        this.addButtonTooltips();
        
        // Status tooltips
        this.addStatusTooltips();
        
        // Action tooltips
        this.addActionTooltips();

        console.log('✅ Tooltips initialized');
    }

    /**
     * Add tooltips to navigation items
     */
    addNavigationTooltips() {
        const navItems = document.querySelectorAll('.nav-item');
        navItems.forEach(item => {
            const panel = item.getAttribute('data-panel');
            let tooltip = '';
            
            switch(panel) {
                case 'orders':
                    tooltip = 'Просмотр и управление заказами';
                    break;
                case 'clients':
                    tooltip = 'База клиентов и контактов';
                    break;
                case 'stats':
                    tooltip = 'Статистика и аналитика';
                    break;
                case 'profile':
                    tooltip = 'Настройки профиля';
                    break;
            }
            
            if (tooltip) {
                window.UIComponents.createTooltip(item, tooltip, { position: 'top' });
            }
        });
    }

    /**
     * Add tooltips to buttons
     */
    addButtonTooltips() {
        // FAB button
        const fab = document.querySelector('.fab');
        if (fab) {
            window.UIComponents.createTooltip(fab, 'Создать новый заказ', { position: 'left' });
        }

        // Close buttons
        const closeButtons = document.querySelectorAll('.modal-close');
        closeButtons.forEach(btn => {
            window.UIComponents.createTooltip(btn, 'Закрыть', { position: 'left' });
        });

        // Search clear buttons
        const clearButtons = document.querySelectorAll('.search-clear');
        clearButtons.forEach(btn => {
            window.UIComponents.createTooltip(btn, 'Очистить поиск', { position: 'bottom' });
        });
    }

    /**
     * Add tooltips to status badges
     */
    addStatusTooltips() {
        // Order status badges
        const statusMap = {
            'pending': 'Ожидает обработки',
            'processing': 'В процессе выполнения',
            'completed': 'Заказ завершен',
            'cancelled': 'Заказ отменен',
            'active': 'Активный клиент',
            'inactive': 'Неактивный клиент'
        };

        // This will be called whenever status badges are rendered
        this.observeStatusBadges(statusMap);
    }

    /**
     * Add tooltips to action buttons
     */
    addActionTooltips() {
        // Dynamic elements observer
        const observer = new MutationObserver((mutations) => {
            mutations.forEach(mutation => {
                mutation.addedNodes.forEach(node => {
                    if (node.nodeType === 1) { // Element node
                        this.addDynamicTooltips(node);
                    }
                });
            });
        });

        // Observe the main content area
        const mainContent = document.querySelector('.app-container');
        if (mainContent) {
            observer.observe(mainContent, {
                childList: true,
                subtree: true
            });
        }
    }

    /**
     * Add tooltips to dynamically added elements
     */
    addDynamicTooltips(element) {
        // Action buttons in cards
        const editButtons = element.querySelectorAll('.btn-edit, [data-action="edit"]');
        editButtons.forEach(btn => {
            if (!this.tooltips.has(btn)) {
                window.UIComponents.createTooltip(btn, 'Редактировать', { position: 'top' });
                this.tooltips.set(btn, true);
            }
        });

        const deleteButtons = element.querySelectorAll('.btn-delete, [data-action="delete"]');
        deleteButtons.forEach(btn => {
            if (!this.tooltips.has(btn)) {
                window.UIComponents.createTooltip(btn, 'Удалить', { position: 'top' });
                this.tooltips.set(btn, true);
            }
        });

        const viewButtons = element.querySelectorAll('.btn-view, [data-action="view"]');
        viewButtons.forEach(btn => {
            if (!this.tooltips.has(btn)) {
                window.UIComponents.createTooltip(btn, 'Подробнее', { position: 'top' });
                this.tooltips.set(btn, true);
            }
        });

        // Stat cards
        const statCards = element.querySelectorAll('.stat-card[data-tooltip]');
        statCards.forEach(card => {
            if (!this.tooltips.has(card)) {
                const tooltip = card.getAttribute('data-tooltip');
                if (tooltip) {
                    window.UIComponents.createTooltip(card, tooltip, { position: 'top' });
                    this.tooltips.set(card, true);
                }
            }
        });
    }

    /**
     * Observe status badges for tooltips
     */
    observeStatusBadges(statusMap) {
        const observer = new MutationObserver((mutations) => {
            mutations.forEach(mutation => {
                mutation.addedNodes.forEach(node => {
                    if (node.nodeType === 1) {
                        const badges = node.querySelectorAll('.status-badge, .status');
                        badges.forEach(badge => {
                            if (!this.tooltips.has(badge)) {
                                const status = badge.textContent.toLowerCase().trim();
                                const tooltip = statusMap[status];
                                if (tooltip) {
                                    window.UIComponents.createTooltip(badge, tooltip, { position: 'top' });
                                    this.tooltips.set(badge, true);
                                }
                            }
                        });
                    }
                });
            });
        });

        const containers = document.querySelectorAll('.orders-list, .clients-list');
        containers.forEach(container => {
            observer.observe(container, {
                childList: true,
                subtree: true
            });
        });
    }

    /**
     * Add tooltip to a specific element
     */
    addTooltip(element, content, position = 'top') {
        if (element && !this.tooltips.has(element)) {
            window.UIComponents.createTooltip(element, content, { position });
            this.tooltips.set(element, true);
        }
    }

    /**
     * Remove all tooltips
     */
    clearTooltips() {
        this.tooltips.clear();
    }
}

// Initialize tooltips manager
window.TooltipsManager = new TooltipsManager();