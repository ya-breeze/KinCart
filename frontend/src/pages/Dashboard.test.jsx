/** @vitest-environment happy-dom */
import { render, screen, fireEvent } from '@testing-library/react';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import Dashboard from './Dashboard';
import { BrowserRouter } from 'react-router-dom';
import { AuthProvider } from '../context/AuthContext';

// Mock AuthContext
const mockToggleMode = vi.fn();
vi.mock('../context/AuthContext', () => ({
    useAuth: () => ({
        user: { username: 'testuser' },
        token: 'fake-token',
        mode: 'manager',
        currency: '₽',
        toggleMode: mockToggleMode,
    }),
    AuthProvider: ({ children }) => <div>{children}</div>,
}));

describe('Dashboard', () => {
    beforeEach(() => {
        vi.clearAllMocks();
        globalThis.fetch = vi.fn(() =>
            Promise.resolve({
                ok: true,
                json: () => Promise.resolve([
                    { id: 1, title: 'List 1', status: 'preparing', items: [] },
                    { id: 2, title: 'List 2', status: 'ready for shopping', items: [] },
                ]),
            })
        );
    });

    it('renders dashboard with lists for manager', async () => {
        render(
            <BrowserRouter>
                <Dashboard />
            </BrowserRouter>
        );

        expect(screen.getByText('KinCart')).toBeInTheDocument();
        expect(await screen.findByText('List 1')).toBeInTheDocument();
        expect(await screen.findByText('List 2')).toBeInTheDocument();
    });

    it('shows no lists message when empty', async () => {
        globalThis.fetch = vi.fn(() =>
            Promise.resolve({
                ok: true,
                json: () => Promise.resolve([]),
            })
        );

        render(
            <BrowserRouter>
                <Dashboard />
            </BrowserRouter>
        );

        expect(await screen.findByText(/No lists yet/i)).toBeInTheDocument();
    });

    it('does not show pending banner when no receipts are pending', async () => {
        globalThis.fetch = vi.fn(() =>
            Promise.resolve({
                ok: true,
                json: () => Promise.resolve([
                    { id: 1, title: 'List 1', status: 'preparing', items: [], receipts: [{ id: 10, status: 'parsed' }] },
                ]),
            })
        );

        render(<BrowserRouter><Dashboard /></BrowserRouter>);

        await screen.findByText('List 1');
        expect(screen.queryByTestId('pending-receipts-banner')).not.toBeInTheDocument();
    });

    it('shows pending receipts banner with correct count', async () => {
        globalThis.fetch = vi.fn(() =>
            Promise.resolve({
                ok: true,
                json: () => Promise.resolve([
                    { id: 1, title: 'Groceries', status: 'preparing', items: [], receipts: [{ id: 10, status: 'new' }, { id: 11, status: 'new' }] },
                    { id: 2, title: 'Hardware', status: 'preparing', items: [], receipts: [{ id: 12, status: 'parsed' }] },
                ]),
            })
        );

        render(<BrowserRouter><Dashboard /></BrowserRouter>);

        const banner = await screen.findByTestId('pending-receipts-banner');
        expect(banner).toBeInTheDocument();
        expect(banner.textContent).toMatch(/2 receipts pending AI processing/i);
    });

    it('expands pending receipts panel on click and shows list names', async () => {
        globalThis.fetch = vi.fn(() =>
            Promise.resolve({
                ok: true,
                json: () => Promise.resolve([
                    { id: 1, title: 'Groceries', status: 'preparing', items: [], receipts: [{ id: 10, status: 'new' }] },
                    { id: 2, title: 'Hardware', status: 'preparing', items: [], receipts: [{ id: 11, status: 'new' }] },
                ]),
            })
        );

        render(<BrowserRouter><Dashboard /></BrowserRouter>);

        const banner = await screen.findByTestId('pending-receipts-banner');
        fireEvent.click(banner);

        // List names appear in both the pending panel and the list cards
        expect(screen.getAllByText('Groceries').length).toBeGreaterThanOrEqual(1);
        expect(screen.getAllByText('Hardware').length).toBeGreaterThanOrEqual(1);
        expect(screen.getByText(/Items will appear automatically/i)).toBeInTheDocument();
    });

    it('shows singular receipt label for exactly 1 pending receipt', async () => {
        globalThis.fetch = vi.fn(() =>
            Promise.resolve({
                ok: true,
                json: () => Promise.resolve([
                    { id: 1, title: 'My List', status: 'preparing', items: [], receipts: [{ id: 10, status: 'new' }] },
                ]),
            })
        );

        render(<BrowserRouter><Dashboard /></BrowserRouter>);

        const banner = await screen.findByTestId('pending-receipts-banner');
        expect(banner.textContent).toMatch(/1 receipt pending AI processing/i);
    });
});
