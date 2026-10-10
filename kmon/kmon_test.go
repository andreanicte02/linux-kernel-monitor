package kmon

import (
	"strings"
	"testing"
)

func testReader() *Reader { return &Reader{Dir: "testdata/proc"} }

func TestSnapshot(t *testing.T) {
	s, err := testReader().Snapshot()
	if err != nil {
		t.Fatal(err)
	}
	if len(s.Processes) != 5 {
		t.Fatalf("got %d processes, want 5", len(s.Processes))
	}
	if got := s.Processes[3].Name; got != `my "quoted" app` {
		t.Errorf("escaped name decoded as %q", got)
	}
	if s.Memory.UsedKB() != 3_000_000 {
		t.Errorf("UsedKB = %d, want 3000000", s.Memory.UsedKB())
	}
	if s.States.Zombie != 1 || s.States.Total != 5 {
		t.Errorf("unexpected states: %+v", s.States)
	}
	if s.Host.Kernel != "6.8.0-1021-azure" || s.Host.UptimeSeconds != 3600.25 {
		t.Errorf("unexpected host: %+v", s.Host)
	}
}

func TestStatesMatchProcessList(t *testing.T) {
	s, err := testReader().Snapshot()
	if err != nil {
		t.Fatal(err)
	}
	counts := map[string]int{}
	for _, p := range s.Processes {
		counts[p.State]++
	}
	if counts["R"] != s.States.Running || counts["S"] != s.States.Sleeping || counts["Z"] != s.States.Zombie {
		t.Errorf("states %+v do not match process list %v", s.States, counts)
	}
}

func TestMissingModule(t *testing.T) {
	r := &Reader{Dir: t.TempDir()}
	if r.Loaded() {
		t.Fatal("Loaded() = true for an empty dir")
	}
	_, err := r.Snapshot()
	if err == nil || !strings.Contains(err.Error(), "is the kernel module loaded?") {
		t.Fatalf("err = %v, want a hint about loading the module", err)
	}
}

func TestUsedKBNeverUnderflows(t *testing.T) {
	if got := (Memory{TotalKB: 10, AvailableKB: 20}).UsedKB(); got != 0 {
		t.Fatalf("UsedKB = %d, want 0", got)
	}
}
