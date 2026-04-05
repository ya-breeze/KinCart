import React, { useState, useCallback, useMemo } from 'react';
import { useNavigate } from 'react-router-dom';
import { useAuth } from '../context/AuthContext';
import { ArrowLeft, Search, X, Loader2, TrendingUp } from 'lucide-react';
import {
    LineChart,
    Line,
    XAxis,
    YAxis,
    CartesianGrid,
    Tooltip,
    Legend,
    ResponsiveContainer
} from 'recharts';
import { usePriceHistory } from '../hooks/usePriceHistory';

const PERIOD_OPTIONS = [
    { value: '3m', label: '3m' },
    { value: '6m', label: '6m' },
    { value: '1y', label: '1y' },
    { value: 'all', label: 'All' },
];

const CustomTooltip = ({ active, payload, label, currency }) => {
    if (!active || !payload || !payload.length) return null;
    const formatP = (price) => `${price.toFixed(2)} ${currency || 'Kč'}`;
    return (
        <div style={{
            background: 'white',
            border: '1px solid var(--border)',
            borderRadius: '8px',
            padding: '0.75rem',
            boxShadow: 'var(--shadow-md)',
            fontSize: '0.875rem',
            maxWidth: '220px'
        }}>
            <div style={{ fontWeight: 700, marginBottom: '0.5rem' }}>{new Date(label).toLocaleDateString()}</div>
            {payload.map((entry, i) => (
                <div key={i} style={{ color: entry.color, marginBottom: '0.25rem' }}>
                    <strong>{entry.name}:</strong> {formatP(entry.value)}
                    {entry.payload?.name && <div style={{ color: 'var(--text)', fontSize: '0.8rem' }}>{entry.payload.name}</div>}
                    {entry.payload?.quantity && <div style={{ color: 'var(--text-muted)', fontSize: '0.8rem' }}>{entry.payload.quantity}</div>}
                </div>
            ))}
        </div>
    );
};

function parseSearchText(text) {
    const parts = text.trim().split(/\s+/).filter(Boolean);
    const terms = [];
    const excludes = [];
    for (const part of parts) {
        if (part.startsWith('-') && part.length > 1) {
            excludes.push(part.slice(1));
        } else {
            terms.push(part);
        }
    }
    return { query: terms.join(' '), excludes };
}

const PriceHistoryPage = () => {
    const { currency } = useAuth();
    const navigate = useNavigate();
    const [searchText, setSearchText] = useState('');
    const [period, setPeriod] = useState('6m');
    const [hiddenShops, setHiddenShops] = useState(new Set());
    const [sortBy, setSortBy] = useState('date');

    const { query, excludes } = useMemo(() => parseSearchText(searchText), [searchText]);

    const { chartData, items, pagination, loading, loadingMore, loadMore } = usePriceHistory(
        query, excludes, period
    );

    const removeExclude = useCallback((word) => {
        setSearchText(prev => {
            const parts = prev.trim().split(/\s+/).filter(Boolean);
            return parts.filter(p => p !== `-${word}`).join(' ');
        });
    }, []);

    const toggleShop = useCallback((shopName) => {
        setHiddenShops(prev => {
            const next = new Set(prev);
            if (next.has(shopName)) next.delete(shopName);
            else next.add(shopName);
            return next;
        });
    }, []);

    const formatPrice = (price) => `${price.toFixed(2)} ${currency || 'Kč'}`;
    const sortedItems = useMemo(() => {
        if (!items) return [];
        return [...items].sort((a, b) => {
            if (sortBy === 'price') return a.price - b.price;
            return new Date(b.start_date) - new Date(a.start_date);
        });
    }, [items, sortBy]);

    const shopColorMap = useMemo(() => {
        const map = {};
        for (const shop of chartData) {
            map[shop.shop_name] = shop.color;
        }
        return map;
    }, [chartData]);

    const totalPoints = chartData.reduce((sum, s) => sum + (s.points?.length || 0), 0);
    const visibleChartData = chartData.filter(s => !hiddenShops.has(s.shop_name));
    const hasResults = chartData.length > 0;

    return (
        <div className="container" style={{ paddingBottom: '5rem' }}>
            <header style={{ marginBottom: '2rem', paddingTop: '1rem' }}>
                <button
                    onClick={() => navigate('/flyers')}
                    style={{
                        background: 'none',
                        border: 'none',
                        color: 'var(--primary)',
                        display: 'flex',
                        alignItems: 'center',
                        gap: '0.5rem',
                        cursor: 'pointer',
                        padding: 0,
                        marginBottom: '1rem',
                        fontWeight: 600
                    }}
                >
                    <ArrowLeft size={20} />
                    Back to Flyer Items
                </button>
                <h1 style={{ fontSize: '1.5rem', fontWeight: 800, display: 'flex', alignItems: 'center', gap: '0.5rem' }}>
                    <TrendingUp size={24} />
                    Price History
                </h1>
            </header>

            {/* Search Section */}
            <section className="card" style={{ marginBottom: '2rem', padding: '1.5rem' }}>
                <div style={{ marginBottom: '1rem' }}>
                    <div style={{ position: 'relative' }}>
                        <Search size={18} style={{ position: 'absolute', left: '12px', top: '50%', transform: 'translateY(-50%)', color: 'var(--text-muted)' }} />
                        <input
                            type="text"
                            value={searchText}
                            onChange={e => setSearchText(e.target.value)}
                            placeholder="e.g. banana -candy -flavour"
                            style={{ paddingLeft: '2.5rem', width: '100%' }}
                        />
                    </div>
                    <p style={{ fontSize: '0.75rem', color: 'var(--text-muted)', marginTop: '0.5rem', margin: '0.5rem 0 0' }}>
                        Tip: Use -word to exclude (e.g. banana -candy)
                    </p>
                    {excludes.length > 0 && (
                        <div style={{ display: 'flex', flexWrap: 'wrap', alignItems: 'center', gap: '0.5rem', marginTop: '0.5rem' }}>
                            <span style={{ fontSize: '0.75rem', color: 'var(--text-muted)' }}>Excluding:</span>
                            {excludes.map(word => (
                                <button
                                    key={word}
                                    onClick={() => removeExclude(word)}
                                    style={{
                                        display: 'flex',
                                        alignItems: 'center',
                                        gap: '0.25rem',
                                        padding: '0.25rem 0.5rem',
                                        borderRadius: '999px',
                                        border: '1px solid var(--border)',
                                        background: 'var(--bg-main)',
                                        cursor: 'pointer',
                                        fontSize: '0.75rem',
                                        fontWeight: 600
                                    }}
                                >
                                    <X size={12} />
                                    {word}
                                </button>
                            ))}
                        </div>
                    )}
                </div>

                <div style={{ display: 'flex', alignItems: 'center', gap: '0.75rem' }}>
                    <span style={{ fontSize: '0.875rem', fontWeight: 600 }}>Period:</span>
                    <div style={{ display: 'flex', gap: '0.5rem' }}>
                        {PERIOD_OPTIONS.map(opt => (
                            <button
                                key={opt.value}
                                onClick={() => setPeriod(opt.value)}
                                style={{
                                    padding: '0.375rem 0.75rem',
                                    borderRadius: '6px',
                                    border: '1px solid',
                                    borderColor: period === opt.value ? 'var(--primary)' : 'var(--border)',
                                    background: period === opt.value ? 'var(--primary)' : 'white',
                                    color: period === opt.value ? 'white' : 'var(--text)',
                                    cursor: 'pointer',
                                    fontSize: '0.875rem',
                                    fontWeight: period === opt.value ? 700 : 400
                                }}
                            >
                                {opt.label}
                            </button>
                        ))}
                    </div>
                </div>
            </section>

            {loading ? (
                <div style={{ textAlign: 'center', padding: '5rem' }}>
                    <Loader2 className="spin" size={48} style={{ color: 'var(--primary)', opacity: 0.5 }} />
                    <p style={{ marginTop: '1rem', color: 'var(--text-muted)' }}>Loading...</p>
                </div>
            ) : !query ? (
                <div style={{ textAlign: 'center', padding: '5rem', background: 'white', borderRadius: '1rem', border: '2px dashed var(--border)' }}>
                    <TrendingUp size={48} style={{ color: 'var(--text-muted)', marginBottom: '1rem', opacity: 0.5 }} />
                    <h3>Enter a search term to see price history</h3>
                    <p style={{ color: 'var(--text-muted)' }}>e.g. &quot;banana&quot; or &quot;milk -organic&quot;</p>
                </div>
            ) : !hasResults ? (
                <div style={{ textAlign: 'center', padding: '5rem', background: 'white', borderRadius: '1rem', border: '2px dashed var(--border)' }}>
                    <TrendingUp size={48} style={{ color: 'var(--text-muted)', marginBottom: '1rem', opacity: 0.5 }} />
                    <h3>No price history found</h3>
                    <p style={{ color: 'var(--text-muted)' }}>Try a different search term or extend the period</p>
                </div>
            ) : (
                <>
                    <div style={{ marginBottom: '1rem', color: 'var(--text-muted)', fontSize: '0.875rem' }}>
                        Found {totalPoints} price points across {chartData.length} {chartData.length === 1 ? 'shop' : 'shops'}
                    </div>

                    {/* Chart */}
                    <section className="card" style={{ marginBottom: '2rem', padding: '1.5rem' }}>
                        <ResponsiveContainer width="100%" height={300}>
                            <LineChart margin={{ top: 5, right: 20, left: 0, bottom: 5 }}>
                                <CartesianGrid strokeDasharray="3 3" stroke="var(--border)" />
                                <XAxis
                                    dataKey="ts"
                                    type="number"
                                    scale="time"
                                    domain={['auto', 'auto']}
                                    tickFormatter={ts => new Date(ts).toLocaleDateString(undefined, { month: 'short', day: 'numeric' })}
                                    tick={{ fontSize: 12, fill: 'var(--text-muted)' }}
                                />
                                <YAxis
                                    tickFormatter={v => `${v}`}
                                    tick={{ fontSize: 12, fill: 'var(--text-muted)' }}
                                    width={45}
                                />
                                <Tooltip content={<CustomTooltip currency={currency} />} />
                                <Legend
                                    onClick={e => toggleShop(e.value)}
                                    formatter={(value) => (
                                        <span style={{
                                            textDecoration: hiddenShops.has(value) ? 'line-through' : 'none',
                                            color: hiddenShops.has(value) ? 'var(--text-muted)' : 'var(--text)',
                                            cursor: 'pointer',
                                            fontSize: '0.875rem'
                                        }}>
                                            {value} ({chartData.find(s => s.shop_name === value)?.points?.length || 0})
                                        </span>
                                    )}
                                />
                                {visibleChartData.map(shop => (
                                    <Line
                                        key={shop.shop_name}
                                        data={shop.points}
                                        dataKey="price"
                                        name={shop.shop_name}
                                        stroke={shop.color}
                                        dot={true}
                                        activeDot={{ r: 6 }}
                                        type="monotone"
                                        connectNulls={false}
                                    />
                                ))}
                            </LineChart>
                        </ResponsiveContainer>
                    </section>

                    {/* Items Table */}
                    <section className="card" style={{ padding: '1.5rem' }}>
                        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '1rem', flexWrap: 'wrap', gap: '0.5rem' }}>
                            <h3 style={{ margin: 0, fontWeight: 700 }}>Matching Items ({pagination?.total || 0})</h3>
                            <div style={{ display: 'flex', alignItems: 'center', gap: '0.5rem' }}>
                                <span style={{ fontSize: '0.875rem', color: 'var(--text-muted)' }}>Sort:</span>
                                <select
                                    value={sortBy}
                                    onChange={e => setSortBy(e.target.value)}
                                    style={{ fontSize: '0.875rem', padding: '0.25rem 0.5rem', borderRadius: '6px', border: '1px solid var(--border)' }}
                                >
                                    <option value="date">Date</option>
                                    <option value="price">Price</option>
                                </select>
                            </div>
                        </div>

                        <div>
                            {sortedItems.map(item => (
                                <div
                                    key={item.id}
                                    style={{
                                        display: 'flex',
                                        alignItems: 'center',
                                        gap: '0.75rem',
                                        padding: '0.75rem 0',
                                        borderBottom: '1px solid var(--border)',
                                        fontSize: '0.875rem'
                                    }}
                                >
                                    <span style={{
                                        width: '10px',
                                        height: '10px',
                                        borderRadius: '50%',
                                        background: shopColorMap[item.shop_name] || '#999',
                                        flexShrink: 0
                                    }} />
                                    <span style={{ color: 'var(--text-muted)', minWidth: '70px', flexShrink: 0 }}>{item.shop_name}</span>
                                    <span style={{ flex: 1, fontWeight: 600, minWidth: 0, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>{item.name}</span>
                                    <span style={{ color: 'var(--text-muted)', fontSize: '0.75rem', flexShrink: 0 }}>{item.quantity}</span>
                                    <span style={{ fontWeight: 700, color: 'var(--primary)', flexShrink: 0 }}>{formatPrice(item.price)}</span>
                                    <span style={{ color: 'var(--text-muted)', fontSize: '0.75rem', flexShrink: 0 }}>
                                        {new Date(item.start_date).toLocaleDateString()}
                                    </span>
                                </div>
                            ))}
                        </div>

                        {pagination?.has_more && (
                            <button
                                onClick={loadMore}
                                disabled={loadingMore}
                                style={{
                                    marginTop: '1rem',
                                    width: '100%',
                                    padding: '0.75rem',
                                    borderRadius: '8px',
                                    border: '1px solid var(--border)',
                                    background: 'white',
                                    cursor: loadingMore ? 'default' : 'pointer',
                                    display: 'flex',
                                    alignItems: 'center',
                                    justifyContent: 'center',
                                    gap: '0.5rem',
                                    fontWeight: 600
                                }}
                            >
                                {loadingMore && <Loader2 className="spin" size={18} />}
                                {loadingMore ? 'Loading...' : 'Load more...'}
                            </button>
                        )}
                    </section>
                </>
            )}
        </div>
    );
};

export default PriceHistoryPage;
