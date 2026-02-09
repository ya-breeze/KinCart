import React, { memo } from 'react';
import { Store, Calendar, Plus, X, Loader2, Tag, ShoppingCart } from 'lucide-react';
import { API_BASE_URL } from '../config';
import LazyImage from './LazyImage';

const FlyerItemCard = memo(({
    item,
    currency,
    showListSelector,
    onToggleListSelector,
    onImagePreview,
    onCategorySearch,
    addForm,
    onAddFormChange,
    categories,
    activeLists,
    addingTo,
    onAddToList,
    onCreateAndAdd
}) => {
    return (
        <div className="card" style={{ padding: 0, overflow: 'hidden', display: 'flex', flexDirection: 'column', position: 'relative' }}>
            <div style={{ height: '200px', background: '#f8f9fa', position: 'relative', overflow: 'hidden' }}>
                {item.local_photo_path ? (
                    <LazyImage
                        src={`${API_BASE_URL}${item.local_photo_path}`}
                        alt={item.name}
                        onClick={() => onImagePreview({ src: `${API_BASE_URL}${item.local_photo_path}`, alt: item.name })}
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
                                onClick={() => onCategorySearch(cat.trim())}
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
                                onClick={() => onCategorySearch(kw.trim())}
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
                        onClick={() => onToggleListSelector(item.id)}
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
                                <button onClick={() => onToggleListSelector(null)} style={{ background: 'none', border: 'none', cursor: 'pointer', color: 'var(--text-muted)' }}>
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
                                            onChange={e => onAddFormChange({ ...addForm, category_id: e.target.value })}
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
                                            onChange={e => onAddFormChange({ ...addForm, quantity: e.target.value })}
                                            style={{ width: '100%', padding: '0.5rem', borderRadius: '8px', border: '1px solid var(--border)', fontSize: '0.875rem' }}
                                        />
                                    </div>
                                    <div style={{ flex: 1, display: 'flex', flexDirection: 'column', gap: '0.25rem' }}>
                                        <label style={{ fontSize: '0.7rem', fontWeight: 800, color: 'var(--text-muted)', textTransform: 'uppercase' }}>Unit</label>
                                        <select
                                            value={addForm.unit}
                                            onChange={e => onAddFormChange({ ...addForm, unit: e.target.value })}
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
                                            onClick={() => onAddToList(item, list.id)}
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
                                        onClick={() => onCreateAndAdd(item)}
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
    );
}, (prevProps, nextProps) => {
    // Custom comparison function for optimization
    return (
        prevProps.item.id === nextProps.item.id &&
        prevProps.showListSelector === nextProps.showListSelector &&
        prevProps.addingTo === nextProps.addingTo &&
        prevProps.currency === nextProps.currency &&
        prevProps.addForm.quantity === nextProps.addForm.quantity &&
        prevProps.addForm.unit === nextProps.addForm.unit &&
        prevProps.addForm.category_id === nextProps.addForm.category_id
    );
});

FlyerItemCard.displayName = 'FlyerItemCard';

export default FlyerItemCard;
