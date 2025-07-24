/**
 * Core Utilities Module
 */
class CoreUtils {
    /**
     * Format phone number
     */
    static formatPhone(phone) {
        if (!phone) return 'Не указан';
        return phone.replace(/(\d{1})(\d{3})(\d{3})(\d{2})(\d{2})/, '+$1 ($2) $3-$4-$5');
    }

    /**
     * Format date
     */
    static formatDate(date) {
        if (!date) return 'Не указана';
        return new Date(date).toLocaleString('ru-RU');
    }

    /**
     * Format currency
     */
    static formatCurrency(amount) {
        if (!amount) return '0 ₽';
        return new Intl.NumberFormat('ru-RU', {
            style: 'currency',
            currency: 'RUB',
            minimumFractionDigits: 0
        }).format(amount);
    }
}

// Export the module
window.CoreUtils = CoreUtils; 