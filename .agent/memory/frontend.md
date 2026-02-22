# Frontend Standards & PWA

## 1. Visual & UX Rules
- **Aesthetics**: Maintain high visual quality using `lucide-react` icons and glassmorphism effects where appropriate.
- **Feedback**: Provide immediate feedback for long-running processes (e.g., "Queued" for receipt uploads).
- **Responsive design**: Prioritize mobile-friendly layouts (PWA focus).

## 2. Performance Patterns
- **Lazy Loading**: Use `IntersectionObserver` (via `LazyImage.jsx`) for deal images in lists.
- **Root Margin**: Set at `200px` to trigger loading before visibility.

## 3. PWA & Native Integration
- **Share Target**: Intercepts `image/*` and `application/pdf` via `sw.js` and `POST /share-target`.
- **Manifest**: 
    - Icons must have `purpose: 'maskable'` and `purpose: 'any'`.
    - `display: 'standalone'` for native experience.
- **IndexedDB**: Use `localforage` for temporary storage of shared files to handle unauthenticated state redirects.
- **Cleanup**: ALWAYS remove temporary share data from IndexedDB after successful upload or discard.
