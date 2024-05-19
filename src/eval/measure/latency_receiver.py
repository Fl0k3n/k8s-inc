import random
import socket
import sys
import time

ID_SIZE = 8
extra_payload_size = 0

if len(sys.argv) < 2:
    print('usage: python latency_receiver.py PORT PAYLOAD_SIZE MEAN_DELAY_MS STD_DELAY_MS')
    exit(1)

port = int(sys.argv[1])
extra_payload_size = int(sys.argv[2])
mu = float(sys.argv[3]) / 1000.
std = float(sys.argv[4]) / 1000.

print(f'receiver: port: {port} payload size: {extra_payload_size}')

sock = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)
sock.setsockopt(socket.SOL_SOCKET, socket.SO_REUSEADDR, 1)
sock.bind(("0.0.0.0", port))

def receiver():
    print('running receiver')
    buff_size = ID_SIZE + extra_payload_size
    while True:
        try:
            msg, addr = sock.recvfrom(buff_size)
            delay = max(0, random.normalvariate(mu, std))
            if delay > 0:
                time.sleep(delay)
            sock.sendto(msg, addr)
            print(f'sent response to {addr}')
        except Exception as e:
            print(e)

receiver()
