import { useEffect, useRef } from 'react';

const InfiniteScrollTrigger = ({ onIntersect, hasMore, loading }) => {
    const triggerRef = useRef();

    useEffect(() => {
        const currentRef = triggerRef.current;

        const observer = new IntersectionObserver(
            ([entry]) => {
                if (entry.isIntersecting && hasMore && !loading) {
                    onIntersect();
                }
            },
            {
                rootMargin: '400px', // Trigger early for smooth UX
                threshold: 0.1
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
    }, [onIntersect, hasMore, loading]);

    return (
        <div
            ref={triggerRef}
            style={{
                height: '20px',
                width: '100%',
                visibility: hasMore ? 'visible' : 'hidden'
            }}
        />
    );
};

export default InfiniteScrollTrigger;
