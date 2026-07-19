import { test, expect, Page } from '@playwright/test';

// Covers the list-shop-route-order change: a list carries an optional shop, and
// the list view groups categories by that shop's saved aisle order without the
// viewer selecting a shop each visit.

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

async function createCategory(page: Page, name: string, sortOrder: number): Promise<string> {
    const resp = await page.request.post('/api/categories', {
        data: { name, sort_order: sortOrder },
        headers: { 'Content-Type': 'application/json' },
    });
    expect(resp.ok()).toBeTruthy();
    return (await resp.json()).id;
}

async function createShop(page: Page, name: string): Promise<string> {
    const resp = await page.request.post('/api/shops', {
        data: { name },
        headers: { 'Content-Type': 'application/json' },
    });
    expect(resp.ok()).toBeTruthy();
    return (await resp.json()).id;
}

/** Save an aisle order for a shop: categoryIds in the order they are walked. */
async function setShopOrder(page: Page, shopId: string, categoryIds: string[]) {
    const resp = await page.request.patch(`/api/shops/${shopId}/order`, {
        data: categoryIds.map((category_id, i) => ({ category_id, sort_order: i })),
        headers: { 'Content-Type': 'application/json' },
    });
    expect(resp.ok()).toBeTruthy();
}

async function addItem(page: Page, listId: string, name: string, categoryId: string) {
    const resp = await page.request.post(`/api/lists/${listId}/items`, {
        data: { name, category_id: categoryId, price: 0 },
        headers: { 'Content-Type': 'application/json' },
    });
    expect(resp.ok()).toBeTruthy();
}

/** Category group headers, top to bottom, as rendered in the list detail view. */
async function renderedGroupOrder(page: Page, names: string[]): Promise<string[]> {
    const texts = await page.locator('div').filter({ hasText: new RegExp(`^(${names.join('|')})$`) }).allTextContents();
    return texts.filter(t => names.includes(t));
}

test.describe('List shop route order', () => {
    test('list adopts its shop aisle order automatically, and falls back without a shop', async ({ page }) => {
        await loginAsManager(page);
        const ts = Date.now();

        // Two categories whose default sort order is Alpha then Beta.
        const alpha = `AAlpha ${ts}`;
        const beta = `BBeta ${ts}`;
        const alphaId = await createCategory(page, alpha, 1);
        const betaId = await createCategory(page, beta, 2);

        // A shop that is walked in the opposite order: Beta first.
        const shopId = await createShop(page, `RouteShop ${ts}`);
        await setShopOrder(page, shopId, [betaId, alphaId]);

        // A list with no shop keeps the default category order.
        const plainList = await (await page.request.post('/api/lists', {
            data: { title: `NoShop ${ts}` },
            headers: { 'Content-Type': 'application/json' },
        })).json();
        await addItem(page, plainList.id, `milk ${ts}`, alphaId);
        await addItem(page, plainList.id, `bread ${ts}`, betaId);

        await page.goto(`/list/${plainList.id}`);
        await expect(page.locator(`text=milk ${ts}`)).toBeVisible({ timeout: 10000 });
        expect(await renderedGroupOrder(page, [alpha, beta])).toEqual([alpha, beta]);

        // The same items on a list bound to the shop follow the shop's aisle order,
        // with no shop selected by hand in this session.
        const shopList = await (await page.request.post('/api/lists', {
            data: { title: `WithShop ${ts}`, shop_id: shopId },
            headers: { 'Content-Type': 'application/json' },
        })).json();
        expect(shopList.shop_id).toBe(shopId);
        await addItem(page, shopList.id, `milk2 ${ts}`, alphaId);
        await addItem(page, shopList.id, `bread2 ${ts}`, betaId);

        await page.goto(`/list/${shopList.id}`);
        await expect(page.locator(`text=milk2 ${ts}`)).toBeVisible({ timeout: 10000 });
        expect(await renderedGroupOrder(page, [alpha, beta])).toEqual([beta, alpha]);

        // Reloading keeps the order — it is persisted, not session state.
        await page.reload();
        await expect(page.locator(`text=milk2 ${ts}`)).toBeVisible({ timeout: 10000 });
        expect(await renderedGroupOrder(page, [alpha, beta])).toEqual([beta, alpha]);
    });

    test('changing the shop on a list persists across a reload', async ({ page }) => {
        await loginAsManager(page);
        const ts = Date.now();

        const shopId = await createShop(page, `PickShop ${ts}`);
        const list = await (await page.request.post('/api/lists', {
            data: { title: `Switch ${ts}` },
            headers: { 'Content-Type': 'application/json' },
        })).json();

        await page.goto(`/list/${list.id}`);
        const selector = page.locator('select').filter({ hasText: 'Default Order' });
        await expect(selector).toBeVisible({ timeout: 10000 });
        await selector.selectOption({ label: `PickShop ${ts}` });

        // The selection is written through to the list, not just held in the view.
        await expect.poll(async () => {
            const resp = await page.request.get(`/api/lists/${list.id}`);
            return (await resp.json()).shop_id;
        }, { timeout: 10000 }).toBe(shopId);

        await page.reload();
        await expect(selector).toHaveValue(shopId, { timeout: 10000 });
    });
});
