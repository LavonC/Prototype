# UniCompute

**Universal P2P Task Execution Engine**

UniCompute is a highly-concurrent, distributed computing framework that turns any cluster of consumer PCs into a decentralized supercomputer. By leveraging WebRTC for peer-to-peer data streaming, gVisor for strict workload isolation, and a cloud-based Coordinator for signaling and matchmaking, UniCompute allows developers to distribute complex tasks globally.

## Features

- 🌐 **P2P Streaming**: Circumvents centralized bandwidth bottlenecks using Pion WebRTC DataChannels for direct Worker-to-Requester communication.
- 🛡️ **gVisor Isolation**: Untrusted tasks are securely sandboxed inside Docker via `runsc`, preventing host compromise.
- 🕵️ **Byzantine Fault Tolerance**: Implements Luis Sarmenta's sabotage-tolerance model using probabilistic spot-checks and dynamic credibility scoring.
- 📦 **Execution Capsules**: Automatically analyzes (Python, Node, Go, Rust, C++) and packages project dependencies into unified `.tar.gz` capsules for remote execution.
- 🔄 **Stream Resilience**: Recovers from network drops by keeping an in-memory Ring Buffer of stdout/stderr, resuming exactly where it left off on reconnect.

## Local Testing & Development

To test the entire distributed architecture locally on a single machine:

### 1. Start the Cloud Coordinator (Local Backend)
The Coordinator manages matchmaking and WebRTC signaling.
```bash
# Spin up the PostgreSQL DB and Coordinator locally
docker-compose -f deploy/docker-compose.yml up -d
```
*(This exposes the WebSocket server on `ws://localhost:8080`)*

### 2. Start a Volunteer Worker Node
The Worker daemon waits for assignments and securely executes them via gVisor.
```bash
go build -o bin/worker.exe ./cmd/worker
./bin/worker.exe
```

### 3. Desktop Dashboard (UI Wrapper)
Visualize the node status and orchestrate jobs through the React/Electron app:
```bash
cd ui && npm install && npm run build
cd ../desktop && npm install && npm start
```

### 4. Dispatch a Task (Requester CLI)
Submit a local Python/Node script to your worker node:
```bash
go build -o bin/unicompute.exe ./cmd/unicompute
./bin/unicompute.exe run ./my-python-project
```

---

## Production Setup

To create a truly global decentralized supercomputer, the **Cloud Coordinator** must be hosted on a public server (AWS, DigitalOcean, Google Cloud) so that remote developers and volunteers can discover each other.

1. **Deploy the Coordinator**:
   Deploy the `deploy/docker-compose.yml` stack to a public VM.
   ```bash
   docker-compose -f deploy/docker-compose.yml up -d
   ```
2. **Configure Domain & SSL**:
   Attach a domain (e.g., `api.unicompute.com`) and secure it with TLS (wss://). WebRTC heavily relies on secure contexts.
3. **TURN Server (Optional but Recommended)**:
   The included `docker-compose.yml` provides a `coturn` instance. This acts as a relay for WebRTC traffic when direct P2P connections fail due to restrictive NATs or firewalls.
4. **Point Clients to Production**:
   Update your Worker and Requester configurations to point to your new public WebSocket URL instead of `localhost`. Once connected, the Coordinator steps out of the way, and data flows directly Peer-to-Peer!

## Architecture

Please see `docs/architecture.md` for a comprehensive deep-dive into the Control Plane, Signaling Hub, and P2P Networking topologies.
