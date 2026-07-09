#!/usr/bin/env bash
# 30 分钟连续运行场景：空闲负载下每秒采样，验证长时间稳定性。
cd "$(dirname "$0")/.."
mkdir -p lab_bench/logs
GOCACHE=/tmp/btop-gocache \
BTOP_BENCH=1 BTOP_SCENE=long BTOP_DURATION=1800 BTOP_INTERVAL=1 \
  go test -run '^TestIntegrationBench$' -tags webkit2_41 -timeout 40m -v . \
  > lab_bench/logs/long.log 2>&1
echo DONE >> lab_bench/logs/long.log
