/**
 * Utility Module
 * Common utility functions used across the application
 */

class UtilsModule {
    constructor(app) {
        this.app = app;
    }

    /**
     * Format phone number
     */
    formatPhone(phone) {
        if (!phone || typeof phone !== 'string') return 'Не указан';
        const cleaned = phone.replace(/\D/g, '');
        if (cleaned.length === 11 && cleaned.startsWith('7')) {
            return cleaned.replace(/(\d{1})(\d{3})(\d{3})(\d{2})(\d{2})/, '+$1 ($2) $3-$4-$5');
        }
        return phone;
    }

    /**
     * Format currency
     */
    formatCurrency(amount) {
        if (!amount || isNaN(amount)) return 'Не указана';
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
        try {
            return new Date(date).toLocaleString('ru-RU', {
                day: '2-digit',
                month: '2-digit',
                year: 'numeric',
                hour: '2-digit',
                minute: '2-digit'
            });
        } catch (error) {
            return 'Неверная дата';
        }
    }

    /**
     * Debounce function
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
     * Get status text
     */
    getStatusText(status) {
        const statuses = {
            'new': 'Новый',
            'in_progress': 'В работе',
            'completed': 'Выполнен',
            'canceled': 'Отменён',
            'awaiting_payment': 'Ожидает оплаты',
            'awaiting_confirmation': 'Ожидает подтверждения',
            'awaiting_cost': 'Ожидает оценки'
        };
        return statuses[status] || status;
    }

    /**
     * Get role text
     */
    getRoleText(role) {
        const roles = {
            'user': 'Клиент',
            'operator': 'Оператор',
            'admin': 'Администратор',
            'owner': 'Владелец',
            'driver': 'Водитель'
        };
        return roles[role] || role;
    }

    /**
     * Generate unique ID
     */
    generateId() {
        return Math.random().toString(36).substr(2, 9);
    }

    /**
     * Copy to clipboard
     */
    async copyToClipboard(text) {
        try {
            await navigator.clipboard.writeText(text);
            return true;
        } catch (err) {
            console.error('Failed to copy:', err);
            return false;
        }
    }

    /**
     * Show notification
     */
    showNotification(message, type = 'info') {
        // Simple notification implementation
        const notification = document.createElement('div');
        notification.className = `notification notification-${type}`;
        notification.textContent = message;
        
        // Style the notification
        Object.assign(notification.style, {
            position: 'fixed',
            top: '20px',
            right: '20px',
            padding: '12px 20px',
            borderRadius: '8px',
            color: 'white',
            zIndex: '10000',
            backgroundColor: type === 'error' ? '#ff3b30' : 
                             type === 'success' ? '#34c759' : 
                             type === 'warning' ? '#ff9500' : '#007aff',
            boxShadow: '0 4px 12px rgba(0,0,0,0.15)',
            animation: 'slideInRight 0.3s ease-out'
        });

        document.body.appendChild(notification);

        // Remove after 3 seconds
        setTimeout(() => {
            notification.style.animation = 'slideOutRight 0.3s ease-out';
            setTimeout(() => {
                document.body.removeChild(notification);
            }, 300);
        }, 3000);
    }

    /**
     * Validate email
     */
    validateEmail(email) {
        const re = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;
        return re.test(email);
    }

    /**
     * Validate phone
     */
    validatePhone(phone) {
        const re = /^[\+]?[7-8]?[\s\-]?\(?[0-9]{3}\)?[\s\-]?[0-9]{3}[\s\-]?[0-9]{2}[\s\-]?[0-9]{2}$/;
        return re.test(phone);
    }
}

// Animation utilities
window.AnimationUtils = {
    /**
     * Initialize animation utilities
     */
    init() {
        // Add animation styles
        const style = document.createElement('style');
        style.textContent = `
            .panel-transition {
                transition: all 0.3s ease-out;
            }
            
            .panel.active {
                display: block;
                opacity: 1;
                transform: translateX(0);
            }
            
            .panel {
                display: none;
                opacity: 0;
                transform: translateX(20px);
            }
            
            @keyframes slideInRight {
                from { transform: translateX(100%); }
                to { transform: translateX(0); }
            }
            
            @keyframes slideOutRight {
                from { transform: translateX(0); }
                to { transform: translateX(100%); }
            }
            
            .ripple {
                position: absolute;
                background: rgba(255, 255, 255, 0.3);
                border-radius: 50%;
                pointer-events: none;
                width: 100px;
                height: 100px;
                transform: translate(-50%, -50%) scale(0);
                animation: ripple 1s linear;
            }

            @keyframes ripple {
                to {
                    transform: translate(-50%, -50%) scale(4);
                    opacity: 0;
                }
            }
        `;
        document.head.appendChild(style);
    },

    /**
     * Animate panel transition
     */
    animatePanelTransition(currentPanel, newPanel) {
        if (currentPanel) {
            currentPanel.classList.remove('active');
        }
        if (newPanel) {
            newPanel.classList.add('active');
        }
    },

    /**
     * Add ripple effect
     */
    addRippleEffect(element, event) {
        const ripple = document.createElement('span');
        ripple.classList.add('ripple');
        
        const rect = element.getBoundingClientRect();
        const size = Math.max(rect.width, rect.height);
        const x = event.clientX - rect.left - size / 2;
        const y = event.clientY - rect.top - size / 2;
        
        ripple.style.width = ripple.style.height = size + 'px';
        ripple.style.left = x + 'px';
        ripple.style.top = y + 'px';
        
        element.appendChild(ripple);
        
        setTimeout(() => {
            ripple.remove();
        }, 1000);
    }
};

// Core utilities for global access
window.CoreUtils = {
    formatPhone: (phone) => new UtilsModule().formatPhone(phone),
    formatCurrency: (amount) => new UtilsModule().formatCurrency(amount),
    formatDate: (date) => new UtilsModule().formatDate(date),
    getStatusText: (status) => new UtilsModule().getStatusText(status),
    getRoleText: (role) => new UtilsModule().getRoleText(role)
};

// Export module
window.UtilsModule = UtilsModule; 