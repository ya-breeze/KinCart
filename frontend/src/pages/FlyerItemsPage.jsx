import React, { useState, useEffect } from 'react';
import { useAuth } from '../context/AuthContext';
import { Search, Store, Calendar, ArrowLeft, Loader2, Filter, Plus, Check, X } from 'lucide-react';
import { useNavigate } from 'react-router-dom';
import { API_BASE_URL } from '../config';
import ImageModal from '../components/ImageModal';
import { usePaginatedFlyerItems } from '../hooks/usePaginatedFlyerItems';
import InfiniteScrollTrigger from '../components/InfiniteScrollTrigger';
import FlyerItemCard from '../components/FlyerItemCard';


const FlyerItemsPage = () => {
    const { token, currency } = useAuth();
    const navigate = useNavigate();
    const [shops, setShops] = useState([]);
    const [categories, setCategories] = useState([]);
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

    // Use pagination hook for items
    const {
        items,
        loading,
        loadingMore,
        hasMore,
        totalCount,
        loadMore,
        clearCache
    } = usePaginatedFlyerItems(token, filters);

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

    useEffect(() => {
        fetchShops();
        fetchActiveLists();
        fetchCategories();
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, []);

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
                clearCache(); // Clear cache when items might change
                setTimeout(() => setMessage(null), 3000);
            }
        } catch {
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
        } catch {
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
                                style={{ paddingLeft: '2.5rem', paddingRight: '2.5rem', width: '100%' }}
                            />
                            {filters.q && (
                                <button
                                    onClick={() => setFilters(prev => ({ ...prev, q: '' }))}
                                    style={{
                                        position: 'absolute',
                                        right: '10px',
                                        top: '50%',
                                        transform: 'translateY(-50%)',
                                        background: 'none',
                                        border: 'none',
                                        cursor: 'pointer',
                                        color: 'var(--text-muted)',
                                        display: 'flex',
                                        alignItems: 'center',
                                        justifyContent: 'center',
                                        padding: '4px',
                                        borderRadius: '50%',
                                        transition: 'all 0.2s'
                                    }}
                                    onMouseEnter={e => e.currentTarget.style.background = 'var(--bg-main)'}
                                    onMouseLeave={e => e.currentTarget.style.background = 'none'}
                                >
                                    <X size={16} />
                                </button>
                            )}
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
                <>
                    {/* Item count */}
                    <div style={{
                        marginBottom: '1rem',
                        color: 'var(--text-muted)',
                        fontSize: '0.875rem',
                        textAlign: 'right'
                    }}>
                        Showing {items.length} of {totalCount} items
                    </div>

                    <div style={{
                        display: 'grid',
                        gridTemplateColumns: 'repeat(auto-fill, minmax(280px, 1fr))',
                        gap: '1.5rem'
                    }}>
                    {items.map(item => (
                        <FlyerItemCard
                            key={item.id}
                            item={item}
                            currency={currency}
                            showListSelector={showListSelector}
                            onToggleListSelector={(id) => setShowListSelector(showListSelector === id ? null : id)}
                            onImagePreview={setPreviewImage}
                            onCategorySearch={(query) => setFilters(prev => ({ ...prev, q: query }))}
                            addForm={addForm}
                            onAddFormChange={setAddForm}
                            categories={categories}
                            activeLists={activeLists}
                            addingTo={addingTo}
                            onAddToList={handleAddItemToList}
                            onCreateAndAdd={handleCreateAndAdd}
                        />
                    ))}
                    </div>

                    {/* Infinite scroll trigger */}
                    <InfiniteScrollTrigger
                        onIntersect={loadMore}
                        hasMore={hasMore}
                        loading={loadingMore}
                    />

                    {/* Loading more indicator */}
                    {loadingMore && (
                        <div style={{
                            textAlign: 'center',
                            padding: '2rem',
                            color: 'var(--text-muted)'
                        }}>
                            <Loader2 className="spin" size={32} style={{ color: 'var(--primary)', opacity: 0.5 }} />
                            <p style={{ marginTop: '0.5rem', fontSize: '0.875rem' }}>
                                Loading more items...
                            </p>
                        </div>
                    )}

                    {/* End of results indicator */}
                    {!hasMore && items.length > 0 && (
                        <div style={{
                            textAlign: 'center',
                            padding: '2rem',
                            color: 'var(--text-muted)',
                            fontSize: '0.875rem',
                            borderTop: '1px solid var(--border)',
                            marginTop: '2rem'
                        }}>
                            You've reached the end of the list
                        </div>
                    )}
                </>
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
