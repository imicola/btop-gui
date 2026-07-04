//go:build linux && cgo

package processctl

import (
	"testing"

	"golang.org/x/sys/unix"
)

func TestForkSleep(t *testing.T) {
	pid, err := ForkSleep(1)
	if err != nil {
		t.Fatal(err)
	}
	if pid <= 1 {
		t.Fatalf("invalid child pid %d", pid)
	}
	if err := unix.Kill(pid, 0); err != nil {
		t.Fatalf("fork child is not alive: %v", err)
	}
}
