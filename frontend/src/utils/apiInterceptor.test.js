import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { setupInterceptor, resetInterceptor } from './apiInterceptor';

describe('apiInterceptor', () => {
    const originalFetch = window.fetch;
    const originalLocation = window.location;

    let fetchMock;

    beforeEach(() => {
        resetInterceptor();

        // Mock window.location
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

    it('adds credentials: include to API requests', async () => {
        setupInterceptor();
        fetchMock.mockResolvedValue({ ok: true, status: 200 });

        await window.fetch('http://localhost/api/test');

        expect(fetchMock).toHaveBeenCalledWith('http://localhost/api/test', expect.objectContaining({
            credentials: 'include',
        }));
    });

    it('does not modify non-API requests', async () => {
        setupInterceptor();
        fetchMock.mockResolvedValue({ ok: true });

        await window.fetch('http://localhost/static/file.js');

        expect(fetchMock).toHaveBeenCalledWith('http://localhost/static/file.js');
    });

    it('handles 401 by refreshing via cookie and retrying', async () => {
        setupInterceptor();

        fetchMock
            .mockResolvedValueOnce({ status: 401 })       // original request → 401
            .mockResolvedValueOnce({ ok: true })           // refresh → success
            .mockResolvedValueOnce({ ok: true, status: 200 }); // retry → 200

        const response = await window.fetch('http://localhost/api/test');

        expect(fetchMock).toHaveBeenCalledWith('/api/auth/refresh', expect.objectContaining({
            method: 'POST',
            credentials: 'include',
        }));
        expect(response.ok).toBe(true);
    });

    it('dispatches auth:session-expired when refresh fails', async () => {
        setupInterceptor();

        fetchMock
            .mockResolvedValueOnce({ status: 401 })
            .mockResolvedValueOnce({ ok: false });

        const eventSpy = vi.fn();
        window.addEventListener('auth:session-expired', eventSpy);

        window.fetch('http://localhost/api/test');

        await vi.waitFor(() => {
            expect(eventSpy).toHaveBeenCalled();
        });

        window.removeEventListener('auth:session-expired', eventSpy);
    });
});
