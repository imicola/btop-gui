//go:build !linux || !cgo

package processctl

import "fmt"

func ForkSleep(seconds int) (int, error) {
	return 0, fmt.Errorf("原生 fork 演示仅支持启用 cgo 的 Linux")
}
