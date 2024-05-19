import argparse
import asyncio
import random

from fastapi import FastAPI, Response
from pydantic import BaseModel


def parse_params():
    parser = argparse.ArgumentParser(description='HTTP receiver.')
    parser.add_argument("-p", "--http_port", default=8000, type=int, 
            help="Port on which HTTP server should be started")
    parser.add_argument("-u", "--mean_latency", default=20, type=int,
            help="Mean number of milliseconds of emulated processing latency, normal distrubution")
    parser.add_argument("-s", "--std_latency", default=10, type=int,
            help="Std of emulated processing latency, normal distrubution")
    
    return parser.parse_args()

class Req(BaseModel):
    response_size: int # this is just content-length use ~800 to have 1KB response size

class RawResponse(Response):
    media_type = "binary/octet-stream"

    def render(self, content: bytes) -> bytes:
        return bytes([b for b in content])

if __name__ == "__main__":
    import uvicorn
    args = parse_params()
    port = args.http_port
    mu = float(args.mean_latency)
    std = float(args.std_latency)
    print(f'delay: N({mu}, {std})')

    app = FastAPI()
    @app.post("/")
    async def handle_request(data: Req):
        delay = max(random.normalvariate(mu, std), 0)
        if delay > 0:
            await asyncio.sleep(delay / 1000)
        resp = random.randbytes(data.response_size)
        return RawResponse(content=resp)
        
    print(f'running server on 0.0.0.0:{port}')
    uvicorn.run(app, host='0.0.0.0', port=port)
