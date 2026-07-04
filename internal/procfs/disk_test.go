package procfs

import (
	"strings"
	"testing"
)

func TestParseDiskSamplesAndRates(t *testing.T) {
	raw := "8 0 sda 10 0 100 1 20 0 200 2 0 30 3 0 0 0 0\n"
	samples, err := parseDiskSamples(strings.NewReader(raw))
	if err != nil {
		t.Fatal(err)
	}
	prev := map[string]DiskSample{"sda": {Name: "sda", ReadSectors: 80, WriteSectors: 150, BusyMillis: 10}}
	got := DiskRates(samples, prev, 2)[0]
	if got.ReadRate != 20*diskSectorBytes/2 || got.WriteRate != 50*diskSectorBytes/2 || got.BusyPercent != 1 {
		t.Fatalf("unexpected disk rate: %+v", got)
	}
}

func TestDiskRatesCounterReset(t *testing.T) {
	cur := []DiskSample{{Name: "sda", ReadSectors: 1, WriteSectors: 1, BusyMillis: 1}}
	prev := map[string]DiskSample{"sda": {Name: "sda", ReadSectors: 2, WriteSectors: 2, BusyMillis: 2}}
	got := DiskRates(cur, prev, 1)[0]
	if got.ReadRate != 0 || got.WriteRate != 0 || got.BusyPercent != 0 {
		t.Fatalf("counter reset produced rates: %+v", got)
	}
}
