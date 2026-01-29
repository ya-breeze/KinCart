import React, { useState } from 'react';
import { X, Upload, Check, Loader } from 'lucide-react';
import { API_BASE_URL } from '../config';

const ReceiptUploadModal = ({ isOpen, onClose, listId, token, onUploadSuccess }) => {
    const [file, setFile] = useState(null);
    const [isUploading, setIsUploading] = useState(false);
    const [error, setError] = useState(null);
    const [success, setSuccess] = useState(false);

    if (!isOpen) return null;

    const handleFileChange = (e) => {
        if (e.target.files && e.target.files[0]) {
            setFile(e.target.files[0]);
            setError(null);
        }
    };

    const handleUpload = async () => {
        if (!file) return;

        setIsUploading(true);
        setError(null);

        const formData = new FormData();
        formData.append('receipt', file);

        try {
            const resp = await fetch(`${API_BASE_URL}/api/lists/${listId}/receipts`, {
                method: 'POST',
                headers: {
                    'Authorization': `Bearer ${token}`
                },
                body: formData
            });

            if (!resp.ok) {
                const data = await resp.json();
                throw new Error(data.error || 'Upload failed');
            }

            setSuccess(true);
            setTimeout(() => {
                onUploadSuccess();
                handleClose();
            }, 1500);

        } catch (err) {
            setError(err.message);
        } finally {
            setIsUploading(false);
        }
    };

    const handleClose = () => {
        setFile(null);
        setIsUploading(false);
        setError(null);
        setSuccess(false);
        onClose();
    };

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
                maxWidth: '400px',
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
                        Upload a photo of your receipt to automatically check off items and track prices.
                    </p>
                </div>

                {!success ? (
                    <>
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
                                accept="image/*"
                                onChange={handleFileChange}
                                style={{ display: 'none' }}
                            />
                            {file ? (
                                <div style={{ textAlign: 'center' }}>
                                    <div style={{ width: '64px', height: '64px', margin: '0 auto 1rem', borderRadius: '8px', overflow: 'hidden' }}>
                                        <img src={URL.createObjectURL(file)} alt="Preview" style={{ width: '100%', height: '100%', objectFit: 'cover' }} />
                                    </div>
                                    <p style={{ fontWeight: 600 }}>{file.name}</p>
                                    <p style={{ fontSize: '0.8rem', color: 'var(--text-muted)' }}>{(file.size / 1024 / 1024).toFixed(2)} MB</p>
                                </div>
                            ) : (
                                <>
                                    <div style={{ padding: '1rem', borderRadius: '50%', background: 'white' }}>
                                        <Upload size={32} color="var(--primary)" />
                                    </div>
                                    <p style={{ fontWeight: 600, color: 'var(--primary)' }}>Click to select photo</p>
                                </>
                            )}
                        </div>

                        {error && (
                            <div style={{ padding: '0.75rem', borderRadius: '8px', background: '#fee2e2', color: 'var(--danger)', fontSize: '0.9rem', textAlign: 'center' }}>
                                {error}
                            </div>
                        )}

                        <button
                            onClick={handleUpload}
                            disabled={!file || isUploading}
                            className="primary-btn"
                            style={{
                                width: '100%',
                                padding: '1rem',
                                display: 'flex',
                                alignItems: 'center',
                                justifyContent: 'center',
                                gap: '0.5rem',
                                opacity: (!file || isUploading) ? 0.7 : 1
                            }}
                        >
                            {isUploading ? (
                                <>
                                    <Loader className="spin" size={20} />
                                    Processing...
                                </>
                            ) : (
                                'Upload & Process'
                            )}
                        </button>
                    </>
                ) : (
                    <div style={{ textAlign: 'center', padding: '2rem 0' }}>
                        <div style={{ 
                            width: '64px', height: '64px', borderRadius: '50%', background: 'var(--success)', 
                            display: 'flex', alignItems: 'center', justifyContent: 'center', margin: '0 auto 1.5rem',
                            color: 'white'
                        }}>
                            <Check size={32} />
                        </div>
                        <h3 style={{ fontSize: '1.25rem', fontWeight: 800, color: 'var(--success)', marginBottom: '0.5rem' }}>Success!</h3>
                        <p style={{ color: 'var(--text-muted)' }}>Receipt processed and items updated.</p>
                    </div>
                )}
            </div>
        </div>
    );
};

export default ReceiptUploadModal;
