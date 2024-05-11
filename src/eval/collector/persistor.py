import logging
import time
from abc import abstractmethod
from contextlib import contextmanager
from typing import Generator, TextIO

from collector_server import ReportConsumer
from intreport import HopMetadata, IntReport

logger = logging.getLogger('int_collector')

class LogPersistor(ReportConsumer):
    @abstractmethod
    def persist(self, report: IntReport):
        pass

    def consume(self, report: IntReport):
        return self.persist(report)

class DummyLogPersistor(LogPersistor):
    def persist(self, report: IntReport):
        pass

class CsvLogPersistor(LogPersistor):
    def __init__(self, file: TextIO, force_flush_period=1000) -> None:
        self.file = file
        self.counter = 0
        self.force_flush_period = force_flush_period

    def write_csv_headers(self):
        headers  = 'collector_timestamp,flow_id,collection_id,src_ip,dst_ip,src_port,dst_port,protocol,seq_num,reporting_sw,hops'
        headers += ',egress_port_tx_util,egress_timestamp,hop_latency,ingress_timestamp'
        headers += ',l1_egress_port_id,l1_ingress_port_id,l2_egress_port_id,l2_ingress_port_id'
        headers += ',queue_occupancy,queue_occupancy_id,switch_id'
        headers += '\n'
        self.file.write(headers)
        self.file.flush()

    def persist(self, report: IntReport):
        f = report.flow_id
        common  = f'{time.time()},{self.counter},{report.collection_id},{f["srcip"]},{f["dstip"]},{f["srcp"]},{f["dstp"]},{f["protocol"]}'
        common += f',{report.seq_num},{report.switch_id},{report.hop_count}'

        for hop in report.hop_metadata:
            metrics  = f',{hop.egress_port_tx_util},{hop.egress_timestamp},{hop.hop_latency},{hop.ingress_timestamp}'
            metrics += f',{hop.l1_egress_port_id},{hop.l1_ingress_port_id},{hop.l2_egress_port_id},{hop.l2_ingress_port_id}'
            metrics += f',{hop.queue_occupancy},{hop.queue_occupancy_id},{hop.switch_id}'
            self.file.write(f'{common}{metrics}\n')
        self.counter += 1
        if self.counter % self.force_flush_period == 0:
            self.file.flush()

@contextmanager
def persistor(output_path='') -> Generator[LogPersistor, None, None]:
    if output_path == '':
        logger.info("Saving logs to csv disabled")
        yield DummyLogPersistor()
    else:
        f = open(output_path, 'w')
        try:
            p = CsvLogPersistor(f)
            p.write_csv_headers()
            yield p
        finally:
            f.close()
