// Code generated by kinda-p4. DO NOT EDIT.

package generated

import "github.com/Fl0k3n/k8s-inc/kinda-sdn/model"

func Measure_gRPC_topo() *model.Topology {
    dev0 := model.NewBmv2IncSwitch(
        "r1", 
        []*model.Link{
            model.NewLink("r2", "00:00:0a:00:00:00", "10.0.0.1", 24),
            model.NewLink("tree-worker2", "00:00:0a:00:00:01", "10.0.2.1", 24),
            model.NewLink("tree-worker", "00:00:0a:00:00:02", "10.0.3.1", 24),
        },
        "127.0.0.1:9560",
    )
    dev1 := model.NewBmv2IncSwitch(
        "r2", 
        []*model.Link{
            model.NewLink("r1", "00:00:0a:00:00:03", "10.0.0.2", 24),
            model.NewLink("r3", "00:00:0a:00:00:04", "10.0.1.1", 24),
        },
        "127.0.0.1:9561",
    )
    dev2 := &model.Host{
        BaseDevice: model.BaseDevice{
            Name: "tree-worker2",
            Links: []*model.Link{
                model.NewLink("r1", "00:00:0a:00:00:05", "10.0.2.2", 24),
            },
        },	
    }
    dev3 := &model.Host{
        BaseDevice: model.BaseDevice{
            Name: "tree-worker",
            Links: []*model.Link{
                model.NewLink("r1", "00:00:0a:00:00:06", "10.0.3.2", 24),
            },
        },	
    }
    dev4 := model.NewBmv2IncSwitch(
        "r3", 
        []*model.Link{
            model.NewLink("r2", "00:00:0a:00:00:07", "10.0.1.2", 24),
            model.NewLink("tree-worker5", "00:00:0a:00:00:08", "10.0.4.1", 24),
            model.NewLink("tree-worker4", "00:00:0a:00:00:09", "10.0.5.1", 24),
            model.NewLink("tree-worker3", "00:00:0a:00:00:0a", "10.0.6.1", 24),
            model.NewLink("tree-control-plane", "00:00:0a:00:00:0b", "10.0.7.1", 24),
        },
        "127.0.0.1:9562",
    )
    dev5 := &model.Host{
        BaseDevice: model.BaseDevice{
            Name: "tree-worker5",
            Links: []*model.Link{
                model.NewLink("r3", "00:00:0a:00:00:0c", "10.0.4.2", 24),
            },
        },	
    }
    dev6 := &model.Host{
        BaseDevice: model.BaseDevice{
            Name: "tree-worker4",
            Links: []*model.Link{
                model.NewLink("r3", "00:00:0a:00:00:0d", "10.0.5.2", 24),
            },
        },	
    }
    dev7 := &model.Host{
        BaseDevice: model.BaseDevice{
            Name: "tree-worker3",
            Links: []*model.Link{
                model.NewLink("r3", "00:00:0a:00:00:0e", "10.0.6.2", 24),
            },
        },	
    }
    dev8 := &model.Host{
        BaseDevice: model.BaseDevice{
            Name: "tree-control-plane",
            Links: []*model.Link{
                model.NewLink("r3", "00:00:0a:00:00:0f", "10.0.7.2", 24),
            },
        },	
    }
    return &model.Topology{
        Devices: []model.Device{
            dev0, dev1, dev2, dev3, dev4, dev5, dev6, dev7, dev8,
        },
    }
}
