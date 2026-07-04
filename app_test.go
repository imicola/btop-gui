package main

import (
	"os"
	"strings"
	"testing"
	"time"

	"btop-gui/internal/procfs"
)

func TestVerifyProcessIdentity(t *testing.T) {
	p, err := procfs.ReadProcess(os.Getpid())
	if err != nil {
		t.Fatalf("ReadProcess(self) error = %v", err)
	}
	if err := verifyProcessIdentity(p.PID, p.StartTime); err != nil {
		t.Fatalf("verifyProcessIdentity(valid) error = %v", err)
	}
	if err := verifyProcessIdentity(p.PID, p.StartTime+1); err == nil || !strings.Contains(err.Error(), "复用") {
		t.Fatalf("verifyProcessIdentity(stale) error = %v", err)
	}
}

func TestGetSnapshotEndToEnd(t *testing.T) {
	a := NewApp()
	first, err := a.GetSnapshot()
	if err != nil {
		t.Fatalf("first GetSnapshot() error = %v", err)
	}
	time.Sleep(20 * time.Millisecond)
	second, err := a.GetSnapshot()
	if err != nil {
		t.Fatalf("second GetSnapshot() error = %v", err)
	}
	if second.Memory.Total == 0 || second.CPU.CoreCount == 0 || second.ProcCount == 0 {
		t.Fatalf("incomplete snapshot: %+v", second)
	}
	if len(second.Procs) != second.ProcCount || first.Timestamp > second.Timestamp {
		t.Fatalf("inconsistent snapshot metadata: first=%d second=%d procs=%d/%d",
			first.Timestamp, second.Timestamp, len(second.Procs), second.ProcCount)
	}
	if second.System.Hostname == "" || !second.Network.Available || len(second.Network.Interfaces) == 0 {
		t.Fatalf("missing system/network data: system=%+v network=%+v", second.System, second.Network)
	}
}

func TestGetProcessDetailAndForkValidation(t *testing.T) {
	a := NewApp()
	p, err := procfs.ReadProcess(os.Getpid())
	if err != nil {
		t.Fatal(err)
	}
	if detail, err := a.GetProcessDetail(p.PID, p.StartTime); err != nil || detail.PID != p.PID {
		t.Fatalf("GetProcessDetail() = %+v, %v", detail, err)
	}
	if _, err := a.ForkDemo(31); err == nil {
		t.Fatal("ForkDemo() accepted an excessive duration")
	}
}

func TestKillProcessRejectsDangerousTargetsAndSignals(t *testing.T) {
	a := NewApp()
	if err := a.KillProcess(1, 1, 15); err == nil {
		t.Fatal("KillProcess() accepted pid 1")
	}
	if err := a.KillProcess(os.Getpid(), 1, 15); err == nil {
		t.Fatal("KillProcess() accepted its own pid")
	}
	if err := a.KillProcess(999999, 1, 0); err == nil {
		t.Fatal("KillProcess() accepted signal 0")
	}
}
