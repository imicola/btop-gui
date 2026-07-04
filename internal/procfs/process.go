package procfs

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// ListProcesses 遍历 /proc 下所有数字命名的目录，读取每个进程的 stat。
// 进程可能在中途退出，遇到读取错误直接跳过。
func ListProcesses() ([]ProcessInfo, error) {
	entries, err := os.ReadDir("/proc")
	if err != nil {
		return nil, err
	}
	procs := make([]ProcessInfo, 0, 256)
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		pid, err := strconv.Atoi(e.Name())
		if err != nil {
			continue
		}
		p, err := ReadProcess(pid)
		if err != nil {
			continue
		}
		procs = append(procs, p)
	}
	if len(procs) == 0 {
		return nil, fmt.Errorf("no readable process stat entries found")
	}
	return procs, nil
}

// ReadProcess 读取一个指定 PID 的进程快照。
func ReadProcess(pid int) (ProcessInfo, error) {
	if pid <= 0 {
		return ProcessInfo{}, fmt.Errorf("invalid pid %d", pid)
	}
	b, err := os.ReadFile(filepath.Join("/proc", strconv.Itoa(pid), "stat"))
	if err != nil {
		return ProcessInfo{PID: pid}, err
	}
	p, err := parseStatLine(string(b))
	if err != nil {
		return ProcessInfo{PID: pid}, fmt.Errorf("parse /proc/%d/stat: %w", pid, err)
	}
	if p.PID != pid {
		return ProcessInfo{PID: pid}, fmt.Errorf("/proc/%d/stat contains pid %d", pid, p.PID)
	}
	return p, nil
}

// parseStatLine 解析 /proc/[pid]/stat 单行内容。
//
// 示例：1 (init) S 0 1 1 0 -1 4194560 ...
//
// 关键陷阱：comm 字段被括号包围，且进程名本身可能含空格或括号，
// 必须用第一个 '(' 与最后一个 ')' 截取 comm，再从 ')' 之后解析其余字段。
//
// ')' 之后的字段索引（以 0 起）对应 man proc 中的字段号：
//
//	0=state(3)  1=ppid(4)  11=utime(14)  12=stime(15)
//	16=nice(19)  17=threads(20)  19=starttime(22)
//	20=vsize(23)  21=rss(24, 单位: 页)
func parseStatLine(line string) (ProcessInfo, error) {
	line = strings.TrimSpace(line)
	if line == "" {
		return ProcessInfo{}, fmt.Errorf("empty stat line")
	}

	sp := strings.IndexByte(line, ' ')
	if sp == -1 {
		return ProcessInfo{}, fmt.Errorf("missing pid separator")
	}
	pid, err := strconv.Atoi(line[:sp])
	if err != nil || pid <= 0 {
		return ProcessInfo{}, fmt.Errorf("invalid pid %q", line[:sp])
	}

	lparen := strings.IndexByte(line, '(')
	rparen := strings.LastIndexByte(line, ')')
	if lparen <= sp || rparen <= lparen {
		return ProcessInfo{}, fmt.Errorf("malformed comm field")
	}
	comm := line[lparen+1 : rparen]
	rest := line[rparen+1:]

	fields := strings.Fields(rest)
	if len(fields) < 22 {
		return ProcessInfo{}, fmt.Errorf("only %d fields after comm, need at least 22", len(fields))
	}
	toInt := func(i int, name string) (int, error) {
		v, err := strconv.Atoi(fields[i])
		if err != nil {
			return 0, fmt.Errorf("invalid %s %q", name, fields[i])
		}
		return v, nil
	}
	toUint64 := func(i int, name string) (uint64, error) {
		v, err := strconv.ParseUint(fields[i], 10, 64)
		if err != nil {
			return 0, fmt.Errorf("invalid %s %q", name, fields[i])
		}
		return v, nil
	}

	ppid, err := toInt(1, "ppid")
	if err != nil {
		return ProcessInfo{}, err
	}
	nice, err := toInt(16, "nice")
	if err != nil {
		return ProcessInfo{}, err
	}
	threads, err := toInt(17, "threads")
	if err != nil {
		return ProcessInfo{}, err
	}
	startTime, err := toUint64(19, "starttime")
	if err != nil {
		return ProcessInfo{}, err
	}
	vsize, err := toUint64(20, "vsize")
	if err != nil {
		return ProcessInfo{}, err
	}
	rssPages, err := strconv.ParseInt(fields[21], 10, 64)
	if err != nil {
		return ProcessInfo{}, fmt.Errorf("invalid rss %q", fields[21])
	}
	utime, err := toUint64(11, "utime")
	if err != nil {
		return ProcessInfo{}, err
	}
	stime, err := toUint64(12, "stime")
	if err != nil {
		return ProcessInfo{}, err
	}

	var rss uint64
	if rssPages > 0 {
		rss = uint64(rssPages) * PageSize
	}
	return ProcessInfo{
		PID: pid, Name: comm, State: fields[0], PPID: ppid, Nice: nice,
		Threads: threads, StartTime: startTime, VSize: vsize, RSS: rss,
		UTime: utime, STime: stime,
	}, nil
}
