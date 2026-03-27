package main

/*
	node-push-exporter 主程序

	该程序通过启动 node_exporter 子进程来采集系统指标，
	并将指标数据推送到 Prometheus Pushgateway。

	支持的操作系统：Linux (node_exporter 支持的所有平台)

	工作流程：
	1. 启动 node_exporter 子进程(监听在本地9100端口)
	2. 定期从 node_exporter 的 /metrics 接口获取指标
	3. 将指标推送到 Prometheus Pushgateway

	指标来源：node_exporter (https://github.com/prometheus/node_exporter)
*/

import (
	"bytes"
	"flag"      // 命令行参数解析
	"fmt"       // 格式化输出
	"io"        // IO操作
	"log"       // 日志记录
	"net/http"  // HTTP客户端
	"os"        // 操作系统功能
	"os/exec"   // 执行子进程
	"os/signal" // 信号处理
	"path/filepath"
	"strings" // 字符串处理
	"syscall" // 系统调用
	"time"    // 时间处理

	"node-push-exporter/src/config" // 配置加载
	"node-push-exporter/src/pusher" // Pushgateway推送
)

// 程序版本和构建时间
var (
	version   = "dev"
	buildTime = "unknown"
)

const defaultConfigPath = "./config.yml"

// nodeExporterProcess 保存 node_exporter 进程信息
type nodeExporterProcess struct {
	cmd *exec.Cmd
}

// 主函数入口
func main() {
	// 默认从当前工作目录读取 config.yml，便于直接运行二进制文件做本地调试。
	// 系统部署场景仍然可以通过 -config 显式指定 /etc 下的配置文件。
	configPath := flag.String("config", defaultConfigPath, "配置文件路径")
	showVersion := flag.Bool("version", false, "显示版本信息")
	flag.Parse()

	// 如果指定了版本参数，显示版本后退出
	if *showVersion {
		fmt.Printf("node-push-exporter version %s (构建时间: %s)\n", version, buildTime)
		fmt.Printf("使用 node_exporter 采集指标\n")
		os.Exit(0)
	}

	// 加载配置文件
	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}

	// 输出启动信息，方便在 systemd 或容器日志中快速确认关键配置。
	log.Printf("启动 node-push-exporter 版本 %s", version)
	log.Printf("Pushgateway地址: %s", cfg.Pushgateway.URL)
	log.Printf("任务名称: %s, 推送间隔: %d秒", cfg.Pushgateway.Job, cfg.Pushgateway.Interval)

	// 启动 node_exporter 子进程
	exporter, err := startNodeExporter(cfg.NodeExporter.Path, cfg.NodeExporter.Port)
	if err != nil {
		log.Fatalf("启动 node_exporter 失败: %v", err)
	}
	defer exporter.Stop() // 程序退出时停止 node_exporter

	log.Printf("node_exporter 已启动，地址: %s", cfg.NodeExporter.MetricsURL)

	// Pushgateway 客户端只负责把已经抓到的 Prometheus 文本内容原样推送出去。
	pusherClient := pusher.NewPusher(
		cfg.Pushgateway.URL,
		pusher.WithJob(cfg.Pushgateway.Job),
		pusher.WithInstance(cfg.Pushgateway.Instance),
		pusher.WithTimeout(time.Duration(cfg.Pushgateway.Timeout)*time.Second),
	)

	// 创建定时器，用于定期推送指标
	ticker := time.NewTicker(time.Duration(cfg.Pushgateway.Interval) * time.Second)
	defer ticker.Stop()

	// 创建信号通道，用于处理优雅退出
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// 等待 node_exporter 完全启动
	time.Sleep(2 * time.Second)

	// 启动后先推一次，避免必须等待一个完整周期后才在 Pushgateway 中看到数据。
	if err := fetchAndPush(cfg.NodeExporter.MetricsURL, pusherClient); err != nil {
		log.Printf("首次推送失败: %v", err)
	}

	// 主循环：等待定时器或信号
	for {
		select {
		case <-ticker.C:
			// 定时器触发，获取并推送指标
			if err := fetchAndPush(cfg.NodeExporter.MetricsURL, pusherClient); err != nil {
				log.Printf("推送失败: %v", err)
			}
		case sig := <-sigChan:
			// 收到退出信号，优雅关闭
			log.Printf("收到信号 %v, 正在关闭...", sig)
			return
		}
	}
}

/*
startNodeExporter 启动 node_exporter 子进程

参数:
executablePath - node_exporter 可执行文件路径
port           - node_exporter 监听端口

返回值:
*nodeExporterProcess - node_exporter 进程管理对象
error - 启动过程中的错误
*/
func startNodeExporter(executablePath string, port int) (*nodeExporterProcess, error) {
	resolvedPath, err := resolveNodeExporterPath(executablePath)
	if err != nil {
		return nil, err
	}

	// 启动一个受当前进程托管的 node_exporter 子进程。
	// 这里只打开与当前用途直接相关的 collectors，保持行为清晰可控。
	cmd := exec.Command(resolvedPath,
		fmt.Sprintf("--web.listen-address=:%d", port),
		"--web.telemetry-path=/metrics",
		"--collector.cpu",
		"--collector.meminfo",
		"--collector.diskstats",
		"--collector.netdev",
		"--collector.filesystem",
		"--collector.loadavg",
		"--collector.stat",
		"--collector.time",
		"--collector.uname",
	)

	var stderr bytes.Buffer
	cmd.Stdout = io.Discard
	cmd.Stderr = &stderr

	// 启动进程
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("启动 node_exporter 失败: %w", err)
	}

	waitCh := make(chan error, 1)
	go func() {
		waitCh <- cmd.Wait()
	}()

	if err := waitForNodeExporterReady(port, waitCh, &stderr); err != nil {
		return nil, err
	}

	log.Printf("node_exporter 进程已启动，PID: %d", cmd.Process.Pid)

	return &nodeExporterProcess{
		cmd: cmd,
	}, nil
}

func resolveNodeExporterPath(executablePath string) (string, error) {
	if strings.Contains(executablePath, string(os.PathSeparator)) {
		if _, err := os.Stat(executablePath); err == nil {
			return executablePath, nil
		}
	}

	localPath := filepath.Join(".", executablePath)
	if _, err := os.Stat(localPath); err == nil {
		absPath, err := filepath.Abs(localPath)
		if err != nil {
			return "", fmt.Errorf("解析当前目录下的 node_exporter 路径失败: %w", err)
		}
		return absPath, nil
	}

	if resolvedPath, err := exec.LookPath(executablePath); err == nil {
		return resolvedPath, nil
	}

	possiblePaths := []string{
		"/usr/local/bin/node_exporter",
		"/usr/bin/node_exporter",
		"/opt/node_exporter/node_exporter",
	}

	for _, path := range possiblePaths {
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
	}

	return "", fmt.Errorf("未找到 node_exporter，请先安装: https://github.com/prometheus/node_exporter")
}

func waitForNodeExporterReady(port int, waitCh <-chan error, stderr *bytes.Buffer) error {
	deadline := time.Now().Add(3 * time.Second)
	metricsURL := fmt.Sprintf("http://127.0.0.1:%d/metrics", port)
	client := &http.Client{Timeout: 200 * time.Millisecond}

	for time.Now().Before(deadline) {
		select {
		case err := <-waitCh:
			return formatNodeExporterStartError("node_exporter 启动后立即退出", err, stderr)
		default:
		}

		resp, err := client.Get(metricsURL)
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return nil
			}
		}

		time.Sleep(100 * time.Millisecond)
	}

	select {
	case err := <-waitCh:
		return formatNodeExporterStartError("node_exporter 启动后立即退出", err, stderr)
	default:
	}

	stderrOutput := strings.TrimSpace(stderr.String())
	if stderrOutput != "" {
		return fmt.Errorf("node_exporter 启动超时，%s 内未能在端口 %d 提供 /metrics，stderr: %s", 3*time.Second, port, stderrOutput)
	}

	return fmt.Errorf("node_exporter 启动超时，%s 内未能在端口 %d 提供 /metrics", 3*time.Second, port)
}

func formatNodeExporterStartError(prefix string, err error, stderr *bytes.Buffer) error {
	stderrOutput := strings.TrimSpace(stderr.String())
	if stderrOutput != "" {
		return fmt.Errorf("%s: %w, stderr: %s", prefix, err, stderrOutput)
	}
	return fmt.Errorf("%s: %w", prefix, err)
}

// Stop 停止 node_exporter 进程
func (p *nodeExporterProcess) Stop() {
	if p.cmd != nil && p.cmd.Process != nil {
		// 主程序退出时显式结束子进程，避免遗留孤儿 node_exporter。
		log.Printf("停止 node_exporter 进程，PID: %d", p.cmd.Process.Pid)
		p.cmd.Process.Kill()
		p.cmd.Wait()
	}
}

/*
fetchAndPush 从 node_exporter 获取指标并推送到 Pushgateway

参数:
metricsURL - node_exporter 的 metrics 接口地址
pusher     - Pushgateway 推送客户端

返回值:
error - 推送过程中的错误信息
*/
func fetchAndPush(metricsURL string, pusher *pusher.Pusher) error {
	// 先抓取本机 node_exporter 指标，再直接转交给 Pushgateway。
	metrics, err := fetchMetrics(metricsURL)
	if err != nil {
		return fmt.Errorf("获取指标失败: %w", err)
	}

	// 推送到 Pushgateway
	if err := pusher.Push([]byte(metrics)); err != nil {
		return fmt.Errorf("推送失败: %w", err)
	}

	log.Printf("指标推送成功，来源: %s", metricsURL)
	return nil
}

/*
fetchMetrics 从指定 URL 获取 Prometheus 格式的指标数据

参数:
url - metrics 接口地址

返回值:
string - 指标数据文本
error - 获取过程中的错误
*/
func fetchMetrics(url string) (string, error) {
	// 读取本机 metrics 时仍然设置超时，防止 node_exporter 卡住时主循环被长期阻塞。
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	// 发送 GET 请求
	resp, err := client.Get(url)
	if err != nil {
		return "", fmt.Errorf("HTTP请求失败: %w", err)
	}
	defer resp.Body.Close()

	// 只有 200 才视为抓取成功，其他状态码都应当进入日志告警。
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP状态码错误: %d", resp.StatusCode)
	}

	// 读取响应内容
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("读取响应失败: %w", err)
	}

	// 转换为字符串
	metrics := string(body)

	// 空响应通常意味着 exporter 异常或拿到了非预期内容，这里直接视为错误。
	if strings.TrimSpace(metrics) == "" {
		return "", fmt.Errorf("指标数据为空")
	}

	return metrics, nil
}
