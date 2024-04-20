import socket
import sys
import threading
import time

payload_size = 940
packets_per_second_per_address = 1000

if len(sys.argv) < 2:
    # e.g. udp_spammer.py 5325:10.10.0.1:4354 5326:10.10.0.2:5643 (src_port:dst_ip:dst_port)
    print('usage: python udp_spammer.py ADDRESSES...') 
    exit(1)

def sender(src_port, dst_ip, dst_port):
    total_sent = 0
    print(f'running sender from port: {src_port} to {dst_ip}:{dst_port}')
    sock = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)
    sock.setsockopt(socket.SOL_SOCKET, socket.SO_REUSEADDR, 1)
    sock.bind(("0.0.0.0", int(src_port)))
    time_to_sleep = 1.0 / packets_per_second_per_address
    msg = ('A' * payload_size).encode()
    log_total_sent_period = 100
    i = 0
    while True:
        sock.sendto(msg, (dst_ip, int(dst_port)))
        i += 1
        total_sent += 1
        if i % log_total_sent_period == 0:
            print(f'total: {total_sent}')
        time.sleep(time_to_sleep)


threads = []
for cfg in sys.argv[1:]:
    th = threading.Thread(target=sender, args=cfg.split(':'))
    threads.append(th)
    th.start()

for th in threads:
    th.join()

print('done')
