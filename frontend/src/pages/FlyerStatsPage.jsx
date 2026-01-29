import React, { useState, useEffect } from 'react';
import { useAuth } from '../context/AuthContext';
import { ArrowLeft, BarChart3, FileText, ImageIcon, CheckCircle2, AlertCircle, ShoppingBag } from 'lucide-react';
import { useNavigate } from 'react-router-dom';
import { API_BASE_URL } from '../config';
import { AreaChart, Area, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer, Legend } from 'recharts';

const FlyerStatsPage = () => {
    const { token } = useAuth();
    const navigate = useNavigate();
    const [stats, setStats] = useState(null);
    const [loading, setLoading] = useState(true);

    useEffect(() => {
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

        fetchStats();
    }, [token]);

    if (loading) return <div className="container" style={{ textAlign: 'center', paddingTop: '5rem' }}>Loading statistics...</div>;

    const summaryCards = [
        { label: 'Total Flyers', value: stats?.total_flyers || 0, icon: <FileText size={24} />, color: 'var(--primary)' },
        { label: 'Total Pages', value: stats?.total_pages || 0, icon: <ImageIcon size={24} />, color: '#6366f1' },
        { label: 'Parsed Pages', value: stats?.parsed_pages || 0, icon: <CheckCircle2 size={24} />, color: 'var(--success)' },
        { label: 'Error Pages', value: stats?.error_pages || 0, icon: <AlertCircle size={24} />, color: 'var(--danger)' },
        { label: 'Extracted Items', value: stats?.total_items || 0, icon: <ShoppingBag size={24} />, color: '#f59e0b' },
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
                    <div key={idx} className="card" style={{
                        display: 'flex',
                        flexDirection: 'column',
                        alignItems: 'center',
                        justifyContent: 'center',
                        padding: '1.5rem',
                        textAlign: 'center'
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
                        <AreaChart data={stats?.history || []} margin={{ top: 10, right: 30, left: 0, bottom: 0 }}>
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
                            />
                            <Area
                                type="monotone"
                                dataKey="parsed"
                                name="Parsed Pages"
                                stroke="var(--success)"
                                strokeWidth={3}
                                fillOpacity={1}
                                fill="url(#colorParsed)"
                            />
                            <Area
                                type="monotone"
                                dataKey="errors"
                                name="Errors"
                                stroke="var(--danger)"
                                strokeWidth={3}
                                fillOpacity={1}
                                fill="url(#colorErrors)"
                            />
                        </AreaChart>
                    </ResponsiveContainer>
                </div>
            </section>
        </div>
    );
};

export default FlyerStatsPage;
