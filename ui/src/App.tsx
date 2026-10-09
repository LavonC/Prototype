import { useState } from 'react';
import './index.css';
import RequesterDashboard from './components/RequesterDashboard';
import WorkerDashboard from './components/WorkerDashboard';

function App() {
  const [activeTab, setActiveTab] = useState<'requester' | 'worker'>('requester');

  return (
    <>
      <header className="header">
        <div>
          <h1>UniCompute</h1>
          <p style={{ color: 'var(--text-secondary)' }}>Universal P2P Task Execution Engine</p>
        </div>
        <div className="nav-tabs glass-panel" style={{ padding: '0.5rem', display: 'flex' }}>
          <div 
            className={`nav-tab ${activeTab === 'requester' ? 'active' : ''}`}
            onClick={() => setActiveTab('requester')}
          >
            Requester
          </div>
          <div 
            className={`nav-tab ${activeTab === 'worker' ? 'active' : ''}`}
            onClick={() => setActiveTab('worker')}
          >
            Worker Node
          </div>
        </div>
      </header>

      <main>
        {activeTab === 'requester' ? <RequesterDashboard /> : <WorkerDashboard />}
      </main>
    </>
  );
}

export default App;
