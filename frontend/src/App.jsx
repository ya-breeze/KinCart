import React from 'react';
import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom';
import { AuthProvider, useAuth } from './context/AuthContext';
import LoginPage from './pages/LoginPage';
import Dashboard from './pages/Dashboard';
import ListDetail from './pages/ListDetail';
import SettingsPage from './pages/SettingsPage';
import FlyerItemsPage from './pages/FlyerItemsPage';
import FlyerStatsPage from './pages/FlyerStatsPage';
import ImportReceipt from './pages/ImportReceipt';
import './index.css';

const ProtectedRoute = ({ children }) => {
  const { token, loading } = useAuth();

  if (loading) return <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'center', height: '100vh' }}>Loading...</div>;
  if (!token) return <Navigate to="/login" />;

  return children;
};

function App() {
  return (
    <AuthProvider>
      <BrowserRouter>
        <Routes>
          <Route path="/login" element={<LoginPage />} />
          <Route
            path="/"
            element={
              <ProtectedRoute>
                <Dashboard />
              </ProtectedRoute>
            }
          />
          <Route
            path="/list/:id"
            element={
              <ProtectedRoute>
                <ListDetail />
              </ProtectedRoute>
            }
          />
          <Route
            path="/settings"
            element={
              <ProtectedRoute>
                <SettingsPage />
              </ProtectedRoute>
            }
          />
          <Route
            path="/flyers"
            element={
              <ProtectedRoute>
                <FlyerItemsPage />
              </ProtectedRoute>
            }
          />
          <Route
            path="/settings/flyer-stats"
            element={
              <ProtectedRoute>
                <FlyerStatsPage />
              </ProtectedRoute>
            }
          />
          <Route
            path="/import-receipt"
            element={
              <ProtectedRoute>
                <ImportReceipt />
              </ProtectedRoute>
            }
          />
          <Route path="*" element={<Navigate to="/" />} />
        </Routes>
      </BrowserRouter>
    </AuthProvider>
  );
}

export default App;
