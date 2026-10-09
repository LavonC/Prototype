# Architecture Specification

## Topology

UniCompute utilizes a hybrid topology:
1. **Control Plane (Star Topology)**: All nodes (Workers and Requesters) maintain lightweight WebSockets to the centralized `Coordinator`.
2. **Data Plane (Mesh/P2P)**: Data is transferred *exclusively* point-to-point via WebRTC DataChannels.

### 1. The Cloud Coordinator

The Coordinator serves three main functions:
- **Registry**: Maintains a PostgreSQL database of all active workers and their hardware capabilities.
- **Matchmaker**: Pairs pending Tasks with available Workers based on CPU/RAM requirements.
- **Signaling Hub**: Relays WebRTC `SDP Offers`, `SDP Answers`, and `ICE Candidates` between peers to broker the initial connection.

### 2. Peer-to-Peer Networking (WebRTC)

To avoid crippling bandwidth costs on the central server, UniCompute streams all execution payloads and results directly between the Requester and the Worker using `pion/webrtc`.

- **Channel A (Streams & Control)**: Multiplexed channel for low-latency, ordered delivery of stdout/stderr logs and out-of-band signals (`RESUME_STREAM`).
- **Channel B (Artifacts)**: Dedicated binary channel for transferring the heavily compressed `.tar.gz` Execution Capsules and the final resultant outputs.

### 3. Fault Tolerance

- **Network Drops**: If the WebRTC connection drops, the Worker caches execution logs in an in-memory `RingBuffer`. When the connection is re-established via the Coordinator, the logs are immediately replayed.
- **Worker Failure**: The Coordinator expects a WebSocket heartbeat every 2 seconds. If a worker misses 3 heartbeats (6 seconds), its active tasks are forcefully stripped and re-queued to the global task pool.
- **Malicious Results**: Handled via Sarmenta's Sabotage-Tolerance model (Spot Checks).
