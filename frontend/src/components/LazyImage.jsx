import React, { useState, useEffect, useRef } from 'react';
import { Loader2, ImageIcon } from 'lucide-react';

const LazyImage = ({ src, alt, style, onClick }) => {
    const [isVisible, setIsVisible] = useState(false);
    const [isLoaded, setIsLoaded] = useState(false);
    const [error, setError] = useState(false);
    const imgRef = useRef();

    useEffect(() => {
        const currentRef = imgRef.current;
        const observer = new IntersectionObserver(
            ([entry]) => {
                if (entry.isIntersecting) {
                    setIsVisible(true);
                    if (currentRef) observer.unobserve(currentRef);
                }
            },
            {
                rootMargin: '200px', // Load before it actually enters
            }
        );

        if (currentRef) {
            observer.observe(currentRef);
        }

        return () => {
            if (currentRef) {
                observer.unobserve(currentRef);
            }
        };
    }, []);

    return (
        <div
            ref={imgRef}
            style={{
                width: '100%',
                height: '100%',
                position: 'relative',
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'center',
                background: 'var(--bg-secondary)',
                ...style
            }}
        >
            {isVisible && !error ? (
                <>
                    {!isLoaded && (
                        <div style={{ position: 'absolute', inset: 0, display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
                            <Loader2 className="spin" size={24} style={{ opacity: 0.3 }} />
                        </div>
                    )}
                    <img
                        src={src}
                        alt={alt}
                        onLoad={() => setIsLoaded(true)}
                        onError={() => setError(true)}
                        style={{
                            width: '100%',
                            height: '100%',
                            objectFit: 'contain',
                            cursor: onClick ? 'zoom-in' : 'default',
                            opacity: isLoaded ? 1 : 0,
                            transition: 'opacity 0.3s ease-in-out'
                        }}
                        onClick={onClick}
                    />
                </>
            ) : error ? (
                <div style={{ display: 'flex', flexDirection: 'column', alignItems: 'center', gap: '0.5rem', color: 'var(--text-muted)' }}>
                    <ImageIcon size={32} style={{ opacity: 0.3 }} />
                    <span style={{ fontSize: '0.75rem' }}>Failed to load</span>
                </div>
            ) : (
                <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
                    <Loader2 className="spin" size={24} style={{ opacity: 0.1 }} />
                </div>
            )}
        </div>
    );
};

export default LazyImage;
