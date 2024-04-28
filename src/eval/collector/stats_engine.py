import datetime
import threading

from pydantic import BaseModel

from collector_server import ReportConsumer
from intreport import HopMetadata, IntReport

MAX_COLLECTION_IDS = 128

class PortMetrics(BaseModel):
    num_packets: int
    egress_port_id: int
    average_latency_micro_s: int
    average_queue_fill_state: int
    one_percentile_slowest_latency: int
    one_percentile_largest_queue_fill_state: int

class SwitchMetrics(BaseModel):
    switch_id: int
    port_metrics: list[PortMetrics]

class Metrics(BaseModel):
    collected_reports: int
    average_path_latency_micro_s: int
    one_percentile_slowest_path_latency_micro_s: int
    device_metrics: list[SwitchMetrics]

class MetricsSummary(BaseModel):
    total_reports: int
    time_windows_seconds: int
    window_metrics: Metrics

class PortStatBuilder:
    def __init__(self, hm: HopMetadata) -> None:
        self.egress_port_id = hm.l1_egress_port_id
        self.latencies = []
        self.queue_fill_states = []

    def combine(self, hm: HopMetadata):
        self.latencies.append(hm.hop_latency)
        self.queue_fill_states.append(hm.queue_occupancy)

    def build(self) -> PortMetrics:
        self.latencies.sort(reverse=True)
        self.queue_fill_states.sort(reverse=True)
        num_packets = len(self.latencies)
        one_percentile = num_packets // 100
        return PortMetrics(
            num_packets=num_packets,
            egress_port_id=self.egress_port_id,
            average_latency_micro_s=int(sum(self.latencies) / num_packets),
            average_queue_fill_state=int(sum(self.queue_fill_states) / num_packets),
            one_percentile_slowest_latency=self.latencies[one_percentile],
            one_percentile_largest_queue_fill_state=self.queue_fill_states[one_percentile]
        )

class SwitchMetricsBuilder:
    def __init__(self, switch_id: int) -> None:
        self.switch_id = switch_id
        self.port_builders: list[PortStatBuilder] = []

    def _port_stat_builder(self, hm: HopMetadata) -> PortStatBuilder:
        while hm.l1_egress_port_id >= len(self.port_builders):
            self.port_builders.append(None)
        pb = self.port_builders[hm.l1_egress_port_id]
        if pb is None:
            pb = PortStatBuilder(hm)
            self.port_builders[hm.l1_egress_port_id] = pb
        return pb

    def add_report(self, hm: HopMetadata):
        self._port_stat_builder(hm).combine(hm)

    def build(self) -> SwitchMetrics:
        return SwitchMetrics(
            switch_id=self.switch_id,
            port_metrics=[pb.build() for pb in self.port_builders if pb is not None]
        )

class DeviceMetricsBuilder:
    def __init__(self) -> None:
        self.switch_metric_builders: list[SwitchMetricsBuilder] = []
    
    def _dispatch(self, hop_metadata: HopMetadata):
        swid = hop_metadata.switch_id
        while swid >= len(self.switch_metric_builders):
            self.switch_metric_builders.append(None)
        swm = self.switch_metric_builders[swid]
        if swm is None:
            swm = SwitchMetricsBuilder(swid)
            self.switch_metric_builders[swid] = swm
        swm.add_report(hop_metadata)

    def add_report(self, report: IntReport):
        # ignore sink's raport as its partially broken
        for hm in report.hop_metadata[:-1]:
            self._dispatch(hm)

    def build(self) -> list[SwitchMetrics]:
        return [sb.build() for sb in self.switch_metric_builders if sb is not None]

class StatsEngine(ReportConsumer):
    def __init__(self, time_window_seconds: int) -> None:
        self.time_window_seconds = time_window_seconds
        self.per_collection_id_total_raports = [0] * MAX_COLLECTION_IDS
        self.per_collection_id_stats: list[list[IntReport]] = []
        self.per_collection_id_locks = []
        for _ in range (MAX_COLLECTION_IDS):
            self.per_collection_id_stats.append([])
            self.per_collection_id_locks.append(threading.Lock())

    def consume(self, report: IntReport):
        with self.per_collection_id_locks[report.collection_id]:
            self.per_collection_id_total_raports[report.collection_id] += 1
            self.per_collection_id_stats[report.collection_id].append(report)
            
    def get_summary(self, collection_id: int) -> MetricsSummary:
        now = datetime.datetime.now()
        min_time = now - datetime.timedelta(seconds=self.time_window_seconds)
        device_metrics_builder = DeviceMetricsBuilder()
        
        with self.per_collection_id_locks[collection_id]:
            full_time_total_raports = self.per_collection_id_total_raports[collection_id]
            history = self.per_collection_id_stats[collection_id]
            collected_reports = 0
            path_latencies = []

            for i in range(len(history)-1, -1, -1):
                if history[i].rcvd_at < min_time:
                    break
                collected_reports += 1
                hm = history[i].hop_metadata
                path_latency = (hm[-1].egress_timestamp - hm[0].ingress_timestamp) / 1000
                path_latencies.append(path_latency)
                device_metrics_builder.add_report(history[i])

        if collected_reports > 0:
            avg_path_latency = sum(path_latencies) / collected_reports
            path_latencies.sort(reverse=True)
            one_percentile = collected_reports // 100
            one_percentile_slowest_path_latency_micro_s = path_latencies[one_percentile]
        else:
            avg_path_latency = 0
            one_percentile_slowest_path_latency_micro_s = 0

        return MetricsSummary(
            total_reports=full_time_total_raports,
            time_windows_seconds=self.time_window_seconds,
            window_metrics=Metrics(
                collected_reports=collected_reports,
                average_path_latency_micro_s=int(avg_path_latency),
                one_percentile_slowest_path_latency_micro_s=int(one_percentile_slowest_path_latency_micro_s),
                device_metrics=device_metrics_builder.build()
            ),
        )

