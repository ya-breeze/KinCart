import { test, expect } from '@playwright/test';

/**
 * Covers the shopper "out of stock" flow end to end:
 *   - marking an item absent sinks it into the collapsed done section
 *   - the manager sees a "Not found" badge for it, in its category group
 *   - "found it after all" flips it to bought and clears the manager badge
 *
 * The bought/absent exclusivity is enforced server-side, so the last case is
 * also the check that the invariant survives a real round trip.
 */
test.describe('Absent (out of stock) items', () => {
    const ensureManagerMode = async (page) => {
        const modeLabel = page.locator('p', { hasText: /Mode/i });
        if ((await modeLabel.textContent())?.includes('Shopper')) {
            await page.click('button:has-text("Switch to Manager")');
        }
        await expect(page.locator('p:has-text("Manager Mode")')).toBeVisible();
    };

    test.beforeEach(async ({ page }) => {
        await page.goto('/');
        await page.fill('#username', 'dad');
        await page.fill('#password', 'pass1');
        await page.click('button:has-text("Sign In")');
        await expect(page.locator('h1').first()).toHaveText('KinCart');
        await ensureManagerMode(page);
    });

    test('absent item sinks to done, badges for the manager, and clears when found', async ({ page }) => {
        const listTitle = `Absent ${Date.now()}`;

        // Create a list with two items.
        await page.click('button:has-text("New List")');
        const titleInput = page.locator('input[placeholder*="Weekly Groceries"]');
        await expect(titleInput).toBeVisible({ timeout: 5000 });
        await titleInput.fill(listTitle);
        await page.click('button:has-text("Create List")');

        const listCard = page.locator('.card', { hasText: listTitle });
        await expect(listCard).toBeVisible({ timeout: 10000 });
        await listCard.click();

        const searchInput = page.locator('input[placeholder="Add item — type, paste, or pick a chip…"]');
        await expect(searchInput).toBeVisible({ timeout: 10000 });

        const addItem = async (name: string) => {
            await searchInput.fill(name);
            await searchInput.press('Enter');
            const addToListBtn = page.locator('button:has-text("Add to List")').last();
            await expect(addToListBtn).toBeVisible({ timeout: 5000 });
            await addToListBtn.click();
            await expect(page.locator('[data-testid="item-name"]', { hasText: name })).toBeVisible({ timeout: 10000 });
        };

        await addItem('Saffron');
        await addItem('Bread');

        // No absent items yet, so the manager sees no badge.
        await expect(page.locator('[data-testid="item-not-found-badge"]')).toHaveCount(0);

        await page.click('[data-testid="status-badge"]');
        await page.click('button[title="Back to Dashboard"]');
        await page.click('button:has-text("Switch to Shopper")');
        await expect(page.locator('p:has-text("Shopper Mode")')).toBeVisible();

        await page.locator('.card', { hasText: listTitle }).first().click();

        // Mark Saffron as not available.
        const absentBtn = page.locator('button[title="Mark as not available in store"]');
        await expect(absentBtn.first()).toBeVisible({ timeout: 10000 });
        const activeBefore = await absentBtn.count();
        await absentBtn.first().click();

        // It leaves the active list and the done section appears.
        await expect(absentBtn).toHaveCount(activeBefore - 1, { timeout: 10000 });
        const doneToggle = page.locator('button:has-text("done")');
        await expect(doneToggle).toBeVisible({ timeout: 10000 });
        await expect(doneToggle).toContainText('1 done');

        // Expand it: the row is labelled "Not found" and offers both actions.
        await doneToggle.click();
        await expect(page.locator('text=Not found').first()).toBeVisible();
        const foundItBtn = page.getByRole('button', { name: /mark Saffron as bought/i });
        await expect(foundItBtn).toBeVisible();

        // Undo returns it to its category group and the done section disappears
        // (spec: "Restoring an item returns it to its category group").
        await page.getByRole('button', { name: /mark Saffron as available/i }).click();
        await expect(absentBtn).toHaveCount(activeBefore, { timeout: 10000 });
        await expect(doneToggle).toHaveCount(0);

        // Put it back to absent for the rest of the flow.
        await absentBtn.first().click();
        await expect(doneToggle).toContainText('1 done', { timeout: 10000 });
        await doneToggle.click();

        // Manager sees the "Not found" badge (task 5.2).
        await page.click('button[title="Back to Dashboard"]');
        await ensureManagerMode(page);
        await page.locator('.card', { hasText: listTitle }).first().click();
        await expect(page.locator('[data-testid="item-not-found-badge"]')).toHaveCount(1, { timeout: 10000 });

        // Back to shopper: "found it after all" (task 5.3).
        await page.click('button[title="Back to Dashboard"]');
        await page.click('button:has-text("Switch to Shopper")');
        await expect(page.locator('p:has-text("Shopper Mode")')).toBeVisible();
        await page.locator('.card', { hasText: listTitle }).first().click();

        await page.locator('button:has-text("done")').click();
        await page.getByRole('button', { name: /mark Saffron as bought/i }).click();

        // It is now bought, not absent — so the row reads "Bought" and the direct
        // bought action is gone.
        await expect(page.locator('text=Bought').first()).toBeVisible({ timeout: 10000 });
        await expect(page.getByRole('button', { name: /mark Saffron as bought/i })).toHaveCount(0);

        // And the manager badge is gone (task 5.3).
        await page.click('button[title="Back to Dashboard"]');
        await ensureManagerMode(page);
        await page.locator('.card', { hasText: listTitle }).first().click();
        await expect(page.locator('[data-testid="item-not-found-badge"]')).toHaveCount(0, { timeout: 10000 });
    });
});
