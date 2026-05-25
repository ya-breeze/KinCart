import React, { useState, useEffect } from 'react';
import { useAuth } from '../context/AuthContext';
import { useToast, getApiError } from '../context/ToastContext';
import { ArrowLeft, Plus, Trash2, Edit2, Check, X, Store, ChevronRight, BarChart3 } from 'lucide-react';
import { useNavigate } from 'react-router-dom';
import { API_BASE_URL } from '../config';
import Modal from '../components/Modal';
import { getCategoryEmoji } from '../utils/categoryEmoji';

const QUICK_EMOJIS = ['🥬','🍎','🥩','🐟','🥛','🧀','🍞','🥚','🥤','🧴','🍿','❄️','🌾','🍝','💧'];

const SettingsPage = () => {
    const { currency, setCurrency } = useAuth();
    const { showToast } = useToast();
    const navigate = useNavigate();
    const [categories, setCategories] = useState([]);
    const [shops, setShops] = useState([]);
    const [selectedShop, setSelectedShop] = useState(null);
    const [newCatName, setNewCatName] = useState('');
    const [newCatIcon, setNewCatIcon] = useState('');
    const [newShopName, setNewShopName] = useState('');
    const [editingCat, setEditingCat] = useState(null);
    const [editName, setEditName] = useState('');
    const [editIcon, setEditIcon] = useState('');
    const [shopCategoryOrders, setShopCategoryOrders] = useState({}); // { shopId: [ { categoryId, sortOrder } ] }
    const [deleteConfirm, setDeleteConfirm] = useState(null); // { type: 'category'|'shop', id, name }
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

    useEffect(() => {
        // eslint-disable-next-line react-hooks/set-state-in-effect
        fetchCategories();
        fetchShops();
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
                body: JSON.stringify({ name: newCatName, icon: newCatIcon.trim(), sort_order: categories.length + 1 })
            });
            if (resp.ok) {
                setNewCatName('');
                setNewCatIcon('');
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
        setEditIcon(cat.icon && cat.icon !== 'package' ? cat.icon : '');
    };

    const saveEdit = async (cat) => {
        try {
            const resp = await fetch(`${API_BASE_URL}/api/categories/${cat.id}`, {
                method: 'PATCH',
                body: JSON.stringify({ ...cat, name: editName, icon: editIcon.trim() })
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
                                <div style={{ flex: 1 }}>
                                    <div style={{ display: 'flex', gap: 4, flexWrap: 'wrap', marginBottom: 6 }}>
                                        {QUICK_EMOJIS.map(e => (
                                            <button key={e} onClick={() => setEditIcon(e)} style={{
                                                fontSize: 18, padding: '2px 4px', borderRadius: 6, border: 'none',
                                                background: editIcon === e ? '#eff6ff' : 'transparent',
                                                cursor: 'pointer', minHeight: 'unset',
                                            }}>{e}</button>
                                        ))}
                                    </div>
                                    <div style={{ display: 'flex', gap: '0.5rem', alignItems: 'center' }}>
                                        <input
                                            data-testid="cat-emoji-input"
                                            value={editIcon}
                                            onChange={e => setEditIcon(e.target.value)}
                                            placeholder="📦"
                                            maxLength={8}
                                            style={{ width: '3rem', textAlign: 'center', fontSize: '1.25rem', padding: '0.25rem', borderRadius: '8px', border: '1px solid var(--border)', outline: 'none', background: 'transparent' }}
                                            title="Emoji (leave blank for auto)"
                                        />
                                        <input
                                            autoFocus
                                            className="card"
                                            style={{ flex: 1, padding: '0.25rem 0.5rem', outline: 'none' }}
                                            value={editName}
                                            onChange={(e) => setEditName(e.target.value)}
                                        />
                                    </div>
                                </div>
                            ) : (
                                <span style={{ flex: 1, fontWeight: 600 }}>{[getCategoryEmoji(cat.name, cat.icon), cat.name].filter(Boolean).join(' ')}</span>
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

                    <div className="card" style={{ border: '2px dashed var(--border)', background: 'transparent' }}>
                        <div style={{ display: 'flex', gap: 4, flexWrap: 'wrap', padding: '4px 8px 0' }}>
                            {QUICK_EMOJIS.map(e => (
                                <button key={e} onClick={() => setNewCatIcon(e)} style={{
                                    fontSize: 18, padding: '2px 4px', borderRadius: 6, border: 'none',
                                    background: newCatIcon === e ? '#eff6ff' : 'transparent',
                                    cursor: 'pointer', minHeight: 'unset',
                                }}>{e}</button>
                            ))}
                        </div>
                        <div style={{ display: 'flex', gap: '0.5rem', flexWrap: 'wrap' }}>
                            <input
                                data-testid="new-cat-emoji-input"
                                value={newCatIcon}
                                onChange={e => setNewCatIcon(e.target.value)}
                                placeholder="📦"
                                maxLength={8}
                                style={{ width: '3rem', textAlign: 'center', fontSize: '1.25rem', padding: '0.5rem', background: 'transparent', border: '1px solid var(--border)', borderRadius: '8px', outline: 'none' }}
                                title="Emoji (leave blank for auto)"
                            />
                            <input
                                placeholder="New category..."
                                value={newCatName}
                                onChange={(e) => setNewCatName(e.target.value)}
                                onKeyDown={e => { if (e.key === 'Enter') addCategory(); }}
                                style={{ flex: '1 1 150px', background: 'transparent', border: 'none', outline: 'none', fontWeight: 600, padding: '0.5rem' }}
                            />
                            <button onClick={addCategory} className="btn-primary" style={{ padding: '0.5rem 1rem', flex: '1 1 auto' }} title="Add a new category">
                                <Plus size={18} />
                                Add
                            </button>
                        </div>
                    </div>

                </div>
            </section>

            <section className="card" style={{ marginTop: '3rem' }}>
                <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', flexWrap: 'wrap', gap: '1rem' }}>
                    <div>
                        <h2 style={{ fontSize: '1.125rem', fontWeight: 700, marginBottom: '0.25rem' }}>Frequent Items</h2>
                        <p style={{ fontSize: '0.875rem', color: 'var(--text-muted)' }}>Restore items you have previously removed from the quick-add chip grid.</p>
                    </div>
                    <button onClick={() => navigate('/settings/frequent-items')} className="btn-primary" style={{ display: 'flex', alignItems: 'center', gap: '0.5rem', padding: '0.75rem 1.25rem' }}>
                        Manage Hidden Items
                    </button>
                </div>
            </section>

            <section className="card" style={{ marginTop: '3rem' }}>
                <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', flexWrap: 'wrap', gap: '1rem' }}>
                    <div>
                        <h2 style={{ fontSize: '1.125rem', fontWeight: 700, marginBottom: '0.25rem' }}>Item Aliases</h2>
                        <p style={{ fontSize: '0.875rem', color: 'var(--text-muted)' }}>Map generic list names to specific products and manage learned aliases.</p>
                    </div>
                    <button onClick={() => navigate('/settings/aliases')} className="btn-primary" style={{ display: 'flex', alignItems: 'center', gap: '0.5rem', padding: '0.75rem 1.25rem' }}>
                        Manage Aliases
                    </button>
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
