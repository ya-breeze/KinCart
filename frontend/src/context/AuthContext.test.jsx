/** @vitest-environment happy-dom */
import { render, screen, act } from '@testing-library/react';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import { AuthProvider, useAuth } from './AuthContext';
import React from 'react';

// Helper component to test the context
const TestComponent = () => {
    const { user, login, logout, token } = useAuth();
    return (
        <div>
            <div data-testid="user">{user ? user.username : 'no user'}</div>
            <div data-testid="token">{token || 'no token'}</div>
            <button onClick={() => login({ username: 'test' }, 'secret-token')}>Login</button>
            <button onClick={logout}>Logout</button>
        </div>
    );
};

describe('AuthContext', () => {
    beforeEach(() => {
        vi.stubGlobal('localStorage', {
            getItem: vi.fn(),
            setItem: vi.fn(),
            clear: vi.fn(),
            removeItem: vi.fn(),
        });
        vi.stubGlobal('fetch', vi.fn(() =>
            Promise.resolve({
                ok: true,
                json: () => Promise.resolve({ username: 'test' }),
            })
        ));
        vi.clearAllMocks();
    });

    it('provides initial null state', () => {
        render(
            <AuthProvider>
                <TestComponent />
            </AuthProvider>
        );

        expect(screen.getByTestId('user')).toHaveTextContent('no user');
        expect(screen.getByTestId('token')).toHaveTextContent('no token');
    });

    it('updates state on login', async () => {
        render(
            <AuthProvider>
                <TestComponent />
            </AuthProvider>
        );

        await act(async () => {
            screen.getByText('Login').click();
        });

        expect(screen.getByTestId('user')).toHaveTextContent('test');
        expect(screen.getByTestId('token')).toHaveTextContent('secret-token');
        expect(window.localStorage.setItem).toHaveBeenCalledWith('token', 'secret-token');
    });

    it('clears state on logout', async () => {
        render(
            <AuthProvider>
                <TestComponent />
            </AuthProvider>
        );

        await act(async () => {
            screen.getByText('Logout').click();
        });

        expect(screen.getByTestId('user')).toHaveTextContent('no user');
        expect(screen.getByTestId('token')).toHaveTextContent('no token');
        expect(window.localStorage.removeItem).toHaveBeenCalledWith('token');
    });
});
