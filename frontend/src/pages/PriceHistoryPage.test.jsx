/** @vitest-environment happy-dom */
import { render, screen, fireEvent } from '@testing-library/react';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import { BrowserRouter } from 'react-router-dom';
import PriceHistoryPage from './PriceHistoryPage';

vi.mock('../context/AuthContext', () => ({
    useAuth: () => ({ currency: 'Kč' }),
}));

vi.mock('recharts', () => {
    const Line = ({ name }) => <div data-testid={`line-${name}`} />;
    return {
        LineChart: ({ children }) => <div data-testid="line-chart">{children}</div>,
        Line,
        XAxis: () => null,
        YAxis: () => null,
        CartesianGrid: () => null,
        Tooltip: () => null,
        Legend: ({ formatter }) => {
            const rendered = formatter('ShopA', { value: 'ShopA' });
            return <div data-testid="legend">{rendered}</div>;
        },
        ResponsiveContainer: ({ children }) => <div>{children}</div>,
    };
});

const mockChartData = [
    {
        shop_name: 'ShopA',
        color: '#ff0000',
        points: [
            { date: '2024-01-10', ts: new Date('2024-01-10').getTime(), price: 3.00 },
            { date: '2024-02-20', ts: new Date('2024-02-20').getTime(), price: 2.75 },
        ],
    },
    {
        shop_name: 'ShopB',
        color: '#0000ff',
        points: [
            { date: '2024-01-05', ts: new Date('2024-01-05').getTime(), price: 3.10 },
        ],
    },
];

const mockItems = [
    { id: 1, name: 'Banana large', price: 3.00, shop_name: 'ShopA', start_date: '2024-01-10', quantity: '1kg' },
    { id: 2, name: 'Banana small', price: 2.75, shop_name: 'ShopA', start_date: '2024-02-20', quantity: '500g' },
];

vi.mock('../hooks/usePriceHistory', () => ({
    usePriceHistory: vi.fn(),
}));

import { usePriceHistory } from '../hooks/usePriceHistory';

const emptyHook = {
    chartData: [], items: [], pagination: null,
    loading: false, loadingMore: false, error: null, loadMore: vi.fn(),
};

const renderPage = () =>
    render(<BrowserRouter><PriceHistoryPage /></BrowserRouter>);

describe('PriceHistoryPage', () => {
    beforeEach(() => {
        vi.clearAllMocks();
        usePriceHistory.mockReturnValue(emptyHook);
    });

    it('shows prompt when no search query entered', () => {
        renderPage();
        expect(screen.getByText(/enter a search term/i)).toBeInTheDocument();
    });

    it('shows loading spinner while fetching', () => {
        usePriceHistory.mockReturnValue({ ...emptyHook, loading: true });
        renderPage();

        const input = screen.getByPlaceholderText(/banana/i);
        fireEvent.change(input, { target: { value: 'banana' } });

        expect(screen.getByText(/loading/i)).toBeInTheDocument();
    });

    it('shows no results message when chart data is empty after search', async () => {
        usePriceHistory.mockImplementation((q) =>
            q ? { ...emptyHook, chartData: [], items: [] } : emptyHook
        );

        renderPage();
        fireEvent.change(screen.getByPlaceholderText(/banana/i), { target: { value: 'xyz' } });

        expect(screen.getByText(/no price history found/i)).toBeInTheDocument();
    });

    it('renders chart and items when data is available', () => {
        usePriceHistory.mockImplementation((q) =>
            q ? {
                ...emptyHook,
                chartData: mockChartData,
                items: mockItems,
                pagination: { total: 2, has_more: false },
            } : emptyHook
        );

        renderPage();
        fireEvent.change(screen.getByPlaceholderText(/banana/i), { target: { value: 'banana' } });

        expect(screen.getByTestId('line-chart')).toBeInTheDocument();
        expect(screen.getByTestId('line-ShopA')).toBeInTheDocument();
        expect(screen.getByTestId('line-ShopB')).toBeInTheDocument();
        expect(screen.getByText('Banana large')).toBeInTheDocument();
        expect(screen.getByText('Banana small')).toBeInTheDocument();
    });

    it('shows correct price point count summary', () => {
        usePriceHistory.mockImplementation((q) =>
            q ? {
                ...emptyHook,
                chartData: mockChartData,
                items: mockItems,
                pagination: { total: 2, has_more: false },
            } : emptyHook
        );

        renderPage();
        fireEvent.change(screen.getByPlaceholderText(/banana/i), { target: { value: 'banana' } });

        expect(screen.getByText(/3 price points across 2 shops/i)).toBeInTheDocument();
    });

    it('excludes tokens shown as removable chips', () => {
        renderPage();
        fireEvent.change(screen.getByPlaceholderText(/banana/i), { target: { value: 'banana -candy' } });

        expect(screen.getByText('candy')).toBeInTheDocument();
    });

    it('removes exclude chip on click', () => {
        renderPage();
        const input = screen.getByPlaceholderText(/banana/i);
        fireEvent.change(input, { target: { value: 'banana -candy' } });

        fireEvent.click(screen.getByText('candy').closest('button'));

        expect(screen.queryByText('candy')).not.toBeInTheDocument();
        expect(input.value).toBe('banana');
    });

    it('period buttons update the active period', () => {
        renderPage();

        const btn3m = screen.getByText('3m');
        fireEvent.click(btn3m);

        expect(usePriceHistory).toHaveBeenCalledWith(
            expect.anything(), expect.anything(), '3m'
        );
    });

    it('show load more button when has_more is true', () => {
        const loadMore = vi.fn();
        usePriceHistory.mockImplementation((q) =>
            q ? {
                ...emptyHook,
                chartData: mockChartData,
                items: mockItems,
                pagination: { total: 10, has_more: true },
                loadMore,
            } : emptyHook
        );

        renderPage();
        fireEvent.change(screen.getByPlaceholderText(/banana/i), { target: { value: 'banana' } });

        const btn = screen.getByText(/load more/i);
        expect(btn).toBeInTheDocument();
        fireEvent.click(btn);
        expect(loadMore).toHaveBeenCalledOnce();
    });
});
