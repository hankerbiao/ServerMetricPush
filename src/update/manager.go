package update

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const (
	UpdateTypeBinary = "binary_update"
	UpdateTypeConfig = "config_update"

	StatusIdle        = "idle"
	StatusAccepted    = "accepted"
	StatusDownloading = "downloading"
	StatusValidating  = "validating"
	StatusInstalling  = "installing"
	StatusRestarting  = "restarting"
	StatusSucceeded   = "succeeded"
	StatusFailed      = "failed"
	StatusRolledBack  = "rolled_back"
)

type Request struct {
	RequestID        string `json:"request_id"`
	UpdateType       string `json:"update_type"`
	TargetVersion    string `json:"target_version,omitempty"`
	DownloadURL      string `json:"download_url,omitempty"`
	FileName         string `json:"file_name,omitempty"`
	PackageType      string `json:"package_type,omitempty"`
	ConfigTemplateID string `json:"config_template_id,omitempty"`
	ConfigContent    string `json:"config_content,omitempty"`
	ConfigVersion    string `json:"config_version,omitempty"`
	Force            bool   `json:"force"`
}

type Task = Request

type StatusSnapshot struct {
	RequestID            string    `json:"request_id,omitempty"`
	UpdateType           string    `json:"update_type,omitempty"`
	Status               string    `json:"status"`
	CurrentVersion       string    `json:"current_version,omitempty"`
	CurrentConfigVersion string    `json:"current_config_version,omitempty"`
	TargetVersion        string    `json:"target_version,omitempty"`
	Error                string    `json:"error,omitempty"`
	RollbackPerformed    bool      `json:"rollback_performed"`
	InProgress           bool      `json:"in_progress"`
	StartedAt            time.Time `json:"started_at,omitempty"`
	FinishedAt           time.Time `json:"finished_at,omitempty"`
}

type Runner interface {
	Run(ctx context.Context, task Task) error
}

type Options struct {
	ListenAddr      string
	AllowedCIDRs    []string
	StatusFile      string
	WorkDir         string
	Runner          Runner
	VersionProvider func() string
}

type Manager struct {
	mu              sync.Mutex
	snapshot        StatusSnapshot
	statusFile      string
	runner          Runner
	versionProvider func() string
	allowedCIDRs    []*net.IPNet
	mux             *http.ServeMux
}

func NewManager(options Options) *Manager {
	manager := &Manager{
		statusFile:      options.StatusFile,
		runner:          options.Runner,
		versionProvider: options.VersionProvider,
	}
	manager.snapshot = StatusSnapshot{
		Status:         StatusIdle,
		CurrentVersion: providerValue(options.VersionProvider),
	}
	_ = manager.load()
	manager.allowedCIDRs = parseAllowedCIDRs(options.AllowedCIDRs)
	manager.mux = http.NewServeMux()
	manager.mux.HandleFunc("/internal/update", manager.handleUpdate)
	manager.mux.HandleFunc("/internal/update/status", manager.handleStatus)
	_ = manager.persist()
	return manager
}

func (m *Manager) Handler() http.Handler {
	return m.mux
}

func (m *Manager) Snapshot() StatusSnapshot {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.snapshot
}

func (m *Manager) Reconcile(currentVersion, currentConfigVersion string) StatusSnapshot {
	m.mu.Lock()
	defer m.mu.Unlock()

	if currentVersion != "" {
		m.snapshot.CurrentVersion = currentVersion
	}
	if currentConfigVersion != "" && m.snapshot.CurrentConfigVersion == "" {
		m.snapshot.CurrentConfigVersion = currentConfigVersion
	}
	if m.snapshot.InProgress {
		switch m.snapshot.UpdateType {
		case UpdateTypeBinary:
			if m.snapshot.TargetVersion != "" && m.snapshot.CurrentVersion == m.snapshot.TargetVersion {
				m.snapshot.Status = StatusSucceeded
				m.snapshot.InProgress = false
				m.snapshot.FinishedAt = time.Now().UTC()
				m.snapshot.Error = ""
			}
		case UpdateTypeConfig:
			if m.snapshot.TargetVersion != "" {
				m.snapshot.CurrentConfigVersion = m.snapshot.TargetVersion
				m.snapshot.Status = StatusSucceeded
				m.snapshot.InProgress = false
				m.snapshot.FinishedAt = time.Now().UTC()
				m.snapshot.Error = ""
			}
		}
		_ = m.persistLocked()
	}
	return m.snapshot
}

func (m *Manager) handleUpdate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !m.allowed(r.RemoteAddr) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	var request Request
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}
	if err := validateRequest(request); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	m.mu.Lock()
	if m.snapshot.InProgress {
		m.mu.Unlock()
		http.Error(w, "update already in progress", http.StatusConflict)
		return
	}
	m.snapshot = StatusSnapshot{
		RequestID:      request.RequestID,
		UpdateType:     request.UpdateType,
		Status:         StatusAccepted,
		CurrentVersion: providerValue(m.versionProvider),
		TargetVersion:  targetVersion(request),
		InProgress:     true,
		StartedAt:      time.Now().UTC(),
	}
	_ = m.persistLocked()
	m.mu.Unlock()

	go m.run(request)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	_ = json.NewEncoder(w).Encode(m.Snapshot())
}

func (m *Manager) handleStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(m.Snapshot())
}

func (m *Manager) run(task Task) {
	err := m.runner.Run(context.Background(), task)

	m.mu.Lock()
	defer m.mu.Unlock()
	m.snapshot.InProgress = false
	m.snapshot.FinishedAt = time.Now().UTC()
	if err != nil {
		m.snapshot.Status = StatusFailed
		m.snapshot.Error = err.Error()
	} else {
		m.snapshot.Status = StatusSucceeded
		if task.UpdateType == UpdateTypeConfig {
			m.snapshot.CurrentConfigVersion = task.ConfigVersion
		} else {
			m.snapshot.CurrentVersion = providerValue(m.versionProvider)
		}
	}
	_ = m.persistLocked()
}

func (m *Manager) persist() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.persistLocked()
}

func (m *Manager) persistLocked() error {
	if m.statusFile == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(m.statusFile), 0o755); err != nil {
		return err
	}
	body, err := json.MarshalIndent(m.snapshot, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(m.statusFile, body, 0o644)
}

func (m *Manager) load() error {
	if m.statusFile == "" {
		return nil
	}
	data, err := os.ReadFile(m.statusFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	var snapshot StatusSnapshot
	if err := json.Unmarshal(data, &snapshot); err != nil {
		return err
	}
	if snapshot.Status != "" {
		m.snapshot = snapshot
		if m.snapshot.CurrentVersion == "" {
			m.snapshot.CurrentVersion = providerValue(m.versionProvider)
		}
	}
	return nil
}

func (m *Manager) allowed(remoteAddr string) bool {
	if len(m.allowedCIDRs) == 0 {
		return true
	}
	host, _, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		host = remoteAddr
	}
	ip := net.ParseIP(host)
	if ip == nil {
		return false
	}
	for _, network := range m.allowedCIDRs {
		if network.Contains(ip) {
			return true
		}
	}
	return false
}

func parseAllowedCIDRs(values []string) []*net.IPNet {
	result := make([]*net.IPNet, 0, len(values))
	for _, value := range values {
		_, network, err := net.ParseCIDR(strings.TrimSpace(value))
		if err == nil {
			result = append(result, network)
		}
	}
	return result
}

func validateRequest(request Request) error {
	if request.RequestID == "" {
		return errors.New("request_id is required")
	}
	switch request.UpdateType {
	case UpdateTypeBinary:
		if request.TargetVersion == "" {
			return errors.New("target_version is required")
		}
		if request.DownloadURL == "" {
			return errors.New("download_url is required")
		}
	case UpdateTypeConfig:
		if request.ConfigContent == "" {
			return errors.New("config_content is required")
		}
		if request.ConfigVersion == "" {
			return errors.New("config_version is required")
		}
	default:
		return errors.New("unsupported update_type")
	}
	return nil
}

func targetVersion(request Request) string {
	if request.TargetVersion != "" {
		return request.TargetVersion
	}
	return request.ConfigVersion
}

func providerValue(fn func() string) string {
	if fn == nil {
		return ""
	}
	return fn()
}
