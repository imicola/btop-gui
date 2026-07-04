package procfs

import "testing"

func TestParseStatLineHandlesParenthesesInComm(t *testing.T) {
	line := "42 (worker ) name) S 1 2 3 4 5 6 7 8 9 10 110 12 13 14 15 -5 3 18 1900 2000 21"
	p, err := parseStatLine(line)
	if err != nil {
		t.Fatalf("parseStatLine() error = %v", err)
	}
	if p.PID != 42 || p.Name != "worker ) name" || p.State != "S" {
		t.Fatalf("unexpected identity: %+v", p)
	}
	if p.UTime != 110 || p.STime != 12 || p.Nice != -5 || p.Threads != 3 {
		t.Fatalf("unexpected scheduling fields: %+v", p)
	}
	if p.StartTime != 1900 || p.VSize != 2000 || p.RSS != 21*PageSize {
		t.Fatalf("unexpected memory/identity fields: %+v", p)
	}
}

func TestParseStatLineRejectsMalformedInput(t *testing.T) {
	for _, line := range []string{"", "x (name) S", "1 name S", "1 (name) S 1"} {
		if _, err := parseStatLine(line); err == nil {
			t.Errorf("parseStatLine(%q) unexpectedly succeeded", line)
		}
	}
}

func TestReadProcessMatchesRequestedPID(t *testing.T) {
	p, err := ReadProcess(1)
	if err != nil {
		t.Fatalf("ReadProcess(1) error = %v", err)
	}
	if p.PID != 1 || p.StartTime == 0 {
		t.Fatalf("unexpected init process: %+v", p)
	}
}
