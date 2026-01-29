/** @vitest-environment happy-dom */
import { render, screen } from '@testing-library/react';
import { describe, it, expect, vi } from 'vitest';
import LoginPage from './LoginPage';
import { BrowserRouter } from 'react-router-dom';
import { AuthProvider } from '../context/AuthContext';

// Mock Lucide icons as they might cause issues in test environment
vi.mock('lucide-react', () => ({
    LogIn: () => <div data-testid="login-icon" />,
    ShoppingCart: () => <div data-testid="cart-icon" />,
}));

// Mock AuthContext
vi.mock('../context/AuthContext', () => ({
    useAuth: () => ({
        login: vi.fn(),
        token: null,
    }),
    AuthProvider: ({ children }) => <div>{children}</div>,
}));

describe('LoginPage', () => {
    it('renders login form', () => {
        render(
            <BrowserRouter>
                <LoginPage />
            </BrowserRouter>
        );

        expect(screen.getByText('KinCart')).toBeInTheDocument();
        expect(screen.getByLabelText(/Username/i)).toBeInTheDocument();
        expect(screen.getByLabelText(/Password/i)).toBeInTheDocument();
        expect(screen.getByRole('button', { name: /Sign In/i })).toBeInTheDocument();
    });
});
