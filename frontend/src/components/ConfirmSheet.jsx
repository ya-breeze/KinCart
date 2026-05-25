import React from 'react';
import { X, Plus } from 'lucide-react';
import { getCategoryEmoji } from '../utils/categoryEmoji';

const UNITS = ['pcs', 'g', 'kg', 'ml', 'L', 'pack'];

const ConfirmSheet = ({ draft, onChange, onCancel, onConfirm, categories, currency }) => {
    if (!draft) return null;

    const set = (k, v) => {
        const update = { ...draft, [k]: v };
        if (k === 'variant' && v?.last_price) update.price = String(v.last_price);
        onChange(update);
    };

    const bumpQty = (delta) => {
        const n = parseFloat(draft.qty) || 0;
        const next = Math.max(0, +(n + delta).toFixed(2));
        set('qty', String(next));
    };

    const isNew = draft.source === 'new';

    const selectedCat = categories.find(c => c.id === draft.category_id) || null;
    const headerEmoji = selectedCat
        ? getCategoryEmoji(selectedCat.name, selectedCat.icon)
        : (draft.emoji || '');

    return (
        <div
            onClick={onCancel}
            style={{
                position: 'fixed', inset: 0,
                background: 'rgba(15,23,42,.55)',
                backdropFilter: 'blur(3px)',
                zIndex: 50,
                display: 'flex',
                alignItems: 'flex-end',
            }}
        >
            <div
                onClick={e => e.stopPropagation()}
                style={{
                    background: '#fff',
                    width: '100%',
                    borderRadius: '20px 20px 0 0',
                    padding: '12px 16px 24px',
                    animation: 'kcSheetUp .25s ease',
                    maxHeight: '92vh',
                    overflowY: 'auto',
                }}
            >
                {/* drag handle */}
                <div style={{ width: 40, height: 4, background: '#cbd5e1', borderRadius: 2, margin: '0 auto 10px' }} />

                {/* header: emoji + name + source + close */}
                <div style={{ display: 'flex', alignItems: 'center', gap: 10, marginBottom: 14 }}>
                    {headerEmoji && <div data-testid="sheet-header-emoji" style={{ width: 44, height: 44, borderRadius: 12, background: '#f1f5f9', display: 'flex', alignItems: 'center', justifyContent: 'center', fontSize: 22, flexShrink: 0 }}>
                        {headerEmoji}
                    </div>}
                    <div style={{ flex: 1, minWidth: 0 }}>
                        <input
                            data-testid="sheet-name-input"
                            value={draft.name}
                            onChange={e => set('name', e.target.value)}
                            style={{ width: '100%', padding: '4px 0', border: 'none', borderBottom: '1.5px dashed #e2e8f0', fontFamily: 'Inter, system-ui, sans-serif', fontSize: 16, fontWeight: 700, color: '#020617', outline: 'none', background: 'transparent', minHeight: 'unset' }}
                        />
                        <div style={{ fontSize: 10.5, color: isNew ? '#92400e' : '#1e3a8a', fontWeight: 600, marginTop: 3, display: 'flex', alignItems: 'center', gap: 4 }}>
                            {isNew ? <>⚠️ New item — confirm details</> : <>✦ Pre-filled from history</>}
                        </div>
                    </div>
                    <button
                        onClick={onCancel}
                        style={{ width: 30, height: 30, borderRadius: '50%', background: '#f1f5f9', color: '#475569', border: 'none', display: 'flex', alignItems: 'center', justifyContent: 'center', cursor: 'pointer', flexShrink: 0, minHeight: 'unset' }}
                    >
                        <X size={16} />
                    </button>
                </div>

                {/* Receipt variant picker */}
                {(draft.variants || []).length > 0 && (
                    <div style={{ marginBottom: 14 }}>
                        <div style={{ fontSize: 10, fontWeight: 700, color: '#64748b', textTransform: 'uppercase', letterSpacing: '.05em', marginBottom: 6, display: 'flex', alignItems: 'center', gap: 6 }}>
                            Receipt Variant
                            <span style={{ fontSize: 9, fontWeight: 500, color: '#94a3b8', textTransform: 'none', letterSpacing: 0 }}>alias → actual SKU on receipt</span>
                        </div>
                        <div style={{ display: 'flex', gap: 6, overflowX: 'auto', scrollbarWidth: 'none', paddingBottom: 2 }}>
                            {draft.variants.map(v => {
                                const sel = draft.variant?.id === v.id;
                                return (
                                    <button key={v.id} onClick={() => set('variant', v)} style={{
                                        flexShrink: 0, padding: '7px 10px', borderRadius: 10, textAlign: 'left', cursor: 'pointer',
                                        border: sel ? '1.5px solid #2563eb' : '1px solid #e2e8f0',
                                        background: sel ? '#eff6ff' : '#fff',
                                        display: 'flex', flexDirection: 'column', gap: 2,
                                        minWidth: 120, maxWidth: 165, minHeight: 'unset',
                                    }}>
                                        {sel && <span style={{ color: '#2563eb', fontSize: 8.5, fontWeight: 800, letterSpacing: '.04em' }}>✓ SELECTED</span>}
                                        <span style={{ fontSize: 12, fontWeight: 600, color: '#0f172a', whiteSpace: 'nowrap', overflow: 'hidden', textOverflow: 'ellipsis', maxWidth: 145 }}>{v.receipt_name}</span>
                                        <span style={{ fontSize: 10.5, color: '#64748b', fontVariantNumeric: 'tabular-nums' }}>
                                            {v.last_price > 0 ? `${v.last_price} ${currency}` : '—'}
                                            {v.shop?.name ? ` · ${v.shop.name}` : ''}
                                        </span>
                                    </button>
                                );
                            })}
                            <button onClick={() => set('variant', null)} style={{
                                flexShrink: 0, padding: '7px 12px', borderRadius: 10, border: '1px dashed #cbd5e1',
                                background: '#f8fafc', cursor: 'pointer', display: 'flex', flexDirection: 'column',
                                alignItems: 'center', justifyContent: 'center', minWidth: 100, gap: 2, minHeight: 'unset',
                            }}>
                                <span style={{ fontSize: 17, color: '#94a3b8' }}>+</span>
                                <span style={{ fontSize: 10.5, color: '#94a3b8', fontWeight: 600, textAlign: 'center', lineHeight: 1.3 }}>Link receipt<br />item</span>
                            </button>
                        </div>
                    </div>
                )}

                {/* Qty row */}
                <div style={{ marginBottom: 10 }}>
                    <div style={{ fontSize: 10, fontWeight: 700, color: '#64748b', textTransform: 'uppercase', letterSpacing: '.05em', marginBottom: 4 }}>Qty</div>
                    <div style={{ display: 'flex', alignItems: 'stretch', borderRadius: 10, border: '1px solid #e2e8f0', overflow: 'hidden', background: '#fff' }}>
                        <button onClick={() => bumpQty(-1)} style={{ width: 44, background: '#f8fafc', border: 'none', color: '#475569', fontSize: 20, fontWeight: 700, cursor: 'pointer', flexShrink: 0, minHeight: 'unset' }}>−</button>
                        <input
                            data-testid="sheet-qty-input"
                            value={draft.qty}
                            onChange={e => set('qty', e.target.value)}
                            style={{ flex: 1, minWidth: 0, padding: '10px 4px', border: 'none', textAlign: 'center', fontFamily: 'Inter, system-ui, sans-serif', fontSize: 16, fontWeight: 700, fontVariantNumeric: 'tabular-nums', outline: 'none', minHeight: 'unset' }}
                        />
                        <button onClick={() => bumpQty(1)} style={{ width: 44, background: '#f8fafc', border: 'none', color: '#475569', fontSize: 20, fontWeight: 700, cursor: 'pointer', flexShrink: 0, minHeight: 'unset' }}>+</button>
                    </div>
                </div>

                {/* Unit / Price row */}
                <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 8, marginBottom: 14 }}>
                    <div>
                        <div style={{ fontSize: 10, fontWeight: 700, color: '#64748b', textTransform: 'uppercase', letterSpacing: '.05em', marginBottom: 4 }}>Unit</div>
                        <select
                            value={draft.unit}
                            onChange={e => set('unit', e.target.value)}
                            style={{ width: '100%', padding: '9px 8px', borderRadius: 10, border: '1px solid #e2e8f0', fontFamily: 'Inter, system-ui, sans-serif', fontSize: 14, fontWeight: 600, outline: 'none', background: '#fff', minHeight: 'unset' }}
                        >
                            {UNITS.map(u => <option key={u}>{u}</option>)}
                        </select>
                    </div>
                    <div>
                        <div style={{ fontSize: 10, fontWeight: 700, color: '#64748b', textTransform: 'uppercase', letterSpacing: '.05em', marginBottom: 4 }}>Price</div>
                        <input
                            value={draft.price}
                            onChange={e => set('price', e.target.value)}
                            placeholder="—"
                            style={{ width: '100%', padding: '9px 10px', borderRadius: 10, border: '1px solid #e2e8f0', fontFamily: 'Inter, system-ui, sans-serif', fontSize: 14, fontWeight: 600, fontVariantNumeric: 'tabular-nums', outline: 'none', minHeight: 'unset', boxSizing: 'border-box' }}
                        />
                    </div>
                </div>

                {/* Category chips */}
                <div style={{ fontSize: 10, fontWeight: 700, color: '#64748b', textTransform: 'uppercase', letterSpacing: '.05em', marginBottom: 6 }}>Category</div>
                <div style={{ display: 'flex', flexWrap: 'wrap', gap: 6, marginBottom: 18 }}>
                    {categories.map(cat => (
                        <button key={cat.id} onClick={() => set('category_id', cat.id)} style={{
                            padding: '6px 10px', borderRadius: 9999, fontSize: 12, fontWeight: 600, cursor: 'pointer', minHeight: 'unset',
                            border: draft.category_id === cat.id ? '1.5px solid #2563eb' : '1px solid #e2e8f0',
                            background: draft.category_id === cat.id ? '#eff6ff' : '#fff',
                            color: draft.category_id === cat.id ? '#1d4ed8' : '#475569',
                            display: 'flex', alignItems: 'center', gap: 4,
                        }}>
                            {[getCategoryEmoji(cat.name, cat.icon), cat.name].filter(Boolean).join(' ')}
                        </button>
                    ))}
                </div>

                {/* Footer buttons */}
                <div style={{ display: 'flex', gap: 8 }}>
                    <button onClick={onCancel} style={{ flex: 1, padding: 12, borderRadius: 12, background: '#fff', color: '#475569', border: '1px solid #e2e8f0', fontFamily: 'Inter, system-ui, sans-serif', fontSize: 14, fontWeight: 600, cursor: 'pointer', minHeight: 'unset' }}>
                        Cancel
                    </button>
                    <button onClick={onConfirm} style={{ flex: 2, padding: 12, borderRadius: 12, background: '#2563eb', color: '#fff', border: 'none', fontFamily: 'Inter, system-ui, sans-serif', fontSize: 14, fontWeight: 700, cursor: 'pointer', display: 'flex', alignItems: 'center', justifyContent: 'center', gap: 6, boxShadow: '0 4px 10px rgba(37,99,235,.32)', minHeight: 'unset' }}>
                        <Plus size={16} /> Add to List
                    </button>
                </div>
            </div>
        </div>
    );
};

export default ConfirmSheet;
