/** @vitest-environment happy-dom */
import { render, screen } from '@testing-library/react';
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
