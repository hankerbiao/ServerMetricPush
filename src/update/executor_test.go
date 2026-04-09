package update

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestExecutorRunBinaryUpdateReplacesBinaryAndRequestsRestart(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	binaryPath := filepath.Join(dir, "node-push-exporter")
	if err := os.WriteFile(binaryPath, []byte("old-binary"), 0o755); err != nil {
		t.Fatalf("os.WriteFile() error = %v", err)
	}

	service := &stubServiceManager{}
	executor := NewExecutor(ExecutorOptions{
		BinaryPath:           binaryPath,
		ConfigPath:           filepath.Join(dir, "config.env"),
		CurrentVersion:       "1.2.3",
		CurrentConfigVersion: "cfg-old",
		Downloader: func(ctx context.Context, url, dest string) error {
			return os.WriteFile(dest, []byte("new-binary"), 0o755)
		},
		ServiceManager:  service,
		ConfigValidator: func(path string) error { return nil },
	})

	err := executor.Run(context.Background(), Task{
		RequestID:     "req-1",
		UpdateType:    UpdateTypeBinary,
		TargetVersion: "1.2.4",
		DownloadURL:   "http://example.com/node-push-exporter",
		FileName:      "node-push-exporter-1.2.4-linux-amd64",
		PackageType:   "binary",
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	data, err := os.ReadFile(binaryPath)
	if err != nil {
		t.Fatalf("os.ReadFile() error = %v", err)
	}
	if string(data) != "new-binary" {
		t.Fatalf("binary content = %q, want %q", string(data), "new-binary")
	}
	if service.restartCalls != 1 {
		t.Fatalf("restartCalls = %d, want %d", service.restartCalls, 1)
	}
}

func TestExecutorRunConfigUpdateWritesConfigAndRestartsService(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.env")
	if err := os.WriteFile(configPath, []byte("pushgateway.url=http://old"), 0o644); err != nil {
		t.Fatalf("os.WriteFile() error = %v", err)
	}

	service := &stubServiceManager{}
	executor := NewExecutor(ExecutorOptions{
		BinaryPath:           filepath.Join(dir, "node-push-exporter"),
		ConfigPath:           configPath,
		CurrentVersion:       "1.2.3",
		CurrentConfigVersion: "cfg-old",
		Downloader:           func(ctx context.Context, url, dest string) error { return nil },
		ServiceManager:       service,
		ConfigValidator: func(path string) error {
			data, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			if string(data) == "" {
				return fmt.Errorf("empty config")
			}
			return nil
		},
	})

	err := executor.Run(context.Background(), Task{
		RequestID:     "req-2",
		UpdateType:    UpdateTypeConfig,
		ConfigContent: "pushgateway.url=http://new\npushgateway.job=node",
		ConfigVersion: "cfg-new",
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("os.ReadFile() error = %v", err)
	}
	if string(data) != "pushgateway.url=http://new\npushgateway.job=node" {
		t.Fatalf("config content = %q", string(data))
	}
	if service.restartCalls != 1 {
		t.Fatalf("restartCalls = %d, want %d", service.restartCalls, 1)
	}
}

func TestExecutorRunRestoresBackupWhenRestartFails(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	binaryPath := filepath.Join(dir, "node-push-exporter")
	if err := os.WriteFile(binaryPath, []byte("old-binary"), 0o755); err != nil {
		t.Fatalf("os.WriteFile() error = %v", err)
	}

	executor := NewExecutor(ExecutorOptions{
		BinaryPath:           binaryPath,
		ConfigPath:           filepath.Join(dir, "config.env"),
		CurrentVersion:       "1.2.3",
		CurrentConfigVersion: "cfg-old",
		Downloader: func(ctx context.Context, url, dest string) error {
			return os.WriteFile(dest, []byte("new-binary"), 0o755)
		},
		ServiceManager:  &stubServiceManager{restartErr: fmt.Errorf("restart failed")},
		ConfigValidator: func(path string) error { return nil },
	})

	err := executor.Run(context.Background(), Task{
		RequestID:     "req-3",
		UpdateType:    UpdateTypeBinary,
		TargetVersion: "1.2.4",
		DownloadURL:   "http://example.com/node-push-exporter",
		FileName:      "node-push-exporter-1.2.4-linux-amd64",
		PackageType:   "binary",
	})
	if err == nil {
		t.Fatal("Run() error = nil, want restart failure")
	}

	data, readErr := os.ReadFile(binaryPath)
	if readErr != nil {
		t.Fatalf("os.ReadFile() error = %v", readErr)
	}
	if string(data) != "old-binary" {
		t.Fatalf("binary content = %q, want rollback to %q", string(data), "old-binary")
	}
}

type stubServiceManager struct {
	restartCalls int
	restartErr   error
}

func (s *stubServiceManager) Restart(ctx context.Context) error {
	s.restartCalls++
	return s.restartErr
}
