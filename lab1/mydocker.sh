#!/usr/bin/env bash
set -euo pipefail

NAME="${NAME:-mydocker}"
MEM="${MEM:-100M}"
CPU_MAX="${CPU_MAX:-50000 100000}"
PIDS_MAX="${PIDS_MAX:-20}"
RUN_AS="${RUN_AS:-1000}"
CG="/sys/fs/cgroup/$NAME"

if [ "$#" -lt 1 ]; then
    echo "usage: $0 command [args...]" >&2
    exit 2
fi

if ! grep -q memory /sys/fs/cgroup/cgroup.subtree_control; then
    mkdir -p /sys/fs/cgroup/init
    for p in $(cat /sys/fs/cgroup/cgroup.procs); do
        echo "$p" > /sys/fs/cgroup/init/cgroup.procs 2>/dev/null || true
    done
    echo "+cpu +memory +pids" > /sys/fs/cgroup/cgroup.subtree_control
fi

mkdir -p "$CG"
echo "$MEM" > "$CG/memory.max"
echo 0 > "$CG/memory.swap.max"
echo "$CPU_MAX" > "$CG/cpu.max"
echo "$PIDS_MAX" > "$CG/pids.max"

echo $$ > "$CG/cgroup.procs"

exec setpriv --reuid="$RUN_AS" --regid="$RUN_AS" --clear-groups \
    unshare --user --map-root-user --pid --fork --mount-proc --mount --uts --ipc --net \
    bash -c 'hostname sandbox
ip link set lo up
exec setpriv --bounding-set=-all --inh-caps=-all --ambient-caps=-all --no-new-privs /usr/local/bin/sandbox "$@"' _ "$@"
