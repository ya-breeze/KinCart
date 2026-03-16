import React, { useState, useEffect } from 'react';
import { X, Check, ChevronDown, Loader, AlertCircle, Plus, Ban } from 'lucide-react';
import { API_BASE_URL } from '../config';

/**
 * ReceiptMatchModal — shown after receipt upload when status is "pending_review".
 * Lets the user confirm/change AI item matches, handle extras and unbought items.
 */
const ReceiptMatchModal = ({ isOpen, onClose, receiptId, token, onDone }) => {
    const [data, setData] = useState(null);
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState(null);
    const [busy, setBusy] = useState({}); // receiptItemId → true while calling API

    useEffect(() => {
        if (!isOpen || !receiptId) return;
        setLoading(true);
        setError(null);
        fetch(`${API_BASE_URL}/api/receipts/${receiptId}/matches`, {
            headers: { Authorization: `Bearer ${token}` },
        })
            .then(r => r.ok ? r.json() : r.json().then(d => Promise.reject(d.error || 'Failed to load')))
            .then(setData)
            .catch(err => setError(String(err)))
            .finally(() => setLoading(false));
    }, [isOpen, receiptId, token]);

    if (!isOpen) return null;

    const reload = () => {
        setLoading(true);
        fetch(`${API_BASE_URL}/api/receipts/${receiptId}/matches`, {
            headers: { Authorization: `Bearer ${token}` },
        })
            .then(r => r.ok ? r.json() : r.json().then(d => Promise.reject(d.error || 'Failed to load')))
            .then(setData)
            .catch(err => setError(String(err)))
            .finally(() => setLoading(false));
    };

    const confirmMatch = async (receiptItemId, plannedItemId) => {
        setBusy(b => ({ ...b, [receiptItemId]: true }));
        try {
            const resp = await fetch(
                `${API_BASE_URL}/api/receipts/${receiptId}/matches/${receiptItemId}`,
                {
                    method: 'PATCH',
                    headers: {
                        Authorization: `Bearer ${token}`,
                        'Content-Type': 'application/json',
                    },
                    body: JSON.stringify({ planned_item_id: plannedItemId ?? null }),
                }
            );
            if (!resp.ok) {
                const d = await resp.json();
                throw new Error(d.error || 'Failed');
            }
            reload();
        } catch (err) {
            setError(String(err));
        } finally {
            setBusy(b => ({ ...b, [receiptItemId]: false }));
        }
    };

    const dismiss = async (receiptItemId) => {
        setBusy(b => ({ ...b, [receiptItemId]: true }));
        try {
            const resp = await fetch(
                `${API_BASE_URL}/api/receipts/${receiptId}/matches/${receiptItemId}/dismiss`,
                {
                    method: 'POST',
                    headers: { Authorization: `Bearer ${token}` },
                }
            );
            if (!resp.ok) {
                const d = await resp.json();
                throw new Error(d.error || 'Failed');
            }
            reload();
        } catch (err) {
            setError(String(err));
        } finally {
            setBusy(b => ({ ...b, [receiptItemId]: false }));
        }
    };

    const confirmAll = async () => {
        setBusy(b => ({ ...b, _all: true }));
        try {
            const resp = await fetch(
                `${API_BASE_URL}/api/receipts/${receiptId}/matches/confirm-all`,
                {
                    method: 'POST',
                    headers: { Authorization: `Bearer ${token}` },
                }
            );
            if (!resp.ok) {
                const d = await resp.json();
                throw new Error(d.error || 'Failed');
            }
            onDone();
            onClose();
        } catch (err) {
            setError(String(err));
        } finally {
            setBusy(b => ({ ...b, _all: false }));
        }
    };

    const pendingCount = data
        ? data.items.filter(i => i.match_status === 'unmatched').length
        : 0;

    return (
        <div style={styles.overlay}>
            <div style={styles.modal} className="card">
                {/* Header */}
                <div style={styles.header}>
                    <div>
                        <h2 style={styles.title}>Review Receipt Matches</h2>
                        {data && (
                            <p style={styles.subtitle}>
                                {data.shop_name && `${data.shop_name} · `}{data.date} · {data.total.toFixed(2)}
                            </p>
                        )}
                    </div>
                    <button onClick={onClose} style={styles.closeBtn}><X size={22} /></button>
                </div>

                {/* Body */}
                <div style={styles.body}>
                    {loading && (
                        <div style={styles.center}><Loader className="spin" size={28} color="var(--primary)" /></div>
                    )}
                    {error && !loading && (
                        <div style={styles.errorBox}><AlertCircle size={16} /> {error}</div>
                    )}

                    {data && !loading && (
                        <>
                            {/* Section 1: Auto-matched */}
                            <Section title="Matched" color="#16a34a" count={data.items.filter(i => i.match_status === 'auto' || i.match_status === 'confirmed').length}>
                                {data.items
                                    .filter(i => i.match_status === 'auto' || i.match_status === 'confirmed')
                                    .map(item => (
                                        <MatchedRow
                                            key={item.receipt_item_id}
                                            item={item}
                                            busy={busy[item.receipt_item_id]}
                                            onUnmatch={() => confirmMatch(item.receipt_item_id, null)}
                                        />
                                    ))}
                            </Section>

                            {/* Section 2: Needs review (has suggestions but unmatched) */}
                            <Section title="Needs Review" color="#d97706" count={data.items.filter(i => i.match_status === 'unmatched' && !i.is_extra).length}>
                                {data.items
                                    .filter(i => i.match_status === 'unmatched' && !i.is_extra)
                                    .map(item => (
                                        <UnmatchedRow
                                            key={item.receipt_item_id}
                                            item={item}
                                            busy={busy[item.receipt_item_id]}
                                            onConfirm={(id) => confirmMatch(item.receipt_item_id, id)}
                                            onAddNew={() => confirmMatch(item.receipt_item_id, null)}
                                            onDismiss={() => dismiss(item.receipt_item_id)}
                                        />
                                    ))}
                            </Section>

                            {/* Section 3: Extra items (not on list) */}
                            <Section title="Not on your list" color="#2563eb" count={data.items.filter(i => i.is_extra).length}>
                                {data.items
                                    .filter(i => i.is_extra)
                                    .map(item => (
                                        <ExtraRow
                                            key={item.receipt_item_id}
                                            item={item}
                                            busy={busy[item.receipt_item_id]}
                                            onAddNew={() => confirmMatch(item.receipt_item_id, null)}
                                            onDismiss={() => dismiss(item.receipt_item_id)}
                                        />
                                    ))}
                            </Section>

                            {/* Section 4: Unbought planned items */}
                            <Section title="Still need to buy" color="#6b7280" count={data.unmatched_planned_items.length}>
                                {data.unmatched_planned_items.map(item => (
                                    <div key={item.id} style={styles.unmatchedPlanned}>
                                        <span style={{ color: 'var(--text-muted)' }}>{item.name}</span>
                                        <span style={{ fontSize: '0.75rem', color: 'var(--text-muted)', background: '#f3f4f6', padding: '2px 8px', borderRadius: 99 }}>not bought</span>
                                    </div>
                                ))}
                            </Section>
                        </>
                    )}
                </div>

                {/* Footer */}
                {data && !loading && (
                    <div style={styles.footer}>
                        {error && <div style={{ ...styles.errorBox, marginBottom: '0.5rem' }}><AlertCircle size={14} /> {error}</div>}
                        <button
                            className="primary-btn"
                            style={{ width: '100%', padding: '0.9rem', display: 'flex', alignItems: 'center', justifyContent: 'center', gap: '0.5rem' }}
                            onClick={confirmAll}
                            disabled={busy._all}
                        >
                            {busy._all ? <Loader className="spin" size={18} /> : <Check size={18} />}
                            {pendingCount > 0
                                ? `Confirm All (add ${pendingCount} item${pendingCount > 1 ? 's' : ''} to list)`
                                : 'Confirm All'}
                        </button>
                    </div>
                )}
            </div>
        </div>
    );
};

/* Sub-components */

const Section = ({ title, color, count, children }) => {
    if (count === 0) return null;
    return (
        <div style={{ marginBottom: '1.5rem' }}>
            <div style={{ display: 'flex', alignItems: 'center', gap: '0.5rem', marginBottom: '0.75rem' }}>
                <span style={{ width: 10, height: 10, borderRadius: '50%', background: color, flexShrink: 0 }} />
                <span style={{ fontWeight: 700, fontSize: '0.85rem', color }}>{title}</span>
                <span style={{ fontSize: '0.75rem', background: '#f3f4f6', color: 'var(--text-muted)', padding: '1px 7px', borderRadius: 99 }}>{count}</span>
            </div>
            <div style={{ display: 'flex', flexDirection: 'column', gap: '0.5rem' }}>
                {children}
            </div>
        </div>
    );
};

const MatchedRow = ({ item, busy, onUnmatch }) => (
    <div style={{ ...styles.row, borderLeft: '3px solid #16a34a' }}>
        <div style={styles.rowNames}>
            <span style={styles.receiptName}>{item.receipt_name}</span>
            {item.matched_item && (
                <span style={styles.arrow}>→ <strong>{item.matched_item.name}</strong></span>
            )}
            {item.confidence > 0 && item.confidence < 100 && (
                <span style={styles.confidence}>{item.confidence}%</span>
            )}
        </div>
        <div style={styles.rowPrice}>{item.price.toFixed(2)}</div>
        <button onClick={onUnmatch} disabled={busy} style={styles.iconBtn} title="Change match">
            {busy ? <Loader className="spin" size={14} /> : <ChevronDown size={16} />}
        </button>
    </div>
);

const UnmatchedRow = ({ item, busy, onConfirm, onAddNew, onDismiss }) => {
    const [open, setOpen] = useState(false);
    return (
        <div style={{ ...styles.row, borderLeft: '3px solid #d97706', flexDirection: 'column', alignItems: 'stretch', gap: '0.5rem' }}>
            <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                <span style={styles.receiptName}>{item.receipt_name}</span>
                <div style={{ display: 'flex', gap: '0.4rem', alignItems: 'center' }}>
                    <span style={styles.rowPrice}>{item.price.toFixed(2)}</span>
                    <button onClick={() => setOpen(o => !o)} style={styles.iconBtn} title="Show suggestions">
                        <ChevronDown size={16} style={{ transform: open ? 'rotate(180deg)' : 'none', transition: '0.2s' }} />
                    </button>
                </div>
            </div>
            {open && (
                <div style={{ display: 'flex', flexDirection: 'column', gap: '0.4rem' }}>
                    {item.suggestions.map(s => (
                        <button
                            key={s.item_id}
                            disabled={busy}
                            onClick={() => onConfirm(s.item_id)}
                            style={styles.suggestionBtn}
                        >
                            {busy ? <Loader className="spin" size={12} /> : <Check size={14} />}
                            {s.item_name}
                            <span style={styles.confidence}>{s.confidence}%</span>
                        </button>
                    ))}
                    <div style={{ display: 'flex', gap: '0.4rem' }}>
                        <button disabled={busy} onClick={onAddNew} style={{ ...styles.actionBtn, flex: 1, color: '#2563eb', borderColor: '#2563eb' }}>
                            <Plus size={14} /> Add as new
                        </button>
                        <button disabled={busy} onClick={onDismiss} style={{ ...styles.actionBtn, color: 'var(--text-muted)' }}>
                            <Ban size={14} /> Dismiss
                        </button>
                    </div>
                </div>
            )}
        </div>
    );
};

const ExtraRow = ({ item, busy, onAddNew, onDismiss }) => (
    <div style={{ ...styles.row, borderLeft: '3px solid #2563eb' }}>
        <div style={styles.rowNames}>
            <span style={styles.receiptName}>{item.receipt_name}</span>
        </div>
        <div style={{ display: 'flex', gap: '0.4rem', alignItems: 'center' }}>
            <span style={styles.rowPrice}>{item.price.toFixed(2)}</span>
            <button disabled={busy} onClick={onAddNew} style={{ ...styles.iconBtn, color: '#2563eb' }} title="Add to list">
                {busy ? <Loader className="spin" size={14} /> : <Plus size={16} />}
            </button>
            <button disabled={busy} onClick={onDismiss} style={{ ...styles.iconBtn, color: 'var(--text-muted)' }} title="Dismiss">
                <Ban size={16} />
            </button>
        </div>
    </div>
);

/* Styles */
const styles = {
    overlay: {
        position: 'fixed', top: 0, left: 0, right: 0, bottom: 0,
        background: 'rgba(0,0,0,0.55)', display: 'flex',
        alignItems: 'center', justifyContent: 'center',
        zIndex: 1100, padding: '1rem',
    },
    modal: {
        background: 'white', width: '100%', maxWidth: '520px',
        maxHeight: '90vh', display: 'flex', flexDirection: 'column',
        borderRadius: '16px', overflow: 'hidden',
        animation: 'fadeIn 0.2s ease-out',
    },
    header: {
        display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start',
        padding: '1.25rem 1.5rem', borderBottom: '1px solid var(--border)',
        flexShrink: 0,
    },
    title: { fontSize: '1.1rem', fontWeight: 800, margin: 0 },
    subtitle: { fontSize: '0.82rem', color: 'var(--text-muted)', margin: '0.25rem 0 0' },
    closeBtn: { background: 'none', border: 'none', cursor: 'pointer', color: 'var(--text-muted)', padding: '4px' },
    body: { overflowY: 'auto', padding: '1.25rem 1.5rem', flex: 1 },
    footer: { padding: '1rem 1.5rem', borderTop: '1px solid var(--border)', flexShrink: 0 },
    center: { display: 'flex', justifyContent: 'center', padding: '3rem 0' },
    errorBox: {
        background: '#fee2e2', color: 'var(--danger)', borderRadius: '8px',
        padding: '0.6rem 0.9rem', fontSize: '0.875rem',
        display: 'flex', alignItems: 'center', gap: '0.4rem',
    },
    row: {
        display: 'flex', alignItems: 'center', gap: '0.5rem',
        padding: '0.6rem 0.75rem', background: 'var(--bg-secondary)',
        borderRadius: '8px',
    },
    rowNames: { flex: 1, display: 'flex', flexWrap: 'wrap', gap: '0.3rem', alignItems: 'center', minWidth: 0 },
    receiptName: { fontWeight: 600, fontSize: '0.9rem', whiteSpace: 'nowrap', overflow: 'hidden', textOverflow: 'ellipsis' },
    arrow: { fontSize: '0.82rem', color: 'var(--text-muted)', whiteSpace: 'nowrap' },
    confidence: { fontSize: '0.72rem', background: '#e5e7eb', borderRadius: 99, padding: '1px 6px', color: '#6b7280', marginLeft: 'auto' },
    rowPrice: { fontSize: '0.85rem', fontWeight: 600, color: 'var(--text-muted)', flexShrink: 0 },
    iconBtn: { background: 'none', border: 'none', cursor: 'pointer', padding: '4px', color: 'var(--text-muted)', display: 'flex', alignItems: 'center', flexShrink: 0 },
    suggestionBtn: {
        display: 'flex', alignItems: 'center', gap: '0.5rem',
        background: '#f0fdf4', border: '1px solid #bbf7d0', borderRadius: '6px',
        padding: '0.4rem 0.75rem', cursor: 'pointer', fontSize: '0.875rem',
        color: '#16a34a', fontWeight: 600, textAlign: 'left',
    },
    actionBtn: {
        display: 'flex', alignItems: 'center', justifyContent: 'center', gap: '0.35rem',
        background: 'none', border: '1px solid var(--border)', borderRadius: '6px',
        padding: '0.35rem 0.7rem', cursor: 'pointer', fontSize: '0.8rem',
    },
    unmatchedPlanned: {
        display: 'flex', justifyContent: 'space-between', alignItems: 'center',
        padding: '0.5rem 0.75rem', background: 'var(--bg-secondary)', borderRadius: '8px',
        borderLeft: '3px solid #d1d5db',
    },
};

export default ReceiptMatchModal;
