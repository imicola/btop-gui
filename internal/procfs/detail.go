package procfs

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"
)

// ReadProcessDetail 按需读取一个进程的 status/cmdline/io/fd 信息。
// expectedStartTime 防止读取过程中 PID 已被复用。
func ReadProcessDetail(pid int, expectedStartTime uint64) (ProcessDetail, error) {
	p, err := ReadProcess(pid)
	if err != nil {
		return ProcessDetail{}, err
	}
	if expectedStartTime == 0 || p.StartTime != expectedStartTime {
		return ProcessDetail{}, fmt.Errorf("pid %d identity changed", pid)
	}
	detail := ProcessDetail{PID: pid, StartTime: p.StartTime, Command: p.Name}
	base := filepath.Join("/proc", strconv.Itoa(pid))

	if f, err := os.Open(filepath.Join(base, "status")); err == nil {
		status, parseErr := parseProcessStatus(f)
		_ = f.Close()
		if parseErr == nil {
			detail.UID = status.uid
			detail.VmPeak = status.vmPeakKB * 1024
			detail.VmSwap = status.vmSwapKB * 1024
			detail.VoluntarySwitches = status.voluntary
			detail.InvoluntarySwitches = status.involuntary
			if u, lookupErr := user.LookupId(strconv.FormatUint(status.uid, 10)); lookupErr == nil {
				detail.User = u.Username
			} else {
				detail.User = strconv.FormatUint(status.uid, 10)
			}
		} else {
			detail.Unavailable = append(detail.Unavailable, "status")
		}
	} else {
		detail.Unavailable = append(detail.Unavailable, "status")
	}

	if b, err := os.ReadFile(filepath.Join(base, "cmdline")); err == nil {
		if command := strings.TrimSpace(string(bytes.ReplaceAll(bytes.TrimRight(b, "\x00"), []byte{0}, []byte{' '}))); command != "" {
			detail.Command = command
		}
	} else {
		detail.Unavailable = append(detail.Unavailable, "cmdline")
	}
	if target, err := os.Readlink(filepath.Join(base, "exe")); err == nil {
		detail.Executable = target
	} else {
		detail.Unavailable = append(detail.Unavailable, "exe")
	}
	if target, err := os.Readlink(filepath.Join(base, "cwd")); err == nil {
		detail.CWD = target
	} else {
		detail.Unavailable = append(detail.Unavailable, "cwd")
	}
	if f, err := os.Open(filepath.Join(base, "io")); err == nil {
		ioData, parseErr := parseProcessIO(f)
		_ = f.Close()
		if parseErr == nil {
			detail.ReadBytes, detail.WriteBytes = ioData[0], ioData[1]
		} else {
			detail.Unavailable = append(detail.Unavailable, "io")
		}
	} else {
		detail.Unavailable = append(detail.Unavailable, "io")
	}
	if entries, err := os.ReadDir(filepath.Join(base, "fd")); err == nil {
		detail.FDCount = len(entries)
	} else {
		detail.FDCount = -1
		detail.Unavailable = append(detail.Unavailable, "fd")
	}
	if uptime, err := ReadUptime(); err == nil {
		detail.Elapsed = uptime - float64(p.StartTime)/ClkTck
		if detail.Elapsed < 0 {
			detail.Elapsed = 0
		}
	}
	return detail, nil
}

type processStatus struct {
	uid         uint64
	vmPeakKB    uint64
	vmSwapKB    uint64
	voluntary   uint64
	involuntary uint64
}

func parseProcessStatus(r io.Reader) (processStatus, error) {
	var result processStatus
	sc := bufio.NewScanner(r)
	for sc.Scan() {
		parts := strings.SplitN(sc.Text(), ":", 2)
		if len(parts) != 2 {
			continue
		}
		key, fields := parts[0], strings.Fields(parts[1])
		if len(fields) == 0 {
			continue
		}
		var dst *uint64
		switch key {
		case "Uid":
			dst = &result.uid
		case "VmPeak":
			dst = &result.vmPeakKB
		case "VmSwap":
			dst = &result.vmSwapKB
		case "voluntary_ctxt_switches":
			dst = &result.voluntary
		case "nonvoluntary_ctxt_switches":
			dst = &result.involuntary
		}
		if dst != nil {
			value, err := strconv.ParseUint(fields[0], 10, 64)
			if err != nil {
				return result, fmt.Errorf("invalid %s value: %w", key, err)
			}
			*dst = value
		}
	}
	return result, sc.Err()
}

func parseProcessIO(r io.Reader) ([2]uint64, error) {
	var result [2]uint64
	sc := bufio.NewScanner(r)
	for sc.Scan() {
		parts := strings.Fields(sc.Text())
		if len(parts) != 2 {
			continue
		}
		var index int
		switch strings.TrimSuffix(parts[0], ":") {
		case "read_bytes":
			index = 0
		case "write_bytes":
			index = 1
		default:
			continue
		}
		value, err := strconv.ParseUint(parts[1], 10, 64)
		if err != nil {
			return result, err
		}
		result[index] = value
	}
	return result, sc.Err()
}
