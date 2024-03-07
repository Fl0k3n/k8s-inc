package generated

import "github.com/Fl0k3n/k8s-inc/micro-sdn/model"

func V3_grpc_topo() *model.Topology {
	r1 := &model.IncSwitch{
		BaseDevice: model.BaseDevice{
			Name: "r1",
			Links: []*model.Link{
				model.NewLink("r2", "00:00:0a:00:00:05", "10.10.1.1", 24),
				model.NewLink("w1", "00:00:0a:00:00:06", "10.10.0.1", 24),
			},
		},
	}

	r2 := &model.IncSwitch{
		BaseDevice: model.BaseDevice{
			Name: "r2",
			Links: []*model.Link{
				model.NewLink("r1", "00:00:0a:00:00:07", "10.10.1.2", 24),
				model.NewLink("r3", "00:00:0a:00:00:08", "10.10.3.1", 24),
				model.NewLink("w2", "00:00:0a:00:00:09", "10.10.2.1", 24),
			},
		},
	}

	r3 := &model.IncSwitch{
		BaseDevice: model.BaseDevice{
			Name: "r3",
			Links: []*model.Link{
				model.NewLink("r2", "00:00:0a:00:00:0a", "10.10.3.2", 24),
				model.NewLink("w3", "00:00:0a:00:00:0b", "10.10.4.1", 24),
				model.NewLink("c1", "00:00:0a:00:00:0c", "10.10.5.1", 24),
			},
		},
	}

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

