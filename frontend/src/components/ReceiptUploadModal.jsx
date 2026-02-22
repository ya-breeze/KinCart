import React, { useState } from 'react';
import { X, Upload, Check, Loader, FileText, Clock } from 'lucide-react';
import { API_BASE_URL } from '../config';

const ReceiptUploadModal = ({ isOpen, onClose, listId, token, onUploadSuccess }) => {
    const [inputMode, setInputMode] = useState('upload'); // 'upload' | 'paste'
    const [file, setFile] = useState(null);
    const [receiptText, setReceiptText] = useState('');
    const [isUploading, setIsUploading] = useState(false);
    const [error, setError] = useState(null);
    // null | 'parsed' | 'queued'
    const [successStatus, setSuccessStatus] = useState(null);

    if (!isOpen) return null;

    const handleFileChange = (e) => {
        if (e.target.files && e.target.files[0]) {
            setFile(e.target.files[0]);
            setError(null);
        }
    };

    const handleUpload = async () => {
        setIsUploading(true);
        setError(null);

        try {
            let resp;
            if (inputMode === 'paste') {
                resp = await fetch(`${API_BASE_URL}/api/lists/${listId}/receipts`, {
                    method: 'POST',
                    headers: {
                        'Authorization': `Bearer ${token}`,
                        'Content-Type': 'application/json',
                    },
                    body: JSON.stringify({ receipt_text: receiptText }),
                });
            } else {
                const formData = new FormData();
                formData.append('receipt', file);
                resp = await fetch(`${API_BASE_URL}/api/lists/${listId}/receipts`, {
                    method: 'POST',
                    headers: {
                        'Authorization': `Bearer ${token}`,
                    },
                    body: formData,
                });
            }

            if (!resp.ok) {
                const data = await resp.json();
                throw new Error(data.error || 'Upload failed');
            }

            const data = await resp.json();
            const status = data.status === 'parsed' ? 'parsed' : 'queued';
            setSuccessStatus(status);

            if (status === 'parsed') {
                setTimeout(() => {
                    onUploadSuccess();
                    handleClose();
                }, 1500);
            }
            // For 'queued': stay open so the user reads the explanation; they close manually.

        } catch (err) {
            setError(err.message);
        } finally {
            setIsUploading(false);
        }
    };

    const handleClose = () => {
        setInputMode('upload');
        setFile(null);
        setReceiptText('');
        setIsUploading(false);
        setError(null);
        setSuccessStatus(null);
        onClose();
    };

    const isSubmitDisabled = isUploading || !!successStatus || (inputMode === 'upload' ? !file : !receiptText.trim());

    const tabStyle = (active) => ({
        flex: 1,
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        gap: '0.4rem',
        padding: '0.5rem 1rem',
        border: 'none',
        borderRadius: '8px',
        cursor: 'pointer',
        fontSize: '0.875rem',
        fontWeight: 600,
        transition: 'all 0.15s',
        background: active ? 'white' : 'transparent',
        color: active ? 'var(--text-primary, #111)' : 'var(--text-muted)',
        boxShadow: active ? '0 1px 4px rgba(0,0,0,0.12)' : 'none',
    });

    return (
        <div style={{
            position: 'fixed',
            top: 0,
            left: 0,
            right: 0,
            bottom: 0,
            background: 'rgba(0,0,0,0.5)',
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            zIndex: 1000,
            padding: '1rem'
        }}>
            <div className="card" style={{
                background: 'white',
                width: '100%',
                maxWidth: '440px',
                padding: '1.5rem',
                display: 'flex',
                flexDirection: 'column',
                gap: '1.5rem',
                position: 'relative',
                animation: 'fadeIn 0.2s ease-out'
            }}>
                <button
                    onClick={handleClose}
                    style={{
                        position: 'absolute',
                        top: '1rem',
                        right: '1rem',
                        background: 'none',
                        border: 'none',
                        cursor: 'pointer',
                        color: 'var(--text-muted)'
                    }}
                >
                    <X size={24} />
                </button>

                <div style={{ textAlign: 'center' }}>
                    <h2 style={{ fontSize: '1.25rem', fontWeight: 800, marginBottom: '0.5rem' }}>Upload Receipt</h2>
                    <p style={{ color: 'var(--text-muted)', fontSize: '0.9rem' }}>
                        Upload a receipt file or paste receipt text to automatically check off items and track prices.
                    </p>
                </div>

                {/* Tab toggle */}
                <div style={{
                    display: 'flex',
                    background: 'var(--bg-secondary)',
                    borderRadius: '10px',
                    padding: '3px',
                    gap: '2px',
                }}>
                    <button
                        style={tabStyle(inputMode === 'upload')}
                        onClick={() => { setInputMode('upload'); setError(null); }}
                        data-testid="tab-upload"
                    >
                        <Upload size={16} />
                        Upload File
                    </button>
                    <button
                        style={tabStyle(inputMode === 'paste')}
                        onClick={() => { setInputMode('paste'); setError(null); }}
                        data-testid="tab-paste"
                    >
                        <FileText size={16} />
                        Paste Text
                    </button>
                </div>

                {!successStatus ? (
                    <>
                        {inputMode === 'upload' ? (
                            <div
                                style={{
                                    border: '2px dashed var(--border)',
                                    borderRadius: '12px',
                                    padding: '2rem',
                                    display: 'flex',
                                    flexDirection: 'column',
                                    alignItems: 'center',
                                    justifyContent: 'center',
                                    gap: '1rem',
                                    cursor: 'pointer',
                                    background: 'var(--bg-secondary)',
                                    transition: 'all 0.2s'
                                }}
                                onClick={() => document.getElementById('receipt-input').click()}
                            >
                                <input
                                    id="receipt-input"
                                    type="file"
                                    accept="image/*,application/pdf,.txt"
                                    onChange={handleFileChange}
                                    style={{ display: 'none' }}
                                />
                                {file ? (
                                    <div style={{ textAlign: 'center' }}>
                                        <div style={{ width: '64px', height: '64px', margin: '0 auto 1rem', borderRadius: '8px', overflow: 'hidden', display: 'flex', alignItems: 'center', justifyContent: 'center', background: '#f3f4f6' }}>
                                            {(file.type === 'application/pdf' || file.name.endsWith('.txt')) ? (
                                                <FileText size={32} color="var(--text-muted)" />
                                            ) : (
                                                <img src={URL.createObjectURL(file)} alt="Preview" style={{ width: '100%', height: '100%', objectFit: 'cover' }} />
                                            )}
                                        </div>
                                        <p style={{ fontWeight: 600 }}>{file.name}</p>
                                        <p style={{ fontSize: '0.8rem', color: 'var(--text-muted)' }}>{(file.size / 1024 / 1024).toFixed(2)} MB</p>
                                    </div>
                                ) : (
                                    <>
                                        <div style={{ padding: '1rem', borderRadius: '50%', background: 'white' }}>
                                            <Upload size={32} color="var(--primary)" />
                                        </div>
                                        <p style={{ fontWeight: 600, color: 'var(--primary)' }}>Click to select file</p>
                                        <p style={{ fontSize: '0.8rem', color: 'var(--text-muted)' }}>Image, PDF, or .txt</p>
                                    </>
                                )}
                            </div>
                        ) : (
                            <textarea
                                value={receiptText}
                                onChange={(e) => { setReceiptText(e.target.value); setError(null); }}
                                placeholder={`Paste receipt text here, e.g.:\n\nLidl\n01.02.2025\nMilk         1,99\nBread        2,49\n---------\nTotal        4,48`}
                                data-testid="receipt-textarea"
                                style={{
                                    width: '100%',
                                    minHeight: '200px',
                                    padding: '0.75rem',
                                    border: '2px dashed var(--border)',
                                    borderRadius: '12px',
                                    background: 'var(--bg-secondary)',
                                    fontFamily: 'monospace',
                                    fontSize: '0.875rem',
                                    resize: 'vertical',
                                    outline: 'none',
                                    boxSizing: 'border-box',
                                    color: 'var(--text-primary)',
                                }}
                            />
                        )}

                        {error && (
                            <div style={{ padding: '0.75rem', borderRadius: '8px', background: '#fee2e2', color: 'var(--danger)', fontSize: '0.9rem', textAlign: 'center' }}>
                                {error}
                            </div>
                        )}

                        <button
                            onClick={handleUpload}
                            disabled={isSubmitDisabled}
                            className="primary-btn"
                            style={{
                                width: '100%',
                                padding: '1rem',
                                display: 'flex',
                                alignItems: 'center',
                                justifyContent: 'center',
                                gap: '0.5rem',
                                opacity: isSubmitDisabled ? 0.7 : 1
                            }}
                        >
                            {isUploading ? (
                                <>
                                    <Loader className="spin" size={20} />
                                    Processing...
                                </>
                            ) : inputMode === 'upload' ? (
                                'Upload & Process'
                            ) : (
                                'Process Receipt'
                            )}
                        </button>
                    </>
                ) : successStatus === 'parsed' ? (
                    <div style={{ textAlign: 'center', padding: '2rem 0' }}>
                        <div style={{
                            width: '64px', height: '64px', borderRadius: '50%', background: 'var(--success)',
                            display: 'flex', alignItems: 'center', justifyContent: 'center', margin: '0 auto 1.5rem',
                            color: 'white'
                        }}>
                            <Check size={32} />
                        </div>
                        <h3 style={{ fontSize: '1.25rem', fontWeight: 800, color: 'var(--success)', marginBottom: '0.5rem' }}>Done!</h3>
                        <p style={{ color: 'var(--text-muted)' }}>Receipt processed and items updated.</p>
                    </div>
                ) : (
                    <div style={{ textAlign: 'center', padding: '2rem 0' }}>
                        <div style={{
                            width: '64px', height: '64px', borderRadius: '50%', background: '#fef3c7',
                            display: 'flex', alignItems: 'center', justifyContent: 'center', margin: '0 auto 1.5rem',
                        }}>
                            <Clock size={32} color="#d97706" />
                        </div>
                        <h3 style={{ fontSize: '1.25rem', fontWeight: 800, color: '#d97706', marginBottom: '0.5rem' }}>Uploaded</h3>
                        <p style={{ color: 'var(--text-muted)', marginBottom: '1.5rem' }}>
                            Receipt saved. Items will appear once AI processing completes — usually within 10 minutes.
                        </p>
                        <button onClick={handleClose} className="primary-btn" style={{ width: '100%', padding: '0.75rem' }}>
                            Got it
                        </button>
                    </div>
                )}
            </div>
        </div>
    );
};

export default ReceiptUploadModal;
