package procfs

import (
	"os"
	"strings"
	"testing"
)

func TestParseProcessStatusAndIO(t *testing.T) {
	status, err := parseProcessStatus(strings.NewReader("Uid:\t1000 1000 1000 1000\nVmPeak: 42 kB\nVmSwap: 3 kB\nvoluntary_ctxt_switches: 7\nnonvoluntary_ctxt_switches: 2\n"))
	if err != nil || status.uid != 1000 || status.vmPeakKB != 42 || status.vmSwapKB != 3 || status.voluntary != 7 || status.involuntary != 2 {
		t.Fatalf("unexpected status: %+v, %v", status, err)
	}
	ioData, err := parseProcessIO(strings.NewReader("read_bytes: 1024\nwrite_bytes: 2048\n"))
	if err != nil || ioData != [2]uint64{1024, 2048} {
		t.Fatalf("unexpected io: %v, %v", ioData, err)
	}
}

func TestReadProcessDetailIdentity(t *testing.T) {
	p, err := ReadProcess(os.Getpid())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ReadProcessDetail(p.PID, p.StartTime+1); err == nil {
		t.Fatal("stale process identity was accepted")
	}
}
