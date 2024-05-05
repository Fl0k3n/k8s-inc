package utils

type PortMetrics struct {
    NumPackets                    int `json:"num_packets"`
    EgressPortID                  int `json:"egress_port_id"`
    AverageLatencyMicroS          int `json:"average_latency_micro_s"`
    AverageQueueFillState         int `json:"average_queue_fill_state"`
    OnePercentileSlowestLatency   int `json:"one_percentile_slowest_latency"`
    OnePercentileLargestQueueFill int `json:"one_percentile_largest_queue_fill_state"`
}

type SwitchMetrics struct {
    SwitchID     int            `json:"switch_id"`
    PortMetrics  []PortMetrics  `json:"port_metrics"`
}

type Metrics struct {
    CollectedReports                      int             `json:"collected_reports"`
    AveragePathLatencyMicroS              int             `json:"average_path_latency_micro_s"`
    OnePercentileSlowestPathLatencyMicroS int             `json:"one_percentile_slowest_path_latency_micro_s"`
    DeviceMetrics                         []SwitchMetrics `json:"device_metrics"`
}

type MetricsSummary struct {
    TotalReports       int     `json:"total_reports"`
    TimeWindowsSeconds int     `json:"time_windows_seconds"`
    WindowMetrics      Metrics `json:"window_metrics"`
}
