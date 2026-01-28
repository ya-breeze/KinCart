import React, { useState, useEffect, useCallback } from 'react';
import { useAuth } from '../context/AuthContext';
import { Search, Store, Calendar, ArrowLeft, Loader2, Filter } from 'lucide-react';
import { useNavigate } from 'react-router-dom';
import { API_BASE_URL } from '../config';

const FlyerItemsPage = () => {
    const { token, currency } = useAuth();
    const navigate = useNavigate();
    const [items, setItems] = useState([]);
    const [shops, setShops] = useState([]);
    const [loading, setLoading] = useState(true);
    const [filters, setFilters] = useState({
        q: '',
        shop: '',
        activity: 'now'
    });

    const fetchShops = async () => {
        try {
            const resp = await fetch(`${API_BASE_URL}/api/flyers/shops`, {
                headers: { 'Authorization': `Bearer ${token}` }
            });
            if (resp.ok) {
                setShops(await resp.json());
            }
        } catch (err) {
            console.error('Failed to fetch shops:', err);
        }
    };

    const fetchItems = useCallback(async () => {
        setLoading(true);
        try {
            const params = new URLSearchParams(filters);
            const resp = await fetch(`${API_BASE_URL}/api/flyers/items?${params.toString()}`, {
                headers: { 'Authorization': `Bearer ${token}` }
            });
            if (resp.ok) {
                setItems(await resp.json());
            }
        } catch (err) {
            console.error('Failed to fetch items:', err);
        } finally {
            setLoading(false);
        }
    }, [token, filters]);

    useEffect(() => {
        fetchShops();
    }, []);

    useEffect(() => {
        const timer = setTimeout(() => {
            fetchItems();
        }, 300);
        return () => clearTimeout(timer);
    }, [fetchItems]);

    const handleFilterChange = (e) => {
        const { name, value } = e.target;
        setFilters(prev => ({ ...prev, [name]: value }));
    };

    return (
        <div className="container">
            <header style={{ marginBottom: '2rem', paddingTop: '1rem' }}>
                <button
                    onClick={() => navigate('/')}
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
                    Back to Dashboard
                </button>
                <h1 style={{ fontSize: '1.5rem', fontWeight: 800 }}>Flyer Items</h1>
                <p style={{ color: 'var(--text-muted)' }}>Browse and filter current deals from all shops</p>
            </header>

            <section className="card" style={{ marginBottom: '2rem', padding: '1.5rem' }}>
                <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(200px, 1fr))', gap: '1rem' }}>
                    <div className="input-group">
                        <label style={{ display: 'block', marginBottom: '0.5rem', fontSize: '0.875rem', fontWeight: 600 }}>Search</label>
                        <div style={{ position: 'relative' }}>
                            <Search size={18} style={{ position: 'absolute', left: '12px', top: '50%', transform: 'translateY(-50%)', color: 'var(--text-muted)' }} />
                            <input
                                type="text"
                                name="q"
                                value={filters.q}
                                onChange={handleFilterChange}
                                placeholder="Name, category, keyword..."
                                style={{ paddingLeft: '2.5rem', width: '100%' }}
                            />
                        </div>
                    </div>

                    <div className="input-group">
                        <label style={{ display: 'block', marginBottom: '0.5rem', fontSize: '0.875rem', fontWeight: 600 }}>Shop</label>
                        <div style={{ position: 'relative' }}>
                            <Store size={18} style={{ position: 'absolute', left: '12px', top: '50%', transform: 'translateY(-50%)', color: 'var(--text-muted)' }} />
                            <select
                                name="shop"
                                value={filters.shop}
                                onChange={handleFilterChange}
                                style={{ paddingLeft: '2.5rem', width: '100%', appearance: 'none' }}
                            >
                                <option value="">All Shops</option>
                                {shops.map(shop => (
                                    <option key={shop} value={shop}>{shop}</option>
                                ))}
                            </select>
                        </div>
                    </div>

                    <div className="input-group">
                        <label style={{ display: 'block', marginBottom: '0.5rem', fontSize: '0.875rem', fontWeight: 600 }}>Activity</label>
                        <div style={{ position: 'relative' }}>
                            <Calendar size={18} style={{ position: 'absolute', left: '12px', top: '50%', transform: 'translateY(-50%)', color: 'var(--text-muted)' }} />
                            <select
                                name="activity"
                                value={filters.activity}
                                onChange={handleFilterChange}
                                style={{ paddingLeft: '2.5rem', width: '100%', appearance: 'none' }}
                            >
                                <option value="now">Active Now</option>
                                <option value="future">Starting Soon</option>
                                <option value="all">All (Not Outdated)</option>
                            </select>
                        </div>
                    </div>
                </div>
            </section>

            {loading ? (
                <div style={{ textAlign: 'center', padding: '5rem' }}>
                    <Loader2 className="spin" size={48} style={{ color: 'var(--primary)', opacity: 0.5 }} />
                    <p style={{ marginTop: '1rem', color: 'var(--text-muted)' }}>Loading items...</p>
                </div>
            ) : items.length === 0 ? (
                <div style={{ textAlign: 'center', padding: '5rem', background: 'white', borderRadius: '1rem', border: '2px dashed var(--border)' }}>
                    <Filter size={48} style={{ color: 'var(--text-muted)', marginBottom: '1rem', opacity: 0.5 }} />
                    <h3>No items found</h3>
                    <p style={{ color: 'var(--text-muted)' }}>Try adjusting your search or filters</p>
                </div>
            ) : (
                <div style={{
                    display: 'grid',
                    gridTemplateColumns: 'repeat(auto-fill, minmax(280px, 1fr))',
                    gap: '1.5rem'
                }}>
                    {items.map(item => (
                        <div key={item.id} className="card" style={{ padding: 0, overflow: 'hidden', display: 'flex', flexDirection: 'column' }}>
                            <div style={{ height: '200px', background: '#f8f9fa', position: 'relative', overflow: 'hidden' }}>
                                {item.local_photo_path ? (
                                    <img
                                        src={`${API_BASE_URL}${item.local_photo_path}`}
                                        alt={item.name}
                                        style={{ width: '100%', height: '100%', objectFit: 'contain' }}
                                    />
                                ) : (
                                    <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'center', height: '100%', color: 'var(--text-muted)' }}>
                                        No Image
                                    </div>
                                )}
                            </div>
                            <div style={{ padding: '1rem', flex: 1, display: 'flex', flexDirection: 'column' }}>
                                <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', gap: '0.5rem', marginBottom: '0.5rem' }}>
                                    <h3 style={{ fontSize: '1.25rem', fontWeight: 800, margin: 0, color: 'var(--primary)', whiteSpace: 'nowrap' }}>
                                        {item.price} <span style={{ fontSize: '0.875rem', fontWeight: 600 }}>{currency}</span>
                                    </h3>
                                    {item.original_price && (
                                        <span style={{ fontSize: '0.875rem', color: 'var(--text-muted)', textDecoration: 'line-through' }}>
                                            {item.original_price} {currency}
                                        </span>
                                    )}
                                </div>
                                <h4 style={{ fontSize: '1rem', fontWeight: 600, marginBottom: '0.75rem', minHeight: '2.5rem', display: '-webkit-box', WebkitLineClamp: 2, WebkitBoxOrient: 'vertical', overflow: 'hidden', color: 'var(--text-main)' }}>
                                    {item.name}
                                </h4>
                                <div style={{ fontSize: '0.875rem', color: 'var(--text-muted)', marginTop: 'auto' }}>
                                    <div style={{ display: 'flex', alignItems: 'center', gap: '0.5rem', marginBottom: '0.25rem', color: 'var(--text-main)', fontWeight: 600 }}>
                                        <Store size={14} />
                                        <span>{item.shop_name}</span>
                                    </div>
                                    <div style={{ display: 'flex', alignItems: 'center', gap: '0.5rem' }}>
                                        <Calendar size={14} />
                                        <span>{new Date(item.start_date).toLocaleDateString()} - {new Date(item.end_date).toLocaleDateString()}</span>
                                    </div>
                                </div>
                                {item.categories && (
                                    <div style={{ marginTop: '0.75rem', display: 'flex', gap: '0.25rem', flexWrap: 'wrap' }}>
                                        {item.categories.split(',').map(cat => (
                                            <button
                                                key={cat}
                                                onClick={() => setFilters(prev => ({ ...prev, q: cat.trim() }))}
                                                style={{
                                                    fontSize: '0.7rem',
                                                    background: 'var(--bg-main)',
                                                    padding: '2px 8px',
                                                    borderRadius: '4px',
                                                    border: '1px solid var(--border)',
                                                    cursor: 'pointer',
                                                    color: 'var(--text-muted)',
                                                    transition: 'all 0.2s'
                                                }}
                                                className="tag-hover"
                                            >
                                                {cat.trim()}
                                            </button>
                                        ))}
                                    </div>
                                )}
                                {item.keywords && (
                                    <div style={{ marginTop: '0.5rem', display: 'flex', gap: '0.25rem', flexWrap: 'wrap' }}>
                                        {item.keywords.split(',').map(kw => (
                                            <button
                                                key={kw}
                                                onClick={() => setFilters(prev => ({ ...prev, q: kw.trim() }))}
                                                style={{
                                                    fontSize: '0.65rem',
                                                    background: 'transparent',
                                                    padding: '1px 6px',
                                                    borderRadius: '4px',
                                                    border: '1px dashed var(--border)',
                                                    cursor: 'pointer',
                                                    color: 'var(--text-muted)',
                                                    fontStyle: 'italic'
                                                }}
                                            >
                                                #{kw.trim()}
                                            </button>
                                        ))}
                                    </div>
                                )}
                            </div>
                        </div>
                    ))}
                </div>
            )}
        </div>
    );
};

export default FlyerItemsPage;
