import React, { useState, useEffect, useCallback } from 'react';
import { useAuth } from '../context/AuthContext';
import { Search, Store, Calendar, ArrowLeft, Loader2, Filter, Plus, ShoppingCart, Check, X, Tag } from 'lucide-react';
import { useNavigate } from 'react-router-dom';
import { API_BASE_URL } from '../config';
import ImageModal from '../components/ImageModal';

const FlyerItemsPage = () => {
    const { token, currency } = useAuth();
    const navigate = useNavigate();
    const [items, setItems] = useState([]);
    const [shops, setShops] = useState([]);
    const [categories, setCategories] = useState([]);
    const [loading, setLoading] = useState(true);
    const [filters, setFilters] = useState({
        q: '',
        shop: '',
        activity: 'now'
    });
    const [activeLists, setActiveLists] = useState([]);
    const [showListSelector, setShowListSelector] = useState(null); // flyerItemID
    const [addingTo, setAddingTo] = useState(null); // listId
    const [message, setMessage] = useState(null);
    const [previewImage, setPreviewImage] = useState(null);

    // Form state for adding item
    const [addForm, setAddForm] = useState({
        quantity: 1,
        unit: 'pcs',
        category_id: ''
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

    const fetchCategories = async () => {
        try {
            const resp = await fetch(`${API_BASE_URL}/api/categories`, {
                headers: { 'Authorization': `Bearer ${token}` }
            });
            if (resp.ok) {
                setCategories(await resp.json());
            }
        } catch (err) {
            console.error('Failed to fetch categories:', err);
        }
    };

    const fetchActiveLists = async () => {
        try {
            const resp = await fetch(`${API_BASE_URL}/api/lists`, {
                headers: { 'Authorization': `Bearer ${token}` }
            });
            if (resp.ok) {
                const allLists = await resp.json();
                setActiveLists(allLists.filter(l => l.status === 'preparing'));
            }
        } catch (err) {
            console.error('Failed to fetch active lists:', err);
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
        fetchActiveLists();
        fetchCategories();
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

    const handleAddItemToList = async (item, listId) => {
        setAddingTo(listId);
        try {
            const resp = await fetch(`${API_BASE_URL}/api/lists/${listId}/items`, {
                method: 'POST',
                headers: {
                    'Authorization': `Bearer ${token}`,
                    'Content-Type': 'application/json'
                },
                body: JSON.stringify({
                    name: item.name,
                    price: item.price,
                    description: `Deal from ${item.shop_name} (${item.quantity})`,
                    local_photo_path: item.local_photo_path,
                    quantity: parseFloat(addForm.quantity),
                    unit: addForm.unit,
                    category_id: addForm.category_id ? parseInt(addForm.category_id) : undefined,
                    flyer_item_id: item.id
                })
            });

            if (resp.ok) {
                setMessage({ type: 'success', text: `Added ${item.name} to list!` });
                setShowListSelector(null);
                setTimeout(() => setMessage(null), 3000);
            }
        } catch (err) {
            setMessage({ type: 'error', text: 'Failed to add item' });
        } finally {
            setAddingTo(null);
        }
    };

    const handleCreateAndAdd = async (item) => {
        const title = prompt('Enter new list title:', `${item.shop_name} Deals`);
        if (!title) return;

        try {
            const resp = await fetch(`${API_BASE_URL}/api/lists`, {
                method: 'POST',
                headers: {
                    'Authorization': `Bearer ${token}`,
                    'Content-Type': 'application/json'
                },
                body: JSON.stringify({ title })
            });

            if (resp.ok) {
                const newList = await resp.json();
                handleAddItemToList(item, newList.id);
                fetchActiveLists(); // Refresh active lists
            }
        } catch (err) {
            setMessage({ type: 'error', text: 'Failed to create list' });
        }
    };

    return (
        <div className="container" style={{ paddingBottom: '5rem' }}>
            {showListSelector && (
                <div
                    onClick={() => setShowListSelector(null)}
                    style={{
                        position: 'fixed',
                        top: 0,
                        left: 0,
                        right: 0,
                        bottom: 0,
                        zIndex: 15,
                        background: 'transparent'
                    }}
                />
            )}
            <header style={{ marginBottom: '2rem', paddingTop: '1rem', position: 'relative' }}>
                {message && (
                    <div className={`badge ${message.type === 'success' ? 'badge-success' : 'badge-error'}`} style={{
                        position: 'fixed',
                        top: '20px',
                        left: '50%',
                        transform: 'translateX(-50%)',
                        zIndex: 1000,
                        padding: '1rem 2rem',
                        boxShadow: 'var(--shadow-lg)',
                        display: 'flex',
                        alignItems: 'center',
                        gap: '0.5rem'
                    }}>
                        {message.type === 'success' ? <Check size={20} /> : <X size={20} />}
                        {message.text}
                    </div>
                )}
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
                        <div key={item.id} className="card" style={{ padding: 0, overflow: 'hidden', display: 'flex', flexDirection: 'column', position: 'relative' }}>
                            <div style={{ height: '200px', background: '#f8f9fa', position: 'relative', overflow: 'hidden' }}>
                                {item.local_photo_path ? (
                                    <img
                                        src={`${API_BASE_URL}${item.local_photo_path}`}
                                        alt={item.name}
                                        style={{ width: '100%', height: '100%', objectFit: 'contain', cursor: 'zoom-in' }}
                                        onClick={() => setPreviewImage({ src: `${API_BASE_URL}${item.local_photo_path}`, alt: item.name })}
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
                                        {item.categories.split(',').filter(Boolean).map(cat => (
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
                                                    color: 'var(--text-muted)'
                                                }}
                                            >
                                                {cat.trim()}
                                            </button>
                                        ))}
                                    </div>
                                )}

                                {item.keywords && (
                                    <div style={{ marginTop: '0.5rem', display: 'flex', gap: '0.25rem', flexWrap: 'wrap' }}>
                                        {item.keywords.split(',').filter(Boolean).map(kw => (
                                            <button
                                                key={kw}
                                                onClick={() => setFilters(prev => ({ ...prev, q: kw.trim() }))}
                                                style={{
                                                    fontSize: '0.65rem',
                                                    background: 'rgba(var(--primary-rgb), 0.05)',
                                                    padding: '1px 6px',
                                                    borderRadius: '4px',
                                                    border: '1px dashed var(--primary)',
                                                    cursor: 'pointer',
                                                    color: 'var(--primary)',
                                                    opacity: 0.8
                                                }}
                                            >
                                                #{kw.trim()}
                                            </button>
                                        ))}
                                    </div>
                                )}

                                <div style={{ marginTop: '1rem' }}>
                                    <button
                                        onClick={() => setShowListSelector(showListSelector === item.id ? null : item.id)}
                                        className="btn-primary"
                                        style={{ width: '100%', height: '40px', fontSize: '0.875rem' }}
                                    >
                                        <Plus size={16} />
                                        Add to List
                                    </button>

                                    {showListSelector === item.id && (
                                        <div style={{
                                            position: 'absolute',
                                            bottom: '10px',
                                            left: '10px',
                                            right: '10px',
                                            background: 'white',
                                            zIndex: 20,
                                            boxShadow: '0 -10px 25px rgba(0,0,0,0.15), var(--shadow-lg)',
                                            borderRadius: '12px',
                                            border: '1px solid var(--border)',
                                            overflow: 'hidden',
                                            display: 'flex',
                                            flexDirection: 'column'
                                        }}>
                                            <div style={{ padding: '0.75rem', background: 'var(--bg-main)', borderBottom: '1px solid var(--border)', display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                                                <span style={{ fontSize: '0.8rem', fontWeight: 800 }}>Add Settings</span>
                                                <button onClick={() => setShowListSelector(null)} style={{ background: 'none', border: 'none', cursor: 'pointer', color: 'var(--text-muted)' }}>
                                                    <X size={16} />
                                                </button>
                                            </div>

                                            <div style={{ padding: '1rem', display: 'flex', flexDirection: 'column', gap: '0.75rem', background: '#fff' }}>
                                                {/* Category Selection */}
                                                <div style={{ display: 'flex', flexDirection: 'column', gap: '0.25rem' }}>
                                                    <label style={{ fontSize: '0.7rem', fontWeight: 800, color: 'var(--text-muted)', textTransform: 'uppercase' }}>Category</label>
                                                    <div style={{ position: 'relative' }}>
                                                        <Tag size={14} style={{ position: 'absolute', left: '10px', top: '50%', transform: 'translateY(-50%)', color: 'var(--text-muted)' }} />
                                                        <select
                                                            value={addForm.category_id}
                                                            onChange={e => setAddForm(prev => ({ ...prev, category_id: e.target.value }))}
                                                            style={{ width: '100%', padding: '0.5rem 1rem 0.5rem 2rem', borderRadius: '8px', border: '1px solid var(--border)', fontSize: '0.875rem' }}
                                                        >
                                                            <option value="">No Category (Optional)</option>
                                                            {categories.map(c => <option key={c.id} value={c.id}>{c.name}</option>)}
                                                        </select>
                                                    </div>
                                                </div>

                                                {/* Qty & Measure */}
                                                <div style={{ display: 'flex', gap: '0.5rem' }}>
                                                    <div style={{ flex: 1, display: 'flex', flexDirection: 'column', gap: '0.25rem' }}>
                                                        <label style={{ fontSize: '0.7rem', fontWeight: 800, color: 'var(--text-muted)', textTransform: 'uppercase' }}>Qty</label>
                                                        <input
                                                            type="number"
                                                            step="0.1"
                                                            value={addForm.quantity}
                                                            onChange={e => setAddForm(prev => ({ ...prev, quantity: e.target.value }))}
                                                            style={{ width: '100%', padding: '0.5rem', borderRadius: '8px', border: '1px solid var(--border)', fontSize: '0.875rem' }}
                                                        />
                                                    </div>
                                                    <div style={{ flex: 1, display: 'flex', flexDirection: 'column', gap: '0.25rem' }}>
                                                        <label style={{ fontSize: '0.7rem', fontWeight: 800, color: 'var(--text-muted)', textTransform: 'uppercase' }}>Unit</label>
                                                        <select
                                                            value={addForm.unit}
                                                            onChange={e => setAddForm(prev => ({ ...prev, unit: e.target.value }))}
                                                            style={{ width: '100%', padding: '0.5rem', borderRadius: '8px', border: '1px solid var(--border)', fontSize: '0.875rem' }}
                                                        >
                                                            {['pcs', 'kg', 'g', '100g', 'l', 'pack'].map(u => <option key={u} value={u}>{u}</option>)}
                                                        </select>
                                                    </div>
                                                </div>
                                            </div>

                                            <div style={{ background: '#f8f9fa', borderTop: '1px solid var(--border)' }}>
                                                <div style={{ padding: '0.5rem', fontSize: '0.7rem', fontWeight: 800, color: 'var(--text-muted)', textTransform: 'uppercase', textAlign: 'center' }}>To List</div>
                                                <div style={{ maxHeight: '120px', overflowY: 'auto' }}>
                                                    {activeLists.map(list => (
                                                        <button
                                                            key={list.id}
                                                            onClick={() => handleAddItemToList(item, list.id)}
                                                            disabled={addingTo === list.id}
                                                            style={{
                                                                width: '100%',
                                                                padding: '0.75rem',
                                                                textAlign: 'left',
                                                                background: 'none',
                                                                border: 'none',
                                                                borderBottom: '1px solid var(--border)',
                                                                fontSize: '0.875rem',
                                                                display: 'flex',
                                                                justifyContent: 'space-between',
                                                                alignItems: 'center',
                                                                cursor: 'pointer'
                                                            }}
                                                        >
                                                            <span>{list.title}</span>
                                                            {addingTo === list.id ? <Loader2 className="spin" size={14} /> : <ShoppingCart size={14} style={{ opacity: 0.5 }} />}
                                                        </button>
                                                    ))}
                                                    <button
                                                        onClick={() => handleCreateAndAdd(item)}
                                                        style={{
                                                            width: '100%',
                                                            padding: '0.75rem',
                                                            textAlign: 'left',
                                                            background: 'var(--bg-main)',
                                                            border: 'none',
                                                            fontSize: '0.875rem',
                                                            color: 'var(--primary)',
                                                            fontWeight: 700,
                                                            display: 'flex',
                                                            alignItems: 'center',
                                                            gap: '0.5rem',
                                                            cursor: 'pointer'
                                                        }}
                                                    >
                                                        <Plus size={14} />
                                                        Create New List
                                                    </button>
                                                </div>
                                            </div>
                                        </div>
                                    )}
                                </div>
                            </div>
                        </div>
                    ))}
                </div>
            )}
            <ImageModal
                src={previewImage?.src}
                alt={previewImage?.alt}
                onClose={() => setPreviewImage(null)}
            />
        </div>
    );
};

export default FlyerItemsPage;
