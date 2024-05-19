import argparse
import asyncio
import queue
import sys
import time

import aiohttp

# python3 ./sender_http.py --server_url "http://0.0.0.0:8000/" --timeout_millis 1000 --request_period_millis 50 --response_size 800 --log_path "./log.csv"

def parse_params():
    parser = argparse.ArgumentParser(description='HTTP sender asyncio.')
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

data = {'response_size': args.response_size}

timed_out = 0
sleep_time = args.request_period_millis / 1000.

max_awaiting = 1
class Counter:
    def __init__(self) -> None:
        self.count = 0

async def sender():
    while True:
        try:
            # connector = aiohttp.TCPConnector(limit=3)
            counter = Counter()
            # async with aiohttp.ClientSession(connector=connector) as session:
            while True:
                counter.count += 1
                asyncio.get_running_loop().create_task(send_request(counter))
                # asyncio.get_running_loop().create_task(send_request(session, counter))
                await asyncio.sleep(sleep_time)
                while counter.count > max_awaiting:
                    await asyncio.sleep(0.1)
        except Exception as e:
            print(e)
            await asyncio.sleep(0.5)

async def send_request(counter):
    global timed_out
    start = time.time()
    try:
        async with aiohttp.ClientSession() as session:
            async with session.post(url, json=data, timeout=timeout) as response:
                await response.read()
    except asyncio.TimeoutError:
        print(f"Request to {url} timed out")
        timed_out += 1
    except Exception as e:
        print(e)
        await asyncio.sleep(0.1) 
        counter.count -= 1
        return

    now = time.time()
    latency = now - start
    log=f'{now},{latency},{timed_out}\n'
    log_file.write(log)
    counter.count -= 1

asyncio.get_event_loop().run_until_complete(sender())
