package exporter

import (
	"errors"
	"testing"

	runtimestate "node-push-exporter/src/runtime"
)

type stubFetcher struct {
	metrics string
	err     error
}

func (s stubFetcher) Fetch() (string, error) {
	return s.metrics, s.err
}

type stubCollector struct {
	metrics string
	err     error
}

func (s stubCollector) Collect() (string, error) {
	return s.metrics, s.err
}

type stubPusher struct {
	pushed []byte
	err    error
}

func (s *stubPusher) Push(metrics []byte) error {
	s.pushed = append([]byte(nil), metrics...)
	return s.err
}

func TestExporter_CollectAndPush_MergesCollectorMetrics(t *testing.T) {
	push := &stubPusher{}
	state := runtimestate.NewState()
	exp := NewWithDependencies(Config{}, Dependencies{
		Fetcher: stubFetcher{metrics: "node_cpu_seconds_total 123\n"},
		Pusher:  push,
		Runtime: state,
		Collectors: []MetricsCollector{
			stubCollector{metrics: "gpu_up 1\n"},
			stubCollector{metrics: "node_hardware_host_info 1\n"},
		},
	})

	if err := exp.CollectAndPush(); err != nil {
		t.Fatalf("CollectAndPush() error = %v", err)
	}

	want := "node_cpu_seconds_total 123\ngpu_up 1\nnode_hardware_host_info 1\n"
	if got := string(push.pushed); got != want {
		t.Fatalf("pushed metrics = %q, want %q", got, want)
	}

	snapshot := state.Snapshot("agent-1")
	if snapshot.Status != "online" {
		t.Fatalf("runtime status = %q, want online", snapshot.Status)
	}
	if snapshot.PushFailCount != 0 {
		t.Fatalf("runtime fail count = %d, want 0", snapshot.PushFailCount)
	}
}

func TestExporter_CollectAndPush_ContinuesWhenCollectorFails(t *testing.T) {
	push := &stubPusher{}
	exp := NewWithDependencies(Config{}, Dependencies{
		Fetcher: stubFetcher{metrics: "node_load1 0.42\n"},
		Pusher:  push,
		Runtime: runtimestate.NewState(),
		Collectors: []MetricsCollector{
			stubCollector{err: errors.New("gpu unavailable")},
			stubCollector{metrics: "node_hardware_host_info 1\n"},
		},
	})

	if err := exp.CollectAndPush(); err != nil {
		t.Fatalf("CollectAndPush() error = %v", err)
	}

	want := "node_load1 0.42\nnode_hardware_host_info 1\n"
	if got := string(push.pushed); got != want {
		t.Fatalf("pushed metrics = %q, want %q", got, want)
	}
}

func TestExporter_CollectAndPush_RecordsFetchFailure(t *testing.T) {
	state := runtimestate.NewState()
	exp := NewWithDependencies(Config{}, Dependencies{
		Fetcher: stubFetcher{err: errors.New("fetch failed")},
		Pusher:  &stubPusher{},
		Runtime: state,
	})

	err := exp.CollectAndPush()
	if err == nil {
		t.Fatal("CollectAndPush() error = nil, want fetch failure")
	}

	snapshot := state.Snapshot("agent-1")
	if snapshot.Status != "degraded" {
		t.Fatalf("runtime status = %q, want degraded", snapshot.Status)
	}
	if snapshot.NodeExporterUp {
		t.Fatal("runtime NodeExporterUp = true, want false")
	}
	if snapshot.PushFailCount != 1 {
		t.Fatalf("runtime fail count = %d, want 1", snapshot.PushFailCount)
	}
}

func TestMergeMetrics(t *testing.T) {
	tests := []struct {
		name  string
		node  string
		extra string
		want  string
	}{
		{name: "both", node: "a 1\n", extra: "b 2\n", want: "a 1\nb 2\n"},
		{name: "node only", node: "a 1\n", extra: "", want: "a 1\n"},
		{name: "extra only", node: "", extra: "b 2\n", want: "b 2\n"},
		{name: "empty", node: "", extra: "", want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := mergeMetrics(tt.node, tt.extra); got != tt.want {
				t.Fatalf("mergeMetrics() = %q, want %q", got, tt.want)
			}
		})
	}
}
