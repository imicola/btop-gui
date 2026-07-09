#!/usr/bin/env python3
"""大量短进程负载：维持一个同时存活的短命进程池，抬高 /proc 下的瞬时 PID 目录数。

子进程为 `sleep 3`，主进程循环补充使存活数稳定在目标值（默认 800），
并及时 poll() 回收僵尸。这样 ListProcesses 遍历 /proc + 逐 PID 读
stat/status 的成本随进程数线性上升，体现"大量短进程"负载。

用法: short_proc_load.py <秒数> [目标并发数]
"""
import subprocess
import sys
import time

SLEEP = "3"


def main():
    secs = float(sys.argv[1]) if len(sys.argv) > 1 else 60
    target = int(sys.argv[2]) if len(sys.argv) > 2 else 800
    end = time.time() + secs
    children = []
    spawned = 0
    peak = 0
    while time.time() < end:
        while len(children) < target and time.time() < end:
            try:
                children.append(subprocess.Popen(["sleep", SLEEP]))
                spawned += 1
            except OSError:
                break
        # poll() 回收已退出的子进程（避免僵尸堆积，同时腾出名额）
        children = [p for p in children if p.poll() is None]
        if len(children) > peak:
            peak = len(children)
        time.sleep(0.05)
    for p in children:
        p.terminate()
    for p in children:
        p.wait(timeout=2)
    print(f"[short_proc_load] spawned={spawned} peak_alive={peak} over {secs}s", flush=True)


if __name__ == "__main__":
    main()
