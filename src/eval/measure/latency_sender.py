from io import TextIOWrapper
import socket
import sys
import threading
import queue
import time

ID_SIZE = 8
period_millis = 100
extra_payload_size = 0

if len(sys.argv) < 3:
    print('usage: python latency_sender.py DEST_IPV4 DEST_PORT SRC_PORT LOG_PATH [PAYLOAD_SIZE] [PERIOD_MILLIS]')
    exit(1)

dest_ip = sys.argv[1]
dest_port = int(sys.argv[2])
src_port = int(sys.argv[3])
log_path = sys.argv[4]

if len(sys.argv) > 5:
    extra_payload_size = int(sys.argv[5])

if len(sys.argv) > 6:
    period_millis = int(sys.argv[6])


print(f'sender: to {dest_ip}:{dest_port}, payload size: {extra_payload_size}, period: {period_millis}ms')

sock = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)
sock.setsockopt(socket.SOL_SOCKET, socket.SO_REUSEADDR, 1)
sock.bind(("0.0.0.0", src_port))

q = queue.SimpleQueue()

def receiver():
    print('running receiver')
    buff_size = ID_SIZE + extra_payload_size
    while True:
        resp, _ = sock.recvfrom(buff_size)
        now = time.time()
        id_ = int(resp[:ID_SIZE].decode().strip())
        q.put((id_, now)) 

def sender(log_file: TextIOWrapper):
    print('running sender')
    msgs = []
    i = 0
    latencies = []
    dropped_repeated_or_out_of_order = 0
    while True:
        while msgs:
            try:
                s_id_, snd_time = msgs[0]
                id_, rcv_time = -1, -1
                while True:
                    id_, rcv_time = q.get_nowait()
                    if id_ < s_id_:
                        dropped_repeated_or_out_of_order += 1
                    else:
                        break
                if id_ != s_id_:
                    while msgs[0][0] < id_:
                        msgs.pop(0)
                        dropped_repeated_or_out_of_order += 1
                    s_id_, snd_time = msgs[0]
                msgs.pop(0)
                latency = (rcv_time - snd_time) / 2
                latencies.append(latency)
                log=f'{time.time()},{latency},{dropped_repeated_or_out_of_order}'
                print(log)
                log_file.write(log+"\n")
            except queue.Empty:
                break
        time.sleep(float(period_millis) / 1000)
        num = str(i)
        num += " " * (ID_SIZE - len(num))
        msg = num + (extra_payload_size * "A")
        now = time.time()
        sock.sendto(msg.encode(), (dest_ip, dest_port))
        msgs.append((i, now))
        i += 1

threading.Thread(target=receiver).start()

with open(log_path, 'w') as f:
    sender(f)
