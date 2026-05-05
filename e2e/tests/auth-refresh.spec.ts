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

        // Mock refresh — auth is cookie-based, no body needed from client
        await page.route('**/api/auth/refresh', async route => {
            refreshCalled = true;
            await route.fulfill({
                status: 200,
                contentType: 'application/json',
                body: JSON.stringify({ message: 'Token refreshed' })
            });
        });

        // Trigger the list fetch by reloading the dashboard
        // We wait a tiny bit to ensure no pending sign-in navigations are interfering
        await page.waitForTimeout(100);
        await page.reload();

        // Auth is cookie-based — verify refresh was called and the original request was retried
        await expect(async () => {
            expect(refreshCalled).toBe(true);
            expect(retryCalled).toBe(true);
        }).toPass({ timeout: 10000 });
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

        // Auth is cookie-based — verify redirect to login page (cookies cleared server-side)
        await expect(page.locator('#username')).toBeVisible({ timeout: 10000 });
    });
});
