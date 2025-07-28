/**
 * Skeleton Loader Module
 * Provides skeleton screens for better loading UX
 */

class SkeletonLoader {
    /**
     * Show skeleton screens for orders list
     */
    static showOrdersSkeleton(container, count = 5) {
        container.innerHTML = '';
        
        for (let i = 0; i < count; i++) {
            const skeleton = document.createElement('div');
            skeleton.className = 'skeleton-card order-card';
            skeleton.innerHTML = `
                <div class="skeleton-header">
                    <div class="skeleton-lines" style="flex: 1;">
                        <div class="skeleton-line" style="width: 50%; height: 18px;"></div>
                        <div class="skeleton-line" style="width: 30%; height: 14px;"></div>
                    </div>
                    <div class="skeleton skeleton-text" style="width: 80px; height: 24px; border-radius: 12px;"></div>
                </div>
                <div class="skeleton-lines">
                    <div class="skeleton-line" style="width: 70%;"></div>
                    <div class="skeleton-line" style="width: 40%;"></div>
                </div>
            `;
            container.appendChild(skeleton);
        }
    }

    /**
     * Show skeleton screens for clients list
     */
    static showClientsSkeleton(container, count = 5) {
        container.innerHTML = '';
        
        for (let i = 0; i < count; i++) {
            const skeleton = document.createElement('div');
            skeleton.className = 'skeleton-card client-card';
            skeleton.innerHTML = `
                <div class="skeleton-header">
                    <div class="skeleton-avatar"></div>
                    <div class="skeleton-lines">
                        <div class="skeleton-line" style="width: 60%; height: 16px;"></div>
                        <div class="skeleton-line" style="width: 40%; height: 12px;"></div>
                    </div>
                </div>
            `;
            container.appendChild(skeleton);
        }
    }

    /**
     * Show skeleton screen for profile
     */
    static showProfileSkeleton(container) {
        container.innerHTML = `
            <div class="skeleton-container">
                <div class="profile-header" style="background: none; border: 1px solid var(--border);">
                    <div class="skeleton skeleton-image" style="width: 120px; height: 120px; border-radius: 50%; margin: 0 auto 20px;"></div>
                    <div class="skeleton skeleton-text" style="width: 200px; height: 28px; margin: 0 auto 12px;"></div>
                    <div class="skeleton skeleton-text" style="width: 150px; height: 20px; margin: 0 auto;"></div>
                </div>
                <div class="profile-stats" style="margin-top: 32px;">
                    ${[1, 2, 3].map(() => `
                        <div class="stat-card">
                            <div class="skeleton skeleton-text" style="width: 60px; height: 36px; margin: 0 auto 8px;"></div>
                            <div class="skeleton skeleton-text" style="width: 80%; height: 14px; margin: 0 auto;"></div>
                        </div>
                    `).join('')}
                </div>
            </div>
        `;
    }

    /**
     * Show skeleton screen for statistics
     */
    static showStatsSkeleton(container) {
        container.innerHTML = `
            <div class="skeleton-container">
                <div class="stats-overview">
                    ${[1, 2, 3, 4].map(() => `
                        <div class="stat-card">
                            <div class="skeleton skeleton-image" style="width: 60px; height: 60px; border-radius: 50%;"></div>
                            <div class="stat-content">
                                <div class="skeleton skeleton-text" style="width: 80px; height: 32px; margin-bottom: 8px;"></div>
                                <div class="skeleton skeleton-text" style="width: 120px; height: 14px;"></div>
                            </div>
                        </div>
                    `).join('')}
                </div>
                <div class="charts-section" style="margin-top: 32px;">
                    <div class="chart-card">
                        <div class="skeleton skeleton-text" style="width: 150px; height: 20px; margin-bottom: 20px;"></div>
                        <div class="skeleton skeleton-image" style="width: 100%; height: 200px; border-radius: 8px;"></div>
                    </div>
                    <div class="chart-card">
                        <div class="skeleton skeleton-text" style="width: 150px; height: 20px; margin-bottom: 20px;"></div>
                        <div class="skeleton skeleton-image" style="width: 100%; height: 200px; border-radius: 8px;"></div>
                    </div>
                </div>
            </div>
        `;
    }

    /**
     * Show skeleton for modal content
     */
    static showModalSkeleton(container) {
        container.innerHTML = `
            <div class="skeleton-container">
                <div class="skeleton skeleton-text" style="width: 200px; height: 24px; margin-bottom: 24px;"></div>
                ${[1, 2, 3, 4].map(() => `
                    <div style="margin-bottom: 16px;">
                        <div class="skeleton skeleton-text" style="width: 40%; height: 14px; margin-bottom: 8px;"></div>
                        <div class="skeleton skeleton-text" style="width: 80%; height: 16px;"></div>
                    </div>
                `).join('')}
            </div>
        `;
    }

    /**
     * Create custom skeleton
     */
    static createSkeleton(type = 'text', options = {}) {
        const skeleton = document.createElement('div');
        skeleton.className = `skeleton skeleton-${type}`;

        // Apply custom styles
        if (options.width) skeleton.style.width = options.width;
        if (options.height) skeleton.style.height = options.height;
        if (options.borderRadius) skeleton.style.borderRadius = options.borderRadius;
        if (options.margin) skeleton.style.margin = options.margin;

        return skeleton;
    }
}

// Export
window.SkeletonLoader = SkeletonLoader;