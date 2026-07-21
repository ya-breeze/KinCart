/** @vitest-environment happy-dom */
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import PasteItemsPanel from './PasteItemsPanel';

const categories = [
    { id: 'cat-dairy', name: 'Dairy' },
    { id: 'cat-snacks', name: 'Snacks' },
];

function renderPanel(extra = {}) {
    return render(
        <PasteItemsPanel
            listId="list-1"
            shops={[]}
            categories={categories}
            onItemsAdded={vi.fn()}
            {...extra}
        />,
    );
}

// Drives the panel through parse → preview.
async function parseWith(previewItems) {
    fetch.mockResolvedValueOnce({ ok: true, json: async () => previewItems });
    fireEvent.change(screen.getByPlaceholderText(/йогурта/), { target: { value: 'yogurt' } });
    fireEvent.click(screen.getByText('Parse list'));
    await waitFor(() => expect(screen.getByText(/Preview/)).toBeInTheDocument());
}

beforeEach(() => {
    vi.clearAllMocks();
    vi.stubGlobal('fetch', vi.fn());
});

describe('PasteItemsPanel — suggested category', () => {
    it('shows the suggested category name from the backend', async () => {
        renderPanel();
        await parseWith([
            { name: 'yogurt', quantity: 1, unit: 'pack', suggested_category_id: 'cat-dairy', suggested_category_name: 'Dairy' },
        ]);
        expect(screen.getByText('Dairy')).toBeInTheDocument();
        // The remembered unit is surfaced too.
        expect(screen.getByText('pack')).toBeInTheDocument();
    });

    it('carries the suggested category_id through bulk-add', async () => {
        renderPanel();
        await parseWith([
            { name: 'yogurt', quantity: 1, unit: 'pack', suggested_category_id: 'cat-dairy' },
        ]);

        fetch.mockResolvedValueOnce({ ok: true, json: async () => ({ created: 1 }) });
        fireEvent.click(screen.getByText(/Add all/));

        await waitFor(() => expect(fetch).toHaveBeenCalledTimes(2));
        const bulkCall = fetch.mock.calls[1];
        expect(bulkCall[0]).toContain('/items/bulk');
        const sent = JSON.parse(bulkCall[1].body);
        expect(sent[0].category_id).toBe('cat-dairy');
        expect(sent[0].unit).toBe('pack');
    });

    it('omits category_id entirely for an uncategorized item', async () => {
        renderPanel();
        await parseWith([
            { name: 'dragonfruit', quantity: 1, unit: 'pcs' }, // no suggested_category_id
        ]);

        fetch.mockResolvedValueOnce({ ok: true, json: async () => ({ created: 1 }) });
        fireEvent.click(screen.getByText(/Add all/));

        await waitFor(() => expect(fetch).toHaveBeenCalledTimes(2));
        const sent = JSON.parse(fetch.mock.calls[1][1].body);
        expect(sent[0]).not.toHaveProperty('category_id');
    });
});
