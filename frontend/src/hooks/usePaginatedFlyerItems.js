import { useState, useEffect, useCallback, useRef } from 'react';
import { API_BASE_URL } from '../config';

export const usePaginatedFlyerItems = (token, filters) => {
    const [items, setItems] = useState([]);
    const [loading, setLoading] = useState(true);
    const [loadingMore, setLoadingMore] = useState(false);
    const [hasMore, setHasMore] = useState(true);
    const [page, setPage] = useState(1);
    const [totalCount, setTotalCount] = useState(0);
    const [error, setError] = useState(null);

    // Cache for filter combinations
    const cacheRef = useRef(new Map());
    const abortControllerRef = useRef(null);

    // Generate cache key from filters and page
    const getCacheKey = useCallback((pageNum) => {
        return JSON.stringify({ ...filters, page: pageNum });
    }, [filters]);

    // Reset pagination when filters change
    useEffect(() => {
        setPage(1);
        setItems([]);
        setHasMore(true);
    }, [filters.q, filters.shop, filters.activity]);

    // Fetch items for current page
    const fetchItems = useCallback(async (pageNum, append = false) => {
        const cacheKey = getCacheKey(pageNum);

        // Check cache first
        if (cacheRef.current.has(cacheKey)) {
            const cached = cacheRef.current.get(cacheKey);
            if (append) {
                setItems(prev => [...prev, ...cached.items]);
            } else {
                setItems(cached.items);
            }
            setTotalCount(cached.totalCount);
            setHasMore(cached.hasMore);
            setLoading(false);
            setLoadingMore(false);
            return;
        }

        // Abort previous request if still pending
        if (abortControllerRef.current) {
            abortControllerRef.current.abort();
        }

        abortControllerRef.current = new AbortController();

        try {
            append ? setLoadingMore(true) : setLoading(true);

            const params = new URLSearchParams({
                ...filters,
                page: pageNum,
                limit: 24
            });

            const resp = await fetch(
                `${API_BASE_URL}/api/flyers/items?${params.toString()}`,
                {
                    headers: { 'Authorization': `Bearer ${token}` },
                    signal: abortControllerRef.current.signal
                }
            );

            if (resp.ok) {
                const data = await resp.json();

                // Cache the result
                cacheRef.current.set(cacheKey, {
                    items: data.items || [],
                    totalCount: data.pagination?.total || 0,
                    hasMore: data.pagination?.has_more || false
                });

                // Limit cache size to 20 entries (approx 480 items max in memory)
                if (cacheRef.current.size > 20) {
                    const firstKey = cacheRef.current.keys().next().value;
                    cacheRef.current.delete(firstKey);
                }

                if (append) {
                    setItems(prev => [...prev, ...(data.items || [])]);
                } else {
                    setItems(data.items || []);
                }

                setTotalCount(data.pagination?.total || 0);
                setHasMore(data.pagination?.has_more || false);
                setError(null);
            } else {
                throw new Error(`HTTP ${resp.status}: ${resp.statusText}`);
            }
        } catch (err) {
            if (err.name !== 'AbortError') {
                console.error('Failed to fetch items:', err);
                setError(err);
            }
        } finally {
            setLoading(false);
            setLoadingMore(false);
        }
    }, [token, filters, getCacheKey]);

    // Load more items
    const loadMore = useCallback(() => {
        if (!loadingMore && hasMore) {
            const nextPage = page + 1;
            setPage(nextPage);
            fetchItems(nextPage, true);
        }
    }, [page, hasMore, loadingMore, fetchItems]);

    // Initial fetch with debounce
    useEffect(() => {
        const timer = setTimeout(() => {
            fetchItems(1, false);
        }, 300);

        return () => {
            clearTimeout(timer);
            if (abortControllerRef.current) {
                abortControllerRef.current.abort();
            }
        };
    }, [fetchItems]);

    // Clear cache when needed (e.g., new items added)
    const clearCache = useCallback(() => {
        cacheRef.current.clear();
    }, []);

    return {
        items,
        loading,
        loadingMore,
        hasMore,
        totalCount,
        error,
        loadMore,
        clearCache
    };
};
