import React, { useState, useEffect } from 'react';
import { useAuth } from '../context/AuthContext';
import { Plus, ShoppingBasket, CheckCircle2, Clock, Copy, Settings, Scroll } from 'lucide-react';
import { useNavigate } from 'react-router-dom';
import { API_BASE_URL } from '../config';

const Dashboard = () => {
    const { user, token, mode, currency, toggleMode } = useAuth();
    const navigate = useNavigate();
    const [lists, setLists] = useState([]);



    useEffect(() => {
        fetchLists();
        // eslint-disable-next-line react-hooks/exhaustive-deps
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

    const visibleLists = [...lists]
        .filter(list => isManager || list.status === 'ready for shopping')
        .sort((a, b) => {
            if (isManager) {
                // Manager View: Grouped by Status
                // Group Order: 'preparing' -> 'ready for shopping' -> 'completed'
                const statusPriority = { 'preparing': 0, 'ready for shopping': 1, 'completed': 2 };
                const priorityA = statusPriority[a.status] !== undefined ? statusPriority[a.status] : 99;
                const priorityB = statusPriority[b.status] !== undefined ? statusPriority[b.status] : 99;

                if (priorityA !== priorityB) {
                    return priorityA - priorityB;
                }

                // If completed, sort by completed_at desc
                if (a.status === 'completed' && a.completed_at && b.completed_at) {
                    return new Date(b.completed_at) - new Date(a.completed_at);
                }

                // Secondary sort by title
                return a.title.localeCompare(b.title);
            }
            // For shoppers: Sort by name
            return a.title.localeCompare(b.title);
        });

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
            <header style={{
                display: 'flex',
                flexWrap: 'wrap',
                justifyContent: 'space-between',
                alignItems: 'center',
                marginBottom: '2rem',
                paddingTop: '1rem',
                gap: '1rem'
            }}>
                <div style={{ minWidth: '200px' }}>
                    <h1 style={{ fontSize: '1.5rem', fontWeight: 800 }}>KinCart</h1>
                    <div style={{ display: 'flex', alignItems: 'center', gap: '0.5rem', flexWrap: 'wrap' }}>
                        <p style={{ color: 'var(--text-muted)', fontSize: '0.875rem' }}>
                            {isManager ? 'Manager Mode' : 'Shopper Mode'} • {user?.username}
                        </p>
                        <button
                            title={`Switch to ${isManager ? 'Shopper' : 'Manager'} mode`}
                            onClick={toggleMode}
                            style={{
                                background: 'white',
                                border: '1px solid var(--border)',
                                borderRadius: '20px',
                                padding: '2px 12px',
                                fontSize: '0.75rem',
                                cursor: 'pointer',
                                fontWeight: 600,
                                color: 'var(--primary)',
                                whiteSpace: 'nowrap'
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
                <div style={{ display: 'flex', gap: '1rem', marginBottom: '2rem' }}>
                    <button
                        onClick={createNewList}
                        className="btn-primary"
                        style={{ flex: 2, height: '60px', fontSize: '1.125rem' }}
                        title="Create a new shopping list for the family"
                    >
                        <Plus size={24} />
                        Create New List
                    </button>
                    <button
                        onClick={() => navigate('/flyers')}
                        className="card"
                        style={{ flex: 1, display: 'flex', flexDirection: 'column', alignItems: 'center', justifyContent: 'center', gap: '0.25rem', border: '2px solid var(--primary)', color: 'var(--primary)', fontWeight: 700 }}
                        title="View all flyer items and deals"
                    >
                        <ShoppingBasket size={24} />
                        Flyer Items
                    </button>
                </div>
            )}

            <section>
                <h2 style={{ fontSize: '1.25rem', fontWeight: 700, marginBottom: '1.25rem' }}>
                    {isManager ? 'All Lists' : 'Active Shopping'}
                </h2>

                <div style={{ display: 'flex', flexDirection: 'column', gap: '2rem' }}>
                    {isManager ? (
                        // Manager View: Grouped by Status
                        ['preparing', 'ready for shopping', 'completed'].map(statusGroup => {
                            const groupLists = visibleLists.filter(l => l.status === statusGroup);
                            if (groupLists.length === 0) return null;

                            return (
                                <div key={statusGroup}>
                                    <h3 style={{
                                        fontSize: '1rem',
                                        fontWeight: 700,
                                        color: 'var(--text-muted)',
                                        textTransform: 'uppercase',
                                        marginBottom: '1rem',
                                        display: 'flex',
                                        alignItems: 'center',
                                        gap: '0.5rem'
                                    }}>
                                        {statusGroup === 'completed' ? <CheckCircle2 size={16} /> : <Clock size={16} />}
                                        {statusGroup}
                                    </h3>
                                    <div style={{ display: 'flex', flexDirection: 'column', gap: '1rem' }}>
                                        {groupLists.map(list => (
                                            <div
                                                key={list.id}
                                                className="card"
                                                onClick={() => navigate(`/list/${list.id}`)}
                                                style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', cursor: 'pointer', flexWrap: 'wrap', gap: '1rem' }}
                                                title={`View details of ${list.title}`}
                                            >
                                                <div style={{ flex: '1 1 200px' }}>
                                                    <h3 style={{ fontSize: '1.125rem', fontWeight: 600, marginBottom: '0.25rem' }}>{list.title}</h3>
                                                    <div style={{ display: 'flex', gap: '1rem', fontSize: '0.875rem', color: 'var(--text-muted)' }}>
                                                        <span>{list.items?.length || 0} items</span>
                                                        {list.receipts?.length > 0 && (
                                                            <span style={{ display: 'flex', alignItems: 'center', gap: '4px', color: 'var(--text-dark)' }}>
                                                                <Scroll size={14} />
                                                                {list.receipts.length}
                                                            </span>
                                                        )}
                                                        <span>≈ {list.estimated_amount || 0} {currency}</span>
                                                    </div>
                                                </div>
                                                <div style={{ textAlign: 'right', display: 'flex', alignItems: 'center', gap: '1rem', flexShrink: 0 }}>
                                                    <div>
                                                        <span className={`badge ${list.status === 'completed' ? 'badge-success' : list.status === 'ready for shopping' ? 'badge-warning' : 'badge-neutral'}`}>
                                                            {list.status}
                                                        </span>
                                                        <p style={{ marginTop: '0.5rem', fontWeight: 700, color: 'var(--success)' }}>
                                                            {list.actual_amount > 0 ? `${list.actual_amount} ${currency}` : ''}
                                                        </p>
                                                    </div>
                                                    <button
                                                        onClick={(e) => duplicateList(e, list.id)}
                                                        style={{ background: 'transparent', border: 'none', color: 'var(--primary)', cursor: 'pointer', padding: '0.5rem' }}
                                                        title="Duplicate List"
                                                    >
                                                        <Copy size={20} />
                                                    </button>
                                                </div>
                                            </div>
                                        ))}
                                    </div>
                                </div>
                            );
                        })
                    ) : (
                        // Shopper View: Flat List
                        visibleLists.map(list => (
                            <div
                                key={list.id}
                                className="card"
                                onClick={() => navigate(`/list/${list.id}`)}
                                style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', cursor: 'pointer', flexWrap: 'wrap', gap: '1rem' }}
                                title={`View details of ${list.title}`}
                            >
                                <div style={{ flex: '1 1 200px' }}>
                                    <h3 style={{ fontSize: '1.125rem', fontWeight: 600, marginBottom: '0.25rem' }}>{list.title}</h3>
                                    <div style={{ display: 'flex', gap: '1rem', fontSize: '0.875rem', color: 'var(--text-muted)' }}>
                                        <span>{list.items?.length || 0} items</span>
                                        {list.receipts?.length > 0 && (
                                            <span style={{ display: 'flex', alignItems: 'center', gap: '4px', color: 'var(--text-dark)' }}>
                                                <Scroll size={14} />
                                                {list.receipts.length}
                                            </span>
                                        )}
                                        <span>≈ {list.estimated_amount || 0} {currency}</span>
                                    </div>
                                </div>
                                <div style={{ textAlign: 'right', display: 'flex', alignItems: 'center', gap: '1rem', flexShrink: 0 }}>
                                    <div>
                                        <span className={`badge ${list.status === 'completed' ? 'badge-success' : list.status === 'ready for shopping' ? 'badge-warning' : 'badge-neutral'}`}>
                                            {list.status === 'completed' ? <CheckCircle2 size={14} style={{ marginRight: '4px' }} /> : <Clock size={14} style={{ marginRight: '4px' }} />}
                                            {list.status}
                                        </span>
                                        {list.status === 'completed' && list.completed_at && (
                                            <p style={{ fontSize: '0.7rem', color: 'var(--text-muted)', marginTop: '4px', textAlign: 'right' }}>
                                                {new Date(list.completed_at).toLocaleDateString(undefined, { day: '2-digit', month: '2-digit' })}
                                            </p>
                                        )}
                                        <p style={{ marginTop: '0.5rem', fontWeight: 700, color: 'var(--success)' }}>
                                            {list.actual_amount > 0 ? `${list.actual_amount} ${currency}` : ''}
                                        </p>
                                    </div>
                                </div>
                            </div>
                        ))
                    )}

                    {visibleLists.length === 0 && (
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
