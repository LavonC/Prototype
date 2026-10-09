import { useEffect, useState } from 'react';

export default function LiveStream({ jobId }: { jobId: string }) {
  const [logs, setLogs] = useState<{ id: number; prefix: string; text: string; isError: boolean }[]>([]);

  useEffect(() => {
    // In a real app, this would connect to the local Requester WebSocket or SSE to get multiplexed WebRTC streams.
    // Simulating incoming logs:
    setLogs([
      { id: 1, prefix: '[task-1a]', text: 'Unpacking capsule...', isError: false },
      { id: 2, prefix: '[task-1a]', text: 'Starting execution environment', isError: false },
      { id: 3, prefix: '[task-1b]', text: 'Unpacking capsule...', isError: false },
      { id: 4, prefix: '[task-1a]', text: 'Processing chunk 1/100', isError: false },
      { id: 5, prefix: '[task-1b]', text: 'Processing chunk 50/100', isError: false },
    ]);
  }, [jobId]);

  return (
    <div className="terminal-viewer">
      {logs.map((log) => (
        <div key={log.id} className="terminal-line" style={{ color: log.isError ? '#bf616a' : '' }}>
          <span className="terminal-prefix">{log.prefix}</span>
          <span>{log.text}</span>
        </div>
      ))}
      <div className="terminal-line" style={{ color: '#88c0d0', opacity: 0.7, marginTop: '1rem' }}>
        &gt; Awaiting further output...
      </div>
    </div>
  );
}
