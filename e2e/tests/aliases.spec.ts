import { test, expect, Page } from '@playwright/test';

// ---------------------------------------------------------------------------
// Shared helpers
// ---------------------------------------------------------------------------

async function loginAsManager(page: Page) {
    await page.goto('/');
    await page.fill('#username', 'testuser');
    await page.fill('#password', 'testpass');
    await page.click('button:has-text("Sign In")');
    await expect(page.locator('h1').first()).toHaveText('KinCart');
    const modeLabel = page.locator('p', { hasText: /Mode/i });
    if ((await modeLabel.textContent())?.includes('Shopper')) {
        await page.click('button:has-text("Switch to Manager")');
    }
    await expect(page.locator('p:has-text("Manager Mode")')).toBeVisible();
}

/** Creates a list via the UI and returns the list UUID from the URL. */
async function createList(page: Page, title: string): Promise<string> {
    // Always navigate to root — the "New List" button only exists on the dashboard
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

/** Adds a planned item via the new quick-add bar + ConfirmSheet. */
async function addPlannedItem(page: Page, name: string, price?: number) {
    const searchInput = page.locator('input[placeholder="Add item — type, paste, or pick a chip…"]');
    await expect(searchInput).toBeVisible({ timeout: 5000 });
    await searchInput.fill(name);
    await searchInput.press('Enter');
    const addToListBtn = page.locator('button:has-text("Add to List")').last();
    await expect(addToListBtn).toBeVisible({ timeout: 5000 });
    if (price !== undefined) await page.locator('input[placeholder="—"]').fill(String(price));
    await addToListBtn.click();
    await expect(page.locator('[data-testid="item-name"]', { hasText: name })).toBeVisible({ timeout: 5000 });
}

/**
 * Adds an item via the API. receipt_item_id = any non-null uint makes the
 * item treated as a "receipt item" in the link-alias UI flow.
 */
async function addItemViaAPI(page: Page, listId: string, item: object) {
    const resp = await page.request.post(`/api/lists/${listId}/items`, {
        data: item,
        headers: { 'Content-Type': 'application/json' },
    });
    expect(resp.ok()).toBeTruthy();
    return resp.json();
}

/** Creates an alias via the API and returns it. */
async function createAliasViaAPI(
    page: Page,
    plannedName: string,
    receiptName: string,
    lastPrice = 0,
) {
    const resp = await page.request.post('/api/family/aliases', {
        data: { planned_name: plannedName, receipt_name: receiptName, last_price: lastPrice },
        headers: { 'Content-Type': 'application/json' },
    });
    expect(resp.ok()).toBeTruthy();
    return resp.json();
}

/** Deletes an alias by ID via the API (test cleanup). */
async function deleteAliasViaAPI(page: Page, aliasId: number) {
    await page.request.delete(`/api/family/aliases/${aliasId}`);
}

/**
 * Expands an item row (if not already expanded) then clicks an action button by title.
 * In the new design, action buttons are only visible in the expanded panel.
 */
async function clickItemButton(page: Page, itemName: string, buttonTitle: string) {
    const nameSpan = page.locator('[data-testid="item-name"]', { hasText: itemName }).first();
    await expect(nameSpan, `Item "${itemName}" not found in list`).toBeVisible({ timeout: 5000 });
    const btn = page.locator(`button[title="${buttonTitle}"]`).first();
    // Only click name to expand if button is not already visible (avoids toggling closed)
    if (!(await btn.isVisible())) {
        await nameSpan.click();
    }
    await expect(btn).toBeVisible({ timeout: 5000 });
    await btn.click();
}

// ---------------------------------------------------------------------------
// Test suite
// ---------------------------------------------------------------------------

test.describe('Alias linking', () => {
    test.beforeEach(async ({ page }) => {
        await loginAsManager(page);
    });

    // -----------------------------------------------------------------------
    // Test 1: link receipt item to planned name — alias created, planned item removed
    // -----------------------------------------------------------------------
    test('link receipt item creates alias and removes planned item', async ({ page }) => {
        const ts = Date.now();
        const plannedName = `Chicken ${ts}`;
        const receiptName = `KURECÍ PRSA ${ts}`;

        const listId = await createList(page, `Alias T1 ${ts}`);
        await addPlannedItem(page, plannedName);

        // Add a "receipt item" via API (any non-null receipt_item_id marks it as one)
        await addItemViaAPI(page, listId, { name: receiptName, price: 89.9, receipt_item_id: 9999 });
        await page.reload();
        await expect(page.locator('[data-testid="item-name"]', { hasText: receiptName })).toBeVisible({ timeout: 5000 });

        // Click 🔗 on the receipt item
        await clickItemButton(page, receiptName, 'Link as alias');

        // Modal opens — type the planned name to surface the suggestion
        const modal = page.locator('.modal-content');
        await expect(modal).toBeVisible({ timeout: 3000 });
        await modal.locator('input').fill(plannedName);

        // Pick suggestion from the combobox list
        await modal.locator('.alias-suggestions li', { hasText: plannedName }).click();

        // Confirm button should say "Create alias & remove planned item"
        const confirmBtn = modal.locator('button:has-text("Create alias & remove planned item")');
        await expect(confirmBtn).toBeVisible();
        await confirmBtn.click();

        // Modal closes; planned item removed; receipt item remains with subtitle
        await expect(modal).not.toBeVisible({ timeout: 5000 });
        await expect(page.locator('[data-testid="item-name"]', { hasText: plannedName })).not.toBeVisible({ timeout: 5000 });
        await expect(page.locator('[data-testid="item-name"]', { hasText: receiptName })).toBeVisible();
        await expect(page.locator('[data-testid="item-alias-label"]', { hasText: `→ ${plannedName}` })).toBeVisible();

        // Verify in Settings → Aliases that the alias exists
        await page.goto('/settings');
        await page.locator('input[placeholder="Filter by item name..."]').fill(plannedName.substring(0, 7));
        // Expand the group
        await page.locator('span', { hasText: plannedName }).click();
        await expect(page.locator('span', { hasText: receiptName })).toBeVisible({ timeout: 5000 });
    });

    // -----------------------------------------------------------------------
    // Test 2: autocomplete shows known alias variants when typing item name
    // -----------------------------------------------------------------------
    test('autocomplete shows known alias variants when typing item name', async ({ page }) => {
        const ts = Date.now();
        const plannedName = `Milk ${ts}`;
        const receiptName = `PLNOTUČNÉ MLÉKO ${ts}`;

        const alias = await createAliasViaAPI(page, plannedName, receiptName, 25.9);

        await createList(page, `Alias T2 ${ts}`);

        // Type a prefix into the quick-add search bar to trigger debounced suggestion fetch (min 2 chars)
        const searchInput = page.locator('input[placeholder="Add item — type, paste, or pick a chip…"]');
        await searchInput.fill('Milk ');

        // Autocomplete dropdown appears — button shows the planned name
        const suggestionBtn = page.locator('button').filter({ hasText: plannedName });
        await expect(suggestionBtn.first()).toBeVisible({ timeout: 5000 });

        // Click the suggestion — opens the ConfirmSheet with pre-filled details
        await suggestionBtn.first().click();

        // ConfirmSheet should open with name pre-filled
        const nameInput = page.locator('[data-testid="sheet-name-input"]');
        await expect(nameInput).toBeVisible({ timeout: 5000 });
        await expect(nameInput).toHaveValue(plannedName);

        // Price field pre-filled from alias last_price
        await expect(page.locator('input[placeholder="—"]')).toHaveValue('25.9');

        await deleteAliasViaAPI(page, alias.id);
    });

    // -----------------------------------------------------------------------
    // Test 3: Receipt Variant dropdown appears and pre-fills price on change
    // -----------------------------------------------------------------------
    test('Receipt Variant cards appear in ConfirmSheet and update price on selection', async ({ page }) => {
        const ts = Date.now();
        const plannedName = `Yogurt ${ts}`;
        const v1 = { receipt: `SELSKÝ JOGURT 2% ${ts}`, price: 19.9 };
        const v2 = { receipt: `BIO JOGURT ${ts}`, price: 34.5 };

        const a1 = await createAliasViaAPI(page, plannedName, v1.receipt, v1.price);
        const a2 = await createAliasViaAPI(page, plannedName, v2.receipt, v2.price);

        await createList(page, `Alias T3 ${ts}`);

        // Type the full planned name to trigger autocomplete
        const searchInput = page.locator('input[placeholder="Add item — type, paste, or pick a chip…"]');
        await searchInput.fill(plannedName);

        // Autocomplete suggestion appears — click it to open ConfirmSheet with both variants
        const suggestionBtn = page.locator('button').filter({ hasText: plannedName });
        await expect(suggestionBtn.first()).toBeVisible({ timeout: 5000 });
        await suggestionBtn.first().click();

        // ConfirmSheet should show receipt variant cards for both aliases
        const v1Card = page.locator('button').filter({ hasText: v1.receipt });
        const v2Card = page.locator('button').filter({ hasText: v2.receipt });
        await expect(v1Card).toBeVisible({ timeout: 5000 });
        await expect(v2Card).toBeVisible();

        // First variant is auto-selected — price is pre-filled from v1
        await expect(page.locator('input[placeholder="—"]')).toHaveValue(String(v1.price));

        // Click variant 2 card — price updates to v2's price
        await v2Card.click();
        await expect(page.locator('input[placeholder="—"]')).toHaveValue(String(v2.price));

        await deleteAliasViaAPI(page, a1.id);
        await deleteAliasViaAPI(page, a2.id);
    });

    // -----------------------------------------------------------------------
    // Test 4 (regression): edit modal name change clears preferred_alias_id
    // Bug: changing item name in edit modal used to send a stale alias ID
    // -----------------------------------------------------------------------
    test('changing item name in edit modal clears the preferred alias', async ({ page }) => {
        const ts = Date.now();
        const plannedName = `Bread ${ts}`;
        const receiptName = `TOASTOVÝ CHLÉB ${ts}`;

        const alias = await createAliasViaAPI(page, plannedName, receiptName, 45.0);
        const listId = await createList(page, `Alias T4 ${ts}`);
        await addPlannedItem(page, plannedName);

        // Open edit modal for the item
        await clickItemButton(page, plannedName, 'Edit Item');

        const editModal = page.locator('.modal-content');
        await expect(editModal.locator('h2:has-text("Edit Item")')).toBeVisible({ timeout: 3000 });

        // Receipt Variant dropdown should be visible (item name has known aliases)
        await expect(editModal.locator('select').filter({ has: page.locator(`option[value="${alias.id}"]`) })).toBeVisible();

        // Change the name to something with no aliases
        const newName = `Renamed ${ts}`;
        // Label and input are adjacent siblings — use CSS sibling selector to be precise
        const nameInput = editModal.locator('label.input-label:has-text("Item Name") + input');
        await nameInput.fill(newName);
        await expect(nameInput).toHaveValue(newName);

        // Receipt Variant dropdown should disappear (new name has no aliases)
        await expect(editModal.locator('select').filter({ has: page.locator(`option[value="${alias.id}"]`) })).not.toBeVisible({ timeout: 3000 });

        // Save
        await editModal.locator('button:has-text("Save Changes")').click();
        await expect(editModal).not.toBeVisible({ timeout: 5000 });

        // Verify item has new name and no alias subtitle
        await expect(page.locator('[data-testid="item-name"]', { hasText: newName })).toBeVisible({ timeout: 5000 });
        await expect(page.locator('[data-testid="item-name"]', { hasText: plannedName })).not.toBeVisible();
        // No alias subtitle for the renamed item (no alias for the new name)
        await expect(page.locator('[data-testid="item-alias-label"]')).not.toBeVisible();

        // Verify via API: preferred_alias_id should be null on the saved item
        const listResp = await page.request.get(`/api/lists/${listId}`);
        const listData = await listResp.json();
        const updatedItem = (listData.items ?? []).find((i: any) => i.name === newName);
        expect(updatedItem, `Item "${newName}" not found in API response`).toBeTruthy();
        expect(updatedItem.preferred_alias_id).toBeNull();

        await deleteAliasViaAPI(page, alias.id);
    });

    // -----------------------------------------------------------------------
    // Test 5 (regression): re-linking uses Remove + Create (no data loss)
    // Bug: old code deleted aliases BEFORE the POST; if POST failed, old
    // aliases were permanently lost. Fix: use Remove button then create fresh.
    // -----------------------------------------------------------------------
    test('re-linking a receipt item: Remove old alias then create new one', async ({ page }) => {
        const ts = Date.now();
        const listId = await createList(page, `Alias T5 ${ts}`);
        const plannedA = `PlanA ${ts}`;
        const plannedB = `PlanB ${ts}`;
        const receiptName = `RECEIPT ITEM ${ts}`;

        await addPlannedItem(page, plannedA);
        await addPlannedItem(page, plannedB);
        await addItemViaAPI(page, listId, { name: receiptName, price: 10.0, receipt_item_id: 9998 });
        await page.reload();
        await expect(page.locator('[data-testid="item-name"]', { hasText: receiptName })).toBeVisible({ timeout: 5000 });

        // — Step 1: Link receipt item → plannedA —
        await clickItemButton(page, receiptName, 'Link as alias');
        const modal = page.locator('.modal-content');
        await expect(modal).toBeVisible({ timeout: 3000 });
        await modal.locator('input').fill(plannedA);
        await modal.locator('.alias-suggestions li', { hasText: plannedA }).click();
        await modal.locator('button:has-text("Create alias & remove planned item")').click();
        await expect(modal).not.toBeVisible({ timeout: 5000 });

        // plannedA removed, subtitle shows → plannedA
        await expect(page.locator('[data-testid="item-name"]', { hasText: plannedA })).not.toBeVisible({ timeout: 5000 });
        await expect(page.locator('[data-testid="item-alias-label"]', { hasText: `→ ${plannedA}` })).toBeVisible();

        // — Step 2: Re-open modal and Remove the old alias —
        await clickItemButton(page, receiptName, 'Link as alias');
        await expect(modal).toBeVisible({ timeout: 3000 });

        // Existing link shown with Remove button
        await expect(modal.locator('span', { hasText: plannedA })).toBeVisible();
        await modal.locator('button:has-text("Remove")').click();

        // Modal closes after removal
        await expect(modal).not.toBeVisible({ timeout: 5000 });
        // Subtitle gone (alias deleted)
        await expect(page.locator('[data-testid="item-alias-label"]', { hasText: `→ ${plannedA}` })).not.toBeVisible({ timeout: 5000 });

        // — Step 3: Link receipt item → plannedB —
        await clickItemButton(page, receiptName, 'Link as alias');
        await expect(modal).toBeVisible({ timeout: 3000 });
        // No "currently linked" section now
        await expect(modal.locator('span', { hasText: plannedA })).not.toBeVisible();
        await modal.locator('input').fill(plannedB);
        await modal.locator('.alias-suggestions li', { hasText: plannedB }).click();
        await modal.locator('button:has-text("Create alias & remove planned item")').click();
        await expect(modal).not.toBeVisible({ timeout: 5000 });

        // plannedB removed from list; subtitle shows → plannedB
        await expect(page.locator('[data-testid="item-name"]', { hasText: plannedB })).not.toBeVisible({ timeout: 5000 });
        await expect(page.locator('[data-testid="item-alias-label"]', { hasText: `→ ${plannedB}` })).toBeVisible();
        await expect(page.locator('[data-testid="item-alias-label"]', { hasText: `→ ${plannedA}` })).not.toBeVisible();

        // API: exactly one alias for this receipt name
        const aliasesResp = await page.request.get('/api/family/aliases');
        const aliases: any[] = await aliasesResp.json();
        const forReceipt = aliases.filter((a: any) => a.receipt_name === receiptName);
        expect(forReceipt).toHaveLength(1);
        expect(forReceipt[0].planned_name).toBe(plannedB);

        await deleteAliasViaAPI(page, forReceipt[0].id);
    });

    // -----------------------------------------------------------------------
    // Test 6: editing alias receipt_name in Settings updates autocomplete
    // -----------------------------------------------------------------------
    test('editing alias receipt_name in Settings updates autocomplete search', async ({ page }) => {
        const ts = Date.now();
        const plannedName = `Butter ${ts}`;
        const originalReceipt = `MÁSLO 250G ${ts}`;
        const updatedReceipt = `JIHOČESKÉ MÁSLO ${ts}`;

        await createAliasViaAPI(page, plannedName, originalReceipt, 55.0);

        // Navigate to Settings → Aliases
        await page.goto('/settings');
        const filterInput = page.locator('input[placeholder="Filter by item name..."]');
        await expect(filterInput).toBeVisible({ timeout: 5000 });

        // Filter to show only our alias group
        await filterInput.fill(plannedName.substring(0, 7));

        // Expand the group by clicking its header
        await page.locator('span', { hasText: plannedName }).first().click();

        // Alias variant row is now visible — click its Edit button
        await expect(page.locator('span', { hasText: originalReceipt })).toBeVisible({ timeout: 5000 });

        // Find the variant row that contains both the receipt name span and an Edit button
        const variantRow = page.locator('div').filter({
            has: page.locator('span', { hasText: originalReceipt }),
        }).filter({
            has: page.locator('button[title="Edit"]'),
        }).last();
        await variantRow.locator('button[title="Edit"]').click();

        // The inline edit shows an auto-focused input with the current receipt name
        const editInput = page.locator('input[value="' + originalReceipt + '"]').or(
            page.locator('input').filter({ hasText: '' }).first()
        );
        // More reliably: the first focused input in the alias row
        const activeInput = page.locator('input:focus');
        await expect(activeInput).toBeVisible({ timeout: 3000 });
        await activeInput.fill(updatedReceipt);

        await page.locator('button[title="Save"]').click();

        // Row now shows updated receipt name
        await expect(page.locator('span', { hasText: updatedReceipt })).toBeVisible({ timeout: 5000 });
        await expect(page.locator('span', { hasText: originalReceipt })).not.toBeVisible();

        // Navigate to a list and verify autocomplete reflects the update
        await createList(page, `Alias T6 ${ts}`);
        const searchInput = page.locator('input[placeholder="Add item — type, paste, or pick a chip…"]');
        await searchInput.fill(plannedName.substring(0, 7));

        // Suggestion should show updated receipt name in its subtitle, not the old one
        await expect(page.locator('button').filter({ hasText: updatedReceipt })).toBeVisible({ timeout: 5000 });
        await expect(page.locator('button').filter({ hasText: originalReceipt })).not.toBeVisible();

        // Cleanup
        const aliasesResp = await page.request.get('/api/family/aliases');
        const aliases: any[] = await aliasesResp.json();
        const updated = aliases.find((a: any) => a.planned_name === plannedName);
        if (updated) await deleteAliasViaAPI(page, updated.id);
    });

    // -----------------------------------------------------------------------
    // Test 7: cross-family isolation — aliases not visible to other families
    // Skipped automatically if a second family user is not seeded.
    // -----------------------------------------------------------------------
    test('aliases are not visible across families', async ({ page, browser }) => {
        const ts = Date.now();
        const plannedName = `Eggs ${ts}`;
        const receiptName = `VEJCE M 30ks ${ts}`;

        // Create alias as the primary user
        const alias = await createAliasViaAPI(page, plannedName, receiptName, 89.0);

        // Try to login as a second family user
        const ctx2 = await browser.newContext();
        const page2 = await ctx2.newPage();
        await page2.goto('/');
        await page2.fill('#username', 'mom');
        await page2.fill('#password', 'pass2');
        await page2.click('button:has-text("Sign In")');

        const loginOk = await page2.locator('h1', { hasText: 'KinCart' })
            .waitFor({ timeout: 3000 })
            .then(() => true)
            .catch(() => false);

        if (!loginOk) {
            test.info().annotations.push({
                type: 'note',
                description: 'Second family user (mom/pass2) not seeded — cross-family isolation check skipped',
            });
            await ctx2.close();
            await deleteAliasViaAPI(page, alias.id);
            return;
        }

        // Second user logged in — verify the alias from the primary family is NOT visible
        const aliasesResp = await page2.request.get('/api/family/aliases');
        const rawAliases = await aliasesResp.json();
        const aliases: any[] = Array.isArray(rawAliases) ? rawAliases : [];
        const leaked = aliases.find((a: any) =>
            a.planned_name === plannedName && a.receipt_name === receiptName,
        );
        expect(leaked, 'Alias leaked across family boundary').toBeUndefined();

        await ctx2.close();
        await deleteAliasViaAPI(page, alias.id);
    });
});
