package procfs

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"syscall"
)

const diskSectorBytes = 512

// DiskSample 对应 /proc/diskstats 的累计扇区数和 I/O 忙碌毫秒数。
type DiskSample struct {
	Name         string
	ReadSectors  uint64
	WriteSectors uint64
	BusyMillis   uint64
}

func ReadDiskSamples() ([]DiskSample, error) {
	f, err := os.Open("/proc/diskstats")
	if err != nil {
		return nil, err
	}
	defer f.Close()
	all, err := parseDiskSamples(f)
	if err != nil {
		return nil, err
	}
	whole := all[:0]
	for _, sample := range all {
		if isWholeBlockDevice(sample.Name) {
			whole = append(whole, sample)
		}
	}
	if len(whole) == 0 {
		return nil, fmt.Errorf("/proc/diskstats contains no whole block devices")
	}
	return whole, nil
}

func parseDiskSamples(r io.Reader) ([]DiskSample, error) {
	var samples []DiskSample
	sc := bufio.NewScanner(r)
	for sc.Scan() {
		fields := strings.Fields(sc.Text())
		if len(fields) < 14 {
			continue
		}
		readSectors, err := strconv.ParseUint(fields[5], 10, 64)
		if err != nil {
			return nil, fmt.Errorf("device %s read sectors: %w", fields[2], err)
		}
		writeSectors, err := strconv.ParseUint(fields[9], 10, 64)
		if err != nil {
			return nil, fmt.Errorf("device %s write sectors: %w", fields[2], err)
		}
		busyMillis, err := strconv.ParseUint(fields[12], 10, 64)
		if err != nil {
			return nil, fmt.Errorf("device %s busy millis: %w", fields[2], err)
		}
		samples = append(samples, DiskSample{
			Name: fields[2], ReadSectors: readSectors,
			WriteSectors: writeSectors, BusyMillis: busyMillis,
		})
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	if len(samples) == 0 {
		return nil, fmt.Errorf("/proc/diskstats contains no valid rows")
	}
	sort.Slice(samples, func(i, j int) bool { return samples[i].Name < samples[j].Name })
	return samples, nil
}

func isWholeBlockDevice(name string) bool {
	if strings.HasPrefix(name, "loop") || strings.HasPrefix(name, "ram") || strings.HasPrefix(name, "fd") {
		return false
	}
	_, err := os.Stat(filepath.Join("/sys/class/block", name, "partition"))
	return os.IsNotExist(err)
}

func DiskRates(current []DiskSample, previous map[string]DiskSample, seconds float64) []DiskDeviceInfo {
	result := make([]DiskDeviceInfo, 0, len(current))
	for _, cur := range current {
		item := DiskDeviceInfo{
			Name: cur.Name, ReadBytes: cur.ReadSectors * diskSectorBytes,
			WriteBytes: cur.WriteSectors * diskSectorBytes,
		}
		if seconds > 0 {
			if prev, ok := previous[cur.Name]; ok {
				if cur.ReadSectors >= prev.ReadSectors {
					item.ReadRate = float64((cur.ReadSectors-prev.ReadSectors)*diskSectorBytes) / seconds
				}
				if cur.WriteSectors >= prev.WriteSectors {
					item.WriteRate = float64((cur.WriteSectors-prev.WriteSectors)*diskSectorBytes) / seconds
				}
				if cur.BusyMillis >= prev.BusyMillis {
					item.BusyPercent = float64(cur.BusyMillis-prev.BusyMillis) / (seconds * 10)
					if item.BusyPercent > 100 {
						item.BusyPercent = 100
					}
				}
			}
		}
		result = append(result, item)
	}
	return result
}

// ReadRootFilesystem 使用 statfs(2) 获取根文件系统容量。
func ReadRootFilesystem() (FilesystemInfo, error) {
	var st syscall.Statfs_t
	if err := syscall.Statfs("/", &st); err != nil {
		return FilesystemInfo{}, err
	}
	total := st.Blocks * uint64(st.Bsize)
	available := st.Bavail * uint64(st.Bsize)
	used := (st.Blocks - st.Bfree) * uint64(st.Bsize)
	info := FilesystemInfo{MountPoint: "/", Total: total, Used: used, Available: available}
	if total > 0 {
		info.Usage = float64(used) / float64(total) * 100
	}
	return info, nil
}
