import React, { createContext, useContext, useState, useEffect } from 'react';
import { API_BASE_URL } from '../config';

const AuthContext = createContext(null);

export const AuthProvider = ({ children }) => {
  const [user, setUser] = useState(null);
  const [loading, setLoading] = useState(true);
  const [mode, setMode] = useState(localStorage.getItem('mode') || 'shopper');
  const [currency, setCurrency] = useState('₽');

  useEffect(() => {
    fetchProfile();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  const fetchProfile = async () => {
    try {
      const resp = await fetch(`${API_BASE_URL}/api/auth/me`, {
        credentials: 'include',
      });
      if (resp.ok) {
        const data = await resp.json();
        setUser(data);
        fetchFamilyConfig();
      } else {
        setUser(null);
      }
    } catch (err) {
      console.warn('fetchProfile network error:', err);
      setUser(null);
    } finally {
      setLoading(false);
    }
  };

  const fetchFamilyConfig = async () => {
    try {
      const resp = await fetch(`${API_BASE_URL}/api/family/config`, {
        credentials: 'include',
      });
      if (resp.ok) {
        const data = await resp.json();
        setCurrency(data.currency || '₽');
      }
    } catch (err) {
      console.error("Failed to fetch family config", err);
    }
  };

  useEffect(() => {
    localStorage.setItem('mode', mode);
  }, [mode]);

  // Listen for session expiry from the API interceptor
  useEffect(() => {
    const handler = () => setUser(null);
    window.addEventListener('auth:session-expired', handler);
    return () => window.removeEventListener('auth:session-expired', handler);
  }, []);

  const toggleMode = () => {
    setMode(prev => prev === 'manager' ? 'shopper' : 'manager');
  };

  const login = React.useCallback((userData) => {
    setUser(userData);
    fetchFamilyConfig();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  const logout = React.useCallback(async () => {
    try {
      await fetch(`${API_BASE_URL}/api/auth/logout`, {
        method: 'POST',
        credentials: 'include',
      });
    } catch (err) {
      console.error("Logout request failed:", err);
    }
    setUser(null);
  }, []);

  const contextValue = React.useMemo(() => ({
    user, mode, currency, setCurrency, toggleMode, login, logout, loading
  }), [user, mode, currency, login, logout, loading]);

  return (
    <AuthContext.Provider value={contextValue}>
      {children}
    </AuthContext.Provider>
  );
};

// eslint-disable-next-line react-refresh/only-export-components
export const useAuth = () => useContext(AuthContext);
