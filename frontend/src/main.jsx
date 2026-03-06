import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import './index.css'
import App from './App.jsx'

// Global fetch interceptor to catch Cloudflare Access redirects behind the PWA
const originalFetch = window.fetch;
window.fetch = async function(...args) {
  const [resource, config] = args;
  const url = typeof resource === 'string' ? resource : (resource instanceof Request ? resource.url : '');
  
  if (url.includes('/api/')) {
    const newConfig = { ...(config || {}), redirect: 'manual' };
    try {
      const response = await originalFetch(resource, newConfig);
      if (response.type === 'opaqueredirect') {
        console.warn('Detected opaque redirect (likely Cloudflare Access session expiration). Unregistering SW and reloading...');
        if ('serviceWorker' in navigator) {
          const registrations = await navigator.serviceWorker.getRegistrations();
          for (const registration of registrations) {
            await registration.unregister();
          }
        }
        window.location.reload();
        return new Promise(() => {}); // Prevent app from crashing while reloading
      }
      return response;
    } catch (err) {
      throw err;
    }
  }
  return originalFetch(...args);
};

createRoot(document.getElementById('root')).render(
  <StrictMode>
    <App />
  </StrictMode>,
)
