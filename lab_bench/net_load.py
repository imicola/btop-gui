#!/usr/bin/env python3
"""回环网络持续打流负载：server 丢弃接收数据，client 持续发送零字节流。

走 lo 回环，无需外部工具。用大缓冲区尽量拉高吞吐，使 /proc/net/dev 计数器
持续增长，模拟"持续网络传输"负载。

用法: net_load.py <秒数>
"""
import multiprocessing as mp
import os
import socket
import sys
import time

PORT = 0


def server(port_q, stop):
    srv = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
    srv.setsockopt(socket.SOL_SOCKET, socket.SO_REUSEADDR, 1)
    srv.bind(("127.0.0.1", 0))
    srv.listen(8)
    port_q.put(srv.getsockname()[1])
    srv.settimeout(1.0)
    while not stop.is_set():
        try:
            conn, _ = srv.accept()
        except socket.timeout:
            continue
        conn.settimeout(1.0)
        try:
            while True:
                data = conn.recv(1 << 20)
                if not data:
                    break
        except socket.timeout:
            pass
        conn.close()
    srv.close()


def client(port, secs):
    time.sleep(0.3)
    s = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
    for _ in range(50):
        try:
            s.connect(("127.0.0.1", port))
            break
        except OSError:
            time.sleep(0.1)
    payload = b"\x00" * (1 << 20)  # 1 MiB
    end = time.time() + secs
    total = 0
    while time.time() < end:
        try:
            s.sendall(payload)
            total += len(payload)
        except OSError:
            break
    try:
        s.close()
    except OSError:
        pass
    print(f"[net_load] sent {total / (1<<20):.0f} MiB over loopback", flush=True)


def main():
    secs = float(sys.argv[1]) if len(sys.argv) > 1 else 60
    port_q = mp.Queue()
    stop = mp.Event()
    srv = mp.Process(target=server, args=(port_q, stop), daemon=True)
    srv.start()
    port = port_q.get(timeout=5)
    cli = mp.Process(target=client, args=(port, secs), daemon=True)
    cli.start()
    print(f"[net_load] loopback transfer for {secs}s (pid={os.getpid()})", flush=True)
    cli.join()
    stop.set()
    srv.join(timeout=3)
    print("[net_load] done", flush=True)


if __name__ == "__main__":
    main()
