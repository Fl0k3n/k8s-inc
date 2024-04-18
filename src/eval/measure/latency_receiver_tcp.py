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

sock = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
sock.setsockopt(socket.SOL_SOCKET, socket.SO_REUSEADDR, 1)
sock.bind(("0.0.0.0", src_port))
sock.listen(1)

def receiver():
    print('running receiver')
    buff_size = ID_SIZE + extra_payload_size
    while True:
        print('waiting for connection')
        client_sock, _ = sock.accept()
        print('connected')
        try:
            while True:
                data = b''
                buff = b'' 
                while len(data) < buff_size:
                    buff = client_sock.recv(buff_size - len(data))
                    if len(buff) == 0:
                        raise Exception("EOF")
                    data += buff
                if client_sock.send(data) < len(data):
                    raise Exception("didn't send full")
                print('sent response')
        except Exception as e:
            client_sock.close()
            print(e)
receiver()
