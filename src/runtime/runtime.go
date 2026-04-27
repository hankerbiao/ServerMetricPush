package runtime

import (
	"sync"
	"time"

	"node-push-exporter/src/controlplane"
)

// State 管理用于控制面报告的运行时状态。
type State struct {
	mu                sync.Mutex
	LastPushAt        time.Time
	LastPushSuccessAt time.Time
	LastPushErrorAt   time.Time
	PushFailCount     int
	LastError         string
	NodeExporterUp    bool
}

// NewState 创建一个新的运行时状态。
func NewState() *State {
	return &State{NodeExporterUp: true}
}

func (s *State) RecordFailure(err error, nodeExporterUp bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UTC()
	s.LastPushAt = now
	s.LastPushErrorAt = now
	s.PushFailCount++
	s.LastError = err.Error()
	s.NodeExporterUp = nodeExporterUp
}

func (s *State) RecordFetchFailure(err error) {
	s.RecordFailure(err, false)
}

func (s *State) RecordPushFailure(err error) {
	s.RecordFailure(err, true)
}

func (s *State) RecordPushSuccess() {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UTC()
	s.LastPushAt = now
	s.LastPushSuccessAt = now
	s.PushFailCount = 0
	s.LastError = ""
	s.NodeExporterUp = true
}

func (s *State) Snapshot(agentID string) controlplane.HeartbeatRequest {
	s.mu.Lock()
	defer s.mu.Unlock()

	status := "online"
	if !s.NodeExporterUp || s.PushFailCount > 0 || s.LastError != "" {
		status = "degraded"
	}

	return controlplane.HeartbeatRequest{
		AgentID:           agentID,
		Status:            status,
		LastError:         s.LastError,
		LastPushAt:        cloneTimePointer(s.LastPushAt),
		LastPushSuccessAt: cloneTimePointer(s.LastPushSuccessAt),
		LastPushErrorAt:   cloneTimePointer(s.LastPushErrorAt),
		PushFailCount:     s.PushFailCount,
		NodeExporterUp:    s.NodeExporterUp,
	}
}

func cloneTimePointer(value time.Time) *time.Time {
	if value.IsZero() {
		return nil
	}
	cloned := value
	return &cloned
}
