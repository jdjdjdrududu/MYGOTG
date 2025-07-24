/**
 * UI Components Module
 * Collection of reusable UI components and utilities
 */

class UIComponents {
    constructor() {
        // Ensure singleton
        if (window.UIComponents) {
            return window.UIComponents;
        }

        this.activeModals = new Set();
        this.notifications = [];
        this.notificationTimeout = 5000;
        
        // Bind methods
        this.showModal = this.showModal.bind(this);
        this.closeModal = this.closeModal.bind(this);
        this.showNotification = this.showNotification.bind(this);
        this.createLoader = this.createLoader.bind(this);

        // Initialize when constructed
        this.init();

        // Save instance
        window.UIComponents = this;
    }

    /**
     * Initialize UI Components
     */
    init() {
        // Create notifications container if it doesn't exist
        if (!document.getElementById('notifications')) {
            const container = document.createElement('div');
            container.id = 'notifications';
            container.className = 'notifications';
            document.body.appendChild(container);
        }

        console.log('✅ UI Components initialized');
    }

    /**
     * Show a modal dialog
     * @param {string} content - Modal content HTML
     * @param {Object} options - Modal options
     * @returns {HTMLElement} - Modal element
     */
    showModal(content, options = {}) {
        const defaultOptions = {
            title: '',
            closeOnEscape: true,
            closeOnOverlayClick: true,
            onClose: null,
            width: '500px',
            height: 'auto'
        };

        const modalOptions = { ...defaultOptions, ...options };

        const modal = document.createElement('div');
        modal.className = 'modal';
        modal.innerHTML = `
            <div class="modal-content" style="width: ${modalOptions.width}; height: ${modalOptions.height};">
                ${modalOptions.title ? `
                    <div class="modal-header">
                        <h3 class="modal-title">${modalOptions.title}</h3>
                        <button class="modal-close">&times;</button>
                    </div>
                ` : ''}
                <div class="modal-body">${content}</div>
            </div>
        `;

        // Event handlers
        const closeModal = () => {
            this.closeModal(modal);
            if (modalOptions.onClose) modalOptions.onClose();
        };

        modal.querySelector('.modal-close')?.addEventListener('click', closeModal);

        if (modalOptions.closeOnOverlayClick) {
            modal.addEventListener('click', (e) => {
                if (e.target === modal) closeModal();
            });
        }

        if (modalOptions.closeOnEscape) {
            document.addEventListener('keydown', (e) => {
                if (e.key === 'Escape') closeModal();
            });
        }

        document.body.appendChild(modal);
        this.activeModals.add(modal);

        // Trigger animation
        requestAnimationFrame(() => modal.classList.add('show'));

        return modal;
    }

    /**
     * Close a modal dialog
     * @param {HTMLElement} modal - Modal element to close
     */
    closeModal(modal) {
        modal.classList.remove('show');
        modal.addEventListener('transitionend', () => {
            modal.remove();
            this.activeModals.delete(modal);
        });
    }

    /**
     * Show a notification
     * @param {string} message - Notification message
     * @param {string} type - Notification type (success, error, warning)
     * @param {Object} options - Additional options
     */
    showNotification(message, type = 'info', options = {}) {
        const defaultOptions = {
            duration: this.notificationTimeout,
            position: 'top-right'
        };

        const notificationOptions = { ...defaultOptions, ...options };

        const notification = document.createElement('div');
        notification.className = `notification ${type}`;
        notification.innerHTML = `
            <div class="notification-content">
                <div class="notification-message">${message}</div>
                ${options.showClose ? '<button class="notification-close">&times;</button>' : ''}
            </div>
        `;

        // Position the notification
        notification.style.cssText = this.getNotificationPosition(notificationOptions.position);

        // Add to DOM
        const container = document.getElementById('notifications');
        if (container) {
            container.appendChild(notification);
            this.notifications.push(notification);

            // Trigger animation
            requestAnimationFrame(() => notification.classList.add('show'));

            // Auto close
            if (notificationOptions.duration) {
                setTimeout(() => this.closeNotification(notification), notificationOptions.duration);
            }

            // Close button handler
            if (options.showClose) {
                notification.querySelector('.notification-close').addEventListener('click', () => {
                    this.closeNotification(notification);
                });
            }
        } else {
            console.warn('Notifications container not found');
            // Fallback to console
            console.log(`${type.toUpperCase()}: ${message}`);
        }

        return notification;
    }

    /**
     * Close a notification
     * @param {HTMLElement} notification - Notification element
     */
    closeNotification(notification) {
        notification.classList.remove('show');
        notification.addEventListener('transitionend', () => {
            notification.remove();
            this.notifications = this.notifications.filter(n => n !== notification);
        });
    }

    /**
     * Get notification position CSS
     * @param {string} position - Position string
     * @returns {string} - CSS text
     */
    getNotificationPosition(position) {
        const positions = {
            'top-right': 'top: 20px; right: 20px;',
            'top-left': 'top: 20px; left: 20px;',
            'bottom-right': 'bottom: 20px; right: 20px;',
            'bottom-left': 'bottom: 20px; left: 20px;'
        };
        return positions[position] || positions['top-right'];
    }

    /**
     * Create a loading spinner
     * @param {string} size - Spinner size (small, medium, large)
     * @returns {HTMLElement} - Spinner element
     */
    createLoader(size = 'medium') {
        const sizes = {
            small: '20px',
            medium: '40px',
            large: '60px'
        };

        const loader = document.createElement('div');
        loader.className = 'loading-spinner';
        loader.style.width = sizes[size];
        loader.style.height = sizes[size];

        return loader;
    }

    /**
     * Create a skeleton loading placeholder
     * @param {string} type - Skeleton type (text, card, image)
     * @param {Object} options - Skeleton options
     * @returns {HTMLElement} - Skeleton element
     */
    createSkeleton(type = 'text', options = {}) {
        const skeleton = document.createElement('div');
        skeleton.className = `skeleton skeleton-${type}`;

        switch (type) {
            case 'text':
                skeleton.style.height = options.height || '20px';
                skeleton.style.width = options.width || '100%';
                break;
            case 'card':
                skeleton.style.height = options.height || '200px';
                skeleton.style.width = options.width || '100%';
                break;
            case 'image':
                skeleton.style.height = options.height || '200px';
                skeleton.style.width = options.width || '200px';
                skeleton.style.borderRadius = options.rounded ? '50%' : '4px';
                break;
        }

        return skeleton;
    }

    /**
     * Create a tooltip
     * @param {HTMLElement} element - Target element
     * @param {string} content - Tooltip content
     * @param {Object} options - Tooltip options
     */
    createTooltip(element, content, options = {}) {
        const defaultOptions = {
            position: 'top',
            showDelay: 200,
            hideDelay: 200
        };

        const tooltipOptions = { ...defaultOptions, ...options };
        let tooltipElement = null;
        let showTimeout = null;
        let hideTimeout = null;

        const showTooltip = () => {
            if (hideTimeout) clearTimeout(hideTimeout);
            
            showTimeout = setTimeout(() => {
                tooltipElement = document.createElement('div');
                tooltipElement.className = `tooltip tooltip-${tooltipOptions.position}`;
                tooltipElement.textContent = content;
                
                document.body.appendChild(tooltipElement);
                
                const elementRect = element.getBoundingClientRect();
                const tooltipRect = tooltipElement.getBoundingClientRect();
                
                this.positionTooltip(tooltipElement, elementRect, tooltipRect, tooltipOptions.position);
                
                requestAnimationFrame(() => tooltipElement.classList.add('show'));
            }, tooltipOptions.showDelay);
        };

        const hideTooltip = () => {
            if (showTimeout) clearTimeout(showTimeout);
            
            hideTimeout = setTimeout(() => {
                if (tooltipElement) {
                    tooltipElement.classList.remove('show');
                    tooltipElement.addEventListener('transitionend', () => {
                        tooltipElement.remove();
                        tooltipElement = null;
                    });
                }
            }, tooltipOptions.hideDelay);
        };

        element.addEventListener('mouseenter', showTooltip);
        element.addEventListener('mouseleave', hideTooltip);
        element.addEventListener('focus', showTooltip);
        element.addEventListener('blur', hideTooltip);
    }

    /**
     * Position a tooltip
     * @param {HTMLElement} tooltip - Tooltip element
     * @param {DOMRect} targetRect - Target element rect
     * @param {DOMRect} tooltipRect - Tooltip element rect
     * @param {string} position - Desired position
     */
    positionTooltip(tooltip, targetRect, tooltipRect, position) {
        const spacing = 8;
        let top, left;

        switch (position) {
            case 'top':
                top = targetRect.top - tooltipRect.height - spacing;
                left = targetRect.left + (targetRect.width - tooltipRect.width) / 2;
                break;
            case 'bottom':
                top = targetRect.bottom + spacing;
                left = targetRect.left + (targetRect.width - tooltipRect.width) / 2;
                break;
            case 'left':
                top = targetRect.top + (targetRect.height - tooltipRect.height) / 2;
                left = targetRect.left - tooltipRect.width - spacing;
                break;
            case 'right':
                top = targetRect.top + (targetRect.height - tooltipRect.height) / 2;
                left = targetRect.right + spacing;
                break;
        }

        tooltip.style.top = `${top}px`;
        tooltip.style.left = `${left}px`;
    }
}

// Initialize UI Components
window.UIComponents = new UIComponents();

// For backwards compatibility and direct script usage
if (typeof module !== 'undefined' && module.exports) {
    module.exports = window.UIComponents;
} 