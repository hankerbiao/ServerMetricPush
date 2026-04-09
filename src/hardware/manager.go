package hardware

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

type commandExecutor interface {
	LookPath(file string) (string, error)
	CombinedOutput(timeout time.Duration, name string, args ...string) ([]byte, error)
}

type osCommandExecutor struct{}

func (osCommandExecutor) LookPath(file string) (string, error) {
	return exec.LookPath(file)
}

func (osCommandExecutor) CombinedOutput(timeout time.Duration, name string, args ...string) ([]byte, error) {
	if timeout <= 0 {
		timeout = 5 * time.Second
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, name, args...)
	return cmd.CombinedOutput()
}

type Manager struct {
	timeout               time.Duration
	executor              commandExecutor
	now                   func() time.Time
	dmiDir                string
	procCPUInfoPath       string
	procMemInfoPath       string
	netClassDir           string
	includeSerials        bool
	includeVirtualDevices bool
}

func NewManager(timeout time.Duration, includeSerials, includeVirtualDevices bool) *Manager {
	if timeout <= 0 {
		timeout = 5 * time.Second
	}

	return &Manager{
		timeout:               timeout,
		executor:              osCommandExecutor{},
		now:                   time.Now,
		dmiDir:                "/sys/class/dmi/id",
		procCPUInfoPath:       "/proc/cpuinfo",
		procMemInfoPath:       "/proc/meminfo",
		netClassDir:           "/sys/class/net",
		includeSerials:        includeSerials,
		includeVirtualDevices: includeVirtualDevices,
	}
}

func (m *Manager) Collect() (string, error) {
	if m.executor == nil {
		m.executor = osCommandExecutor{}
	}
	if m.now == nil {
		m.now = time.Now
	}

	lines := []string{
		fmt.Sprintf("node_push_exporter_hardware_scrape_timestamp_seconds %d", m.now().UTC().Unix()),
	}

	lines = append(lines, m.collectHost()...)
	lines = append(lines, m.collectCPU()...)
	lines = append(lines, m.collectMemory()...)
	lines = append(lines, m.collectDisks()...)
	lines = append(lines, m.collectNICs()...)
	lines = append(lines, "node_push_exporter_hardware_scrape_success 1")

	return strings.Join(lines, "\n") + "\n", nil
}

func (m *Manager) collectHost() []string {
	labels := make(map[string]string)
	for _, field := range []string{
		"system_vendor",
		"product_name",
		"product_version",
		"product_serial",
		"product_uuid",
		"board_name",
		"board_vendor",
		"board_version",
		"board_serial",
		"bios_vendor",
		"bios_version",
		"bios_date",
	} {
		if !m.includeSerials && (strings.Contains(field, "serial") || strings.Contains(field, "uuid")) {
			continue
		}
		addIfNotEmpty(labels, field, readTrimmedFile(filepath.Join(m.dmiDir, field)))
	}

	if len(labels) == 0 {
		return nil
	}
	return []string{metricLine("node_hardware_host_info", labels, 1)}
}

func (m *Manager) collectCPU() []string {
	content := readTrimmedFile(m.procCPUInfoPath)
	if content == "" {
		return nil
	}

	blocks := strings.Split(content, "\n\n")
	packages := map[string]struct{}{}
	totalCores := 0
	labels := map[string]string{}

	for _, block := range blocks {
		if strings.TrimSpace(block) == "" {
			continue
		}
		fields := parseColonLines(block)
		addIfEmpty(labels, "vendor_id", fields["vendor_id"])
		addIfEmpty(labels, "model_name", fields["model name"])

		if physicalID := fields["physical id"]; physicalID != "" {
			packages[physicalID] = struct{}{}
		}
		if cores, err := strconv.Atoi(fields["cpu cores"]); err == nil && cores > 0 {
			totalCores += cores
		}
	}

	if len(packages) == 0 && totalCores > 0 {
		packages["0"] = struct{}{}
	}

	lines := []string{}
	if len(labels) > 0 {
		lines = append(lines, metricLine("node_hardware_cpu_info", labels, 1))
	}
	if len(packages) > 0 {
		lines = append(lines, fmt.Sprintf("node_hardware_cpu_packages %d", len(packages)))
	}
	if totalCores > 0 {
		lines = append(lines, fmt.Sprintf("node_hardware_cpu_cores %d", totalCores))
	}
	return lines
}

func (m *Manager) collectMemory() []string {
	lines := m.collectMemorySummary()
	dimmLines, err := m.collectMemoryMetrics()
	if err == nil {
		lines = append(lines, dimmLines...)
	}
	return lines
}

func (m *Manager) collectMemorySummary() []string {
	fields := parseColonLines(readTrimmedFile(m.procMemInfoPath))
	memTotal := strings.Fields(fields["MemTotal"])
	if len(memTotal) == 0 {
		return nil
	}

	kib, err := strconv.ParseFloat(memTotal[0], 64)
	if err != nil {
		return nil
	}
	totalBytes := kib * 1024

	return []string{
		metricLine("node_hardware_memory_info", map[string]string{"total_bytes": strconv.FormatInt(int64(totalBytes), 10)}, 1),
		fmt.Sprintf("node_hardware_memory_total_bytes %g", totalBytes),
	}
}

func (m *Manager) collectMemoryMetrics() ([]string, error) {
	if m.executor == nil {
		return nil, nil
	}
	dmidecodePath, err := m.resolveDMIDecodePath()
	if err != nil {
		return nil, err
	}

	output, err := m.executor.CombinedOutput(m.timeout, dmidecodePath, "--type", "memory")
	if err != nil || len(output) == 0 {
		return nil, err
	}

	devices := parseDMISectionRecords(string(output), "Memory Device")
	lines := make([]string, 0, len(devices)*2)
	for _, fields := range devices {
		sizeBytes, ok := parseDMIMemorySize(fields["Size"])
		if !ok || sizeBytes <= 0 {
			continue
		}

		labels := map[string]string{}
		addIfNotEmpty(labels, "locator", fields["Locator"])
		addIfNotEmpty(labels, "bank_locator", fields["Bank Locator"])
		addIfNotEmpty(labels, "manufacturer", fields["Manufacturer"])
		addIfNotEmpty(labels, "part_number", fields["Part Number"])
		addIfNotEmpty(labels, "type", fields["Type"])
		addIfNotEmpty(labels, "type_detail", fields["Type Detail"])
		addIfNotEmpty(labels, "form_factor", fields["Form Factor"])
		labels["size_bytes"] = strconv.FormatInt(sizeBytes, 10)

		if speed, ok := parseDMISpeed(fields["Speed"]); ok {
			labels["speed_mt_s"] = strconv.FormatInt(speed, 10)
		}
		if m.includeSerials {
			addIfNotEmpty(labels, "serial", fields["Serial Number"])
		}

		if len(labels) == 0 {
			continue
		}
		lines = append(lines, metricLine("node_hardware_memory_dimm_info", labels, 1))

		sizeMetricLabels := map[string]string{}
		if labels["locator"] != "" {
			sizeMetricLabels["locator"] = labels["locator"]
		} else if labels["bank_locator"] != "" {
			sizeMetricLabels["bank_locator"] = labels["bank_locator"]
		}
		lines = append(lines, metricLine("node_hardware_memory_dimm_size_bytes", sizeMetricLabels, float64(sizeBytes)))
	}

	return lines, nil
}

func (m *Manager) resolveDMIDecodePath() (string, error) {
	if m.executor == nil {
		return "", nil
	}

	paths := []string{
		"dmidecode",
		"/usr/sbin/dmidecode",
		"/usr/local/sbin/dmidecode",
		"/sbin/dmidecode",
	}

	var errs []error
	for _, path := range paths {
		resolved, err := m.executor.LookPath(path)
		if err == nil && resolved != "" {
			return resolved, nil
		}
		if err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) == 0 {
		return "", fmt.Errorf("dmidecode not found")
	}
	return "", errors.Join(errs...)
}

func (m *Manager) collectDisks() []string {
	if _, err := m.executor.LookPath("lsblk"); err != nil {
		return nil
	}

	output, err := m.executor.CombinedOutput(m.timeout, "lsblk", "-J", "-o", "NAME,TYPE,MODEL,VENDOR,SERIAL,SIZE,ROTA,WWN,TRAN")
	if err != nil || len(output) == 0 {
		return nil
	}

	var payload struct {
		BlockDevices []struct {
			Name   string `json:"name"`
			Type   string `json:"type"`
			Model  string `json:"model"`
			Vendor string `json:"vendor"`
			Serial string `json:"serial"`
			Size   string `json:"size"`
			WWN    string `json:"wwn"`
			Tran   string `json:"tran"`
		} `json:"blockdevices"`
	}
	if err := json.Unmarshal(output, &payload); err != nil {
		return nil
	}

	lines := []string{}
	for _, device := range payload.BlockDevices {
		if device.Type != "disk" || device.Name == "" {
			continue
		}

		labels := map[string]string{"device": device.Name}
		addIfNotEmpty(labels, "model", device.Model)
		addIfNotEmpty(labels, "vendor", device.Vendor)
		addIfNotEmpty(labels, "size_bytes", device.Size)
		addIfNotEmpty(labels, "wwn", device.WWN)
		addIfNotEmpty(labels, "transport", device.Tran)
		if m.includeSerials {
			addIfNotEmpty(labels, "serial", device.Serial)
		}

		lines = append(lines, metricLine("node_hardware_disk_info", labels, 1))
		if size, err := strconv.ParseFloat(device.Size, 64); err == nil && size > 0 {
			lines = append(lines, metricLine("node_hardware_disk_size_bytes", map[string]string{"device": device.Name}, size))
		}
	}
	return lines
}

func (m *Manager) collectNICs() []string {
	entries, err := os.ReadDir(m.netClassDir)
	if err != nil {
		return nil
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Name() < entries[j].Name() })

	lines := []string{}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		iface := entry.Name()
		if !m.includeVirtualDevices && isVirtualInterface(iface) {
			continue
		}

		base := filepath.Join(m.netClassDir, iface)
		labels := map[string]string{"iface": iface}
		addIfNotEmpty(labels, "address", readTrimmedFile(filepath.Join(base, "address")))
		addIfNotEmpty(labels, "vendor_id", readTrimmedFile(filepath.Join(base, "device/vendor")))
		addIfNotEmpty(labels, "device_id", readTrimmedFile(filepath.Join(base, "device/device")))

		if speedMbps := readTrimmedFile(filepath.Join(base, "speed")); speedMbps != "" {
			if speed, err := strconv.ParseFloat(speedMbps, 64); err == nil && speed > 0 {
				speedBits := speed * 1000 * 1000
				labels["speed_bits"] = strconv.FormatInt(int64(speedBits), 10)
				lines = append(lines, metricLine("node_hardware_nic_speed_bits", map[string]string{"iface": iface}, speedBits))
			}
		}

		if len(labels) == 1 {
			continue
		}
		lines = append(lines, metricLine("node_hardware_nic_info", labels, 1))
	}
	return lines
}

func parseColonLines(content string) map[string]string {
	values := make(map[string]string)
	for _, line := range strings.Split(content, "\n") {
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}
		values[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
	}
	return values
}

func parseDMISectionRecords(content, sectionName string) []map[string]string {
	lines := strings.Split(content, "\n")
	records := []map[string]string{}

	for i := 0; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) != sectionName {
			continue
		}

		record := map[string]string{}
		for j := i + 1; j < len(lines); j++ {
			line := lines[j]
			trimmed := strings.TrimSpace(line)
			if trimmed == "" {
				if len(record) > 0 {
					records = append(records, record)
				}
				i = j
				break
			}
			if !strings.HasPrefix(line, "\t") && !strings.HasPrefix(line, " ") {
				if len(record) > 0 {
					records = append(records, record)
				}
				i = j - 1
				break
			}

			parts := strings.SplitN(trimmed, ":", 2)
			if len(parts) != 2 {
				continue
			}
			record[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])

			if j == len(lines)-1 && len(record) > 0 {
				records = append(records, record)
				i = j
			}
		}
	}

	return records
}

func parseDMIMemorySize(value string) (int64, bool) {
	fields := strings.Fields(strings.TrimSpace(value))
	if len(fields) < 2 {
		return 0, false
	}
	if strings.EqualFold(fields[0], "No") || strings.EqualFold(fields[0], "Unknown") {
		return 0, false
	}

	size, err := strconv.ParseFloat(fields[0], 64)
	if err != nil || size <= 0 {
		return 0, false
	}

	unit := strings.ToUpper(fields[1])
	switch unit {
	case "KB", "KIB":
		return int64(size * 1024), true
	case "MB", "MIB":
		return int64(size * 1024 * 1024), true
	case "GB", "GIB":
		return int64(size * 1024 * 1024 * 1024), true
	case "TB", "TIB":
		return int64(size * 1024 * 1024 * 1024 * 1024), true
	default:
		return 0, false
	}
}

func parseDMISpeed(value string) (int64, bool) {
	fields := strings.Fields(strings.TrimSpace(value))
	if len(fields) == 0 {
		return 0, false
	}
	speed, err := strconv.ParseInt(fields[0], 10, 64)
	if err != nil || speed <= 0 {
		return 0, false
	}
	return speed, true
}

func metricLine(name string, labels map[string]string, value any) string {
	if len(labels) == 0 {
		return fmt.Sprintf("%s %v", name, value)
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

	return fmt.Sprintf("%s{%s} %v", name, strings.Join(parts, ","), value)
}

func escapeLabelValue(value string) string {
	replacer := strings.NewReplacer(`\`, `\\`, "\n", `\n`, `"`, `\"`)
	return replacer.Replace(value)
}

func addIfNotEmpty(labels map[string]string, key, value string) {
	value = strings.TrimSpace(value)
	if value != "" {
		labels[key] = value
	}
}

func addIfEmpty(labels map[string]string, key, value string) {
	if labels[key] == "" {
		addIfNotEmpty(labels, key, value)
	}
}

func readTrimmedFile(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

func isVirtualInterface(iface string) bool {
	return iface == "lo" ||
		strings.HasPrefix(iface, "veth") ||
		strings.HasPrefix(iface, "docker") ||
		strings.HasPrefix(iface, "br-") ||
		strings.HasPrefix(iface, "virbr") ||
		strings.HasPrefix(iface, "tun") ||
		strings.HasPrefix(iface, "tap")
}
