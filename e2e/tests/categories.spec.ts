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

        // Create a category via API (no icon — will default to keyword fallback)
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
    // Test 3: Category with blank emoji falls back to keyword map
    // -----------------------------------------------------------------------
    test('blank emoji falls back to keyword-based icon', async ({ page }) => {
        const ts = Date.now();
        const name = `Dairy ${ts}`;

        await page.goto('/settings');

        // Leave emoji input blank, just fill name
        await page.locator('input[placeholder="New category..."]').fill(name);
        await page.click('button[title="Add a new category"]');

        // The keyword map maps "dairy" → 🥛
        const row = page.locator('.card', { hasText: name });
        await expect(row).toBeVisible({ timeout: 5000 });
        await expect(row.locator(`span:has-text("🥛 ${name}")`)).toBeVisible();

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

        // Cleanup
        await page.locator('button').filter({ hasText: 'Cancel' }).click();
        const resp = await page.request.get('/api/categories');
        const cats = await resp.json();
        const cat = cats.find((c: any) => c.name === catName);
        if (cat) await deleteCategoryViaAPI(page, cat.id);
    });
});
