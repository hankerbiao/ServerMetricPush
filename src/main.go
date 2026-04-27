package main

import (
	"crypto/sha256"
	"encoding/hex"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"runtime"
	"strings"
	"syscall"
	"time"

	"node-push-exporter/src/config"
	"node-push-exporter/src/controlplane"
	"node-push-exporter/src/exporter"
	"node-push-exporter/src/process"
)

var (
	version   = "dev"
	buildTime = "unknown"
)

const defaultConfigPath = "./config.yml"

func main() {
	configPath := flag.String("config", defaultConfigPath, "配置文件路径")
	showVersion := flag.Bool("version", false, "显示版本信息")
	flag.Parse()

	if *showVersion {
		fmt.Printf("node-push-exporter version %s (构建时间: %s)\n", version, buildTime)
		fmt.Printf("使用 node_exporter 采集指标\n")
		os.Exit(0)
	}

	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}

	log.Printf("启动 node-push-exporter 版本 %s", version)
	log.Printf("Pushgateway地址: %s", cfg.Pushgateway.URL)
	log.Printf("任务名称: %s, 推送间隔: %d秒", cfg.Pushgateway.Job, cfg.Pushgateway.Interval)

	pushInstance, err := effectivePushInstance(cfg.Pushgateway.Instance)
	if err != nil {
		log.Fatalf("获取节点IP失败: %v", err)
	}
	log.Printf("Pushgateway实例标识: %s", pushInstance)

	// 启动 node_exporter 进程
	var nodeExporter *process.Process
	nodeExporter, err = process.Start(process.Config{
		ExecutablePath: cfg.NodeExporter.Path,
		Port:           cfg.NodeExporter.Port,
	})
	if err != nil {
		log.Fatalf("启动 node_exporter 失败: %v", err)
	}
	defer nodeExporter.Stop()
	log.Printf("node_exporter 已启动，地址: %s", cfg.NodeExporter.MetricsURL)

	// 创建导出器
	exp := exporter.New(exporter.Config{
		NodeExporterMetricsURL: cfg.NodeExporter.MetricsURL,
		PushURL:                cfg.Pushgateway.URL,
		PushJob:                cfg.Pushgateway.Job,
		PushInstance:           pushInstance,
		PushTimeout:            time.Duration(cfg.Pushgateway.Timeout) * time.Second,
		GPUEnabled:             true,
	})

	// 设置控制面
	var controlPlaneClient *controlplane.Client
	var registerRequest controlplane.RegisterRequest
	var heartbeatCh <-chan time.Time
	registered := false
	if cfg.ControlPlane.Enabled() {
		controlPlaneClient = controlplane.NewClient(
			cfg.ControlPlane.URL,
			time.Duration(cfg.Pushgateway.Timeout)*time.Second,
		)
		registerRequest, err = buildRegisterRequest(cfg)
		if err != nil {
			log.Fatalf("获取节点信息失败: %v", err)
		}
		if err := controlPlaneClient.Register(registerRequest); err != nil {
			log.Printf("控制面注册失败，后续会继续重试: %v", err)
		} else {
			registered = true
			log.Printf("控制面注册成功，节点ID: %s", registerRequest.AgentID)
		}

		heartbeatTicker := time.NewTicker(time.Duration(cfg.ControlPlane.HeartbeatInterval) * time.Second)
		defer heartbeatTicker.Stop()
		heartbeatCh = heartbeatTicker.C
	}

	// 主循环
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	ticker := time.NewTicker(time.Duration(cfg.Pushgateway.Interval) * time.Second)
	defer ticker.Stop()

	time.Sleep(2 * time.Second)

	// 首次推送
	if err := exp.CollectAndPush(); err != nil {
		log.Printf("首次推送失败: %v", err)
	}

	for {
		select {
		case <-ticker.C:
			if err := exp.CollectAndPush(); err != nil {
				log.Printf("推送失败: %v", err)
			}
		case <-heartbeatCh:
			if controlPlaneClient == nil {
				continue
			}

			if !registered {
				if err := controlPlaneClient.Register(registerRequest); err != nil {
					log.Printf("控制面注册重试失败: %v", err)
					continue
				}
				registered = true
				log.Printf("控制面注册成功，节点ID: %s", registerRequest.AgentID)
			}

			heartbeatPayload := exp.Runtime().Snapshot(registerRequest.AgentID)
			if err := controlPlaneClient.Heartbeat(heartbeatPayload); err != nil {
				if apiErr, ok := err.(*controlplane.APIError); ok && apiErr.StatusCode == 404 {
					registered = false
					log.Printf("控制面心跳返回节点不存在，下次周期重新注册: %v", err)
					continue
				}
				log.Printf("控制面心跳失败: %v", err)
				continue
			}
		case sig := <-sigChan:
			log.Printf("收到信号 %v, 正在关闭...", sig)
			return
		}
	}
}

func effectivePushInstance(configured string) (string, error) {
	if configured != "" {
		return configured, nil
	}
	ip, err := detectNodeIP()
	if err != nil {
		return "", err
	}
	return ip, nil
}

func buildRegisterRequest(cfg *config.Config) (controlplane.RegisterRequest, error) {
	hostname, _ := os.Hostname()
	if hostname == "" {
		hostname = "unknown-host"
	}

	ip, err := detectNodeIP()
	if err != nil {
		return controlplane.RegisterRequest{}, err
	}

	return controlplane.RegisterRequest{
		AgentID:                buildAgentID(hostname),
		Hostname:               hostname,
		Version:                version,
		OS:                     runtime.GOOS,
		Arch:                   runtime.GOARCH,
		IP:                     ip,
		PushgatewayURL:         cfg.Pushgateway.URL,
		PushIntervalSeconds:    cfg.Pushgateway.Interval,
		NodeExporterPort:       cfg.NodeExporter.Port,
		NodeExporterMetricsURL: cfg.NodeExporter.MetricsURL,
		StartedAt:              time.Now().UTC(),
	}, nil
}

func buildAgentID(hostname string) string {
	seed := readMachineID()
	if seed == "" {
		seed = hostname
	}
	sum := sha256.Sum256([]byte("node-push-exporter:" + seed))
	return hex.EncodeToString(sum[:16])
}

func readMachineID() string {
	paths := []string{"/etc/machine-id", "/var/lib/dbus/machine-id"}
	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err == nil {
			return normalizeMachineID(string(data))
		}
	}
	return ""
}

func normalizeMachineID(value string) string {
	return strings.TrimSpace(value)
}

func detectNodeIP() (string, error) {
	interfaces, err := net.Interfaces()
	if err != nil {
		return "", fmt.Errorf("获取网络接口失败: %v", err)
	}

	var reasons []string

	for _, iface := range interfaces {
		if iface.Flags&net.FlagUp == 0 {
			reasons = append(reasons, fmt.Sprintf("接口 %s (未启用)", iface.Name))
			continue
		}
		if iface.Flags&net.FlagLoopback != 0 {
			reasons = append(reasons, fmt.Sprintf("接口 %s (回环)", iface.Name))
			continue
		}

		addrs, err := iface.Addrs()
		if err != nil {
			reasons = append(reasons, fmt.Sprintf("接口 %s 地址读取失败: %v", iface.Name, err))
			continue
		}

		for _, addr := range addrs {
			ip := extractIP(addr)
			if ip == nil {
				reasons = append(reasons, fmt.Sprintf("接口 %s 地址类型不支持", iface.Name))
				continue
			}
			if ip.IsLoopback() {
				reasons = append(reasons, fmt.Sprintf("接口 %s 地址 %s (回环)", iface.Name, ip.String()))
				continue
			}
			if ipv4 := ip.To4(); ipv4 != nil {
				return ipv4.String(), nil
			} else {
				reasons = append(reasons, fmt.Sprintf("接口 %s 地址 %s (非IPv4)", iface.Name, ip.String()))
			}
		}
	}

	if len(reasons) > 0 {
		return "", fmt.Errorf("无法获取有效IP地址，原因: %s", strings.Join(reasons, "; "))
	}
	return "", fmt.Errorf("未找到任何有效网络接口")
}

func extractIP(addr net.Addr) net.IP {
	switch value := addr.(type) {
	case *net.IPNet:
		return value.IP
	case *net.IPAddr:
		return value.IP
	default:
		return nil
	}
}