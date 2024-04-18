import socket
import sys

ID_SIZE = 8
extra_payload_size = 0

if len(sys.argv) < 2:
    print('usage: python latency_rcvr.py PORT [PAYLOAD_SIZE]')
    exit(1)

src_port = int(sys.argv[1])

if len(sys.argv) > 2:
    extra_payload_size = int(sys.argv[2])

print(f'receiver: port: {src_port} payload size: {extra_payload_size}')

sock = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)
sock.setsockopt(socket.SOL_SOCKET, socket.SO_REUSEADDR, 1)
sock.bind(("0.0.0.0", src_port))

def receiver():
    print('running receiver')
    buff_size = ID_SIZE + extra_payload_size
    while True:
        msg, addr = sock.recvfrom(buff_size)
        sock.sendto(msg, addr)
        print('sent response')

receiver()
