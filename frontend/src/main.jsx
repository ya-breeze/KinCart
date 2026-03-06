import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import './index.css'
import App from './App.jsx'

import { setupInterceptor } from './utils/apiInterceptor'

// Initialize API interceptors (Cloudflare redirects + Refresh Tokens)
setupInterceptor();

createRoot(document.getElementById('root')).render(
  <StrictMode>
    <App />
  </StrictMode>,
)
