import { useReducer, useEffect } from 'react';
import { useAuth } from '../context/AuthContext';
import { API_BASE_URL } from '../config';
import './ReceiptViewerModal.css';

const initialFileState = {
    selectedReceiptId: null,
    blobUrl: null,
    textContent: null,
    isLoadingFile: false,
};

function fileReducer(state, action) {
    switch (action.type) {
        case 'SELECT': return { ...state, selectedReceiptId: action.id, blobUrl: null, textContent: null, isLoadingFile: true };
        case 'LOADED_BLOB': return { ...state, blobUrl: action.url, isLoadingFile: false };
        case 'LOADED_TEXT': return { ...state, textContent: action.text, isLoadingFile: false };
        case 'ERROR': return { ...state, isLoadingFile: false };
        case 'BACK': return { ...state, selectedReceiptId: null, blobUrl: null, textContent: null, isLoadingFile: false };
        case 'CLOSE': return initialFileState;
        default: return state;
    }
}

export default function ReceiptViewerModal({ receipts, isOpen, onClose }) {
    const { token } = useAuth();
    const [fileState, dispatch] = useReducer(fileReducer, initialFileState);
    const { selectedReceiptId, blobUrl, textContent, isLoadingFile } = fileState;

    const selectedReceipt = receipts?.find(r => r.id === selectedReceiptId) || null;

    // Load file when a receipt is selected
    useEffect(() => {
        if (!selectedReceiptId || !selectedReceipt) return;

        let cancelled = false;
        let createdBlobUrl = null;

        const isTextReceipt = selectedReceipt.image_path?.endsWith('.txt');

        fetch(`${API_BASE_URL}/api/receipts/${selectedReceiptId}/file`, {
            headers: { Authorization: `Bearer ${token}` },
        })
            .then(res => {
                if (!res.ok) throw new Error('Failed to load receipt file');
                return isTextReceipt ? res.text() : res.blob();
            })
            .then(data => {
                if (cancelled) return;
                if (isTextReceipt) {
                    dispatch({ type: 'LOADED_TEXT', text: data });
                } else {
                    createdBlobUrl = URL.createObjectURL(data);
                    dispatch({ type: 'LOADED_BLOB', url: createdBlobUrl });
                }
            })
            .catch(err => {
                console.error('Receipt file load error:', err);
                if (!cancelled) dispatch({ type: 'ERROR' });
            });

        return () => {
            cancelled = true;
            if (createdBlobUrl) URL.revokeObjectURL(createdBlobUrl);
        };
    }, [selectedReceiptId]); // eslint-disable-line react-hooks/exhaustive-deps

    // Revoke blob and reset state when modal closes
    useEffect(() => {
        if (!isOpen) {
            if (blobUrl) URL.revokeObjectURL(blobUrl);
            dispatch({ type: 'CLOSE' });
        }
    }, [isOpen]); // eslint-disable-line react-hooks/exhaustive-deps

    const handleBack = () => {
        if (blobUrl) URL.revokeObjectURL(blobUrl);
        dispatch({ type: 'BACK' });
    };

    const handleDownload = async () => {
        if (!selectedReceipt) return;
        const ext = selectedReceipt.image_path?.split('.').pop() || 'bin';
        const date = selectedReceipt.date
            ? new Date(selectedReceipt.date).toISOString().split('T')[0]
            : 'unknown';
        const shop = selectedReceipt.shop?.name?.toLowerCase().replace(/\s+/g, '-') || null;
        const filename = shop
            ? `receipt-${date}-${shop}.${ext}`
            : `receipt-${selectedReceipt.id}.${ext}`;

        try {
            const res = await fetch(`${API_BASE_URL}/api/receipts/${selectedReceipt.id}/file`, {
                headers: { Authorization: `Bearer ${token}` },
            });
            const blob = await res.blob();
            const url = URL.createObjectURL(blob);
            const a = document.createElement('a');
            a.href = url;
            a.download = filename;
            document.body.appendChild(a);
            a.click();
            document.body.removeChild(a);
            URL.revokeObjectURL(url);
        } catch (err) {
            console.error('Download error:', err);
        }
    };

    const statusLabel = status => {
        if (status === 'parsed') return 'Parsed';
        if (status === 'error') return 'Error';
        return 'Pending';
    };

    if (!isOpen) return null;

    return (
        <div className="receipt-viewer-overlay" data-testid="receipt-viewer-overlay" onClick={onClose}>
            <div className="receipt-viewer-modal" onClick={e => e.stopPropagation()}>
                {selectedReceiptId ? (
                    <>
                        <div className="receipt-viewer-header">
                            <button data-testid="receipt-viewer-back" onClick={handleBack}>
                                ← Back
                            </button>
                            <button
                                data-testid="receipt-viewer-download"
                                className="receipt-download-btn"
                                onClick={handleDownload}
                            >
                                ⬇ Download
                            </button>
                        </div>
                        <div className="receipt-detail">
                            <div className="receipt-detail-file">
                                {isLoadingFile ? (
                                    <div className="receipt-loading">Loading…</div>
                                ) : selectedReceipt?.image_path?.endsWith('.txt') ? (
                                    <pre className="receipt-text">{textContent}</pre>
                                ) : selectedReceipt?.image_path?.endsWith('.pdf') ? (
                                    blobUrl ? (
                                        <embed src={blobUrl} type="application/pdf" style={{ width: '100%', height: '400px' }} />
                                    ) : (
                                        <div className="receipt-loading">Loading PDF…</div>
                                    )
                                ) : (
                                    blobUrl && <img src={blobUrl} alt="Receipt" />
                                )}
                            </div>
                            <div className="receipt-detail-items">
                                {selectedReceipt?.status === 'parsed' ? (
                                    <>
                                        <div className="receipt-detail-meta">
                                            {selectedReceipt.shop?.name && (
                                                <span>{selectedReceipt.shop.name}</span>
                                            )}
                                            {selectedReceipt.date && (
                                                <span>
                                                    {new Date(selectedReceipt.date).toLocaleDateString()}
                                                </span>
                                            )}
                                        </div>
                                        <table className="receipt-items-table">
                                            <thead>
                                                <tr>
                                                    <th>Item</th>
                                                    <th>Qty</th>
                                                    <th>Price</th>
                                                </tr>
                                            </thead>
                                            <tbody>
                                                {(selectedReceipt.items ?? []).map(item => (
                                                    <tr key={item.id}>
                                                        <td>{item.name}</td>
                                                        <td>
                                                            {item.quantity} {item.unit}
                                                        </td>
                                                        <td>${item.total_price?.toFixed(2)}</td>
                                                    </tr>
                                                ))}
                                                {(selectedReceipt.items ?? []).length === 0 && (
                                                    <tr>
                                                        <td colSpan={3} style={{ textAlign: 'center', color: 'var(--text-muted)' }}>
                                                            No items available
                                                        </td>
                                                    </tr>
                                                )}
                                            </tbody>
                                        </table>
                                        <div className="receipt-total-row">
                                            <span>Total</span>
                                            <span>${selectedReceipt.total?.toFixed(2)}</span>
                                        </div>
                                    </>
                                ) : selectedReceipt?.status === 'new' ? (
                                    <div className="receipt-parse-pending">⏳ Still processing…</div>
                                ) : (
                                    <div className="receipt-parse-error">
                                        Could not parse this receipt
                                    </div>
                                )}
                            </div>
                        </div>
                    </>
                ) : (
                    <>
                        <div className="receipt-viewer-header">
                            <h3>Receipts ({receipts?.length ?? 0})</h3>
                            <button onClick={onClose}>✕</button>
                        </div>
                        <div className="receipt-list">
                            {(receipts ?? []).map(receipt => (
                                <div
                                    key={receipt.id}
                                    className="receipt-list-item"
                                    onClick={() => dispatch({ type: 'SELECT', id: receipt.id })}
                                    data-testid={`receipt-list-item-${receipt.id}`}
                                >
                                    <div className="receipt-list-icon">
                                        {receipt.image_path?.endsWith('.txt') ? '📝' : '🖼'}
                                    </div>
                                    <div className="receipt-list-info">
                                        <div>
                                            {receipt.date
                                                ? new Date(receipt.date).toLocaleDateString()
                                                : 'Unknown date'}
                                        </div>
                                        {receipt.shop?.name && (
                                            <div className="receipt-shop">{receipt.shop.name}</div>
                                        )}
                                        {receipt.status === 'parsed' && (
                                            <div className="receipt-total">
                                                ${receipt.total?.toFixed(2)}
                                            </div>
                                        )}
                                    </div>
                                    <div
                                        className={`receipt-status-badge receipt-status-${receipt.status}`}
                                    >
                                        {statusLabel(receipt.status)}
                                    </div>
                                </div>
                            ))}
                        </div>
                    </>
                )}
            </div>
        </div>
    );
}
