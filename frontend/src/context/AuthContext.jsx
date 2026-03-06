import React, { createContext, useContext, useState, useEffect } from 'react';
import { API_BASE_URL } from '../config';

const AuthContext = createContext(null);

export const AuthProvider = ({ children }) => {
  const [user, setUser] = useState(null);
  const [token, setToken] = useState(localStorage.getItem('token'));
  const [refreshToken, setRefreshToken] = useState(localStorage.getItem('refresh_token'));
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    if (token) {
      // For MVP, we'll assume the token is valid or fetch /me
      fetchProfile();
    } else {
      setLoading(false);
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [token]);

  const fetchProfile = async () => {
    try {
      const resp = await fetch(`${API_BASE_URL}/api/auth/me`, {
        headers: { 'Authorization': `Bearer ${token}` }
      });
      if (resp.ok) {
        const data = await resp.json();
        setUser(data);
      } else if (resp.status === 401 || resp.status === 403) {
        // Only log out if specifically unauthorized by our backend
        logout();
      } else {
        // Leave token intact for 5xx errors or weird Cloudflare states
        console.warn(`fetchProfile returned ${resp.status}, but keeping token`);
      }
    } catch (err) {
      // Network errors (like Cloudflare opaque redirects) shouldn't log the user out
      console.warn('fetchProfile network error, keeping token:', err);
    } finally {
      setLoading(false);
    }
  };

  const [mode, setMode] = useState(localStorage.getItem('mode') || 'shopper');
  const [currency, setCurrency] = useState('₽');

  useEffect(() => {
    localStorage.setItem('mode', mode);
  }, [mode]);

  useEffect(() => {
    if (token) {
      fetchFamilyConfig();
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [token]);

  const fetchFamilyConfig = async () => {
    try {
      const resp = await fetch(`${API_BASE_URL}/api/family/config`, {
        headers: { 'Authorization': `Bearer ${token}` }
      });
      if (resp.ok) {
        const data = await resp.json();
        setCurrency(data.currency || '₽');
      }
    } catch (err) {
      console.error("Failed to fetch family config", err);
    }
  };

  const toggleMode = () => {
    setMode(prev => prev === 'manager' ? 'shopper' : 'manager');
  };

  const login = (userData, userToken, userRefreshToken) => {
    setUser(userData);
    setToken(userToken);
    setRefreshToken(userRefreshToken);
    localStorage.setItem('token', userToken);
    localStorage.setItem('refresh_token', userRefreshToken);
  };

  const logout = async () => {
    // Call backend logout endpoint to blacklist token and revoke refresh token
    if (token) {
      try {
        await fetch(`${API_BASE_URL}/api/auth/logout`, {
          method: 'POST',
          headers: {
            'Authorization': `Bearer ${token}`,
            'Content-Type': 'application/json'
          },
          body: JSON.stringify({ refresh_token: refreshToken || localStorage.getItem('refresh_token') })
        });
      } catch (err) {
        console.error("Logout request failed:", err);
      }
    }

    setUser(null);
    setToken(null);
    setRefreshToken(null);
    localStorage.removeItem('token');
    localStorage.removeItem('refresh_token');
    localStorage.removeItem('mode');
  };

  return (
    <AuthContext.Provider value={{ user, token, mode, currency, setCurrency, toggleMode, login, logout, loading }}>
      {children}
    </AuthContext.Provider>
  );
};

// eslint-disable-next-line react-refresh/only-export-components
export const useAuth = () => useContext(AuthContext);
