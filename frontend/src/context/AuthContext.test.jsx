/** @vitest-environment happy-dom */
import { render, screen, act } from '@testing-library/react';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import { AuthProvider, useAuth } from './AuthContext';
import React from 'react';

// Helper component to test the context
const TestComponent = () => {
    const { user, login, logout } = useAuth();
    return (
        <div>
            <div data-testid="user">{user ? user.username : 'no user'}</div>
            <button onClick={() => login({ username: 'test' })}>Login</button>
            <button onClick={logout}>Logout</button>
        </div>
    );
};

describe('AuthContext', () => {
    beforeEach(() => {
        vi.stubGlobal('localStorage', {
            getItem: vi.fn().mockReturnValue(null),
            setItem: vi.fn(),
            clear: vi.fn(),
            removeItem: vi.fn(),
        });
        vi.stubGlobal('fetch', vi.fn(() =>
            Promise.resolve({
                ok: false,
                status: 401,
                json: () => Promise.resolve({}),
            })
        ));
        vi.clearAllMocks();
    });

    it('provides initial null state', async () => {
        await act(async () => {
            render(
                <AuthProvider>
                    <TestComponent />
                </AuthProvider>
            );
        });

        expect(screen.getByTestId('user')).toHaveTextContent('no user');
    });

    it('updates state on login', async () => {
        await act(async () => {
            render(
                <AuthProvider>
                    <TestComponent />
                </AuthProvider>
            );
        });

        await act(async () => {
            screen.getByText('Login').click();
        });

        expect(screen.getByTestId('user')).toHaveTextContent('test');
    });

    it('clears state on logout', async () => {
        await act(async () => {
            render(
                <AuthProvider>
                    <TestComponent />
                </AuthProvider>
            );
        });

        // Login first
        await act(async () => {
            screen.getByText('Login').click();
        });

        // Then logout
        await act(async () => {
            screen.getByText('Logout').click();
        });

        expect(screen.getByTestId('user')).toHaveTextContent('no user');
    });
});
