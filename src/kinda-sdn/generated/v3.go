package generated

import (
	"github.com/Fl0k3n/k8s-inc/kinda-sdn/model"
	t "github.com/Fl0k3n/k8s-inc/kinda-sdn/telemetry"
	"github.com/Fl0k3n/k8s-inc/libs/p4-connector/connector"
)

func V3_grpc_topo() *model.Topology {
	r1 := model.NewBmv2IncSwitch(
		"r1", 
		[]*model.Link{
			model.NewLink("r2", "00:00:0a:00:00:05", "10.10.1.1", 24),
			model.NewLink("w1", "00:00:0a:00:00:06", "10.10.0.1", 24),
		},
		"127.0.0.1:9560",
	)

	r2 := model.NewBmv2IncSwitch(
		"r2",
		[]*model.Link{
			model.NewLink("r1", "00:00:0a:00:00:07", "10.10.1.2", 24),
			model.NewLink("r3", "00:00:0a:00:00:08", "10.10.3.1", 24),
			model.NewLink("w2", "00:00:0a:00:00:09", "10.10.2.1", 24),
		},
		"127.0.0.1:9561",
	)

	r3 := model.NewBmv2IncSwitch(
		"r3",
		[]*model.Link{
			model.NewLink("r2", "00:00:0a:00:00:0a", "10.10.3.2", 24),
			model.NewLink("w3", "00:00:0a:00:00:0b", "10.10.4.1", 24),
			model.NewLink("c1", "00:00:0a:00:00:0c", "10.10.5.1", 24),
		},
		"127.0.0.1:9562",
	)

	w1 := &model.Host{
		BaseDevice: model.BaseDevice{
			Name: "w1",
			Links: []*model.Link{
				model.NewLink("r1", "00:00:0a:00:00:01", "10.10.0.2", 24),
			},
		},	
	}

	w2 := &model.Host{
		BaseDevice: model.BaseDevice{
			Name: "w2",
			Links: []*model.Link{
				model.NewLink("r2", "00:00:0a:00:00:02", "10.10.2.2", 24),
			},
		},	
	}

	w3 := &model.Host{
		BaseDevice: model.BaseDevice{
			Name: "w3",
			Links: []*model.Link{
				model.NewLink("r3", "00:00:0a:00:00:03", "10.10.4.2", 24),
			},
		},	
	}

	c1 := &model.Host{
		BaseDevice: model.BaseDevice{
			Name: "c1",
			Links: []*model.Link{
				model.NewLink("r3", "00:00:0a:00:00:04", "10.10.5.2", 24),
			},
		},	
	}

	return &model.Topology{
		Devices:[]model.Device{
			r1, r2, r3, w1, w2, w3, c1,
		},
	}
}

func V3_grpc_p4_conf_raw(addTelemetryEntries bool) map[model.DeviceName][]connector.RawTableEntry {
	res := map[model.DeviceName][]connector.RawTableEntry {
		"r1": {
			t.Forward("10.10.0.2/32", "00:00:0a:00:00:06", "00:00:0a:00:00:01", "2"),
			t.Forward("10.10.2.0/24", "00:00:0a:00:00:05", "00:00:0a:00:00:07", "1"),
			t.Forward("10.10.3.0/24", "00:00:0a:00:00:05", "00:00:0a:00:00:07", "1"),
			t.Forward("10.10.4.0/24", "00:00:0a:00:00:05", "00:00:0a:00:00:07", "1"),
			t.Forward("10.10.5.0/24", "00:00:0a:00:00:05", "00:00:0a:00:00:07", "1"),
			t.Arp("10.10.0.1", "00:00:0a:00:00:06"),
		},
		"r2": {
			t.Forward("10.10.2.2/32", "00:00:0a:00:00:09", "00:00:0a:00:00:02", "3"),
			t.Forward("10.10.0.0/24", "00:00:0a:00:00:07", "00:00:0a:00:00:05", "1"),
			t.Forward("10.10.3.0/24", "00:00:0a:00:00:08", "00:00:0a:00:00:0a", "2"),
			t.Forward("10.10.4.0/24", "00:00:0a:00:00:08", "00:00:0a:00:00:0a", "2"),
			t.Forward("10.10.5.0/24", "00:00:0a:00:00:08", "00:00:0a:00:00:0a", "2"),
			t.Arp("10.10.2.1", "00:00:0a:00:00:09"),
		},
		"r3": {
			t.Forward("10.10.4.2/32", "00:00:0a:00:00:0b", "00:00:0a:00:00:03", "2"),
			t.Forward("10.10.5.2/32", "00:00:0a:00:00:0c", "00:00:0a:00:00:04", "3"),
			t.Forward("10.10.0.0/24", "00:00:0a:00:00:0a", "00:00:0a:00:00:08", "1"),
			t.Forward("10.10.1.0/24", "00:00:0a:00:00:0a", "00:00:0a:00:00:08", "1"),
			t.Forward("10.10.2.0/24", "00:00:0a:00:00:0a", "00:00:0a:00:00:08", "1"),
			t.Arp("10.10.4.1", "00:00:0a:00:00:0b"),
			t.Arp("10.10.5.1", "00:00:0a:00:00:0c"),
		},
	}
	
	if addTelemetryEntries {
		res["r1"] = append(res["r1"], []connector.RawTableEntry{
			t.ActivateSource(2),
			t.ConfigureSource(
				"10.10.0.2&&&0xFFFFFFFF", "10.10.4.2&&&0xFFFFFFFF", 0x11FF, 0x22FF, 4, 10, 8, 0xFF00,
			),
			t.Transit(1, 1500),
		}...)
		res["r2"] = append(res["r2"], []connector.RawTableEntry{
			t.Transit(2, 1500),
		}...)

		res["r3"] = append(res["r3"], []connector.RawTableEntry{
			t.ConfigureSink(2, 1),
			t.Reporting("00:00:0a:00:00:0a", "10.10.3.2", "00:00:0a:00:00:08", "10.10.2.2", 6000),
			t.Transit(3, 1500),
		}...)
	}

	return res
}

func V3_telemetry_artifact_paths() (binPath string, p4infoPath string) {
	binPath = "/home/flok3n/develop/virtual/telemetry2/int-platforms/p4src/int_v4.0/int4.json"
	p4infoPath = "/home/flok3n/develop/virtual/telemetry2/int-platforms/p4src/int_v4.0/int4.txt"
	return
}
