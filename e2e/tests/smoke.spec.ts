import { test, expect } from '@playwright/test';

test.describe('Smoke Tests', () => {
    test('homepage has correct title and login form', async ({ page }) => {
        await page.goto('/');
        await expect(page).toHaveTitle(/KinCart/);
        await expect(page.locator('h1')).toHaveText('KinCart');
        await expect(page.locator('#username')).toBeVisible();
        await expect(page.locator('#password')).toBeVisible();
    });

    test('can log in and see dashboard', async ({ page }) => {
        await page.goto('/');
        await page.fill('#username', 'dad');
        await page.fill('#password', 'pass1');
        await page.click('button:has-text("Sign In")');

        await expect(page).toHaveURL(/\//);
        await expect(page.locator('h1').first()).toHaveText('KinCart');
        // Default mode is shopper
        await expect(page.locator('p:has-text("Shopper Mode")')).toBeVisible();
    });
});
