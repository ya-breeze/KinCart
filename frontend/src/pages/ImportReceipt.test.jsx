/** @vitest-environment happy-dom */
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import ImportReceipt from './ImportReceipt';
import { BrowserRouter } from 'react-router-dom';

// Mock URL.createObjectURL
globalThis.URL.createObjectURL = vi.fn(() => 'blob:test');

// Mock localforage
const mockSharedFiles = [
    { name: 'receipt.jpg', type: 'image/jpeg', blob: new Blob(['contents'], { type: 'image/jpeg' }), timestamp: Date.now() }
];

vi.mock('localforage', () => ({
    default: {
        config: vi.fn(),
        getItem: vi.fn((key) => {
            if (key === 'pending_shared_receipts') return Promise.resolve(mockSharedFiles);
            return Promise.resolve(null);
        }),
        removeItem: vi.fn(() => Promise.resolve()),
    }
}));

// Mock AuthContext
vi.mock('../context/AuthContext', () => ({
    useAuth: () => ({
        token: 'fake-token',
    }),
}));

describe('ImportReceipt', () => {
    beforeEach(() => {
        vi.clearAllMocks();
        globalThis.fetch = vi.fn((url) => {
            if (typeof url === 'string' && url.includes('/api/lists')) {
                return Promise.resolve({
                    ok: true,
                    json: () => Promise.resolve([
                        { id: 1, title: 'Weekly Groceries' },
                        { id: 2, title: 'Household' }
                    ]),
                });
            }
            return Promise.resolve({ ok: true, json: () => Promise.resolve({}) });
        });
    });

    it('renders shared file info and list of available shopping lists', async () => {
        render(
            <BrowserRouter>
                <ImportReceipt />
            </BrowserRouter>
        );

        expect(await screen.findByText('1 file detected')).toBeInTheDocument();
        expect(await screen.findByText('Weekly Groceries')).toBeInTheDocument();
        expect(await screen.findByText('Household')).toBeInTheDocument();
    });

    it('uploads the file to the selected list', async () => {
        render(
            <BrowserRouter>
                <ImportReceipt />
            </BrowserRouter>
        );

        // Initial render might auto-select if only one list, but here we have two
        const listBtn = await screen.findByText('Household');
        fireEvent.click(listBtn);

        const submitBtn = screen.getByText(/Confirm & Import/i);
        fireEvent.click(submitBtn);

        await waitFor(() => {
            expect(globalThis.fetch).toHaveBeenCalledWith(
                expect.stringContaining('/api/lists/2/receipts'),
                expect.objectContaining({ method: 'POST' })
            );
        });
    });

    it('supports creating a new list on the fly', async () => {
        render(
            <BrowserRouter>
                <ImportReceipt />
            </BrowserRouter>
        );

        const createBtn = await screen.findByText(/\+ Create a new list/i);
        fireEvent.click(createBtn);

        const input = screen.getByPlaceholderText(/e.g. Weekly Groceries/i);
        fireEvent.change(input, { target: { value: 'Vacation Trip' } });

        // Mock responses for creation and then upload
        globalThis.fetch = vi.fn((url) => {
            if (typeof url === 'string') {
                if (url.endsWith('/api/lists')) {
                    return Promise.resolve({
                        ok: true,
                        json: () => Promise.resolve({ id: 99, title: 'Vacation Trip' })
                    });
                }
                if (url.includes('/api/lists/99/receipts')) {
                    return Promise.resolve({ ok: true, json: () => Promise.resolve({ id: 500 }) });
                }
            }
            return Promise.resolve({ ok: true });
        });

        const submitBtn = screen.getByText(/Confirm & Import/i);
        fireEvent.click(submitBtn);

        await waitFor(() => {
            expect(globalThis.fetch).toHaveBeenCalledWith(
                expect.stringContaining('/api/lists'),
                expect.objectContaining({
                    method: 'POST',
                    body: JSON.stringify({ title: 'Vacation Trip' })
                })
            );
        });
    });

    it('cleans up IndexedDB after successful upload', async () => {
        const localforage = (await import('localforage')).default;

        render(
            <BrowserRouter>
                <ImportReceipt />
            </BrowserRouter>
        );

        const listBtn = await screen.findByText('Weekly Groceries');
        fireEvent.click(listBtn);

        const submitBtn = screen.getByText(/Confirm & Import/i);
        fireEvent.click(submitBtn);

        await waitFor(() => {
            expect(localforage.removeItem).toHaveBeenCalledWith('pending_shared_receipts');
        });
    });
});
