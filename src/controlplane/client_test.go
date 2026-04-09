package controlplane

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestClient_RegisterSendsBearerTokenAndPayload(t *testing.T) {
	t.Parallel()

	var gotPayload RegisterRequest

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/agents/register" {
			t.Fatalf("path = %q, want %q", r.URL.Path, "/api/agents/register")
		}
		if err := json.NewDecoder(r.Body).Decode(&gotPayload); err != nil {
			t.Fatalf("Decode() error = %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"heartbeat_interval_seconds":30,"offline_timeout_seconds":90}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, 5*time.Second)
	err := client.Register(RegisterRequest{
		AgentID:                "agent-1",
		Hostname:               "host-01",
		Version:                "1.2.3",
		OS:                     "linux",
		Arch:                   "amd64",
		IP:                     "10.0.0.1",
		PushgatewayURL:         "http://pushgateway:9091",
		PushIntervalSeconds:    30,
		NodeExporterPort:       9100,
		NodeExporterMetricsURL: "http://127.0.0.1:9100/metrics",
		UpdateListenAddr:       "10.0.0.1:18080",
		CurrentConfigVersion:   "default",
		StartedAt:              time.Date(2026, 3, 27, 12, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	if gotPayload.AgentID != "agent-1" {
		t.Fatalf("AgentID = %q, want %q", gotPayload.AgentID, "agent-1")
	}
	if gotPayload.UpdateListenAddr != "10.0.0.1:18080" {
		t.Fatalf("UpdateListenAddr = %q, want %q", gotPayload.UpdateListenAddr, "10.0.0.1:18080")
	}
	if gotPayload.CurrentConfigVersion != "default" {
		t.Fatalf("CurrentConfigVersion = %q, want %q", gotPayload.CurrentConfigVersion, "default")
	}
}

func TestClient_HeartbeatSendsUpdateStatusFields(t *testing.T) {
	t.Parallel()

	var gotPayload HeartbeatRequest

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/agents/heartbeat" {
			t.Fatalf("path = %q, want %q", r.URL.Path, "/api/agents/heartbeat")
		}
		if err := json.NewDecoder(r.Body).Decode(&gotPayload); err != nil {
			t.Fatalf("Decode() error = %v", err)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewClient(server.URL, 5*time.Second)
	err := client.Heartbeat(HeartbeatRequest{
		AgentID:              "agent-1",
		Status:               "online",
		UpdateInProgress:     true,
		LastUpdateRequestID:  "req-1",
		LastUpdateType:       "binary_update",
		LastUpdateStatus:     "installing",
		LastUpdateTarget:     "1.2.4",
		LastUpdateError:      "",
		CurrentConfigVersion: "cfg-1",
	})
	if err != nil {
		t.Fatalf("Heartbeat() error = %v", err)
	}

	if !gotPayload.UpdateInProgress {
		t.Fatal("UpdateInProgress = false, want true")
	}
	if gotPayload.LastUpdateRequestID != "req-1" {
		t.Fatalf("LastUpdateRequestID = %q, want %q", gotPayload.LastUpdateRequestID, "req-1")
	}
	if gotPayload.LastUpdateStatus != "installing" {
		t.Fatalf("LastUpdateStatus = %q, want %q", gotPayload.LastUpdateStatus, "installing")
	}
	if gotPayload.CurrentConfigVersion != "cfg-1" {
		t.Fatalf("CurrentConfigVersion = %q, want %q", gotPayload.CurrentConfigVersion, "cfg-1")
	}
}
