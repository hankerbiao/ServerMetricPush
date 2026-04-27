package exporter

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"node-push-exporter/src/gpu"
	"node-push-exporter/src/pusher"
	"node-push-exporter/src/runtime"
)

// MetricsCollector 定义指标采集器的接口。
type MetricsCollector interface {
	Collect() (string, error)
}

// MetricsFetcher 定义如何加载 node_exporter 指标。
type MetricsFetcher interface {
	Fetch() (string, error)
}

// PushClient 定义 Exporter 所需的 Pushgateway 客户端行为。
type PushClient interface {
	Push([]byte) error
}

// Config 保存导出器配置。
type Config struct {
	NodeExporterMetricsURL string
	PushURL                string
	PushJob                string
	PushInstance           string
	PushTimeout            time.Duration
	GPUEnabled             bool
}

// Dependencies 允许测试和调用方替换具体实现。
type Dependencies struct {
	Fetcher    MetricsFetcher
	Pusher     PushClient
	Collectors []MetricsCollector
	Runtime    *runtime.State
}

// Exporter 协调指标采集和推送。
type Exporter struct {
	config       Config
	fetcher      MetricsFetcher
	pusher       PushClient
	collectors   []MetricsCollector
	runtimeState *runtime.State
}

// New 创建一个新的 Exporter。
func New(cfg Config) *Exporter {
	return NewWithDependencies(cfg, Dependencies{})
}

// NewWithDependencies 使用可选的依赖注入创建一个导出器。
func NewWithDependencies(cfg Config, deps Dependencies) *Exporter {
	fetcher := deps.Fetcher
	if fetcher == nil {
		fetcher = newHTTPFetcher(cfg.NodeExporterMetricsURL, 10*time.Second)
	}

	pushClient := deps.Pusher
	if pushClient == nil {
		pushClient = newPusher(cfg)
	}

	collectors := deps.Collectors
	if collectors == nil {
		collectors = defaultCollectors(cfg)
	}

	state := deps.Runtime
	if state == nil {
		state = runtime.NewState()
	}

	return &Exporter{
		config:       cfg,
		fetcher:      fetcher,
		pusher:       pushClient,
		collectors:   collectors,
		runtimeState: state,
	}
}

func newPusher(cfg Config) *pusher.Pusher {
	opts := []pusher.Option{
		pusher.WithJob(cfg.PushJob),
		pusher.WithInstance(cfg.PushInstance),
		pusher.WithTimeout(cfg.PushTimeout),
	}
	return pusher.NewPusher(cfg.PushURL, opts...)
}

func defaultCollectors(cfg Config) []MetricsCollector {
	var collectors []MetricsCollector

	if cfg.GPUEnabled {
		collectors = append(collectors, gpu.NewManager(5*time.Second))
	}

	return collectors
}

// Runtime 返回用于控制面报告的运行时状态。
func (e *Exporter) Runtime() *runtime.State {
	return e.runtimeState
}

// CollectAndPush 从所有采集器获取指标并推送到 Pushgateway。
func (e *Exporter) CollectAndPush() error {
	metrics, err := e.fetcher.Fetch()
	if err != nil {
		e.runtimeState.RecordFetchFailure(err)
		return fmt.Errorf("获取指标失败: %w", err)
	}

	// 从所有已注册的采集器收集指标
	for _, collector := range e.collectors {
		extraMetrics, err := collector.Collect()
		if err != nil {
			log.Printf("指标采集失败，继续推送已有指标: %v", err)
			continue
		}
		metrics = mergeMetrics(metrics, extraMetrics)
	}

	// 推送到 Pushgateway
	if err := e.pusher.Push([]byte(metrics)); err != nil {
		e.runtimeState.RecordPushFailure(err)
		return fmt.Errorf("推送失败: %w", err)
	}

	e.runtimeState.RecordPushSuccess()
	log.Printf("指标推送成功，来源: %s", e.config.NodeExporterMetricsURL)
	return nil
}

type httpFetcher struct {
	url    string
	client *http.Client
}

func newHTTPFetcher(url string, timeout time.Duration) MetricsFetcher {
	return httpFetcher{
		url: url,
		client: &http.Client{
			Timeout: timeout,
		},
	}
}

func (f httpFetcher) Fetch() (string, error) {
	resp, err := f.client.Get(f.url)
	if err != nil {
		return "", fmt.Errorf("HTTP请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP状态码错误: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("读取响应失败: %w", err)
	}

	metrics := string(body)
	if strings.TrimSpace(metrics) == "" {
		return "", fmt.Errorf("指标数据为空")
	}

	return metrics, nil
}

func mergeMetrics(nodeMetrics, extraMetrics string) string {
	nodeMetrics = strings.TrimRight(nodeMetrics, "\n")
	extraMetrics = strings.TrimSpace(extraMetrics)

	switch {
	case nodeMetrics == "":
		if extraMetrics == "" {
			return ""
		}
		return extraMetrics + "\n"
	case extraMetrics == "":
		return nodeMetrics + "\n"
	default:
		return nodeMetrics + "\n" + extraMetrics + "\n"
	}
}
