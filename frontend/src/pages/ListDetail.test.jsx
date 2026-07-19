/** @vitest-environment happy-dom */
import { render, screen, fireEvent } from '@testing-library/react';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import ListDetail from './ListDetail';
import { BrowserRouter } from 'react-router-dom';

vi.mock('react-router-dom', async () => {
    const actual = await vi.importActual('react-router-dom');
    return { ...actual, useParams: () => ({ id: 'list-1' }), useNavigate: () => vi.fn() };
});

const mockMode = { current: 'manager' };
vi.mock('../context/AuthContext', () => ({
    useAuth: () => ({
        user: { username: 'testuser' },
        token: 'fake-token',
        get mode() { return mockMode.current; },
        currency: '₽',
        toggleMode: vi.fn(),
    }),
    AuthProvider: ({ children }) => <div>{children}</div>,
}));

const item = (over) => ({
    id: 'i-1', name: 'Item', quantity: 1, unit: 'pcs', price: 0,
    is_bought: false, is_absent: false, is_urgent: false,
    category_id: '00000000-0000-0000-0000-000000000000',
    list_id: 'list-1', ...over,
});

// The component fires five GETs on mount; route each to a minimal shape.
const mockFetchWithItems = (items) => {
    globalThis.fetch = vi.fn((url) => {
        const body =
            url.includes('/api/lists/') ? { id: 'list-1', title: 'Groceries', status: 'ready for shopping', items, receipts: [] }
                : []; // categories, shops, frequent-items, aliases
        return Promise.resolve({ ok: true, json: () => Promise.resolve(body) });
    });
};

const renderList = () => render(<BrowserRouter><ListDetail /></BrowserRouter>);

describe('ListDetail — manager "not found" badge', () => {
    beforeEach(() => {
        vi.clearAllMocks();
        mockMode.current = 'manager';
    });

    it('badges an absent item', async () => {
        mockFetchWithItems([item({ id: 'a', name: 'Saffron', is_absent: true })]);
        renderList();

        expect(await screen.findByText('Saffron')).toBeInTheDocument();
        expect(screen.getByTestId('item-not-found-badge')).toBeInTheDocument();
    });

    it('does not badge a bought item', async () => {
        mockFetchWithItems([item({ id: 'b', name: 'Butter', is_bought: true })]);
        renderList();

        expect(await screen.findByText('Butter')).toBeInTheDocument();
        expect(screen.queryByTestId('item-not-found-badge')).not.toBeInTheDocument();
    });

    it('does not badge a plain active item', async () => {
        mockFetchWithItems([item({ id: 'c', name: 'Bread' })]);
        renderList();

        expect(await screen.findByText('Bread')).toBeInTheDocument();
        expect(screen.queryByTestId('item-not-found-badge')).not.toBeInTheDocument();
    });

    it('badges only the absent item when several states are mixed', async () => {
        mockFetchWithItems([
            item({ id: 'a', name: 'Saffron', is_absent: true }),
            item({ id: 'b', name: 'Butter', is_bought: true }),
            item({ id: 'c', name: 'Bread' }),
        ]);
        renderList();

        expect(await screen.findByText('Saffron')).toBeInTheDocument();
        // Exclusivity is enforced server-side, so exactly one badge is correct here.
        expect(screen.getAllByTestId('item-not-found-badge')).toHaveLength(1);
    });
});

describe('ListDetail — shopper done section', () => {
    beforeEach(() => {
        vi.clearAllMocks();
        mockMode.current = 'shopper';
    });

    it('offers a direct "Bought" action on an absent row, and Undo', async () => {
        mockFetchWithItems([item({ id: 'a', name: 'Saffron', is_absent: true })]);
        renderList();

        // Done section is collapsed by default; expand it.
        fireEvent.click(await screen.findByText(/1 done/i));

        expect(screen.getByRole('button', { name: /mark Saffron as bought/i })).toBeInTheDocument();
        expect(screen.getByRole('button', { name: /mark Saffron as available/i })).toBeInTheDocument();
    });

    it('a bought row offers Undo but no "Bought" action', async () => {
        mockFetchWithItems([item({ id: 'b', name: 'Butter', is_bought: true })]);
        renderList();

        fireEvent.click(await screen.findByText(/1 done/i));

        expect(screen.queryByRole('button', { name: /mark Butter as bought/i })).not.toBeInTheDocument();
        expect(screen.getByRole('button', { name: /mark Butter as not bought/i })).toBeInTheDocument();
    });

    it('excludes absent items from the total', async () => {
        // An out-of-stock item is never paid for, so it must not inflate the
        // figure the shopper reads as "what this costs". design.md promises this.
        mockFetchWithItems([
            item({ id: 'a', name: 'Saffron', price: 100, is_absent: true }),
            item({ id: 'b', name: 'Bread', price: 200 }),
        ]);
        renderList();

        expect(await screen.findByText('Bread')).toBeInTheDocument();
        // 200.00 also appears as Bread's own price line, so the load-bearing
        // assertion is the negative one: 300.00 is what the header showed before
        // absent items were excluded.
        expect(screen.getAllByText(/200\.00/).length).toBeGreaterThan(0);
        expect(screen.queryByText(/300\.00/)).not.toBeInTheDocument();
    });

    it('hides the done section entirely when nothing is done', async () => {
        mockFetchWithItems([item({ id: 'c', name: 'Bread' })]);
        renderList();

        expect(await screen.findByText('Bread')).toBeInTheDocument();
        expect(screen.queryByText(/done$/i)).not.toBeInTheDocument();
    });
});
