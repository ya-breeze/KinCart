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
    already_bought_items: [],
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

        fireEvent.click(screen.getByTestId(`expand-${RECEIPT_ITEM_ID_PRAZMA}`));

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
        fireEvent.click(screen.getByTestId(`expand-${RECEIPT_ITEM_ID_PRAZMA}`));

        // молоко is in suggestions — should NOT appear in planned section too
        expect(screen.queryByTestId(`link-planned-${PLANNED_ID_MILK}`)).toBeNull();
        // дорадо is NOT in suggestions — should appear in planned section
        expect(screen.getByTestId(`link-planned-${PLANNED_ID_DORADO}`)).toBeInTheDocument();
    });

    it('stages a link decision when planned item is clicked (no API call on click)', async () => {
        mockFetchWithData(defaultMatchData);
        render(<ReceiptMatchModal {...defaultProps} />);
        await waitFor(() => screen.getByText('*ASC MC PRAŽMA KR.'));

        // Initially confirm is blocked (1 unresolved item)
        expect(screen.getByTestId('confirm-all-btn')).toHaveTextContent('still need a decision');

        // Expand and link to дорадо
        fireEvent.click(screen.getByTestId(`expand-${RECEIPT_ITEM_ID_PRAZMA}`));
        fireEvent.click(screen.getByTestId(`link-planned-${PLANNED_ID_DORADO}`));

        // Only the initial fetch was called — no PATCH fired on click
        expect(fetch).toHaveBeenCalledTimes(1);

        // Decision staged: confirm button now shows ready state
        expect(screen.getByTestId('confirm-all-btn')).toHaveTextContent('Confirm & done');
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
        fireEvent.click(screen.getByTestId(`expand-${RECEIPT_ITEM_ID_PRAZMA}`));

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
        fireEvent.click(screen.getByTestId(`expand-${RECEIPT_ITEM_ID_PRAZMA}`));

        expect(screen.queryByTestId(`link-planned-${PLANNED_ID_DORADO}`)).toBeNull();
        expect(screen.queryByTestId(`link-planned-${PLANNED_ID_MILK}`)).toBeNull();
        expect(screen.queryByText('from your list')).toBeNull();
    });

    it('does not render when isOpen is false', () => {
        render(<ReceiptMatchModal {...defaultProps} isOpen={false} />);
        expect(screen.queryByText('Review Receipt')).toBeNull();
    });

    it('shows error when fetch fails', async () => {
        fetch.mockResolvedValue({
            ok: false,
            json: async () => ({ error: 'unauthorized' }),
        });
        render(<ReceiptMatchModal {...defaultProps} />);

        await waitFor(() => screen.getByText('unauthorized'));
    });

    // --- New tests for the redesigned local-staging architecture ---

    it('renders already_bought_items as link options in DecisionRow', async () => {
        const BOUGHT_ID = 'bought-uuid-1';
        const dataWithBought = {
            ...defaultMatchData,
            already_bought_items: [{ id: BOUGHT_ID, name: 'яйца' }],
        };
        mockFetchWithData(dataWithBought);
        render(<ReceiptMatchModal {...defaultProps} />);

        await waitFor(() => screen.getByText('*ASC MC PRAŽMA KR.'));
        fireEvent.click(screen.getByTestId(`expand-${RECEIPT_ITEM_ID_PRAZMA}`));

        expect(screen.getByTestId(`link-bought-${BOUGHT_ID}`)).toBeInTheDocument();
        expect(screen.getByTestId(`link-bought-${BOUGHT_ID}`)).toHaveTextContent('яйца');
        expect(screen.getByText('already bought')).toBeInTheDocument();
    });

    it('undo reverts last decision', async () => {
        mockFetchWithData(defaultMatchData);
        render(<ReceiptMatchModal {...defaultProps} />);

        await waitFor(() => screen.getByText('*ASC MC PRAŽMA KR.'));

        // Initially no Undo button; confirm blocked (1 pending item)
        expect(screen.queryByText('Undo')).toBeNull();
        expect(screen.getByTestId('confirm-all-btn')).toHaveTextContent('still need a decision');

        // Expand and link to дорадо
        fireEvent.click(screen.getByTestId(`expand-${RECEIPT_ITEM_ID_PRAZMA}`));
        fireEvent.click(screen.getByTestId(`link-planned-${PLANNED_ID_DORADO}`));

        // After linking: confirm ready, Undo button visible
        expect(screen.getByText('Undo')).toBeInTheDocument();
        expect(screen.getByTestId('confirm-all-btn')).toHaveTextContent('Confirm & done');

        // Click Undo
        fireEvent.click(screen.getByText('Undo'));

        // Decision reverted: Undo gone, confirm blocked again
        expect(screen.queryByText('Undo')).toBeNull();
        expect(screen.getByTestId('confirm-all-btn')).toHaveTextContent('still need a decision');
    });

    it('reset restores initial state after multiple decisions', async () => {
        const dataWithTwoItems = {
            ...defaultMatchData,
            items: [
                { ...defaultMatchData.items[0] },
                {
                    receipt_item_id: RECEIPT_ITEM_ID_EXTRA,
                    receipt_name: 'EXTRA ITEM',
                    price: 50,
                    quantity: 1,
                    total_price: 50,
                    match_status: 'unmatched',
                    confidence: 0,
                    matched_item: null,
                    suggestions: [],
                    is_extra: false,
                },
            ],
        };
        mockFetchWithData(dataWithTwoItems);
        render(<ReceiptMatchModal {...defaultProps} />);

        await waitFor(() => screen.getByText('*ASC MC PRAŽMA KR.'));

        // Make decisions on both items
        fireEvent.click(screen.getByTestId(`expand-${RECEIPT_ITEM_ID_PRAZMA}`));
        fireEvent.click(screen.getByTestId(`link-planned-${PLANNED_ID_DORADO}`));
        fireEvent.click(screen.getByTestId(`expand-${RECEIPT_ITEM_ID_EXTRA}`));
        fireEvent.click(screen.getByTestId(`skip-${RECEIPT_ITEM_ID_EXTRA}`));

        // Both decided → confirm ready, Reset visible
        expect(screen.getByTestId('confirm-all-btn')).toHaveTextContent('Confirm & done');
        expect(screen.getByText('Reset')).toBeInTheDocument();

        // Click Reset
        fireEvent.click(screen.getByText('Reset'));

        // Both items pending again; Reset and Undo gone
        expect(screen.queryByText('Reset')).toBeNull();
        expect(screen.queryByText('Undo')).toBeNull();
        expect(screen.getByTestId('confirm-all-btn')).toHaveTextContent('still need a decision');
    });

    it('confirm all fires PATCH with null planned_item_id for add-as-new decision', async () => {
        fetch.mockResolvedValueOnce({ ok: true, json: async () => defaultMatchData });
        fetch.mockResolvedValue({ ok: true, json: async () => ({}) });

        render(<ReceiptMatchModal {...defaultProps} />);
        await waitFor(() => screen.getByText('*ASC MC PRAŽMA KR.'));

        // Expand and click "Add as new"
        fireEvent.click(screen.getByTestId(`expand-${RECEIPT_ITEM_ID_PRAZMA}`));
        fireEvent.click(screen.getByTestId(`addnew-${RECEIPT_ITEM_ID_PRAZMA}`));

        // Confirm button shows new-item count
        expect(screen.getByTestId('confirm-all-btn')).toHaveTextContent('add 1 item');

        // Confirm
        fireEvent.click(screen.getByTestId('confirm-all-btn'));

        // initial load + PATCH + confirm-all = 3 calls
        await waitFor(() => expect(fetch).toHaveBeenCalledTimes(3));

        const patchCall = fetch.mock.calls[1];
        expect(patchCall[0]).toContain(`/api/receipts/${RECEIPT_ID}/matches/${RECEIPT_ITEM_ID_PRAZMA}`);
        expect(patchCall[1].method).toBe('PATCH');
        expect(JSON.parse(patchCall[1].body)).toEqual({ planned_item_id: null });
    });

    it('confirm button is blocked when a match is removed (null decision)', async () => {
        const MATCHED_RECEIPT_ID = 55;
        const dataWithMatched = {
            ...defaultMatchData,
            items: [{
                receipt_item_id: MATCHED_RECEIPT_ID,
                receipt_name: 'MATCHED ITEM',
                price: 100,
                quantity: 1,
                total_price: 100,
                match_status: 'auto',
                confidence: 90,
                matched_item: { id: PLANNED_ID_MILK, name: 'молоко' },
                suggestions: [],
                is_extra: false,
            }],
            unmatched_planned_items: [],
        };
        mockFetchWithData(dataWithMatched);
        render(<ReceiptMatchModal {...defaultProps} />);

        await waitFor(() => screen.getByText('MATCHED ITEM'));

        // Auto-matched with no manual decisions → confirm ready
        expect(screen.getByTestId('confirm-all-btn')).toHaveTextContent('Confirm & done');

        // Expand matched row and remove the link
        fireEvent.click(screen.getByTestId(`match-expand-${MATCHED_RECEIPT_ID}`));
        fireEvent.click(screen.getByText('Remove link'));

        // Confirm now blocked: removed match is pending
        await waitFor(() =>
            expect(screen.getByTestId('confirm-all-btn')).toHaveTextContent('still need a decision')
        );

        // Clicking confirm does not fire additional API calls
        const callsBefore = fetch.mock.calls.length;
        fireEvent.click(screen.getByTestId('confirm-all-btn'));
        expect(fetch).toHaveBeenCalledTimes(callsBefore);
    });
});
