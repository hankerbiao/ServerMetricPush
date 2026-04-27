package runtime

import (
	"errors"
	"testing"
)

func TestState_RecordFetchFailureMarksExporterDown(t *testing.T) {
	state := NewState()

	state.RecordFetchFailure(errors.New("fetch failed"))

	snapshot := state.Snapshot("agent-1")
	if snapshot.Status != "degraded" {
		t.Fatalf("Snapshot().Status = %q, want degraded", snapshot.Status)
	}
	if snapshot.NodeExporterUp {
		t.Fatal("Snapshot().NodeExporterUp = true, want false")
	}
	if snapshot.PushFailCount != 1 {
		t.Fatalf("Snapshot().PushFailCount = %d, want 1", snapshot.PushFailCount)
	}
	if snapshot.LastPushErrorAt == nil {
		t.Fatal("Snapshot().LastPushErrorAt = nil, want timestamp")
	}
}

func TestState_RecordPushSuccessResetsFailureState(t *testing.T) {
	state := NewState()
	state.RecordPushFailure(errors.New("push failed"))

	state.RecordPushSuccess()

	snapshot := state.Snapshot("agent-1")
	if snapshot.Status != "online" {
		t.Fatalf("Snapshot().Status = %q, want online", snapshot.Status)
	}
	if snapshot.PushFailCount != 0 {
		t.Fatalf("Snapshot().PushFailCount = %d, want 0", snapshot.PushFailCount)
	}
	if snapshot.LastError != "" {
		t.Fatalf("Snapshot().LastError = %q, want empty", snapshot.LastError)
	}
	if snapshot.LastPushSuccessAt == nil {
		t.Fatal("Snapshot().LastPushSuccessAt = nil, want timestamp")
	}
}
