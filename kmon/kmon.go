// Package kmon reads the JSON files that the kmon kernel module exposes under
// /proc and combines them into a single snapshot.
package kmon

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"
)

// Process is one entry of /proc/kmon_procs.
type Process struct {
	PID     int    `json:"pid"`
	PPID    int    `json:"ppid"`
	Name    string `json:"name"`
	State   string `json:"state"` // one-letter state as shown by ps: R, S, D, T, Z, I...
	UID     uint32 `json:"uid"`
	Threads int    `json:"threads"`
	RSSKB   uint64 `json:"rss_kb"`
}

// Memory is the content of /proc/kmon_mem, in KiB.
type Memory struct {
	TotalKB     uint64 `json:"total_kb"`
	FreeKB      uint64 `json:"free_kb"`
	AvailableKB uint64 `json:"available_kb"`
	BuffersKB   uint64 `json:"buffers_kb"`
	SharedKB    uint64 `json:"shared_kb"`
}

// UsedKB is memory that cannot be reclaimed right now (total minus available).
func (m Memory) UsedKB() uint64 {
	if m.AvailableKB > m.TotalKB {
		return 0
	}
	return m.TotalKB - m.AvailableKB
}

// States is the content of /proc/kmon_states.
type States struct {
	Running   int `json:"running"`
	Sleeping  int `json:"sleeping"`
	DiskSleep int `json:"disk_sleep"`
	Stopped   int `json:"stopped"`
	Zombie    int `json:"zombie"`
	Idle      int `json:"idle"`
	Other     int `json:"other"`
	Total     int `json:"total"`
}

// Host describes the machine the snapshot was taken on.
type Host struct {
	Kernel        string  `json:"kernel"`
	CPUs          int     `json:"cpus"`
	UptimeSeconds float64 `json:"uptime_seconds"`
}

// Snapshot is everything the dashboard needs, read at one point in time.
type Snapshot struct {
	TakenAt   time.Time `json:"taken_at"`
	Host      Host      `json:"host"`
	Memory    Memory    `json:"memory"`
	States    States    `json:"states"`
	Processes []Process `json:"processes"`
}

// Reader reads kmon files from a proc directory (normally /proc).
type Reader struct {
	Dir string
}

// NewReader returns a Reader for the real /proc.
func NewReader() *Reader { return &Reader{Dir: "/proc"} }

// Loaded reports whether the kernel module's files are present.
func (r *Reader) Loaded() bool {
	_, err := os.Stat(filepath.Join(r.Dir, "kmon_procs"))
	return err == nil
}

func (r *Reader) Processes() ([]Process, error) {
	var ps []Process
	return ps, r.readJSON("kmon_procs", &ps)
}

func (r *Reader) Memory() (Memory, error) {
	var m Memory
	return m, r.readJSON("kmon_mem", &m)
}

func (r *Reader) States() (States, error) {
	var s States
	return s, r.readJSON("kmon_states", &s)
}

// Snapshot reads every kmon file plus basic host information.
func (r *Reader) Snapshot() (*Snapshot, error) {
	ps, err := r.Processes()
	if err != nil {
		return nil, err
	}
	mem, err := r.Memory()
	if err != nil {
		return nil, err
	}
	st, err := r.States()
	if err != nil {
		return nil, err
	}
	return &Snapshot{
		TakenAt:   time.Now().UTC(),
		Host:      r.host(),
		Memory:    mem,
		States:    st,
		Processes: ps,
	}, nil
}

func (r *Reader) readJSON(name string, v any) error {
	data, err := os.ReadFile(filepath.Join(r.Dir, name))
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("kmon: %s not found in %s (is the kernel module loaded?)", name, r.Dir)
		}
		return fmt.Errorf("kmon: %w", err)
	}
	if err := json.Unmarshal(data, v); err != nil {
		return fmt.Errorf("kmon: parse %s: %w", name, err)
	}
	return nil
}

// host is best effort: missing files just leave fields empty.
func (r *Reader) host() Host {
	h := Host{CPUs: runtime.NumCPU()}
	if b, err := os.ReadFile(filepath.Join(r.Dir, "sys/kernel/osrelease")); err == nil {
		h.Kernel = strings.TrimSpace(string(b))
	}
	if b, err := os.ReadFile(filepath.Join(r.Dir, "uptime")); err == nil {
		if fields := strings.Fields(string(b)); len(fields) > 0 {
			h.UptimeSeconds, _ = strconv.ParseFloat(fields[0], 64)
		}
	}
	return h
}
