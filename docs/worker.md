# Worker Security & Volunteering Guide

Welcome to the UniCompute network! By running a Worker Node, you are volunteering your idle CPU and RAM to help developers process massive datasets globally. 

## How It Works

1. **Signaling**: Your worker connects to the Cloud Coordinator via WebSocket. It advertises its Hardware Profile (CPUs, RAM, GPU availability).
2. **Matchmaking**: The Coordinator matches you to a Developer (Requester) who needs your specific hardware.
3. **WebRTC**: You establish a direct P2P connection to the Developer, entirely bypassing the cloud for data transfer.
4. **Execution**: The Developer streams an `Execution Capsule` to you. You run it, and stream the logs and result back.

## The Security Model (gVisor)

We treat all Developer code as *hostile*. To ensure your personal machine remains absolutely safe, UniCompute relies on **gVisor**:

- gVisor (`runsc`) is a user-space kernel written by Google.
- When an Execution Capsule runs, it does *not* talk to your real Linux kernel.
- It is heavily isolated inside a sandbox. It has zero access to your host filesystem, network (outbound access is blocked by default), or processes.
- Memory and CPU limits are strictly enforced by the Worker Daemon's `ResourceMonitor`.

## Credibility (Byzantine Verification)

UniCompute uses a trust system to detect malicious workers (saboteurs). 

When you start volunteering, your Credibility Score is `0.5`. Periodically, the Coordinator will secretly assign you a "Spot-Check" task—a fake task where the Coordinator already knows the exact output hash.

- **If you return the correct hash**: Your credibility asymptotically increases toward `1.0`. Highly credible workers earn higher priority and better workloads.
- **If you return a fake hash**: You are flagged as a saboteur. Your credibility drops to `0.0`, and all your recently completed tasks are automatically audited, invalidated, and re-run by honest peers.
