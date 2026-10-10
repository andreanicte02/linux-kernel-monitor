#!/usr/bin/env bash
# Builds kmon.ko for the running kernel, loads it, checks every /proc file,
# takes a snapshot for the dashboard and unloads it. Used by CI; also works
# on any Linux box with kernel headers and sudo.
set -euo pipefail
cd "$(dirname "$0")/.."

echo "::group::Build for $(uname -r)"
make -C module 2>&1 | tee module/build.log
if grep -E "kmon_[a-z]+\.c:[0-9]+:[0-9]+: (warning|error)" module/build.log; then
  echo "kmon: compiler warnings in the module" >&2
  exit 1
fi
echo "::endgroup::"

sudo dmesg --clear
sudo insmod module/kmon.ko
trap 'sudo rmmod kmon 2>/dev/null || true' EXIT
echo "loaded: $(lsmod | grep '^kmon')"

for f in kmon_procs kmon_mem kmon_states; do
  python3 -m json.tool "/proc/$f" > /dev/null
  echo "/proc/$f: valid JSON ($(wc -c < "/proc/$f") bytes)"
done

# A process whose kernel name (comm) contains a quote and a backslash: if the
# module did not escape it, /proc/kmon_procs would stop being valid JSON.
python3 -c 'import ctypes, time; ctypes.CDLL(None).prctl(15, b"we\"ird\\nm", 0, 0, 0); time.sleep(30)' &
WEIRD_PID=$!
sleep 0.5

# Read the process list while processes are being created and destroyed,
# to exercise the RCU walk under churn.
for i in $(seq 200); do
  (sleep 0.01 &)
  python3 -m json.tool /proc/kmon_procs > /dev/null
done
echo "200 reads under process churn: ok"

WEIRD_PID=$WEIRD_PID python3 - <<'PY'
import json, os
procs = json.load(open("/proc/kmon_procs"))
states = json.load(open("/proc/kmon_states"))
mem = json.load(open("/proc/kmon_mem"))
pids = {p["pid"] for p in procs}
weird = [p for p in procs if p["pid"] == int(os.environ["WEIRD_PID"])]
assert weird and weird[0]["name"] == 'we"ird\\nm', weird
print("escaped process name round-trips:", weird[0]["name"])
assert 1 in pids, "init (PID 1) missing"
assert len(pids) == len(procs), "duplicate PIDs"
assert abs(states["total"] - len(procs)) <= 20, (states["total"], len(procs))
assert 0 < mem["available_kb"] <= mem["total_kb"], mem
meminfo = {l.split(":")[0]: int(l.split()[1]) for l in open("/proc/meminfo")}
assert abs(mem["total_kb"] - meminfo["MemTotal"]) <= 4, (mem["total_kb"], meminfo["MemTotal"])
print(f"{len(procs)} processes, {mem['total_kb'] // 1024} MB RAM: consistent with /proc/meminfo")
PY

kill "$WEIRD_PID"
go run ./cmd/kmon snapshot > web/snapshot.json
echo "snapshot: $(python3 -c 'import json;d=json.load(open("web/snapshot.json"));print(len(d["processes"]), "processes on", d["host"]["kernel"])')"

sudo rmmod kmon
trap - EXIT
test ! -e /proc/kmon_procs
echo "unloaded cleanly"

if sudo dmesg | grep -E "BUG:|Oops|WARNING:|general protection"; then
  echo "kmon: kernel reported a problem" >&2
  exit 1
fi
sudo dmesg | grep kmon || true
