import React, { useState, useEffect, useCallback } from 'react';
import { useNavigate, useSearchParams } from 'react-router-dom';
import localforage from 'localforage';
import { useAuth } from '../context/AuthContext';
import { FileText, Loader, Check, ArrowRight, Clock } from 'lucide-react';
import { API_BASE_URL } from '../config';

// Configure localforage to match SW
localforage.config({
    name: 'KinCart',
    storeName: 'shared_files'
});

const ImportReceipt = () => {
    const { token } = useAuth();
    const navigate = useNavigate();
    const [searchParams] = useSearchParams();

    const [pendingFiles, setPendingFiles] = useState([]);
    const [lists, setLists] = useState([]);
    const [isLoading, setIsLoading] = useState(true);
    const [isUploading, setIsUploading] = useState(false);
    const [error, setError] = useState(null);
    const [selectedListId, setSelectedListId] = useState('');
    const [newListTitle, setNewListTitle] = useState('');
    const [showCreateNew, setShowCreateNew] = useState(false);

    const fetchLists = useCallback(async () => {
        try {
            const resp = await fetch(`${API_BASE_URL}/api/lists`, {
                headers: { 'Authorization': `Bearer ${token}` }
            });
            if (resp.ok) {
                const data = await resp.json();
                setLists(data);
                // Auto-select the first list if only one exists
                if (data.length === 1) {
                    setSelectedListId(data[0].id);
                } else if (data.length === 0) {
                    setShowCreateNew(true);
                }
            }
        } catch (err) {
            console.error('Failed to fetch lists:', err);
        }
    }, [token]);

    useEffect(() => {
        const init = async () => {
            try {
                const sharedData = await localforage.getItem('pending_shared_receipts');
                if (sharedData && sharedData.length > 0) {
                    setPendingFiles(sharedData);
                } else if (searchParams.get('shared') !== 'true') {
                    // If we got here directly without a shared flag and no data, go home
                    navigate('/');
                    return;
                }

                await fetchLists();
            } catch (err) {
                console.error('Failed to load shared data:', err);
                setError('Failed to load shared receipt data.');
            } finally {
                setIsLoading(false);
            }
        };

        init();
    }, [navigate, searchParams, fetchLists]);

    const handleUpload = async () => {
        let listId = selectedListId;

        if (showCreateNew) {
            listId = await createNewList();
            if (!listId) return;
        }

        if (!listId) {
            setError('Please select or create a list');
            return;
        }

        setIsUploading(true);
        setError(null);

        try {
            for (const fileData of pendingFiles) {
                const formData = new FormData();
                // fileData.blob is a Blob stored in IDB
                formData.append('receipt', fileData.blob, fileData.name);

                const resp = await fetch(`${API_BASE_URL}/api/lists/${listId}/receipts`, {
                    method: 'POST',
                    headers: { 'Authorization': `Bearer ${token}` },
                    body: formData,
                });

                if (!resp.ok) {
                    throw new Error('Failed to upload receipt');
                }
            }

            // Cleanup
            await localforage.removeItem('pending_shared_receipts');
            navigate(`/list/${listId}`);
        } catch (err) {
            setError(err.message);
            setIsUploading(false);
        }
    };

    const createNewList = async () => {
        if (!newListTitle.trim()) {
            setError('Please enter a list title');
            return null;
        }

        try {
            const resp = await fetch(`${API_BASE_URL}/api/lists`, {
                method: 'POST',
                headers: {
                    'Authorization': `Bearer ${token}`,
                    'Content-Type': 'application/json'
                },
                body: JSON.stringify({ title: newListTitle.trim() })
            });

            if (resp.ok) {
                const data = await resp.json();
                return data.id;
            } else {
                const data = await resp.json();
                setError(data.error || 'Failed to create new list');
                return null;
            }
        } catch {
            setError('Connection error while creating list');
            return null;
        }
    };

    const handleDiscard = async () => {
        await localforage.removeItem('pending_shared_receipts');
        navigate('/');
    };

    if (isLoading) {
        return (
            <div className="container" style={{ display: 'flex', flexDirection: 'column', alignItems: 'center', justifyContent: 'center', minHeight: '80vh' }}>
                <Loader className="spin" size={48} color="var(--primary)" />
                <p style={{ marginTop: '1rem', fontWeight: 600 }}>Loading shared receipt...</p>
            </div>
        );
    }

    if (pendingFiles.length === 0) {
        return (
            <div className="container" style={{ textAlign: 'center', paddingTop: '4rem' }}>
                <Clock size={64} style={{ opacity: 0.2, marginBottom: '1rem' }} />
                <h2>No pending shared files</h2>
                <p style={{ color: 'var(--text-muted)', marginBottom: '2rem' }}>It looks like there's nothing here to import.</p>
                <button onClick={() => navigate('/')} className="btn-primary">Go to Dashboard</button>
            </div>
        );
    }

    const firstFile = pendingFiles[0];
    const isImage = firstFile.type.startsWith('image/');

    return (
        <div className="container" style={{ paddingBottom: '4rem' }}>
            <header style={{ marginBottom: '2rem', textAlign: 'center', paddingTop: '1rem' }}>
                <h1 style={{ fontSize: '1.5rem', fontWeight: 800 }}>Import Shared Receipt</h1>
                <p style={{ color: 'var(--text-muted)' }}>Choose which list to attach this receipt to.</p>
            </header>

            <div className="card" style={{ marginBottom: '2rem', display: 'flex', flexDirection: 'column', alignItems: 'center', gap: '1rem' }}>
                <div style={{ width: '100%', maxWidth: '300px', borderRadius: '12px', overflow: 'hidden', border: '1px solid var(--border)', background: 'var(--bg-secondary)' }}>
                    {isImage ? (
                        <img
                            src={URL.createObjectURL(firstFile.blob)}
                            alt="Shared"
                            style={{ width: '100%', height: 'auto', display: 'block' }}
                        />
                    ) : (
                        <div style={{ padding: '2rem', display: 'flex', flexDirection: 'column', alignItems: 'center', gap: '1rem' }}>
                            <FileText size={48} color="var(--primary)" />
                            <p style={{ fontWeight: 600, fontSize: '0.8rem', textAlign: 'center' }}>{firstFile.name}</p>
                        </div>
                    )}
                </div>
                <p style={{ fontSize: '0.9rem', fontWeight: 700 }}>
                    {pendingFiles.length} file{pendingFiles.length > 1 ? 's' : ''} detected
                </p>
            </div>

            <section className="card" style={{
                background: 'var(--surface)',
                border: '1.5px solid var(--primary)',
                boxShadow: '0 4px 12px rgba(37, 99, 235, 0.1)'
            }}>
                <h3 style={{ marginBottom: '1.25rem', fontSize: '1.1rem', fontWeight: 800 }}>Where should we add it?</h3>

                {lists.length > 0 && !showCreateNew ? (
                    <div style={{ display: 'flex', flexDirection: 'column', gap: '0.75rem' }}>
                        <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fill, minmax(200px, 1fr))', gap: '0.75rem' }}>
                            {lists.map(list => (
                                <button
                                    key={list.id}
                                    onClick={() => setSelectedListId(list.id)}
                                    style={{
                                        padding: '1rem',
                                        borderRadius: '12px',
                                        border: '2px solid',
                                        borderColor: selectedListId === list.id ? 'var(--primary)' : 'var(--border)',
                                        background: selectedListId === list.id ? '#eff6ff' : 'white',
                                        color: selectedListId === list.id ? 'var(--primary)' : 'inherit',
                                        fontWeight: 700,
                                        textAlign: 'left',
                                        cursor: 'pointer',
                                        transition: 'all 0.2s',
                                        position: 'relative',
                                        minHeight: '60px'
                                    }}
                                >
                                    {list.title}
                                    {selectedListId === list.id && (
                                        <div style={{ position: 'absolute', right: '12px', top: '50%', transform: 'translateY(-50%)', background: 'var(--primary)', color: 'white', borderRadius: '50%', padding: '2px' }}>
                                            <Check size={14} />
                                        </div>
                                    )}
                                </button>
                            ))}
                        </div>

                        <button
                            onClick={() => setShowCreateNew(true)}
                            style={{
                                padding: '0.75rem',
                                background: 'transparent',
                                border: '1px dashed var(--border)',
                                borderRadius: '8px',
                                color: 'var(--text-muted)',
                                fontWeight: 600,
                                cursor: 'pointer',
                                marginTop: '0.5rem',
                                fontSize: '0.9rem'
                            }}
                        >
                            + Create a new list instead
                        </button>
                    </div>
                ) : (
                    <div style={{ display: 'flex', flexDirection: 'column', gap: '1rem' }}>
                        <div style={{ display: 'flex', flexDirection: 'column', gap: '0.5rem' }}>
                            <label className="input-label" style={{ fontSize: '0.85rem' }}>New List Title</label>
                            <input
                                autoFocus
                                placeholder="e.g. Weekly Groceries"
                                value={newListTitle}
                                onChange={e => setNewListTitle(e.target.value)}
                                style={{ padding: '0.75rem', borderRadius: '12px', border: '1px solid var(--border)', fontWeight: 700, width: '100%', outline: 'none' }}
                            />
                        </div>
                        {lists.length > 0 && (
                            <button
                                onClick={() => setShowCreateNew(false)}
                                style={{ alignSelf: 'flex-start', background: 'transparent', border: 'none', color: 'var(--primary)', fontWeight: 600, cursor: 'pointer', fontSize: '0.85rem' }}
                            >
                                ← Back to existing lists
                            </button>
                        )}
                    </div>
                )}

                {error && (
                    <div style={{ marginTop: '1.5rem', padding: '0.75rem', background: '#fee2e2', color: 'var(--danger)', borderRadius: '8px', fontSize: '0.9rem', fontWeight: 600 }}>
                        {error}
                    </div>
                )}

                <div style={{ marginTop: '2.5rem', display: 'flex', flexDirection: 'column', gap: '1rem' }}>
                    <button
                        onClick={handleUpload}
                        disabled={isUploading || (!selectedListId && !newListTitle.trim())}
                        className="btn-primary"
                        style={{
                            width: '100%',
                            height: '56px',
                            fontSize: '1.1rem',
                            display: 'flex',
                            alignItems: 'center',
                            justifyContent: 'center',
                            gap: '0.75rem',
                            borderRadius: '28px'
                        }}
                    >
                        {isUploading ? (
                            <>
                                <Loader className="spin" size={20} />
                                Uploading...
                            </>
                        ) : (
                            <>
                                Confirm & Import
                                <ArrowRight size={20} />
                            </>
                        )}
                    </button>

                    <button
                        disabled={isUploading}
                        onClick={handleDiscard}
                        style={{
                            background: 'transparent',
                            border: 'none',
                            color: 'var(--text-muted)',
                            fontWeight: 600,
                            cursor: 'pointer',
                            padding: '0.5rem',
                            fontSize: '0.95rem'
                        }}
                    >
                        Discard & Cancel
                    </button>
                </div>
            </section>
        </div>
    );
};

export default ImportReceipt;
