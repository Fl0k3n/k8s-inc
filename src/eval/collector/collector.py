import argparse
import logging
import threading

from fastapi import FastAPI
from pydantic import BaseModel

from collector_server import start_udp_server
from persistor import persistor
from stats_engine import MetricsSummary, StatsEngine

log_format = "[%(asctime)s] [%(levelname)s] - %(message)s"
logging.basicConfig(level=logging.INFO, format=log_format)
logger = logging.getLogger('int_collector')

def parse_params():
    parser = argparse.ArgumentParser(description='INTCollector server.')

    parser.add_argument("-i", "--int_port", default=6000, type=int,
        help="Destination port of INT Telemetry reports")
    
    parser.add_argument("-p", "--http_port", default=8000, type=int,
        help="Port on which HTTP metrics API should be exposed")
    
    parser.add_argument("-o", "--output_path", default='', type=str,
        help="File where reports should be serialized in json")
    
    parser.add_argument("-w", "--time_window", default=30, type=int,
        help="Number of seconds for which metrics should be collected")

    return parser.parse_args()

class MetricsRequest(BaseModel):
    collection_id: int

if __name__ == "__main__":
    import uvicorn
    args = parse_params()
    collector_port = args.int_port
    reports_output_path = args.output_path
    http_port = args.http_port
    time_window = args.time_window
    stats_engine = StatsEngine(time_window)

    app = FastAPI()

    @app.post("/")
    async def compute_metrics(req: MetricsRequest) -> MetricsSummary:
        return stats_engine.get_summary(req.collection_id)

    with persistor(reports_output_path) as p:
        report_consumers = [p, stats_engine]
        collector_thread = threading.Thread(target=start_udp_server, args=(collector_port, report_consumers))
        collector_thread.start()
        uvicorn.run(app, host='0.0.0.0', port=http_port)
        collector_thread.join()
