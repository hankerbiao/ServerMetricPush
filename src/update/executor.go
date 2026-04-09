package update

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
)

type Downloader func(ctx context.Context, url, dest string) error

type ServiceManager interface {
	Restart(ctx context.Context) error
}

type ConfigValidator func(path string) error

type ExecutorOptions struct {
	BinaryPath           string
	ConfigPath           string
	CurrentVersion       string
	CurrentConfigVersion string
	Downloader           Downloader
	ServiceManager       ServiceManager
	ConfigValidator      ConfigValidator
}

type Executor struct {
	binaryPath           string
	configPath           string
	currentVersion       string
	currentConfigVersion string
	downloader           Downloader
	serviceManager       ServiceManager
	configValidator      ConfigValidator
}

func NewExecutor(options ExecutorOptions) *Executor {
	return &Executor{
		binaryPath:           options.BinaryPath,
		configPath:           options.ConfigPath,
		currentVersion:       options.CurrentVersion,
		currentConfigVersion: options.CurrentConfigVersion,
		downloader:           options.Downloader,
		serviceManager:       options.ServiceManager,
		configValidator:      options.ConfigValidator,
	}
}

func (e *Executor) Run(ctx context.Context, task Task) error {
	switch task.UpdateType {
	case UpdateTypeBinary:
		return e.runBinaryUpdate(ctx, task)
	case UpdateTypeConfig:
		return e.runConfigUpdate(ctx, task)
	default:
		return fmt.Errorf("unsupported update type: %s", task.UpdateType)
	}
}

func (e *Executor) runBinaryUpdate(ctx context.Context, task Task) error {
	if e.downloader == nil {
		return fmt.Errorf("binary downloader is not configured")
	}
	tempPath := filepath.Join(filepath.Dir(e.binaryPath), fmt.Sprintf(".%s.download", filepath.Base(e.binaryPath)))
	downloadPath := tempPath
	if task.PackageType == "tar.gz" {
		downloadPath = tempPath + ".tar.gz"
	}
	backupPath := e.binaryPath + ".bak"

	if err := e.downloader(ctx, task.DownloadURL, downloadPath); err != nil {
		return fmt.Errorf("download binary failed: %w", err)
	}
	if task.PackageType == "tar.gz" {
		if err := extractBinaryFromTarGz(downloadPath, filepath.Base(e.binaryPath), tempPath); err != nil {
			_ = os.Remove(downloadPath)
			return fmt.Errorf("extract binary failed: %w", err)
		}
		_ = os.Remove(downloadPath)
	}
	if err := copyFile(e.binaryPath, backupPath, 0o755); err != nil {
		return fmt.Errorf("backup binary failed: %w", err)
	}
	if err := os.Rename(tempPath, e.binaryPath); err != nil {
		_ = os.Remove(tempPath)
		return fmt.Errorf("replace binary failed: %w", err)
	}
	if err := os.Chmod(e.binaryPath, 0o755); err != nil {
		_ = restoreFileWithMode(backupPath, e.binaryPath, 0o755)
		return fmt.Errorf("chmod binary failed: %w", err)
	}
	if err := e.restart(ctx); err != nil {
		_ = restoreFileWithMode(backupPath, e.binaryPath, 0o755)
		return err
	}
	_ = os.Remove(backupPath)
	return nil
}

func (e *Executor) runConfigUpdate(ctx context.Context, task Task) error {
	tempPath := e.configPath + ".tmp"
	backupPath := e.configPath + ".bak"

	if err := os.MkdirAll(filepath.Dir(e.configPath), 0o755); err != nil {
		return fmt.Errorf("prepare config dir failed: %w", err)
	}
	if err := os.WriteFile(tempPath, []byte(task.ConfigContent), 0o644); err != nil {
		return fmt.Errorf("write temp config failed: %w", err)
	}
	if e.configValidator != nil {
		if err := e.configValidator(tempPath); err != nil {
			_ = os.Remove(tempPath)
			return fmt.Errorf("validate config failed: %w", err)
		}
	}
	if _, err := os.Stat(e.configPath); err == nil {
		if err := copyFile(e.configPath, backupPath, 0o644); err != nil {
			_ = os.Remove(tempPath)
			return fmt.Errorf("backup config failed: %w", err)
		}
	}
	if err := os.Rename(tempPath, e.configPath); err != nil {
		_ = os.Remove(tempPath)
		return fmt.Errorf("replace config failed: %w", err)
	}
	if err := e.restart(ctx); err != nil {
		if _, statErr := os.Stat(backupPath); statErr == nil {
			_ = restoreFileWithMode(backupPath, e.configPath, 0o644)
		}
		return err
	}
	_ = os.Remove(backupPath)
	return nil
}

func (e *Executor) restart(ctx context.Context) error {
	if e.serviceManager == nil {
		return nil
	}
	if err := e.serviceManager.Restart(ctx); err != nil {
		return fmt.Errorf("restart service failed: %w", err)
	}
	return nil
}

func copyFile(src, dest string, mode os.FileMode) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dest, data, mode)
}

func restoreFileWithMode(src, dest string, mode os.FileMode) error {
	if err := copyFile(src, dest, mode); err != nil {
		return err
	}
	return os.Remove(src)
}

func HTTPDownloader(ctx context.Context, url, dest string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return err
	}
	file, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = io.Copy(file, resp.Body)
	return err
}

type SystemdServiceManager struct {
	ServiceName string
}

func (s SystemdServiceManager) Restart(ctx context.Context) error {
	serviceName := s.ServiceName
	if serviceName == "" {
		serviceName = "node-push-exporter"
	}
	cmd := exec.CommandContext(ctx, "systemctl", "restart", serviceName)
	return cmd.Run()
}

func extractBinaryFromTarGz(archivePath, binaryName, destPath string) error {
	file, err := os.Open(archivePath)
	if err != nil {
		return err
	}
	defer file.Close()

	gzipReader, err := gzip.NewReader(file)
	if err != nil {
		return err
	}
	defer gzipReader.Close()

	tarReader := tar.NewReader(gzipReader)
	for {
		header, err := tarReader.Next()
		if err != nil {
			if err == io.EOF {
				break
			}
			return err
		}
		if filepath.Base(header.Name) != binaryName {
			continue
		}
		output, err := os.Create(destPath)
		if err != nil {
			return err
		}
		if _, err := io.Copy(output, tarReader); err != nil {
			output.Close()
			return err
		}
		if err := output.Close(); err != nil {
			return err
		}
		return os.Chmod(destPath, 0o755)
	}
	return fmt.Errorf("binary %q not found in archive", binaryName)
}
