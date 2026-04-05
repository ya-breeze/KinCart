let originalFetch = null;
let isRefreshing = false;
let refreshSubscribers = [];

function subscribeTokenRefresh(cb) {
    refreshSubscribers.push(cb);
}

function onRefreshed(success) {
    refreshSubscribers.forEach((cb) => cb(success));
    refreshSubscribers = [];
}

export const setupInterceptor = () => {
    if (!originalFetch) {
        originalFetch = window.fetch;
    }

    window.fetch = async function (...args) {
        const [resource, config] = args;
        const url = typeof resource === 'string' ? resource : (resource instanceof Request ? resource.url : '');
        const isApiRequest = url.includes('/api/');
        const isRefreshRequest = url.includes('/api/auth/refresh');

        if (isApiRequest) {
            // Always send cookies with API requests
            const newConfig = { ...(config || {}), credentials: 'include', redirect: 'manual' };

            const response = await originalFetch(resource, newConfig);

            // Handle 401 Unauthorized — try cookie-based refresh
            if (response.status === 401 && !isRefreshRequest) {
                if (!isRefreshing) {
                    isRefreshing = true;
                    originalFetch('/api/auth/refresh', {
                        method: 'POST',
                        credentials: 'include',
                    }).then(refreshResp => {
                        isRefreshing = false;
                        if (refreshResp.ok) {
                            onRefreshed(true);
                        } else {
                            onRefreshed(false);
                            window.dispatchEvent(new Event('auth:session-expired'));
                        }
                    }).catch(err => {
                        isRefreshing = false;
                        console.error('Refresh request failed', err);
                        onRefreshed(false);
                        window.dispatchEvent(new Event('auth:session-expired'));
                    });
                }

                // Wait for refresh then retry original request (or return original 401 on failure)
                const originalResponse = response;
                return new Promise((resolve) => {
                    subscribeTokenRefresh((success) => {
                        if (success) {
                            resolve(originalFetch(resource, newConfig));
                        } else {
                            resolve(originalResponse);
                        }
                    });
                });
            }

            return response;
        }
        return originalFetch(...args);
    };
};

export const resetInterceptor = () => {
    if (originalFetch) {
        window.fetch = originalFetch;
    }
    originalFetch = null;
    isRefreshing = false;
    refreshSubscribers = [];
};
