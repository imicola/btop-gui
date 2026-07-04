package procfs

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// CPUSample 表示某一时刻 /proc/stat 中一行 cpu 的累计时间片（单位 USER_HZ）
type CPUSample struct {
	Name                                                  string
	User, Nice, System, Idle, IOWait, IRQ, SoftIRQ, Steal uint64
}

// Total 总累计 tick
func (s CPUSample) Total() uint64 {
	return s.User + s.Nice + s.System + s.Idle + s.IOWait + s.IRQ + s.SoftIRQ + s.Steal
}

// IdleTotal 空闲 tick（idle + iowait）
func (s CPUSample) IdleTotal() uint64 {
	return s.Idle + s.IOWait
}

// ReadCPUSamples 读取 /proc/stat，返回 (总体, 每核)
//
// /proc/stat 第一行格式：
//
//	cpu  user nice system idle iowait irq softirq steal guest guest_nice
//
// "cpu" 为所有核心汇总，"cpu0".."cpuN" 为单核
func ReadCPUSamples() (CPUSample, []CPUSample, error) {
	f, err := os.Open("/proc/stat")
	if err != nil {
		return CPUSample{}, nil, err
	}
	defer f.Close()

	var all CPUSample
	foundAll := false
	var perCore []CPUSample
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 4096), 65536)
	for sc.Scan() {
		line := sc.Text()
		if !strings.HasPrefix(line, "cpu") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		sample, err := parseCPUFields(fields[0], fields[1:])
		if err != nil {
			return CPUSample{}, nil, err
		}
		if fields[0] == "cpu" {
			all = sample
			foundAll = true
		} else {
			perCore = append(perCore, sample)
		}
	}
	if err := sc.Err(); err != nil {
		return CPUSample{}, nil, err
	}
	if !foundAll {
		return CPUSample{}, nil, fmt.Errorf("/proc/stat does not contain aggregate cpu line")
	}
	return all, perCore, nil
}

// parseCPUFields 解析 stat 行中数字部分（跳过 "cpu"/"cpuN" 前缀）
func parseCPUFields(name string, fields []string) (CPUSample, error) {
	if len(fields) < 4 {
		return CPUSample{}, fmt.Errorf("%s has only %d numeric fields", name, len(fields))
	}
	s := CPUSample{Name: name}
	get := func(i int) (uint64, error) {
		if i >= len(fields) {
			return 0, nil
		}
		v, err := strconv.ParseUint(fields[i], 10, 64)
		if err != nil {
			return 0, fmt.Errorf("%s field %d: %w", name, i+1, err)
		}
		return v, nil
	}
	values := []*uint64{&s.User, &s.Nice, &s.System, &s.Idle, &s.IOWait, &s.IRQ, &s.SoftIRQ, &s.Steal}
	for i, dst := range values {
		v, err := get(i)
		if err != nil {
			return CPUSample{}, err
		}
		*dst = v
	}
	return s, nil
}

// CPUUsagePercent 计算两次采样之间的 CPU 使用率（0-100）
//
// 公式：usage% = (total_delta - idle_delta) / total_delta * 100
func CPUUsagePercent(prev, curr CPUSample) float64 {
	prevTotal, currTotal := prev.Total(), curr.Total()
	prevIdle, currIdle := prev.IdleTotal(), curr.IdleTotal()
	// CPU 热插拔、休眠恢复或计数器重置时累计值可能倒退。先判断再做
	// uint64 减法，避免下溢产生看似合理但错误的百分比。
	if currTotal <= prevTotal || currIdle < prevIdle {
		return 0
	}
	totalDelta := currTotal - prevTotal
	idleDelta := currIdle - prevIdle
	if idleDelta >= totalDelta {
		return 0
	}
	usage := float64(totalDelta-idleDelta) / float64(totalDelta) * 100.0
	if usage < 0 {
		return 0
	}
	if usage > 100 {
		return 100
	}
	return usage
}
