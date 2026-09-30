package ix

import (
	"encoding/xml"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strings"

	"github.com/hawkli-1994/gpu_tools/pkg/gpu"
)

func init() {
	smiPaths = scanCorexSmiPaths("/usr/local")
	gpu.Register(&ixGPU{})
}

var logger = slog.New(slog.NewTextHandler(os.Stdout, nil))
var smiPaths = []string{}

// scanCorexSmiPaths 扫描指定目录下所有以 corex 开头的文件夹，并返回对应的 ixsmi 路径
func scanCorexSmiPaths(rootDir string) []string {
	var paths []string

	logger.Info("scanCorexSmiPaths: scanning directory", "dir", rootDir)
	dirs, err := os.ReadDir(rootDir)
	if err != nil {
		logger.Error("scanCorexSmiPaths: failed to read directory", "dir", rootDir, "error", err)
		return paths
	}

	for _, d := range dirs {
		logger.Info("scanCorexSmiPaths: checking dir", "name", d.Name(), "isDir", d.IsDir())
		if d.IsDir() && strings.HasPrefix(d.Name(), "corex") {
			smiPath := filepath.Join(rootDir, d.Name(), "bin", "ixsmi")
			logger.Info("scanCorexSmiPaths: checking smiPath", "smiPath", smiPath)
			if _, err := os.Stat(smiPath); err == nil {
				paths = append(paths, smiPath)
			}
		}
	}

	// 可选：对路径进行排序，确保版本顺序
	slices.Sort(paths)
	return paths
}

func autoFindSmiPath() string {
	for _, path := range smiPaths {
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}
	return ""
}

// ixsmiEnv 返回运行 ixsmi 所需的环境变量：将 corex 工具包的 bin/lib
// 目录前置到 PATH 和 LD_LIBRARY_PATH。只作用于单个子进程，
// 不修改 dauadm-server 进程自身的全局环境，避免影响其他 GPU loader。
func ixsmiEnv(smiPath string) []string {
	smiDir := strings.TrimSuffix(smiPath, "/bin/ixsmi")
	overrides := map[string]string{
		"PATH":            smiDir + "/bin:" + os.Getenv("PATH"),
		"LD_LIBRARY_PATH": fmt.Sprintf("%s/lib:%s/lib64:%s", smiDir, smiDir, os.Getenv("LD_LIBRARY_PATH")),
	}

	env := make([]string, 0, len(os.Environ())+len(overrides))
	seen := make(map[string]bool, len(overrides))
	for _, kv := range os.Environ() {
		k, _, _ := strings.Cut(kv, "=")
		if v, ok := overrides[k]; ok {
			env = append(env, k+"="+v)
			seen[k] = true
		} else {
			env = append(env, kv)
		}
	}
	for k, v := range overrides {
		if !seen[k] {
			env = append(env, k+"="+v)
		}
	}
	return env
}

type ixGPU struct {
}

func (a *ixGPU) Load() (*gpu.GPUInfoList, error) {
	smiPath := autoFindSmiPath()
	if smiPath == "" {
		logger.Error("ixGPU Load autoFindSmiPath", "smiPath", smiPath, "error", fmt.Errorf("ixsmi not found"))
		return nil, fmt.Errorf("ixsmi not found")
	}

	cmd := exec.Command(smiPath, "-q", "-x")
	cmd.Env = ixsmiEnv(smiPath)
	data, err := cmd.Output()
	if err != nil {
		logger.Error("ixGPU Load get data", "cmd", cmd.String(), "error", err)
		return nil, err
	}
	logger.Info("ixGPU Load get data success")

	result, err := ParseIXSMI(string(data))
	if err != nil {
		logger.Error("ixGPU Load ParseIXSMI", "error", err)
		return nil, err
	}
	logger.Info("ixGPU Load ParseIXSMI success")
	return result, nil
}

func (a *ixGPU) Vendor() string {
	return "Iluvatar"
}

func (a *ixGPU) Available() bool {
	if runtime.GOOS != "linux" {
		logger.Info("ixGPU Available check os", "os", runtime.GOOS, "arch", runtime.GOARCH, "return", false)
		return false
	}
	smiPath := autoFindSmiPath()
	if smiPath == "" {
		return false
	}
	logger.Info("ixGPU Available", "smiPath", smiPath)

	_, err := exec.LookPath(smiPath)
	if err != nil {
		logger.Error("ixGPU lookpath ixsmi", "smiPath", smiPath, "error", err)
		return false
	}
	cmd := exec.Command(smiPath, "-q", "-x")
	cmd.Env = ixsmiEnv(smiPath)
	if err := cmd.Run(); err != nil {
		// 无 Iluvatar GPU 或未装驱动时 ixsmi 会失败（如 exit 255），
		// 这是预期情况，降级为 Info 避免误导为故障
		logger.Info("ixGPU test ixsmi failed, skipping Iluvatar loader", "cmd", cmd.String(), "error", err)
		return false
	}
	logger.Info("ixGPU Available success", "cmd", cmd.String(), "return", true)
	return true
}

func ParseIXSMI(data string) (*gpu.GPUInfoList, error) {
	type MemoryUsage struct {
		Total string `xml:"total"`
		Used  string `xml:"used"`
	}
	type Utilization struct {
		GPUUtil    string `xml:"gpu_util"`
		MemoryUtil string `xml:"memory_util"`
	}
	type Temperature struct {
		GPUTemp string `xml:"gpu_temp"`
	}
	type PCI struct {
		Bus         string `xml:"pci_bus"`
		Device      string `xml:"pci_device"`
		Domain      string `xml:"pci_domain"`
		DeviceID    string `xml:"pci_device_id"`
		BusID       string `xml:"pci_bus_id"`
		SubSystemID string `xml:"pci_sub_system_id"`
	}
	type GPU struct {
		ID      string      `xml:"id,attr"`
		Product string      `xml:"product_name"`
		Serial  string      `xml:"serial"`
		Minor   string      `xml:"minor_number"`
		Memory  MemoryUsage `xml:"memory_usage"`
		Util    Utilization `xml:"utilization"`
		Temp    Temperature `xml:"temperature"`
		PCI     PCI         `xml:"pci"`
	}

	type IXSMILog struct {
		XMLName xml.Name `xml:"ixsmi_log"`
		GPUs    []GPU    `xml:"gpu"`
	}

	var log IXSMILog
	if err := xml.Unmarshal([]byte(data), &log); err != nil {
		return nil, err
	}

	var infos []gpu.GPUInfo
	for i, g := range log.GPUs {
		memTotal := parseMiBToBytes(g.Memory.Total)
		memUsed := parseMiBToBytes(g.Memory.Used)
		temp := parseTempC(g.Temp.GPUTemp)
		gpuUse := parsePercent(g.Util.GPUUtil)
		var pcibus string
		if g.PCI.BusID != "" {
			buf := strings.Split(g.PCI.BusID, ":")
			if len(buf) > 0 {
				buf[0] = g.PCI.Domain
			}
			pcibus = strings.Join(buf, ":")
		}
		infos = append(infos, gpu.GPUInfo{
			Num:                 i,
			DeviceID:            g.ID,
			SerialNumber:        g.Serial,
			VRAMTotalMemory:     memTotal,
			VRAMTotalUsedMemory: memUsed,
			TemperatureEdge:     temp,
			TemperatureJunction: temp,
			TemperatureMemory:   temp,
			GPUUse:              gpuUse,
			CardSeries:          g.Product,
			CardModel:           g.Product,
			CardVendor:          "Iluvatar",
			PCIBus:              pcibus,
		})
	}
	return &gpu.GPUInfoList{GPUInfos: infos}, nil
}

func parseMiBToBytes(s string) string {
	var v float64
	fmt.Sscanf(s, "%f", &v)
	return fmt.Sprintf("%.0f", v*1024*1024)
}

func parseTempC(s string) string {
	var v int
	fmt.Sscanf(s, "%d", &v)
	return fmt.Sprintf("%d", v)
}

func parsePercent(s string) string {
	var v int
	fmt.Sscanf(s, "%d", &v)
	return fmt.Sprintf("%d", v)
}
