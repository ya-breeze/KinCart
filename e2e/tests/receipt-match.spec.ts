import { test, expect, Page } from '@playwright/test';

// Receipt parsing via Gemini can take 20-40 seconds; give each test 2 minutes.
test.setTimeout(120000);

// ---------------------------------------------------------------------------
// Shared helpers
// ---------------------------------------------------------------------------

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

async function createListViaAPI(page: Page, name: string): Promise<{ id: string }> {
    const resp = await page.request.post('/api/lists', {
        data: { name },
        headers: { 'Content-Type': 'application/json' },
    });
    expect(resp.ok()).toBeTruthy();
    return resp.json();
}

async function addItemViaAPI(page: Page, listId: string, name: string): Promise<{ id: string }> {
    const resp = await page.request.post(`/api/lists/${listId}/items`, {
        data: { name, price: 0 },
        headers: { 'Content-Type': 'application/json' },
    });
    expect(resp.ok()).toBeTruthy();
    return resp.json();
}

async function patchItem(page: Page, itemId: string, data: object) {
    const resp = await page.request.patch(`/api/items/${itemId}`, {
        data,
        headers: { 'Content-Type': 'application/json' },
    });
    expect(resp.ok()).toBeTruthy();
}

async function patchList(page: Page, listId: string, data: object) {
    const resp = await page.request.patch(`/api/lists/${listId}`, {
        data,
        headers: { 'Content-Type': 'application/json' },
    });
    expect(resp.ok()).toBeTruthy();
}

async function createAliasViaAPI(page: Page, plannedName: string, receiptName: string) {
    const resp = await page.request.post('/api/family/aliases', {
        data: { planned_name: plannedName, receipt_name: receiptName, last_price: 0 },
        headers: { 'Content-Type': 'application/json' },
    });
    expect(resp.ok()).toBeTruthy();
}

/** Upload receipt as text and wait for the upload response. Returns the fetch response. */
async function uploadReceiptText(page: Page, text: string) {
    await page.locator('[data-testid="tab-paste"]').click();
    await page.locator('textarea').fill(text);
    const [response] = await Promise.all([
        page.waitForResponse(resp => resp.url().includes('/receipts'), { timeout: 60000 }),
        page.locator('button:has-text("Process Receipt")').click(),
    ]);
    return response;
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

test.describe('Receipt matching', () => {
    test.beforeEach(async ({ page }) => {
        await loginAsManager(page);
    });

    // -----------------------------------------------------------------------
    // Test 1: already-bought items appear as link options in match modal
    // Also verifies the core bug fix: items bought before upload are still
    // matched via aliases instead of becoming duplicates.
    // -----------------------------------------------------------------------
    test('already_bought_items appear as link options in match modal', async ({ page }) => {
        const ts = Date.now();
        const appleReceipt = `KINCART-APPLE-${ts}`;
        const applePlanned = `Apple ${ts}`;
        const orangePlanned = `Orange ${ts}`;
        const breadPlanned = `Bread ${ts}`;
        const extraReceipt = `KINCART-EXTRA-${ts}`;

        // Create list with 3 planned items
        const list = await createListViaAPI(page, `RM Test 1 ${ts}`);
        const appleItem = await addItemViaAPI(page, list.id, applePlanned);
        const orangeItem = await addItemViaAPI(page, list.id, orangePlanned);
        await addItemViaAPI(page, list.id, breadPlanned); // bread is NOT bought

        // Alias for apple only — ensures deterministic auto-match
        await createAliasViaAPI(page, applePlanned, appleReceipt);

        // Mark apple and orange as bought (simulating manual check-off during shopping)
        await patchItem(page, appleItem.id, { is_bought: true });
        await patchItem(page, orangeItem.id, { is_bought: true });

        // Complete the list to enable the receipt upload button
        await patchList(page, list.id, { name: `RM Test 1 ${ts}`, status: 'completed' });

        // Navigate to the list and open the receipt upload dialog
        await page.goto(`/list/${list.id}`);
        await page.locator('button[title="Upload receipt"]').click();
        await expect(page.locator('text=Upload Receipt')).toBeVisible({ timeout: 5000 });

        // Upload receipt text: first item matches apple via alias, second is extra
        await uploadReceiptText(page, `${appleReceipt} 3.50\n${extraReceipt} 5.00`);

        // Wait for the match modal to appear
        // (pending_review: extra item is unmatched + bread is unbought)
        const confirmBtn = page.locator('[data-testid="confirm-all-btn"]');
        await expect(confirmBtn).toBeVisible({ timeout: 60000 });
        await expect(page.locator('h2:has-text("Review Receipt")')).toBeVisible();

        // Confirm button should show "still need a decision" (EXTRA item unresolved)
        await expect(confirmBtn).toHaveText(/still need a decision/);

        // Expand the first "Show options" row — this is the EXTRA item in "Not on your list"
        await page.locator('button[title="Show options"]').first().click();

        // Orange (bought, not linked to any receipt item) should appear under "already bought"
        await expect(page.locator('text=already bought')).toBeVisible({ timeout: 5000 });
        const orangeBtn = page.locator(`button:has-text("${orangePlanned}")`);
        await expect(orangeBtn).toBeVisible({ timeout: 3000 });

        // Bread (not bought, unmatched) should appear in the planned items section
        await expect(page.locator(`button:has-text("${breadPlanned}")`)).toBeVisible();

        // Link the EXTRA receipt item to Orange
        await orangeBtn.click();

        // Confirm button should now be ready
        await expect(confirmBtn).not.toHaveText(/still need/);

        // Confirm all
        await confirmBtn.click();
        await expect(confirmBtn).not.toBeVisible({ timeout: 10000 });

        // Verify list still has exactly 3 items (no duplicates)
        await page.goto(`/list/${list.id}`);
        await page.waitForLoadState('networkidle');
        const itemCount = await page.locator('[data-testid="item-name"]').count();
        expect(itemCount).toBe(3);
    });

    // -----------------------------------------------------------------------
    // Test 2: no duplicates when all items were manually bought before upload
    // This is the core regression test for the duplicate-items bug.
    // -----------------------------------------------------------------------
    test('no duplicate items when all items manually bought before receipt upload', async ({ page }) => {
        const ts = Date.now();
        const appleReceipt = `KINCART-APPLE2-${ts}`;
        const applePlanned = `Apple2 ${ts}`;

        const list = await createListViaAPI(page, `RM Test 2 ${ts}`);
        const appleItem = await addItemViaAPI(page, list.id, applePlanned);
        await createAliasViaAPI(page, applePlanned, appleReceipt);

        // Manually mark item as bought (as if shopper ticked it during shopping)
        await patchItem(page, appleItem.id, { is_bought: true });

        // Complete the list
        await patchList(page, list.id, { name: `RM Test 2 ${ts}`, status: 'completed' });

        // Open upload dialog and process receipt
        await page.goto(`/list/${list.id}`);
        await page.locator('button[title="Upload receipt"]').click();
        await expect(page.locator('text=Upload Receipt')).toBeVisible({ timeout: 5000 });

        const uploadResp = await uploadReceiptText(page, `${appleReceipt} 3.50`);
        const uploadData = await uploadResp.json();

        if (uploadData.status === 'pending_review') {
            // Match modal appeared: confirm immediately (should not block)
            const confirmBtn = page.locator('[data-testid="confirm-all-btn"]');
            await expect(confirmBtn).toBeVisible({ timeout: 10000 });
            await expect(confirmBtn).not.toHaveText(/still need a decision/);
            await confirmBtn.click();
            await expect(confirmBtn).not.toBeVisible({ timeout: 10000 });
        } else {
            // "parsed" — auto-matched, "Done!" shows and auto-closes
            // Wait for upload modal to close (Done! auto-closes after 1.5s)
            await expect(page.locator('h2:has-text("Upload Receipt")')).not.toBeVisible({ timeout: 10000 });
        }

        // Navigate to list and verify: exactly 1 item, no duplicates
        await page.goto(`/list/${list.id}`);
        await page.waitForLoadState('networkidle');
        const itemCount = await page.locator('[data-testid="item-name"]').count();
        expect(itemCount).toBe(1);
    });

    // -----------------------------------------------------------------------
    // Test 3: undo and reset work in the match modal
    // -----------------------------------------------------------------------
    test('undo and reset work in match modal', async ({ page }) => {
        const ts = Date.now();
        const appleReceipt = `KINCART-APPLE3-${ts}`;
        const applePlanned = `Apple3 ${ts}`;
        const breadPlanned = `Bread3 ${ts}`;

        const list = await createListViaAPI(page, `RM Test 3 ${ts}`);
        await addItemViaAPI(page, list.id, applePlanned);
        await addItemViaAPI(page, list.id, breadPlanned);
        await createAliasViaAPI(page, applePlanned, appleReceipt);
        await patchList(page, list.id, { name: `RM Test 3 ${ts}`, status: 'completed' });

        // Upload receipt with one matched item and one extra
        await page.goto(`/list/${list.id}`);
        await page.locator('button[title="Upload receipt"]').click();
        await expect(page.locator('text=Upload Receipt')).toBeVisible({ timeout: 5000 });
        await uploadReceiptText(page, `${appleReceipt} 3.50\nKINCART-EXTRA3-${ts} 5.00`);

        // Wait for match modal
        const confirmBtn = page.locator('[data-testid="confirm-all-btn"]');
        await expect(confirmBtn).toBeVisible({ timeout: 60000 });

        // Initially confirm is blocked (EXTRA item needs a decision)
        await expect(confirmBtn).toHaveText(/still need a decision/);

        // Expand the EXTRA item (first "Show options" button)
        await page.locator('button[title="Show options"]').first().click();

        // Click "Add as new"
        await page.locator('button:has-text("Add as new")').click();

        // Undo button should now be visible; confirm should say "add 1 item"
        await expect(page.locator('button:has-text("Undo")')).toBeVisible();
        await expect(confirmBtn).toHaveText(/add 1 item/);

        // Click Undo — decision is reverted
        await page.locator('button:has-text("Undo")').click();

        // Confirm should go back to blocked
        await expect(confirmBtn).toHaveText(/still need a decision/);
        // Undo button should disappear (no more history)
        await expect(page.locator('button:has-text("Undo")')).not.toBeVisible();

        // Now skip the extra item
        await page.locator('button[title="Show options"]').first().click();
        await page.locator('button:has-text("Skip")').click();

        // Reset button should be visible now; confirm should be ready
        await expect(page.locator('button:has-text("Reset")')).toBeVisible();
        await expect(confirmBtn).toHaveText(/Confirm & done/);

        // Click Reset
        await page.locator('button:has-text("Reset")').click();

        // Confirm blocked again; Reset gone
        await expect(confirmBtn).toHaveText(/still need a decision/);
        await expect(page.locator('button:has-text("Reset")')).not.toBeVisible();

        // Finally: skip and confirm
        await page.locator('button[title="Show options"]').first().click();
        await page.locator('button:has-text("Skip")').click();
        await expect(confirmBtn).toHaveText(/Confirm & done/);
        await confirmBtn.click();
        await expect(confirmBtn).not.toBeVisible({ timeout: 10000 });

        // Verify: list still has 2 items (apple + bread), no EXTRA added
        await page.goto(`/list/${list.id}`);
        await page.waitForLoadState('networkidle');
        const itemCount = await page.locator('[data-testid="item-name"]').count();
        expect(itemCount).toBe(2);
    });
});
