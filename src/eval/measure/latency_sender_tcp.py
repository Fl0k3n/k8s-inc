import socket
import sys
import threading
import queue
import time

ID_SIZE = 8
period_millis = 100
extra_payload_size = 0

if len(sys.argv) < 3:
    print('usage: python latency_sender.py DEST_IPV4 DEST_PORT SRC_PORT [PAYLOAD_SIZE] [PERIOD_MILLIS]')
    exit(1)

dest_ip = sys.argv[1]
dest_port = int(sys.argv[2])
src_port = int(sys.argv[3])

if len(sys.argv) > 4:
    extra_payload_size = int(sys.argv[4])

if len(sys.argv) > 5:
    period_millis = int(sys.argv[5])


print(f'sender: to {dest_ip}:{dest_port}, payload size: {extra_payload_size}, period: {period_millis}ms')

sock = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
sock.setsockopt(socket.SOL_SOCKET, socket.SO_REUSEADDR, 1)
sock.bind(("0.0.0.0", src_port))
print('connecting')
sock.connect((dest_ip, dest_port))
print('connected')

q = queue.SimpleQueue()

def receiver():
    print('running receiver')
    buff_size = ID_SIZE + extra_payload_size
    try:
        data = b''
        while True:
            buff = b'' 
            while len(data) < buff_size:
                buff = sock.recv(buff_size - len(data))
                if len(buff) == 0:
                    raise Exception("EOF")
                data += buff
                now = time.time()
                id_ = int(data[:ID_SIZE].decode().strip())
                data = data[buff_size:]
                q.put((id_, now)) 
            print('sent response')
    except Exception as e:
        sock.close()
        print(e)

def sender():
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
                    j = 0
                    while msgs[j][0] < id_:
                        j += 1
                    dropped_repeated_or_out_of_order += j
                    msgs = msgs[j:]
                    s_id_, snd_time = msgs[0]
                else:
                    msgs.pop(0)
                latency = (rcv_time - snd_time) / 2
                latencies.append(latency)
                print(f'lat: {latency}s\tdropped_ish: {dropped_repeated_or_out_of_order}')
            except queue.Empty:
                break
        time.sleep(float(period_millis) / 1000)
        num = str(i)
        num += " " * (ID_SIZE - len(num))
        msg = num + (extra_payload_size * "A")
        now = time.time()
        data = msg.encode()
        if sock.send(data) < len(data):
            raise Exception("didn't send all")
        msgs.append((i, now))
        i += 1

threading.Thread(target=receiver).start()
sender()
