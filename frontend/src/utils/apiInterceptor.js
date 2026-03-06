let originalFetch = null;
let isRefreshing = false;
let refreshSubscribers = [];

function subscribeTokenRefresh(cb) {
    refreshSubscribers.push(cb);
}

function onTokenRefreshed(token) {
    refreshSubscribers.map((cb) => cb(token));
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
            const newConfig = { ...(config || {}), redirect: 'manual' };

            // Inject the current token if it exists in localStorage and not already provided
            if (!newConfig.headers) {
                newConfig.headers = {};
            }
            if (!newConfig.headers['Authorization']) {
                const token = localStorage.getItem('token');
                if (token) {
                    newConfig.headers['Authorization'] = `Bearer ${token}`;
                }
            }

            const response = await originalFetch(resource, newConfig);

            // Handle KinCart 401 Unauthorized (Token Expired)
            if (response.status === 401 && !isRefreshRequest) {
                const refreshToken = localStorage.getItem('refresh_token');
                if (!refreshToken) {
                    return response; // No refresh token, let the app handle 401 (log out)
                }

                if (!isRefreshing) {
                    isRefreshing = true;
                    // Use originalFetch to avoid infinite loops
                    originalFetch('/api/auth/refresh', {
                        method: 'POST',
                        headers: { 'Content-Type': 'application/json' },
                        body: JSON.stringify({ refresh_token: refreshToken })
                    }).then(async refreshResp => {
                        isRefreshing = false;
                        if (refreshResp.ok) {
                            const data = await refreshResp.json();
                            localStorage.setItem('token', data.token);
                            onTokenRefreshed(data.token);
                        } else {
                            // Refresh failed, clear everything and reload or let subscribers fail
                            console.error('Refresh token invalid or expired');
                            localStorage.removeItem('token');
                            localStorage.removeItem('refresh_token');
                            window.location.reload();
                        }
                    }).catch(err => {
                        isRefreshing = false;
                        console.error('Refresh request failed', err);
                    });
                }

                // Wait for refresh to complete
                return new Promise((resolve) => {
                    subscribeTokenRefresh((newToken) => {
                        const retryConfig = { ...newConfig };
                        retryConfig.headers = { ...retryConfig.headers, 'Authorization': `Bearer ${newToken}` };
                        resolve(originalFetch(resource, retryConfig));
                    });
                });
            }

            return response;
        }
        return originalFetch(...args);
    };
};

export const resetInterceptor = () => {
    originalFetch = null;
    isRefreshing = false;
    refreshSubscribers = [];
};
