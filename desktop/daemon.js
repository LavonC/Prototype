const { spawn } = require('child_process');
const path = require('path');
const fs = require('fs');
const os = require('os');

class DaemonManager {
  constructor() {
    this.workerProcess = null;
    this.requesterProcess = null;
    
    // In production, binaries would be packaged in resources. 
    // For development, we assume they are built at the project root.
    const platformExt = os.platform() === 'win32' ? '.exe' : '';
    this.workerBin = path.join(__dirname, '..', 'worker' + platformExt);
    this.requesterBin = path.join(__dirname, '..', 'unicompute' + platformExt);
  }

  startWorker() {
    if (this.workerProcess) {
      console.log('Worker daemon is already running.');
      return;
    }

    if (!fs.existsSync(this.workerBin)) {
      console.error(`Worker binary not found at ${this.workerBin}. Please build it first.`);
      return;
    }

    console.log('Starting Worker Daemon...');
    this.workerProcess = spawn(this.workerBin, [], {
      stdio: ['ignore', 'pipe', 'pipe']
    });

    this.workerProcess.stdout.on('data', (data) => {
      console.log(`[Worker] ${data}`);
    });

    this.workerProcess.stderr.on('data', (data) => {
      console.error(`[Worker Error] ${data}`);
    });

    this.workerProcess.on('close', (code) => {
      console.log(`Worker process exited with code ${code}`);
      this.workerProcess = null;
    });
  }

  stopWorker() {
    if (this.workerProcess) {
      console.log('Stopping Worker Daemon...');
      this.workerProcess.kill('SIGTERM');
      this.workerProcess = null;
    }
  }

  startRequester() {
    // Currently, requester is typically invoked as a CLI command (e.g. `unicompute run <path>`)
    // If we build a resident daemon for Requester later, it goes here.
    console.log('Requester daemon feature placeholder - usually invoked per job.');
  }
}

module.exports = new DaemonManager();
