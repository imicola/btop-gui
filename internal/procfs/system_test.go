package procfs

import (
	"strings"
	"testing"
)

func TestParseCPUInfo(t *testing.T) {
	model, freq, err := parseCPUInfo(strings.NewReader("model name : Test CPU\ncpu MHz : 2400.500\n"))
	if err != nil || model != "Test CPU" || freq != 2400.5 {
		t.Fatalf("parseCPUInfo() = %q, %v, %v", model, freq, err)
	}
}

func TestParseLoadAvg(t *testing.T) {
	got, err := parseLoadAvg("0.52 0.58 0.59 2/1034 12345\n")
	if err != nil {
		t.Fatalf("parseLoadAvg() error = %v", err)
	}
	if got != [3]float64{0.52, 0.58, 0.59} {
		t.Fatalf("parseLoadAvg() = %v", got)
	}
	for _, raw := range []string{"", "1 2", "bad 2 3"} {
		if _, err := parseLoadAvg(raw); err == nil {
			t.Errorf("parseLoadAvg(%q) unexpectedly succeeded", raw)
		}
	}
}

func TestParseUptime(t *testing.T) {
	if got, err := parseUptime("123.45 99.0\n"); err != nil || got != 123.45 {
		t.Fatalf("parseUptime() = %v, %v", got, err)
	}
	for _, raw := range []string{"", "bad 1", "-1 0"} {
		if _, err := parseUptime(raw); err == nil {
			t.Errorf("parseUptime(%q) unexpectedly succeeded", raw)
		}
	}
}
