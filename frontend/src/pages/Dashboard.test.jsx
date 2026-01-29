/** @vitest-environment happy-dom */
import { render, screen } from '@testing-library/react';
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
});
