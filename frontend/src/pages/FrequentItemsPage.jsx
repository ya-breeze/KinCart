import React, { useState, useEffect } from 'react';
import { useToast, getApiError } from '../context/ToastContext';
import { ArrowLeft, RotateCcw } from 'lucide-react';
import { useNavigate } from 'react-router-dom';
import { API_BASE_URL } from '../config';

const FrequentItemsPage = () => {
    const { showToast } = useToast();
    const navigate = useNavigate();
    const [hiddenItems, setHiddenItems] = useState([]);
    const [restoring, setRestoring] = useState(null);

    const fetchHiddenItems = async () => {
        try {
            const resp = await fetch(`${API_BASE_URL}/api/family/frequent-items/hidden`);
            if (resp.ok) setHiddenItems(await resp.json());
            else showToast(await getApiError(resp, 'Failed to load hidden items'));
        } catch {
            showToast('Network error — could not load hidden items');
        }
    };

    useEffect(() => {
        fetchHiddenItems();
    // eslint-disable-next-line react-hooks/exhaustive-deps
    }, []);

    const restoreItem = async (item) => {
        setRestoring(item.id);
        try {
            const resp = await fetch(`${API_BASE_URL}/api/family/frequent-items/${item.id}/restore`, { method: 'PATCH' });
            if (resp.ok) {
                setHiddenItems(prev => prev.filter(i => i.id !== item.id));
                showToast(`"${item.item_name}" restored`, 'success');
            } else {
                showToast(await getApiError(resp, 'Failed to restore item'));
            }
        } catch {
            showToast('Network error — could not restore item');
        } finally {
            setRestoring(null);
        }
    };

    return (
        <div style={{ maxWidth: '700px', margin: '0 auto', padding: '1.5rem 1rem' }}>
            <div style={{ display: 'flex', alignItems: 'center', gap: '0.75rem', marginBottom: '1.5rem' }}>
                <button
                    onClick={() => navigate('/settings')}
                    style={{ background: 'none', border: 'none', cursor: 'pointer', padding: '0.25rem', color: 'var(--text-muted)', display: 'flex' }}
                    aria-label="Back to settings"
                >
                    <ArrowLeft size={22} />
                </button>
                <h1 style={{ fontSize: '1.25rem', fontWeight: 700, margin: 0 }}>Hidden Frequent Items</h1>
            </div>

            <p style={{ fontSize: '0.875rem', color: 'var(--text-muted)', marginBottom: '1.5rem' }}>
                Items you have removed from the quick-add chip grid. Restore any item to make it reappear.
            </p>

            <div className="card">
                {hiddenItems.length === 0 ? (
                    <p style={{ textAlign: 'center', color: 'var(--text-muted)', padding: '2rem 0', margin: 0 }}>
                        No hidden items — your chip grid is fully visible.
                    </p>
                ) : (
                    <ul style={{ listStyle: 'none', margin: 0, padding: 0 }}>
                        {hiddenItems.map((item, idx) => (
                            <li
                                key={item.id}
                                style={{
                                    display: 'flex',
                                    alignItems: 'center',
                                    justifyContent: 'space-between',
                                    padding: '0.75rem 0',
                                    borderTop: idx === 0 ? 'none' : '1px solid var(--border-color)',
                                }}
                            >
                                <div>
                                    <span style={{ fontWeight: 500 }}>{item.item_name}</span>
                                    {item.last_price > 0 && (
                                        <span style={{ fontSize: '0.8125rem', color: 'var(--text-muted)', marginLeft: '0.5rem' }}>
                                            last: {item.last_price}
                                        </span>
                                    )}
                                </div>
                                <button
                                    onClick={() => restoreItem(item)}
                                    disabled={restoring === item.id}
                                    className="btn-secondary"
                                    style={{ display: 'flex', alignItems: 'center', gap: '0.375rem', padding: '0.375rem 0.75rem', fontSize: '0.875rem' }}
                                    aria-label={`Restore ${item.item_name}`}
                                >
                                    <RotateCcw size={14} />
                                    Restore
                                </button>
                            </li>
                        ))}
                    </ul>
                )}
            </div>
        </div>
    );
};

export default FrequentItemsPage;
