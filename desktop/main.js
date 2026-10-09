const { app, BrowserWindow, ipcMain } = require('electron');
const path = require('path');
const daemonManager = require('./daemon');

// Handle creating/removing shortcuts on Windows when installing/uninstalling.
if (require('electron-squirrel-startup')) {
  app.quit();
}

const createWindow = () => {
  // Create the browser window.
  const mainWindow = new BrowserWindow({
    width: 1200,
    height: 800,
    minWidth: 900,
    minHeight: 600,
    webPreferences: {
      preload: path.join(__dirname, 'preload.js'),
      nodeIntegration: false, // Security: Keep false, use contextBridge
      contextIsolation: true
    },
    autoHideMenuBar: true,
    titleBarStyle: 'hiddenInset',
    backgroundColor: '#0d0f12' // Match the UI's dark background to avoid white flashing on load
  });

  // Check if we are running in dev mode or production
  // Assuming the React UI is built into ../ui/dist
  const isDev = !app.isPackaged && process.env.NODE_ENV !== 'production';
  
  if (isDev) {
    // In dev mode, we would load the Vite dev server URL.
    // For this boilerplate, we'll try to load the dist folder, 
    // or fallback to localhost if it's running.
    // Assuming UI was built using `npm run build` as instructed earlier.
    mainWindow.loadFile(path.join(__dirname, '..', 'ui', 'dist', 'index.html'))
      .catch(() => {
         console.log("Failed to load dist/index.html, trying dev server...");
         mainWindow.loadURL('http://localhost:5173');
      });
  } else {
    mainWindow.loadFile(path.join(__dirname, '..', 'ui', 'dist', 'index.html'));
  }

  // Open the DevTools automatically in dev mode.
  // if (isDev) mainWindow.webContents.openDevTools();
};

// This method will be called when Electron has finished initialization
app.on('ready', () => {
  createWindow();

  // Set up IPC listeners
  ipcMain.on('start-worker', (event) => {
    daemonManager.startWorker();
    event.reply('worker-status', { online: true });
  });

  ipcMain.on('stop-worker', (event) => {
    daemonManager.stopWorker();
    event.reply('worker-status', { online: false });
  });
});

// Quit when all windows are closed, except on macOS
app.on('window-all-closed', () => {
  if (process.platform !== 'darwin') {
    app.quit();
  }
});

app.on('will-quit', () => {
  // Ensure daemons are killed when the app shuts down
  daemonManager.stopWorker();
});

app.on('activate', () => {
  if (BrowserWindow.getAllWindows().length === 0) {
    createWindow();
  }
});
