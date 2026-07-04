package procfs

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"syscall"
)

// ClkTck 是 sysconf(_SC_CLK_TCK) 即 USER_HZ。
// Linux x86/x86_64/ARM 固定为 100，/proc 中所有 CPU 时间字段都以这个为单位。
const ClkTck = 100

// PageSize 是内存页大小（bytes），/proc/[pid]/stat 的 rss 字段单位是"页"。
var PageSize = uint64(syscall.Getpagesize())

// ReadSystemInfo 读取 hostname、kernel release 和 /proc/cpuinfo。
func ReadSystemInfo() SystemInfo {
	info := SystemInfo{}
	info.Hostname, _ = os.Hostname()
	if b, err := os.ReadFile("/proc/sys/kernel/osrelease"); err == nil {
		info.Kernel = strings.TrimSpace(string(b))
	}
	if f, err := os.Open("/proc/cpuinfo"); err == nil {
		defer f.Close()
		model, freq, _ := parseCPUInfo(f)
		info.CPUModel, info.CPUFreqMHz = model, freq
	}
	return info
}

func parseCPUInfo(r io.Reader) (string, float64, error) {
	var model string
	var freq float64
	sc := bufio.NewScanner(r)
	for sc.Scan() {
		parts := strings.SplitN(sc.Text(), ":", 2)
		if len(parts) != 2 {
			continue
		}
		key, value := strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])
		switch key {
		case "model name", "Processor", "Hardware":
			if model == "" {
				model = value
			}
		case "cpu MHz":
			if freq == 0 {
				parsed, err := strconv.ParseFloat(value, 64)
				if err != nil {
					return "", 0, fmt.Errorf("invalid cpu MHz %q: %w", value, err)
				}
				freq = parsed
			}
		}
		if model != "" && freq != 0 {
			break
		}
	}
	if err := sc.Err(); err != nil {
		return "", 0, err
	}
	return model, freq, nil
}

// ReadLoadAvg 读取 /proc/loadavg，返回 1/5/15 分钟平均负载
//
// /proc/loadavg 格式："0.52 0.58 0.59 2/1034 12345"
func ReadLoadAvg() ([3]float64, error) {
	b, err := os.ReadFile("/proc/loadavg")
	if err != nil {
		return [3]float64{}, err
	}
	return parseLoadAvg(string(b))
}

func parseLoadAvg(raw string) ([3]float64, error) {
	var la [3]float64
	fields := strings.Fields(raw)
	if len(fields) < 3 {
		return la, fmt.Errorf("loadavg has %d fields, need at least 3", len(fields))
	}
	for i := 0; i < 3; i++ {
		v, err := strconv.ParseFloat(fields[i], 64)
		if err != nil {
			return la, fmt.Errorf("invalid load average %q: %w", fields[i], err)
		}
		la[i] = v
	}
	return la, nil
}

// ReadUptime 读取 /proc/uptime，返回系统启动后经过的秒数
//
// /proc/uptime 格式："123456.78 99877.66"（第二列是各 CPU idle 之和）
func ReadUptime() (float64, error) {
	b, err := os.ReadFile("/proc/uptime")
	if err != nil {
		return 0, err
	}
	return parseUptime(string(b))
}

func parseUptime(raw string) (float64, error) {
	fields := strings.Fields(raw)
	if len(fields) == 0 {
		return 0, fmt.Errorf("uptime is empty")
	}
	v, err := strconv.ParseFloat(fields[0], 64)
	if err != nil || v < 0 {
		return 0, fmt.Errorf("invalid uptime %q", fields[0])
	}
	return v, nil
}
