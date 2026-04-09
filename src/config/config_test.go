package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoad_ParsesCompleteConfig(t *testing.T) {
	t.Parallel()

	path := writeConfigFile(t, `
pushgateway.url=http://pushgateway:9091
pushgateway.job=node-prod
pushgateway.instance=host-01
pushgateway.interval=30
pushgateway.timeout=15
node_exporter.path=/usr/local/bin/node_exporter
node_exporter.port=9200
node_exporter.metrics_url=http://127.0.0.1:9200/metrics
hardware.enabled=false
hardware.timeout=12
hardware.include_serials=false
hardware.include_virtual_devices=true
`)

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.Pushgateway.URL != "http://pushgateway:9091" {
		t.Fatalf("Pushgateway.URL = %q, want %q", cfg.Pushgateway.URL, "http://pushgateway:9091")
	}
	if cfg.Pushgateway.Job != "node-prod" {
		t.Fatalf("Pushgateway.Job = %q, want %q", cfg.Pushgateway.Job, "node-prod")
	}
	if cfg.Pushgateway.Instance != "host-01" {
		t.Fatalf("Pushgateway.Instance = %q, want %q", cfg.Pushgateway.Instance, "host-01")
	}
	if cfg.Pushgateway.Interval != 30 {
		t.Fatalf("Pushgateway.Interval = %d, want %d", cfg.Pushgateway.Interval, 30)
	}
	if cfg.Pushgateway.Timeout != 15 {
		t.Fatalf("Pushgateway.Timeout = %d, want %d", cfg.Pushgateway.Timeout, 15)
	}
	if cfg.NodeExporter.Path != "/usr/local/bin/node_exporter" {
		t.Fatalf("NodeExporter.Path = %q, want %q", cfg.NodeExporter.Path, "/usr/local/bin/node_exporter")
	}
	if cfg.NodeExporter.Port != 9200 {
		t.Fatalf("NodeExporter.Port = %d, want %d", cfg.NodeExporter.Port, 9200)
	}
	if cfg.NodeExporter.MetricsURL != "http://127.0.0.1:9200/metrics" {
		t.Fatalf("NodeExporter.MetricsURL = %q, want %q", cfg.NodeExporter.MetricsURL, "http://127.0.0.1:9200/metrics")
	}
	if cfg.Hardware.Enabled {
		t.Fatalf("Hardware.Enabled = %v, want false", cfg.Hardware.Enabled)
	}
	if cfg.Hardware.Timeout != 12 {
		t.Fatalf("Hardware.Timeout = %d, want %d", cfg.Hardware.Timeout, 12)
	}
	if cfg.Hardware.IncludeSerials {
		t.Fatalf("Hardware.IncludeSerials = %v, want false", cfg.Hardware.IncludeSerials)
	}
	if !cfg.Hardware.IncludeVirtualDevices {
		t.Fatalf("Hardware.IncludeVirtualDevices = %v, want true", cfg.Hardware.IncludeVirtualDevices)
	}
}

func TestLoad_ReturnsErrorWhenRequiredFieldMissing(t *testing.T) {
	t.Parallel()

	path := writeConfigFile(t, `
pushgateway.url=http://pushgateway:9091
pushgateway.job=node-prod
pushgateway.interval=30
pushgateway.timeout=15
node_exporter.path=/usr/local/bin/node_exporter
node_exporter.port=9200
`)

	_, err := Load(path)
	if err == nil {
		t.Fatal("Load() error = nil, want missing required field error")
	}
	if !strings.Contains(err.Error(), "node_exporter.metrics_url") {
		t.Fatalf("Load() error = %q, want missing node_exporter.metrics_url", err)
	}
}

func TestLoad_ParsesControlPlaneConfigWhenPresent(t *testing.T) {
	t.Parallel()

	path := writeConfigFile(t, `
pushgateway.url=http://pushgateway:9091
pushgateway.job=node-prod
pushgateway.instance=host-01
pushgateway.interval=30
pushgateway.timeout=15
node_exporter.path=/usr/local/bin/node_exporter
node_exporter.port=9200
node_exporter.metrics_url=http://127.0.0.1:9200/metrics
control_plane.url=http://control-plane:8080
control_plane.heartbeat_interval=30
update.enabled=true
update.listen_addr=10.0.0.5:18080
update.allowed_cidrs=10.0.0.0/8,192.168.0.0/16
update.status_file=/var/lib/node-push-exporter/update-status.json
update.work_dir=/var/lib/node-push-exporter/update-work
`)

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.ControlPlane.URL != "http://control-plane:8080" {
		t.Fatalf("ControlPlane.URL = %q, want %q", cfg.ControlPlane.URL, "http://control-plane:8080")
	}
	if cfg.ControlPlane.HeartbeatInterval != 30 {
		t.Fatalf("ControlPlane.HeartbeatInterval = %d, want %d", cfg.ControlPlane.HeartbeatInterval, 30)
	}
	if !cfg.Update.Enabled {
		t.Fatalf("Update.Enabled = %v, want true", cfg.Update.Enabled)
	}
	if cfg.Update.ListenAddr != "10.0.0.5:18080" {
		t.Fatalf("Update.ListenAddr = %q, want %q", cfg.Update.ListenAddr, "10.0.0.5:18080")
	}
	if len(cfg.Update.AllowedCIDRs) != 2 {
		t.Fatalf("Update.AllowedCIDRs length = %d, want %d", len(cfg.Update.AllowedCIDRs), 2)
	}
	if cfg.Update.StatusFile != "/var/lib/node-push-exporter/update-status.json" {
		t.Fatalf("Update.StatusFile = %q, want %q", cfg.Update.StatusFile, "/var/lib/node-push-exporter/update-status.json")
	}
	if cfg.Update.WorkDir != "/var/lib/node-push-exporter/update-work" {
		t.Fatalf("Update.WorkDir = %q, want %q", cfg.Update.WorkDir, "/var/lib/node-push-exporter/update-work")
	}
}

func TestLoad_ReturnsErrorWhenControlPlaneConfigIncomplete(t *testing.T) {
	t.Parallel()

	path := writeConfigFile(t, `
pushgateway.url=http://pushgateway:9091
pushgateway.job=node-prod
pushgateway.instance=host-01
pushgateway.interval=30
pushgateway.timeout=15
node_exporter.path=/usr/local/bin/node_exporter
node_exporter.port=9200
node_exporter.metrics_url=http://127.0.0.1:9200/metrics
control_plane.url=http://control-plane:8080
`)

	_, err := Load(path)
	if err == nil {
		t.Fatal("Load() error = nil, want incomplete control plane config error")
	}
	if !strings.Contains(err.Error(), "control_plane.heartbeat_interval") {
		t.Fatalf("Load() error = %q, want control_plane heartbeat validation error", err)
	}
}

func TestLoad_AppliesHardwareDefaultsWhenUnset(t *testing.T) {
	t.Parallel()

	path := writeConfigFile(t, `
pushgateway.url=http://pushgateway:9091
pushgateway.job=node-prod
pushgateway.interval=30
pushgateway.timeout=15
node_exporter.path=/usr/local/bin/node_exporter
node_exporter.port=9200
node_exporter.metrics_url=http://127.0.0.1:9200/metrics
`)

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if !cfg.Hardware.Enabled {
		t.Fatalf("Hardware.Enabled = %v, want true", cfg.Hardware.Enabled)
	}
	if cfg.Hardware.Timeout != 5 {
		t.Fatalf("Hardware.Timeout = %d, want %d", cfg.Hardware.Timeout, 5)
	}
	if !cfg.Hardware.IncludeSerials {
		t.Fatalf("Hardware.IncludeSerials = %v, want true", cfg.Hardware.IncludeSerials)
	}
	if cfg.Hardware.IncludeVirtualDevices {
		t.Fatalf("Hardware.IncludeVirtualDevices = %v, want false", cfg.Hardware.IncludeVirtualDevices)
	}
}

func writeConfigFile(t *testing.T, content string) string {
	t.Helper()

	dir := t.TempDir()
	path := filepath.Join(dir, "config.env")
	if err := os.WriteFile(path, []byte(strings.TrimSpace(content)), 0o644); err != nil {
		t.Fatalf("os.WriteFile() error = %v", err)
	}

	return path
}
