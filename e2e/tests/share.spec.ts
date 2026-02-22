import { test, expect } from '@playwright/test';

test.describe('PWA Share Target Flow', () => {
    test.beforeEach(async ({ page }) => {
        // Log in first as it's a protected route
        console.log('Logging in...');
        await page.goto('/');

        // Unregister service workers to avoid stale cache issues during E2E
        await page.evaluate(async () => {
            if ('serviceWorker' in navigator) {
                const registrations = await navigator.serviceWorker.getRegistrations();
                for (const reg of registrations) {
                    await reg.unregister();
                }
            }
            localStorage.clear();
        });

        await page.fill('#username', 'dad');
        await page.fill('#password', 'pass1');
        await page.click('button:has-text("Sign In")');

        // Wait for the Dashboard to appear and token to be in localStorage
        await expect(page.locator('h1').first()).toHaveText('KinCart');
        await page.waitForFunction(() => !!localStorage.getItem('token'));
        console.log('Login confirmed, token is in localStorage.');
    });

    test('successfully imports a shared receipt', async ({ page }) => {
        // 1. Simulate the Service Worker intercepting a share by injecting data into IndexedDB
        console.log('Injecting data into IndexedDB...');
        await page.evaluate(async () => {
            return new Promise((resolve, reject) => {
                const request = indexedDB.open('KinCart');

                request.onupgradeneeded = (event: any) => {
                    const db = event.target.result;
                    if (!db.objectStoreNames.contains('shared_files')) {
                        db.createObjectStore('shared_files');
                    }
                };

                request.onsuccess = (event: any) => {
                    const db = event.target.result;
                    if (!db.objectStoreNames.contains('shared_files')) {
                        db.close();
                        reject('Store shared_files not found');
                        return;
                    }

                    const transaction = db.transaction(['shared_files'], 'readwrite');
                    const store = transaction.objectStore('shared_files');

                    const testFile = {
                        name: 'receipt.png',
                        type: 'image/png',
                        blob: new Blob(['fake image data'], { type: 'image/png' }),
                        timestamp: Date.now()
                    };

                    store.put([testFile], 'pending_shared_receipts');

                    transaction.oncomplete = () => {
                        db.close();
                        resolve(true);
                    };
                    transaction.onerror = () => reject('IDB transaction error');
                };
                request.onerror = () => reject('IDB open error');
            });
        });

        // 2. Navigate to the import page
        console.log('Navigating to /import-receipt?shared=true');
        await page.goto('/import-receipt?shared=true');
        console.log('Current URL:', page.url());

        // 3. Verify the page content
        const h1 = page.locator('h1');

        // Increasing timeout and adding a small wait to allow for hydration/context re-init
        await page.waitForTimeout(500);

        const h1Text = await h1.innerText();
        console.log('H1 text found:', h1Text);

        if (h1Text === 'KinCart') {
            const bodyTxt = await page.innerText('body');
            console.log('Body snippet:', bodyTxt.substring(0, 500));
            console.log('LocalStorage token:', await page.evaluate(() => localStorage.getItem('token') ? 'exists' : 'missing'));
        }

        await expect(h1).toHaveText('Import Shared Receipt', { timeout: 15000 });
        await expect(page.locator('text=1 file detected')).toBeVisible();

        // 4. Create a new list for this receipt
        await page.click('text=+ Create a new list instead');
        await page.fill('input[placeholder="e.g. Weekly Groceries"]', 'E2E Shared List');

        // 5. Submit
        await page.click('button:has-text("Confirm & Import")');

        // 6. Verify we are redirected to the list detail page
        await expect(page).toHaveURL(/\/list\/\d+/);
        await expect(page.locator('h1')).toHaveText('E2E Shared List');

        // 7. Verify IndexedDB is cleaned up
        const isCleaned = await page.evaluate(async () => {
            return new Promise((resolve) => {
                const request = indexedDB.open('KinCart');
                request.onsuccess = (event: any) => {
                    const db = event.target.result;
                    if (!db.objectStoreNames.contains('shared_files')) {
                        resolve(true);
                        return;
                    }
                    const transaction = db.transaction(['shared_files'], 'readonly');
                    const store = transaction.objectStore('shared_files');
                    const getReq = store.get('pending_shared_receipts');
                    getReq.onsuccess = () => {
                        const result = getReq.result;
                        db.close();
                        resolve(result === undefined);
                    };
                    getReq.onerror = () => resolve(true);
                };
                request.onerror = () => resolve(true);
            });
        });
        expect(isCleaned).toBe(true);
    });
});
