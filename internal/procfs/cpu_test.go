package procfs

import "testing"

func TestCPUUsagePercent(t *testing.T) {
	tests := []struct {
		name       string
		prev, curr CPUSample
		want       float64
	}{
		{
			name: "half busy",
			prev: CPUSample{User: 100, Idle: 100},
			curr: CPUSample{User: 150, Idle: 150},
			want: 50,
		},
		{
			name: "counter reset does not underflow",
			prev: CPUSample{User: 1000, Idle: 1000},
			curr: CPUSample{User: 10, Idle: 10},
			want: 0,
		},
		{
			name: "idle delta cannot exceed total delta",
			prev: CPUSample{User: 100, Idle: 100},
			curr: CPUSample{User: 90, Idle: 130},
			want: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := CPUUsagePercent(tt.prev, tt.curr); got != tt.want {
				t.Fatalf("CPUUsagePercent() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseCPUFieldsRejectsMalformedInput(t *testing.T) {
	if _, err := parseCPUFields("cpu0", []string{"1", "2", "bad", "4"}); err == nil {
		t.Fatal("parseCPUFields() accepted a non-numeric field")
	}
	if _, err := parseCPUFields("cpu0", []string{"1", "2", "3"}); err == nil {
		t.Fatal("parseCPUFields() accepted too few fields")
	}
}
