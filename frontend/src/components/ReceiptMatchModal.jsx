import React, { useState, useEffect } from 'react';
import { X, Check, ChevronDown, Loader, AlertCircle, Plus, Ban, RotateCcw, Undo2 } from 'lucide-react';
import { API_BASE_URL } from '../config';

// Sentinel: means "this item had no decision before" — distinct from null (= user removed a match)
const UNSET = Symbol('unset');

const ReceiptMatchModal = ({ isOpen, onClose, receiptId, onDone }) => {
    const [data, setData] = useState(null);
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState(null);
    const [busy, setBusy] = useState(false);

    // decisions: receiptItemId → decision object | null | undefined-key (UNSET sentinel in history)
    // undefined key = untouched, confirm-all handles server-side
    // null value     = user removed a match, PATCH {planned_item_id:null} on confirm
    // {action,..}    = user's explicit choice
    const [decisions, setDecisions] = useState({});
    const [initialDecisions, setInitialDecisions] = useState({});
    const [history, setHistory] = useState([]);

    useEffect(() => {
        if (!isOpen || !receiptId) return;
        setLoading(true);
        setError(null);
        fetch(`${API_BASE_URL}/api/receipts/${receiptId}/matches`)
            .then(r => r.ok ? r.json() : r.json().then(d => Promise.reject(d.error || 'Failed to load')))
            .then(d => {
                setData(d);
                const initDec = {};
                d.items.forEach(item => {
                    if (item.match_status === 'auto' || item.match_status === 'confirmed') {
                        initDec[item.receipt_item_id] = {
                            action: 'matched',
                            targetId: item.matched_item?.id,
                            targetName: item.matched_item?.name,
                        };
                    } else if (item.match_status === 'dismissed') {
                        initDec[item.receipt_item_id] = { action: 'skipped' };
                    }
                });
                setDecisions(initDec);
                setInitialDecisions(initDec);
                setHistory([]);
            })
            .catch(err => setError(String(err)))
            .finally(() => setLoading(false));
    }, [isOpen, receiptId]);

    if (!isOpen) return null;

    const makeDecision = (receiptItemId, decision) => {
        const prev = receiptItemId in decisions ? decisions[receiptItemId] : UNSET;
        setHistory(h => [...h, { receiptItemId, prev }]);
        setDecisions(d => ({ ...d, [receiptItemId]: decision }));
    };

    const undoLast = () => {
        if (!history.length) return;
        const { receiptItemId, prev } = history[history.length - 1];
        setHistory(h => h.slice(0, -1));
        setDecisions(d => {
            const next = { ...d };
            if (prev === UNSET) delete next[receiptItemId];
            else next[receiptItemId] = prev;
            return next;
        });
    };

    const resetAll = () => {
        setDecisions({ ...initialDecisions });
        setHistory([]);
    };

    const decisionsChanged = history.length > 0;

    // Pending = items that need a decision before confirm
    const pendingCount = data ? data.items.filter(i => {
        const dec = decisions[i.receipt_item_id];
        // Untouched server-matched items are fine (confirm-all handles them)
        if (dec === undefined && (i.match_status === 'auto' || i.match_status === 'confirmed' || i.match_status === 'dismissed')) return false;
        // null = user removed a match
        if (dec === null) return true;
        // undefined on a non-matched item = needs a decision
        if (dec === undefined) return true;
        return false;
    }).length : 0;

    const newItemCount = data ? data.items.filter(i => decisions[i.receipt_item_id]?.action === 'addnew').length : 0;

    const confirmAll = async () => {
        if (pendingCount > 0) return;
        setBusy(true);
        setError(null);
        try {
            for (const item of data.items) {
                const dec = decisions[item.receipt_item_id];
                if (dec === undefined) continue;
                if (dec !== null && dec.action === 'matched') continue;

                if (dec === null || dec?.action === 'linked' || dec?.action === 'addnew') {
                    const plannedId = (dec !== null && dec.action === 'linked') ? dec.targetId : null;
                    const resp = await fetch(
                        `${API_BASE_URL}/api/receipts/${receiptId}/matches/${item.receipt_item_id}`,
                        {
                            method: 'PATCH',
                            headers: { 'Content-Type': 'application/json' },
                            body: JSON.stringify({ planned_item_id: plannedId }),
                        }
                    );
                    if (!resp.ok) { const d = await resp.json(); throw new Error(d.error || 'Failed'); }
                } else if (dec?.action === 'skipped') {
                    const resp = await fetch(
                        `${API_BASE_URL}/api/receipts/${receiptId}/matches/${item.receipt_item_id}/dismiss`,
                        { method: 'POST' }
                    );
                    if (!resp.ok) { const d = await resp.json(); throw new Error(d.error || 'Failed'); }
                }
            }
            const resp = await fetch(
                `${API_BASE_URL}/api/receipts/${receiptId}/matches/confirm-all`,
                { method: 'POST' }
            );
            if (!resp.ok) { const d = await resp.json(); throw new Error(d.error || 'Failed'); }
            onDone();
            onClose();
        } catch (err) {
            setError(String(err));
        } finally {
            setBusy(false);
        }
    };

    let confirmLabel = 'Confirm & done';
    if (pendingCount > 0) confirmLabel = `${pendingCount} item${pendingCount !== 1 ? 's' : ''} still need a decision`;
    else if (newItemCount > 0) confirmLabel = `Confirm · add ${newItemCount} item${newItemCount !== 1 ? 's' : ''} to list`;

    const matchedItems   = data?.items.filter(i => i.match_status === 'auto' || i.match_status === 'confirmed') ?? [];
    const reviewItems    = data?.items.filter(i => i.match_status === 'unmatched' && !i.is_extra) ?? [];
    const extraItems     = data?.items.filter(i => i.is_extra) ?? [];
    const unboughtItems  = data?.unmatched_planned_items ?? [];
    const alreadyBought  = data?.already_bought_items ?? [];

    return (
        <div style={styles.overlay}>
            <div style={styles.modal} className="card">
                {/* Header */}
                <div style={styles.header}>
                    <div style={{ flex: 1, minWidth: 0 }}>
                        <h2 style={styles.title}>Review Receipt</h2>
                        {data && (
                            <p style={styles.subtitle}>
                                {data.shop_name && `${data.shop_name} · `}{data.date} · {data.total.toFixed(2)}
                            </p>
                        )}
                    </div>
                    <div style={{ display: 'flex', alignItems: 'center', gap: '0.4rem', flexShrink: 0 }}>
                        {history.length > 0 && (
                            <button onClick={undoLast} style={styles.subtleBtn} title="Undo last action">
                                <Undo2 size={14} /> Undo
                            </button>
                        )}
                        {decisionsChanged && (
                            <button onClick={resetAll} style={styles.subtleBtn} title="Reset all decisions">
                                <RotateCcw size={14} /> Reset
                            </button>
                        )}
                        <button onClick={onClose} style={styles.closeBtn}><X size={20} /></button>
                    </div>
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
                            <Section title="Matched" color="#16a34a" count={matchedItems.length}>
                                {matchedItems.map(item => (
                                    <MatchedRow
                                        key={item.receipt_item_id}
                                        item={item}
                                        decision={decisions[item.receipt_item_id]}
                                        onRemoveLink={() => makeDecision(item.receipt_item_id, null)}
                                        onRestore={() => makeDecision(item.receipt_item_id, {
                                            action: 'matched',
                                            targetId: item.matched_item?.id,
                                            targetName: item.matched_item?.name,
                                        })}
                                    />
                                ))}
                            </Section>

                            <Section title="Needs Review" color="#d97706" count={reviewItems.length}>
                                {reviewItems.map(item => (
                                    <DecisionRow
                                        key={item.receipt_item_id}
                                        item={item}
                                        decision={decisions[item.receipt_item_id]}
                                        unmatchedPlanned={unboughtItems}
                                        alreadyBought={alreadyBought}
                                        onDecide={(dec) => makeDecision(item.receipt_item_id, dec)}
                                        color="#d97706"
                                    />
                                ))}
                            </Section>

                            <Section title="Not on your list" color="#2563eb" count={extraItems.length}>
                                {extraItems.map(item => (
                                    <DecisionRow
                                        key={item.receipt_item_id}
                                        item={item}
                                        decision={decisions[item.receipt_item_id]}
                                        unmatchedPlanned={unboughtItems}
                                        alreadyBought={alreadyBought}
                                        onDecide={(dec) => makeDecision(item.receipt_item_id, dec)}
                                        color="#2563eb"
                                    />
                                ))}
                            </Section>

                            <Section title="Still need to buy" color="#6b7280" count={unboughtItems.length}>
                                {unboughtItems.map(item => (
                                    <div key={item.id} style={styles.unboughtRow}>
                                        <span style={{ color: 'var(--text-muted)', fontSize: '0.875rem' }}>{item.name}</span>
                                        <span style={styles.unboughtChip}>Not in receipt</span>
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
                            style={{
                                width: '100%', padding: '0.9rem',
                                display: 'flex', alignItems: 'center', justifyContent: 'center', gap: '0.5rem',
                                opacity: pendingCount > 0 ? 0.6 : 1,
                                cursor: pendingCount > 0 ? 'not-allowed' : 'pointer',
                            }}
                            onClick={confirmAll}
                            disabled={busy}
                            data-testid="confirm-all-btn"
                        >
                            {busy ? <Loader className="spin" size={18} /> : <Check size={18} />}
                            {confirmLabel}
                        </button>
                    </div>
                )}
            </div>
        </div>
    );
};

/* ── Section wrapper ──────────────────────────────────────────── */
const Section = ({ title, color, count, children }) => {
    if (count === 0) return null;
    return (
        <div style={{ marginBottom: '1.5rem' }}>
            <div style={{ display: 'flex', alignItems: 'center', gap: '0.5rem', marginBottom: '0.75rem' }}>
                <span style={{ width: 8, height: 8, borderRadius: '50%', background: color, flexShrink: 0 }} />
                <span style={{ fontWeight: 700, fontSize: '0.82rem', color }}>{title}</span>
                <span style={{ fontSize: '0.72rem', background: '#f3f4f6', color: 'var(--text-muted)', padding: '1px 7px', borderRadius: 99 }}>{count}</span>
            </div>
            <div style={{ display: 'flex', flexDirection: 'column', gap: '0.5rem' }}>
                {children}
            </div>
        </div>
    );
};

/* ── Matched row (expandable to show current link + remove option) ── */
const MatchedRow = ({ item, decision, onRemoveLink, onRestore }) => {
    const [open, setOpen] = useState(false);
    const isRemoved = decision === null;

    return (
        <div style={{ ...styles.row, borderLeft: `3px solid ${isRemoved ? '#d97706' : '#16a34a'}`, flexDirection: 'column', alignItems: 'stretch', gap: 0 }}>
            <div style={{ display: 'flex', alignItems: 'center', gap: '0.5rem', padding: '0.15rem 0' }}
                data-testid={`match-expand-${item.receipt_item_id}`}
                onClick={() => setOpen(o => !o)}>
                <div style={styles.rowNames}>
                    <span style={{ ...styles.receiptName, textDecoration: isRemoved ? 'line-through' : 'none', opacity: isRemoved ? 0.5 : 1 }}>
                        {item.receipt_name}
                    </span>
                    {item.matched_item && !isRemoved && (
                        <span style={styles.arrow}>→ <strong>{item.matched_item.name}</strong></span>
                    )}
                    {isRemoved && <span style={{ fontSize: '0.75rem', color: '#d97706', fontWeight: 600 }}>link removed</span>}
                </div>
                <span style={styles.rowPrice}>{item.price.toFixed(2)}</span>
                <button style={{ ...styles.iconBtn, fontSize: '0.72rem', fontWeight: 600, color: 'var(--text-muted)' }}
                    title={open ? 'Collapse' : 'Change match'}>
                    <ChevronDown size={15} style={{ transform: open ? 'rotate(180deg)' : 'none', transition: '0.2s' }} />
                    <span style={{ marginLeft: 2 }}>Change</span>
                </button>
            </div>
            {open && (
                <div style={{ paddingTop: '0.6rem', borderTop: '1px solid var(--border)', marginTop: '0.5rem', display: 'flex', flexDirection: 'column', gap: '0.4rem' }}>
                    {item.matched_item && !isRemoved && (
                        <div style={{ fontSize: '0.78rem', color: 'var(--text-muted)', padding: '0.3rem 0' }}>
                            Currently linked to <strong style={{ color: 'var(--text)' }}>{item.matched_item.name}</strong>
                        </div>
                    )}
                    {isRemoved ? (
                        <button style={{ ...styles.actionBtn, color: '#16a34a', borderColor: '#16a34a' }}
                            onClick={(e) => { e.stopPropagation(); onRestore(); setOpen(false); }}>
                            <Check size={14} /> Restore link
                        </button>
                    ) : (
                        <button style={{ ...styles.actionBtn, color: 'var(--danger)', borderColor: 'var(--danger)' }}
                            onClick={(e) => { e.stopPropagation(); onRemoveLink(); setOpen(false); }}>
                            <X size={14} /> Remove link
                        </button>
                    )}
                </div>
            )}
        </div>
    );
};

/* ── DecisionRow: used for both "Needs Review" and "Not on your list" ── */
const DecisionRow = ({ item, decision, unmatchedPlanned, alreadyBought, onDecide, color }) => {
    const [open, setOpen] = useState(false);

    const suggestedIds = new Set((item.suggestions ?? []).map(s => s.item_id));
    const otherPlanned = (unmatchedPlanned ?? []).filter(p => !suggestedIds.has(p.id));

    const hasLinked = decision?.action === 'linked';
    const isAddNew  = decision?.action === 'addnew';
    const isSkipped = decision?.action === 'skipped';

    let subline = null;
    if (hasLinked) subline = <span style={styles.arrow}>→ <strong>{decision.targetName}</strong></span>;
    else if (isAddNew) subline = <span style={{ fontSize: '0.72rem', color: '#2563eb', fontWeight: 600 }}>Adding as new item</span>;
    else if (isSkipped) subline = <span style={{ fontSize: '0.72rem', color: 'var(--text-muted)', fontWeight: 600 }}>Skipped</span>;
    else subline = <span style={{ fontSize: '0.72rem', color, fontWeight: 600 }}>Tap to decide</span>;

    return (
        <div style={{ ...styles.row, borderLeft: `3px solid ${color}`, flexDirection: 'column', alignItems: 'stretch', gap: '0.5rem' }}>
            <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}
                onClick={() => setOpen(o => !o)}>
                <div style={styles.rowNames}>
                    <span style={styles.receiptName}>{item.receipt_name}</span>
                    {subline}
                </div>
                <div style={{ display: 'flex', gap: '0.4rem', alignItems: 'center' }}>
                    <span style={styles.rowPrice}>{item.price.toFixed(2)}</span>
                    <button style={styles.iconBtn}
                        data-testid={`expand-${item.receipt_item_id}`}
                        title="Show options">
                        <ChevronDown size={16} style={{ transform: open ? 'rotate(180deg)' : 'none', transition: '0.2s' }} />
                    </button>
                </div>
            </div>

            {open && (
                <div style={{ display: 'flex', flexDirection: 'column', gap: '0.4rem' }}>
                    {/* AI suggestions */}
                    {(item.suggestions ?? []).map(s => (
                        <button key={s.item_id} style={styles.suggestionBtn}
                            data-testid={`suggest-${s.item_id}`}
                            onClick={() => { onDecide({ action: 'linked', targetId: s.item_id, targetName: s.item_name }); setOpen(false); }}>
                            <Check size={14} />
                            {s.item_name}
                            <span style={styles.confidence}>{s.confidence}%</span>
                        </button>
                    ))}

                    {/* Other unmatched planned items */}
                    {otherPlanned.length > 0 && (
                        <>
                            {(item.suggestions ?? []).length > 0 && (
                                <div style={styles.divider}>from your list</div>
                            )}
                            {otherPlanned.map(p => (
                                <button key={p.id} style={styles.plannedBtn}
                                    data-testid={`link-planned-${p.id}`}
                                    onClick={() => { onDecide({ action: 'linked', targetId: p.id, targetName: p.name }); setOpen(false); }}>
                                    <Check size={14} /> {p.name}
                                </button>
                            ))}
                        </>
                    )}

                    {/* Already-bought items */}
                    {(alreadyBought ?? []).length > 0 && (
                        <>
                            <div style={styles.divider}>already bought</div>
                            {(alreadyBought ?? []).map(p => (
                                <button key={p.id} style={styles.plannedBtn}
                                    data-testid={`link-bought-${p.id}`}
                                    onClick={() => { onDecide({ action: 'linked', targetId: p.id, targetName: p.name }); setOpen(false); }}>
                                    <Check size={14} />
                                    {p.name}
                                    <span style={{ ...styles.confidence, background: '#dcfce7', color: '#16a34a' }}>bought</span>
                                </button>
                            ))}
                        </>
                    )}

                    {/* Add new / Skip */}
                    <div style={{ display: 'flex', gap: '0.4rem', marginTop: '0.2rem' }}>
                        <button style={{ ...styles.actionBtn, flex: 1, color: '#2563eb', borderColor: '#2563eb' }}
                            data-testid={`addnew-${item.receipt_item_id}`}
                            onClick={() => { onDecide({ action: 'addnew' }); setOpen(false); }}>
                            <Plus size={14} /> Add as new
                        </button>
                        <button style={{ ...styles.actionBtn, color: 'var(--text-muted)' }}
                            data-testid={`skip-${item.receipt_item_id}`}
                            onClick={() => { onDecide({ action: 'skipped' }); setOpen(false); }}>
                            <Ban size={14} /> Skip
                        </button>
                    </div>
                </div>
            )}
        </div>
    );
};

/* ── Styles ─────────────────────────────────────────────────────── */
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
        flexShrink: 0, gap: '0.75rem',
    },
    title: { fontSize: '1.1rem', fontWeight: 800, margin: 0 },
    subtitle: { fontSize: '0.82rem', color: 'var(--text-muted)', margin: '0.25rem 0 0' },
    closeBtn: { background: 'none', border: 'none', cursor: 'pointer', color: 'var(--text-muted)', padding: '4px', flexShrink: 0 },
    subtleBtn: {
        background: 'none', border: '1px solid var(--border)', borderRadius: '6px',
        cursor: 'pointer', color: 'var(--text-muted)', padding: '4px 8px',
        fontSize: '0.75rem', fontWeight: 600, display: 'flex', alignItems: 'center', gap: '0.25rem',
        flexShrink: 0,
    },
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
        borderRadius: '8px', cursor: 'pointer',
    },
    rowNames: { flex: 1, display: 'flex', flexWrap: 'wrap', gap: '0.3rem', alignItems: 'center', minWidth: 0 },
    receiptName: { fontWeight: 600, fontSize: '0.9rem', whiteSpace: 'nowrap', overflow: 'hidden', textOverflow: 'ellipsis' },
    arrow: { fontSize: '0.82rem', color: 'var(--text-muted)', whiteSpace: 'nowrap' },
    rowPrice: { fontSize: '0.85rem', fontWeight: 600, color: 'var(--text-muted)', flexShrink: 0 },
    iconBtn: { background: 'none', border: 'none', cursor: 'pointer', padding: '4px', color: 'var(--text-muted)', display: 'flex', alignItems: 'center', flexShrink: 0 },
    confidence: { fontSize: '0.72rem', background: '#e5e7eb', borderRadius: 99, padding: '1px 6px', color: '#6b7280', marginLeft: 'auto' },
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
    divider: {
        fontSize: '0.72rem', color: 'var(--text-muted)', textTransform: 'uppercase',
        letterSpacing: '0.05em', padding: '0.15rem 0', textAlign: 'center',
        borderTop: '1px solid var(--border)',
    },
    plannedBtn: {
        display: 'flex', alignItems: 'center', gap: '0.5rem',
        background: '#f5f3ff', border: '1px solid #ddd6fe', borderRadius: '6px',
        padding: '0.4rem 0.75rem', cursor: 'pointer', fontSize: '0.875rem',
        color: '#7c3aed', fontWeight: 600, textAlign: 'left',
    },
    unboughtRow: {
        display: 'flex', justifyContent: 'space-between', alignItems: 'center',
        padding: '0.5rem 0.75rem', background: 'var(--bg-secondary)', borderRadius: '8px',
        borderLeft: '3px solid #d1d5db',
    },
    unboughtChip: {
        fontSize: '0.68rem', fontWeight: 600, padding: '0.15rem 0.5rem',
        borderRadius: 99, background: '#f3f4f6', color: 'var(--text-muted)',
        border: '1px solid var(--border)',
    },
};

export default ReceiptMatchModal;
