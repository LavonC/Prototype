import { useState } from 'react';
import LiveStream from './LiveStream';

// Mock data for UI development
const mockJobs = [
  { id: 'job-1a2b3c', status: 'RUNNING', progress: 65, tasks: 4, completedTasks: 2 },
  { id: 'job-9f8e7d', status: 'COMPLETED', progress: 100, tasks: 1, completedTasks: 1 },
];

export default function RequesterDashboard() {
  const [jobs] = useState(mockJobs);
  const [selectedJob, setSelectedJob] = useState<string | null>(null);

  return (
    <div>
      <div className="header" style={{ borderBottom: 'none', marginBottom: '1rem' }}>
        <h2>My Distributed Jobs</h2>
        <button>+ Submit New Task</button>
      </div>

      <div className="dashboard-grid">
        {jobs.map((job) => (
          <div 
            key={job.id} 
            className="glass-panel" 
            style={{ cursor: 'pointer', border: selectedJob === job.id ? '1px solid var(--accent-color)' : '' }}
            onClick={() => setSelectedJob(job.id)}
          >
            <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: '1rem' }}>
              <h3 style={{ fontSize: '1.25rem' }}>{job.id}</h3>
              <span className={`status-badge ${job.status === 'COMPLETED' ? 'online' : 'pending'}`}>
                {job.status}
              </span>
            </div>
            
            <div style={{ color: 'var(--text-secondary)', marginBottom: '1rem', fontSize: '0.875rem' }}>
              Sub-tasks: {job.completedTasks} / {job.tasks} Completed
            </div>

            <div style={{ display: 'flex', justifyContent: 'space-between', fontSize: '0.875rem', marginBottom: '0.25rem' }}>
              <span>Progress</span>
              <span>{job.progress}%</span>
            </div>
            <div className="progress-container">
              <div className="progress-bar" style={{ width: `${job.progress}%` }}></div>
            </div>
          </div>
        ))}
      </div>

      {selectedJob && (
        <div className="glass-panel" style={{ marginTop: '2rem' }}>
          <h3 style={{ marginBottom: '1rem' }}>Live Stream - {selectedJob}</h3>
          <LiveStream jobId={selectedJob} />
        </div>
      )}
    </div>
  );
}
