import { test, expect, Page } from '@playwright/test';

async function loginAsManager(page: Page) {
    await page.goto('/');
    await page.fill('#username', 'dad');
    await page.fill('#password', 'pass1');
    await page.click('button:has-text("Sign In")');
    await expect(page.locator('h1').first()).toHaveText('KinCart');
    const modeLabel = page.locator('p', { hasText: /Mode/i });
    if ((await modeLabel.textContent())?.includes('Shopper')) {
        await page.click('button:has-text("Switch to Manager")');
    }
    await expect(page.locator('p:has-text("Manager Mode")')).toBeVisible();
}

async function deleteCategoryViaAPI(page: Page, catId: string) {
    await page.request.delete(`/api/categories/${catId}`);
}

async function createList(page: Page, title: string): Promise<string> {
    await page.goto('/');
    await page.click('button:has-text("New List")');
    const input = page.locator('input[placeholder*="Weekly Groceries"]');
    await expect(input).toBeVisible({ timeout: 5000 });
    await input.fill(title);
    await page.click('button:has-text("Create List")');
    await page.locator('.card', { hasText: title }).click();
    await expect(page.locator('h1', { hasText: title })).toBeVisible({ timeout: 5000 });
    return page.url().split('/list/')[1];
}

test.describe('Category emoji icons', () => {
    test.beforeEach(async ({ page }) => {
        await loginAsManager(page);
    });

    // -----------------------------------------------------------------------
    // Test 1: Create a category with a custom emoji
    // -----------------------------------------------------------------------
    test('create category with custom emoji shows emoji in list', async ({ page }) => {
        const ts = Date.now();
        const name = `Pizza ${ts}`;
        const emoji = '🍕';

        await page.goto('/settings');
        // Wait for create form to be ready
        await expect(page.locator('button[title="Add a new category"]')).toBeVisible({ timeout: 10000 });

        // Type emoji directly into the emoji input
        await page.locator('[data-testid="new-cat-emoji-input"]').fill(emoji);

        // Fill name and create
        await page.locator('input[placeholder="New category..."]').fill(name);
        await page.click('button[title="Add a new category"]');

        // Category row should show emoji + name
        const row = page.locator('.card', { hasText: name });
        await expect(row).toBeVisible({ timeout: 5000 });
        await expect(row.locator(`span:has-text("${emoji} ${name}")`)).toBeVisible();

        // Cleanup
        const resp = await page.request.get('/api/categories');
        const cats = await resp.json();
        const cat = cats.find((c: any) => c.name === name);
        if (cat) await deleteCategoryViaAPI(page, cat.id);
    });

    // -----------------------------------------------------------------------
    // Test 2: Edit an existing category's emoji
    // -----------------------------------------------------------------------
    test('edit category emoji updates the displayed icon', async ({ page }) => {
        const ts = Date.now();
        const name = `EditMe ${ts}`;
        const emoji = '🥩';

        // Create a category via API with no icon
        const createResp = await page.request.post('/api/categories', {
            data: { name, icon: '', sort_order: 999 },
            headers: { 'Content-Type': 'application/json' },
        });
        expect(createResp.ok()).toBeTruthy();
        const cat = await createResp.json();

        await page.goto('/settings');

        // Click edit on the row
        const row = page.locator('.card', { hasText: name });
        await expect(row).toBeVisible({ timeout: 5000 });
        await row.locator('button[title="Edit category name"]').click();

        // Quick-pick the new emoji
        await page.locator(`button:has-text("${emoji}")`).first().click();
        await expect(page.locator('[data-testid="cat-emoji-input"]')).toHaveValue(emoji);

        // Save
        await page.locator('button[title="Save changes"]').first().click();

        // Row should now show the new emoji
        await expect(page.locator('.card span', { hasText: `${emoji} ${name}` })).toBeVisible({ timeout: 5000 });

        await deleteCategoryViaAPI(page, cat.id);
    });

    // -----------------------------------------------------------------------
    // Test 3: Category with blank emoji shows no emoji (no keyword fallback)
    // -----------------------------------------------------------------------
    test('blank emoji shows no emoji (no keyword fallback)', async ({ page }) => {
        const ts = Date.now();
        const name = `Dairy ${ts}`;

        await page.goto('/settings');

        // Leave emoji input blank, just fill name
        await page.locator('input[placeholder="New category..."]').fill(name);
        await page.click('button[title="Add a new category"]');

        // Row should show only the name, with no emoji prefix
        const row = page.locator('.card', { hasText: name });
        await expect(row).toBeVisible({ timeout: 5000 });
        const nameSpan = row.locator('span').filter({ hasText: name }).first();
        await expect(nameSpan).toHaveText(name);

        // Cleanup
        const resp = await page.request.get('/api/categories');
        const cats = await resp.json();
        const cat = cats.find((c: any) => c.name === name);
        if (cat) await deleteCategoryViaAPI(page, cat.id);
    });

    // -----------------------------------------------------------------------
    // Test 4: Custom emoji propagates to ConfirmSheet category chips
    // -----------------------------------------------------------------------
    test('custom emoji appears in ConfirmSheet category chips', async ({ page }) => {
        const ts = Date.now();
        const catName = `Snacks ${ts}`;
        const emoji = '🍟';

        // Create category with custom emoji via Settings
        await page.goto('/settings');
        await expect(page.locator('button[title="Add a new category"]')).toBeVisible({ timeout: 10000 });
        // Type emoji directly into the emoji input
        await page.locator('[data-testid="new-cat-emoji-input"]').fill(emoji);
        await page.locator('input[placeholder="New category..."]').fill(catName);
        await page.click('button[title="Add a new category"]');
        await expect(page.locator('.card', { hasText: catName })).toBeVisible({ timeout: 5000 });

        // Open a list and trigger ConfirmSheet
        await createList(page, `Cat Emoji Test ${ts}`);
        const searchInput = page.locator('input[placeholder="Add item — type, paste, or pick a chip…"]');
        await expect(searchInput).toBeVisible({ timeout: 5000 });
        await searchInput.fill('TestItem');
        await searchInput.press('Enter');

        // ConfirmSheet should open — verify our category chip has the custom emoji
        await expect(page.locator('button:has-text("Add to List")').last()).toBeVisible({ timeout: 5000 });
        await expect(page.locator('button').filter({ hasText: `${emoji} ${catName}` })).toBeVisible({ timeout: 5000 });

        // Select the category — header emoji box should appear with the emoji
        await page.locator('button').filter({ hasText: `${emoji} ${catName}` }).click();
        await expect(page.locator('[data-testid="sheet-header-emoji"]')).toBeVisible({ timeout: 5000 });
        await expect(page.locator('[data-testid="sheet-header-emoji"]')).toHaveText(emoji);

        // Cleanup
        await page.locator('button').filter({ hasText: 'Cancel' }).click();
        const resp = await page.request.get('/api/categories');
        const cats = await resp.json();
        const cat = cats.find((c: any) => c.name === catName);
        if (cat) await deleteCategoryViaAPI(page, cat.id);
    });

    // -----------------------------------------------------------------------
    // Test 5: Category without emoji shows clean chip in ConfirmSheet (no 📦)
    // -----------------------------------------------------------------------
    test('no-emoji category shows name only in ConfirmSheet category chips', async ({ page }) => {
        const ts = Date.now();
        const catName = `NoEmoji ${ts}`;

        // Create category with no emoji via API
        const createResp = await page.request.post('/api/categories', {
            data: { name: catName, icon: '', sort_order: 999 },
            headers: { 'Content-Type': 'application/json' },
        });
        expect(createResp.ok()).toBeTruthy();
        const cat = await createResp.json();

        // Open a list and trigger ConfirmSheet
        await createList(page, `NoEmoji Test ${ts}`);
        const searchInput = page.locator('input[placeholder="Add item — type, paste, or pick a chip…"]');
        await expect(searchInput).toBeVisible({ timeout: 5000 });
        await searchInput.fill('TestItem');
        await searchInput.press('Enter');

        await expect(page.locator('button:has-text("Add to List")').last()).toBeVisible({ timeout: 5000 });

        // Header emoji box should be absent (no category selected, no draft emoji)
        await expect(page.locator('[data-testid="sheet-header-emoji"]')).not.toBeAttached();

        // Chip should show the category name with no emoji prefix
        const chip = page.locator('button').filter({ hasText: catName });
        await expect(chip).toBeVisible({ timeout: 5000 });
        await expect(chip).not.toContainText('📦');
        // Name should appear without a leading space
        await expect(chip).toHaveText(catName);

        // Cleanup
        await page.locator('button').filter({ hasText: 'Cancel' }).click();
        await deleteCategoryViaAPI(page, cat.id);
    });
});
