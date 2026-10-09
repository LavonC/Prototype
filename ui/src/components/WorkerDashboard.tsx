import { useState } from 'react';

export default function WorkerDashboard() {
  const [isOnline, setIsOnline] = useState(false);
  const [credibility] = useState(0.85);

  return (
    <div>
      <div className="header" style={{ borderBottom: 'none', marginBottom: '1rem' }}>
        <h2>Worker Node Status</h2>
        <button 
          className={isOnline ? 'secondary' : ''}
          onClick={() => setIsOnline(!isOnline)}
          style={{ background: isOnline ? 'rgba(239, 68, 68, 0.1)' : '', color: isOnline ? '#ef4444' : '', borderColor: isOnline ? '#ef4444' : '' }}
        >
          {isOnline ? 'Stop Volunteering' : 'Start Volunteering'}
        </button>
      </div>

      <div className="dashboard-grid">
        <div className="glass-panel">
          <h3 style={{ color: 'var(--text-secondary)', fontSize: '1rem', marginBottom: '0.5rem' }}>Status</h3>
          <div style={{ display: 'flex', alignItems: 'center', gap: '1rem' }}>
            <span className={`status-badge ${isOnline ? 'online' : 'offline'}`} style={{ fontSize: '1rem', padding: '0.5rem 1rem' }}>
              {isOnline ? 'ONLINE & READY' : 'OFFLINE'}
            </span>
          </div>
        </div>

        <div className="glass-panel">
          <h3 style={{ color: 'var(--text-secondary)', fontSize: '1rem', marginBottom: '0.5rem' }}>Credibility Score</h3>
          <div style={{ display: 'flex', alignItems: 'flex-end', gap: '0.5rem' }}>
            <span style={{ fontSize: '2rem', fontWeight: 'bold', color: '#fff' }}>{(credibility * 100).toFixed(1)}</span>
            <span style={{ color: 'var(--text-secondary)', marginBottom: '0.25rem' }}>/ 100</span>
          </div>
          <div className="progress-container">
            <div className="progress-bar" style={{ width: `${credibility * 100}%`, background: credibility > 0.8 ? 'var(--success-color)' : 'var(--accent-color)' }}></div>
          </div>
        </div>
        
        <div className="glass-panel">
          <h3 style={{ color: 'var(--text-secondary)', fontSize: '1rem', marginBottom: '0.5rem' }}>Resources Assigned</h3>
          <ul style={{ color: '#e2e8f0', listStyle: 'none', display: 'flex', flexDirection: 'column', gap: '0.5rem' }}>
            <li style={{ display: 'flex', justifyContent: 'space-between' }}>
              <span style={{ color: 'var(--text-secondary)' }}>CPU Cores</span>
              <span>4 / 8 Available</span>
            </li>
            <li style={{ display: 'flex', justifyContent: 'space-between' }}>
              <span style={{ color: 'var(--text-secondary)' }}>Memory</span>
              <span>4096 MB Limit</span>
            </li>
            <li style={{ display: 'flex', justifyContent: 'space-between' }}>
              <span style={{ color: 'var(--text-secondary)' }}>Concurrent Tasks</span>
              <span>2 Max</span>
            </li>
          </ul>
        </div>
      </div>

      {isOnline && (
        <div className="glass-panel">
          <h3 style={{ marginBottom: '1rem' }}>Active Assignments</h3>
          <div style={{ padding: '1rem', background: 'rgba(255, 255, 255, 0.02)', borderRadius: '8px', border: '1px solid var(--surface-border)' }}>
            <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: '0.5rem' }}>
              <strong>Task: task-7b8c9d</strong>
              <span className="status-badge online" style={{ background: 'transparent', border: '1px solid var(--accent-color)', color: 'var(--accent-color)' }}>Executing</span>
            </div>
            <p style={{ color: 'var(--text-secondary)', fontSize: '0.875rem' }}>Runtime: unicompute/runtime-python:3.11</p>
          </div>
        </div>
      )}
    </div>
  );
}
