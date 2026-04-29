import { test, expect } from '@playwright/test';

test.describe('Shopping Lists Flow', () => {
    test.beforeEach(async ({ page }) => {
        // Login as manager (dad)
        await page.goto('/');
        await page.fill('#username', 'dad');
        await page.fill('#password', 'pass1');
        await page.click('button:has-text("Sign In")');

        // Wait for dashboard
        await expect(page.locator('h1').first()).toHaveText('KinCart');

        // Ensure we are in manager mode
        const modeLabel = page.locator('p', { hasText: /Mode/i });
        if ((await modeLabel.textContent())?.includes('Shopper')) {
            await page.click('button:has-text("Switch to Manager")');
        }
        await expect(page.locator('p:has-text("Manager Mode")')).toBeVisible();
    });

    test('Manager flows: create, add items, and prepare for shopper', async ({ page }) => {
        const listTitle = `Weekly ${Date.now()}`;

        // 1. Create List via modal ("New List" button opens a custom modal — not a browser dialog)
        await page.click('button:has-text("New List")');

        // Fill in the modal input and submit
        const titleInput = page.locator('input[placeholder*="Weekly Groceries"]');
        await expect(titleInput).toBeVisible({ timeout: 5000 });
        await titleInput.fill(listTitle);
        await page.click('button:has-text("Create List")');

        const listCard = page.locator('.card', { hasText: listTitle });
        await expect(listCard).toBeVisible({ timeout: 10000 });
        await listCard.click();

        // 2. Add three items
        const nameInput = page.locator('input[placeholder="e.g. Organic Bananas"]');
        await expect(nameInput).toBeVisible({ timeout: 10000 });

        const addItem = async (name: string, qty: string, unit: string, price: string) => {
            await nameInput.fill(name);
            await page.fill('input[placeholder="1.5"]', qty);
            const unitSelect = page.locator('select').first();
            await unitSelect.selectOption(unit);
            await page.fill('input[placeholder="$"]', price);
            await page.click('button:has-text("Add Item to List")');
            // Wait for the item to appear before adding the next one
            await expect(page.locator('p.text-break', { hasText: name })).toBeVisible({ timeout: 10000 });
        };

        await addItem('Apples', '2', 'kg', '2.50');
        await addItem('Milk', '1', 'pcs', '1.20');
        await addItem('Bread', '3', 'pcs', '0.90');

        // Verify all three items are present
        await expect(page.locator('p.text-break', { hasText: 'Apples' })).toBeVisible();
        await expect(page.locator('p.text-break', { hasText: 'Milk' })).toBeVisible();
        await expect(page.locator('p.text-break', { hasText: 'Bread' })).toBeVisible();
        await expect(page.locator('span:has-text("2 kg")')).toBeVisible();

        // 3. Mark as Ready for Shopping
        console.log('Marking as ready...');
        await page.click('button:has-text("ready")');

        // 4. Switch to Shopper and toggle
        console.log('Returning to dashboard...');
        await page.click('button[title="Back to Dashboard"]');
        console.log('Switching to shopper...');
        await page.click('button:has-text("Switch to Shopper")');
        await expect(page.locator('p:has-text("Shopper Mode")')).toBeVisible();

        // In shopper mode, find the ready list
        console.log('Finding list as shopper...');
        const shopperListCard = page.locator('.card', { hasText: listTitle });
        await expect(shopperListCard).toBeVisible({ timeout: 10000 });
        await shopperListCard.first().click();

        // Toggle item
        console.log('Toggling item...');
        const checkBtn = page.locator('button[title="Mark as bought"]');
        await expect(checkBtn).toBeVisible({ timeout: 10000 });
        await checkBtn.click();

        // Progress should update
        console.log('Checking progress...');
        await expect(page.locator('span', { hasText: '100%' })).toBeVisible({ timeout: 10000 });

        // Complete shopping
        console.log('Completing shopping...');
        await page.click('button:has-text("Complete Shopping")');

        // Verify we're redirected back to dashboard
        await expect(page).toHaveURL(/\//);
        console.log('Shopping completed successfully!');
    });
});
