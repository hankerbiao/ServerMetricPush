package update

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestManagerAcceptsBinaryUpdateAndPersistsStatus(t *testing.T) {
	t.Parallel()

	statusFile := filepath.Join(t.TempDir(), "update-status.json")
	manager := NewManager(Options{
		ListenAddr:   "127.0.0.1:18080",
		AllowedCIDRs: []string{"127.0.0.0/8"},
		StatusFile:   statusFile,
		WorkDir:      t.TempDir(),
		Runner: stubRunner(func(ctx context.Context, task Task) error {
			return nil
		}),
		VersionProvider: func() string { return "1.2.3" },
	})

	requestBody, err := json.Marshal(Request{
		RequestID:     "req-1",
		UpdateType:    UpdateTypeBinary,
		TargetVersion: "1.2.4",
		DownloadURL:   "http://example.com/node-push-exporter.tar.gz",
		FileName:      "node-push-exporter-1.2.4-linux-amd64.tar.gz",
		PackageType:   "tar.gz",
	})
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/internal/update", bytes.NewReader(requestBody))
	req.RemoteAddr = "127.0.0.1:34567"
	recorder := httptest.NewRecorder()

	manager.Handler().ServeHTTP(recorder, req)

	if recorder.Code != http.StatusAccepted {
		t.Fatalf("status code = %d, want %d", recorder.Code, http.StatusAccepted)
	}

	eventually(t, func() bool {
		snapshot := manager.Snapshot()
		return snapshot.Status == StatusSucceeded
	})

	data, err := os.ReadFile(statusFile)
	if err != nil {
		t.Fatalf("os.ReadFile() error = %v", err)
	}

	var persisted StatusSnapshot
	if err := json.Unmarshal(data, &persisted); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if persisted.TargetVersion != "1.2.4" {
		t.Fatalf("persisted.TargetVersion = %q, want %q", persisted.TargetVersion, "1.2.4")
	}
}

func TestManagerRejectsConcurrentUpdate(t *testing.T) {
	t.Parallel()

	blockCh := make(chan struct{})
	manager := NewManager(Options{
		ListenAddr:   "127.0.0.1:18080",
		AllowedCIDRs: []string{"127.0.0.0/8"},
		StatusFile:   filepath.Join(t.TempDir(), "update-status.json"),
		WorkDir:      t.TempDir(),
		Runner: stubRunner(func(ctx context.Context, task Task) error {
			<-blockCh
			return nil
		}),
		VersionProvider: func() string { return "1.2.3" },
	})

	first := httptest.NewRequest(http.MethodPost, "/internal/update", bytes.NewReader(binaryRequestBody(t, "req-1")))
	first.RemoteAddr = "127.0.0.1:34567"
	firstRecorder := httptest.NewRecorder()
	manager.Handler().ServeHTTP(firstRecorder, first)
	if firstRecorder.Code != http.StatusAccepted {
		t.Fatalf("first status code = %d, want %d", firstRecorder.Code, http.StatusAccepted)
	}

	eventually(t, func() bool {
		return manager.Snapshot().InProgress
	})

	second := httptest.NewRequest(http.MethodPost, "/internal/update", bytes.NewReader(binaryRequestBody(t, "req-2")))
	second.RemoteAddr = "127.0.0.1:45678"
	secondRecorder := httptest.NewRecorder()
	manager.Handler().ServeHTTP(secondRecorder, second)

	close(blockCh)

	if secondRecorder.Code != http.StatusConflict {
		t.Fatalf("second status code = %d, want %d", secondRecorder.Code, http.StatusConflict)
	}
}

func TestManagerRejectsRequestOutsideAllowedCIDRs(t *testing.T) {
	t.Parallel()

	manager := NewManager(Options{
		ListenAddr:   "127.0.0.1:18080",
		AllowedCIDRs: []string{"10.0.0.0/8"},
		StatusFile:   filepath.Join(t.TempDir(), "update-status.json"),
		WorkDir:      t.TempDir(),
		Runner: stubRunner(func(ctx context.Context, task Task) error {
			return nil
		}),
		VersionProvider: func() string { return "1.2.3" },
	})

	req := httptest.NewRequest(http.MethodPost, "/internal/update", bytes.NewReader(binaryRequestBody(t, "req-1")))
	req.RemoteAddr = "127.0.0.1:34567"
	recorder := httptest.NewRecorder()

	manager.Handler().ServeHTTP(recorder, req)

	if recorder.Code != http.StatusForbidden {
		t.Fatalf("status code = %d, want %d", recorder.Code, http.StatusForbidden)
	}
}

func TestManagerStatusEndpointReturnsLatestStatus(t *testing.T) {
	t.Parallel()

	manager := NewManager(Options{
		ListenAddr:   "127.0.0.1:18080",
		AllowedCIDRs: []string{"127.0.0.0/8"},
		StatusFile:   filepath.Join(t.TempDir(), "update-status.json"),
		WorkDir:      t.TempDir(),
		Runner: stubRunner(func(ctx context.Context, task Task) error {
			return nil
		}),
		VersionProvider: func() string { return "1.2.3" },
	})

	req := httptest.NewRequest(http.MethodPost, "/internal/update", bytes.NewReader(binaryRequestBody(t, "req-3")))
	req.RemoteAddr = "127.0.0.1:34567"
	manager.Handler().ServeHTTP(httptest.NewRecorder(), req)

	eventually(t, func() bool {
		return manager.Snapshot().Status == StatusSucceeded
	})

	statusReq := httptest.NewRequest(http.MethodGet, "/internal/update/status", nil)
	statusRecorder := httptest.NewRecorder()
	manager.Handler().ServeHTTP(statusRecorder, statusReq)

	if statusRecorder.Code != http.StatusOK {
		t.Fatalf("status code = %d, want %d", statusRecorder.Code, http.StatusOK)
	}

	var payload StatusSnapshot
	if err := json.Unmarshal(statusRecorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if payload.RequestID != "req-3" {
		t.Fatalf("payload.RequestID = %q, want %q", payload.RequestID, "req-3")
	}
}

type stubRunner func(ctx context.Context, task Task) error

func (s stubRunner) Run(ctx context.Context, task Task) error {
	return s(ctx, task)
}

func binaryRequestBody(t *testing.T, requestID string) []byte {
	t.Helper()

	body, err := json.Marshal(Request{
		RequestID:     requestID,
		UpdateType:    UpdateTypeBinary,
		TargetVersion: "1.2.4",
		DownloadURL:   "http://example.com/node-push-exporter.tar.gz",
		FileName:      "node-push-exporter-1.2.4-linux-amd64.tar.gz",
		PackageType:   "tar.gz",
	})
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	return body
}

func eventually(t *testing.T, fn func() bool) {
	t.Helper()

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if fn() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}

	t.Fatal("condition did not become true before timeout")
}
