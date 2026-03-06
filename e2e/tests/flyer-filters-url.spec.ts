import { test, expect } from '@playwright/test';

test.describe('Flyer Filters URL Synchronization', () => {
    test.beforeEach(async ({ page }) => {
        // Login
        await page.goto('/');
        await page.fill('#username', 'dad');
        await page.fill('#password', 'pass1');
        await page.click('button:has-text("Sign In")');

        // Ensure manager mode
        await expect(page.locator('h1').first()).toHaveText('KinCart');
        const modeLabel = page.locator('p', { hasText: /Mode/i });
        if ((await modeLabel.textContent())?.includes('Shopper')) {
            await page.click('button:has-text("Switch to Manager")');
        }
    });

    test('UI interactions update the URL correctly', async ({ page }) => {
        await page.click('button:has-text("Flyer Items")');
        await expect(page).toHaveURL(/\/flyers/);

        // Filter by search query
        await page.fill('input[name="q"]', 'milk');
        await page.waitForTimeout(500); // Wait for debounce/sync
        await expect(page).toHaveURL(/q=milk/);

        // Filter by shop
        await page.selectOption('select[name="shop"]', 'Lidl');
        await expect(page).toHaveURL(/shop=Lidl/);
        await expect(page).toHaveURL(/q=milk/); // q should persist

        // Filter by activity
        await page.selectOption('select[name="activity"]', 'future');
        await expect(page).toHaveURL(/activity=future/);

        // Reset search query via the input's clear button
        // The clear button is the only button inside the relative container after the input
        await page.locator('.input-group', { hasText: 'Search' }).locator('button').click();
        await expect(page).not.toHaveURL(/q=milk/);
        await expect(page).toHaveURL(/shop=Lidl/);
        await expect(page).toHaveURL(/activity=future/);

        // Reset activity to 'now'
        await page.selectOption('select[name="activity"]', 'now');
        await expect(page).not.toHaveURL(/activity=now/); // Should be removed from URL as it is the default
    });

    test('deep links correctly restore filter state', async ({ page }) => {
        // Navigate directly with query params
        // Using 'Lidl' as it's known to exist in E2E seed data
        await page.goto('/flyers?q=cheese&shop=Lidl&activity=all');

        // Check search input
        await expect(page.locator('input[name="q"]')).toHaveValue('cheese');

        // Wait for shop options to load from API (attached to DOM, not necessarily visible)
        await page.waitForSelector('select[name="shop"] option[value="Lidl"]', { state: 'attached', timeout: 10000 });
        await expect(page.locator('select[name="shop"]')).toHaveValue('Lidl');

        await expect(page.locator('select[name="activity"]')).toHaveValue('all');
    });
});
