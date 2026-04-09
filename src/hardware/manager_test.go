package hardware

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

type stubCommandExecutor struct {
	outputs map[string][]byte
	errs    map[string]error
	paths   map[string]string
}

func (s stubCommandExecutor) LookPath(file string) (string, error) {
	if path, ok := s.paths[file]; ok {
		return path, nil
	}
	return "", fmt.Errorf("%s not found", file)
}

func (s stubCommandExecutor) CombinedOutput(_ time.Duration, name string, args ...string) ([]byte, error) {
	key := name + " " + strings.Join(args, " ")
	if err := s.errs[key]; err != nil {
		return nil, err
	}
	if output, ok := s.outputs[key]; ok {
		return output, nil
	}
	return nil, nil
}

func TestManager_Collect_EmitsHardwareInventoryMetrics(t *testing.T) {
	rootDir := t.TempDir()
	writeTestFile(t, filepath.Join(rootDir, "sys/class/dmi/id/system_vendor"), "Acme")
	writeTestFile(t, filepath.Join(rootDir, "sys/class/dmi/id/product_name"), "RackServer 9000")
	writeTestFile(t, filepath.Join(rootDir, "sys/class/dmi/id/product_serial"), "SN123")
	writeTestFile(t, filepath.Join(rootDir, "proc/meminfo"), "MemTotal:       32768000 kB\n")
	writeTestFile(t, filepath.Join(rootDir, "proc/cpuinfo"), strings.Join([]string{
		"processor\t: 0",
		"physical id\t: 0",
		"vendor_id\t: GenuineIntel",
		"model name\t: Intel(R) Xeon(R)",
		"cpu cores\t: 16",
		"",
		"processor\t: 1",
		"physical id\t: 1",
		"vendor_id\t: GenuineIntel",
		"model name\t: Intel(R) Xeon(R)",
		"cpu cores\t: 16",
		"",
	}, "\n"))
	writeTestFile(t, filepath.Join(rootDir, "sys/class/net/eth0/address"), "00:11:22:33:44:55\n")
	writeTestFile(t, filepath.Join(rootDir, "sys/class/net/eth0/speed"), "10000\n")
	writeTestFile(t, filepath.Join(rootDir, "sys/class/net/eth0/device/vendor"), "0x8086\n")
	writeTestFile(t, filepath.Join(rootDir, "sys/class/net/eth0/device/device"), "0x100f\n")
	writeTestFile(t, filepath.Join(rootDir, "sys/class/net/lo/address"), "00:00:00:00:00:00\n")

	manager := &Manager{
		timeout: 2 * time.Second,
		executor: stubCommandExecutor{
			paths: map[string]string{
				"dmidecode": "dmidecode",
				"lsblk":     "lsblk",
			},
			outputs: map[string][]byte{
				"dmidecode --type memory": []byte(strings.Join([]string{
					"Memory Device",
					"\tSize: 16384 MB",
					"\tForm Factor: DIMM",
					"\tLocator: DIMM_A1",
					"\tBank Locator: NODE 1",
					"\tType: DDR4",
					"\tType Detail: Synchronous Unbuffered (Unregistered)",
					"\tSpeed: 3200 MT/s",
					"\tManufacturer: Samsung",
					"\tPart Number: M393A2K40DB3-CWE",
					"\tSerial Number: MEM-001",
					"",
					"Memory Device",
					"\tSize: 16384 MB",
					"\tForm Factor: DIMM",
					"\tLocator: DIMM_B1",
					"\tBank Locator: NODE 1",
					"\tType: DDR4",
					"\tSpeed: 3200 MT/s",
					"\tManufacturer: Samsung",
					"\tPart Number: M393A2K40DB3-CWE",
					"\tSerial Number: MEM-002",
				}, "\n")),
				"lsblk -J -o NAME,TYPE,MODEL,VENDOR,SERIAL,SIZE,ROTA,WWN,TRAN": []byte(`{"blockdevices":[{"name":"sda","type":"disk","model":"SSD Pro","vendor":"ACME","serial":"DISK-001","size":"1073741824","rota":false,"wwn":"wwn-1","tran":"sata"}]}`),
			},
		},
		now:                   func() time.Time { return time.Unix(1712131200, 0) },
		dmiDir:                filepath.Join(rootDir, "sys/class/dmi/id"),
		procCPUInfoPath:       filepath.Join(rootDir, "proc/cpuinfo"),
		procMemInfoPath:       filepath.Join(rootDir, "proc/meminfo"),
		netClassDir:           filepath.Join(rootDir, "sys/class/net"),
		includeSerials:        true,
		includeVirtualDevices: false,
	}

	metrics, err := manager.Collect()
	if err != nil {
		t.Fatalf("Collect() error = %v", err)
	}

	for _, want := range []string{
		`node_push_exporter_hardware_scrape_success 1`,
		`node_hardware_host_info{product_name="RackServer 9000",product_serial="SN123",system_vendor="Acme"} 1`,
		`node_hardware_cpu_info{model_name="Intel(R) Xeon(R)",vendor_id="GenuineIntel"} 1`,
		`node_hardware_cpu_packages 2`,
		`node_hardware_cpu_cores 32`,
		`node_hardware_memory_info{total_bytes="33554432000"} 1`,
		`node_hardware_memory_total_bytes 3.3554432e+10`,
		`node_hardware_memory_dimm_info{bank_locator="NODE 1",form_factor="DIMM",locator="DIMM_A1",manufacturer="Samsung",part_number="M393A2K40DB3-CWE",serial="MEM-001",size_bytes="17179869184",speed_mt_s="3200",type="DDR4",type_detail="Synchronous Unbuffered (Unregistered)"} 1`,
		`node_hardware_memory_dimm_info{bank_locator="NODE 1",form_factor="DIMM",locator="DIMM_B1",manufacturer="Samsung",part_number="M393A2K40DB3-CWE",serial="MEM-002",size_bytes="17179869184",speed_mt_s="3200",type="DDR4"} 1`,
		`node_hardware_memory_dimm_size_bytes{locator="DIMM_A1"} 1.7179869184e+10`,
		`node_hardware_memory_dimm_size_bytes{locator="DIMM_B1"} 1.7179869184e+10`,
		`node_hardware_disk_info{device="sda",model="SSD Pro",serial="DISK-001",size_bytes="1073741824",transport="sata",vendor="ACME",wwn="wwn-1"} 1`,
		`node_hardware_disk_size_bytes{device="sda"} 1.073741824e+09`,
		`node_hardware_nic_info{address="00:11:22:33:44:55",device_id="0x100f",iface="eth0",speed_bits="10000000000",vendor_id="0x8086"} 1`,
		`node_hardware_nic_speed_bits{iface="eth0"} 1e+10`,
	} {
		if !strings.Contains(metrics, want) {
			t.Fatalf("Collect() output missing %q\nfull output:\n%s", want, metrics)
		}
	}

	if strings.Contains(metrics, `iface="lo"`) {
		t.Fatalf("Collect() output included loopback interface:\n%s", metrics)
	}
}

func TestManager_Collect_OmitsMemorySerialsWhenDisabled(t *testing.T) {
	manager := &Manager{
		timeout: 2 * time.Second,
		executor: stubCommandExecutor{
			paths: map[string]string{
				"dmidecode": "dmidecode",
			},
			outputs: map[string][]byte{
				"dmidecode --type memory": []byte(strings.Join([]string{
					"Memory Device",
					"\tSize: 8192 MB",
					"\tLocator: DIMM_A1",
					"\tManufacturer: Kingston",
					"\tPart Number: KSM32RS8/8HDR",
					"\tSerial Number: MEM-001",
				}, "\n")),
			},
		},
		includeSerials: false,
	}

	metrics, err := manager.collectMemoryMetrics()
	if err != nil {
		t.Fatalf("collectMemoryMetrics() error = %v", err)
	}
	if len(metrics) != 2 {
		t.Fatalf("collectMemoryMetrics() len = %d, want 2", len(metrics))
	}
	if strings.Contains(strings.Join(metrics, "\n"), `serial="MEM-001"`) {
		t.Fatalf("collectMemoryMetrics() included serial when disabled:\n%s", strings.Join(metrics, "\n"))
	}
}

func TestManager_CollectMemoryMetrics_FallsBackToKnownDmidecodePaths(t *testing.T) {
	manager := &Manager{
		timeout: 2 * time.Second,
		executor: stubCommandExecutor{
			paths: map[string]string{
				"/usr/sbin/dmidecode": "/usr/sbin/dmidecode",
			},
			outputs: map[string][]byte{
				"/usr/sbin/dmidecode --type memory": []byte(strings.Join([]string{
					"Memory Device",
					"\tSize: 65536 MB",
					"\tLocator: CPU0_DIMMA0",
					"\tBank Locator: P0 CHANNEL A",
					"\tManufacturer: Samsung",
					"\tPart Number: M321R8GA0PB0-CWMCJ",
					"\tType: DDR5",
					"\tSpeed: 5600 MT/s",
				}, "\n")),
			},
		},
		includeSerials: true,
	}

	metrics, err := manager.collectMemoryMetrics()
	if err != nil {
		t.Fatalf("collectMemoryMetrics() error = %v", err)
	}
	if len(metrics) != 2 {
		t.Fatalf("collectMemoryMetrics() len = %d, want 2", len(metrics))
	}
	joined := strings.Join(metrics, "\n")
	if !strings.Contains(joined, `node_hardware_memory_dimm_info{bank_locator="P0 CHANNEL A",locator="CPU0_DIMMA0",manufacturer="Samsung",part_number="M321R8GA0PB0-CWMCJ",size_bytes="68719476736",speed_mt_s="5600",type="DDR5"} 1`) {
		t.Fatalf("collectMemoryMetrics() missing DIMM info metric:\n%s", joined)
	}
	if !strings.Contains(joined, `node_hardware_memory_dimm_size_bytes{locator="CPU0_DIMMA0"} 6.8719476736e+10`) {
		t.Fatalf("collectMemoryMetrics() missing DIMM size metric:\n%s", joined)
	}
}

func writeTestFile(t *testing.T, path, content string) {
	t.Helper()

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("os.MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("os.WriteFile() error = %v", err)
	}
}
