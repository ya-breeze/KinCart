import React, { useState } from 'react';
import { useToast, getApiError } from '../context/ToastContext';
import { API_BASE_URL } from '../config';
import { Plus, Search, ChevronRight, ChevronDown } from 'lucide-react';

const UNITS = ['pcs', 'kg', 'g', '100g', 'l', 'ml', 'pack'];

const PasteItemsPanel = ({ listId, shops, onItemsAdded }) => {
    const { showToast } = useToast();
    const [text, setText] = useState('');
    const [selectedShopId, setSelectedShopId] = useState('');
    const [parsedItems, setParsedItems] = useState(null);
    const [expandedIndex, setExpandedIndex] = useState(null);
    const [loading, setLoading] = useState(false);

    const handleParse = async () => {
        if (!text.trim()) return;
        setLoading(true);
        try {
            const resp = await fetch(`${API_BASE_URL}/api/lists/${listId}/parse-text`, {
                method: 'POST',
                body: JSON.stringify({ text: text.trim(), shop_id: selectedShopId || '' })
            });
            if (resp.ok) {
                const data = await resp.json();
                setParsedItems(data.map(item => ({
                    ...item,
                    price: item.suggested_price || 0,
                    description: item.variants?.[0]?.receipt_name || '',
                })));
                setExpandedIndex(null);
            } else {
                showToast(await getApiError(resp, 'Failed to parse list'));
            }
        } catch {
            showToast('Network error — could not parse list');
        } finally {
            setLoading(false);
        }
    };

    const handleAddAll = async () => {
        if (!parsedItems || parsedItems.length === 0) return;
        setLoading(true);
        try {
            const items = parsedItems.map(item => ({
                name: item.name,
                quantity: item.quantity,
                unit: item.unit,
                price: item.price || 0,
                description: item.description || '',
            }));
            const resp = await fetch(`${API_BASE_URL}/api/lists/${listId}/items/bulk`, {
                method: 'POST',
                body: JSON.stringify(items)
            });
            if (resp.ok) {
                const result = await resp.json();
                onItemsAdded();
                setText('');
                setParsedItems(null);
                setSelectedShopId('');
                setExpandedIndex(null);
                showToast(`Added ${result.created} items`, 'success');
            } else {
                showToast(await getApiError(resp, 'Failed to add items'));
            }
        } catch {
            showToast('Network error — could not add items');
        } finally {
            setLoading(false);
        }
    };

    const updateItem = (index, fields) => {
        setParsedItems(prev => prev.map((item, i) => i === index ? { ...item, ...fields } : item));
    };

    const removeItem = (index) => {
        const updated = parsedItems.filter((_, i) => i !== index);
        setParsedItems(updated.length > 0 ? updated : null);
        if (expandedIndex === index) setExpandedIndex(null);
        else if (expandedIndex !== null && expandedIndex > index) setExpandedIndex(expandedIndex - 1);
    };

    const toggleExpand = (index) => {
        setExpandedIndex(prev => prev === index ? null : index);
    };

    if (parsedItems) {
        return (
            <div style={{ display: 'flex', flexDirection: 'column', gap: '1rem' }}>
                <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                    <span style={{ fontSize: '0.8rem', fontWeight: 800, color: 'var(--text-muted)', textTransform: 'uppercase' }}>
                        Preview — {parsedItems.length} items
                    </span>
                    <button
                        onClick={() => { setParsedItems(null); setExpandedIndex(null); }}
                        style={{ background: 'none', border: 'none', cursor: 'pointer', color: 'var(--text-muted)', fontSize: '0.8rem', fontWeight: 600, minHeight: 'unset' }}
                    >
                        ← Back
                    </button>
                </div>

                <div style={{ display: 'flex', flexDirection: 'column', gap: '0.4rem' }}>
                    {parsedItems.map((item, i) => {
                        const isExpanded = expandedIndex === i;
                        return (
                            <div key={i} style={{
                                background: 'var(--bg-secondary)',
                                borderRadius: '10px',
                                overflow: 'hidden',
                                border: isExpanded ? '1px solid var(--primary)' : '1px solid transparent',
                            }}>
                                {/* Collapsed row */}
                                <div style={{ display: 'flex', alignItems: 'center', gap: '0.5rem', padding: '0.5rem 0.75rem' }}>
                                    <button
                                        onClick={() => toggleExpand(i)}
                                        style={{ background: 'none', border: 'none', cursor: 'pointer', color: 'var(--text-muted)', padding: 0, display: 'flex', alignItems: 'center', minHeight: 'unset', flexShrink: 0 }}
                                        title={isExpanded ? 'Collapse' : 'Edit'}
                                    >
                                        {isExpanded ? <ChevronDown size={16} /> : <ChevronRight size={16} />}
                                    </button>
                                    <span
                                        style={{ flex: 1, fontWeight: 700, fontSize: '0.95rem', cursor: 'pointer' }}
                                        onClick={() => toggleExpand(i)}
                                    >
                                        {item.name}
                                        {item.description && (
                                            <span style={{ fontWeight: 400, fontSize: '0.8rem', color: 'var(--text-muted)', marginLeft: '0.4rem' }}>
                                                {item.description}
                                            </span>
                                        )}
                                    </span>
                                    <input
                                        type="number"
                                        step="0.1"
                                        min="0.1"
                                        value={item.quantity}
                                        onChange={e => updateItem(i, { quantity: parseFloat(e.target.value) || 1 })}
                                        style={{ width: '56px', padding: '4px 6px', borderRadius: '8px', border: '1px solid var(--border)', fontWeight: 700, textAlign: 'center' }}
                                    />
                                    <span style={{ fontSize: '0.72rem', background: 'var(--success)', color: 'white', padding: '2px 8px', borderRadius: '20px', fontWeight: 700, whiteSpace: 'nowrap' }}>
                                        {item.unit}
                                    </span>
                                    {item.price > 0 && (
                                        <span style={{ fontSize: '0.72rem', color: 'var(--text-muted)', fontWeight: 600, whiteSpace: 'nowrap' }}>
                                            ~{item.price.toFixed(2)}
                                        </span>
                                    )}
                                    <button
                                        onClick={() => removeItem(i)}
                                        style={{ background: 'none', border: 'none', cursor: 'pointer', color: 'var(--danger)', fontSize: '1.1rem', fontWeight: 800, lineHeight: 1, padding: '2px 4px', minHeight: 'unset', flexShrink: 0 }}
                                        title="Remove"
                                    >
                                        ×
                                    </button>
                                </div>

                                {/* Expanded editor */}
                                {isExpanded && (
                                    <div style={{ padding: '0.75rem', borderTop: '1px solid var(--border)', display: 'flex', flexDirection: 'column', gap: '0.6rem' }}>
                                        <div style={{ display: 'flex', flexDirection: 'column', gap: '0.2rem' }}>
                                            <label style={{ fontSize: '0.7rem', fontWeight: 800, color: 'var(--text-muted)', textTransform: 'uppercase' }}>Name</label>
                                            <input
                                                value={item.name}
                                                onChange={e => updateItem(i, { name: e.target.value })}
                                                style={{ padding: '0.4rem 0.6rem', borderRadius: '8px', border: '1px solid var(--border)', fontWeight: 700 }}
                                            />
                                        </div>

                                        {item.variants && item.variants.length > 0 && (
                                            <div style={{ display: 'flex', flexDirection: 'column', gap: '0.2rem' }}>
                                                <label style={{ fontSize: '0.7rem', fontWeight: 800, color: 'var(--text-muted)', textTransform: 'uppercase' }}>Variant</label>
                                                <select
                                                    value={item.description}
                                                    onChange={e => {
                                                        const v = item.variants.find(v => v.receipt_name === e.target.value);
                                                        updateItem(i, { description: e.target.value, price: v?.last_price || item.price });
                                                    }}
                                                    style={{ padding: '0.4rem 0.6rem', borderRadius: '8px', border: '1px solid var(--border)', background: 'white', fontWeight: 600 }}
                                                >
                                                    <option value="">— none —</option>
                                                    {item.variants.map(v => (
                                                        <option key={v.receipt_name} value={v.receipt_name}>
                                                            {v.receipt_name}{v.shop_name ? ` (${v.shop_name})` : ''} · {v.last_price.toFixed(2)} ×{v.count}
                                                        </option>
                                                    ))}
                                                </select>
                                            </div>
                                        )}

                                        {(!item.variants || item.variants.length === 0) && (
                                            <div style={{ display: 'flex', flexDirection: 'column', gap: '0.2rem' }}>
                                                <label style={{ fontSize: '0.7rem', fontWeight: 800, color: 'var(--text-muted)', textTransform: 'uppercase' }}>Description</label>
                                                <input
                                                    value={item.description}
                                                    onChange={e => updateItem(i, { description: e.target.value })}
                                                    placeholder="e.g. specific brand or note"
                                                    style={{ padding: '0.4rem 0.6rem', borderRadius: '8px', border: '1px solid var(--border)' }}
                                                />
                                            </div>
                                        )}

                                        <div style={{ display: 'flex', gap: '0.75rem' }}>
                                            <div style={{ display: 'flex', flexDirection: 'column', gap: '0.2rem', flex: 1 }}>
                                                <label style={{ fontSize: '0.7rem', fontWeight: 800, color: 'var(--text-muted)', textTransform: 'uppercase' }}>Unit</label>
                                                <select
                                                    value={item.unit}
                                                    onChange={e => updateItem(i, { unit: e.target.value })}
                                                    style={{ padding: '0.4rem 0.6rem', borderRadius: '8px', border: '1px solid var(--border)', background: 'white', fontWeight: 600 }}
                                                >
                                                    {UNITS.map(u => <option key={u} value={u}>{u}</option>)}
                                                </select>
                                            </div>
                                            <div style={{ display: 'flex', flexDirection: 'column', gap: '0.2rem', flex: 1 }}>
                                                <label style={{ fontSize: '0.7rem', fontWeight: 800, color: 'var(--text-muted)', textTransform: 'uppercase' }}>Price</label>
                                                <input
                                                    type="number"
                                                    step="0.01"
                                                    min="0"
                                                    value={item.price}
                                                    onChange={e => updateItem(i, { price: parseFloat(e.target.value) || 0 })}
                                                    style={{ padding: '0.4rem 0.6rem', borderRadius: '8px', border: '1px solid var(--border)' }}
                                                />
                                            </div>
                                        </div>
                                    </div>
                                )}
                            </div>
                        );
                    })}
                </div>

                <button
                    onClick={handleAddAll}
                    disabled={loading || parsedItems.length === 0}
                    className="btn-primary"
                    style={{ padding: '0.85rem', fontSize: '1rem', borderRadius: '12px' }}
                >
                    <Plus size={20} />
                    {loading ? 'Adding...' : `Add all ${parsedItems.length} items`}
                </button>
            </div>
        );
    }

    return (
        <div style={{ display: 'flex', flexDirection: 'column', gap: '1rem' }}>
            <textarea
                placeholder={'e.g. 2 йогурта, 3 творога, минералка 5, кефир 4+2'}
                value={text}
                onChange={e => setText(e.target.value)}
                rows={4}
                style={{
                    padding: '0.75rem',
                    borderRadius: '12px',
                    border: '1px solid var(--border)',
                    fontFamily: 'inherit',
                    fontSize: '0.95rem',
                    resize: 'vertical',
                    width: '100%',
                    boxSizing: 'border-box'
                }}
            />

            {shops.length > 0 && (
                <div style={{ display: 'flex', flexDirection: 'column', gap: '0.25rem' }}>
                    <label style={{ fontSize: '0.75rem', fontWeight: 800, color: 'var(--text-muted)', textTransform: 'uppercase' }}>
                        Shop (optional — for price hints)
                    </label>
                    <select
                        value={selectedShopId}
                        onChange={e => setSelectedShopId(e.target.value)}
                        style={{ padding: '0.5rem', borderRadius: '12px', border: '1px solid var(--border)', background: 'white', fontWeight: 600 }}
                    >
                        <option value="">Any shop</option>
                        {shops.map(s => (
                            <option key={s.id} value={s.id}>{s.name}</option>
                        ))}
                    </select>
                </div>
            )}

            <button
                onClick={handleParse}
                disabled={loading || !text.trim()}
                className="btn-primary"
                style={{ padding: '0.85rem', fontSize: '1rem', borderRadius: '12px' }}
            >
                <Search size={20} />
                {loading ? 'Parsing...' : 'Parse list'}
            </button>
        </div>
    );
};

export default PasteItemsPanel;
