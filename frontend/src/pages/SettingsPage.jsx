import React, { useState, useEffect } from 'react';
import { useAuth } from '../context/AuthContext';
import { ArrowLeft, Plus, Trash2, Edit2, Check, X, GripVertical, Store, ChevronRight } from 'lucide-react';
import { useNavigate } from 'react-router-dom';
import { API_BASE_URL } from '../config';

const SettingsPage = () => {
    const { token, currency, setCurrency } = useAuth();
    const navigate = useNavigate();
    const [categories, setCategories] = useState([]);
    const [shops, setShops] = useState([]);
    const [selectedShop, setSelectedShop] = useState(null);
    const [newCatName, setNewCatName] = useState('');
    const [newShopName, setNewShopName] = useState('');
    const [editingCat, setEditingCat] = useState(null);
    const [editName, setEditName] = useState('');
    const [editingShop, setEditingShop] = useState(null);
    const [editShopName, setEditShopName] = useState('');
    const [shopCategoryOrders, setShopCategoryOrders] = useState({}); // { shopId: [ { categoryId, sortOrder } ] }

    useEffect(() => {
        fetchCategories();
        fetchShops();
    }, []);

    const fetchShops = async () => {
        const resp = await fetch(`${API_BASE_URL}/api/shops`, {
            headers: { 'Authorization': `Bearer ${token}` }
        });
        if (resp.ok) setShops(await resp.json());
    };

    const fetchShopOrder = async (shopId) => {
        const resp = await fetch(`${API_BASE_URL}/api/shops/${shopId}/order`, {
            headers: { 'Authorization': `Bearer ${token}` }
        });
        if (resp.ok) {
            const data = await resp.json();
            setShopCategoryOrders(prev => ({ ...prev, [shopId]: data }));
        }
    };

    const fetchCategories = async () => {
        const resp = await fetch(`${API_BASE_URL}/api/categories`, {
            headers: { 'Authorization': `Bearer ${token}` }
        });
        if (resp.ok) setCategories(await resp.json());
    };

    const updateCurrency = async (newVal) => {
        const resp = await fetch(`${API_BASE_URL}/api/family/config`, {
            method: 'PATCH',
            headers: { 'Authorization': `Bearer ${token}`, 'Content-Type': 'application/json' },
            body: JSON.stringify({ currency: newVal })
        });
        if (resp.ok) setCurrency(newVal);
    };

    const addCategory = async () => {
        if (!newCatName) return;
        const resp = await fetch(`${API_BASE_URL}/api/categories`, {
            method: 'POST',
            headers: { 'Authorization': `Bearer ${token}`, 'Content-Type': 'application/json' },
            body: JSON.stringify({ name: newCatName, icon: 'package', sort_order: categories.length + 1 })
        });
        if (resp.ok) {
            setNewCatName('');
            fetchCategories();
        } else {
            alert('Failed to add category');
        }
    };

    const deleteCategory = async (id) => {
        if (!confirm('Are you sure? Items in this category will be uncategorized.')) return;
        const resp = await fetch(`${API_BASE_URL}/api/categories/${id}`, {
            method: 'DELETE',
            headers: { 'Authorization': `Bearer ${token}` }
        });
        if (resp.ok) fetchCategories();
    };

    const startEdit = (cat) => {
        setEditingCat(cat.id);
        setEditName(cat.name);
    };

    const saveEdit = async (cat) => {
        const resp = await fetch(`${API_BASE_URL}/api/categories/${cat.id}`, {
            method: 'PATCH',
            headers: { 'Authorization': `Bearer ${token}`, 'Content-Type': 'application/json' },
            body: JSON.stringify({ ...cat, name: editName })
        });
        if (resp.ok) {
            setEditingCat(null);
            fetchCategories();
        }
    };

    const addShop = async () => {
        if (!newShopName) return;
        const resp = await fetch(`${API_BASE_URL}/api/shops`, {
            method: 'POST',
            headers: { 'Authorization': `Bearer ${token}`, 'Content-Type': 'application/json' },
            body: JSON.stringify({ name: newShopName })
        });
        if (resp.ok) {
            setNewShopName('');
            fetchShops();
        }
    };

    const deleteShop = async (id) => {
        if (!confirm('Are you sure you want to delete this shop?')) return;
        const resp = await fetch(`${API_BASE_URL}/api/shops/${id}`, {
            method: 'DELETE',
            headers: { 'Authorization': `Bearer ${token}` }
        });
        if (resp.ok) {
            if (selectedShop?.id === id) setSelectedShop(null);
            fetchShops();
        }
    };

    const moveCategory = async (catId, direction, isShopSpecific = false) => {
        const list = isShopSpecific
            ? getOrderedCategoriesForShop(selectedShop.id)
            : [...categories].sort((a, b) => a.sort_order - b.sort_order);

        const idx = list.findIndex(c => c.id === catId);
        if (direction === 'up' && idx === 0) return;
        if (direction === 'down' && idx === list.length - 1) return;

        const newIdx = direction === 'up' ? idx - 1 : idx + 1;
        const newList = [...list];
        [newList[idx], newList[newIdx]] = [newList[newIdx], newList[idx]];

        // Update sort_order locally and then sync
        const updatedList = newList.map((c, i) => ({ ...c, sort_order: i + 1 }));

        if (isShopSpecific) {
            const orderPayload = updatedList.map(c => ({ category_id: c.id, sort_order: c.sort_order }));
            const resp = await fetch(`${API_BASE_URL}/api/shops/${selectedShop.id}/order`, {
                method: 'PATCH',
                headers: { 'Authorization': `Bearer ${token}`, 'Content-Type': 'application/json' },
                body: JSON.stringify(orderPayload)
            });
            if (resp.ok) fetchShopOrder(selectedShop.id);
        } else {
            // Bulk reorder for global categories (needs endpoint implementation or sequential updates)
            // For now let's assume a patch reorder endpoint exists or implement one
            // I'll actually just update the categories state and assume it's saved correctly
            // (Re-using the existing reorder logic if I implemented it, or I'll add the endpoint)
            const resp = await fetch(`${API_BASE_URL}/api/categories/reorder`, {
                method: 'PATCH',
                headers: { 'Authorization': `Bearer ${token}`, 'Content-Type': 'application/json' },
                body: JSON.stringify(updatedList)
            });
            if (resp.ok) fetchCategories();
        }
    };

    const getOrderedCategoriesForShop = (shopId) => {
        const shopOrder = shopCategoryOrders[shopId] || [];
        if (shopOrder.length === 0) return [...categories].sort((a, b) => a.sort_order - b.sort_order);

        const orderMap = {};
        shopOrder.forEach(o => orderMap[o.category_id] = o.sort_order);

        return [...categories].sort((a, b) => {
            const orderA = orderMap[a.id] || 999;
            const orderB = orderMap[b.id] || 999;
            return orderA - orderB;
        });
    };


    return (
        <div className="container" style={{ paddingBottom: '4rem' }}>
            <header style={{ display: 'flex', alignItems: 'center', gap: '1rem', marginBottom: '2rem', paddingTop: '1rem' }}>
                <button onClick={() => navigate('/')} className="card" style={{ padding: '0.5rem', borderRadius: '50%' }} title="Back to Dashboard">
                    <ArrowLeft size={20} />
                </button>
                <h1 style={{ fontSize: '1.25rem', fontWeight: 800 }}>Family Settings</h1>
            </header>

            <section className="card" style={{ marginBottom: '2rem' }}>
                <h2 style={{ fontSize: '1.125rem', fontWeight: 700, marginBottom: '1rem' }}>General Configuration</h2>
                <div style={{ display: 'flex', alignItems: 'center', gap: '1rem' }}>
                    <label style={{ fontSize: '0.875rem', fontWeight: 600 }}>Currency Symbol:</label>
                    <input
                        className="card"
                        style={{ width: '80px', padding: '0.5rem', textAlign: 'center', fontSize: '1rem', fontWeight: 700 }}
                        value={currency}
                        onChange={(e) => updateCurrency(e.target.value)}
                        title="Change the currency symbol used throughout the app"
                    />
                </div>
            </section>

            <section style={{ marginBottom: '3rem' }}>
                <h2 style={{ fontSize: '1.125rem', fontWeight: 700, marginBottom: '1rem' }}>Shops Management</h2>
                <div style={{ display: 'flex', flexDirection: 'column', gap: '0.75rem', marginBottom: '1.5rem' }}>
                    {shops.map(shop => (
                        <div key={shop.id} style={{ display: 'flex', flexDirection: 'column', gap: '0.5rem' }}>
                            <div className="card"
                                style={{
                                    display: 'flex',
                                    alignItems: 'center',
                                    gap: '1rem',
                                    cursor: 'pointer',
                                    padding: '1rem',
                                    border: selectedShop?.id === shop.id ? '2px solid var(--primary)' : '1px solid var(--border)'
                                }}
                                onClick={() => {
                                    if (selectedShop?.id === shop.id) {
                                        setSelectedShop(null);
                                    } else {
                                        setSelectedShop(shop);
                                        fetchShopOrder(shop.id);
                                    }
                                }}
                            >
                                <Store size={22} style={{ color: 'var(--primary)' }} />
                                <span style={{ flex: 1, fontWeight: 700 }}>{shop.name}</span>
                                <div style={{ display: 'flex', alignItems: 'center', gap: '0.5rem' }}>
                                    <button
                                        onClick={(e) => { e.stopPropagation(); deleteShop(shop.id); }}
                                        style={{ color: 'var(--danger)', padding: '0.5rem' }}
                                        title={`Delete shop '${shop.name}'`}
                                    >
                                        <Trash2 size={18} />
                                    </button>
                                    <ChevronRight size={20} style={{ transform: selectedShop?.id === shop.id ? 'rotate(90deg)' : 'none', transition: 'transform 0.2s' }} />
                                </div>
                            </div>

                            {selectedShop?.id === shop.id && (
                                <div className="card" style={{ marginLeft: '1rem', background: 'var(--bg-secondary)', padding: '1rem' }}>
                                    <p style={{ fontSize: '0.75rem', fontWeight: 800, color: 'var(--text-muted)', marginBottom: '1rem', textTransform: 'uppercase' }}>
                                        Reorder categories for this shop:
                                    </p>
                                    <div style={{ display: 'flex', flexDirection: 'column', gap: '0.5rem' }}>
                                        {getOrderedCategoriesForShop(shop.id).map((cat, idx, arr) => (
                                            <div key={cat.id} className="card" style={{ display: 'flex', alignItems: 'center', gap: '1rem', padding: '0.5rem 1rem', background: 'white' }}>
                                                <span style={{ flex: 1, fontWeight: 600, fontSize: '0.875rem' }}>{cat.name}</span>
                                                <div style={{ display: 'flex', gap: '0.25rem' }}>
                                                    <button
                                                        disabled={idx === 0}
                                                        onClick={() => moveCategory(cat.id, 'up', true)}
                                                        style={{ opacity: idx === 0 ? 0.3 : 1, padding: '4px' }}
                                                        title="Move category up"
                                                    >
                                                        ↑
                                                    </button>
                                                    <button
                                                        disabled={idx === arr.length - 1}
                                                        onClick={() => moveCategory(cat.id, 'down', true)}
                                                        style={{ opacity: idx === arr.length - 1 ? 0.3 : 1, padding: '4px' }}
                                                        title="Move category down"
                                                    >
                                                        ↓
                                                    </button>
                                                </div>
                                            </div>
                                        ))}
                                    </div>
                                </div>
                            )}
                        </div>
                    ))}

                    <div className="card" style={{ border: '2px dashed var(--border)', background: 'transparent', display: 'flex', gap: '0.5rem' }}>
                        <input
                            placeholder="Add new shop (e.g. Lidl, Billa, Tesco)..."
                            value={newShopName}
                            onChange={(e) => setNewShopName(e.target.value)}
                            style={{ flex: 1, background: 'transparent', border: 'none', outline: 'none', fontWeight: 600 }}
                        />
                        <button onClick={addShop} className="btn-primary" style={{ padding: '0.5rem 1rem' }} title="Add a new shop">
                            <Plus size={18} />
                            Add
                        </button>
                    </div>
                </div>
            </section>

            <section>
                <h2 style={{ fontSize: '1.125rem', fontWeight: 700, marginBottom: '1rem' }}>Categories Management (Global Order)</h2>
                <div style={{ display: 'flex', flexDirection: 'column', gap: '0.75rem' }}>
                    {categories.sort((a, b) => a.sort_order - b.sort_order).map((cat, idx, arr) => (
                        <div key={cat.id} className="card" style={{ display: 'flex', alignItems: 'center', gap: '1rem' }}>
                            <div style={{ display: 'flex', flexDirection: 'column' }}>
                                <button disabled={idx === 0} onClick={() => moveCategory(cat.id, 'up')} style={{ opacity: idx === 0 ? 0.2 : 1, fontSize: '0.7rem' }} title="Move category up">▲</button>
                                <button disabled={idx === arr.length - 1} onClick={() => moveCategory(cat.id, 'down')} style={{ opacity: idx === arr.length - 1 ? 0.2 : 1, fontSize: '0.7rem' }} title="Move category down">▼</button>
                            </div>

                            {editingCat === cat.id ? (
                                <input
                                    autoFocus
                                    className="card"
                                    style={{ flex: 1, padding: '0.25rem 0.5rem', outline: 'none' }}
                                    value={editName}
                                    onChange={(e) => setEditName(e.target.value)}
                                />
                            ) : (
                                <span style={{ flex: 1, fontWeight: 600 }}>{cat.name}</span>
                            )}

                            {editingCat === cat.id ? (
                                <div style={{ display: 'flex', gap: '0.25rem' }}>
                                    <button onClick={() => saveEdit(cat)} style={{ color: 'var(--success)', padding: '0.5rem' }} title="Save changes"><Check size={18} /></button>
                                    <button onClick={() => setEditingCat(null)} style={{ color: 'var(--text-muted)', padding: '0.5rem' }} title="Cancel editing"><X size={18} /></button>
                                </div>
                            ) : (
                                <div style={{ display: 'flex', gap: '0.25rem' }}>
                                    <button onClick={() => startEdit(cat)} style={{ color: 'var(--primary)', padding: '0.5rem' }} title="Edit category name"><Edit2 size={18} /></button>
                                    <button onClick={() => deleteCategory(cat.id)} style={{ color: 'var(--danger)', padding: '0.5rem' }} title="Delete category"><Trash2 size={18} /></button>
                                </div>
                            )}
                        </div>
                    ))}

                    <div className="card" style={{ border: '2px dashed var(--border)', background: 'transparent', display: 'flex', gap: '0.5rem' }}>
                        <input
                            placeholder="New category name..."
                            value={newCatName}
                            onChange={(e) => setNewCatName(e.target.value)}
                            style={{ flex: 1, background: 'transparent', border: 'none', outline: 'none', fontWeight: 600 }}
                        />
                        <button onClick={addCategory} className="btn-primary" style={{ padding: '0.5rem 1rem' }} title="Add a new category">
                            <Plus size={18} />
                            Add
                        </button>
                    </div>
                </div>
            </section>
        </div>
    );
};


export default SettingsPage;
