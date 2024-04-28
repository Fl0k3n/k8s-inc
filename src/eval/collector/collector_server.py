import logging
import socket
from abc import ABC, abstractmethod

from intreport import IntReport

logger = logging.getLogger('int_collector')

def unpack_int_report(packet):
    return IntReport(packet)

class ReportConsumer(ABC):
    @abstractmethod
    def consume(self, report: IntReport):
        pass
    
def start_udp_server(port: int, consumers: list[ReportConsumer]):
    bufferSize  = 65565

    sock = socket.socket(family=socket.AF_INET, type=socket.SOCK_DGRAM)
    sock.bind(("0.0.0.0", port))
    logger.info("UDP server up and listening at UDP port: %d" % port)

    while True:
        message, address = sock.recvfrom(bufferSize)
        logger.info(f"Received INT report ({len(message)} bytes) from: {address}")
        try:
            report = unpack_int_report(message)
            if report:
                logger.debug(f'report: {report}')
                for consumer in consumers:
                    consumer.consume(report)
        except Exception as e:
            logger.error("Exception during handling the INT report", exc_info=e)
