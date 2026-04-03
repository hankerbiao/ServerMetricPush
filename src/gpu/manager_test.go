package gpu

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"
)

type stubCommandExecutor struct {
	paths   map[string]bool
	outputs map[string]stubCommandResult
}

type stubCommandResult struct {
	output string
	err    error
}

func (s stubCommandExecutor) LookPath(file string) (string, error) {
	if s.paths[file] {
		return "/usr/bin/" + file, nil
	}
	return "", fmt.Errorf("%s not found", file)
}

func (s stubCommandExecutor) CombinedOutput(ctx context.Context, name string, args ...string) ([]byte, error) {
	key := name + " " + strings.Join(args, " ")
	result, ok := s.outputs[key]
	if !ok {
		return nil, fmt.Errorf("unexpected command: %s", key)
	}
	return []byte(result.output), result.err
}

func TestManager_Collect_ReturnsTimestampOnlyWhenNoGPUCommandsExist(t *testing.T) {
	manager := &Manager{
		timeout:  time.Second,
		executor: stubCommandExecutor{},
		now: func() time.Time {
			return time.Unix(1700000000, 0).UTC()
		},
	}

	metrics, err := manager.Collect()
	if err != nil {
		t.Fatalf("Collect() error = %v", err)
	}

	want := "node_push_exporter_gpu_scrape_timestamp_seconds 1700000000\n"
	if metrics != want {
		t.Fatalf("Collect() = %q, want %q", metrics, want)
	}
}

func TestManager_Collect_ExportsNvidiaMetrics(t *testing.T) {
	manager := &Manager{
		timeout: time.Second,
		executor: stubCommandExecutor{
			paths: map[string]bool{
				"nvidia-smi": true,
			},
			outputs: map[string]stubCommandResult{
				"nvidia-smi -L": {
					output: "GPU 0: NVIDIA A800 (UUID: GPU-123)\n",
				},
				"nvidia-smi --query-gpu=index,name,uuid,temperature.gpu,utilization.gpu,memory.total,memory.used,power.draw --format=csv,noheader,nounits": {
					output: "0, NVIDIA A800, GPU-123, 52, 78, 81920, 40960, 250.5\n",
				},
			},
		},
		now: func() time.Time {
			return time.Unix(1700000000, 0).UTC()
		},
	}

	metrics, err := manager.Collect()
	if err != nil {
		t.Fatalf("Collect() error = %v", err)
	}

	assertContainsMetric(t, metrics, `node_push_exporter_gpu_scrape_success{vendor="nvidia"} 1`)
	assertContainsMetric(t, metrics, `node_push_exporter_gpu_devices_detected{vendor="nvidia"} 1`)
	assertContainsMetric(t, metrics, `gpu_info{gpu="0",name="NVIDIA A800",uuid="GPU-123",vendor="nvidia"} 1`)
	assertContainsMetric(t, metrics, `gpu_temperature_celsius{gpu="0",name="NVIDIA A800",uuid="GPU-123",vendor="nvidia"} 52`)
	assertContainsMetric(t, metrics, `gpu_memory_total_bytes{gpu="0",name="NVIDIA A800",uuid="GPU-123",vendor="nvidia"} 85899345920`)
}

func assertContainsMetric(t *testing.T, metrics, want string) {
	t.Helper()
	if !strings.Contains(metrics, want) {
		t.Fatalf("metrics = %q, want substring %q", metrics, want)
	}
}
