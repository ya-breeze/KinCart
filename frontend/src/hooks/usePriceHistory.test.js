import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { renderHook, act } from '@testing-library/react';
import { usePriceHistory } from './usePriceHistory';

const makeApiResponse = (overrides = {}) => ({
    chart_data: [
        {
            shop_name: 'ShopA',
            color: '#ff0000',
            points: [
                { date: '2024-03-15', price: 2.50, name: 'Banana' },
                { date: '2024-01-10', price: 3.00, name: 'Banana' },
                { date: '2024-02-20', price: 2.75, name: 'Banana' },
            ],
        },
        {
            shop_name: 'ShopB',
            color: '#0000ff',
            points: [
                { date: '2024-02-01', price: 2.90, name: 'Banana' },
                { date: '2024-01-05', price: 3.10, name: 'Banana' },
            ],
        },
    ],
    items: [{ id: 1, name: 'Banana', price: 2.50, shop_name: 'ShopA', start_date: '2024-03-15' }],
    pagination: { total: 1, has_more: false, page: 1, limit: 50 },
    ...overrides,
});

const flush = () => act(async () => { await vi.runAllTimersAsync(); });

describe('usePriceHistory', () => {
    beforeEach(() => {
        vi.useFakeTimers();
        globalThis.fetch = vi.fn(() =>
            Promise.resolve({
                ok: true,
                json: () => Promise.resolve(makeApiResponse()),
            })
        );
    });

    afterEach(() => {
        vi.useRealTimers();
        vi.restoreAllMocks();
    });

    it('returns empty state when query is empty', async () => {
        const { result } = renderHook(() =>
            usePriceHistory('', [], '6m')
        );

        await flush();

        expect(result.current.chartData).toEqual([]);
        expect(result.current.items).toEqual([]);
        expect(result.current.pagination).toBeNull();
        expect(fetch).not.toHaveBeenCalled();
    });

    it('fetches data after debounce when query is provided', async () => {
        renderHook(() => usePriceHistory('banana', [], '6m'));

        expect(fetch).not.toHaveBeenCalled();

        await flush();

        expect(fetch).toHaveBeenCalledOnce();
        const url = fetch.mock.calls[0][0];
        expect(url).toContain('/api/flyers/items/history');
        expect(url).toContain('q=banana');
        expect(url).toContain('period=6m');
    });

    it('includes exclude param when excludeWords provided', async () => {
        renderHook(() => usePriceHistory('banana', ['candy', 'flavour'], '6m'));

        await flush();

        expect(fetch).toHaveBeenCalledOnce();
        expect(fetch.mock.calls[0][0]).toContain('exclude=candy%2Cflavour');
    });

    it('sorts each shop points by date ascending', async () => {
        const { result } = renderHook(() =>
            usePriceHistory('banana', [], '6m')
        );

        await flush();

        const shopAPoints = result.current.chartData[0].points;
        expect(shopAPoints[0].date).toBe('2024-01-10');
        expect(shopAPoints[1].date).toBe('2024-02-20');
        expect(shopAPoints[2].date).toBe('2024-03-15');

        const shopBPoints = result.current.chartData[1].points;
        expect(shopBPoints[0].date).toBe('2024-01-05');
        expect(shopBPoints[1].date).toBe('2024-02-01');
    });

    it('adds ts (millisecond timestamp) to each point', async () => {
        const { result } = renderHook(() =>
            usePriceHistory('banana', [], '6m')
        );

        await flush();

        for (const shop of result.current.chartData) {
            for (const point of shop.points) {
                expect(typeof point.ts).toBe('number');
                expect(point.ts).toBe(new Date(point.date).getTime());
            }
        }
    });

    it('ts values are proportional to real time gaps', async () => {
        const { result } = renderHook(() =>
            usePriceHistory('banana', [], '6m')
        );

        await flush();

        const points = result.current.chartData[0].points;
        const DAY_MS = 24 * 60 * 60 * 1000;

        // Jan 10 → Feb 20 = 41 days; Feb 20 → Mar 15 = 24 days
        const gap1 = points[1].ts - points[0].ts;
        const gap2 = points[2].ts - points[1].ts;
        expect(gap1).toBe(41 * DAY_MS);
        expect(gap2).toBe(24 * DAY_MS);
    });

    it('resets chart data and items when query changes', async () => {
        let query = 'banana';
        const { result, rerender } = renderHook(() =>
            usePriceHistory(query, [], '6m')
        );

        await flush();
        expect(result.current.chartData).toHaveLength(2);

        act(() => { query = 'milk'; });
        rerender();

        expect(result.current.chartData).toEqual([]);
        expect(result.current.items).toEqual([]);
    });

    it('loadMore fetches next page and appends items', async () => {
        globalThis.fetch = vi.fn()
            .mockResolvedValueOnce({
                ok: true,
                json: () => Promise.resolve(makeApiResponse({
                    items: [{ id: 1, name: 'Banana', price: 2.50, shop_name: 'ShopA', start_date: '2024-03-15' }],
                    pagination: { total: 2, has_more: true, page: 1, limit: 1 },
                })),
            })
            .mockResolvedValueOnce({
                ok: true,
                json: () => Promise.resolve(makeApiResponse({
                    items: [{ id: 2, name: 'Banana', price: 2.90, shop_name: 'ShopB', start_date: '2024-02-01' }],
                    pagination: { total: 2, has_more: false, page: 2, limit: 1 },
                })),
            });

        const { result } = renderHook(() =>
            usePriceHistory('banana', [], '6m')
        );

        await flush();
        expect(result.current.items).toHaveLength(1);

        await act(async () => {
            result.current.loadMore();
            await vi.runAllTimersAsync();
        });

        expect(result.current.items).toHaveLength(2);
        expect(result.current.items[0].id).toBe(1);
        expect(result.current.items[1].id).toBe(2);
    });
});
