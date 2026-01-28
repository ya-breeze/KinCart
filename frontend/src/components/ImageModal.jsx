import React from 'react';
import { X } from 'lucide-react';

const ImageModal = ({ src, alt, onClose }) => {
    if (!src) return null;

    return (
        <div
            onClick={onClose}
            style={{
                position: 'fixed',
                top: 0,
                left: 0,
                right: 0,
                bottom: 0,
                backgroundColor: 'rgba(0, 0, 0, 0.85)',
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'center',
                zIndex: 1000,
                cursor: 'pointer',
                backdropFilter: 'blur(8px)',
                padding: '1rem'
            }}
        >
            <button
                onClick={(e) => { e.stopPropagation(); onClose(); }}
                style={{
                    position: 'absolute',
                    top: '1.5rem',
                    right: '1.5rem',
                    background: 'rgba(255, 255, 255, 0.2)',
                    border: 'none',
                    color: 'white',
                    padding: '0.5rem',
                    borderRadius: '50%',
                    display: 'flex',
                    alignItems: 'center',
                    justifyContent: 'center',
                    zIndex: 1001
                }}
            >
                <X size={24} />
            </button>
            <img
                src={src}
                alt={alt}
                style={{
                    maxWidth: '100%',
                    maxHeight: '100%',
                    objectFit: 'contain',
                    borderRadius: '8px',
                    boxShadow: '0 25px 50px -12px rgba(0, 0, 0, 0.5)'
                }}
            />
        </div>
    );
};

export default ImageModal;
