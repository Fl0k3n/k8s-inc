import queue
import select
import socket
import sys
import time
from io import TextIOWrapper

ID_SIZE = 8
period_millis = 100
extra_payload_size = 0

if len(sys.argv) < 3:
    print('usage: python latency_sender.py DEST_IPV4 DEST_PORT TIMEOUT_MS LOG_PATH [PAYLOAD_SIZE] [PERIOD_MILLIS]')
    exit(1)

dest_ip = sys.argv[1]
dest_port = int(sys.argv[2])
timeout_ms = int(sys.argv[3])
log_path = sys.argv[4]

if len(sys.argv) > 5:
    extra_payload_size = int(sys.argv[5])

if len(sys.argv) > 6:
    period_millis = int(sys.argv[6])


print(f'sender: to {dest_ip}:{dest_port}, payload size: {extra_payload_size}, period: {period_millis}ms, timeout: {timeout_ms}ms')

def sender(log_file: TextIOWrapper):
    print('running sender')
    i = 0
    dropped_repeated_or_out_of_order = 0
    while True:
        try:
            num = str(i)
            num += " " * (ID_SIZE - len(num))
            msg = num + (extra_payload_size * "A")
            
            sock = socket.socket(socket.AF_INET, socket.SOCK_DGRAM) 
            sock.setblocking(0)
            start = time.time()
            sock.sendto(msg.encode(), (dest_ip, dest_port))

            buff_size = ID_SIZE + extra_payload_size
            ready = select.select([sock], [], [], timeout_ms / 1000.0)
            now = time.time()
            if ready[0]:
                resp, _ = sock.recvfrom(buff_size)
                id_ = int(resp[:ID_SIZE].decode().strip())
                if id_ != i:
                    dropped_repeated_or_out_of_order += 1
            else:
                print('timeout')
                dropped_repeated_or_out_of_order += 1
            latency = now - start
            log=f'{time.time()},{latency},{dropped_repeated_or_out_of_order}'
            print(log)
            log_file.write(log+"\n")
            time_to_sleep = max(0., (period_millis / 1000.) - latency)
            if time_to_sleep > 0:
                time.sleep(time_to_sleep)
            now = time.time()
            i += 1
            sock.close()
        except Exception as e:
            print(e)
            time.sleep(period_millis / 1000.)

with open(log_path, 'w') as f:
    sender(f)
