/**
 * Keyboard Shortcuts Module
 * Provides keyboard navigation for better accessibility and power users
 */

class KeyboardShortcuts {
    constructor(app) {
        this.app = app;
        this.shortcuts = new Map();
        this.isEnabled = true;
        
        // Define shortcuts
        this.defineShortcuts();
        
        // Initialize
        this.init();
    }

    /**
     * Define all keyboard shortcuts
     */
    defineShortcuts() {
        // Navigation shortcuts
        this.shortcuts.set('1', { 
            key: '1', 
            description: 'Открыть заказы', 
            action: () => this.navigateTo('orders')
        });
        
        this.shortcuts.set('2', { 
            key: '2', 
            description: 'Открыть клиенты', 
            action: () => this.navigateTo('clients')
        });
        
        this.shortcuts.set('3', { 
            key: '3', 
            description: 'Открыть статистику', 
            action: () => this.navigateTo('stats')
        });
        
        this.shortcuts.set('4', { 
            key: '4', 
            description: 'Открыть профиль', 
            action: () => this.navigateTo('profile')
        });
        
        // Search shortcut
        this.shortcuts.set('/', { 
            key: '/', 
            description: 'Фокус на поиске', 
            action: () => this.focusSearch()
        });
        
        // Escape
        this.shortcuts.set('Escape', { 
            key: 'Escape', 
            description: 'Закрыть модальное окно / Очистить поиск', 
            action: () => this.handleEscape()
        });
        
        // Refresh
        this.shortcuts.set('r', { 
            key: 'r', 
            description: 'Обновить данные', 
            action: () => this.refreshCurrentPanel()
        });
        
        // Help
        this.shortcuts.set('?', { 
            key: '?', 
            description: 'Показать справку по горячим клавишам', 
            action: () => this.showHelp()
        });
        
        // Arrow navigation for lists
        this.shortcuts.set('ArrowDown', { 
            key: '↓', 
            description: 'Следующий элемент', 
            action: () => this.navigateList('down')
        });
        
        this.shortcuts.set('ArrowUp', { 
            key: '↑', 
            description: 'Предыдущий элемент', 
            action: () => this.navigateList('up')
        });
        
        this.shortcuts.set('Enter', { 
            key: 'Enter', 
            description: 'Открыть выбранный элемент', 
            action: () => this.openSelectedItem()
        });
    }

    /**
     * Initialize keyboard shortcuts
     */
    init() {
        // Add keyboard event listener
        document.addEventListener('keydown', this.handleKeyPress.bind(this));
        
        console.log('✅ Keyboard shortcuts initialized');
    }

    /**
     * Handle keyboard press
     */
    handleKeyPress(event) {
        // Don't handle if disabled or user is typing
        if (!this.isEnabled || this.isTyping(event)) {
            return;
        }

        // Get the key
        let key = event.key;
        
        // Special handling for shift+?
        if (event.shiftKey && key === '?') {
            key = '?';
        }
        
        // Check if shortcut exists
        if (this.shortcuts.has(key)) {
            event.preventDefault();
            const shortcut = this.shortcuts.get(key);
            shortcut.action();
        }
    }

    /**
     * Check if user is typing in an input field
     */
    isTyping(event) {
        const tagName = event.target.tagName.toLowerCase();
        return tagName === 'input' || tagName === 'textarea' || tagName === 'select';
    }

    /**
     * Navigate to a specific panel
     */
    navigateTo(panel) {
        // Check if panel exists and is visible
        const navButton = document.querySelector(`[data-panel="${panel}"]`);
        if (navButton && navButton.style.display !== 'none') {
            navButton.click();
        }
    }

    /**
     * Focus on the search input of current panel
     */
    focusSearch() {
        // Get current panel from global variable or DOM
        const activePanel = document.querySelector('.panel.active');
        const currentPanel = activePanel ? activePanel.id.replace('-panel', '') : 'orders';
        let searchInput;
        
        switch(currentPanel) {
            case 'orders':
                searchInput = document.getElementById('orders-search');
                break;
            case 'clients':
                searchInput = document.getElementById('clients-search');
                break;
        }
        
        if (searchInput) {
            searchInput.focus();
            searchInput.select();
        }
    }

    /**
     * Handle escape key
     */
    handleEscape() {
        // Check if modal is open
        const modal = document.getElementById('modal');
        if (modal && modal.classList.contains('active')) {
            // Trigger close button click or hide modal
            const closeBtn = modal.querySelector('.modal-close');
            if (closeBtn) {
                closeBtn.click();
            } else {
                modal.classList.remove('active');
            }
            return;
        }
        
        // Clear search if focused
        const activeElement = document.activeElement;
        if (activeElement && activeElement.classList.contains('search-input')) {
            activeElement.value = '';
            activeElement.dispatchEvent(new Event('input'));
            activeElement.blur();
        }
    }

    /**
     * Refresh current panel data
     */
    refreshCurrentPanel() {
        // Get current panel from DOM
        const activePanel = document.querySelector('.panel.active');
        const currentPanel = activePanel ? activePanel.id.replace('-panel', '') : 'orders';
        
        // Call appropriate load function from global UI object
        switch(currentPanel) {
            case 'orders':
                if (window.UI && window.UI.loadOrders) {
                    window.UI.loadOrders();
                    window.UI.showToast('Заказы обновлены', 'success');
                }
                break;
            case 'clients':
                if (window.UI && window.UI.loadClients) {
                    window.UI.loadClients();
                    window.UI.showToast('Клиенты обновлены', 'success');
                }
                break;
            case 'stats':
                if (window.UI && window.UI.loadStats) {
                    window.UI.loadStats();
                    window.UI.showToast('Статистика обновлена', 'success');
                }
                break;
            case 'profile':
                if (window.UI && window.UI.loadProfile) {
                    window.UI.loadProfile();
                    window.UI.showToast('Профиль обновлен', 'success');
                }
                break;
        }
    }

    /**
     * Navigate through list items
     */
    navigateList(direction) {
        // Get current panel from DOM
        const activePanel = document.querySelector('.panel.active');
        const currentPanel = activePanel ? activePanel.id.replace('-panel', '') : 'orders';
        let items;
        
        // Get list items based on current panel
        switch(currentPanel) {
            case 'orders':
                items = document.querySelectorAll('.order-card');
                break;
            case 'clients':
                items = document.querySelectorAll('.client-card');
                break;
            default:
                return;
        }
        
        if (!items.length) return;
        
        // Find currently selected item
        let selectedIndex = -1;
        items.forEach((item, index) => {
            if (item.classList.contains('keyboard-selected')) {
                selectedIndex = index;
            }
        });
        
        // Calculate new index
        let newIndex;
        if (direction === 'down') {
            newIndex = selectedIndex < items.length - 1 ? selectedIndex + 1 : 0;
        } else {
            newIndex = selectedIndex > 0 ? selectedIndex - 1 : items.length - 1;
        }
        
        // Update selection
        items.forEach(item => item.classList.remove('keyboard-selected'));
        items[newIndex].classList.add('keyboard-selected');
        items[newIndex].scrollIntoView({ behavior: 'smooth', block: 'nearest' });
    }

    /**
     * Open selected item
     */
    openSelectedItem() {
        const selected = document.querySelector('.keyboard-selected');
        if (selected) {
            selected.click();
        }
    }

    /**
     * Show keyboard shortcuts help
     */
    showHelp() {
        const helpContent = `
            <div class="keyboard-help">
                <h3>Горячие клавиши</h3>
                <div class="shortcuts-list">
                    ${Array.from(this.shortcuts.values()).map(shortcut => `
                        <div class="shortcut-item">
                            <kbd>${shortcut.key}</kbd>
                            <span>${shortcut.description}</span>
                        </div>
                    `).join('')}
                </div>
                <p class="help-note">
                    Горячие клавиши не работают при вводе текста в поля
                </p>
            </div>
        `;
        
        // Create modal
        const modal = document.createElement('div');
        modal.className = 'modal-overlay keyboard-help-modal';
        modal.innerHTML = `
            <div class="modal-container">
                <div class="modal-header">
                    <h2>Справка по горячим клавишам</h2>
                    <button class="modal-close" onclick="this.closest('.modal-overlay').remove()">
                        <i class="fas fa-times"></i>
                    </button>
                </div>
                <div class="modal-body">
                    ${helpContent}
                </div>
            </div>
        `;
        
        document.body.appendChild(modal);
        
        // Add styles if not exist
        if (!document.getElementById('keyboard-shortcuts-styles')) {
            const style = document.createElement('style');
            style.id = 'keyboard-shortcuts-styles';
            style.textContent = `
                .keyboard-selected {
                    outline: 2px solid var(--accent) !important;
                    outline-offset: 2px;
                }
                
                .keyboard-help {
                    padding: 20px;
                }
                
                .keyboard-help h3 {
                    margin-bottom: 20px;
                    color: var(--text-primary);
                }
                
                .shortcuts-list {
                    display: flex;
                    flex-direction: column;
                    gap: 12px;
                    margin-bottom: 20px;
                }
                
                .shortcut-item {
                    display: flex;
                    align-items: center;
                    gap: 16px;
                    padding: 12px;
                    background: var(--bg-glass);
                    border-radius: var(--radius-sm);
                }
                
                .shortcut-item kbd {
                    display: inline-flex;
                    align-items: center;
                    justify-content: center;
                    min-width: 40px;
                    padding: 6px 12px;
                    background: var(--bg-tertiary);
                    border: 1px solid var(--border);
                    border-radius: 6px;
                    font-family: monospace;
                    font-size: 14px;
                    font-weight: 600;
                    color: var(--accent);
                    box-shadow: 0 2px 4px rgba(0,0,0,0.2);
                }
                
                .shortcut-item span {
                    color: var(--text-secondary);
                    font-size: 14px;
                }
                
                .help-note {
                    color: var(--text-hint);
                    font-size: 13px;
                    font-style: italic;
                    text-align: center;
                }
                
                .keyboard-help-modal {
                    position: fixed;
                    top: 0;
                    left: 0;
                    width: 100%;
                    height: 100%;
                    background: rgba(0, 0, 0, 0.8);
                    display: flex;
                    align-items: center;
                    justify-content: center;
                    z-index: 10000;
                    backdrop-filter: blur(4px);
                    animation: fadeIn 0.3s ease-out;
                }
                
                .keyboard-help-modal .modal-container {
                    max-width: 500px;
                }
            `;
            document.head.appendChild(style);
        }
    }

    /**
     * Toggle shortcuts on/off
     */
    toggle(enabled) {
        this.isEnabled = enabled;
    }
}

// Export
window.KeyboardShortcuts = KeyboardShortcuts;