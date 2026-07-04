package procfs

import (
	"strings"
	"testing"
)

func TestParseNetworkSamplesAndRates(t *testing.T) {
	raw := "Inter-| Receive | Transmit\n face |bytes |bytes\n  eth0: 1000 1 0 0 0 0 0 0 2000 1 0 0 0 0 0 0\n    lo: 10 1 0 0 0 0 0 0 10 1 0 0 0 0 0 0\n"
	samples, err := parseNetworkSamples(strings.NewReader(raw))
	if err != nil {
		t.Fatal(err)
	}
	prev := map[string]NetworkSample{"eth0": {Name: "eth0", RXBytes: 800, TXBytes: 1200}}
	rates := NetworkRates(samples, prev, 2)
	if rates[0].Name != "eth0" || rates[0].RXRate != 100 || rates[0].TXRate != 400 {
		t.Fatalf("unexpected rates: %+v", rates)
	}
	if PrimaryNetworkInterface(rates) != "eth0" {
		t.Fatalf("unexpected primary interface: %q", PrimaryNetworkInterface(rates))
	}
}

func TestNetworkRatesCounterReset(t *testing.T) {
	cur := []NetworkSample{{Name: "eth0", RXBytes: 1, TXBytes: 1}}
	prev := map[string]NetworkSample{"eth0": {Name: "eth0", RXBytes: 2, TXBytes: 2}}
	got := NetworkRates(cur, prev, 1)[0]
	if got.RXRate != 0 || got.TXRate != 0 {
		t.Fatalf("counter reset produced rates: %+v", got)
	}
}
