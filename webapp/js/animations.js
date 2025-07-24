/**
 * Animation Utilities Module
 */
class AnimationUtils {
    static init() {
        // Add ripple effect to all buttons
        document.querySelectorAll('button').forEach(button => {
            button.addEventListener('click', (e) => this.addRippleEffect(button, e));
        });
    }

    /**
     * Add ripple effect to element
     */
    static addRippleEffect(element, event) {
        const ripple = document.createElement('span');
        ripple.classList.add('ripple');
        
        const rect = element.getBoundingClientRect();
        const size = Math.max(rect.width, rect.height);
        
        ripple.style.width = ripple.style.height = `${size}px`;
        ripple.style.left = `${event.clientX - rect.left}px`;
        ripple.style.top = `${event.clientY - rect.top}px`;
        
        element.appendChild(ripple);
        ripple.addEventListener('animationend', () => ripple.remove());
    }

    /**
     * Animate panel transition
     */
    static animatePanelTransition(currentPanel, newPanel) {
        if (currentPanel) {
            currentPanel.classList.add('fade-out');
            setTimeout(() => {
                currentPanel.classList.remove('active', 'fade-out');
                currentPanel.classList.add('hidden');
            }, window.APP_CONFIG.ANIMATION_DURATION);
        }

        if (newPanel) {
            newPanel.classList.remove('hidden');
            newPanel.classList.add('active', 'fade-in');
            setTimeout(() => {
                newPanel.classList.remove('fade-in');
            }, window.APP_CONFIG.ANIMATION_DURATION);
        }
    }
}

// Export the module
window.AnimationUtils = AnimationUtils; 