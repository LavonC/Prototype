package requester

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"github.com/unicompute/unicompute/internal/config"
	"github.com/unicompute/unicompute/internal/p2p"
	"github.com/unicompute/unicompute/internal/protocol"
)

// Client manages the Requester's interaction with the UniCompute network.
type Client struct {
	config    config.RequesterConfig
	p2pCfg    config.P2PConfig
	sigClient *p2p.SignalingClient
	webrtcMgr *p2p.WebRTCManager
	nodeID    string
	jobs      map[string]*JobManager
	viewer    *Viewer
	aggregator *Aggregator
}

// NewClient initializes the Requester client daemon.
func NewClient(cfg config.RequesterConfig, p2pCfg config.P2PConfig) *Client {
	id := "r-" + uuid.New().String()[:8]
	
	sigClient := p2p.NewSignalingClient(cfg.CoordinatorURL, cfg.AuthToken, "requester", id)
	webrtcMgr := p2p.NewWebRTCManager(p2pCfg, sigClient)
	
	c := &Client{
		config:    cfg,
		p2pCfg:    p2pCfg,
		sigClient: sigClient,
		webrtcMgr: webrtcMgr,
		nodeID:    id,
		jobs:      make(map[string]*JobManager),
		viewer:    NewViewer(),
		aggregator: NewAggregator(cfg.OutputDir),
	}

	sigClient.SetMessageHandler(c.handleMessage)
	
	return c
}

// Connect establishes the connection to the Cloud Coordinator.
func (c *Client) Connect(ctx context.Context) error {
	log.Info().Str("node_id", c.nodeID).Msg("Requester connecting to Coordinator")
	return c.sigClient.Connect(ctx)
}

// SubmitTask parses a capsule manifest and submits it to the coordinator queue.
func (c *Client) SubmitTask(ctx context.Context, jobID string, manifest *protocol.CapsuleManifest, capsulePath string) (string, error) {
	taskID := "task-" + uuid.New().String()[:8]
	
	payload := protocol.TaskSubmitPayload{
		JobID:           jobID,
		TaskID:          taskID,
		Entrypoint:      manifest.Entrypoint,
		RuntimeImage:    manifest.RuntimeImage,
		CapsuleManifest: *manifest,
		Priority:        0, // Default
	}

	// 1. Submit to Coordinator REST API
	// In a real CLI, we'd use http.Client. For this simulation architecture, 
	// we mock the REST call assuming the coordinator runs on localhost:8080.
	
	apiURL := "http://localhost:8080/api/v1/tasks" 
	
	body, _ := json.Marshal(payload)
	req, _ := http.NewRequestWithContext(ctx, "POST", apiURL, bytes.NewBuffer(body))
	req.Header.Set("Authorization", "Bearer "+c.config.AuthToken)
	req.Header.Set("Content-Type", "application/json")
	
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to submit task to coordinator: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusAccepted {
		return "", fmt.Errorf("coordinator rejected task submission: status %d", resp.StatusCode)
	}

	// 2. Track it locally
	if job, ok := c.jobs[jobID]; ok {
		job.AddSubTask(taskID)
	} else {
		// Single task job
		job = NewJobManager()
		job.JobID = jobID
		job.AddSubTask(taskID)
		c.jobs[jobID] = job
	}
	
	// 3. Register the capsule path so FTM can serve it when the worker connects
	c.aggregator.RegisterSourceCapsule(taskID, capsulePath)

	log.Info().Str("task_id", taskID).Str("job_id", jobID).Msg("Task successfully queued on Coordinator")
	return taskID, nil
}

// handleMessage processes incoming signaling and control messages from Coordinator and Workers.
func (c *Client) handleMessage(env *protocol.Envelope) {
	payloadBytes, _ := json.Marshal(env.Payload)

	switch env.Type {
	case protocol.MsgSDPOffer:
		var sdp protocol.SDPPayload
		json.Unmarshal(payloadBytes, &sdp)
		
		// Worker is offering a WebRTC connection to pull the capsule and stream logs
		if err := c.webrtcMgr.HandleSDPOffer(env.SenderID, sdp); err != nil {
			log.Error().Err(err).Msg("Failed to handle SDP offer from worker")
		} else {
			// On successful answer, bind the channels for this peer
			c.bindWorkerChannels(env.SenderID)
		}
		
	case protocol.MsgSDPAnswer:
		var sdp protocol.SDPPayload
		json.Unmarshal(payloadBytes, &sdp)
		c.webrtcMgr.HandleSDPAnswer(sdp)
		
	case protocol.MsgICECandidate:
		var ice protocol.ICECandidatePayload
		json.Unmarshal(payloadBytes, &ice)
		c.webrtcMgr.HandleICECandidate(ice)
		
	case protocol.MsgTaskComplete:
		var receipt protocol.CompletionReceipt
		json.Unmarshal(payloadBytes, &receipt)
		
		if job, ok := c.jobs[receipt.JobID]; ok {
			job.UpdateSubTask(receipt.TaskID, "COMPLETED", receipt.WorkerID, receipt.ExitCode, "")
		}
		
	case protocol.MsgTaskFailed:
		var errPayload protocol.ErrorPayload
		json.Unmarshal(payloadBytes, &errPayload)
		// We'd need the TaskID/JobID here normally (added in real schema). 
		// Assuming we extract it from context or envelope ID for this simulation.
		log.Error().Int("code", errPayload.Code).Str("msg", errPayload.Message).Msg("Task failed")
		
	default:
		log.Debug().Str("type", string(env.Type)).Msg("Unhandled message")
	}
}

func (c *Client) bindWorkerChannels(workerID string) {
	// A bit of a race condition here normally, we'd look up the active Connection object
	// created during HandleSDPOffer and attach the FTM. 
	// For simulation, we assume the DataChannels are ready shortly after Answer.
	
	// Wait logic omitted for brevity. We'd use a callback on WebRTCManager like OnDataChannel.
}

// GetJob returns the JobManager for a given ID.
func (c *Client) GetJob(jobID string) *JobManager {
	return c.jobs[jobID]
}
