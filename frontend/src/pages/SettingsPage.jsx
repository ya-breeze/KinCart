import React, { useState, useEffect, useRef } from 'react';
import { useAuth } from '../context/AuthContext';
import { useToast, getApiError } from '../context/ToastContext';
import { ArrowLeft, Plus, Trash2, Edit2, Check, X, GripVertical, Store, ChevronRight, BarChart3 } from 'lucide-react';
import { useNavigate } from 'react-router-dom';
import { API_BASE_URL } from '../config';
import Modal from '../components/Modal';

const SettingsPage = () => {
    const { currency, setCurrency } = useAuth();
    const { showToast } = useToast();
    const navigate = useNavigate();
    const [categories, setCategories] = useState([]);
    const [shops, setShops] = useState([]);
    const [selectedShop, setSelectedShop] = useState(null);
    const [newCatName, setNewCatName] = useState('');
    const [newShopName, setNewShopName] = useState('');
    const [editingCat, setEditingCat] = useState(null);
    const [editName, setEditName] = useState('');
    const [shopCategoryOrders, setShopCategoryOrders] = useState({}); // { shopId: [ { categoryId, sortOrder } ] }
    const [deleteConfirm, setDeleteConfirm] = useState(null); // { type: 'category'|'shop', id, name }
    const [aliases, setAliases] = useState([]);
    const [aliasSearch, setAliasSearch] = useState('');
    const [newAlias, setNewAlias] = useState({ planned_name: '', receipt_name: '', shop_id: '', last_price: '' });
    const [expandedGroups, setExpandedGroups] = useState(new Set());
    const [editingAlias, setEditingAlias] = useState(null); // { id, receipt_name, shop_id, last_price }
    const newAliasReceiptRef = useRef(null);

    const fetchShops = async () => {
        const resp = await fetch(`${API_BASE_URL}/api/shops`, {
        });
        if (resp.ok) setShops(await resp.json());
    };

    const fetchCategories = async () => {
        const resp = await fetch(`${API_BASE_URL}/api/categories`, {
        });
        if (resp.ok) setCategories(await resp.json());
    };

    const fetchAliases = async (q = '') => {
        const url = q.length >= 2
            ? `${API_BASE_URL}/api/family/aliases?q=${encodeURIComponent(q)}`
            : `${API_BASE_URL}/api/family/aliases`;
        const resp = await fetch(url);
        if (resp.ok) setAliases(await resp.json());
    };

    const createAlias = async () => {
        if (!newAlias.planned_name || !newAlias.receipt_name) return;
        const body = {
            planned_name: newAlias.planned_name,
            receipt_name: newAlias.receipt_name,
            shop_id: newAlias.shop_id || null,
            last_price: parseFloat(newAlias.last_price) || 0
        };
        try {
            const resp = await fetch(`${API_BASE_URL}/api/family/aliases`, {
                method: 'POST',
                body: JSON.stringify(body)
            });
            if (resp.ok) {
                setNewAlias({ planned_name: '', receipt_name: '', shop_id: '', last_price: '' });
                fetchAliases(aliasSearch);
            } else {
                showToast(await getApiError(resp, 'Failed to create alias'));
            }
        } catch {
            showToast('Network error — could not create alias');
        }
    };

    const deleteAlias = async (aliasId) => {
        try {
            const resp = await fetch(`${API_BASE_URL}/api/family/aliases/${aliasId}`, { method: 'DELETE' });
            if (resp.ok) fetchAliases(aliasSearch);
            else showToast(await getApiError(resp, 'Failed to delete alias'));
        } catch {
            showToast('Network error — could not delete alias');
        }
    };

    const toggleGroup = (name) => setExpandedGroups(prev => {
        const next = new Set(prev);
        next.has(name) ? next.delete(name) : next.add(name);
        return next;
    });

    const startEditAlias = (alias) => setEditingAlias({
        id: alias.id,
        receipt_name: alias.receipt_name,
        shop_id: alias.shop_id || '',
        last_price: alias.last_price || ''
    });

    const saveEditAlias = async () => {
        try {
            const resp = await fetch(`${API_BASE_URL}/api/family/aliases/${editingAlias.id}`, {
                method: 'PATCH',
                body: JSON.stringify({
                    receipt_name: editingAlias.receipt_name,
                    shop_id: editingAlias.shop_id || null,
                    last_price: parseFloat(editingAlias.last_price) || 0
                })
            });
            if (resp.ok) { setEditingAlias(null); fetchAliases(aliasSearch); }
            else showToast(await getApiError(resp, 'Failed to save alias'));
        } catch {
            showToast('Network error — could not save alias');
        }
    };

    const prefillGroup = (groupName) => {
        setNewAlias(prev => ({ ...prev, planned_name: groupName, receipt_name: '' }));
        setTimeout(() => newAliasReceiptRef.current?.focus(), 50);
    };

    useEffect(() => {
        fetchCategories();
        fetchShops();
        fetchAliases();
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, []);

    const fetchShopOrder = async (shopId) => {
        const resp = await fetch(`${API_BASE_URL}/api/shops/${shopId}/order`, {
        });
        if (resp.ok) {
            const data = await resp.json();
            setShopCategoryOrders(prev => ({ ...prev, [shopId]: data }));
        }
    };

    const updateCurrency = async (newVal) => {
        try {
            const resp = await fetch(`${API_BASE_URL}/api/family/config`, {
                method: 'PATCH',
                body: JSON.stringify({ currency: newVal })
            });
            if (resp.ok) setCurrency(newVal);
            else showToast(await getApiError(resp, 'Failed to update currency'));
        } catch {
            showToast('Network error — could not update currency');
        }
    };

    const addCategory = async () => {
        if (!newCatName) return;
        try {
            const resp = await fetch(`${API_BASE_URL}/api/categories`, {
                method: 'POST',
                body: JSON.stringify({ name: newCatName, icon: 'package', sort_order: categories.length + 1 })
            });
            if (resp.ok) {
                setNewCatName('');
                fetchCategories();
            } else {
                showToast(await getApiError(resp, 'Failed to add category'));
            }
        } catch {
            showToast('Network error — could not add category');
        }
    };

    const deleteCategory = async (id) => {
        const cat = categories.find(c => c.id === id);
        setDeleteConfirm({ type: 'category', id, name: cat?.name });
    };

    const confirmDelete = async () => {
        if (!deleteConfirm) return;
        const endpoint = deleteConfirm.type === 'category' ? `categories/${deleteConfirm.id}` : `shops/${deleteConfirm.id}`;
        try {
            const resp = await fetch(`${API_BASE_URL}/api/${endpoint}`, { method: 'DELETE' });
            if (resp.ok) {
                if (deleteConfirm.type === 'category') {
                    fetchCategories();
                } else {
                    if (selectedShop?.id === deleteConfirm.id) setSelectedShop(null);
                    fetchShops();
                }
            } else {
                showToast(await getApiError(resp, `Failed to delete ${deleteConfirm.type}`));
            }
        } catch {
            showToast(`Network error — could not delete ${deleteConfirm.type}`);
        }
        setDeleteConfirm(null);
    };

    const startEdit = (cat) => {
        setEditingCat(cat.id);
        setEditName(cat.name);
    };

    const saveEdit = async (cat) => {
        try {
            const resp = await fetch(`${API_BASE_URL}/api/categories/${cat.id}`, {
                method: 'PATCH',
                body: JSON.stringify({ ...cat, name: editName })
            });
            if (resp.ok) {
                setEditingCat(null);
                fetchCategories();
            } else {
                showToast(await getApiError(resp, 'Failed to rename category'));
            }
        } catch {
            showToast('Network error — could not rename category');
        }
    };

    const addShop = async () => {
        if (!newShopName) return;
        try {
            const resp = await fetch(`${API_BASE_URL}/api/shops`, {
                method: 'POST',
                body: JSON.stringify({ name: newShopName })
            });
            if (resp.ok) {
                setNewShopName('');
                fetchShops();
            } else {
                showToast(await getApiError(resp, 'Failed to add shop'));
            }
        } catch {
            showToast('Network error — could not add shop');
        }
    };

    const deleteShop = async (id) => {
        const shop = shops.find(s => s.id === id);
        setDeleteConfirm({ type: 'shop', id, name: shop?.name });
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
            try {
                const resp = await fetch(`${API_BASE_URL}/api/shops/${selectedShop.id}/order`, {
                    method: 'PATCH',
                    body: JSON.stringify(orderPayload)
                });
                if (resp.ok) fetchShopOrder(selectedShop.id);
                else showToast(await getApiError(resp, 'Failed to save shop category order'));
            } catch {
                showToast('Network error — could not save order');
            }
        } else {
            try {
                const resp = await fetch(`${API_BASE_URL}/api/categories/reorder`, {
                    method: 'PATCH',
                    body: JSON.stringify(updatedList)
                });
                if (resp.ok) fetchCategories();
                else showToast(await getApiError(resp, 'Failed to reorder categories'));
            } catch {
                showToast('Network error — could not reorder categories');
            }
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
            <header style={{
                display: 'flex',
                alignItems: 'center',
                gap: '1rem',
                marginBottom: '2rem',
                paddingTop: '1rem',
                flexWrap: 'wrap'
            }}>
                <button onClick={() => navigate('/')} className="card" style={{ padding: '0.5rem', borderRadius: '50%', flexShrink: 0 }} title="Back to Dashboard">
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

                    <div className="card" style={{ border: '2px dashed var(--border)', background: 'transparent', display: 'flex', gap: '0.5rem', flexWrap: 'wrap' }}>
                        <input
                            placeholder="Add new shop..."
                            value={newShopName}
                            onChange={(e) => setNewShopName(e.target.value)}
                            style={{ flex: '1 1 200px', background: 'transparent', border: 'none', outline: 'none', fontWeight: 600, padding: '0.5rem' }}
                        />
                        <button onClick={addShop} className="btn-primary" style={{ padding: '0.5rem 1rem', flex: '1 1 auto' }} title="Add a new shop">
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

                    <div className="card" style={{ border: '2px dashed var(--border)', background: 'transparent', display: 'flex', gap: '0.5rem', flexWrap: 'wrap' }}>
                        <input
                            placeholder="New category..."
                            value={newCatName}
                            onChange={(e) => setNewCatName(e.target.value)}
                            style={{ flex: '1 1 150px', background: 'transparent', border: 'none', outline: 'none', fontWeight: 600, padding: '0.5rem' }}
                        />
                        <button onClick={addCategory} className="btn-primary" style={{ padding: '0.5rem 1rem', flex: '1 1 auto' }} title="Add a new category">
                            <Plus size={18} />
                            Add
                        </button>
                    </div>

                </div>
            </section>

            <section style={{ marginTop: '3rem', marginBottom: '3rem' }}>
                <h2 style={{ fontSize: '1.125rem', fontWeight: 700, marginBottom: '0.5rem' }}>Item Aliases</h2>
                <p style={{ fontSize: '0.875rem', color: 'var(--text-muted)', marginBottom: '1rem' }}>
                    Map generic list names to specific products. Aliases are also learned automatically from confirmed receipts.
                </p>

                <input
                    placeholder="Filter by item name..."
                    value={aliasSearch}
                    onChange={e => { setAliasSearch(e.target.value); fetchAliases(e.target.value); }}
                    style={{ width: '100%', padding: '0.5rem 0.75rem', borderRadius: '8px', border: '1px solid var(--border)', outline: 'none', fontWeight: 600, marginBottom: '1rem', boxSizing: 'border-box' }}
                />

                {(() => {
                    const aliasGroups = aliases.reduce((acc, a) => {
                        if (!acc[a.planned_name]) acc[a.planned_name] = [];
                        acc[a.planned_name].push(a);
                        return acc;
                    }, {});
                    const groupNames = Object.keys(aliasGroups);

                    if (groupNames.length === 0) {
                        return (
                            <p style={{ color: 'var(--text-muted)', fontSize: '0.875rem', textAlign: 'center', padding: '1rem', marginBottom: '1.5rem' }}>
                                No aliases yet. They are created automatically when you confirm receipts, or add one below.
                            </p>
                        );
                    }

                    return (
                        <div style={{ display: 'flex', flexDirection: 'column', gap: '0.375rem', marginBottom: '1.5rem' }}>
                            {groupNames.map(groupName => {
                                const variants = aliasGroups[groupName];
                                const isOpen = expandedGroups.has(groupName);
                                return (
                                    <div key={groupName}>
                                        {/* Group header */}
                                        <div
                                            onClick={() => toggleGroup(groupName)}
                                            style={{ display: 'flex', alignItems: 'center', gap: '0.75rem', padding: '0.75rem 1rem', borderRadius: '12px', background: 'var(--bg-secondary)', cursor: 'pointer', userSelect: 'none' }}
                                        >
                                            <ChevronRight size={16} style={{ color: 'var(--text-muted)', flexShrink: 0, transition: 'transform 0.15s', transform: isOpen ? 'rotate(90deg)' : 'none' }} />
                                            <span style={{ fontWeight: 800, fontSize: '0.95rem', flex: 1 }}>{groupName}</span>
                                            <span style={{ fontSize: '0.7rem', fontWeight: 700, background: 'var(--border)', color: 'var(--text-muted)', padding: '0.15rem 0.5rem', borderRadius: '999px' }}>
                                                {variants.length}
                                            </span>
                                            <button
                                                onClick={e => { e.stopPropagation(); prefillGroup(groupName); }}
                                                style={{ display: 'flex', alignItems: 'center', gap: '0.2rem', fontSize: '0.75rem', fontWeight: 700, color: 'var(--primary)', background: 'none', border: 'none', cursor: 'pointer', padding: '0.25rem 0.5rem', borderRadius: '6px' }}
                                                title={`Add variant for "${groupName}"`}
                                            >
                                                <Plus size={13} /> add
                                            </button>
                                        </div>

                                        {/* Variant rows */}
                                        {isOpen && (
                                            <div style={{ display: 'flex', flexDirection: 'column' }}>
                                                {variants.map(alias => (
                                                    <div key={alias.id} style={{ display: 'flex', alignItems: 'center', gap: '0.75rem', padding: '0.5rem 1rem 0.5rem 2.5rem', borderBottom: '1px solid var(--border)' }}>
                                                        {editingAlias?.id === alias.id ? (
                                                            <>
                                                                <input
                                                                    autoFocus
                                                                    value={editingAlias.receipt_name}
                                                                    onChange={e => setEditingAlias({ ...editingAlias, receipt_name: e.target.value })}
                                                                    style={{ flex: '2 1 160px', padding: '0.25rem 0.5rem', borderRadius: '8px', border: '1px solid var(--border)', outline: 'none', fontWeight: 600, fontSize: '0.875rem' }}
                                                                />
                                                                <select
                                                                    value={editingAlias.shop_id}
                                                                    onChange={e => setEditingAlias({ ...editingAlias, shop_id: e.target.value })}
                                                                    style={{ flex: '1 1 100px', padding: '0.25rem 0.5rem', borderRadius: '8px', border: '1px solid var(--border)', outline: 'none', background: 'white', fontSize: '0.875rem' }}
                                                                >
                                                                    <option value="">Any shop</option>
                                                                    {shops.map(s => <option key={s.id} value={s.id}>{s.name}</option>)}
                                                                </select>
                                                                <input
                                                                    type="number"
                                                                    value={editingAlias.last_price}
                                                                    onChange={e => setEditingAlias({ ...editingAlias, last_price: e.target.value })}
                                                                    style={{ flex: '0 1 70px', padding: '0.25rem 0.5rem', borderRadius: '8px', border: '1px solid var(--border)', outline: 'none', fontSize: '0.875rem' }}
                                                                    placeholder="Price"
                                                                />
                                                                <button onClick={saveEditAlias} style={{ color: 'var(--success)', padding: '0.25rem', background: 'none', border: 'none', cursor: 'pointer' }} title="Save"><Check size={16} /></button>
                                                                <button onClick={() => setEditingAlias(null)} style={{ color: 'var(--text-muted)', padding: '0.25rem', background: 'none', border: 'none', cursor: 'pointer' }} title="Cancel"><X size={16} /></button>
                                                            </>
                                                        ) : (
                                                            <>
                                                                <span style={{ flex: 1, fontWeight: 600, fontSize: '0.9rem' }}>{alias.receipt_name}</span>
                                                                {alias.shop && <span style={{ fontSize: '0.75rem', color: 'var(--text-muted)' }}>{alias.shop.name}</span>}
                                                                {alias.last_price > 0 && <span style={{ fontSize: '0.75rem', color: 'var(--success)', fontWeight: 600 }}>{alias.last_price.toFixed(2)}</span>}
                                                                <span style={{ fontSize: '0.7rem', color: 'var(--text-muted)' }}>×{alias.purchase_count}</span>
                                                                <button onClick={() => startEditAlias(alias)} style={{ color: 'var(--primary)', padding: '0.25rem', background: 'none', border: 'none', cursor: 'pointer' }} title="Edit"><Edit2 size={14} /></button>
                                                                <button onClick={() => deleteAlias(alias.id)} style={{ color: 'var(--danger)', padding: '0.25rem', background: 'none', border: 'none', cursor: 'pointer' }} title="Delete"><Trash2 size={14} /></button>
                                                            </>
                                                        )}
                                                    </div>
                                                ))}
                                            </div>
                                        )}
                                    </div>
                                );
                            })}
                        </div>
                    );
                })()}

                <div className="card" style={{ border: '2px dashed var(--border)', background: 'transparent', padding: '1rem', display: 'flex', flexDirection: 'column', gap: '0.75rem' }}>
                    <p style={{ fontSize: '0.75rem', fontWeight: 800, color: 'var(--text-muted)', textTransform: 'uppercase', margin: 0 }}>Add Alias</p>
                    <div style={{ display: 'flex', gap: '0.5rem', flexWrap: 'wrap' }}>
                        <input
                            placeholder="Generic name (e.g. jogurt)"
                            value={newAlias.planned_name}
                            onChange={e => setNewAlias({ ...newAlias, planned_name: e.target.value })}
                            style={{ flex: '1 1 140px', padding: '0.5rem', borderRadius: '8px', border: '1px solid var(--border)', outline: 'none', fontWeight: 600 }}
                        />
                        <input
                            ref={newAliasReceiptRef}
                            placeholder="Specific product (e.g. selský jogurt 1%)"
                            value={newAlias.receipt_name}
                            onChange={e => setNewAlias({ ...newAlias, receipt_name: e.target.value })}
                            style={{ flex: '2 1 200px', padding: '0.5rem', borderRadius: '8px', border: '1px solid var(--border)', outline: 'none', fontWeight: 600 }}
                        />
                        <select
                            value={newAlias.shop_id}
                            onChange={e => setNewAlias({ ...newAlias, shop_id: e.target.value })}
                            style={{ flex: '1 1 120px', padding: '0.5rem', borderRadius: '8px', border: '1px solid var(--border)', outline: 'none', background: 'white', fontWeight: 600 }}
                        >
                            <option value="">Any shop</option>
                            {shops.map(s => <option key={s.id} value={s.id}>{s.name}</option>)}
                        </select>
                        <input
                            type="number"
                            placeholder="Price"
                            value={newAlias.last_price}
                            onChange={e => setNewAlias({ ...newAlias, last_price: e.target.value })}
                            style={{ flex: '0 1 80px', padding: '0.5rem', borderRadius: '8px', border: '1px solid var(--border)', outline: 'none' }}
                        />
                        <button onClick={createAlias} className="btn-primary" style={{ padding: '0.5rem 1rem', flex: '0 1 auto', display: 'flex', alignItems: 'center', gap: '0.4rem' }}>
                            <Plus size={16} />
                            Add
                        </button>
                    </div>
                </div>
            </section>

            <section className="card" style={{ marginTop: '3rem' }}>
                <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', flexWrap: 'wrap', gap: '1rem' }}>
                    <div>
                        <h2 style={{ fontSize: '1.125rem', fontWeight: 700, marginBottom: '0.25rem' }}>Flyer Statistics</h2>
                        <p style={{ fontSize: '0.875rem', color: 'var(--text-muted)' }}>View detailed metrics about flyer downloads, parsing status, and errors.</p>
                    </div>
                    <button
                        onClick={() => navigate('/settings/flyer-stats')}
                        className="btn-primary"
                        style={{ display: 'flex', alignItems: 'center', gap: '0.5rem', padding: '0.75rem 1.25rem' }}
                    >
                        <BarChart3 size={20} />
                        View Execution Stats
                    </button>
                </div>
            </section>

            {/* Deletion Confirmation Modal */}
            <Modal
                isOpen={!!deleteConfirm}
                onClose={() => setDeleteConfirm(null)}
                title={`Delete ${deleteConfirm?.type === 'category' ? 'Category' : 'Shop'}?`}
                footer={(
                    <>
                        <button onClick={() => setDeleteConfirm(null)} className="btn-secondary" style={{ padding: '0.5rem 1rem', borderRadius: '8px', border: '1px solid var(--border)', background: 'white' }}>Cancel</button>
                        <button onClick={confirmDelete} className="btn-primary" style={{ padding: '0.5rem 1.5rem', borderRadius: '8px', background: 'var(--danger)' }}>Confirm Delete</button>
                    </>
                )}
            >
                <p style={{ fontWeight: 600, color: 'var(--text-main)' }}>
                    Are you sure you want to delete <strong>{deleteConfirm?.name}</strong>?
                    {deleteConfirm?.type === 'category' && " Items in this category will be uncategorized."}
                    {deleteConfirm?.type === 'shop' && " This action cannot be undone."}
                </p>
            </Modal>
        </div>
    );
};


export default SettingsPage;
