#!/usr/bin/env bash
# 集成测试编排：逐场景启动负载 → 运行采样基准 → 抓取 RESULT 行。
#
# 4 个短场景各 90s（负载先启动 10s 稳定后采样），30 分钟连续场景后台单独跑。
# 结果汇总到 lab_bench/results.tsv。
set -o pipefail
cd "$(dirname "$0")/.."

PY=python3
RES=lab_bench/results.tsv
LOG=lab_bench/logs
mkdir -p "$LOG"
: > "$RES"

run_go_bench() {  # $1=scene $2=duration
  local scene=$1 dur=$2
  GOCACHE=/tmp/btop-gocache \
  BTOP_BENCH=1 BTOP_SCENE="$scene" BTOP_DURATION="$dur" BTOP_INTERVAL=1 \
  go test -run '^TestIntegrationBench$' -tags webkit2_41 -timeout "${3:-5}m" -v . \
    > "$LOG/$scene.log" 2>&1
  grep -m1 '^RESULT' "$LOG/$scene.log" >> "$RES" \
    || echo -e "RESULT\t$scene\tFETCH_FAILED"
}

start_load() {  # $1=scene -> 后台启动负载，写 PID 到 $LOG/$1.pid
  local scene=$1
  case $scene in
    cpu)   $PY lab_bench/cpu_load.py 120 20 & echo $! > "$LOG/cpu.pid" ;;
    net)   $PY lab_bench/net_load.py 120 & echo $! > "$LOG/net.pid" ;;
    short) $PY lab_bench/short_proc_load.py 120 800 & echo $! > "$LOG/short.pid" ;;
    idle|long) : ;;
  esac
}

kill_load() {  # $1=scene
  local scene=$1
  local pf="$LOG/$scene.pid"
  [[ -f $pf ]] && kill "$(cat "$pf")" 2>/dev/null; rm -f "$pf"
}

echo "===== [1/4] 系统空闲 ====="
start_load idle; sleep 3; run_go_bench idle 90 5; kill_load idle

echo "===== [2/4] 多核 CPU 负载 ====="
start_load cpu; sleep 12; run_go_bench cpu 90 5; kill_load cpu; sleep 3

echo "===== [3/4] 持续网络传输 ====="
start_load net; sleep 12; run_go_bench net 90 5; kill_load net; sleep 3

echo "===== [4/4] 大量短进程 ====="
start_load short; sleep 12; run_go_bench short 90 5; kill_load short

echo "===== 短场景完成，结果见 $RES ====="
cat "$RES"
