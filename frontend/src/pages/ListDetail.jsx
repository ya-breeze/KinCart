import React, { useState, useEffect, useRef, useMemo } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import { useAuth } from '../context/AuthContext';
import { useToast, getApiError } from '../context/ToastContext';
import { ArrowLeft, Check, Trash2, Plus, AlertCircle, ShoppingCart, Store, Edit2, X, Receipt, Upload, FileText, Link2, ChevronDown, Mic, Search, List } from 'lucide-react';
import { API_BASE_URL } from '../config';
import ImageModal from '../components/ImageModal';
import ReceiptUploadModal from '../components/ReceiptUploadModal';
import ReceiptViewerModal from '../components/ReceiptViewerModal';
import Modal from '../components/Modal';
import ConfirmSheet from '../components/ConfirmSheet';
import PasteItemsPanel from '../components/PasteItemsPanel';
import { getCategoryEmoji } from '../utils/categoryEmoji';

// ─── Status badge config ──────────────────────────────────────────────────────
const STATUS_STYLE = {
    'preparing':           { bg: '#dbeafe', color: '#1d4ed8', label: 'PREPARING' },
    'ready for shopping':  { bg: '#fef3c7', color: '#92400e', label: 'READY' },
    'completed':           { bg: '#dcfce7', color: '#166534', label: 'COMPLETED' },
};
const STATUS_CYCLE = ['preparing', 'ready for shopping', 'completed'];

// ─── ListDetail ───────────────────────────────────────────────────────────────
const ListDetail = () => {
    const { id } = useParams();
    const navigate = useNavigate();
    const { mode, currency } = useAuth();
    const { showToast } = useToast();

    // ── core data ──
    const [list, setList] = useState(null);
    const [categories, setCategories] = useState([]);
    const [shops, setShops] = useState([]);
    const [selectedShopId, setSelectedShopId] = useState('');
    const [shopOrder, setShopOrder] = useState([]);
    const [frequentItems, setFrequentItems] = useState([]);
    const [aliases, setAliases] = useState([]);

    // ── autocomplete suggestions (manager quick-add) ──
    const [suggestions, setSuggestions] = useState([]);
    const [showSuggestions, setShowSuggestions] = useState(false);

    // ── modals / overlays ──
    const [selectedPhoto2, setSelectedPhoto2] = useState(null);  // for image modal preview
    const [editingItemId, setEditingItemId] = useState(null);
    const [editItemData, setEditItemData] = useState({});
    const [isRenaming, setIsRenaming] = useState(false);
    const [renameValue, setRenameValue] = useState('');
    const [isReceiptModalOpen, setIsReceiptModalOpen] = useState(false);
    const [isReceiptViewerOpen, setIsReceiptViewerOpen] = useState(false);
    const [isDeleteModalOpen, setIsDeleteModalOpen] = useState(false);
    const [itemToDelete, setItemToDelete] = useState(null);
    const [linkAliasItem, setLinkAliasItem] = useState(null);
    const [linkAliasTarget, setLinkAliasTarget] = useState('');
    const [linkAliasInput, setLinkAliasInput] = useState('');
    const [linkAliasSelected, setLinkAliasSelected] = useState(null);
    const [linking, setLinking] = useState(false);
    const [chipsExpanded, setChipsExpanded] = useState(false);
    const [chipsOverflow, setChipsOverflow] = useState(false);
    const chipsContainerRef = useRef(null);

    // ── manager quick-add ──
    const [query, setQuery] = useState('');
    const [draft, setDraft] = useState(null);
    const [showPasteModal, setShowPasteModal] = useState(false);
    const isListMode = query.includes(',');
    const [justAddedId, setJustAddedId] = useState(null);

    // ── manager inline row expansion ──
    const [expandedId, setExpandedId] = useState(null);
    const [doneExpanded, setDoneExpanded] = useState(false);  // shopper "done" section, collapsed by default
    const [expandedEdits, setExpandedEdits] = useState({});

    const queryDebounceRef = useRef(null);
    const queryInputRef = useRef(null);

    useEffect(() => {
        fetchList();
        fetchCategories();
        fetchShops();
        fetchFrequentItems();
        fetchAliases();
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [id]);

    useEffect(() => { setChipsExpanded(false); setChipsOverflow(false); setDoneExpanded(false); }, [id]);

    useEffect(() => {
        const el = chipsContainerRef.current;
        if (!el) return;
        let mounted = true;
        const check = () => { if (mounted) setChipsOverflow(el.scrollHeight > 64); };
        const ro = new ResizeObserver(check);
        check();
        ro.observe(el);
        return () => { mounted = false; ro.disconnect(); };
    }, [frequentItems, list?.id]);

    // ── fetch helpers ──────────────────────────────────────────────────────────

    const fetchShops = async () => {
        const resp = await fetch(`${API_BASE_URL}/api/shops`);
        if (resp.ok) setShops(await resp.json());
    };

    const fetchShopOrder = async (shopId) => {
        if (!shopId) { setShopOrder([]); return; }
        const resp = await fetch(`${API_BASE_URL}/api/shops/${shopId}/order`);
        if (resp.ok) setShopOrder(await resp.json());
    };

    const handleShopChange = (e) => {
        const shopId = e.target.value;
        setSelectedShopId(shopId);
        fetchShopOrder(shopId);
    };

    const fetchFrequentItems = async () => {
        const resp = await fetch(`${API_BASE_URL}/api/family/frequent-items`);
        if (resp.ok) setFrequentItems(await resp.json());
    };

    const deleteFrequentItem = async (e, id) => {
        e.stopPropagation();
        try {
            const resp = await fetch(`${API_BASE_URL}/api/family/frequent-items/${id}`, { method: 'DELETE' });
            if (resp.ok) setFrequentItems(prev => prev.filter(fi => fi.id !== id));
            else showToast(await getApiError(resp, 'Failed to delete frequent item'));
        } catch {
            showToast('Network error — could not delete frequent item');
        }
    };

    const fetchCategories = async () => {
        const resp = await fetch(`${API_BASE_URL}/api/categories`);
        if (resp.ok) setCategories(await resp.json());
    };

    const fetchList = async () => {
        try {
            const resp = await fetch(`${API_BASE_URL}/api/lists/${id}`);
            if (resp.ok) setList(await resp.json());
            else showToast(await getApiError(resp, 'Failed to load list'));
        } catch { showToast('Network error — could not load list'); }
    };

    const fetchAliases = async () => {
        try {
            const resp = await fetch(`${API_BASE_URL}/api/family/aliases`);
            if (resp.ok) setAliases(await resp.json());
        } catch { /* non-critical */ }
    };

    const fetchSuggestions = async (q) => {
        if (q.length < 2) { setSuggestions([]); setShowSuggestions(false); return; }
        const resp = await fetch(`${API_BASE_URL}/api/family/item-suggestions?q=${encodeURIComponent(q)}`);
        if (resp.ok) {
            const data = await resp.json();
            setSuggestions(data);
            setShowSuggestions(data.length > 0);
        }
    };

    // ── derived lookups ────────────────────────────────────────────────────────

    const isManager = mode === 'manager';
    const isShopper = mode === 'shopper';

    const aliasesByPlanned = useMemo(() => {
        const m = {};
        for (const a of aliases) {
            const k = a.planned_name.toLowerCase();
            if (!m[k]) m[k] = [];
            m[k].push(a);
        }
        return m;
    }, [aliases]);

    const aliasesByReceipt = useMemo(() => {
        const m = {};
        for (const a of aliases) {
            const k = a.receipt_name.toLowerCase();
            if (!m[k]) m[k] = [];
            m[k].push(a);
        }
        return m;
    }, [aliases]);

    // ── mutations ──────────────────────────────────────────────────────────────

    // Manager quick-add: open draft from a frequent chip or typed name
    const openDraftNew = (rawName) => {
        const name = (rawName ?? query).trim();
        if (!name) return;
        const variants = aliasesByPlanned[name.toLowerCase()] || [];
        const firstVariant = variants[0] || null;
        setDraft({
            name,
            qty: '1',
            unit: 'pcs',
            price: firstVariant?.last_price ? String(firstVariant.last_price) : '',
            category_id: firstVariant ? '' : (categories[0]?.id || ''),
            source: variants.length > 0 ? 'history' : 'new',
            emoji: '📦',
            variants,
            variant: firstVariant,
        });
        setQuery('');
        setShowSuggestions(false);
    };

    // Manager quick-add: open draft from an autocomplete suggestion row
    const openDraftFromSuggestion = (suggestion) => {
        const variants = aliasesByPlanned[suggestion.planned_name.toLowerCase()] || [];
        const firstVariant = variants[0] || null;
        setDraft({
            name: suggestion.planned_name,
            qty: '1',
            unit: 'pcs',
            price: firstVariant?.last_price ? String(firstVariant.last_price) : '',
            category_id: '',
            source: 'history',
            emoji: '📦',
            variants,
            variant: firstVariant,
        });
        setQuery('');
        setShowSuggestions(false);
    };

    const addFromDraft = async () => {
        if (!draft?.name) return;
        try {
            const resp = await fetch(`${API_BASE_URL}/api/lists/${id}/items`, {
                method: 'POST',
                body: JSON.stringify({
                    name: draft.name,
                    category_id: draft.category_id || undefined,
                    price: parseFloat(draft.price) || 0,
                    quantity: parseFloat(draft.qty) || 1,
                    unit: draft.unit || 'pcs',
                    preferred_alias_id: draft.variant?.id || null,
                })
            });
            if (resp.ok) {
                const addedItem = await resp.json();
                setJustAddedId(addedItem.id);
                setDraft(null);
                await fetchList();
                fetchFrequentItems();
                setTimeout(() => setJustAddedId(null), 1200);
            } else {
                showToast(await getApiError(resp, 'Failed to add item'));
            }
        } catch { showToast('Network error — could not add item'); }
    };

    // Inline row edit: toggle expand/collapse; save on collapse
    const toggleExpand = async (item) => {
        if (expandedId === item.id) {
            const edits = expandedEdits[item.id];
            if (edits) {
                const changed =
                    String(edits.quantity) !== String(item.quantity || 1) ||
                    edits.unit !== (item.unit || 'pcs') ||
                    String(edits.price) !== String(item.price || 0) ||
                    edits.category_id !== (item.category_id || '');
                if (changed) await saveInlineEdit(item.id, edits);
            }
            setExpandedId(null);
        } else {
            setExpandedId(item.id);
            setExpandedEdits(prev => ({
                ...prev,
                [item.id]: {
                    quantity: item.quantity || 1,
                    unit: item.unit || 'pcs',
                    price: item.price || 0,
                    category_id: item.category_id || '',
                },
            }));
        }
    };

    const saveInlineEdit = async (itemId, edits) => {
        try {
            const resp = await fetch(`${API_BASE_URL}/api/items/${itemId}`, {
                method: 'PATCH',
                body: JSON.stringify({
                    quantity: parseFloat(edits.quantity) || 1,
                    unit: edits.unit,
                    price: parseFloat(edits.price) || 0,
                    category_id: edits.category_id || undefined,
                })
            });
            if (resp.ok) fetchList();
            else showToast(await getApiError(resp, 'Failed to save'));
        } catch { showToast('Network error'); }
    };

    const toggleItem = async (item) => {
        try {
            const resp = await fetch(`${API_BASE_URL}/api/items/${item.id}`, { method: 'PATCH', body: JSON.stringify({ is_bought: !item.is_bought }) });
            if (resp.ok) fetchList();
            else showToast(await getApiError(resp, 'Failed to update item'));
        } catch { showToast('Network error — could not update item'); }
    };

    const toggleAbsent = async (item) => {
        try {
            const resp = await fetch(`${API_BASE_URL}/api/items/${item.id}`, { method: 'PATCH', body: JSON.stringify({ is_absent: !item.is_absent }) });
            if (resp.ok) fetchList();
            // The backend rejects marking a bought item absent with a 400.
            // The control is hidden for bought items, so this only fires on a
            // stale view; refetching resyncs it.
            else showToast(await getApiError(resp, 'Failed to update item'));
        } catch { showToast('Network error — could not update item'); }
    };

    const deleteItem = (itemId) => {
        setItemToDelete(list.items.find(i => i.id === itemId));
    };

    const confirmDeleteItem = async () => {
        if (!itemToDelete) return;
        try {
            const resp = await fetch(`${API_BASE_URL}/api/items/${itemToDelete.id}`, { method: 'DELETE' });
            if (resp.ok) { fetchList(); setItemToDelete(null); setExpandedId(null); }
            else showToast(await getApiError(resp, 'Failed to delete item'));
        } catch { showToast('Network error — could not delete item'); }
    };

    const updateStatus = async (status) => {
        try {
            const resp = await fetch(`${API_BASE_URL}/api/lists/${id}`, { method: 'PATCH', body: JSON.stringify({ status }) });
            if (resp.ok) fetchList();
            else showToast(await getApiError(resp, 'Failed to update list status'));
        } catch { showToast('Network error — could not update status'); }
    };

    const cycleStatus = () => {
        const idx = STATUS_CYCLE.indexOf(list.status);
        const next = STATUS_CYCLE[(idx + 1) % STATUS_CYCLE.length];
        updateStatus(next);
    };

    const startEditing = (item) => {
        // Merge any in-progress inline edits into the modal's initial state, then
        // clear them so collapsing the row after the modal saves is a no-op.
        const pending = expandedEdits[item.id] || {};
        setExpandedEdits(prev => { const next = { ...prev }; delete next[item.id]; return next; });
        setEditingItemId(item.id);
        setEditItemData({
            ...item,
            quantity:    pending.quantity    ?? item.quantity    ?? 1,
            unit:        pending.unit        ?? item.unit        ?? 'pcs',
            price:       pending.price       ?? item.price       ?? 0,
            category_id: pending.category_id ?? item.category_id,
        });
    };

    const saveEdit = async () => {
        try {
            const resp = await fetch(`${API_BASE_URL}/api/items/${editingItemId}`, {
                method: 'PATCH',
                body: JSON.stringify({ ...editItemData, price: parseFloat(editItemData.price) || 0, quantity: parseFloat(editItemData.quantity) || 1, category_id: editItemData.category_id || undefined })
            });
            if (resp.ok) { setEditingItemId(null); fetchList(); }
            else showToast(await getApiError(resp, 'Failed to save item'));
        } catch { showToast('Network error — could not save item'); }
    };

    const handleLinkAlias = async () => {
        if (linking) return;
        const isReceiptItem = !!linkAliasItem.receipt_item_id;
        let body;
        if (isReceiptItem) {
            const trimmed = linkAliasInput.trim();
            let resolved = linkAliasSelected;
            if (!resolved && trimmed !== '') {
                const exact = (list.items || []).find(i => !i.receipt_item_id && i.id !== linkAliasItem.id && i.name.trim().toLowerCase() === trimmed.toLowerCase());
                if (exact) resolved = { id: exact.id, name: exact.name };
            }
            body = resolved ? { receipt_item_id: linkAliasItem.id, planned_item_id: resolved.id } : { receipt_item_id: linkAliasItem.id, planned_name: trimmed };
        } else {
            body = { receipt_item_id: linkAliasTarget, planned_item_id: linkAliasItem.id };
        }
        setLinking(true);
        try {
            const resp = await fetch(`${API_BASE_URL}/api/items/link-alias`, { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(body) });
            if (resp.ok) {
                showToast(body.planned_item_id ? 'Alias created, planned item removed' : 'Alias created', 'success');
                setLinkAliasItem(null);
                fetchList();
                fetchAliases();
            } else { showToast(await getApiError(resp, 'Failed to link alias')); }
        } catch { showToast('Network error — could not link alias'); }
        finally { setLinking(false); }
    };

    const handleRemoveAlias = async (aliasId) => {
        setLinking(true);
        try {
            const resp = await fetch(`${API_BASE_URL}/api/family/aliases/${aliasId}`, { method: 'DELETE' });
            if (resp.ok) { showToast('Alias removed', 'success'); setLinkAliasItem(null); fetchAliases(); }
            else showToast(await getApiError(resp, 'Failed to remove alias'));
        } catch { showToast('Network error — could not remove alias'); }
        finally { setLinking(false); }
    };

    const handleRenameList = async () => {
        if (!renameValue.trim() || renameValue === list.title) { setIsRenaming(false); return; }
        try {
            const resp = await fetch(`${API_BASE_URL}/api/lists/${id}`, { method: 'PATCH', body: JSON.stringify({ title: renameValue.trim() }) });
            if (resp.ok) { setList({ ...list, title: renameValue.trim() }); setIsRenaming(false); }
            else showToast(await getApiError(resp, 'Failed to rename list'));
        } catch { showToast('Network error — could not rename list'); }
    };

    const handleDeleteList = async () => { setIsDeleteModalOpen(true); };

    const confirmDeleteList = async () => {
        try {
            const resp = await fetch(`${API_BASE_URL}/api/lists/${id}`, { method: 'DELETE' });
            if (resp.ok) navigate('/');
            else showToast(await getApiError(resp, 'Failed to delete list'));
        } catch { showToast('Network error — could not delete list'); }
        setIsDeleteModalOpen(false);
    };

    // ── early exits ────────────────────────────────────────────────────────────

    if (!list) return <div className="container">Loading...</div>;

    // ── category / item grouping (shared) ──────────────────────────────────────

    const getSortedCategoryIds = () => {
        const allCatIds = categories.map(c => c.id);
        if (selectedShopId && shopOrder.length > 0) {
            const orderMap = {};
            shopOrder.forEach(o => (orderMap[o.category_id] = o.sort_order));
            return [...allCatIds].sort((a, b) => (orderMap[a] || 999) - (orderMap[b] || 999));
        }
        return [...allCatIds].sort((a, b) => {
            const catA = categories.find(c => c.id === a);
            const catB = categories.find(c => c.id === b);
            return (catA?.sort_order || 0) - (catB?.sort_order || 0);
        });
    };

    const ZERO_UUID = '00000000-0000-0000-0000-000000000000';
    const groupedItems = list.items?.reduce((acc, item) => {
        const catId = (item.category_id && item.category_id !== ZERO_UUID) ? item.category_id : 'uncategorized';
        if (!acc[catId]) acc[catId] = [];
        acc[catId].push(item);
        return acc;
    }, {}) || {};

    const sortedCatIds = getSortedCategoryIds();
    const finalSortedCatIds = [...sortedCatIds];
    Object.keys(groupedItems).forEach(catId => { if (!finalSortedCatIds.includes(catId)) finalSortedCatIds.push(catId); });

    const getCategoryName = (catId) => categories.find(c => c.id === catId)?.name || (catId === 'uncategorized' ? 'Uncategorized' : 'Unknown');

    // Absent items are out of stock, so they will never be paid for — excluding
    // them keeps this figure honest as "what this costs" rather than drifting
    // further from reality with every item the shopper can't find.
    const total = list.items?.filter(i => !i.is_absent)
        .reduce((s, i) => s + ((i.price || 0) * (i.quantity || 1)), 0) || 0;
    const absentCount = list.items?.filter(i => i.is_absent).length || 0;

    // ── shared modals (used in both views) ────────────────────────────────────

    const sharedModals = (
        <>
            {/* Edit Item Modal */}
            <Modal isOpen={!!editingItemId} onClose={() => setEditingItemId(null)} title="Edit Item"
                footer={(<>
                    <button onClick={() => setEditingItemId(null)} className="btn-secondary" style={{ padding: '0.5rem 1rem', borderRadius: '8px', border: '1px solid var(--border)', background: 'white' }}>Cancel</button>
                    <button onClick={saveEdit} className="btn-primary" style={{ padding: '0.5rem 1.5rem', borderRadius: '8px' }}>Save Changes</button>
                </>)}
            >
                <div style={{ display: 'flex', flexDirection: 'column', gap: '1rem' }}>
                    <div style={{ display: 'flex', flexDirection: 'column', gap: '0.25rem' }}>
                        <label className="input-label">Item Name</label>
                        <input disabled={!!editItemData.flyer_item_id} style={{ padding: '0.75rem', borderRadius: '12px', border: '1px solid var(--border)', fontWeight: 700, background: editItemData.flyer_item_id ? 'var(--bg-secondary)' : 'white' }} value={editItemData.name || ''} onChange={e => setEditItemData({ ...editItemData, name: e.target.value, preferred_alias_id: null })} />
                    </div>
                    <div className="form-row compact">
                        <div style={{ display: 'flex', flexDirection: 'column', gap: '0.25rem' }}>
                            <label className="input-label">Qty</label>
                            <input type="number" step="0.1" style={{ padding: '0.5rem', borderRadius: '12px', border: '1px solid var(--border)', fontWeight: 700, width: '100%', boxSizing: 'border-box' }} value={editItemData.quantity || ''} onChange={e => setEditItemData({ ...editItemData, quantity: e.target.value })} />
                        </div>
                        <div style={{ display: 'flex', flexDirection: 'column', gap: '0.25rem' }}>
                            <label className="input-label">Unit</label>
                            <select style={{ padding: '0.5rem', borderRadius: '12px', border: '1px solid var(--border)', background: 'white', fontWeight: 600, width: '100%', boxSizing: 'border-box' }} value={editItemData.unit || 'pcs'} onChange={e => setEditItemData({ ...editItemData, unit: e.target.value })}>
                                {['pcs', 'kg', 'g', '100g', 'l', 'pack'].map(u => <option key={u} value={u}>{u}</option>)}
                            </select>
                        </div>
                        <div style={{ display: 'flex', flexDirection: 'column', gap: '0.25rem' }}>
                            <label className="input-label">Price</label>
                            <input disabled={!!editItemData.flyer_item_id} type="number" style={{ padding: '0.5rem', borderRadius: '12px', border: '1px solid var(--border)', background: editItemData.flyer_item_id ? 'var(--bg-secondary)' : 'white', width: '100%', boxSizing: 'border-box' }} value={editItemData.price || ''} onChange={e => setEditItemData({ ...editItemData, price: e.target.value })} />
                        </div>
                    </div>
                    <div style={{ display: 'flex', flexDirection: 'column', gap: '0.25rem' }}>
                        <label className="input-label">Category</label>
                        <select style={{ padding: '0.75rem', borderRadius: '12px', border: '1px solid var(--border)', background: 'white', fontWeight: 600 }} value={editItemData.category_id || ''} onChange={e => setEditItemData({ ...editItemData, category_id: e.target.value })}>
                            {categories.map(c => <option key={c.id} value={c.id}>{c.name}</option>)}
                        </select>
                    </div>
                    {!editItemData.receipt_item_id && !editItemData.flyer_item_id && (() => {
                        const itemAliases = aliasesByPlanned[(editItemData.name || '').toLowerCase()] || [];
                        if (!itemAliases.length) return null;
                        return (
                            <div style={{ display: 'flex', flexDirection: 'column', gap: '0.25rem' }}>
                                <label className="input-label">Receipt Variant</label>
                                <select style={{ padding: '0.75rem', borderRadius: '12px', border: '1px solid var(--border)', background: 'white', fontWeight: 600 }} value={editItemData.preferred_alias_id || ''} onChange={e => { const aliasId = e.target.value ? parseInt(e.target.value) : null; const alias = itemAliases.find(a => a.id === aliasId); setEditItemData({ ...editItemData, preferred_alias_id: aliasId, price: alias ? alias.last_price : editItemData.price }); }}>
                                    <option value="">— no preference —</option>
                                    {itemAliases.map(a => <option key={a.id} value={a.id}>{a.receipt_name}{a.shop?.name ? ` @ ${a.shop.name}` : ''}{a.last_price > 0 ? ` · ${a.last_price.toFixed(2)} ${currency}` : ''}</option>)}
                                </select>
                            </div>
                        );
                    })()}
                    <div style={{ display: 'flex', flexDirection: 'column', gap: '0.25rem' }}>
                        <label className="input-label">Description</label>
                        <textarea style={{ padding: '0.75rem', borderRadius: '12px', border: '1px solid var(--border)', minHeight: '80px', fontFamily: 'inherit' }} value={editItemData.description || ''} onChange={e => setEditItemData({ ...editItemData, description: e.target.value })} />
                    </div>
                </div>
            </Modal>

            {/* Delete List Modal */}
            <Modal isOpen={isDeleteModalOpen} onClose={() => setIsDeleteModalOpen(false)} title="Delete List?"
                footer={(<>
                    <button onClick={() => setIsDeleteModalOpen(false)} className="btn-secondary" style={{ padding: '0.5rem 1rem', borderRadius: '8px', border: '1px solid var(--border)', background: 'white' }}>Cancel</button>
                    <button onClick={confirmDeleteList} className="btn-primary" style={{ padding: '0.5rem 1.5rem', borderRadius: '8px', background: 'var(--danger)' }}>Confirm Delete</button>
                </>)}
            >
                <p style={{ fontWeight: 600, color: 'var(--text-main)' }}>Are you sure you want to delete <strong style={{ color: 'var(--text-dark)' }}>{list.title}</strong>? This action cannot be undone.</p>
            </Modal>

            {/* Delete Item Modal */}
            <Modal isOpen={!!itemToDelete} onClose={() => setItemToDelete(null)} title="Delete Item?"
                footer={(<>
                    <button onClick={() => setItemToDelete(null)} className="btn-secondary" style={{ padding: '0.5rem 1rem', borderRadius: '8px', border: '1px solid var(--border)', background: 'white' }}>Cancel</button>
                    <button onClick={confirmDeleteItem} className="btn-primary" style={{ padding: '0.5rem 1.5rem', borderRadius: '8px', background: 'var(--danger)' }}>Confirm Delete</button>
                </>)}
            >
                <p style={{ fontWeight: 600, color: 'var(--text-main)' }}>Are you sure you want to delete <strong style={{ color: 'var(--text-dark)' }}>{itemToDelete?.name}</strong>?</p>
            </Modal>

            {/* Link Alias Modal */}
            {linkAliasItem && (() => {
                const isReceiptItem = !!linkAliasItem.receipt_item_id;
                const existingAliases = isReceiptItem ? (aliasesByReceipt[linkAliasItem.name.toLowerCase()] || []) : [];
                const trimmedInput = linkAliasInput.trim();
                const willRemovePlanned = isReceiptItem ? (linkAliasSelected !== null || (trimmedInput !== '' && (list?.items || []).some(i => !i.receipt_item_id && i.id !== linkAliasItem.id && i.name.trim().toLowerCase() === trimmedInput.toLowerCase()))) : true;
                const confirmDisabled = linking || (isReceiptItem ? !trimmedInput : !linkAliasTarget);
                const confirmLabel = willRemovePlanned ? 'Create alias & remove planned item' : 'Create alias';
                let modalBody;
                if (isReceiptItem) {
                    const suggs = (list?.items || []).filter(i => !i.receipt_item_id && i.id !== linkAliasItem.id && (linkAliasInput === '' || i.name.toLowerCase().includes(linkAliasInput.toLowerCase())));
                    modalBody = (<>
                        {existingAliases.length > 0 && (<div style={{ marginBottom: '0.75rem', padding: '0.5rem 0.75rem', background: 'var(--bg-secondary)', borderRadius: '6px', border: '1px solid var(--border)' }}>
                            {existingAliases.map(ea => (<div key={ea.id} style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                                <span style={{ fontSize: '0.85rem' }}>Currently linked to: <strong>{ea.planned_name}</strong></span>
                                <button onClick={() => handleRemoveAlias(ea.id)} disabled={linking} style={{ color: 'var(--danger)', padding: '2px 6px', fontSize: '0.75rem', border: '1px solid var(--danger)', borderRadius: '4px', background: 'transparent', minHeight: 'unset' }}>Remove</button>
                            </div>))}
                        </div>)}
                        <p style={{ marginBottom: '0.5rem', fontSize: '0.9rem' }}>{existingAliases.length > 0 ? 'Change link to:' : `"${linkAliasItem.name}" is the receipt/store name for:`}</p>
                        <input value={linkAliasInput} onChange={e => { setLinkAliasInput(e.target.value); setLinkAliasSelected(null); }} placeholder="Type planned name or pick from list…" autoFocus style={{ width: '100%', padding: '0.5rem', borderRadius: '6px', border: '1px solid var(--border)', marginBottom: '0.25rem' }} />
                        {suggs.length > 0 && (<ul className="alias-suggestions">{suggs.map(s => <li key={s.id} onClick={() => { setLinkAliasInput(s.name); setLinkAliasSelected(s); }}>{s.name}</li>)}</ul>)}
                        {willRemovePlanned && trimmedInput !== '' && (<p style={{ color: 'var(--text-secondary)', fontSize: '0.85em', marginTop: '0.5rem' }}>Will remove the matching planned item from the list after creating alias.</p>)}
                    </>);
                } else {
                    const candidates = (list?.items || []).filter(i => !!i.receipt_item_id && i.id !== linkAliasItem.id);
                    modalBody = (<>
                        <p style={{ marginBottom: '0.75rem' }}>"{linkAliasItem.name}" maps to which receipt item?</p>
                        {candidates.length === 0 ? <p style={{ color: 'var(--text-secondary)' }}>No receipt items in this list.</p>
                            : <select value={linkAliasTarget} onChange={e => setLinkAliasTarget(e.target.value)} style={{ width: '100%', padding: '0.5rem', borderRadius: '6px', border: '1px solid var(--border)' }}>
                                <option value="">— select —</option>
                                {candidates.map(c => <option key={c.id} value={c.id}>{c.name}</option>)}
                            </select>}
                    </>);
                }
                return (
                    <Modal isOpen={true} onClose={() => setLinkAliasItem(null)} title={isReceiptItem ? 'Link as alias' : 'Assign alias'}
                        footer={(<>
                            <button onClick={() => setLinkAliasItem(null)} disabled={linking} className="btn-secondary" style={{ padding: '0.5rem 1rem', borderRadius: '8px', border: '1px solid var(--border)', background: 'white' }}>Cancel</button>
                            <button onClick={handleLinkAlias} disabled={confirmDisabled} className="btn-primary" style={{ padding: '0.5rem 1.5rem', borderRadius: '8px', background: 'var(--primary)' }}>{linking ? 'Linking…' : confirmLabel}</button>
                        </>)}
                    >{modalBody}</Modal>
                );
            })()}

            <ImageModal src={selectedPhoto2?.src} alt={selectedPhoto2?.alt} onClose={() => setSelectedPhoto2(null)} />
            <ReceiptUploadModal isOpen={isReceiptModalOpen} onClose={() => setIsReceiptModalOpen(false)} listId={id} onUploadSuccess={() => { fetchList(); fetchFrequentItems(); }} />
            <ReceiptViewerModal receipts={list.receipts || []} isOpen={isReceiptViewerOpen} onClose={() => setIsReceiptViewerOpen(false)} />
        </>
    );

    // ══════════════════════════════════════════════════════════════════════════
    // MANAGER VIEW — new KC3 design
    // ══════════════════════════════════════════════════════════════════════════

    if (isManager) {
        const statusStyle = STATUS_STYLE[list.status] || STATUS_STYLE['preparing'];
        const isCompleted = list.status === 'completed';

        // Autocomplete: local filter on suggestions already fetched from API
        const showDropdown = showSuggestions && suggestions.length > 0;

        return (
            <>
                <style>{`
                    @keyframes kcSheetUp { from { transform: translateY(100%) } to { transform: translateY(0) } }
                    @keyframes kcRowPop { 0%{transform:scale(.97);opacity:.4} 60%{transform:scale(1.01);opacity:1} 100%{transform:scale(1)} }
.kc-variants::-webkit-scrollbar { display: none; }
                `}</style>

                <div style={{ position: 'fixed', inset: 0, display: 'flex', flexDirection: 'column', background: '#f8fafc', fontFamily: 'Inter, system-ui, sans-serif', overflow: 'hidden' }}>

                    {/* ── Header ───────────────────────────────────────────────── */}
                    <div style={{ padding: '8px 14px 10px', background: '#fff', borderBottom: '1px solid #f1f5f9', flexShrink: 0 }}>
                        <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>

                            {/* Back */}
                            <button onClick={() => navigate('/')} title="Back to Dashboard" style={{ width: 34, height: 34, minHeight: 'unset', borderRadius: 10, background: '#f1f5f9', border: 'none', display: 'flex', alignItems: 'center', justifyContent: 'center', cursor: 'pointer', color: '#475569', flexShrink: 0 }}>
                                <ArrowLeft size={20} strokeWidth={2.2} />
                            </button>

                            {/* Title + rename */}
                            <div style={{ flex: 1, minWidth: 0 }}>
                                {isRenaming ? (
                                    <input value={renameValue} onChange={e => setRenameValue(e.target.value)} onBlur={handleRenameList} onKeyDown={e => e.key === 'Enter' && handleRenameList()} autoFocus style={{ fontSize: 17, fontWeight: 800, color: '#020617', border: 'none', borderBottom: '1.5px dashed #e2e8f0', outline: 'none', background: 'transparent', width: '100%', padding: '2px 0', minHeight: 'unset' }} />
                                ) : (
                                    <div style={{ display: 'flex', alignItems: 'center', gap: 6 }}>
                                        <h1 style={{ fontSize: 17, fontWeight: 800, color: '#020617', whiteSpace: 'nowrap', overflow: 'hidden', textOverflow: 'ellipsis', margin: 0 }}>{list.title}</h1>
                                        <button onClick={() => { setIsRenaming(true); setRenameValue(list.title); }} style={{ display: 'flex', color: '#94a3b8', background: 'none', border: 'none', padding: 0, cursor: 'pointer', minHeight: 'unset', flexShrink: 0 }}>
                                            <Edit2 size={16} />
                                        </button>
                                    </div>
                                )}
                                <div style={{ display: 'flex', alignItems: 'center', gap: 6, marginTop: 2 }}>
                                    <span style={{ fontSize: 11, color: '#64748b', fontVariantNumeric: 'tabular-nums' }}>
                                        {/* The count covers every item; the estimate excludes absent
                                            ones, since they will never be paid for. Say so rather than
                                            leaving the two numbers quietly covering different sets. */}
                                        {list.items?.length || 0} items · est. <b style={{ color: '#0f172a' }}>{total.toFixed(0)} {currency}</b>
                                        {absentCount > 0 && ` (excl. ${absentCount} not found)`}
                                    </span>
                                    <button onClick={cycleStatus} data-testid="status-badge" title="Click to change status" style={{ fontSize: 10, fontWeight: 700, textTransform: 'uppercase', letterSpacing: '.04em', padding: '1px 6px', borderRadius: 9999, background: statusStyle.bg, color: statusStyle.color, border: 'none', cursor: 'pointer', minHeight: 'unset' }}>
                                        {statusStyle.label}
                                    </button>
                                </div>
                            </div>

                            {/* Upload button — enabled only when completed */}
                            <button
                                onClick={() => isCompleted && setIsReceiptModalOpen(true)}
                                disabled={!isCompleted}
                                title={isCompleted ? 'Upload receipt' : 'Only available after list is completed'}
                                style={{ width: 34, height: 34, minHeight: 'unset', borderRadius: 10, flexShrink: 0, display: 'flex', alignItems: 'center', justifyContent: 'center', border: 'none', cursor: isCompleted ? 'pointer' : 'not-allowed', background: isCompleted ? '#2563eb' : '#f8fafc', color: isCompleted ? '#fff' : '#cbd5e1', ...(isCompleted ? {} : { border: '1px dashed #cbd5e1' }) }}
                            >
                                <Upload size={14} />
                            </button>

                            {/* View receipts (if any) */}
                            {(list.receipts?.length > 0) && (
                                <button onClick={() => setIsReceiptViewerOpen(true)} style={{ width: 34, height: 34, minHeight: 'unset', borderRadius: 10, background: '#f1f5f9', border: 'none', display: 'flex', alignItems: 'center', justifyContent: 'center', cursor: 'pointer', color: '#2563eb', flexShrink: 0, position: 'relative' }}>
                                    <FileText size={14} />
                                    <span style={{ position: 'absolute', top: -4, right: -4, background: '#2563eb', color: '#fff', fontSize: '0.5rem', fontWeight: 700, width: 14, height: 14, borderRadius: 7, display: 'flex', alignItems: 'center', justifyContent: 'center' }}>{list.receipts.length}</span>
                                </button>
                            )}

                            {/* Delete list */}
                            <button onClick={handleDeleteList} title="Delete List" style={{ width: 34, height: 34, minHeight: 'unset', borderRadius: 10, background: '#f1f5f9', border: 'none', display: 'flex', alignItems: 'center', justifyContent: 'center', cursor: 'pointer', color: 'var(--danger)', flexShrink: 0 }}>
                                <Trash2 size={16} />
                            </button>
                        </div>
                    </div>

                    {/* ── Scrollable item list (order 2) ───────────────────────── */}
                    <div style={{ flex: 1, overflowY: 'auto', padding: '10px 12px 16px', order: 2 }}>
                        {finalSortedCatIds.filter(catId => groupedItems[catId]).map(catId => {
                            const catName = getCategoryName(catId);
                            const catObj  = categories.find(c => c.id === catId) || null;
                            const catItems = groupedItems[catId];
                            const catEmoji = getCategoryEmoji(catName, catObj?.icon);
                            return (
                                <div key={catId} style={{ background: '#fff', borderRadius: 12, border: '1px solid #e2e8f0', overflow: 'hidden', marginBottom: 8, boxShadow: '0 1px 2px rgba(0,0,0,.04)' }}>
                                    {/* Category header */}
                                    <div style={{ display: 'flex', alignItems: 'center', gap: 8, padding: '8px 12px', background: '#f8fafc' }}>
                                        {catEmoji && <span style={{ fontSize: 15 }}>{catEmoji}</span>}
                                        <span style={{ fontSize: 12, fontWeight: 700, color: '#0f172a', textTransform: 'uppercase', letterSpacing: '.04em' }}>{catName}</span>
                                        <span style={{ fontSize: 11, color: '#94a3b8', marginLeft: 'auto' }}>{catItems.length}</span>
                                    </div>

                                    {/* Item rows */}
                                    {catItems.map(item => {
                                        const expanded = expandedId === item.id;
                                        const edits = expandedEdits[item.id] || {};
                                        const isJustAdded = justAddedId === item.id;
                                        const plannedAliases = aliasesByPlanned[item.name.toLowerCase()] || [];
                                        const receiptAliases = aliasesByReceipt[item.name.toLowerCase()] || [];

                                        return (
                                            <div key={item.id} style={{ borderTop: '1px solid #f1f5f9', background: isJustAdded ? '#f0fdf4' : '#fff', transition: 'background .6s ease', animation: isJustAdded ? 'kcRowPop .35s ease' : undefined }}>
                                                {/* Collapsed row */}
                                                <div onClick={() => toggleExpand(item)} style={{ display: 'flex', alignItems: 'center', gap: 10, padding: '10px 12px', cursor: 'pointer' }}>
                                                    <div style={{ width: 36, height: 36, borderRadius: 8, background: '#f1f5f9', display: 'flex', alignItems: 'center', justifyContent: 'center', fontSize: 19, flexShrink: 0 }}>
                                                        {catEmoji}
                                                    </div>
                                                    <div style={{ flex: 1, minWidth: 0 }}>
                                                        <div style={{ display: 'flex', alignItems: 'center', gap: 6, flexWrap: 'wrap' }}>
                                                            <span data-testid="item-name" style={{ fontSize: 14, fontWeight: 600, color: '#0f172a' }}>{item.name}</span>
                                                            {item.is_urgent && (
                                                                <span style={{ fontSize: 9.5, fontWeight: 800, color: '#b91c1c', background: '#fee2e2', padding: '1px 6px', borderRadius: 9999 }}>URGENT</span>
                                                            )}
                                                            {/* Shopper couldn't find it. Bought wins over absent, so a
                                                                bought item never carries this. */}
                                                            {item.is_absent && (
                                                                <span data-testid="item-not-found-badge" style={{ fontSize: 9.5, fontWeight: 800, color: '#9a3412', background: '#ffedd5', padding: '1px 6px', borderRadius: 9999 }}>Not found</span>
                                                            )}
                                                            {item.flyer_item_id && (
                                                                <span style={{ fontSize: 9.5, fontWeight: 800, color: '#065f46', background: '#d1fae5', padding: '1px 6px', borderRadius: 9999 }}>Sale Deal</span>
                                                            )}
                                                        </div>
                                                        <div style={{ fontSize: 12, color: '#64748b', marginTop: 1, fontVariantNumeric: 'tabular-nums', display: 'flex', gap: 4, alignItems: 'center', flexWrap: 'wrap' }}>
                                                            <span>{item.quantity || 1} {item.unit || 'pcs'} · ≈{((item.price || 0) * (item.quantity || 1)).toFixed(0)} {currency}</span>
                                                            {item.description && <span style={{ color: '#94a3b8', fontStyle: 'italic' }}>· {item.description}</span>}
                                                            {plannedAliases.length > 0 && <span style={{ color: '#94a3b8', fontStyle: 'italic' }}>· {plannedAliases[0].receipt_name}</span>}
                                                            {receiptAliases.length > 0 && <span data-testid="item-alias-label" style={{ color: '#94a3b8', fontStyle: 'italic' }}>→ {receiptAliases[0].planned_name}</span>}
                                                        </div>
                                                    </div>
                                                    <span style={{ color: '#94a3b8', display: 'flex', transform: expanded ? 'rotate(180deg)' : 'none', transition: 'transform .2s', flexShrink: 0 }}>
                                                        <ChevronDown size={14} />
                                                    </span>
                                                </div>

                                                {/* Expanded panel */}
                                                {expanded && (
                                                    <div style={{ padding: '0 12px 12px', background: '#f8fafc', borderTop: '1px dashed #e2e8f0' }}>
                                                        {/* Qty / Unit / Price */}
                                                        <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr 1fr', gap: 8, paddingTop: 10 }}>
                                                            {[
                                                                { label: 'QTY', key: 'quantity', value: edits.quantity ?? item.quantity ?? 1, testid: 'inline-qty-input' },
                                                                { label: 'UNIT', key: 'unit', value: edits.unit ?? item.unit ?? 'pcs', testid: 'inline-unit-input' },
                                                                { label: 'PRICE', key: 'price', value: edits.price ?? item.price ?? 0, testid: 'inline-price-input' },
                                                            ].map(({ label, key, value, testid }) => (
                                                                <div key={key}>
                                                                    <div style={{ fontSize: 9.5, fontWeight: 700, color: '#64748b', textTransform: 'uppercase', letterSpacing: '.05em', marginBottom: 4 }}>{label}</div>
                                                                    <input
                                                                        data-testid={testid}
                                                                        value={value}
                                                                        onChange={e => setExpandedEdits(prev => ({ ...prev, [item.id]: { ...prev[item.id], [key]: e.target.value } }))}
                                                                        style={{ width: '100%', padding: '8px 10px', borderRadius: 8, border: '1px solid #e2e8f0', fontFamily: 'Inter, system-ui, sans-serif', fontSize: 13, fontWeight: 600, outline: 'none', boxSizing: 'border-box', minHeight: 'unset' }}
                                                                    />
                                                                </div>
                                                            ))}
                                                        </div>

                                                        {/* Category chips */}
                                                        <div style={{ display: 'flex', alignItems: 'center', gap: 5, marginTop: 10, flexWrap: 'wrap' }}>
                                                            <span style={{ fontSize: 10, fontWeight: 700, color: '#64748b', textTransform: 'uppercase', letterSpacing: '.05em', marginRight: 2, flexShrink: 0 }}>Cat:</span>
                                                            {categories.map(cat => (
                                                                <button key={cat.id} onClick={() => setExpandedEdits(prev => ({ ...prev, [item.id]: { ...prev[item.id], category_id: cat.id } }))} style={{
                                                                    padding: '4px 9px', borderRadius: 9999, fontSize: 11, fontWeight: 600, cursor: 'pointer', minHeight: 'unset',
                                                                    border: (edits.category_id ?? item.category_id) === cat.id ? '1.5px solid #2563eb' : '1px solid #e2e8f0',
                                                                    background: (edits.category_id ?? item.category_id) === cat.id ? '#eff6ff' : '#fff',
                                                                    color: (edits.category_id ?? item.category_id) === cat.id ? '#1d4ed8' : '#475569',
                                                                }}>
                                                                    {[getCategoryEmoji(cat.name, cat.icon), cat.name].filter(Boolean).join(' ')}
                                                                </button>
                                                            ))}
                                                        </div>

                                                        {/* Action row: edit modal + link alias + delete */}
                                                        <div style={{ display: 'flex', gap: 6, marginTop: 10, justifyContent: 'flex-end' }}>
                                                            <button title="Link as alias" onClick={() => { setLinkAliasItem(item); setLinkAliasTarget(''); setLinkAliasInput(''); setLinkAliasSelected(null); }} style={{ padding: '5px 9px', borderRadius: 8, background: '#eff6ff', color: '#2563eb', border: 'none', fontFamily: 'Inter, system-ui, sans-serif', fontSize: 11, fontWeight: 700, cursor: 'pointer', display: 'flex', alignItems: 'center', gap: 4, minHeight: 'unset' }}>
                                                                <Link2 size={12} /> Alias
                                                            </button>
                                                            <button title="Edit Item" onClick={() => startEditing(item)} style={{ padding: '5px 9px', borderRadius: 8, background: '#f1f5f9', color: '#475569', border: 'none', fontFamily: 'Inter, system-ui, sans-serif', fontSize: 11, fontWeight: 700, cursor: 'pointer', display: 'flex', alignItems: 'center', gap: 4, minHeight: 'unset' }}>
                                                                <Edit2 size={12} /> Edit
                                                            </button>
                                                            <button title="Delete Item" onClick={() => deleteItem(item.id)} style={{ padding: '5px 9px', borderRadius: 8, background: '#fee2e2', color: '#b91c1c', border: 'none', fontFamily: 'Inter, system-ui, sans-serif', fontSize: 11, fontWeight: 700, cursor: 'pointer', display: 'flex', alignItems: 'center', gap: 4, minHeight: 'unset' }}>
                                                                <Trash2 size={12} /> Delete
                                                            </button>
                                                        </div>
                                                    </div>
                                                )}
                                            </div>
                                        );
                                    })}
                                </div>
                            );
                        })}

                        {(!list.items || list.items.length === 0) && (
                            <div style={{ textAlign: 'center', padding: '40px 20px', color: '#94a3b8', fontSize: 14 }}>
                                No items yet — add some below ↓
                            </div>
                        )}
                    </div>

                    {/* ── Bottom quick-add bar (order 3) ───────────────────────── */}
                    {!isCompleted && <div style={{ padding: '10px 14px 8px', background: '#f8fafc', borderTop: '1px solid #f1f5f9', order: 3, flexShrink: 0 }}>
                        {/* Frequent-use chip grid */}
                        {frequentItems.length > 0 && (
                            <div style={{ marginBottom: 8 }}>
                                <div ref={chipsContainerRef} style={{ display: 'flex', flexWrap: 'wrap', gap: 5, maxHeight: chipsExpanded ? 'none' : 64, overflow: 'hidden', paddingBottom: 1 }}>
                                    {frequentItems.map(fi => (
                                        <button key={fi.id} onClick={() => openDraftNew(fi.item_name)} style={{ padding: '5px 8px 5px 10px', borderRadius: 9999, background: '#fff', border: '1px solid #e2e8f0', fontFamily: 'Inter, system-ui, sans-serif', fontSize: 12, fontWeight: 600, color: '#0f172a', cursor: 'pointer', whiteSpace: 'nowrap', display: 'flex', alignItems: 'center', gap: 3, boxShadow: '0 1px 1px rgba(0,0,0,.03)', minHeight: 'unset' }}>
                                            <span style={{ color: '#22c55e', display: 'flex', alignItems: 'center' }}><Plus size={11} /></span>
                                            {fi.item_name}
                                            <span onClick={(e) => deleteFrequentItem(e, fi.id)} style={{ color: '#94a3b8', display: 'flex', alignItems: 'center', marginLeft: 2, cursor: 'pointer' }} title="Remove"><X size={11} /></span>
                                        </button>
                                    ))}
                                </div>
                                {chipsOverflow && (
                                    <button onClick={() => setChipsExpanded(e => !e)} style={{ marginTop: 3, padding: '2px 6px', borderRadius: 6, background: 'none', border: 'none', fontSize: 11, color: '#64748b', cursor: 'pointer', display: 'flex', alignItems: 'center', gap: 2, minHeight: 'unset' }}>
                                        <ChevronDown size={12} style={{ transform: chipsExpanded ? 'rotate(180deg)' : 'none', transition: 'transform 0.15s' }} />
                                        {chipsExpanded ? 'show less' : 'show more'}
                                    </button>
                                )}
                            </div>
                        )}

                        {/* Search input + autocomplete */}
                        <div style={{ position: 'relative' }}>
                            <span style={{ position: 'absolute', left: 12, top: '50%', transform: 'translateY(-50%)', color: isListMode ? '#16a34a' : '#94a3b8', display: 'flex', pointerEvents: 'none' }}>
                                {isListMode ? <List size={16} /> : <Search size={16} />}
                            </span>
                            <input
                                ref={queryInputRef}
                                value={query}
                                onChange={e => {
                                    const val = e.target.value;
                                    setQuery(val);
                                    if (!val.includes(',')) {
                                        clearTimeout(queryDebounceRef.current);
                                        queryDebounceRef.current = setTimeout(() => fetchSuggestions(val), 250);
                                    }
                                }}
                                onKeyDown={e => {
                                    if (e.key === 'Enter') {
                                        if (isListMode) setShowPasteModal(true);
                                        else openDraftNew();
                                    }
                                }}
                                onBlur={() => setTimeout(() => setShowSuggestions(false), 150)}
                                placeholder="Add item — type, paste, or pick a chip…"
                                style={{ width: '100%', padding: '11px 80px 11px 36px', borderRadius: 12, border: `1.5px solid ${isListMode ? '#16a34a' : '#2563eb'}`, fontFamily: 'Inter, system-ui, sans-serif', fontSize: 14, fontWeight: 500, outline: 'none', background: '#fff', boxShadow: query ? `0 0 0 3px ${isListMode ? 'rgba(22,163,74,.12)' : 'rgba(37,99,235,.12)'}` : 'none', minHeight: 'unset' }}
                            />
                            <div style={{ position: 'absolute', right: 6, top: '50%', transform: 'translateY(-50%)', display: 'flex', gap: 4 }}>
                                <button title="Voice (coming soon)" style={{ width: 30, height: 30, minHeight: 'unset', borderRadius: 8, background: '#f1f5f9', color: '#475569', border: 'none', display: 'flex', alignItems: 'center', justifyContent: 'center', cursor: 'pointer' }}>
                                    <Mic size={14} />
                                </button>
                                <button
                                    onClick={() => { if (isListMode) setShowPasteModal(true); else openDraftNew(); }}
                                    disabled={!query}
                                    style={{ width: 30, height: 30, minHeight: 'unset', borderRadius: 8, background: query ? (isListMode ? '#16a34a' : '#2563eb') : '#cbd5e1', color: '#fff', border: 'none', display: 'flex', alignItems: 'center', justifyContent: 'center', cursor: query ? 'pointer' : 'not-allowed' }}
                                    title={isListMode ? 'Add as list' : 'Add item'}
                                >
                                    {isListMode ? <List size={14} /> : <Plus size={14} />}
                                </button>
                            </div>

                            {/* Autocomplete dropdown — appears above the input */}
                            {showDropdown && !isListMode && (
                                <div style={{ position: 'absolute', bottom: 'calc(100% + 6px)', left: 0, right: 0, background: '#fff', borderRadius: 12, border: '1px solid #e2e8f0', boxShadow: '0 10px 25px rgba(15,23,42,.12)', overflow: 'hidden', zIndex: 10 }}>
                                    <div style={{ padding: '6px 12px', fontSize: 10, fontWeight: 700, color: '#64748b', textTransform: 'uppercase', letterSpacing: '.05em', background: '#f8fafc', display: 'flex', alignItems: 'center', gap: 4 }}>
                                        ✦ From your history
                                    </div>
                                    {suggestions.slice(0, 3).map((s, i) => {
                                        const suggestionEmoji = getCategoryEmoji(s.planned_name);
                                        return (
                                        <button key={i} onMouseDown={() => openDraftFromSuggestion(s)} style={{ width: '100%', padding: '9px 12px', background: '#fff', border: 'none', borderTop: '1px solid #f1f5f9', display: 'flex', alignItems: 'center', gap: 10, cursor: 'pointer', textAlign: 'left', minHeight: 'unset' }}>
                                            {suggestionEmoji && <div style={{ width: 32, height: 32, borderRadius: 8, background: '#f1f5f9', display: 'flex', alignItems: 'center', justifyContent: 'center', fontSize: 17, flexShrink: 0 }}>{suggestionEmoji}</div>}
                                            <div style={{ flex: 1, minWidth: 0 }}>
                                                <div style={{ fontSize: 13, fontWeight: 600, color: '#0f172a' }}>{s.planned_name}</div>
                                                {s.variants?.[0] && (
                                                    <div style={{ fontSize: 11, color: '#64748b', fontVariantNumeric: 'tabular-nums' }}>
                                                        {s.variants[0].receipt_name}
                                                        {s.variants[0].last_price > 0 ? ` · ~${s.variants[0].last_price} ${currency}` : ''}
                                                    </div>
                                                )}
                                            </div>
                                        </button>
                                        );
                                    })}
                                    <button onMouseDown={() => openDraftNew()} style={{ width: '100%', padding: '9px 12px', background: '#f8fafc', border: 'none', borderTop: '1px solid #f1f5f9', display: 'flex', alignItems: 'center', gap: 8, cursor: 'pointer', textAlign: 'left', color: '#2563eb', fontWeight: 600, fontSize: 13, minHeight: 'unset' }}>
                                        <Plus size={14} /> Add "{query}" as new item
                                    </button>
                                </div>
                            )}
                        </div>
                    </div>}
                </div>

                {/* Confirm sheet overlay */}
                {draft && (
                    <ConfirmSheet
                        draft={draft}
                        onChange={setDraft}
                        onCancel={() => setDraft(null)}
                        onConfirm={addFromDraft}
                        categories={categories}
                        currency={currency}
                    />
                )}

                <Modal isOpen={showPasteModal} onClose={() => setShowPasteModal(false)} title="Add list of items">
                    <PasteItemsPanel
                        listId={id}
                        shops={shops}
                        initialText={query}
                        onItemsAdded={() => { setShowPasteModal(false); setQuery(''); fetchList(); fetchFrequentItems(); }}
                    />
                </Modal>

                {sharedModals}
            </>
        );
    }

    // ══════════════════════════════════════════════════════════════════════════
    // SHOPPER VIEW — existing layout (unchanged)
    // ══════════════════════════════════════════════════════════════════════════

    // Shopper view only: bought and absent items leave their category groups and
    // collect in one "done" section at the bottom. groupedItems itself is left
    // alone because the manager view renders from it and must stay unchanged.
    const isDone = (item) => item.is_bought || item.is_absent;
    const doneItems = list.items?.filter(isDone) || [];
    const activeGroupedItems = Object.fromEntries(
        Object.entries(groupedItems)
            .map(([catId, items]) => [catId, items.filter(i => !isDone(i))])
            .filter(([, items]) => items.length > 0)
    );

    // Absent items are resolved for this trip, so they count as progress --
    // otherwise the bar sticks below 100% when the only leftovers are out of stock.
    // Exclusivity means an item is never counted twice here.
    const progress = list.items?.length ? (list.items.filter(isDone).length / list.items.length) * 100 : 0;

    return (
        <div className="container">
            <header style={{ display: 'flex', alignItems: 'center', gap: '0.75rem', marginBottom: '1rem', paddingTop: '0.5rem', flexWrap: 'wrap', background: 'var(--surface)', borderRadius: 'var(--radius)', padding: '0.75rem' }}>
                <button onClick={() => navigate('/')} className="card" style={{ padding: '0.4rem', borderRadius: '50%', flexShrink: 0, minHeight: 'unset' }} title="Back to Dashboard">
                    <ArrowLeft size={18} />
                </button>
                <div style={{ flex: '1 1 150px', minWidth: 0 }}>
                    <div style={{ display: 'flex', alignItems: 'center', gap: '0.4rem', flexWrap: 'nowrap' }}>
                        <h1 style={{ fontSize: '1.1rem', fontWeight: 800, margin: 0, whiteSpace: 'nowrap', overflow: 'hidden', textOverflow: 'ellipsis' }}>{list.title}</h1>
                    </div>
                    <div style={{ display: 'flex', gap: '0.5rem', marginTop: '0.15rem', flexWrap: 'wrap', alignItems: 'center' }}>
                        <span className={`badge ${list.status === 'completed' ? 'badge-success' : list.status === 'ready for shopping' ? 'badge-warning' : 'badge-neutral'}`} style={{ fontSize: '0.6rem', padding: '2px 6px' }}>{list.status}</span>
                        <div style={{ display: 'flex', alignItems: 'center', gap: '0.3rem', color: 'var(--text-muted)', fontSize: '0.7rem', fontWeight: 600 }}>
                            <span style={{ color: 'var(--success)' }}>{total.toFixed(2)} {currency}</span>
                        </div>
                    </div>
                </div>
                {isShopper && list.status === 'ready for shopping' && (
                    <div style={{ position: 'relative' }}>
                        <select value={selectedShopId} onChange={handleShopChange} title="Select a shop to reorder the list according to its layout" style={{ appearance: 'none', padding: '0.5rem 2rem 0.5rem 2.5rem', borderRadius: '20px', border: '1px solid var(--border)', fontSize: '0.8rem', fontWeight: 700, background: 'white', outline: 'none' }}>
                            <option value="">Default Order</option>
                            {shops.map(shop => <option key={shop.id} value={shop.id}>{shop.name}</option>)}
                        </select>
                        <Store size={16} style={{ position: 'absolute', left: '10px', top: '50%', transform: 'translateY(-50%)', color: 'var(--primary)' }} />
                    </div>
                )}
                <div style={{ display: 'flex', border: '1px solid var(--border)', borderRadius: '8px', overflow: 'hidden', flexShrink: 0, marginLeft: 'auto' }}>
                    <button onClick={() => setIsReceiptModalOpen(true)} style={{ display: 'flex', alignItems: 'center', gap: '4px', padding: '0.35rem 0.6rem', background: 'var(--surface)', border: 'none', borderRight: list.receipts?.length > 0 ? '1px solid var(--border)' : 'none', color: 'var(--text-muted)', cursor: 'pointer', fontSize: '0.72rem', fontWeight: 600, minHeight: 'unset' }} title="Upload Receipt">
                        <Upload size={14} /> Upload
                    </button>
                    {list.receipts?.length > 0 && (
                        <button data-testid="view-receipts-btn" onClick={() => setIsReceiptViewerOpen(true)} style={{ display: 'flex', alignItems: 'center', gap: '4px', padding: '0.35rem 0.6rem', background: 'var(--surface)', border: 'none', color: 'var(--primary)', cursor: 'pointer', fontSize: '0.72rem', fontWeight: 600, minHeight: 'unset' }} title="View Receipts">
                            <FileText size={14} /> View
                            <span style={{ background: 'var(--primary)', color: 'white', fontSize: '0.55rem', fontWeight: 700, minWidth: '14px', height: '14px', borderRadius: '7px', padding: '0 3px', display: 'flex', alignItems: 'center', justifyContent: 'center' }}>{list.receipts.length}</span>
                        </button>
                    )}
                </div>
            </header>

            {isShopper && list.status === 'ready for shopping' && (
                <div className="card" style={{ marginBottom: '2rem', background: '#22c55e', color: 'white', border: 'none' }}>
                    <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: '0.75rem', fontWeight: 600 }}>
                        <span>Progressing...</span><span>{Math.round(progress)}%</span>
                    </div>
                    <div style={{ height: '8px', background: 'rgba(255,255,255,0.3)', borderRadius: '4px', overflow: 'hidden' }}>
                        <div style={{ width: `${progress}%`, height: '100%', background: 'white', transition: 'width 0.3s ease' }} />
                    </div>
                </div>
            )}

            {isShopper && list.items?.some(i => i.is_urgent && !isDone(i)) && (
                <div className="card" style={{ background: '#f97316', color: 'white', marginBottom: '1.5rem', display: 'flex', alignItems: 'center', gap: '1rem', border: 'none' }}>
                    <AlertCircle size={24} />
                    <div style={{ flex: 1 }}>
                        <p style={{ fontWeight: 800, fontSize: '0.9rem' }}>URGENT ADDITION!</p>
                        <p style={{ fontSize: '0.8rem' }}>Manager added a new item</p>
                    </div>
                    <button className="glass" style={{ padding: '0.5rem 1rem', borderRadius: '8px', color: 'white', fontWeight: 600 }}>OK</button>
                </div>
            )}

            {/* Grouped Items */}
            <div style={{ display: 'flex', flexDirection: 'column', gap: '1.5rem', marginBottom: '6rem' }}>
                {finalSortedCatIds.filter(catId => activeGroupedItems[catId]).map(catId => (
                    <div key={catId}>
                        <h3 style={{ fontSize: '1rem', fontWeight: 700, marginBottom: '0.75rem', color: 'var(--text-muted)', textTransform: 'uppercase', letterSpacing: '0.05em' }}>{getCategoryName(catId)}</h3>
                        <div style={{ display: 'flex', flexDirection: 'column', gap: '0.75rem' }}>
                            {activeGroupedItems[catId].map(item => (
                                // Everything here is active: activeGroupedItems filters out
                                // anything bought or absent, so neither flag can be true.
                                <div key={item.id} className="card" style={{ display: 'flex', alignItems: 'center', gap: '1rem', borderLeft: item.is_urgent ? '4px solid var(--danger)' : 'none', flexWrap: 'wrap' }}>
                                    {isShopper ? (
                                        <>
                                            <button onClick={() => toggleItem(item)} aria-label="Mark as bought" title="Mark as bought" style={{ width: '28px', height: '28px', borderRadius: '6px', border: '2px solid var(--border)', background: 'transparent', display: 'flex', alignItems: 'center', justifyContent: 'center', flexShrink: 0 }} />
                                            {/* Bought wins over absent, so this control is hidden once an
                                                item is bought -- the backend would reject the PATCH anyway. */}
                                            {!item.is_bought && (
                                                <button onClick={() => toggleAbsent(item)} aria-label="Mark as not available in store" title="Mark as not available in store" style={{ width: '28px', height: '28px', borderRadius: '6px', border: '2px solid var(--border)', background: 'transparent', display: 'flex', alignItems: 'center', justifyContent: 'center', color: 'var(--text-muted)', flexShrink: 0 }}>
                                                    <X size={16} />
                                                </button>
                                            )}
                                        </>
                                    ) : (
                                        <div style={{ width: '8px', height: '8px', borderRadius: '50%', background: 'var(--primary)', flexShrink: 0 }} />
                                    )}
                                    {item.local_photo_path && (
                                        <div style={{ width: '48px', height: '48px', borderRadius: '8px', overflow: 'hidden', flexShrink: 0 }}>
                                            <img src={`${API_BASE_URL}${item.local_photo_path}`} alt={item.name} style={{ width: '100%', height: '100%', objectFit: 'cover', cursor: 'zoom-in' }} onClick={() => setSelectedPhoto2({ src: `${API_BASE_URL}${item.local_photo_path}`, alt: item.name })} />
                                        </div>
                                    )}
                                    <div style={{ flex: 1 }}>
                                        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', flexWrap: 'wrap', gap: '0.5rem' }}>
                                            <div style={{ display: 'flex', alignItems: 'center', gap: '0.75rem', flexWrap: 'wrap' }}>
                                                <p className="text-break" style={{ fontWeight: 800, fontSize: '1.15rem', color: 'var(--text-dark)' }}>{item.name}</p>
                                                {item.flyer_item_id && (<span style={{ fontSize: '0.65rem', background: 'var(--primary)', color: 'white', padding: '2px 8px', borderRadius: '6px', fontWeight: 900, display: 'flex', alignItems: 'center', gap: '4px', textTransform: 'uppercase' }}><ShoppingCart size={10} /> Sale Deal</span>)}
                                                {item.receipt_item_id && (<span style={{ fontSize: '0.65rem', background: 'var(--text-dark)', color: 'white', padding: '2px 8px', borderRadius: '6px', fontWeight: 900, display: 'flex', alignItems: 'center', gap: '4px', textTransform: 'uppercase' }}><Receipt size={10} /> Found in Receipt</span>)}
                                                <span style={{ fontSize: '0.9rem', background: 'var(--success)', color: 'white', padding: '4px 12px', borderRadius: '20px', fontWeight: 800, boxShadow: 'var(--shadow-sm)', whiteSpace: 'nowrap' }}>{item.quantity || 1} {item.unit || 'pcs'}</span>
                                                {item.is_urgent && (<span style={{ fontSize: '0.7rem', background: 'var(--danger)', color: 'white', padding: '2px 8px', borderRadius: '6px', fontWeight: 900, letterSpacing: '0.05em' }}>URGENT</span>)}
                                            </div>
                                            <div style={{ textAlign: 'right' }}>
                                                {item.price > 0 && (
                                                    <div style={{ display: 'flex', flexDirection: 'column', alignItems: 'flex-end' }}>
                                                        <p style={{ fontWeight: 800, fontSize: '1rem', color: 'var(--text-dark)' }}>≈ {((item.price) * (item.quantity || 1)).toFixed(2)} {currency}</p>
                                                        <p style={{ fontSize: '0.75rem', color: 'var(--text-muted)', fontWeight: 700 }}>({item.price}/{item.unit || 'pcs'})</p>
                                                    </div>
                                                )}
                                            </div>
                                        </div>
                                        {item.description && (<p style={{ fontSize: '0.95rem', color: isShopper ? 'var(--primary)' : 'var(--text-main)', marginTop: '0.6rem', paddingTop: '0.4rem', borderTop: '1px solid var(--border)', lineHeight: '1.4', fontWeight: isShopper ? 600 : 400 }}>{item.description}</p>)}
                                    </div>
                                </div>
                            ))}
                        </div>
                    </div>
                ))}

                {/* Done section: bought + absent, collapsed by default, hidden when empty */}
                {doneItems.length > 0 && (
                    <div>
                        <button
                            onClick={() => setDoneExpanded(v => !v)}
                            aria-expanded={doneExpanded}
                            title={doneExpanded ? 'Hide done items' : 'Show done items'}
                            style={{ display: 'flex', alignItems: 'center', gap: '0.5rem', width: '100%', padding: '0.5rem 0', color: 'var(--text-muted)', fontWeight: 700, fontSize: '0.9rem', textTransform: 'uppercase', letterSpacing: '0.05em' }}
                        >
                            <ChevronDown size={16} style={{ transform: doneExpanded ? 'rotate(0deg)' : 'rotate(-90deg)', transition: 'transform 0.15s' }} />
                            {doneItems.length} done
                        </button>
                        {doneExpanded && (
                            <div style={{ display: 'flex', flexDirection: 'column', gap: '0.5rem', marginTop: '0.5rem' }}>
                                {doneItems.map(item => (
                                    <div key={item.id} className="card" style={{ display: 'flex', alignItems: 'center', gap: '0.75rem', opacity: 0.65, flexWrap: 'wrap' }}>
                                        <span style={{ fontSize: '0.65rem', background: item.is_bought ? 'var(--success)' : 'var(--danger)', color: 'white', padding: '2px 8px', borderRadius: '6px', fontWeight: 900, textTransform: 'uppercase', flexShrink: 0 }}>
                                            {item.is_bought ? 'Bought' : 'Not found'}
                                        </span>
                                        <p className="text-break" style={{ flex: 1, fontWeight: 700, textDecoration: item.is_bought ? 'line-through' : 'none', color: 'var(--text-dark)' }}>{item.name}</p>
                                        {/* "Found it after all": marking an absent item bought is a
                                            real in-store event, so it gets a direct control rather
                                            than forcing undo → hunt for the row → check off. The
                                            server clears is_absent on this transition. */}
                                        {!item.is_bought && (
                                            <button
                                                onClick={() => toggleItem(item)}
                                                aria-label={`Mark ${item.name} as bought`}
                                                title="Found it after all — mark as bought"
                                                style={{ padding: '0.25rem 0.6rem', borderRadius: '6px', border: '1px solid var(--success)', background: 'var(--success)', color: 'white', fontWeight: 700, fontSize: '0.8rem', flexShrink: 0 }}
                                            >
                                                Bought
                                            </button>
                                        )}
                                        <button
                                            onClick={() => (item.is_bought ? toggleItem(item) : toggleAbsent(item))}
                                            aria-label={item.is_bought ? `Mark ${item.name} as not bought` : `Mark ${item.name} as available`}
                                            title={item.is_bought ? 'Mark as not bought' : 'Mark as available'}
                                            style={{ padding: '0.25rem 0.6rem', borderRadius: '6px', border: '1px solid var(--border)', color: 'var(--text-muted)', fontWeight: 700, fontSize: '0.8rem', flexShrink: 0 }}
                                        >
                                            Undo
                                        </button>
                                    </div>
                                ))}
                            </div>
                        )}
                    </div>
                )}
            </div>

            {isShopper && list.status === 'ready for shopping' && (
                <footer style={{ position: 'fixed', bottom: 0, left: 0, right: 0, padding: '1rem', background: 'white', borderTop: '1px solid var(--border)', display: 'flex', justifyContent: 'center' }}>
                    <div className="container" style={{ padding: 0 }}>
                        <button onClick={() => updateStatus('completed')} className="btn-primary" style={{ width: '100%' }} title="Mark the shopping as completed and finish the list">
                            <Check size={20} /> Complete Shopping
                        </button>
                    </div>
                </footer>
            )}

            {sharedModals}
        </div>
    );
};

export default ListDetail;
