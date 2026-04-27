package process

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// Config 保存进程配置。
type Config struct {
	ExecutablePath string
	Port           int
}

// Process 管理子进程。
type Process struct {
	cmd *exec.Cmd
}

// Start 启动配置的进程。
func Start(cfg Config) (*Process, error) {
	resolvedPath, err := resolvePath(cfg.ExecutablePath)
	if err != nil {
		return nil, err
	}

	cmd := exec.Command(resolvedPath,
		fmt.Sprintf("--web.listen-address=:%d", cfg.Port),
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

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("启动失败: %w", err)
	}

	waitCh := make(chan error, 1)
	go func() {
		waitCh <- cmd.Wait()
	}()

	if err := waitForReady(cfg.Port, waitCh, &stderr); err != nil {
		return nil, err
	}

	log.Printf("进程已启动，PID: %d", cmd.Process.Pid)
	return &Process{cmd: cmd}, nil
}

// Stop 终止进程。
func (p *Process) Stop() {
	if p.cmd != nil && p.cmd.Process != nil {
		log.Printf("停止进程，PID: %d", p.cmd.Process.Pid)
		p.cmd.Process.Kill()
		p.cmd.Wait()
	}
}

func resolvePath(executablePath string) (string, error) {
	// 检查是否为绝对/相对路径
	if strings.Contains(executablePath, string(os.PathSeparator)) {
		if _, err := os.Stat(executablePath); err == nil {
			return executablePath, nil
		}
	}

	// 在当前目录中检查
	localPath := filepath.Join(".", executablePath)
	if _, err := os.Stat(localPath); err == nil {
		absPath, err := filepath.Abs(localPath)
		if err != nil {
			return "", fmt.Errorf("解析路径失败: %w", err)
		}
		return absPath, nil
	}

	// 在 PATH 中检查
	if resolvedPath, err := exec.LookPath(executablePath); err == nil {
		return resolvedPath, nil
	}

	// 检查常见路径
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

	return "", fmt.Errorf("未找到可执行文件，请先安装")
}

func waitForReady(port int, waitCh <-chan error, stderr *bytes.Buffer) error {
	deadline := time.Now().Add(3 * time.Second)
	metricsURL := fmt.Sprintf("http://127.0.0.1:%d/metrics", port)
	client := &http.Client{Timeout: 200 * time.Millisecond}

	for time.Now().Before(deadline) {
		select {
		case err := <-waitCh:
			return formatError("进程启动后立即退出", err, stderr)
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
		return formatError("进程启动后立即退出", err, stderr)
	default:
	}

	stderrOutput := strings.TrimSpace(stderr.String())
	if stderrOutput != "" {
		return fmt.Errorf("启动超时，%s 内未能在端口 %d 提供 /metrics，stderr: %s", 3*time.Second, port, stderrOutput)
	}

	return fmt.Errorf("启动超时，%s 内未能在端口 %d 提供 /metrics", 3*time.Second, port)
}

func formatError(prefix string, err error, stderr *bytes.Buffer) error {
	stderrOutput := strings.TrimSpace(stderr.String())
	if stderrOutput != "" {
		return fmt.Errorf("%s: %w, stderr: %s", prefix, err, stderrOutput)
	}
	return fmt.Errorf("%s: %w", prefix, err)
}
