import React, { useState, useEffect } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import { useAuth } from '../context/AuthContext';
import { ArrowLeft, Check, Send, Trash2, Plus, AlertCircle, ShoppingCart, Image as ImageIcon, Store, Edit2, X } from 'lucide-react';
import { API_BASE_URL } from '../config';
import ImageModal from '../components/ImageModal';

const ListDetail = () => {
    const { id } = useParams();
    const navigate = useNavigate();
    const { user, token, mode, currency } = useAuth();
    const [list, setList] = useState(null);
    const [categories, setCategories] = useState([]);
    const [shops, setShops] = useState([]);
    const [selectedShopId, setSelectedShopId] = useState('');
    const [shopOrder, setShopOrder] = useState([]);
    const [frequentItems, setFrequentItems] = useState([]);
    const [newItem, setNewItem] = useState({ name: '', category_id: '', price: '', description: '', is_urgent: false, quantity: 1, unit: 'pcs' });
    const [selectedPhoto, setSelectedPhoto] = useState(null);
    const [editingItemId, setEditingItemId] = useState(null);
    const [editItemData, setEditItemData] = useState({});
    const [isCreatingCategory, setIsCreatingCategory] = useState(false);
    const [newCategoryName, setNewCategoryName] = useState('');
    const [isRenaming, setIsRenaming] = useState(false);
    const [renameValue, setRenameValue] = useState('');
    const [previewImage, setPreviewImage] = useState(null);

    useEffect(() => {
        fetchList();
        fetchCategories();
        fetchShops();
        fetchFrequentItems();
    }, [id]);

    const fetchShops = async () => {
        const resp = await fetch(`${API_BASE_URL}/api/shops`, {
            headers: { 'Authorization': `Bearer ${token}` }
        });
        if (resp.ok) setShops(await resp.json());
    };

    const fetchShopOrder = async (shopId) => {
        if (!shopId) {
            setShopOrder([]);
            return;
        }
        const resp = await fetch(`${API_BASE_URL}/api/shops/${shopId}/order`, {
            headers: { 'Authorization': `Bearer ${token}` }
        });
        if (resp.ok) setShopOrder(await resp.json());
    };

    const handleShopChange = (e) => {
        const shopId = e.target.value;
        setSelectedShopId(shopId);
        fetchShopOrder(shopId);
    };

    const fetchFrequentItems = async () => {
        const resp = await fetch(`${API_BASE_URL}/api/family/frequent-items`, {
            headers: { 'Authorization': `Bearer ${token}` }
        });
        if (resp.ok) setFrequentItems(await resp.json());
    };

    const fetchCategories = async () => {
        const resp = await fetch(`${API_BASE_URL}/api/categories`, {
            headers: { 'Authorization': `Bearer ${token}` }
        });
        if (resp.ok) setCategories(await resp.json());
    };

    const fetchList = async () => {
        const resp = await fetch(`${API_BASE_URL}/api/lists/${id}`, {
            headers: { 'Authorization': `Bearer ${token}` }
        });
        if (resp.ok) {
            setList(await resp.json());
        }
    };

    const isManager = mode === 'manager';
    const isShopper = mode === 'shopper';

    const addItem = async (e) => {
        e.preventDefault();
        if (!newItem.name) return;

        const resp = await fetch(`${API_BASE_URL}/api/lists/${id}/items`, {
            method: 'POST',
            headers: { 'Authorization': `Bearer ${token}`, 'Content-Type': 'application/json' },
            body: JSON.stringify({
                ...newItem,
                price: parseFloat(newItem.price) || 0,
                quantity: parseFloat(newItem.quantity) || 1,
                category_id: newItem.category_id ? parseInt(newItem.category_id) : undefined
            })
        });

        if (resp.ok) {
            const addedItem = await resp.json();

            // Upload photo if selected
            if (selectedPhoto) {
                const formData = new FormData();
                formData.append('photo', selectedPhoto);
                await fetch(`${API_BASE_URL}/api/items/${addedItem.id}/photo`, {
                    method: 'POST',
                    headers: { 'Authorization': `Bearer ${token}` },
                    body: formData
                });
            }

            setNewItem({ name: '', category_id: '', price: '', description: '', is_urgent: false, quantity: 1, unit: 'pcs' });
            setSelectedPhoto(null);
            fetchList();
            fetchFrequentItems();
        }
    };

    const toggleItem = async (item) => {
        const resp = await fetch(`${API_BASE_URL}/api/items/${item.id}`, {
            method: 'PATCH',
            headers: { 'Authorization': `Bearer ${token}`, 'Content-Type': 'application/json' },
            body: JSON.stringify({ is_bought: !item.is_bought })
        });
        if (resp.ok) fetchList();
    };

    const deleteItem = async (itemId) => {
        const resp = await fetch(`${API_BASE_URL}/api/items/${itemId}`, {
            method: 'DELETE',
            headers: { 'Authorization': `Bearer ${token}` }
        });
        if (resp.ok) fetchList();
    };

    const updateStatus = async (status) => {
        const resp = await fetch(`${API_BASE_URL}/api/lists/${id}`, {
            method: 'PATCH',
            headers: { 'Authorization': `Bearer ${token}`, 'Content-Type': 'application/json' },
            body: JSON.stringify({ status })
        });
        if (resp.ok) fetchList();
    };

    const startEditing = (item) => {
        setEditingItemId(item.id);
        setEditItemData({
            ...item,
            quantity: item.quantity || 1,
            unit: item.unit || 'pcs'
        });
    };

    const saveEdit = async () => {
        const resp = await fetch(`${API_BASE_URL}/api/items/${editingItemId}`, {
            method: 'PATCH',
            headers: { 'Authorization': `Bearer ${token}`, 'Content-Type': 'application/json' },
            body: JSON.stringify({
                ...editItemData,
                price: parseFloat(editItemData.price) || 0,
                quantity: parseFloat(editItemData.quantity) || 1,
                category_id: editItemData.category_id ? parseInt(editItemData.category_id) : undefined
            })
        });
        if (resp.ok) {
            setEditingItemId(null);
            fetchList();
        }
    };

    const handleCreateCategory = async () => {
        if (!newCategoryName.trim()) return;
        const resp = await fetch(`${API_BASE_URL}/api/categories`, {
            method: 'POST',
            headers: { 'Authorization': `Bearer ${token}`, 'Content-Type': 'application/json' },
            body: JSON.stringify({ name: newCategoryName.trim() })
        });
        if (resp.ok) {
            const newCat = await resp.json();
            setCategories([...categories, newCat]);
            setNewItem({ ...newItem, category_id: newCat.id });
            setIsCreatingCategory(false);
            setNewCategoryName('');
        }
    };

    const handleRenameList = async () => {
        if (!renameValue.trim() || renameValue === list.title) {
            setIsRenaming(false);
            return;
        }

        const resp = await fetch(`${API_BASE_URL}/api/lists/${id}`, {
            method: 'PATCH',
            headers: { 'Authorization': `Bearer ${token}`, 'Content-Type': 'application/json' },
            body: JSON.stringify({ title: renameValue.trim() })
        });

        if (resp.ok) {
            setList({ ...list, title: renameValue.trim() });
            setIsRenaming(false);
        }
    };

    const handleDeleteList = async () => {
        if (!window.confirm('Are you sure you want to delete this list? This action cannot be undone.')) return;

        const resp = await fetch(`${API_BASE_URL}/api/lists/${id}`, {
            method: 'DELETE',
            headers: { 'Authorization': `Bearer ${token}` }
        });

        if (resp.ok) {
            navigate('/');
        }
    };

    if (!list) return <div className="container">Loading...</div>;

    const progress = list.items?.length ? (list.items.filter(i => i.is_bought).length / list.items.length) * 100 : 0;

    // Sort categories based on shop order or global order
    const getSortedCategoryIds = () => {
        const allCatIds = categories.map(c => c.id);
        if (selectedShopId && shopOrder.length > 0) {
            const orderMap = {};
            shopOrder.forEach(o => orderMap[o.category_id] = o.sort_order);
            return [...allCatIds].sort((a, b) => (orderMap[a] || 999) - (orderMap[b] || 999));
        }
        return [...allCatIds].sort((a, b) => {
            const catA = categories.find(c => c.id === a);
            const catB = categories.find(c => c.id === b);
            return (catA?.sort_order || 0) - (catB?.sort_order || 0);
        });
    };

    // Group items by category and sort categories
    const sortedCatIds = getSortedCategoryIds();
    const groupedItems = list.items?.reduce((acc, item) => {
        const catId = item.category_id || 'uncategorized';
        if (!acc[catId]) acc[catId] = [];
        acc[catId].push(item);
        return acc;
    }, {}) || {};

    // Ensure all categories in groupedItems are present in sorted list
    const finalSortedCatIds = [...sortedCatIds];
    Object.keys(groupedItems).forEach(id => {
        const numId = id === 'uncategorized' ? id : parseInt(id);
        if (!finalSortedCatIds.includes(numId)) {
            finalSortedCatIds.push(numId);
        }
    });

    const getCategoryName = (id) => categories.find(c => c.id === id)?.name || (id === 'uncategorized' ? 'Uncategorized' : 'Unknown Category');

    return (
        <div className="container">
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
                <div style={{ flex: '1 1 200px', minWidth: '200px' }}>
                    {isRenaming ? (
                        <div style={{ display: 'flex', alignItems: 'center', gap: '0.5rem' }}>
                            <input
                                value={renameValue}
                                onChange={(e) => setRenameValue(e.target.value)}
                                onBlur={handleRenameList}
                                onKeyDown={(e) => e.key === 'Enter' && handleRenameList()}
                                autoFocus
                                style={{
                                    fontSize: '1.25rem',
                                    fontWeight: 800,
                                    padding: '0.25rem',
                                    borderRadius: '8px',
                                    border: '1px solid var(--border)',
                                    width: '100%'
                                }}
                            />
                        </div>
                    ) : (
                        <div style={{ display: 'flex', alignItems: 'center', gap: '0.5rem', flexWrap: 'wrap' }}>
                            <h1 style={{ fontSize: '1.25rem', fontWeight: 800, margin: 0 }}>{list.title}</h1>
                            {isManager && (
                                <button
                                    onClick={() => { setIsRenaming(true); setRenameValue(list.title); }}
                                    style={{ padding: '4px', color: 'var(--text-muted)', cursor: 'pointer', background: 'transparent', border: 'none' }}
                                    title="Rename List"
                                >
                                    <Edit2 size={16} />
                                </button>
                            )}
                        </div>
                    )}
                    <div style={{ display: 'flex', gap: '0.8rem', marginTop: '0.25rem', flexWrap: 'wrap', alignItems: 'center' }}>
                        {isManager ? (
                            <div style={{ display: 'flex', gap: '0.25rem', background: 'var(--bg-secondary)', padding: '2px', borderRadius: '8px', flexWrap: 'wrap' }}>
                                {['preparing', 'ready for shopping', 'completed'].map(status => (
                                    <button
                                        key={status}
                                        onClick={() => updateStatus(status)}
                                        style={{
                                            padding: '4px 8px',
                                            borderRadius: '6px',
                                            fontSize: '0.65rem',
                                            fontWeight: 700,
                                            textTransform: 'uppercase',
                                            cursor: 'pointer',
                                            border: 'none',
                                            background: list.status === status ? 'var(--primary)' : 'transparent',
                                            color: list.status === status ? 'white' : 'var(--text-muted)',
                                            transition: 'all 0.2s',
                                            whiteSpace: 'nowrap'
                                        }}
                                        title={`Mark list as ${status}`}
                                    >
                                        {status.replace('ready for shopping', 'ready')}
                                    </button>
                                ))}
                            </div>
                        ) : (
                            <span className={`badge ${list.status === 'completed' ? 'badge-success' : list.status === 'ready for shopping' ? 'badge-warning' : 'badge-neutral'}`} style={{ fontSize: '0.65rem' }}>
                                {list.status}
                            </span>
                        )}
                        <div style={{ display: 'flex', alignItems: 'center', gap: '0.5rem', color: 'var(--text-muted)', fontSize: '0.75rem', fontWeight: 600, flexWrap: 'wrap' }}>
                            <span>{list.items?.length || 0} items</span>
                            <span>•</span>
                            <span style={{ color: 'var(--success)' }}>
                                Total: {list.items?.reduce((h, i) => h + (i.price * (i.quantity || 1)), 0).toFixed(2)} {currency}
                            </span>
                        </div>
                    </div>
                </div>
                {isShopper && list.status === 'ready for shopping' && (
                    <div style={{ position: 'relative' }}>
                        <select
                            value={selectedShopId}
                            onChange={handleShopChange}
                            title="Select a shop to reorder the list according to its layout"
                            style={{
                                appearance: 'none',
                                padding: '0.5rem 2rem 0.5rem 2.5rem',
                                borderRadius: '20px',
                                border: '1px solid var(--border)',
                                fontSize: '0.8rem',
                                fontWeight: 700,
                                background: 'white',
                                outline: 'none'
                            }}
                        >
                            <option value="">Default Order</option>
                            {shops.map(shop => (
                                <option key={shop.id} value={shop.id}>{shop.name}</option>
                            ))}
                        </select>
                        <Store size={16} style={{ position: 'absolute', left: '10px', top: '50%', transform: 'translateY(-50%)', color: 'var(--primary)' }} />
                    </div>
                )}

                {isManager && (
                    <button
                        onClick={handleDeleteList}
                        className="card"
                        style={{ padding: '0.5rem', borderRadius: '50%', color: 'var(--danger)', border: '1px solid var(--border)' }}
                        title="Delete List"
                    >
                        <Trash2 size={20} />
                    </button>
                )}
            </header>

            {isShopper && list.status === 'ready for shopping' && (
                <div className="card" style={{ marginBottom: '2rem', background: '#22c55e', color: 'white', border: 'none' }}>
                    <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: '0.75rem', fontWeight: 600 }}>
                        <span>Progressing...</span>
                        <span>{Math.round(progress)}%</span>
                    </div>
                    <div style={{ height: '8px', background: 'rgba(255,255,255,0.3)', borderRadius: '4px', overflow: 'hidden' }}>
                        <div style={{ width: `${progress}%`, height: '100%', background: 'white', transition: 'width 0.3s ease' }} />
                    </div>
                </div>
            )}

            {/* Urgent Update Simulation Mockup */}
            {isShopper && list.items?.some(i => i.is_urgent && !i.is_bought) && (
                <div className="card" style={{ background: '#f97316', color: 'white', marginBottom: '1.5rem', display: 'flex', alignItems: 'center', gap: '1rem', border: 'none' }}>
                    <AlertCircle size={24} />
                    <div style={{ flex: 1 }}>
                        <p style={{ fontWeight: 800, fontSize: '0.9rem' }}>URGENT ADDITION!</p>
                        <p style={{ fontSize: '0.8rem' }}>Manager added a new item</p>
                    </div>
                    <button className="glass" style={{ padding: '0.5rem 1rem', borderRadius: '8px', color: 'white', fontWeight: 600 }}>OK</button>
                </div>
            )}

            {/* Grouped Items List */}
            <div style={{ display: 'flex', flexDirection: 'column', gap: '1.5rem', marginBottom: '6rem' }}>
                {finalSortedCatIds.filter(catId => groupedItems[catId]).map(catId => (
                    <div key={catId}>
                        <h3 style={{ fontSize: '1rem', fontWeight: 700, marginBottom: '0.75rem', color: 'var(--text-muted)', textTransform: 'uppercase', letterSpacing: '0.05em' }}>
                            {getCategoryName(catId)}
                        </h3>
                        <div style={{ display: 'flex', flexDirection: 'column', gap: '0.75rem' }}>
                            {groupedItems[catId].map(item => (
                                <div key={item.id} className="card" style={{
                                    display: 'flex',
                                    alignItems: 'center',
                                    gap: '1rem',
                                    opacity: item.is_bought ? 0.6 : 1,
                                    borderLeft: item.is_urgent ? '4px solid var(--danger)' : 'none',
                                    flexWrap: 'wrap'
                                }}>
                                    {isShopper ? (
                                        <button
                                            onClick={() => toggleItem(item)}
                                            title={item.is_bought ? "Mark as not bought" : "Mark as bought"}
                                            style={{
                                                width: '28px',
                                                height: '28px',
                                                borderRadius: '6px',
                                                border: `2px solid ${item.is_bought ? 'var(--success)' : 'var(--border)'}`,
                                                background: item.is_bought ? 'var(--success)' : 'transparent',
                                                display: 'flex',
                                                alignItems: 'center',
                                                justifyContent: 'center',
                                                color: 'white',
                                                flexShrink: 0
                                            }}
                                        >
                                            {item.is_bought && <Check size={18} />}
                                        </button>
                                    ) : (
                                        <div style={{ width: '8px', height: '8px', borderRadius: '50%', background: 'var(--primary)', flexShrink: 0 }} />
                                    )}

                                    {item.local_photo_path && (
                                        <div style={{ width: '48px', height: '48px', borderRadius: '8px', overflow: 'hidden', flexShrink: 0 }}>
                                            <img
                                                src={`${API_BASE_URL}${item.local_photo_path}`}
                                                alt={item.name}
                                                style={{ width: '100%', height: '100%', objectFit: 'cover', cursor: 'zoom-in' }}
                                                onClick={() => setPreviewImage({ src: `${API_BASE_URL}${item.local_photo_path}`, alt: item.name })}
                                            />
                                        </div>
                                    )}

                                    {editingItemId === item.id ? (
                                        <div style={{ flex: 1, display: 'flex', flexDirection: 'column', gap: '0.75rem' }}>
                                            {editItemData.flyer_item_id && (
                                                <div style={{
                                                    display: 'flex',
                                                    justifyContent: 'space-between',
                                                    alignItems: 'center',
                                                    background: 'rgba(var(--primary-rgb), 0.1)',
                                                    padding: '0.5rem 0.75rem',
                                                    borderRadius: '8px',
                                                    border: '1px solid var(--primary)',
                                                    marginBottom: '0.25rem'
                                                }}>
                                                    <span style={{ fontSize: '0.7rem', fontWeight: 800, color: 'var(--primary)', display: 'flex', alignItems: 'center', gap: '4px' }}>
                                                        <ShoppingCart size={12} /> LINKED TO FLYER DEAL
                                                    </span>
                                                    <button
                                                        type="button"
                                                        onClick={() => setEditItemData({ ...editItemData, flyer_item_id: null })}
                                                        style={{
                                                            fontSize: '0.65rem',
                                                            fontWeight: 900,
                                                            color: 'var(--danger)',
                                                            background: 'white',
                                                            border: '1px solid var(--danger)',
                                                            padding: '4px 8px',
                                                            borderRadius: '6px',
                                                            cursor: 'pointer',
                                                            textTransform: 'uppercase'
                                                        }}
                                                    >
                                                        Unlink
                                                    </button>
                                                </div>
                                            )}
                                            <div style={{ display: 'flex', gap: '0.5rem' }}>
                                                <input
                                                    disabled={!!editItemData.flyer_item_id}
                                                    style={{
                                                        flex: 2,
                                                        padding: '0.5rem',
                                                        borderRadius: '8px',
                                                        border: '1px solid var(--border)',
                                                        fontWeight: 700,
                                                        background: editItemData.flyer_item_id ? 'var(--bg-secondary)' : 'white',
                                                        cursor: editItemData.flyer_item_id ? 'not-allowed' : 'text'
                                                    }}
                                                    value={editItemData.name}
                                                    onChange={e => setEditItemData({ ...editItemData, name: e.target.value })}
                                                    title={editItemData.flyer_item_id ? "Cannot change name of a linked item" : ""}
                                                />
                                                <select
                                                    style={{ flex: 1, padding: '0.5rem', borderRadius: '8px', border: '1px solid var(--border)', background: 'white', fontWeight: 600 }}
                                                    value={editItemData.category_id}
                                                    onChange={e => setEditItemData({ ...editItemData, category_id: e.target.value })}
                                                >
                                                    {categories.map(c => <option key={c.id} value={c.id}>{c.name}</option>)}
                                                </select>
                                            </div>
                                            <div style={{ display: 'flex', gap: '0.5rem', alignItems: 'center' }}>
                                                <div style={{ display: 'flex', flex: 1, gap: '0.25rem' }}>
                                                    <input
                                                        type="number"
                                                        step="0.1"
                                                        style={{ width: '70px', padding: '0.5rem', borderRadius: '8px', border: '1px solid var(--border)', fontWeight: 700 }}
                                                        value={editItemData.quantity}
                                                        onChange={e => setEditItemData({ ...editItemData, quantity: e.target.value })}
                                                    />
                                                    <select
                                                        style={{ width: '80px', padding: '0.5rem', borderRadius: '8px', border: '1px solid var(--border)', background: 'white', fontWeight: 600 }}
                                                        value={editItemData.unit}
                                                        onChange={e => setEditItemData({ ...editItemData, unit: e.target.value })}
                                                    >
                                                        {['pcs', 'kg', 'g', '100g', 'l', 'pack'].map(u => <option key={u} value={u}>{u}</option>)}
                                                    </select>
                                                </div>
                                                <input
                                                    disabled={!!editItemData.flyer_item_id}
                                                    type="number"
                                                    style={{
                                                        flex: 1,
                                                        padding: '0.5rem',
                                                        borderRadius: '8px',
                                                        border: '1px solid var(--border)',
                                                        color: 'var(--text-muted)',
                                                        background: editItemData.flyer_item_id ? 'var(--bg-secondary)' : 'white',
                                                        cursor: editItemData.flyer_item_id ? 'not-allowed' : 'text'
                                                    }}
                                                    placeholder="Price (Opt)"
                                                    value={editItemData.price}
                                                    onChange={e => setEditItemData({ ...editItemData, price: e.target.value })}
                                                    title={editItemData.flyer_item_id ? "Cannot change price of a linked item" : ""}
                                                />
                                            </div>
                                            <textarea
                                                style={{ padding: '0.5rem', borderRadius: '8px', border: '1px solid var(--border)', minHeight: '60px', fontFamily: 'inherit' }}
                                                placeholder="Description / Instructions"
                                                value={editItemData.description}
                                                onChange={e => setEditItemData({ ...editItemData, description: e.target.value })}
                                            />
                                        </div>
                                    ) : (
                                        <div style={{ flex: 1 }}>
                                            <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', flexWrap: 'wrap', gap: '0.5rem' }}>
                                                <div style={{ display: 'flex', alignItems: 'center', gap: '0.75rem', flexWrap: 'wrap' }}>
                                                    <p className="text-break" style={{ fontWeight: 800, textDecoration: item.is_bought ? 'line-through' : 'none', fontSize: '1.15rem', color: 'var(--text-dark)' }}>
                                                        {item.name}
                                                    </p>
                                                    {item.flyer_item_id && (
                                                        <span style={{
                                                            fontSize: '0.65rem',
                                                            background: 'var(--primary)',
                                                            color: 'white',
                                                            padding: '2px 8px',
                                                            borderRadius: '6px',
                                                            fontWeight: 900,
                                                            display: 'flex',
                                                            alignItems: 'center',
                                                            gap: '4px',
                                                            textTransform: 'uppercase'
                                                        }}>
                                                            <ShoppingCart size={10} />
                                                            Sale Deal
                                                        </span>
                                                    )}
                                                    <span style={{
                                                        fontSize: '0.9rem',
                                                        background: 'var(--success)',
                                                        color: 'white',
                                                        padding: '4px 12px',
                                                        borderRadius: '20px',
                                                        fontWeight: 800,
                                                        boxShadow: 'var(--shadow-sm)',
                                                        whiteSpace: 'nowrap'
                                                    }}>
                                                        {item.quantity || 1} {item.unit || 'pcs'}
                                                    </span>
                                                    {item.is_urgent && (
                                                        <span style={{
                                                            fontSize: '0.7rem',
                                                            background: 'var(--danger)',
                                                            color: 'white',
                                                            padding: '2px 8px',
                                                            borderRadius: '6px',
                                                            fontWeight: 900,
                                                            letterSpacing: '0.05em'
                                                        }}>
                                                            URGENT
                                                        </span>
                                                    )}
                                                </div>
                                                <div style={{ textAlign: 'right' }}>
                                                    {item.price > 0 && (
                                                        <div style={{ display: 'flex', flexDirection: 'column', alignItems: 'flex-end' }}>
                                                            <p style={{ fontWeight: 800, fontSize: '1rem', color: 'var(--text-dark)' }}>
                                                                ≈ {(item.price * (item.quantity || 1)).toFixed(2)} {currency}
                                                            </p>
                                                            <p style={{ fontSize: '0.75rem', color: 'var(--text-muted)', fontWeight: 700 }}>
                                                                ({item.price}/{item.unit || 'pcs'})
                                                            </p>
                                                        </div>
                                                    )}
                                                </div>
                                            </div>
                                            {item.description && (
                                                <p style={{
                                                    fontSize: '0.95rem',
                                                    color: 'var(--text-main)',
                                                    marginTop: '0.6rem',
                                                    paddingTop: '0.4rem',
                                                    borderTop: '1px solid var(--border)',
                                                    lineHeight: '1.4'
                                                }}>
                                                    {item.description}
                                                </p>
                                            )}
                                        </div>
                                    )}

                                    {isManager && (
                                        <div style={{ display: 'flex', gap: '0.25rem' }}>
                                            {editingItemId === item.id ? (
                                                <>
                                                    <button onClick={saveEdit} style={{ color: 'var(--success)', padding: '0.5rem' }} title="Save Changes"><Check size={18} /></button>
                                                    <button onClick={() => setEditingItemId(null)} style={{ color: 'var(--text-muted)', padding: '0.5rem' }} title="Cancel"><X size={18} /></button>
                                                </>
                                            ) : (
                                                <>
                                                    <button onClick={() => startEditing(item)} style={{ color: 'var(--primary)', padding: '0.5rem' }} title="Edit Item"><Edit2 size={18} /></button>
                                                    <button onClick={() => deleteItem(item.id)} style={{ color: 'var(--danger)', padding: '0.5rem' }} title="Delete Item"><Trash2 size={18} /></button>
                                                </>
                                            )}
                                        </div>
                                    )}
                                </div>
                            ))}
                        </div>
                    </div>
                ))}

                {isManager && (
                    <div className="card" style={{ marginTop: '1rem', border: '2px dashed var(--border)', background: 'transparent' }}>
                        <h3 style={{ marginBottom: '1rem', fontSize: '1rem', fontWeight: 700 }}>Add New Item</h3>

                        {frequentItems.length > 0 && (
                            <div style={{ marginBottom: '1rem' }}>
                                <p style={{ fontSize: '0.75rem', color: 'var(--text-muted)', marginBottom: '0.5rem', fontWeight: 600 }}>FREQUENTLY USED:</p>
                                <div style={{ display: 'flex', gap: '0.5rem', flexWrap: 'wrap' }}>
                                    {frequentItems.map(fi => (
                                        <button
                                            key={fi.id}
                                            onClick={() => setNewItem({ ...newItem, name: fi.item_name })}
                                            className="glass"
                                            style={{ padding: '4px 10px', borderRadius: '20px', fontSize: '0.75rem', fontWeight: 600 }}
                                            title={`Quick add ${fi.item_name}`}
                                        >
                                            {fi.item_name}
                                        </button>
                                    ))}
                                </div>
                            </div>
                        )}

                        <form onSubmit={addItem} style={{ display: 'flex', flexDirection: 'column', gap: '1.25rem' }}>
                            <div style={{ display: 'flex', flexDirection: 'column', gap: '0.5rem' }}>
                                <label style={{ fontSize: '0.8rem', fontWeight: 800, color: 'var(--text-muted)', textTransform: 'uppercase' }}>Item Name</label>
                                <input
                                    style={{ padding: '1rem', borderRadius: '12px', border: '1px solid var(--border)', outline: 'none', fontSize: '1.1rem', fontWeight: 700 }}
                                    placeholder="e.g. Organic Bananas"
                                    value={newItem.name}
                                    onChange={e => setNewItem({ ...newItem, name: e.target.value })}
                                />
                            </div>

                            <div className="flex-responsive" style={{ gap: '1rem', flexWrap: 'wrap' }}>
                                <div style={{ flex: '1 1 200px', display: 'flex', flexDirection: 'column', gap: '0.5rem' }}>
                                    <label style={{ fontSize: '0.8rem', fontWeight: 800, color: 'var(--text-muted)', textTransform: 'uppercase' }}>Quantity & Measure</label>
                                    <div style={{ display: 'flex', gap: '0.5rem' }}>
                                        <input
                                            type="number"
                                            step="0.1"
                                            style={{ flex: 1, padding: '0.75rem', borderRadius: '12px', border: '1px solid var(--border)', outline: 'none', fontWeight: 700 }}
                                            placeholder="1.5"
                                            value={newItem.quantity}
                                            onChange={e => setNewItem({ ...newItem, quantity: e.target.value })}
                                        />
                                        <select
                                            style={{ width: '100px', padding: '0.75rem', borderRadius: '12px', border: '1px solid var(--border)', outline: 'none', background: 'white', fontWeight: 600 }}
                                            value={newItem.unit}
                                            onChange={e => setNewItem({ ...newItem, unit: e.target.value })}
                                        >
                                            <option value="pcs">pcs</option>
                                            <option value="kg">kg</option>
                                            <option value="g">g</option>
                                            <option value="100g">100g</option>
                                            <option value="l">l</option>
                                            <option value="pack">pack</option>
                                        </select>
                                    </div>
                                </div>
                                <div style={{ flex: '1 1 200px', display: 'flex', flexDirection: 'column', gap: '0.5rem' }}>
                                    <label style={{ fontSize: '0.8rem', fontWeight: 800, color: 'var(--text-muted)', textTransform: 'uppercase' }}>Category</label>
                                    {isCreatingCategory ? (
                                        <div style={{ display: 'flex', gap: '0.5rem' }}>
                                            <input
                                                style={{ flex: 1, padding: '0.75rem', borderRadius: '12px', border: '1px solid var(--border)', outline: 'none', fontWeight: 600 }}
                                                placeholder="New Category Name"
                                                autoFocus
                                                value={newCategoryName}
                                                onChange={e => setNewCategoryName(e.target.value)}
                                            />
                                            <button
                                                type="button"
                                                onClick={handleCreateCategory}
                                                style={{ padding: '0.5rem', borderRadius: '8px', background: 'var(--success)', color: 'white', border: 'none' }}
                                                title="Save Category"
                                            >
                                                <Check size={20} />
                                            </button>
                                            <button
                                                type="button"
                                                onClick={() => { setIsCreatingCategory(false); setNewCategoryName(''); }}
                                                style={{ padding: '0.5rem', borderRadius: '8px', background: 'var(--bg-secondary)', color: 'var(--text-muted)', border: 'none' }}
                                                title="Cancel"
                                            >
                                                <X size={20} />
                                            </button>
                                        </div>
                                    ) : (
                                        <div style={{ display: 'flex', gap: '0.5rem' }}>
                                            <select
                                                style={{ flex: 1, padding: '0.75rem', borderRadius: '12px', border: '1px solid var(--border)', outline: 'none', background: 'white', fontWeight: 600 }}
                                                value={newItem.category_id}
                                                onChange={e => {
                                                    if (e.target.value === 'NEW') {
                                                        setIsCreatingCategory(true);
                                                    } else {
                                                        setNewItem({ ...newItem, category_id: e.target.value });
                                                    }
                                                }}
                                            >
                                                <option value="">Select...</option>
                                                {categories.map(c => (
                                                    <option key={c.id} value={c.id}>{c.name}</option>
                                                ))}
                                                <option value="NEW" style={{ fontWeight: 800, color: 'var(--primary)' }}>+ Create New Category</option>
                                            </select>
                                        </div>
                                    )}
                                </div>
                            </div>

                            <div style={{ display: 'flex', flexDirection: 'column', gap: '0.5rem' }}>
                                <label style={{ fontSize: '0.8rem', fontWeight: 800, color: 'var(--text-muted)', textTransform: 'uppercase' }}>Description / Specific Instructions</label>
                                <textarea
                                    style={{ padding: '1rem', borderRadius: '12px', border: '1px solid var(--border)', outline: 'none', minHeight: '80px', fontFamily: 'inherit', resize: 'vertical' }}
                                    placeholder="e.g. Get the ripest ones, preferably with some brown spots."
                                    value={newItem.description}
                                    onChange={e => setNewItem({ ...newItem, description: e.target.value })}
                                />
                            </div>

                            <div className="flex-responsive" style={{ gap: '1rem', alignItems: 'flex-end', flexWrap: 'wrap' }}>
                                <div style={{ flex: '1 1 200px', display: 'flex', flexDirection: 'column', gap: '0.5rem' }}>
                                    <label style={{ fontSize: '0.8rem', fontWeight: 800, color: 'var(--text-muted)', textTransform: 'uppercase' }}>Price (Optional)</label>
                                    <input
                                        type="number"
                                        style={{ padding: '0.75rem', borderRadius: '12px', border: '1px solid var(--border)', outline: 'none', color: 'var(--text-muted)' }}
                                        placeholder={`Price per ${newItem.unit}`}
                                        value={newItem.price}
                                        onChange={e => setNewItem({ ...newItem, price: e.target.value })}
                                    />
                                </div>
                                <label style={{ flex: '1 1 200px', display: 'flex', alignItems: 'center', gap: '0.75rem', fontSize: '0.95rem', fontWeight: 700, cursor: 'pointer', padding: '0.75rem', borderRadius: '12px', background: newItem.is_urgent ? '#fee2e2' : 'var(--bg-secondary)', border: `1px solid ${newItem.is_urgent ? 'var(--danger)' : 'var(--border)'}`, transition: 'all 0.2s', height: 'fit-content' }}>
                                    <input
                                        type="checkbox"
                                        checked={newItem.is_urgent}
                                        onChange={e => setNewItem({ ...newItem, is_urgent: e.target.checked })}
                                        style={{ width: '18px', height: '18px' }}
                                    />
                                    Mark as Urgent
                                </label>
                            </div>

                            {newItem.price > 0 && newItem.quantity > 0 && (
                                <div style={{ padding: '0.75rem', background: 'var(--bg-secondary)', borderRadius: '12px', border: '1px solid var(--border)', textAlign: 'center' }}>
                                    <p style={{ fontSize: '0.85rem', color: 'var(--text-muted)', fontWeight: 700, marginBottom: '0.25rem' }}>ESTIMATED TOTAL</p>
                                    <p style={{ fontSize: '1.25rem', fontWeight: 900, color: 'var(--success)' }}>
                                        {(newItem.quantity * newItem.price).toFixed(2)} {currency}
                                    </p>
                                    <p style={{ fontSize: '0.75rem', color: 'var(--text-muted)', fontWeight: 600 }}>
                                        {newItem.quantity} {newItem.unit} × {newItem.price} {currency}/{newItem.unit}
                                    </p>
                                </div>
                            )}

                            <div style={{ display: 'flex', alignItems: 'center', gap: '1rem' }}>
                                <input
                                    type="file"
                                    id="photo-upload"
                                    accept="image/*"
                                    style={{ display: 'none' }}
                                    onChange={e => setSelectedPhoto(e.target.files[0])}
                                />
                                <label
                                    htmlFor="photo-upload"
                                    className="glass"
                                    style={{
                                        display: 'flex',
                                        alignItems: 'center',
                                        gap: '0.5rem',
                                        padding: '0.75rem 1.25rem',
                                        borderRadius: '12px',
                                        cursor: 'pointer',
                                        fontSize: '0.9rem',
                                        fontWeight: 700,
                                        color: selectedPhoto ? 'var(--success)' : 'inherit',
                                        border: '1px solid var(--border)'
                                    }}
                                >
                                    <ImageIcon size={20} />
                                    {selectedPhoto ? 'Photo Added' : 'Add Photo'}
                                </label>
                                {selectedPhoto && (
                                    <span style={{ fontSize: '0.8rem', color: 'var(--text-muted)', fontWeight: 600 }}>{selectedPhoto.name}</span>
                                )}
                            </div>

                            <button type="submit" className="btn-primary" style={{ padding: '1rem', fontSize: '1.1rem', borderRadius: '12px' }}>
                                <Plus size={24} />
                                Add Item to List
                            </button>
                        </form>
                    </div>
                )}
            </div>

            {isShopper && list.status === 'ready for shopping' && (
                <footer style={{ position: 'fixed', bottom: 0, left: 0, right: 0, padding: '1rem', background: 'white', borderTop: '1px solid var(--border)', display: 'flex', justifyContent: 'center' }}>
                    <div className="container" style={{ padding: 0 }}>
                        <button onClick={() => updateStatus('completed')} className="btn-primary" style={{ width: '100%' }} title="Mark the shopping as completed and finish the list">
                            <Check size={20} />
                            Complete Shopping
                        </button>
                    </div>
                </footer>
            )}
            <ImageModal
                src={previewImage?.src}
                alt={previewImage?.alt}
                onClose={() => setPreviewImage(null)}
            />
        </div>
    );
};

export default ListDetail;
