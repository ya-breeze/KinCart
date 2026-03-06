import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { setupInterceptor, resetInterceptor } from './apiInterceptor';

describe('apiInterceptor', () => {
    const originalFetch = window.fetch;
    const originalLocation = window.location;

    let fetchMock;

    beforeEach(() => {
        resetInterceptor();
        vi.stubGlobal('localStorage', {
            getItem: vi.fn(),
            setItem: vi.fn(),
            removeItem: vi.fn(),
        });

        // Mock window.location.reload
        delete window.location;
        window.location = { reload: vi.fn() };

        fetchMock = vi.fn();
        vi.stubGlobal('fetch', fetchMock);
    });

    afterEach(() => {
        window.fetch = originalFetch;
        window.location = originalLocation;
        vi.restoreAllMocks();
    });

    it('injects Authorization header if token exists', async () => {
        setupInterceptor();
        localStorage.getItem.mockReturnValue('test-token');
        fetchMock.mockResolvedValue({ ok: true });

        await window.fetch('http://localhost/api/test');

        expect(fetchMock).toHaveBeenCalledWith('http://localhost/api/test', expect.objectContaining({
            headers: expect.objectContaining({
                'Authorization': 'Bearer test-token'
            })
        }));
    });

    it('handles Cloudflare opaque redirects by reloading', async () => {
        setupInterceptor();
        fetchMock.mockResolvedValue({ type: 'opaqueredirect' });

        // This call will return a never-resolving promise by design
        window.fetch('http://localhost/api/test');

        await vi.waitFor(() => {
            expect(window.location.reload).toHaveBeenCalled();
        });
    });

    it('handles KinCart 401 by refreshing token', async () => {
        setupInterceptor();
        localStorage.getItem.mockImplementation((key) => {
            if (key === 'token') return 'expired-token';
            if (key === 'refresh_token') return 'valid-refresh-token';
            return null;
        });

        // First call returns 401
        // Second call (refresh) returns new token
        // Third call (retry) returns 200
        fetchMock
            .mockResolvedValueOnce({ status: 401 })
            .mockResolvedValueOnce({
                ok: true,
                json: () => Promise.resolve({ token: 'new-token' })
            })
            .mockResolvedValueOnce({ ok: true });

        const response = await window.fetch('http://localhost/api/test');

        expect(fetchMock).toHaveBeenCalledWith('/api/auth/refresh', expect.any(Object));
        expect(localStorage.setItem).toHaveBeenCalledWith('token', 'new-token');
        expect(response.ok).toBe(true);
    });

    it('handles failed refresh by logging out and reloading', async () => {
        setupInterceptor();
        localStorage.getItem.mockReturnValue('some-token');

        fetchMock
            .mockResolvedValueOnce({ status: 401 })
            .mockResolvedValueOnce({ ok: false }); // Refresh failed

        window.fetch('http://localhost/api/test');

        await vi.waitFor(() => {
            expect(localStorage.removeItem).toHaveBeenCalledWith('token');
            expect(localStorage.removeItem).toHaveBeenCalledWith('refresh_token');
            expect(window.location.reload).toHaveBeenCalled();
        });
    });
});
