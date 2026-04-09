package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	Pushgateway  PushgatewayConfig
	NodeExporter NodeExporterConfig
	ControlPlane ControlPlaneConfig
	Hardware     HardwareConfig
	Update       UpdateConfig
}

type PushgatewayConfig struct {
	URL      string
	Job      string
	Instance string
	Interval int
	Timeout  int
}

type NodeExporterConfig struct {
	Path       string
	Port       int
	MetricsURL string
}

type ControlPlaneConfig struct {
	URL               string
	HeartbeatInterval int
}

type HardwareConfig struct {
	Enabled               bool
	Timeout               int
	IncludeSerials        bool
	IncludeVirtualDevices bool
}

type UpdateConfig struct {
	Enabled      bool
	ListenAddr   string
	AllowedCIDRs []string
	StatusFile   string
	WorkDir      string
}

func (c ControlPlaneConfig) Enabled() bool {
	return c.URL != ""
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("读取配置文件失败: %w", err)
	}

	cfg := defaultConfig()
	lines := strings.Split(string(data), "\n")
	for index, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		switch key {
		case "pushgateway.url":
			cfg.Pushgateway.URL = value
		case "pushgateway.job":
			cfg.Pushgateway.Job = value
		case "pushgateway.instance":
			cfg.Pushgateway.Instance = value
		case "pushgateway.interval":
			i, err := parseInt(key, value, index)
			if err != nil {
				return nil, err
			}
			cfg.Pushgateway.Interval = i
		case "pushgateway.timeout":
			i, err := parseInt(key, value, index)
			if err != nil {
				return nil, err
			}
			cfg.Pushgateway.Timeout = i
		case "node_exporter.path":
			cfg.NodeExporter.Path = value
		case "node_exporter.port":
			i, err := parseInt(key, value, index)
			if err != nil {
				return nil, err
			}
			cfg.NodeExporter.Port = i
		case "node_exporter.metrics_url":
			cfg.NodeExporter.MetricsURL = value
		case "control_plane.url":
			cfg.ControlPlane.URL = value
		case "control_plane.heartbeat_interval":
			i, err := parseInt(key, value, index)
			if err != nil {
				return nil, err
			}
			cfg.ControlPlane.HeartbeatInterval = i
		case "hardware.enabled":
			b, err := parseBool(key, value, index)
			if err != nil {
				return nil, err
			}
			cfg.Hardware.Enabled = b
		case "hardware.timeout":
			i, err := parseInt(key, value, index)
			if err != nil {
				return nil, err
			}
			cfg.Hardware.Timeout = i
		case "hardware.include_serials":
			b, err := parseBool(key, value, index)
			if err != nil {
				return nil, err
			}
			cfg.Hardware.IncludeSerials = b
		case "hardware.include_virtual_devices":
			b, err := parseBool(key, value, index)
			if err != nil {
				return nil, err
			}
			cfg.Hardware.IncludeVirtualDevices = b
		case "update.enabled":
			b, err := parseBool(key, value, index)
			if err != nil {
				return nil, err
			}
			cfg.Update.Enabled = b
		case "update.listen_addr":
			cfg.Update.ListenAddr = value
		case "update.allowed_cidrs":
			cfg.Update.AllowedCIDRs = parseCSV(value)
		case "update.status_file":
			cfg.Update.StatusFile = value
		case "update.work_dir":
			cfg.Update.WorkDir = value
		}
	}

	if err := validate(cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}

func defaultConfig() *Config {
	return &Config{
		Hardware: HardwareConfig{
			Enabled:               true,
			Timeout:               5,
			IncludeSerials:        true,
			IncludeVirtualDevices: false,
		},
		Update: UpdateConfig{
			Enabled:      false,
			ListenAddr:   "127.0.0.1:18080",
			AllowedCIDRs: []string{"127.0.0.0/8", "10.0.0.0/8", "172.16.0.0/12", "192.168.0.0/16"},
			StatusFile:   "/var/lib/node-push-exporter/update-status.json",
			WorkDir:      "/var/lib/node-push-exporter/update-work",
		},
	}
}

func validate(cfg *Config) error {
	requiredFields := []struct {
		name  string
		valid bool
	}{
		{name: "pushgateway.url", valid: cfg.Pushgateway.URL != ""},
		{name: "pushgateway.job", valid: cfg.Pushgateway.Job != ""},
		{name: "pushgateway.interval", valid: cfg.Pushgateway.Interval > 0},
		{name: "pushgateway.timeout", valid: cfg.Pushgateway.Timeout > 0},
		{name: "node_exporter.path", valid: cfg.NodeExporter.Path != ""},
		{name: "node_exporter.port", valid: cfg.NodeExporter.Port > 0},
		{name: "node_exporter.metrics_url", valid: cfg.NodeExporter.MetricsURL != ""},
	}

	for _, field := range requiredFields {
		if !field.valid {
			return fmt.Errorf("缺少必填配置项: %s", field.name)
		}
	}

	if cfg.ControlPlane.Enabled() && cfg.ControlPlane.HeartbeatInterval <= 0 {
		return fmt.Errorf("缺少必填配置项: control_plane.heartbeat_interval")
	}
	if !cfg.ControlPlane.Enabled() && cfg.ControlPlane.HeartbeatInterval > 0 {
		return fmt.Errorf("缺少必填配置项: control_plane.url")
	}
	if cfg.Hardware.Timeout <= 0 {
		return fmt.Errorf("缺少必填配置项: hardware.timeout")
	}
	if cfg.Update.Enabled {
		requiredFields := []struct {
			name  string
			valid bool
		}{
			{name: "update.listen_addr", valid: cfg.Update.ListenAddr != ""},
			{name: "update.status_file", valid: cfg.Update.StatusFile != ""},
			{name: "update.work_dir", valid: cfg.Update.WorkDir != ""},
		}
		for _, field := range requiredFields {
			if !field.valid {
				return fmt.Errorf("缺少必填配置项: %s", field.name)
			}
		}
		if len(cfg.Update.AllowedCIDRs) == 0 {
			return fmt.Errorf("缺少必填配置项: update.allowed_cidrs")
		}
	}

	return nil
}

func parseInt(key, value string, index int) (int, error) {
	i, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("配置项 %s 的值无效(第%d行): %w", key, index+1, err)
	}
	return i, nil
}

func parseBool(key, value string, index int) (bool, error) {
	b, err := strconv.ParseBool(value)
	if err != nil {
		return false, fmt.Errorf("配置项 %s 的值无效(第%d行): %w", key, index+1, err)
	}
	return b, nil
}

func parseCSV(value string) []string {
	if value == "" {
		return nil
	}

	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			result = append(result, part)
		}
	}
	return result
}
