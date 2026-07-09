#!/usr/bin/env python3
"""多核 CPU 负载生成：开 N 个进程各跑满一个核的死循环。

用法: cpu_load.py <秒数> [核数]
默认核数 = CPU 逻辑核数（打满整机）。
"""
import multiprocessing as mp
import os
import sys
import time


def burn():
    while True:
        pass


def main():
    secs = float(sys.argv[1]) if len(sys.argv) > 1 else 60
    n = int(sys.argv[2]) if len(sys.argv) > 2 else os.cpu_count()
    ps = [mp.Process(target=burn, daemon=True) for _ in range(n)]
    for p in ps:
        p.start()
    print(f"[cpu_load] {n} cores burning for {secs}s (pid={os.getpid()})", flush=True)
    time.sleep(secs)
    for p in ps:
        p.terminate()
    for p in ps:
        p.join(timeout=2)
    print("[cpu_load] done", flush=True)


if __name__ == "__main__":
    main()
