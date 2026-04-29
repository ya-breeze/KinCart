import React, { createContext, useContext, useState, useCallback, useRef } from 'react';

const ToastContext = createContext(null);

export const getApiError = async (resp, fallback = 'Something went wrong') => {
    try {
        const data = await resp.json();
        return data?.error || fallback;
    } catch {
        return fallback;
    }
};

export const ToastProvider = ({ children }) => {
    const [toasts, setToasts] = useState([]);
    const counterRef = useRef(0);

    const showToast = useCallback((message, type = 'error') => {
        const id = ++counterRef.current;
        setToasts(prev => [...prev, { id, message, type }]);
        setTimeout(() => {
            setToasts(prev => prev.filter(t => t.id !== id));
        }, 4000);
    }, []);

    const dismiss = useCallback((id) => {
        setToasts(prev => prev.filter(t => t.id !== id));
    }, []);

    return (
        <ToastContext.Provider value={{ showToast }}>
            {children}
            {toasts.length > 0 && (
                <div style={{
                    position: 'fixed',
                    bottom: '1.5rem',
                    right: '1.5rem',
                    display: 'flex',
                    flexDirection: 'column',
                    gap: '0.5rem',
                    zIndex: 9999,
                    maxWidth: 'min(360px, calc(100vw - 3rem))',
                    pointerEvents: 'none',
                }}>
                    {toasts.map(toast => (
                        <div key={toast.id} style={{
                            display: 'flex',
                            alignItems: 'flex-start',
                            gap: '0.75rem',
                            padding: '0.875rem 1rem',
                            background: 'var(--surface)',
                            borderRadius: 'var(--radius)',
                            boxShadow: 'var(--shadow-lg)',
                            borderLeft: `4px solid ${toast.type === 'error' ? 'var(--danger)' : 'var(--success)'}`,
                            fontSize: '0.875rem',
                            pointerEvents: 'auto',
                        }}>
                            <span style={{ flex: 1, color: 'var(--text-main)', lineHeight: 1.5 }}>{toast.message}</span>
                            <button
                                onClick={() => dismiss(toast.id)}
                                style={{ background: 'none', border: 'none', cursor: 'pointer', color: 'var(--text-muted)', padding: '0', lineHeight: 1, flexShrink: 0, fontSize: '1rem' }}
                                aria-label="Dismiss"
                            >✕</button>
                        </div>
                    ))}
                </div>
            )}
        </ToastContext.Provider>
    );
};

const noopToast = { showToast: () => {} };

// eslint-disable-next-line react-refresh/only-export-components
export const useToast = () => useContext(ToastContext) ?? noopToast;
