package main

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"node-push-exporter/src/pusher"
)

type stubGPUMetricsCollector struct {
	metrics string
	err     error
}

func (s stubGPUMetricsCollector) Collect() (string, error) {
	return s.metrics, s.err
}

type stubHardwareMetricsCollector struct {
	metrics string
	err     error
}

func (s stubHardwareMetricsCollector) Collect() (string, error) {
	return s.metrics, s.err
}

func TestDefaultConfigPath(t *testing.T) {
	if defaultConfigPath != "./config.yml" {
		t.Fatalf("defaultConfigPath = %q, want %q", defaultConfigPath, "./config.yml")
	}
}

func TestResolveNodeExporterPath_UsesCurrentDirectoryBinary(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("os.Getwd() error = %v", err)
	}

	tempDir := t.TempDir()
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("os.Chdir() error = %v", err)
	}
	defer func() {
		if err := os.Chdir(wd); err != nil {
			t.Fatalf("restore working directory error = %v", err)
		}
	}()

	localBinary := filepath.Join(tempDir, "node_exporter")
	if err := os.WriteFile(localBinary, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatalf("os.WriteFile() error = %v", err)
	}

	resolvedPath, err := resolveNodeExporterPath("node_exporter")
	if err != nil {
		t.Fatalf("resolveNodeExporterPath() error = %v", err)
	}

	resolvedInfo, err := os.Stat(resolvedPath)
	if err != nil {
		t.Fatalf("os.Stat(resolvedPath) error = %v", err)
	}
	localInfo, err := os.Stat(localBinary)
	if err != nil {
		t.Fatalf("os.Stat(localBinary) error = %v", err)
	}
	if !os.SameFile(resolvedInfo, localInfo) {
		t.Fatalf("resolveNodeExporterPath() = %q, want same file as %q", resolvedPath, localBinary)
	}
}

func TestStartNodeExporter_ReturnsErrorWhenProcessExitsImmediately(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("os.Getwd() error = %v", err)
	}

	tempDir := t.TempDir()
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("os.Chdir() error = %v", err)
	}
	defer func() {
		if err := os.Chdir(wd); err != nil {
			t.Fatalf("restore working directory error = %v", err)
		}
	}()

	localBinary := filepath.Join(tempDir, "node_exporter")
	script := "#!/bin/sh\n" +
		"echo 'mock node_exporter failed to start' >&2\n" +
		"exit 1\n"
	if err := os.WriteFile(localBinary, []byte(script), 0o755); err != nil {
		t.Fatalf("os.WriteFile() error = %v", err)
	}

	_, err = startNodeExporter("node_exporter", 9100)
	if err == nil {
		t.Fatal("startNodeExporter() error = nil, want startup failure")
	}
	if !strings.Contains(err.Error(), "mock node_exporter failed to start") {
		t.Fatalf("startNodeExporter() error = %q, want stderr output", err)
	}
}

func TestEffectivePushInstance_PrefersConfiguredValue(t *testing.T) {
	if got := effectivePushInstance("custom-instance"); got != "custom-instance" {
		t.Fatalf("effectivePushInstance() = %q, want %q", got, "custom-instance")
	}
}

func TestEffectivePushInstance_FallsBackWhenEmpty(t *testing.T) {
	got := effectivePushInstance("")
	if got == "" {
		t.Fatal("effectivePushInstance() = empty, want detected ip or hostname")
	}
}

func TestFetchAndPush_AppendsGPUMetricsWhenCollectionSucceeds(t *testing.T) {
	var (
		mu          sync.Mutex
		pushedBody  string
		requestSeen bool
	)

	pushgateway := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("io.ReadAll(r.Body) error = %v", err)
		}

		mu.Lock()
		defer mu.Unlock()
		requestSeen = true
		pushedBody = string(body)
		w.WriteHeader(http.StatusOK)
	}))
	defer pushgateway.Close()

	metricsServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "node_cpu_seconds_total 123\n")
	}))
	defer metricsServer.Close()

	p := pusher.NewPusher(pushgateway.URL)
	err := fetchAndPush(metricsServer.URL, p, nil, stubGPUMetricsCollector{
		metrics: "gpu_up 1\n",
	}, nil)
	if err != nil {
		t.Fatalf("fetchAndPush() error = %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	if !requestSeen {
		t.Fatal("Pushgateway did not receive any request")
	}
	if pushedBody != "node_cpu_seconds_total 123\ngpu_up 1\n" {
		t.Fatalf("pushed body = %q, want merged node and gpu metrics", pushedBody)
	}
}

func TestFetchAndPush_PushesNodeMetricsWhenGPUCollectionFails(t *testing.T) {
	var (
		mu          sync.Mutex
		pushedBody  string
		requestSeen bool
	)

	pushgateway := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("io.ReadAll(r.Body) error = %v", err)
		}

		mu.Lock()
		defer mu.Unlock()
		requestSeen = true
		pushedBody = string(body)
		w.WriteHeader(http.StatusOK)
	}))
	defer pushgateway.Close()

	metricsServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "node_load1 0.42\n")
	}))
	defer metricsServer.Close()

	p := pusher.NewPusher(pushgateway.URL)
	err := fetchAndPush(metricsServer.URL, p, nil, stubGPUMetricsCollector{
		err: fmt.Errorf("gpu unavailable"),
	}, nil)
	if err != nil {
		t.Fatalf("fetchAndPush() error = %v, want nil when gpu collection fails", err)
	}

	mu.Lock()
	defer mu.Unlock()
	if !requestSeen {
		t.Fatal("Pushgateway did not receive any request")
	}
	if pushedBody != "node_load1 0.42\n" {
		t.Fatalf("pushed body = %q, want only node metrics when gpu collection fails", pushedBody)
	}
}

func TestFetchAndPush_AppendsHardwareMetricsWhenCollectionSucceeds(t *testing.T) {
	var (
		mu          sync.Mutex
		pushedBody  string
		requestSeen bool
	)

	pushgateway := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("io.ReadAll(r.Body) error = %v", err)
		}

		mu.Lock()
		defer mu.Unlock()
		requestSeen = true
		pushedBody = string(body)
		w.WriteHeader(http.StatusOK)
	}))
	defer pushgateway.Close()

	metricsServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "node_memory_MemAvailable_bytes 42\n")
	}))
	defer metricsServer.Close()

	p := pusher.NewPusher(pushgateway.URL)
	err := fetchAndPush(metricsServer.URL, p, nil, stubGPUMetricsCollector{
		metrics: "gpu_up 1\n",
	}, stubHardwareMetricsCollector{
		metrics: "node_hardware_host_info 1\n",
	})
	if err != nil {
		t.Fatalf("fetchAndPush() error = %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	if !requestSeen {
		t.Fatal("Pushgateway did not receive any request")
	}
	if pushedBody != "node_memory_MemAvailable_bytes 42\ngpu_up 1\nnode_hardware_host_info 1\n" {
		t.Fatalf("pushed body = %q, want merged node, gpu and hardware metrics", pushedBody)
	}
}

func TestFetchAndPush_PushesOtherMetricsWhenHardwareCollectionFails(t *testing.T) {
	var (
		mu          sync.Mutex
		pushedBody  string
		requestSeen bool
	)

	pushgateway := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("io.ReadAll(r.Body) error = %v", err)
		}

		mu.Lock()
		defer mu.Unlock()
		requestSeen = true
		pushedBody = string(body)
		w.WriteHeader(http.StatusOK)
	}))
	defer pushgateway.Close()

	metricsServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "node_load5 1.5\n")
	}))
	defer metricsServer.Close()

	p := pusher.NewPusher(pushgateway.URL)
	err := fetchAndPush(metricsServer.URL, p, nil, stubGPUMetricsCollector{
		metrics: "gpu_up 1\n",
	}, stubHardwareMetricsCollector{
		err: fmt.Errorf("hardware unavailable"),
	})
	if err != nil {
		t.Fatalf("fetchAndPush() error = %v, want nil when hardware collection fails", err)
	}

	mu.Lock()
	defer mu.Unlock()
	if !requestSeen {
		t.Fatal("Pushgateway did not receive any request")
	}
	if pushedBody != "node_load5 1.5\ngpu_up 1\n" {
		t.Fatalf("pushed body = %q, want node and gpu metrics when hardware collection fails", pushedBody)
	}
}
