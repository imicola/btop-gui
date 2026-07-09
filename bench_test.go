package main

import (
	"math"
	"os"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	"btop-gui/internal/procfs"
)

// 集成测试基准：复用生产采样路径 App.GetSnapshot()，循环采样并统计
// 平均/P95 采样耗时、程序自身 CPU% 与 RSS、进程数、错误数。
//
// 通过环境变量驱动（避免污染普通 go test）：
//
//	BTOP_BENCH=1                          启用（否则跳过）
//	BTOP_SCENE=idle|cpu|net|short|long    场景标签，仅用于输出
//	BTOP_DURATION=90                      单次运行秒数
//	BTOP_INTERVAL=1                       采样间隔秒（默认 1，与前端默认刷新率一致）
//
//	go test -run '^TestIntegrationBench$' -timeout 40m -tags webkit2_41 -v
//
// 结果以 "RESULT<TAB>..." 一行 TSV 输出，便于脚本抓取。
func TestIntegrationBench(t *testing.T) {
	if os.Getenv("BTOP_BENCH") != "1" {
		t.Skip("set BTOP_BENCH=1 to run integration benchmark")
	}
	scene := getenvDefault("BTOP_SCENE", "idle")
	duration := getenvInt("BTOP_DURATION", 90)
	interval := getenvInt("BTOP_INTERVAL", 1)

	app := NewApp()
	// 预热：首次 GetSnapshot 的 CPU% 恒为 0（无前序样本），预热让差分进入稳态。
	if _, err := app.GetSnapshot(); err != nil {
		t.Fatalf("warmup GetSnapshot failed: %v", err)
	}

	cpu0, _, _ := readSelfUsage()
	wall0 := time.Now()

	var (
		latencies []float64
		procCount []int
		rssMaxKB  uint64
		errCount  int
	)

	sampleOnce := func() {
		t0 := time.Now()
		snap, err := app.GetSnapshot()
		lat := float64(time.Since(t0).Microseconds()) / 1000.0
		if err != nil {
			errCount++
			return
		}
		latencies = append(latencies, lat)
		procCount = append(procCount, snap.ProcCount)
		if _, m, e := readSelfUsage(); e == nil && m.rssKB > rssMaxKB {
			rssMaxKB = m.rssKB
		}
	}

	// 立即采第一次，之后每个 interval 采一次，直到跑满 duration 秒。
	samples := duration / interval
	if samples < 1 {
		samples = 1
	}
	sampleOnce()
	ticker := time.NewTicker(time.Duration(interval) * time.Second)
	for i := 1; i < samples; i++ {
		<-ticker.C
		sampleOnce()
	}
	ticker.Stop()

	cpu1, _, _ := readSelfUsage()
	wallDelta := time.Since(wall0).Seconds()
	ticksDelta := (cpu1.utime + cpu1.stime) - (cpu0.utime + cpu0.stime)
	cpuPct := 0.0
	if wallDelta > 0 {
		// 单核口径：pcpu = ticks_delta / (wall * ClkTck) * 100，与 top %CPU 一致。
		// 满载一个核 = 100%，多线程进程可 > 100%。
		cpuPct = float64(ticksDelta) / (wallDelta * float64(procfs.ClkTck)) * 100.0
	}

	avg, p95 := stats(latencies)
	rssMB := float64(rssMaxKB) / 1024.0

	// RESULT <scene> <procs> <avg_ms> <p95_ms> <cpu_pct> <rss_mb> <errors> <samples>
	row := []string{
		"RESULT", scene,
		strconv.Itoa(meanInt(procCount)),
		fmtFloat(avg, 3),
		fmtFloat(p95, 3),
		fmtFloat(cpuPct, 2),
		fmtFloat(rssMB, 1),
		strconv.Itoa(errCount),
		strconv.Itoa(len(latencies)),
	}
	out := strings.Join(row, "\t")
	t.Log(out)
	// go test 缓冲 t.Log，println 直达 stderr 做双保险，便于脚本抓取。
	println(out)
}

type selfCPU struct{ utime, stime uint64 }
type selfMem struct{ rssKB uint64 }

// readSelfUsage 解析 /proc/self/stat 的 utime/stime 与 /proc/self/status 的 VmRSS。
func readSelfUsage() (selfCPU, selfMem, error) {
	b, err := os.ReadFile("/proc/self/stat")
	if err != nil {
		return selfCPU{}, selfMem{}, err
	}
	// comm 字段含空格且在括号内，用最后一个 ')' 定位后再 Fields 切分。
	rparen := strings.LastIndexByte(string(b), ')')
	rest := strings.Fields(string(b)[rparen+1:])
	// /proc/[pid]/stat 字段号（man proc）：1 pid 2 comm 3 state ... 14 utime 15 stime。
	// 切掉 "pid (comm)" 后，rest[0]=state，故 utime/stime 落在 rest[11]/rest[12]。
	const utimeIdx, stimeIdx = 11, 12
	var c selfCPU
	if len(rest) > stimeIdx {
		c.utime, _ = strconv.ParseUint(rest[utimeIdx], 10, 64)
		c.stime, _ = strconv.ParseUint(rest[stimeIdx], 10, 64)
	}

	var m selfMem
	if sb, e := os.ReadFile("/proc/self/status"); e == nil {
		for _, line := range strings.Split(string(sb), "\n") {
			if strings.HasPrefix(line, "VmRSS:") {
				if f := strings.Fields(line); len(f) >= 2 {
					m.rssKB, _ = strconv.ParseUint(f[1], 10, 64)
				}
				break
			}
		}
	}
	return c, m, nil
}

func stats(xs []float64) (avg, p95 float64) {
	if len(xs) == 0 {
		return 0, 0
	}
	cp := append([]float64(nil), xs...)
	sort.Float64s(cp)
	var sum float64
	for _, v := range cp {
		sum += v
	}
	avg = sum / float64(len(cp))
	idx := int(math.Ceil(0.95*float64(len(cp)))) - 1
	if idx < 0 {
		idx = 0
	}
	if idx >= len(cp) {
		idx = len(cp) - 1
	}
	p95 = cp[idx]
	return
}

func meanInt(xs []int) int {
	if len(xs) == 0 {
		return 0
	}
	var s int
	for _, v := range xs {
		s += v
	}
	return s / len(xs)
}

func fmtFloat(v float64, prec int) string { return strconv.FormatFloat(v, 'f', prec, 64) }

func getenvDefault(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}

func getenvInt(k string, d int) int {
	if v := os.Getenv(k); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return d
}
