package gpu

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"sort"
	"strconv"
	"strings"
	"time"
)

type commandExecutor interface {
	LookPath(file string) (string, error)
	CombinedOutput(ctx context.Context, name string, args ...string) ([]byte, error)
}

type osCommandExecutor struct{}

// LookPath 检查命令是否存在于当前系统 PATH 中。
func (osCommandExecutor) LookPath(file string) (string, error) {
	return exec.LookPath(file)
}

// CombinedOutput 在带超时控制的上下文中执行命令，并返回标准输出和标准错误的合并结果。
func (osCommandExecutor) CombinedOutput(ctx context.Context, name string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	return cmd.CombinedOutput()
}

type Manager struct {
	timeout  time.Duration
	executor commandExecutor
	now      func() time.Time
}

// deviceMetrics 表示单张 GPU 在导出前的统一指标视图。
// 不同厂商的原始输出格式不同，先汇总到该结构，再统一渲染为 Prometheus 指标。
type deviceMetrics struct {
	Vendor            string
	GPU               string
	Name              string
	UUID              string
	DeviceID          string
	Temperature       *float64
	TemperatureEdge   *float64
	TemperatureJunc   *float64
	TemperatureMem    *float64
	TemperatureCore   *float64
	Utilization       *float64
	MemoryUsedPercent *float64
	MemoryUsedBytes   *float64
	MemoryTotalBytes  *float64
	PowerDrawWatts    *float64
}

// NewManager 创建 GPU 指标管理器，并在未传入有效超时时使用默认值。
func NewManager(timeout time.Duration) *Manager {
	if timeout <= 0 {
		timeout = 5 * time.Second
	}

	return &Manager{
		timeout:  timeout,
		executor: osCommandExecutor{},
		now:      time.Now,
	}
}

// Collect 聚合当前主机可用的 GPU 指标。
// 如果某个厂商工具不存在，则直接跳过；如果命令执行失败，则返回该厂商的失败状态指标。
func (m *Manager) Collect() (string, error) {
	if m.executor == nil {
		m.executor = osCommandExecutor{}
	}
	if m.now == nil {
		m.now = time.Now
	}

	sections := []string{
		fmt.Sprintf("node_push_exporter_gpu_scrape_timestamp_seconds %d", m.now().UTC().Unix()),
	}

	nvidiaMetrics := m.collectNvidia()
	if nvidiaMetrics != "" {
		sections = append(sections, nvidiaMetrics)
	}

	rocmMetrics := m.collectROCM()
	if rocmMetrics != "" {
		sections = append(sections, rocmMetrics)
	}

	return strings.Join(sections, "\n") + "\n", nil
}

// collectNvidia 使用 nvidia-smi 先探测设备数量，再批量查询指标。
// 先执行 `nvidia-smi -L` 是为了在查询失败前仍能区分“没有设备”和“采集失败”。
func (m *Manager) collectNvidia() string {
	if _, err := m.executor.LookPath("nvidia-smi"); err != nil {
		return ""
	}

	ctx, cancel := context.WithTimeout(context.Background(), m.timeout)
	defer cancel()

	listOutput, err := m.executor.CombinedOutput(ctx, "nvidia-smi", "-L")
	if err != nil {
		return vendorFailureMetrics("nvidia", err)
	}

	deviceCount := countNonEmptyLines(string(listOutput))
	if deviceCount == 0 {
		return vendorCountMetrics("nvidia", 0)
	}

	ctx, cancel = context.WithTimeout(context.Background(), m.timeout)
	defer cancel()

	queryOutput, err := m.executor.CombinedOutput(
		ctx,
		"nvidia-smi",
		"--query-gpu=index,name,uuid,temperature.gpu,utilization.gpu,memory.total,memory.used,power.draw",
		"--format=csv,noheader,nounits",
	)
	if err != nil {
		return vendorFailureMetrics("nvidia", err)
	}

	devices := parseNvidiaCSV(string(queryOutput))
	if len(devices) == 0 {
		return vendorFailureMetrics("nvidia", fmt.Errorf("未解析到NVIDIA GPU指标"))
	}

	return renderVendorMetrics("nvidia", devices)
}

// collectROCM 通过 rocm-smi 的 JSON 输出采集 AMD/ROCm 设备指标。
func (m *Manager) collectROCM() string {
	if _, err := m.executor.LookPath("rocm-smi"); err != nil {
		return ""
	}

	ctx, cancel := context.WithTimeout(context.Background(), m.timeout)
	defer cancel()

	output, err := m.executor.CombinedOutput(ctx, "rocm-smi", "--showtemp", "--showuse", "--showmemuse", "--json")
	if err != nil {
		return vendorFailureMetrics("rocm", err)
	}

	devices := parseROCMJSON(output)
	if len(devices) == 0 {
		return vendorCountMetrics("rocm", 0)
	}

	return renderVendorMetrics("rocm", devices)
}

// renderVendorMetrics 将统一结构渲染为最终的 Prometheus 文本格式。
func renderVendorMetrics(vendor string, devices []deviceMetrics) string {
	lines := []string{
		metricLine("node_push_exporter_gpu_scrape_success", map[string]string{"vendor": vendor}, 1),
		metricLine("node_push_exporter_gpu_devices_detected", map[string]string{"vendor": vendor}, float64(len(devices))),
	}

	for _, device := range devices {
		labels := make(map[string]string)
		labels["vendor"] = device.Vendor
		if device.GPU != "" {
			labels["gpu"] = device.GPU
		}
		if device.Name != "" {
			labels["name"] = device.Name
		}
		if device.UUID != "" {
			labels["uuid"] = device.UUID
		}
		if device.DeviceID != "" {
			labels["device_id"] = device.DeviceID
		}

		lines = append(lines, metricLine("gpu_up", labels, 1))
		lines = append(lines, metricLine("gpu_info", labels, 1))
		appendMetric(&lines, "gpu_temperature_celsius", labels, device.Temperature)
		appendMetric(&lines, "gpu_temperature_edge_celsius", labels, device.TemperatureEdge)
		appendMetric(&lines, "gpu_temperature_junction_celsius", labels, device.TemperatureJunc)
		appendMetric(&lines, "gpu_temperature_mem_celsius", labels, device.TemperatureMem)
		appendMetric(&lines, "gpu_temperature_core_celsius", labels, device.TemperatureCore)
		appendMetric(&lines, "gpu_utilization_percent", labels, device.Utilization)
		appendMetric(&lines, "gpu_memory_used_percent", labels, device.MemoryUsedPercent)
		appendMetric(&lines, "gpu_memory_used_bytes", labels, device.MemoryUsedBytes)
		appendMetric(&lines, "gpu_memory_total_bytes", labels, device.MemoryTotalBytes)
		appendMetric(&lines, "gpu_power_draw_watts", labels, device.PowerDrawWatts)
	}

	return strings.Join(lines, "\n")
}

// vendorFailureMetrics 在厂商采集失败时输出基础状态指标，并附带注释形式的错误信息。
func vendorFailureMetrics(vendor string, err error) string {
	lines := []string{
		metricLine("node_push_exporter_gpu_scrape_success", map[string]string{"vendor": vendor}, 0),
		metricLine("node_push_exporter_gpu_devices_detected", map[string]string{"vendor": vendor}, 0),
	}
	if err != nil {
		lines = append(lines, "# gpu scrape error for "+vendor+": "+sanitizeComment(err.Error()))
	}
	return strings.Join(lines, "\n")
}

// vendorCountMetrics 在采集成功但未必存在设备时，输出厂商级别的成功状态和设备数量指标。
func vendorCountMetrics(vendor string, count int) string {
	return strings.Join([]string{
		metricLine("node_push_exporter_gpu_scrape_success", map[string]string{"vendor": vendor}, 1),
		metricLine("node_push_exporter_gpu_devices_detected", map[string]string{"vendor": vendor}, float64(count)),
	}, "\n")
}

// parseNvidiaCSV 解析 nvidia-smi 的 CSV 输出，并把显存从 MiB 转成字节。
func parseNvidiaCSV(output string) []deviceMetrics {
	lines := strings.Split(strings.TrimSpace(output), "\n")
	devices := make([]deviceMetrics, 0, len(lines))

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		fields := strings.Split(line, ",")
		if len(fields) < 8 {
			continue
		}

		device := deviceMetrics{
			Vendor: "nvidia",
			GPU:    strings.TrimSpace(fields[0]),
			Name:   strings.TrimSpace(fields[1]),
			UUID:   strings.TrimSpace(fields[2]),
		}
		device.Temperature = parseMetricValue(fields[3])
		device.Utilization = parseMetricValue(fields[4])
		device.MemoryTotalBytes = parseMiBToBytes(fields[5])
		device.MemoryUsedBytes = parseMiBToBytes(fields[6])
		device.PowerDrawWatts = parseMetricValue(fields[7])

		devices = append(devices, device)
	}

	return devices
}

// parseROCMJSON 兼容 rocm-smi 不同版本字段名的差异，尽量从多个候选键中提取指标。
func parseROCMJSON(output []byte) []deviceMetrics {
	var raw map[string]any
	if err := json.Unmarshal(output, &raw); err != nil {
		return nil
	}

	keys := make([]string, 0, len(raw))
	for key := range raw {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	devices := make([]deviceMetrics, 0, len(keys))
	for _, key := range keys {
		entry, ok := raw[key].(map[string]any)
		if !ok {
			continue
		}

		device := deviceMetrics{
			Vendor:   "rocm",
			GPU:      extractTrailingDigits(key),
			DeviceID: key,
			Name:     valueAsString(findValue(entry, "Card series", "Card SKU", "Card model")),
		}
		device.Temperature = parseMetricAny(findValue(entry, "Temperature (Sensor edge) (C)", "Temperature (edge) (C)"))
		device.TemperatureEdge = parseMetricAny(findValue(entry, "Temperature (Sensor edge) (C)", "Temperature (edge) (C)"))
		device.TemperatureJunc = parseMetricAny(findValue(entry, "Temperature (Sensor junction) (C)", "Temperature (junction) (C)"))
		device.TemperatureMem = parseMetricAny(findValue(entry, "Temperature (Sensor mem) (C)", "Temperature (mem) (C)"))
		device.TemperatureCore = parseMetricAny(findValue(entry, "Temperature (Sensor core) (C)", "Temperature (core) (C)"))
		device.Utilization = parseMetricAny(findValue(entry, "GPU use (%)", "GPU use", "HCU use (%)", "HCU use"))
		device.MemoryUsedPercent = parseMetricAny(findValue(entry, "GPU memory use (%)", "GPU Memory Allocated (%)", "VRAM use (%)", "HCU memory use (%)"))

		devices = append(devices, device)
	}

	return devices
}

// appendMetric 仅在指标值存在时追加一行，避免导出无意义的空值指标。
func appendMetric(lines *[]string, name string, labels map[string]string, value *float64) {
	if value == nil {
		return
	}
	*lines = append(*lines, metricLine(name, labels, *value))
}

// metricLine 负责统一拼接 Prometheus 指标行，并对 label 按 key 排序，保证输出稳定。
func metricLine(name string, labels map[string]string, value float64) string {
	if len(labels) == 0 {
		return fmt.Sprintf("%s %s", name, strconv.FormatFloat(value, 'f', -1, 64))
	}

	keys := make([]string, 0, len(labels))
	for key := range labels {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, fmt.Sprintf(`%s="%s"`, key, escapeLabelValue(labels[key])))
	}

	return fmt.Sprintf("%s{%s} %s", name, strings.Join(parts, ","), strconv.FormatFloat(value, 'f', -1, 64))
}

// escapeLabelValue 转义 Prometheus label 中的反斜杠、换行和双引号。
func escapeLabelValue(value string) string {
	replacer := strings.NewReplacer(`\`, `\\`, "\n", `\n`, `"`, `\"`)
	return replacer.Replace(value)
}

// sanitizeComment 将错误信息压缩为单行，避免注释内容破坏指标文本格式。
func sanitizeComment(value string) string {
	return strings.ReplaceAll(value, "\n", " ")
}

// countNonEmptyLines 统计非空行数量，用于从命令输出中估算检测到的设备数。
func countNonEmptyLines(value string) int {
	count := 0
	for _, line := range strings.Split(value, "\n") {
		if strings.TrimSpace(line) != "" {
			count++
		}
	}
	return count
}

// parseMetricValue 清洗常见单位后解析数值；空值或 N/A 统一视为缺失指标。
func parseMetricValue(value string) *float64 {
	value = strings.TrimSpace(value)
	value = strings.TrimSuffix(value, "%")
	value = strings.TrimSuffix(value, "W")
	value = strings.TrimSpace(value)
	if value == "" || strings.EqualFold(value, "N/A") {
		return nil
	}

	parsed, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return nil
	}
	return &parsed
}

// parseMiBToBytes 将以 MiB 表示的显存值转换为字节，便于统一导出字节单位指标。
func parseMiBToBytes(value string) *float64 {
	parsed := parseMetricValue(value)
	if parsed == nil {
		return nil
	}
	bytes := *parsed * 1024 * 1024
	return &bytes
}

// parseMetricAny 处理 JSON 中可能出现的多种数值表示方式。
func parseMetricAny(value any) *float64 {
	switch v := value.(type) {
	case nil:
		return nil
	case float64:
		return &v
	case string:
		return parseMetricValue(cleanMetricString(v))
	default:
		return parseMetricValue(fmt.Sprint(v))
	}
}

// cleanMetricString 去掉 rocm-smi 输出中常见的温度、功耗和容量单位。
func cleanMetricString(value string) string {
	value = strings.TrimSpace(value)
	replacer := strings.NewReplacer(
		"C", "",
		"c", "",
		"MiB", "",
		"MB", "",
		"mW", "",
		"W", "",
	)
	return strings.TrimSpace(replacer.Replace(value))
}

// findValue 按优先级查找第一个存在的键，用于兼容不同版本工具的字段命名。
func findValue(values map[string]any, keys ...string) any {
	for _, key := range keys {
		if value, ok := values[key]; ok {
			return value
		}
	}
	return nil
}

// valueAsString 将任意值转换为字符串，并清理首尾空白。
func valueAsString(value any) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(fmt.Sprint(value))
}

// extractTrailingDigits 提取字符串末尾连续数字，通常用于从设备键名中推导 GPU 编号。
func extractTrailingDigits(value string) string {
	for i := len(value) - 1; i >= 0; i-- {
		if value[i] < '0' || value[i] > '9' {
			if i == len(value)-1 {
				return ""
			}
			return value[i+1:]
		}
	}
	return value
}
