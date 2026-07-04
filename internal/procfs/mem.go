package procfs

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// ReadMemoryInfo 解析 /proc/meminfo，返回结构化内存数据
//
// /proc/meminfo 行格式："MemTotal:       16384000 kB"
func ReadMemoryInfo() (MemoryInfo, error) {
	f, err := os.Open("/proc/meminfo")
	if err != nil {
		return MemoryInfo{}, err
	}
	defer f.Close()

	vals := make(map[string]uint64, 32)
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := sc.Text()
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		rest := strings.Fields(parts[1])
		if len(rest) == 0 {
			continue
		}
		v, err := strconv.ParseUint(rest[0], 10, 64)
		if err != nil {
			return MemoryInfo{}, err
		}
		vals[key] = v
	}
	if err := sc.Err(); err != nil {
		return MemoryInfo{}, err
	}

	m := MemoryInfo{
		Total:     vals["MemTotal"],
		Free:      vals["MemFree"],
		Available: vals["MemAvailable"],
		Cached:    vals["Cached"],
		Buffers:   vals["Buffers"],
		SwapTotal: vals["SwapTotal"],
		SwapFree:  vals["SwapFree"],
	}
	if m.Total == 0 {
		return MemoryInfo{}, fmt.Errorf("/proc/meminfo has no valid MemTotal")
	}
	// Used = Total - Available（MemAvailable 综合考虑了 cache/buffer 可回收部分，
	// 比 MemFree 更贴近"真实使用"）
	if m.Available > m.Total {
		if m.Free < m.Total {
			m.Used = m.Total - m.Free
		}
	} else {
		m.Used = m.Total - m.Available
	}
	if m.Total > 0 {
		m.Usage = float64(m.Used) / float64(m.Total) * 100.0
	}
	if m.SwapTotal > 0 {
		if m.SwapFree > m.SwapTotal {
			m.SwapFree = m.SwapTotal
		}
		m.SwapUsage = float64(m.SwapTotal-m.SwapFree) / float64(m.SwapTotal) * 100.0
	}
	return m, nil
}
