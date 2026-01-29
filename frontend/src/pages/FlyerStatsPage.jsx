import React, { useState, useEffect } from 'react';
import { useAuth } from '../context/AuthContext';
import { ArrowLeft, BarChart3, FileText, ImageIcon, CheckCircle2, AlertCircle, ShoppingBag, X, ExternalLink, ChevronLeft, ChevronRight, Calendar } from 'lucide-react';
import { useNavigate } from 'react-router-dom';
import { API_BASE_URL } from '../config';
import { AreaChart, Area, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer, Legend } from 'recharts';

const DetailsModal = ({ isOpen, onClose, title, loading, data, type, selectedDate, onDateChange }) => {
    if (!isOpen) return null;

    const getImageUrl = (path) => {
        if (!path) return '';
        if (path.startsWith('http')) return path;
        const base = API_BASE_URL || '';
        const normalizedPath = path.startsWith('/') ? path : `/${path}`;
        if (base === '/') return normalizedPath;
        if (base.endsWith('/') && normalizedPath.startsWith('/')) {
            return base + normalizedPath.substring(1);
        }
        return base + normalizedPath;
    };

    const isDateNavigable = ['pages', 'parsed', 'errors', 'items', 'activity'].includes(type);

    const handlePrevDay = () => {
        const d = new Date(selectedDate);
        d.setDate(d.getDate() - 1);
        onDateChange(d.toISOString().split('T')[0]);
    };

    const handleNextDay = () => {
        const d = new Date(selectedDate);
        d.setDate(d.getDate() + 1);
        onDateChange(d.toISOString().split('T')[0]);
    };

    return (
        <div className="modal-overlay" style={{
            position: 'fixed',
            top: 0,
            left: 0,
            right: 0,
            bottom: 0,
            backgroundColor: 'rgba(0,0,0,0.5)',
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            zIndex: 1000,
            backdropFilter: 'blur(4px)'
        }} onClick={onClose}>
            <div className="card" style={{
                width: '90%',
                maxWidth: '800px',
                maxHeight: '80vh',
                overflow: 'hidden',
                display: 'flex',
                flexDirection: 'column',
                padding: 0
            }} onClick={e => e.stopPropagation()}>
                <header style={{
                    padding: '1.5rem',
                    borderBottom: '1px solid var(--border)',
                    display: 'flex',
                    flexDirection: 'column',
                    gap: '1rem'
                }}>
                    <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
                        <h2 style={{ fontSize: '1.25rem', fontWeight: 800 }}>{title}</h2>
                        <button onClick={onClose} style={{ padding: '0.5rem', borderRadius: '50%', color: 'var(--text-muted)' }}>
                            <X size={24} />
                        </button>
                    </div>

                    {isDateNavigable && selectedDate && (
                        <div style={{
                            display: 'flex',
                            alignItems: 'center',
                            justifyContent: 'center',
                            gap: '1rem',
                            paddingTop: '0.5rem'
                        }}>
                            <button onClick={handlePrevDay} className="btn-icon" style={{ padding: '0.25rem', borderRadius: '50%' }}>
                                <ChevronLeft size={20} />
                            </button>
                            <div style={{ display: 'flex', alignItems: 'center', gap: '0.5rem', fontWeight: 700, background: 'var(--bg-muted)', padding: '0.5rem 1rem', borderRadius: '20px' }}>
                                <Calendar size={16} />
                                <input
                                    type="date"
                                    value={selectedDate}
                                    onChange={(e) => onDateChange(e.target.value)}
                                    style={{
                                        border: 'none',
                                        background: 'transparent',
                                        fontSize: '0.875rem',
                                        fontWeight: 700,
                                        cursor: 'pointer',
                                        color: 'inherit',
                                        outline: 'none'
                                    }}
                                />
                            </div>
                            <button onClick={handleNextDay} className="btn-icon" style={{ padding: '0.25rem', borderRadius: '50%' }}>
                                <ChevronRight size={20} />
                            </button>
                        </div>
                    )}
                </header>
                <div style={{ padding: '1.5rem', overflowY: 'auto', flex: 1 }}>
                    {loading ? (
                        <div style={{ textAlign: 'center', padding: '2rem' }}>Loading details...</div>
                    ) : !data || (Array.isArray(data) && data.length === 0) || (type === 'activity' && (!data.pages || data.pages.length === 0) && (!data.items || data.items.length === 0)) ? (
                        <div style={{ textAlign: 'center', padding: '2rem', color: 'var(--text-muted)' }}>No data for this date</div>
                    ) : (
                        <div style={{ display: 'flex', flexDirection: 'column', gap: '0.75rem' }}>
                            {type === 'flyers' && data.map(item => (
                                <div key={item.id} className="card" style={{ padding: '1rem', display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                                    <div>
                                        <div style={{ fontWeight: 700 }}>{item.shop_name}</div>
                                        <div style={{ fontSize: '0.875rem', color: 'var(--text-muted)' }}>{new Date(item.start_date).toLocaleDateString()} - {new Date(item.end_date).toLocaleDateString()}</div>
                                    </div>
                                    <a href={item.url} target="_blank" rel="noreferrer" style={{ color: 'var(--primary)' }}><ExternalLink size={18} /></a>
                                </div>
                            ))}
                            {(type === 'pages' || type === 'parsed' || type === 'errors') && data?.map(item => (
                                <div key={item.id} className="card" style={{ padding: '1rem', display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                                    <div style={{ flex: 1, minWidth: 0 }}>
                                        <div style={{ fontWeight: 700, whiteSpace: 'nowrap', overflow: 'hidden', textOverflow: 'ellipsis' }}>{item.source_url || 'Local File'}</div>
                                        <div style={{ fontSize: '0.875rem', display: 'flex', gap: '0.5rem', alignItems: 'center' }}>
                                            <span style={{ color: item.is_parsed ? 'var(--success)' : (item.last_error ? 'var(--danger)' : 'var(--text-muted)') }}>
                                                {item.is_parsed ? 'Parsed' : (item.last_error ? 'Error' : 'Pending')}
                                            </span>
                                            <span style={{ color: 'var(--text-muted)' }}>•</span>
                                            <span style={{ fontWeight: 600, color: 'var(--primary)' }}>{item.shop_name}</span>
                                            <span style={{ color: 'var(--text-muted)' }}>•</span>
                                            <span style={{ color: 'var(--text-muted)' }}>{new Date(item.updated_at).toLocaleString()}</span>
                                        </div>
                                        {item.last_error && <div style={{ fontSize: '0.75rem', color: 'var(--danger)', marginTop: '0.25rem' }}>{item.last_error}</div>}
                                    </div>
                                </div>
                            ))}
                            {type === 'items' && data.map((item, idx) => (
                                <div key={idx} className="card" style={{ padding: '1rem', display: 'flex', gap: '1rem' }}>
                                    {item.local_photo_path && (
                                        <img src={getImageUrl(item.local_photo_path)} alt={item.name} style={{ width: '60px', height: '60px', objectFit: 'contain', borderRadius: '8px', background: '#f8fafc' }} />
                                    )}
                                    <div style={{ flex: 1 }}>
                                        <div style={{ fontWeight: 700 }}>{item.name}</div>
                                        <div style={{ display: 'flex', gap: '1rem', alignItems: 'center', marginTop: '0.25rem' }}>
                                            <div style={{ fontWeight: 800, color: 'var(--primary)' }}>{item.price}</div>
                                            <div style={{ fontSize: '0.875rem', color: 'var(--text-muted)' }}>{item.quantity}</div>
                                        </div>
                                        <div style={{ fontSize: '0.75rem', color: 'var(--text-muted)', marginTop: '0.25rem' }}>{item.shop_name} • {item.categories}</div>
                                    </div>
                                </div>
                            ))}
                            {type === 'activity' && (
                                <>
                                    <h3 style={{ fontSize: '0.875rem', fontWeight: 700, color: 'var(--text-muted)', textTransform: 'uppercase', marginBottom: '0.5rem' }}>Pages Processed</h3>
                                    {data.pages?.map(item => (
                                        <div key={item.id} className="card" style={{ padding: '0.75rem', marginBottom: '0.5rem' }}>
                                            <div style={{ fontWeight: 600, fontSize: '0.875rem' }}>{item.source_url || 'Local File'}</div>
                                            <div style={{ fontSize: '0.75rem', display: 'flex', justifyContent: 'space-between', marginTop: '0.25rem' }}>
                                                <div style={{ display: 'flex', gap: '0.5rem' }}>
                                                    <span style={{ color: item.is_parsed ? 'var(--success)' : (item.last_error ? 'var(--danger)' : 'var(--text-muted)') }}>
                                                        {item.is_parsed ? 'Parsed' : (item.last_error ? 'Error' : 'Pending')}
                                                    </span>
                                                    <span style={{ color: 'var(--text-muted)' }}>•</span>
                                                    <span style={{ fontWeight: 600, color: 'var(--primary)' }}>{item.shop_name}</span>
                                                </div>
                                                <span style={{ color: 'var(--text-muted)' }}>{new Date(item.updated_at).toLocaleTimeString()}</span>
                                            </div>
                                        </div>
                                    ))}
                                    <div style={{ marginTop: '1rem' }}></div>
                                    <h3 style={{ fontSize: '0.875rem', fontWeight: 700, color: 'var(--text-muted)', textTransform: 'uppercase', marginBottom: '0.5rem' }}>Items Extracted</h3>
                                    {data.items?.map((item, idx) => (
                                        <div key={idx} className="card" style={{ padding: '0.75rem', marginBottom: '0.5rem', display: 'flex', gap: '0.75rem', alignItems: 'center' }}>
                                            {item.local_photo_path && (
                                                <img src={getImageUrl(item.local_photo_path)} alt={item.name} style={{ width: '40px', height: '40px', objectFit: 'contain', borderRadius: '4px', background: '#f8fafc' }} />
                                            )}
                                            <div style={{ flex: 1 }}>
                                                <div style={{ fontWeight: 600, fontSize: '0.875rem' }}>{item.name}</div>
                                                <div style={{ display: 'flex', gap: '1rem', alignItems: 'center' }}>
                                                    <div style={{ fontWeight: 700, color: 'var(--primary)', fontSize: '0.875rem' }}>{item.price}</div>
                                                    <div style={{ color: 'var(--text-muted)', fontSize: '0.75rem' }}>{new Date(item.created_at).toLocaleTimeString()}</div>
                                                </div>
                                            </div>
                                        </div>
                                    ))}
                                </>
                            )}
                        </div>
                    )}
                </div>
            </div>
        </div>
    );
};

const FlyerStatsPage = () => {
    const { token } = useAuth();
    const navigate = useNavigate();
    const [stats, setStats] = useState(null);
    const [loading, setLoading] = useState(true);

    const [modal, setModal] = useState({
        isOpen: false,
        title: '',
        loading: false,
        data: null,
        type: '',
        selectedDate: ''
    });

    const fetchStats = async () => {
        try {
            const resp = await fetch(`${API_BASE_URL}/api/flyers/stats`, {
                headers: { 'Authorization': `Bearer ${token}` }
            });
            if (resp.ok) {
                const data = await resp.json();
                setStats(data);
            }
        } catch (err) {
            console.error('Failed to fetch stats:', err);
        } finally {
            setLoading(false);
        }
    };

    useEffect(() => {
        fetchStats();
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [token]);

    const fetchModalData = async (type, date) => {
        setModal(prev => ({ ...prev, loading: true }));
        let url = '';
        if (type === 'flyers') url = `${API_BASE_URL}/api/flyers`;
        else if (type === 'pages') url = `${API_BASE_URL}/api/flyers/pages?date=${date}`;
        else if (type === 'parsed') url = `${API_BASE_URL}/api/flyers/pages?is_parsed=true&date=${date}`;
        else if (type === 'errors') url = `${API_BASE_URL}/api/flyers/pages?has_error=true&date=${date}`;
        else if (type === 'items') url = `${API_BASE_URL}/api/flyers/items-detailed?date=${date}`;
        else if (type === 'activity') url = `${API_BASE_URL}/api/flyers/activity?date=${date}`;

        try {
            const resp = await fetch(url, {
                headers: { 'Authorization': `Bearer ${token}` }
            });
            if (resp.ok) {
                const data = await resp.json();
                setModal(prev => ({ ...prev, loading: false, data }));
            }
        } catch (err) {
            console.error('Failed to fetch details:', err);
            setModal(prev => ({ ...prev, loading: false }));
        }
    };

    const handleCardClick = async (card) => {
        let date = modal.selectedDate;

        // If no date selected, fetch the latest activity date
        if (!date && ['pages', 'parsed', 'errors', 'items'].includes(card.id)) {
            try {
                const resp = await fetch(`${API_BASE_URL}/api/flyers/activity-stats`, {
                    headers: { 'Authorization': `Bearer ${token}` }
                });
                if (resp.ok) {
                    const data = await resp.json();
                    date = data.latest_date;
                }
            } catch (err) {
                console.error('Failed to fetch activity stats:', err);
            }
        }

        // If still no date (empty database), use today as a fallback
        if (!date) date = new Date().toISOString().split('T')[0];

        setModal({
            isOpen: true,
            title: card.label,
            loading: true,
            data: null,
            type: card.id,
            selectedDate: date
        });

        fetchModalData(card.id, date);
    };

    const handleDateChange = (newDate) => {
        setModal(prev => ({ ...prev, selectedDate: newDate }));
        fetchModalData(modal.type, newDate);
    };

    const handleChartClick = async (data) => {
        if (!data || !data.activeLabel) return;
        const date = data.activeLabel;
        setModal({
            isOpen: true,
            title: `Activity Log`,
            loading: true,
            data: null,
            type: 'activity',
            selectedDate: date
        });

        fetchModalData('activity', date);
    };

    if (loading) return <div className="container" style={{ textAlign: 'center', paddingTop: '5rem' }}>Loading statistics...</div>;

    const summaryCards = [
        { id: 'flyers', label: 'Total Flyers', value: stats?.total_flyers || 0, icon: <FileText size={24} />, color: 'var(--primary)' },
        { id: 'pages', label: 'Total Pages', value: stats?.total_pages || 0, icon: <ImageIcon size={24} />, color: '#6366f1' },
        { id: 'parsed', label: 'Parsed Pages', value: stats?.parsed_pages || 0, icon: <CheckCircle2 size={24} />, color: 'var(--success)' },
        { id: 'errors', label: 'Error Pages', value: stats?.error_pages || 0, icon: <AlertCircle size={24} />, color: 'var(--danger)' },
        { id: 'items', label: 'Extracted Items', value: stats?.total_items || 0, icon: <ShoppingBag size={24} />, color: '#f59e0b' },
    ];

    return (
        <div className="container" style={{ paddingBottom: '4rem' }}>
            <header style={{
                display: 'flex',
                alignItems: 'center',
                gap: '1rem',
                marginBottom: '2rem',
                paddingTop: '1rem'
            }}>
                <button onClick={() => navigate('/settings')} className="card" style={{ padding: '0.5rem', borderRadius: '50%', flexShrink: 0 }}>
                    <ArrowLeft size={20} />
                </button>
                <h1 style={{ fontSize: '1.25rem', fontWeight: 800 }}>Flyer Execution Statistics</h1>
            </header>

            <div style={{
                display: 'grid',
                gridTemplateColumns: 'repeat(auto-fit, minmax(180px, 1fr))',
                gap: '1rem',
                marginBottom: '2.5rem'
            }}>
                {summaryCards.map((card, idx) => (
                    <div key={idx} className="card interactive" onClick={() => handleCardClick(card)} style={{
                        display: 'flex',
                        flexDirection: 'column',
                        alignItems: 'center',
                        justifyContent: 'center',
                        padding: '1.5rem',
                        textAlign: 'center',
                        cursor: 'pointer'
                    }}>
                        <div style={{ color: card.color, marginBottom: '0.75rem' }}>{card.icon}</div>
                        <div style={{ fontSize: '1.5rem', fontWeight: 800, marginBottom: '0.25rem' }}>{card.value}</div>
                        <div style={{ fontSize: '0.75rem', fontWeight: 600, color: 'var(--text-muted)', textTransform: 'uppercase' }}>{card.label}</div>
                    </div>
                ))}
            </div>

            <section className="card" style={{ padding: '1.5rem' }}>
                <div style={{ display: 'flex', alignItems: 'center', gap: '0.75rem', marginBottom: '1.5rem' }}>
                    <BarChart3 size={20} style={{ color: 'var(--primary)' }} />
                    <h2 style={{ fontSize: '1.125rem', fontWeight: 700 }}>Activity History (Last 14 Days)</h2>
                </div>

                <div style={{ width: '100%', height: 350 }}>
                    <ResponsiveContainer>
                        <AreaChart
                            data={stats?.history || []}
                            margin={{ top: 10, right: 30, left: 0, bottom: 0 }}
                            onClick={handleChartClick}
                            style={{ cursor: 'pointer' }}
                        >
                            <defs>
                                <linearGradient id="colorTotal" x1="0" y1="0" x2="0" y2="1">
                                    <stop offset="5%" stopColor="#6366f1" stopOpacity={0.8} />
                                    <stop offset="95%" stopColor="#6366f1" stopOpacity={0} />
                                </linearGradient>
                                <linearGradient id="colorParsed" x1="0" y1="0" x2="0" y2="1">
                                    <stop offset="5%" stopColor="var(--success)" stopOpacity={0.8} />
                                    <stop offset="95%" stopColor="var(--success)" stopOpacity={0} />
                                </linearGradient>
                                <linearGradient id="colorErrors" x1="0" y1="0" x2="0" y2="1">
                                    <stop offset="5%" stopColor="var(--danger)" stopOpacity={0.8} />
                                    <stop offset="95%" stopColor="var(--danger)" stopOpacity={0} />
                                </linearGradient>
                            </defs>
                            <CartesianGrid strokeDasharray="3 3" vertical={false} stroke="var(--border)" />
                            <XAxis
                                dataKey="date"
                                tick={{ fontSize: 10, fontWeight: 600 }}
                                axisLine={false}
                                tickLine={false}
                                dy={10}
                                tickFormatter={(str) => {
                                    const d = new Date(str);
                                    return d.toLocaleDateString([], { month: 'short', day: 'numeric' });
                                }}
                            />
                            <YAxis
                                tick={{ fontSize: 10, fontWeight: 600 }}
                                axisLine={false}
                                tickLine={false}
                                dx={-10}
                            />
                            <Tooltip
                                contentStyle={{
                                    background: 'white',
                                    border: '1px solid var(--border)',
                                    borderRadius: '12px',
                                    boxShadow: 'var(--shadow-lg)',
                                    padding: '12px'
                                }}
                                itemStyle={{ fontWeight: 700 }}
                            />
                            <Legend wrapperStyle={{ paddingTop: '20px' }} />
                            <Area
                                type="monotone"
                                dataKey="total"
                                name="Downloaded Pages"
                                stroke="#6366f1"
                                strokeWidth={3}
                                fillOpacity={1}
                                fill="url(#colorTotal)"
                                activeDot={{ r: 6, strokeWidth: 0 }}
                            />
                            <Area
                                type="monotone"
                                dataKey="parsed"
                                name="Parsed Pages"
                                stroke="var(--success)"
                                strokeWidth={3}
                                fillOpacity={1}
                                fill="url(#colorParsed)"
                                activeDot={{ r: 6, strokeWidth: 0 }}
                            />
                            <Area
                                type="monotone"
                                dataKey="errors"
                                name="Errors"
                                stroke="var(--danger)"
                                strokeWidth={3}
                                fillOpacity={1}
                                fill="url(#colorErrors)"
                                activeDot={{ r: 6, strokeWidth: 0 }}
                            />
                        </AreaChart>
                    </ResponsiveContainer>
                </div>
            </section>

            <DetailsModal
                isOpen={modal.isOpen}
                onClose={() => setModal(prev => ({ ...prev, isOpen: false }))}
                title={modal.title}
                loading={modal.loading}
                data={modal.data}
                type={modal.type}
                selectedDate={modal.selectedDate}
                onDateChange={handleDateChange}
            />
        </div>
    );
};

export default FlyerStatsPage;
