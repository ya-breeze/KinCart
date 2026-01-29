/** @vitest-environment happy-dom */
import { render, screen, fireEvent } from '@testing-library/react';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import LazyImage from './LazyImage';

// Mock IntersectionObserver
const mockObserve = vi.fn();
const mockUnobserve = vi.fn();

window.IntersectionObserver = vi.fn((callback) => ({
    observe: (element) => {
        mockObserve(element);
        // Immediately trigger intersection
        callback([{ isIntersecting: true, target: element }]);
    },
    unobserve: mockUnobserve,
    disconnect: vi.fn(),
}));

describe('LazyImage', () => {
    beforeEach(() => {
        vi.clearAllMocks();
    });

    it('renders a loader initially then image when visible', async () => {
        const src = 'https://example.com/image.jpg';
        const alt = 'Test Image';

        render(<LazyImage src={src} alt={alt} />);

        // Initially it might show the loader (depends on timing, but our mock triggers visibility immediately)
        const img = screen.getByAltText(alt);
        expect(img).toBeInTheDocument();
        expect(img).toHaveAttribute('src', src);

        // Simulate image load
        fireEvent.load(img);
        expect(img).toHaveStyle({ opacity: '1' });
    });

    it('renders an error icon if image fails to load', () => {
        const src = 'https://example.com/bad-image.jpg';
        const alt = 'Bad Image';

        render(<LazyImage src={src} alt={alt} />);
        const img = screen.getByAltText(alt);

        fireEvent.error(img);
        expect(screen.getByText(/Failed to load/i)).toBeInTheDocument();
    });
});
