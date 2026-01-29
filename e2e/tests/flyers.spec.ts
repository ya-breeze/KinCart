import { test, expect } from '@playwright/test';

test.describe('Flyer Interactions', () => {
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

    test('Filter and add flyer item to new list', async ({ page }) => {
        await page.click('button:has-text("Flyer Items")');
        await expect(page).toHaveURL(/\/flyers/);

        // Filter by shop
        await page.selectOption('select[name="shop"]', 'Lidl');

        // Wait for results
        const flyerCard = page.locator('.card').filter({ has: page.locator('span:has-text("Lidl")') }).first();
        await expect(flyerCard).toBeVisible({ timeout: 20000 });

        const itemNameLocator = flyerCard.locator('h4');
        await expect(itemNameLocator).toBeVisible();
        const itemName = await itemNameLocator.textContent();

        // Add to list
        console.log('Opening list selector...');
        await flyerCard.locator('button:has-text("Add to List")').click();

        const newListTitle = `Flyer List ${Date.now()}`;
        console.log('Setting up dialog handler...');
        page.on('dialog', async dialog => {
            console.log('Dialog detected, accepting with:', newListTitle);
            await dialog.accept(newListTitle);
        });

        console.log('Clicking Create New List...');
        await page.locator('button:has-text("Create New List")').last().click();

        // Success message
        console.log('Waiting for success message...');
        await expect(page.locator('.badge-success')).toBeVisible({ timeout: 10000 });

        // Verify in dashboard
        await page.click('button:has-text("Back to Dashboard")');
        const listCard = page.locator('.card', { hasText: newListTitle });
        await expect(listCard).toBeVisible();
        await listCard.click();

        await expect(page.locator('p', { hasText: itemName || "" })).toBeVisible();
        await expect(page.locator('span:has-text("Sale Deal")')).toBeVisible();
    });
});
