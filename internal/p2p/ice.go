package p2p

import (
	"encoding/json"
	"strings"

	"github.com/pion/webrtc/v3"
	"github.com/unicompute/unicompute/internal/config"
	"github.com/unicompute/unicompute/internal/protocol"
)

// ParseICEServers converts standard configuration STUN/TURN URLs into Pion configuration formats.
func ParseICEServers(cfg config.P2PConfig) []webrtc.ICEServer {
	var servers []webrtc.ICEServer
	
	if len(cfg.STUNServers) > 0 {
		servers = append(servers, webrtc.ICEServer{
			URLs: cfg.STUNServers,
		})
	}
	
	for _, turn := range cfg.TURNServers {
		urls := []string{turn.URL}
		if !strings.HasPrefix(turn.URL, "turn:") && !strings.HasPrefix(turn.URL, "turns:") {
			urls[0] = "turn:" + turn.URL
		}
		
		servers = append(servers, webrtc.ICEServer{
			URLs:       urls,
			Username:   turn.Username,
			Credential: turn.Credential,
		})
	}
	
	return servers
}

// ConvertToPionCandidate converts protocol ICE payload to pion ICE candidate
func ConvertToPionCandidate(payload protocol.ICECandidatePayload) (webrtc.ICECandidateInit, error) {
	return webrtc.ICECandidateInit{
		Candidate:     payload.Candidate,
		SDPMid:        &payload.SDPMid,
		SDPMLineIndex: &payload.SDPMLineIndex,
	}, nil
}

// ExtractPionCandidate parses an ICE candidate from Pion back to protocol schema
func ExtractPionCandidate(candidate *webrtc.ICECandidate) *protocol.ICECandidatePayload {
	if candidate == nil {
		return nil
	}
	
	// Convert to JSON and parse back since Pion doesn't easily expose the raw SDP candidate string cleanly via struct access
	jsonBytes, err := json.Marshal(candidate.ToJSON())
	if err != nil {
		return nil
	}
	
	var init webrtc.ICECandidateInit
	if err := json.Unmarshal(jsonBytes, &init); err != nil {
		return nil
	}
	
	var mid string
	if init.SDPMid != nil {
		mid = *init.SDPMid
	}
	
	var mLineIndex uint16
	if init.SDPMLineIndex != nil {
		mLineIndex = *init.SDPMLineIndex
	}
	
	return &protocol.ICECandidatePayload{
		Candidate:     init.Candidate,
		SDPMid:        mid,
		SDPMLineIndex: mLineIndex,
	}
}
