import React, { useState, useEffect } from 'react';
import { useAuth } from '../context/AuthContext';
import { Plus, ShoppingBasket, CheckCircle2, Clock, Copy, Settings } from 'lucide-react';
import { useNavigate } from 'react-router-dom';
import { API_BASE_URL } from '../config';

const Dashboard = () => {
    const { user, token, mode, currency, toggleMode } = useAuth();
    const navigate = useNavigate();
    const [lists, setLists] = useState([]);

    useEffect(() => {
        fetchLists();
    }, []);

    const fetchLists = async () => {
        const resp = await fetch(`${API_BASE_URL}/api/lists`, {
            headers: { 'Authorization': `Bearer ${token}` }
        });
        if (resp.ok) {
            setLists(await resp.json());
        }
    };

    const isManager = mode === 'manager';

    const createNewList = async () => {
        const title = prompt('Enter list title:');
        if (!title) return;

        const resp = await fetch(`${API_BASE_URL}/api/lists`, {
            method: 'POST',
            headers: {
                'Authorization': `Bearer ${token}`,
                'Content-Type': 'application/json'
            },
            body: JSON.stringify({ title })
        });

        if (resp.ok) {
            fetchLists();
        }
    };

    const duplicateList = async (e, id) => {
        e.stopPropagation();
        const resp = await fetch(`${API_BASE_URL}/api/lists/${id}/duplicate`, {
            method: 'POST',
            headers: { 'Authorization': `Bearer ${token}` }
        });
        if (resp.ok) {
            fetchLists();
        }
    };

    return (
        <div className="container">
            <header style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '2rem', paddingTop: '1rem' }}>
                <div>
                    <h1 style={{ fontSize: '1.5rem', fontWeight: 800 }}>KinCart</h1>
                    <div style={{ display: 'flex', alignItems: 'center', gap: '1rem' }}>
                        <p style={{ color: 'var(--text-muted)', fontSize: '0.875rem' }}>
                            {isManager ? 'Manager Mode' : 'Shopper Mode'} • {user?.username}
                        </p>
                        <button
                            title={`Switch to ${isManager ? 'Shopper' : 'Manager'} mode`}
                            onClick={toggleMode}
                            style={{
                                background: 'var(--card-bg)',
                                border: '1px solid var(--border)',
                                borderRadius: '20px',
                                padding: '2px 12px',
                                fontSize: '0.75rem',
                                cursor: 'pointer',
                                fontWeight: 600,
                                color: 'var(--primary)'
                            }}
                        >
                            Switch to {isManager ? 'Shopper' : 'Manager'}
                        </button>
                    </div>
                </div>
                <div style={{ display: 'flex', gap: '0.5rem' }}>
                    <button
                        onClick={() => navigate('/settings')}
                        className="card"
                        style={{ padding: '0.5rem', borderRadius: '50%', display: 'flex', alignItems: 'center', justifyContent: 'center' }}
                        title="Family and Application Settings"
                    >
                        <Settings size={20} />
                    </button>
                    <button className="card" style={{ padding: '0.5rem', borderRadius: '50%' }} title="User Profile">
                        {/* User Icon Placeholder */}
                    </button>
                </div>
            </header >

            {isManager && (
                <button
                    onClick={createNewList}
                    className="btn-primary"
                    style={{ width: '100%', marginBottom: '2rem', height: '60px', fontSize: '1.125rem' }}
                    title="Create a new shopping list for the family"
                >
                    <Plus size={24} />
                    Create New List
                </button>
            )}

            <section>
                <h2 style={{ fontSize: '1.25rem', fontWeight: 700, marginBottom: '1.25rem' }}>
                    {isManager ? 'All Lists' : 'Active Shopping'}
                </h2>

                <div style={{ display: 'flex', flexDirection: 'column', gap: '1rem' }}>
                    {lists
                        .filter(list => isManager || list.status === 'ready for shopping')
                        .map(list => (
                            <div
                                key={list.id}
                                className="card"
                                onClick={() => navigate(`/list/${list.id}`)}
                                style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', cursor: 'pointer' }}
                                title={`View details of ${list.title}`}
                            >
                                <div style={{ flex: 1 }}>
                                    <h3 style={{ fontSize: '1.125rem', fontWeight: 600, marginBottom: '0.25rem' }}>{list.title}</h3>
                                    <div style={{ display: 'flex', gap: '1rem', fontSize: '0.875rem', color: 'var(--text-muted)' }}>
                                        <span>{list.items?.length || 0} items</span>
                                        <span>≈ {list.estimated_amount || 0} {currency}</span>
                                    </div>
                                </div>
                                <div style={{ textAlign: 'right', display: 'flex', alignItems: 'center', gap: '1rem' }}>
                                    <div>
                                        <span className={`badge ${list.status === 'completed' ? 'badge-success' : list.status === 'ready for shopping' ? 'badge-warning' : 'badge-neutral'}`}>
                                            {list.status === 'completed' ? <CheckCircle2 size={14} style={{ marginRight: '4px' }} /> : <Clock size={14} style={{ marginRight: '4px' }} />}
                                            {list.status}
                                        </span>
                                        <p style={{ marginTop: '0.5rem', fontWeight: 700, color: 'var(--success)' }}>
                                            {list.actual_amount > 0 ? `${list.actual_amount} ${currency}` : ''}
                                        </p>
                                    </div>
                                    {isManager && (
                                        <button
                                            onClick={(e) => duplicateList(e, list.id)}
                                            style={{ background: 'transparent', border: 'none', color: 'var(--primary)', cursor: 'pointer', padding: '0.5rem' }}
                                            title="Duplicate List"
                                        >
                                            <Copy size={20} />
                                        </button>
                                    )}
                                </div>
                            </div>
                        ))}

                    {lists.filter(list => isManager || list.status === 'ready for shopping').length === 0 && (
                        <div style={{ textAlign: 'center', padding: '3rem', color: 'var(--text-muted)' }}>
                            <ShoppingBasket size={48} style={{ opacity: 0.2, marginBottom: '1rem' }} />
                            <p>No {isManager ? '' : 'active'} lists yet. {isManager ? "Let's start planning!" : "Wait for a manager to assign one."}</p>
                        </div>
                    )}
                </div>
            </section>
        </div >
    );
};

export default Dashboard;
