package procfs

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"sort"
	"strconv"
	"strings"
)

// NetworkSample 保存 /proc/net/dev 的累计字节数。
type NetworkSample struct {
	Name    string
	RXBytes uint64
	TXBytes uint64
}

// ReadNetworkSamples 手写解析 /proc/net/dev。
// 冒号左侧是接口名；接收 bytes 是字段 1，发送 bytes 是字段 9。
func ReadNetworkSamples() ([]NetworkSample, error) {
	f, err := os.Open("/proc/net/dev")
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return parseNetworkSamples(f)
}

func parseNetworkSamples(r io.Reader) ([]NetworkSample, error) {
	var samples []NetworkSample
	sc := bufio.NewScanner(r)
	for sc.Scan() {
		line := sc.Text()
		colon := strings.IndexByte(line, ':')
		if colon < 0 {
			continue
		}
		name := strings.TrimSpace(line[:colon])
		fields := strings.Fields(line[colon+1:])
		if name == "" || len(fields) < 16 {
			return nil, fmt.Errorf("malformed /proc/net/dev row %q", line)
		}
		rx, err := strconv.ParseUint(fields[0], 10, 64)
		if err != nil {
			return nil, fmt.Errorf("interface %s rx bytes: %w", name, err)
		}
		tx, err := strconv.ParseUint(fields[8], 10, 64)
		if err != nil {
			return nil, fmt.Errorf("interface %s tx bytes: %w", name, err)
		}
		samples = append(samples, NetworkSample{Name: name, RXBytes: rx, TXBytes: tx})
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	if len(samples) == 0 {
		return nil, fmt.Errorf("/proc/net/dev contains no interfaces")
	}
	sort.Slice(samples, func(i, j int) bool { return samples[i].Name < samples[j].Name })
	return samples, nil
}

// NetworkRates 将累计计数转换为 bytes/s；首次出现或计数回退时速率为 0。
func NetworkRates(current []NetworkSample, previous map[string]NetworkSample, seconds float64) []NetworkInterfaceInfo {
	result := make([]NetworkInterfaceInfo, 0, len(current))
	for _, cur := range current {
		item := NetworkInterfaceInfo{Name: cur.Name, RXBytes: cur.RXBytes, TXBytes: cur.TXBytes}
		if seconds > 0 {
			if prev, ok := previous[cur.Name]; ok {
				if cur.RXBytes >= prev.RXBytes {
					item.RXRate = float64(cur.RXBytes-prev.RXBytes) / seconds
				}
				if cur.TXBytes >= prev.TXBytes {
					item.TXRate = float64(cur.TXBytes-prev.TXBytes) / seconds
				}
			}
		}
		result = append(result, item)
	}
	return result
}

// PrimaryNetworkInterface 优先选择流量最大的非回环接口。
func PrimaryNetworkInterface(items []NetworkInterfaceInfo) string {
	primary := ""
	var best float64
	for _, item := range items {
		if item.Name == "lo" {
			continue
		}
		score := item.RXRate + item.TXRate + float64(item.RXBytes+item.TXBytes)/1e12
		if primary == "" || score > best {
			primary, best = item.Name, score
		}
	}
	if primary == "" && len(items) > 0 {
		primary = items[0].Name
	}
	return primary
}
