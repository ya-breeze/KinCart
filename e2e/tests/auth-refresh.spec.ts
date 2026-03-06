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
        await page.goto('/');
        await page.fill('#username', 'dad');
        await page.fill('#password', 'pass1');
        await page.click('button:has-text("Sign In")');

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
            // Verify it receives the initial refresh token
            const postData = route.request().postDataJSON();
            if (postData.refresh_token === 'initial-refresh-token') {
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

        // Trigger the list fetch (e.g. by navigating to lists or just reload dashboard)
        // Dashboard calls /api/lists
        await page.reload();

        // Verify refresh occurred and retry happened
        await expect(async () => {
            const token = await page.evaluate(() => localStorage.getItem('token'));
            expect(token).toBe('new-access-token');
        }).toPass();

        expect(refreshCalled).toBe(true);
        expect(retryCalled).toBe(true);
    });

    test('should log out when refresh token is invalid', async ({ page }) => {
        await page.goto('/');
        await page.fill('#username', 'dad');
        await page.fill('#password', 'pass1');
        await page.click('button:has-text("Sign In")');

        await page.route('**/api/lists', async route => {
            await route.fulfill({ status: 401 });
        });

        await page.route('**/api/auth/refresh', async route => {
            await route.fulfill({ status: 401, body: JSON.stringify({ error: 'Invalid refresh token' }) });
        });

        await page.reload();

        // Should return to login page
        await expect(page.locator('#username')).toBeVisible();
        const token = await page.evaluate(() => localStorage.getItem('token'));
        expect(token).toBeNull();
    });
});
