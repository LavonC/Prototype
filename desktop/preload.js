const { contextBridge, ipcRenderer } = require('electron');

contextBridge.exposeInMainWorld('unicompute', {
  // Methods to expose to the React frontend
  startWorker: () => ipcRenderer.send('start-worker'),
  stopWorker: () => ipcRenderer.send('stop-worker'),
  onWorkerStatusUpdate: (callback) => ipcRenderer.on('worker-status', (_event, value) => callback(value)),
});
