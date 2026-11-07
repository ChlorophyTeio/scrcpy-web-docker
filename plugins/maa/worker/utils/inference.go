package utils

import (
	"context"
	"encoding/json"
	"fmt"
	"maa-server/config"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

type InferenceHardware struct {
	Model       string `json:"cpu_model"`
	Allowed     []int  `json:"allowed_cpus"`
	Performance []int  `json:"p_cores"`
}

func ParseCPUList(value string) ([]int, error) {
	out := []int{}
	seen := map[int]bool{}
	for _, part := range strings.Split(strings.TrimSpace(value), ",") {
		r := strings.Split(part, "-")
		if len(r) > 2 {
			return nil, fmt.Errorf("invalid CPU list")
		}
		a, e := strconv.Atoi(r[0])
		if e != nil || a < 0 || a > 4095 {
			return nil, fmt.Errorf("invalid CPU")
		}
		b := a
		if len(r) == 2 {
			b, e = strconv.Atoi(r[1])
			if e != nil || b < a || b > 4095 {
				return nil, fmt.Errorf("invalid CPU range")
			}
		}
		for n := a; n <= b; n++ {
			if !seen[n] {
				out = append(out, n)
				seen[n] = true
			}
		}
	}
	sort.Ints(out)
	return out, nil
}
func CPUList(cpus []int) string {
	values := []string{}
	for _, n := range cpus {
		values = append(values, strconv.Itoa(n))
	}
	return strings.Join(values, ",")
}
func InferenceCPUInfo() InferenceHardware {
	h := InferenceHardware{Allowed: []int{}, Performance: []int{}}
	b, _ := os.ReadFile("/proc/cpuinfo")
	for _, line := range strings.Split(string(b), "\n") {
		if strings.HasPrefix(line, "model name") {
			_, v, _ := strings.Cut(line, ":")
			h.Model = strings.TrimSpace(v)
			break
		}
	}
	b, _ = os.ReadFile("/proc/self/status")
	for _, line := range strings.Split(string(b), "\n") {
		if strings.HasPrefix(line, "Cpus_allowed_list:") {
			h.Allowed, _ = ParseCPUList(strings.TrimPrefix(line, "Cpus_allowed_list:"))
		}
	}
	max := 0
	freq := map[int]int{}
	for _, cpu := range h.Allowed {
		b, e := os.ReadFile(fmt.Sprintf("/sys/devices/system/cpu/cpu%d/cpufreq/cpuinfo_max_freq", cpu))
		if e != nil {
			continue
		}
		n, _ := strconv.Atoi(strings.TrimSpace(string(b)))
		freq[cpu] = n
		if n > max {
			max = n
		}
	}
	// Only distinguish a performance tier when every permitted CPU has a frequency.
	if len(freq) == len(h.Allowed) && max > 0 {
		for _, cpu := range h.Allowed {
			if freq[cpu] == max {
				h.Performance = append(h.Performance, cpu)
			}
		}
		if len(h.Performance) == len(h.Allowed) {
			h.Performance = []int{}
		}
	}
	return h
}
func InferencePath() string {
	return filepath.Join(config.D.HomeDir, "config", "inference-performance.json")
}
func InferenceMode() string {
	b, _ := os.ReadFile(InferencePath())
	var v struct {
		Mode string `json:"mode"`
	}
	_ = json.Unmarshal(b, &v)
	if v.Mode == "performance" {
		return v.Mode
	}
	return "balanced"
}
func MaaCommand(ctx context.Context, args ...string) *exec.Cmd {
	if len(args) > 0 && args[0] == "run" && InferenceMode() == "performance" {
		h := InferenceCPUInfo()
		if len(h.Performance) > 0 {
			if path, e := exec.LookPath("taskset"); e == nil {
				return exec.CommandContext(ctx, path, append([]string{"-c", CPUList(h.Performance), "maa"}, args...)...)
			}
		}
	}
	return exec.CommandContext(ctx, "maa", args...)
}
