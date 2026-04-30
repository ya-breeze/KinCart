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

        // Toggle all items as bought (button title changes after each click, so always click .first())
        console.log('Toggling items...');
        const checkBtns = page.locator('button[title="Mark as bought"]');
        await expect(checkBtns.first()).toBeVisible({ timeout: 10000 });
        while ((await checkBtns.count()) > 0) {
            await checkBtns.first().click();
            // Wait for the re-render (toggle is async — fetchList is called after PATCH)
            await expect(page.locator('button[title="Mark as bought"]').first().or(
                page.locator('span:has-text("100%")')
            )).toBeVisible({ timeout: 5000 });
        }

        // Progress should update to 100%
        console.log('Checking progress...');
        await expect(page.locator('span', { hasText: '100%' })).toBeVisible({ timeout: 10000 });

        // Complete shopping
        console.log('Completing shopping...');
        await page.click('button:has-text("Complete Shopping")');

        // Verify we're redirected back to dashboard
        await expect(page).toHaveURL(/\//);
        console.log('Shopping completed successfully!');
    });

    test('create two lists back-to-back — both get unique IDs (UNIQUE constraint regression)', async ({ page }) => {
        // Regression: CreateList previously didn't set TenantModel.ID = uuid.New().
        // The first list got zero UUID (succeeded); the second hit UNIQUE constraint and
        // the "New List" button appeared to do nothing.
        const title1 = `Regression A ${Date.now()}`;
        const title2 = `Regression B ${Date.now() + 1}`;

        for (const title of [title1, title2]) {
            await page.click('button:has-text("New List")');
            const input = page.locator('input[placeholder*="Weekly Groceries"]');
            await expect(input).toBeVisible({ timeout: 5000 });
            await input.fill(title);
            await page.click('button:has-text("Create List")');
            await expect(page.locator('.card', { hasText: title })).toBeVisible({ timeout: 10000 });
        }

        // Both cards must be visible simultaneously on the dashboard
        await expect(page.locator('.card', { hasText: title1 })).toBeVisible();
        await expect(page.locator('.card', { hasText: title2 })).toBeVisible();
    });

    test('manager can delete a list (GORM zero-value delete regression)', async ({ page }) => {
        // Regression: DeleteList called db.Delete(&list) which GORM silently dropped
        // the id condition for zero-UUID primary keys, causing a 500. The fix uses
        // explicit WHERE("id = ?", listID).Delete(&models.ShoppingList{}).
        const title = `Delete Me ${Date.now()}`;

        // Create the list
        await page.click('button:has-text("New List")');
        const input = page.locator('input[placeholder*="Weekly Groceries"]');
        await expect(input).toBeVisible({ timeout: 5000 });
        await input.fill(title);
        await page.click('button:has-text("Create List")');
        const card = page.locator('.card', { hasText: title });
        await expect(card).toBeVisible({ timeout: 10000 });

        // Open the list
        await card.click();
        await expect(page.locator('h1', { hasText: title })).toBeVisible({ timeout: 5000 });

        // Click the delete button (trash icon in header)
        await page.click('button[title="Delete List"]');

        // Confirm in the modal
        await page.click('button:has-text("Confirm Delete")');

        // Should be back on dashboard and the card should be gone
        await expect(page).toHaveURL(/\/$/);
        await expect(page.locator('.card', { hasText: title })).not.toBeVisible();
    });

    test('manager can add and delete individual items', async ({ page }) => {
        // Covers: AddItemToList (non-zero UUID), DeleteItem (explicit WHERE fix),
        // and item rendering (zero-UUID category_id treated as uncategorized).
        const title = `Item Delete Test ${Date.now()}`;

        // Create list
        await page.click('button:has-text("New List")');
        const input = page.locator('input[placeholder*="Weekly Groceries"]');
        await expect(input).toBeVisible({ timeout: 5000 });
        await input.fill(title);
        await page.click('button:has-text("Create List")');
        await page.locator('.card', { hasText: title }).click();

        // Add two items
        const nameInput = page.locator('input[placeholder="e.g. Organic Bananas"]');
        await expect(nameInput).toBeVisible({ timeout: 10000 });
        for (const name of ['KeepMe', 'DeleteMe']) {
            await nameInput.fill(name);
            await page.click('button:has-text("Add Item to List")');
            await expect(page.locator('p.text-break', { hasText: name })).toBeVisible({ timeout: 10000 });
        }

        // Delete the second item
        const deleteBtn = page.locator('button[title="Delete Item"]').last();
        await deleteBtn.click();
        await page.click('button:has-text("Confirm Delete")');

        // DeleteMe should be gone; KeepMe should remain
        await expect(page.locator('p.text-break', { hasText: 'DeleteMe' })).not.toBeVisible({ timeout: 5000 });
        await expect(page.locator('p.text-break', { hasText: 'KeepMe' })).toBeVisible();
    });
});
