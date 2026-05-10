import React, { useState, useEffect, useRef } from 'react';
import { useToast, getApiError } from '../context/ToastContext';
import { ArrowLeft, Plus, Trash2, Edit2, Check, X, ChevronRight } from 'lucide-react';
import { useNavigate } from 'react-router-dom';
import { API_BASE_URL } from '../config';

const AliasesPage = () => {
    const { showToast } = useToast();
    const navigate = useNavigate();
    const [shops, setShops] = useState([]);
    const [aliases, setAliases] = useState([]);
    const [aliasSearch, setAliasSearch] = useState('');
    const [expandedGroups, setExpandedGroups] = useState(new Set());
    const [editingAlias, setEditingAlias] = useState(null);
    const [editingGroup, setEditingGroup] = useState(null);
    const [newAlias, setNewAlias] = useState({ planned_name: '', receipt_name: '', shop_id: '', last_price: '' });
    const newAliasReceiptRef = useRef(null);

    const fetchShops = async () => {
        const resp = await fetch(`${API_BASE_URL}/api/shops`);
        if (resp.ok) setShops(await resp.json());
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

    const renameGroup = async (oldName, newName) => {
        if (!newName.trim() || newName.trim() === oldName) { setEditingGroup(null); return; }
        try {
            const resp = await fetch(`${API_BASE_URL}/api/family/aliases/groups/${encodeURIComponent(oldName)}`, {
                method: 'PATCH',
                body: JSON.stringify({ new_name: newName.trim() })
            });
            if (resp.ok) { setEditingGroup(null); fetchAliases(aliasSearch); }
            else showToast(await getApiError(resp, 'Failed to rename group'));
        } catch {
            showToast('Network error — could not rename group');
        }
    };

    const deleteGroup = async (groupName, count) => {
        if (!window.confirm(`Delete all ${count} alias${count === 1 ? '' : 'es'} for "${groupName}"?`)) return;
        try {
            const resp = await fetch(`${API_BASE_URL}/api/family/aliases/groups/${encodeURIComponent(groupName)}`, { method: 'DELETE' });
            if (resp.ok) fetchAliases(aliasSearch);
            else showToast(await getApiError(resp, 'Failed to delete group'));
        } catch {
            showToast('Network error — could not delete group');
        }
    };

    const prefillGroup = (groupName) => {
        setNewAlias(prev => ({ ...prev, planned_name: groupName, receipt_name: '' }));
        setTimeout(() => newAliasReceiptRef.current?.focus(), 50);
    };

    useEffect(() => {
        fetchShops();
        fetchAliases();
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, []);

    const aliasGroups = aliases.reduce((acc, a) => {
        if (!acc[a.planned_name]) acc[a.planned_name] = [];
        acc[a.planned_name].push(a);
        return acc;
    }, {});
    const groupNames = Object.keys(aliasGroups);

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
                <button onClick={() => navigate('/settings')} className="card" style={{ padding: '0.5rem', borderRadius: '50%', flexShrink: 0 }} title="Back to Settings">
                    <ArrowLeft size={20} />
                </button>
                <h1 style={{ fontSize: '1.25rem', fontWeight: 800 }}>Item Aliases</h1>
            </header>

            <p style={{ fontSize: '0.875rem', color: 'var(--text-muted)', marginBottom: '1.5rem' }}>
                Map generic list names to specific products. Aliases are also learned automatically from confirmed receipts.
            </p>

            <input
                placeholder="Filter by item name..."
                value={aliasSearch}
                onChange={e => { setAliasSearch(e.target.value); fetchAliases(e.target.value); }}
                style={{ width: '100%', padding: '0.5rem 0.75rem', borderRadius: '8px', border: '1px solid var(--border)', outline: 'none', fontWeight: 600, marginBottom: '1rem', boxSizing: 'border-box' }}
            />

            {groupNames.length === 0 ? (
                <p style={{ color: 'var(--text-muted)', fontSize: '0.875rem', textAlign: 'center', padding: '1rem', marginBottom: '1.5rem' }}>
                    No aliases yet. They are created automatically when you confirm receipts, or add one below.
                </p>
            ) : (
                <div style={{ display: 'flex', flexDirection: 'column', gap: '0.375rem', marginBottom: '1.5rem' }}>
                    {groupNames.map(groupName => {
                        const variants = aliasGroups[groupName];
                        const isOpen = expandedGroups.has(groupName);
                        const isEditingGroup = editingGroup?.name === groupName;
                        return (
                            <div key={groupName}>
                                <div
                                    onClick={() => { if (!isEditingGroup) toggleGroup(groupName); }}
                                    style={{ display: 'flex', alignItems: 'center', gap: '0.75rem', padding: '0.75rem 1rem', borderRadius: '12px', background: 'var(--bg-secondary)', cursor: isEditingGroup ? 'default' : 'pointer', userSelect: 'none' }}
                                >
                                    {isEditingGroup ? (
                                        <>
                                            <input
                                                autoFocus
                                                value={editingGroup.newName}
                                                onChange={e => setEditingGroup({ ...editingGroup, newName: e.target.value })}
                                                onKeyDown={e => { if (e.key === 'Enter') renameGroup(groupName, editingGroup.newName); if (e.key === 'Escape') setEditingGroup(null); }}
                                                onClick={e => e.stopPropagation()}
                                                style={{ flex: 1, padding: '0.25rem 0.5rem', borderRadius: '8px', border: '1px solid var(--border)', outline: 'none', fontWeight: 700, fontSize: '0.95rem' }}
                                            />
                                            <button onClick={e => { e.stopPropagation(); renameGroup(groupName, editingGroup.newName); }} style={{ color: 'var(--success)', padding: '0.25rem', background: 'none', border: 'none', cursor: 'pointer' }} title="Save rename"><Check size={16} /></button>
                                            <button onClick={e => { e.stopPropagation(); setEditingGroup(null); }} style={{ color: 'var(--text-muted)', padding: '0.25rem', background: 'none', border: 'none', cursor: 'pointer' }} title="Cancel"><X size={16} /></button>
                                        </>
                                    ) : (
                                        <>
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
                                            <button
                                                onClick={e => { e.stopPropagation(); setEditingGroup({ name: groupName, newName: groupName }); }}
                                                style={{ color: 'var(--primary)', padding: '0.25rem', background: 'none', border: 'none', cursor: 'pointer' }}
                                                title={`Rename "${groupName}"`}
                                            >
                                                <Edit2 size={14} />
                                            </button>
                                            <button
                                                onClick={e => { e.stopPropagation(); deleteGroup(groupName, variants.length); }}
                                                style={{ color: 'var(--danger)', padding: '0.25rem', background: 'none', border: 'none', cursor: 'pointer' }}
                                                title={`Delete all aliases for "${groupName}"`}
                                            >
                                                <Trash2 size={14} />
                                            </button>
                                        </>
                                    )}
                                </div>

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
            )}

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
        </div>
    );
};

export default AliasesPage;
