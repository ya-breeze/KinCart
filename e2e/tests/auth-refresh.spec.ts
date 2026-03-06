import { test, expect } from '@playwright/test';

test.describe('Auth Refresh Token Flow', () => {
    test.beforeEach(async ({ page }) => {
        // standalone mock - no backend needed
        await page.route('**/api/auth/login', async route => {
            await route.fulfill({
                status: 200,
                contentType: 'application/json',
                body: JSON.stringify({
                    token: 'initial-token',
                    refresh_token: 'initial-refresh-token',
                    user: { username: 'dad' }
                })
            });
        });

        await page.route('**/api/auth/me', async route => {
            await route.fulfill({
                status: 200,
                contentType: 'application/json',
                body: JSON.stringify({ username: 'dad' })
            });
        });

        await page.route('**/api/family/config', async route => {
            await route.fulfill({ status: 200, body: JSON.stringify({ currency: '₽' }) });
        });
    });

    test('should automatically refresh token when access token expires', async ({ page }) => {
        // Set up initial session via localStorage instead of manual login to avoid race conditions
        await page.goto('/');
        await page.evaluate(() => {
            localStorage.setItem('token', 'initial-token');
            localStorage.setItem('refresh_token', 'initial-refresh-token');
            localStorage.setItem('mode', 'manager');
        });

        // Reload to enter the dashboard
        await page.goto('/');
        await expect(page.locator('h1').first()).toHaveText('KinCart');

        let refreshCalled = false;
        let retryCalled = false;

        // Mock another API call that will return 401 first
        await page.route('**/api/lists', async route => {
            if (!refreshCalled) {
                await route.fulfill({
                    status: 401,
                    contentType: 'application/json',
                    body: JSON.stringify({ error: 'Token expired' })
                });
            } else {
                retryCalled = true;
                await route.fulfill({
                    status: 200,
                    contentType: 'application/json',
                    body: JSON.stringify([])
                });
            }
        });

        // Mock refresh
        await page.route('**/api/auth/refresh', async route => {
            const request = route.request();
            let postData;
            try {
                postData = request.postDataJSON();
            } catch (e) {
                console.error('Failed to parse refresh POST data', e);
            }

            if (postData && postData.refresh_token === 'initial-refresh-token') {
                refreshCalled = true;
                await route.fulfill({
                    status: 200,
                    contentType: 'application/json',
                    body: JSON.stringify({
                        token: 'new-access-token',
                        user: { username: 'dad' }
                    })
                });
            } else {
                await route.fulfill({ status: 401 });
            }
        });

        // Trigger the list fetch by reloading the dashboard
        // We wait a tiny bit to ensure no pending sign-in navigations are interfering
        await page.waitForTimeout(100);
        await page.reload();

        // Verify refresh occurred and retry happened
        await expect(async () => {
            const token = await page.evaluate(() => localStorage.getItem('token'));
            expect(token).toBe('new-access-token');
        }).toPass({ timeout: 10000 });

        expect(refreshCalled).toBe(true);
        expect(retryCalled).toBe(true);
    });

    test('should log out when refresh token is invalid', async ({ page }) => {
        // Set up initial session via localStorage
        await page.goto('/');
        await page.evaluate(() => {
            localStorage.setItem('token', 'initial-token');
            localStorage.setItem('refresh_token', 'initial-refresh-token');
        });
        await page.goto('/');
        await expect(page.locator('h1').first()).toHaveText('KinCart');

        await page.route('**/api/lists', async route => {
            await route.fulfill({ status: 401 });
        });

        await page.route('**/api/auth/refresh', async route => {
            await route.fulfill({ status: 401, body: JSON.stringify({ error: 'Invalid refresh token' }) });
        });

        await page.waitForTimeout(100);
        await page.reload();

        // Should return to login page
        await expect(page.locator('#username')).toBeVisible({ timeout: 10000 });
        const token = await page.evaluate(() => localStorage.getItem('token'));
        expect(token).toBeNull();
    });
});
