import React, { useEffect } from 'react';
import { X } from 'lucide-react';

const Modal = ({ isOpen, onClose, title, children, footer }) => {
    useEffect(() => {
        if (isOpen) {
            document.body.style.overflow = 'hidden';
        } else {
            document.body.style.overflow = 'unset';
        }
        return () => {
            document.body.style.overflow = 'unset';
        };
    }, [isOpen]);

    if (!isOpen) return null;

    return (
        <div className="modal-overlay" onClick={onClose}>
            <div className="modal-content" onClick={e => e.stopPropagation()}>
                <header className="modal-header">
                    <h2 className="modal-title">{title}</h2>
                    <button onClick={onClose} className="modal-close-btn" title="Close">
                        <X size={24} />
                    </button>
                </header>
                <div className="modal-body">
                    {children}
                </div>
                {footer && (
                    <footer className="modal-footer">
                        {footer}
                    </footer>
                )}
            </div>
        </div>
    );
};

export default Modal;
