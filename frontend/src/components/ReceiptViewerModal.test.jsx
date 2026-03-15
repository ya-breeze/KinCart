/** @vitest-environment happy-dom */
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import ReceiptViewerModal from './ReceiptViewerModal';

// Mock useAuth
vi.mock('../context/AuthContext', () => ({
    useAuth: () => ({ token: 'test-token' }),
}));

// Mock config
vi.mock('../config', () => ({
    API_BASE_URL: 'http://localhost:8080',
}));

const mockReceipts = [
    {
        id: 1,
        status: 'parsed',
        date: '2026-03-10T00:00:00Z',
        total: 42.5,
        image_path: 'families/1/receipts/2026/03/receipt.jpg',
        shop: { id: 1, name: 'Costco' },
        items: [
            { id: 1, name: 'Milk', quantity: 2, unit: 'L', price: 2.99, total_price: 5.98 },
        ],
    },
    {
        id: 2,
        status: 'new',
        date: '2026-03-12T00:00:00Z',
        total: 0,
        image_path: 'families/1/receipts/2026/03/receipt.txt',
        shop: null,
        items: [],
    },
    {
        id: 3,
        status: 'error',
        date: '2026-03-13T00:00:00Z',
        total: 0,
        image_path: 'families/1/receipts/2026/03/receipt2.jpg',
        shop: null,
        items: [],
    },
];

describe('ReceiptViewerModal', () => {
    let fetchMock;

    beforeEach(() => {
        fetchMock = vi.fn();
        globalThis.fetch = fetchMock;
        globalThis.URL.createObjectURL = vi.fn(() => 'blob:mock-url');
        globalThis.URL.revokeObjectURL = vi.fn();
    });

    afterEach(() => {
        vi.restoreAllMocks();
    });

    it('renders nothing when closed', () => {
        render(
            <ReceiptViewerModal receipts={mockReceipts} isOpen={false} onClose={vi.fn()} />
        );
        expect(screen.queryByText('Receipts (3)')).toBeNull();
    });

    it('renders receipt list when open', () => {
        render(
            <ReceiptViewerModal receipts={mockReceipts} isOpen={true} onClose={vi.fn()} />
        );
        expect(screen.getByText('Receipts (3)')).toBeTruthy();
        expect(screen.getByTestId('receipt-list-item-1')).toBeTruthy();
        expect(screen.getByTestId('receipt-list-item-2')).toBeTruthy();
        expect(screen.getByTestId('receipt-list-item-3')).toBeTruthy();
    });

    it('shows correct status badges', () => {
        render(
            <ReceiptViewerModal receipts={mockReceipts} isOpen={true} onClose={vi.fn()} />
        );
        expect(screen.getByText('Parsed')).toBeTruthy();
        expect(screen.getByText('Pending')).toBeTruthy();
        expect(screen.getByText('Error')).toBeTruthy();
    });

    it('shows shop name for parsed receipts', () => {
        render(
            <ReceiptViewerModal receipts={mockReceipts} isOpen={true} onClose={vi.fn()} />
        );
        expect(screen.getByText('Costco')).toBeTruthy();
    });

    it('navigates to detail view on receipt click', async () => {
        fetchMock.mockResolvedValueOnce({
            ok: true,
            blob: () => Promise.resolve(new Blob(['img-data'], { type: 'image/jpeg' })),
        });

        render(
            <ReceiptViewerModal receipts={mockReceipts} isOpen={true} onClose={vi.fn()} />
        );

        fireEvent.click(screen.getByTestId('receipt-list-item-1'));

        await waitFor(() => {
            expect(screen.getByTestId('receipt-viewer-back')).toBeTruthy();
            expect(screen.getByTestId('receipt-viewer-download')).toBeTruthy();
        });
    });

    it('shows parsed items in detail view', async () => {
        fetchMock.mockResolvedValueOnce({
            ok: true,
            blob: () => Promise.resolve(new Blob(['img-data'], { type: 'image/jpeg' })),
        });

        render(
            <ReceiptViewerModal receipts={mockReceipts} isOpen={true} onClose={vi.fn()} />
        );
        fireEvent.click(screen.getByTestId('receipt-list-item-1'));

        await waitFor(() => {
            expect(screen.getByText('Milk')).toBeTruthy();
            expect(screen.getByText('$5.98')).toBeTruthy();
        });
    });

    it('shows "still processing" for new receipts', async () => {
        fetchMock.mockResolvedValueOnce({
            ok: true,
            text: () => Promise.resolve(''),
        });

        render(
            <ReceiptViewerModal receipts={mockReceipts} isOpen={true} onClose={vi.fn()} />
        );
        fireEvent.click(screen.getByTestId('receipt-list-item-2'));

        await waitFor(() => {
            expect(screen.getByText(/Still processing/)).toBeTruthy();
        });
    });

    it('shows error message for failed receipts', async () => {
        fetchMock.mockResolvedValueOnce({
            ok: true,
            blob: () => Promise.resolve(new Blob(['img-data'], { type: 'image/jpeg' })),
        });

        render(
            <ReceiptViewerModal receipts={mockReceipts} isOpen={true} onClose={vi.fn()} />
        );
        fireEvent.click(screen.getByTestId('receipt-list-item-3'));

        await waitFor(() => {
            expect(screen.getByText(/Could not parse/)).toBeTruthy();
        });
    });

    it('fetches text content for .txt receipts', async () => {
        fetchMock.mockResolvedValueOnce({
            ok: true,
            text: () => Promise.resolve('Store: Lidl\nTotal: 10.00'),
        });

        render(
            <ReceiptViewerModal receipts={mockReceipts} isOpen={true} onClose={vi.fn()} />
        );
        fireEvent.click(screen.getByTestId('receipt-list-item-2'));

        await waitFor(() => {
            expect(screen.getByText(/Store: Lidl/)).toBeTruthy();
        });
    });

    it('returns to list view on back button click', async () => {
        fetchMock.mockResolvedValueOnce({
            ok: true,
            blob: () => Promise.resolve(new Blob()),
        });

        render(
            <ReceiptViewerModal receipts={mockReceipts} isOpen={true} onClose={vi.fn()} />
        );
        fireEvent.click(screen.getByTestId('receipt-list-item-1'));

        await waitFor(() => screen.getByTestId('receipt-viewer-back'));
        fireEvent.click(screen.getByTestId('receipt-viewer-back'));

        expect(screen.getByText('Receipts (3)')).toBeTruthy();
    });

    it('calls onClose when overlay is clicked', () => {
        const onClose = vi.fn();
        render(
            <ReceiptViewerModal receipts={mockReceipts} isOpen={true} onClose={onClose} />
        );
        fireEvent.click(screen.getByTestId('receipt-viewer-overlay'));
        expect(onClose).toHaveBeenCalled();
    });

    it('sends auth header when fetching receipt file', async () => {
        fetchMock.mockResolvedValueOnce({
            ok: true,
            blob: () => Promise.resolve(new Blob()),
        });

        render(
            <ReceiptViewerModal receipts={mockReceipts} isOpen={true} onClose={vi.fn()} />
        );
        fireEvent.click(screen.getByTestId('receipt-list-item-1'));

        await waitFor(() => {
            expect(fetchMock).toHaveBeenCalledWith(
                'http://localhost:8080/api/receipts/1/file',
                expect.objectContaining({
                    headers: expect.objectContaining({ Authorization: 'Bearer test-token' }),
                })
            );
        });
    });

    it('shows empty items state gracefully', async () => {
        const noItemsReceipts = [
            {
                id: 10,
                status: 'parsed',
                date: '2026-03-10T00:00:00Z',
                total: 5.0,
                image_path: 'families/1/receipts/2026/03/r.jpg',
                shop: null,
                items: [],
            },
        ];

        fetchMock.mockResolvedValueOnce({
            ok: true,
            blob: () => Promise.resolve(new Blob()),
        });

        render(
            <ReceiptViewerModal receipts={noItemsReceipts} isOpen={true} onClose={vi.fn()} />
        );
        fireEvent.click(screen.getByTestId('receipt-list-item-10'));

        await waitFor(() => {
            expect(screen.getByText('No items available')).toBeTruthy();
        });
    });
});
