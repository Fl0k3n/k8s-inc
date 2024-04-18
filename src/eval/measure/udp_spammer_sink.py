import socket
import sys
import time

ID_SIZE = 8
buff_size = 64000

if len(sys.argv) < 2:
    print('usage: python udp_spammer_sink.py PORT...')
    exit(1)


sock = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)
sock.setsockopt(socket.SOL_SOCKET, socket.SO_REUSEADDR, 1)

port = int(sys.argv[1])
sock.bind(("0.0.0.0", port))

def receiver():
    print('running receiver')
    total_rcvd_from_start = 0
    start_time = 0
    total_size = 0
    approx_header_size = 42 # eth,ip,udp
    log_period = 100
    i = 0
    reset_period_seconds = 3
    while True:
        data, _ = sock.recvfrom(buff_size)
        total_rcvd_from_start += 1
        if start_time == 0:
            start_time = time.time()
        total_size += len(data) + approx_header_size
        i += 1
        if i % log_period == 0:
            now = time.time()
            throughput = total_size / (now - start_time)
            print(f'{throughput / 1000.0}kBps\trcvd: {total_rcvd_from_start}')
        if time.time() - start_time > reset_period_seconds:
            i = 0
            start_time = 0
            total_size = 0

receiver()
