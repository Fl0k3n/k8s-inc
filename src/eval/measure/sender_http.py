import argparse
import json
import sys
import time

import requests

# python3 ./sender_http.py --server_url "http://0.0.0.0:8000/" --timeout_millis 1000 --request_period_millis 50 --response_size 800 --log_path "./log.csv"

def parse_params():
    parser = argparse.ArgumentParser(description='INTCollector client.')
    parser.add_argument("-u", "--server_url", default="http://0.0.0.0:8000/", type=str,
            help="URL on which HTTP server is running")
    parser.add_argument("-t", "--timeout_millis", default=1000, type=int,
            help="http request timeout")
    parser.add_argument("-q", "--request_period_millis", default=50, type=int, 
            help="minimum number of milliseconds between each request")
    parser.add_argument("-s", "--response_size", default=800, type=int, 
            help="response content-length")
    parser.add_argument("-l", "--log_path", default="", type=str, 
            help="path where logs should be saved")
    return parser.parse_args()

args = parse_params()
url = args.server_url
timeout = float(args.timeout_millis) / 1000.
log_path = args.log_path
log_file = None
if log_path == "":
    log_file = sys.stdout
else:
    log_file = open(log_path, 'w')

data = json.dumps({'response_size': args.response_size})

timed_out = 0
sleep_time = args.request_period_millis / 1000.

while True:
    start = time.time()
    try:
        resp = requests.post(url, data=data, timeout=timeout)
    except Exception as e:
        print(e)
        timed_out += 1
    latency = time.time() - start

    log=f'{time.time()},{latency},{timed_out}\n'
    log_file.write(log)
    if log_path != "":
        print(log)

    sleep = sleep_time - latency
    if sleep > 0:
        time.sleep(sleep)
