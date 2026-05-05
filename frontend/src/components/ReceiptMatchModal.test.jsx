/** @vitest-environment happy-dom */
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import ReceiptMatchModal from './ReceiptMatchModal';

const RECEIPT_ID = 'receipt-uuid-1';
const PLANNED_ID_DORADO = 'planned-uuid-dorado';
const PLANNED_ID_MILK = 'planned-uuid-milk';
const RECEIPT_ITEM_ID_PRAZMA = 42;
const RECEIPT_ITEM_ID_EXTRA = 99;

const defaultMatchData = {
    receipt_id: RECEIPT_ID,
    status: 'pending_review',
    shop_name: 'Makro',
    date: '2026-04-05',
    total: 369.0,
    items: [
        {
            receipt_item_id: RECEIPT_ITEM_ID_PRAZMA,
            receipt_name: '*ASC MC PRAŽMA KR.',
            price: 369.0,
            quantity: 1,
            total_price: 369.0,
            match_status: 'unmatched',
            confidence: 0,
            matched_item: null,
            suggestions: [],
            is_extra: false,
        },
    ],
    unmatched_planned_items: [
        { id: PLANNED_ID_DORADO, name: 'дорадо' },
        { id: PLANNED_ID_MILK, name: 'молоко' },
    ],
};

const defaultProps = {
    isOpen: true,
    onClose: vi.fn(),
    receiptId: RECEIPT_ID,
    onDone: vi.fn(),
};

beforeEach(() => {
    vi.clearAllMocks();
    vi.stubGlobal('fetch', vi.fn());
});

function mockFetchWithData(data) {
    fetch.mockResolvedValue({
        ok: true,
        json: async () => data,
    });
}

describe('ReceiptMatchModal — planned item linking', () => {
    it('shows planned items in dropdown when receipt item has no AI suggestions', async () => {
        mockFetchWithData(defaultMatchData);
        render(<ReceiptMatchModal {...defaultProps} />);

        await waitFor(() => screen.getByText('*ASC MC PRAŽMA KR.'));

        // Expand the unmatched row
        fireEvent.click(screen.getByTestId(`unmatch-expand-${RECEIPT_ITEM_ID_PRAZMA}`));

        expect(screen.getByTestId(`link-planned-${PLANNED_ID_DORADO}`)).toBeInTheDocument();
        expect(screen.getByTestId(`link-planned-${PLANNED_ID_MILK}`)).toBeInTheDocument();
        expect(screen.getByTestId(`link-planned-${PLANNED_ID_DORADO}`)).toHaveTextContent('дорадо');
    });

    it('does not show suggested items in the planned-items section', async () => {
        const dataWithSuggestion = {
            ...defaultMatchData,
            items: [
                {
                    ...defaultMatchData.items[0],
                    suggestions: [{ item_id: PLANNED_ID_MILK, item_name: 'молоко', confidence: 72 }],
                },
            ],
        };
        mockFetchWithData(dataWithSuggestion);
        render(<ReceiptMatchModal {...defaultProps} />);

        await waitFor(() => screen.getByText('*ASC MC PRAŽMA KR.'));
        fireEvent.click(screen.getByTestId(`unmatch-expand-${RECEIPT_ITEM_ID_PRAZMA}`));

        // молоко is in suggestions — should NOT appear in planned section too
        expect(screen.queryByTestId(`link-planned-${PLANNED_ID_MILK}`)).toBeNull();
        // дорадо is NOT in suggestions — should appear in planned section
        expect(screen.getByTestId(`link-planned-${PLANNED_ID_DORADO}`)).toBeInTheDocument();
    });

    it('calls confirmMatch with the planned item id when a planned item is clicked', async () => {
        mockFetchWithData(defaultMatchData);
        // After confirm, reload returns the same data
        fetch.mockResolvedValue({ ok: true, json: async () => defaultMatchData });

        render(<ReceiptMatchModal {...defaultProps} />);
        await waitFor(() => screen.getByText('*ASC MC PRAŽMA KR.'));
        fireEvent.click(screen.getByTestId(`unmatch-expand-${RECEIPT_ITEM_ID_PRAZMA}`));

        // Click on дорадо
        const doradoBtn = screen.getByTestId(`link-planned-${PLANNED_ID_DORADO}`);
        fireEvent.click(doradoBtn);

        await waitFor(() => expect(fetch).toHaveBeenCalledTimes(2)); // initial load + confirm

        const [url, options] = fetch.mock.calls[1];
        expect(url).toContain(`/api/receipts/${RECEIPT_ID}/matches/${RECEIPT_ITEM_ID_PRAZMA}`);
        expect(options.method).toBe('PATCH');
        expect(JSON.parse(options.body)).toEqual({ planned_item_id: PLANNED_ID_DORADO });
    });

    it('shows divider between suggestions and planned items when both exist', async () => {
        const dataWithBoth = {
            ...defaultMatchData,
            items: [
                {
                    ...defaultMatchData.items[0],
                    suggestions: [{ item_id: PLANNED_ID_MILK, item_name: 'молоко', confidence: 72 }],
                },
            ],
        };
        mockFetchWithData(dataWithBoth);
        render(<ReceiptMatchModal {...defaultProps} />);

        await waitFor(() => screen.getByText('*ASC MC PRAŽMA KR.'));
        fireEvent.click(screen.getByTestId(`unmatch-expand-${RECEIPT_ITEM_ID_PRAZMA}`));

        expect(screen.getByText('from your list')).toBeInTheDocument();
    });

    it('does not show planned-items section when all planned items are already in suggestions', async () => {
        const dataAllSuggested = {
            ...defaultMatchData,
            items: [
                {
                    ...defaultMatchData.items[0],
                    suggestions: [
                        { item_id: PLANNED_ID_DORADO, item_name: 'дорадо', confidence: 80 },
                        { item_id: PLANNED_ID_MILK, item_name: 'молоко', confidence: 60 },
                    ],
                },
            ],
        };
        mockFetchWithData(dataAllSuggested);
        render(<ReceiptMatchModal {...defaultProps} />);

        await waitFor(() => screen.getByText('*ASC MC PRAŽMA KR.'));
        fireEvent.click(screen.getByTestId(`unmatch-expand-${RECEIPT_ITEM_ID_PRAZMA}`));

        expect(screen.queryByTestId(`link-planned-${PLANNED_ID_DORADO}`)).toBeNull();
        expect(screen.queryByTestId(`link-planned-${PLANNED_ID_MILK}`)).toBeNull();
        expect(screen.queryByText('from your list')).toBeNull();
    });

    it('does not render when isOpen is false', () => {
        render(<ReceiptMatchModal {...defaultProps} isOpen={false} />);
        expect(screen.queryByText('Review Receipt Matches')).toBeNull();
    });

    it('shows error when fetch fails', async () => {
        fetch.mockResolvedValue({
            ok: false,
            json: async () => ({ error: 'unauthorized' }),
        });
        render(<ReceiptMatchModal {...defaultProps} />);

        await waitFor(() => screen.getByText('unauthorized'));
    });
});
