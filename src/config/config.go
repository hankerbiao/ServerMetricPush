package config

/*
	配置加载模块

	该模块负责从配置文件加载应用程序配置。
	配置文件采用简单的key=value格式，便于手动编辑和脚本生成。

	配置项说明:
	  - pushgateway: Pushgateway服务器配置
	  - node_exporter: node_exporter 相关配置(可选)
*/

import (
	"fmt"     // 格式化错误
	"os"      // 文件操作
	"strconv" // 字符串转数值
	"strings" // 字符串处理
)

// Config 应用程序配置结构
type Config struct {
	Pushgateway  PushgatewayConfig  // Pushgateway相关配置
	NodeExporter NodeExporterConfig // node_exporter相关配置
}

// PushgatewayConfig Pushgateway服务器配置
type PushgatewayConfig struct {
	URL      string // Pushgateway地址，如http://localhost:9091
	Job      string // 任务名称，用于标识这批指标
	Instance string // 实例名称，如主机名
	Interval int    // 推送间隔(秒)
	Timeout  int    // HTTP请求超时(秒)
}

// NodeExporterConfig node_exporter配置
type NodeExporterConfig struct {
	Path       string // node_exporter可执行文件路径
	Port       int    // node_exporter监听端口
	MetricsURL string // 主程序抓取 node_exporter 指标时使用的完整URL
}

/*
Load 从配置文件加载配置

参数:
path - 配置文件路径

返回:
*Config - 解析后的配置对象
error - 加载或解析错误
*/
func Load(path string) (*Config, error) {
	// 读取配置文件
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("读取配置文件失败: %w", err)
	}

	// 当前版本不再做默认值兜底。
	// 这样可以强制部署配置显式声明关键参数，避免“看似启动成功，实际推错地址”的隐患。
	cfg := &Config{}

	// 解析配置文件内容
	lines := strings.Split(string(data), "\n")
	for index, line := range lines {
		// 去除首尾空白
		line = strings.TrimSpace(line)
		// 跳过空行和注释行
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// 配置文件采用最简单的 key=value 格式，便于 shell 脚本直接生成。
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		// 提取键和值
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		// 仅识别当前主流程真正使用到的配置项，其余键会被忽略。
		switch key {
		case "pushgateway.url":
			cfg.Pushgateway.URL = value
		case "pushgateway.job":
			cfg.Pushgateway.Job = value
		case "pushgateway.instance":
			cfg.Pushgateway.Instance = value
		case "pushgateway.interval":
			i, err := strconv.Atoi(value)
			if err != nil {
				return nil, fmt.Errorf("配置项 %s 的值无效(第%d行): %w", key, index+1, err)
			}
			cfg.Pushgateway.Interval = i
		case "pushgateway.timeout":
			i, err := strconv.Atoi(value)
			if err != nil {
				return nil, fmt.Errorf("配置项 %s 的值无效(第%d行): %w", key, index+1, err)
			}
			cfg.Pushgateway.Timeout = i
		case "node_exporter.path":
			cfg.NodeExporter.Path = value
		case "node_exporter.port":
			i, err := strconv.Atoi(value)
			if err != nil {
				return nil, fmt.Errorf("配置项 %s 的值无效(第%d行): %w", key, index+1, err)
			}
			cfg.NodeExporter.Port = i
		case "node_exporter.metrics_url":
			cfg.NodeExporter.MetricsURL = value
		}
	}

	if err := validate(cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}

func validate(cfg *Config) error {
	// 统一做必填校验，让 Load 的调用方只需要处理“配置可用/不可用”这一层结果。
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

	return nil
}
