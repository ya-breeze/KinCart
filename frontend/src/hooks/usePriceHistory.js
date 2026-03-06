import { useState, useEffect, useCallback, useRef } from 'react';
import { API_BASE_URL } from '../config';

export const usePriceHistory = (token, q, excludeWords, period) => {
    const [chartData, setChartData] = useState([]);
    const [items, setItems] = useState([]);
    const [pagination, setPagination] = useState(null);
    const [loading, setLoading] = useState(false);
    const [loadingMore, setLoadingMore] = useState(false);
    const [error, setError] = useState(null);

    const abortControllerRef = useRef(null);
    const pageRef = useRef(1);

    const excludeStr = excludeWords.join(',');

    const fetchData = useCallback(async (pageNum, append = false) => {
        if (!q) {
            setChartData([]);
            setItems([]);
            setPagination(null);
            setLoading(false);
            return;
        }

        if (abortControllerRef.current) {
            abortControllerRef.current.abort();
        }
        abortControllerRef.current = new AbortController();

        try {
            append ? setLoadingMore(true) : setLoading(true);

            const params = new URLSearchParams({ q, period, page: pageNum, limit: 50 });
            if (excludeStr) {
                params.set('exclude', excludeStr);
            }

            const resp = await fetch(
                `${API_BASE_URL}/api/flyers/items/history?${params}`,
                {
                    headers: { Authorization: `Bearer ${token}` },
                    signal: abortControllerRef.current.signal
                }
            );

            if (resp.ok) {
                const data = await resp.json();
                if (append) {
                    setItems(prev => [...prev, ...(data.items || [])]);
                } else {
                    setChartData(data.chart_data || []);
                    setItems(data.items || []);
                }
                setPagination(data.pagination);
                setError(null);
            } else {
                throw new Error(`HTTP ${resp.status}`);
            }
        } catch (err) {
            if (err.name !== 'AbortError') {
                setError(err);
            }
        } finally {
            setLoading(false);
            setLoadingMore(false);
        }
    }, [token, q, excludeStr, period]);

    // Reset on param change
    useEffect(() => {
        pageRef.current = 1;
        setItems([]);
        setChartData([]);
        setPagination(null);
    }, [q, excludeStr, period]);

    // Debounced fetch
    useEffect(() => {
        const timer = setTimeout(() => {
            fetchData(1, false);
        }, 500);

        return () => {
            clearTimeout(timer);
            if (abortControllerRef.current) {
                abortControllerRef.current.abort();
            }
        };
    }, [fetchData]);

    const loadMore = useCallback(() => {
        if (!loadingMore && pagination?.has_more) {
            pageRef.current += 1;
            fetchData(pageRef.current, true);
        }
    }, [loadingMore, pagination, fetchData]);

    return { chartData, items, pagination, loading, loadingMore, error, loadMore };
};
