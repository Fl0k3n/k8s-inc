package generated

import (
	"fmt"
	"math/rand"

	"github.com/Fl0k3n/k8s-inc/kinda-sdn/model"
)

// fake topology (without emulated/real devices) used in the work to evaluate reconciliation performance
// we use fat tree

func programs() []*model.P4ProgramDetails {
	forward := &model.P4ProgramDetails{
        Name: "forward",
        ImplementedInterfaces: []string{
            
        },
        Artifacts: []model.P4Artifacts{
            {
                Arch: "bmv2",
                P4InfoPath: "/home/flok3n/develop/virtual/telemetry2/int-platforms/p4src/int_v6.0/int.txt",
                P4PipelinePath: "/home/flok3n/develop/virtual/telemetry2/int-platforms/p4src/int_v6.0/int.json",
            }, 
        },
    }
    telemetry := &model.P4ProgramDetails{
        Name: "telemetry",
        ImplementedInterfaces: []string{
            "inc.kntp.com/v1alpha1/telemetry",
        },
        Artifacts: []model.P4Artifacts{
            {
                Arch: "bmv2",
                P4InfoPath: "/home/flok3n/develop/virtual/telemetry2/int-platforms/p4src/int_v6.0/int.txt",
                P4PipelinePath: "/home/flok3n/develop/virtual/telemetry2/int-platforms/p4src/int_v6.0/int.json",
            }, 
        },
    }
	return []*model.P4ProgramDetails{forward, telemetry}
}

func makeNetDevice(
	name string,
	rng *rand.Rand,
	incSwitchProbability float32,
) model.Device {
	// for testing make sure that s-0-0-1 (aggregation switch in first pod, above first rack) is incSwitch
	// and s-0-1-1 (second rack) isn't incSwitch
	isIncSwitch := rng.Float32() < incSwitchProbability || (name == "s-0-0-1" && incSwitchProbability > 0)
	if isIncSwitch && name == "s-0-1-1" {
		isIncSwitch = false
	}
	if isIncSwitch {
		return model.NewBmv2IncSwitch(
			name, 
			[]*model.Link{},
			"",
			"forward",
		)
	}
	return &model.NetDevice{
		BaseDevice: model.BaseDevice{
			Name: name,
		},
	}
}

func makeNode(name string) model.Device {
	return &model.Host{
        BaseDevice: model.BaseDevice{
            Name: name,
            Links: []*model.Link{},
        },	
    }
}

func addLink(dev1 model.Device, dev2 model.Device) {
	dev1.AddLink(model.NewLink(dev2.GetName(), "12:34:56:78:9a:bc", "123.123.123.123", 24))
	dev2.AddLink(model.NewLink(dev1.GetName(), "12:34:56:78:9a:bc", "123.123.123.123", 24))
}

// https://www.cs.cornell.edu/courses/cs5413/2014fa/lectures/08-fattree.pdf page 12
func FatTreeFakeCluster(k int, incSwitchFraction float32) (*model.Topology, []*model.P4ProgramDetails)  {
	rng := rand.New(rand.NewSource(42))
	linkCount := 0
	
	devices := []model.Device{}
	coreSwitches := []model.Device{}
	for i := 0; i < (k / 2) * (k / 2); i ++ {
		name := fmt.Sprintf("s-core-%d", i)
		dev := makeNetDevice(name, rng, incSwitchFraction)
		coreSwitches = append(coreSwitches, dev)
		devices = append(devices, dev)
	}
	for pod := 0; pod < k; pod++ {
		edgeSwitches := []model.Device{}
		aggSwitches := []model.Device{}
		for rack := 0; rack < k / 2; rack++ {
			aggDev := makeNetDevice(fmt.Sprintf("s-%d-%d-1", pod, rack), rng, incSwitchFraction)
			devices = append(devices, aggDev)
			aggSwitches = append(aggSwitches, aggDev)

			edgeDev := makeNetDevice(fmt.Sprintf("s-%d-%d-0", pod, rack), rng, incSwitchFraction)
			devices = append(devices, edgeDev)
			edgeSwitches = append(edgeSwitches, edgeDev)

			for i := 0; i < k / 2; i++ {
				name := fmt.Sprintf("n-%d-%d-%d", pod, rack, i)
				node := makeNode(name)
				devices = append(devices, node)
				addLink(node, edgeDev)
				linkCount++
			}
		}
		for _, aggSwitch := range aggSwitches {
			for _, edgeSwitch := range edgeSwitches {
				addLink(aggSwitch, edgeSwitch)
				linkCount++
			}
		}
		coreSwitchCounter := 0
		for _, aggSwitch := range aggSwitches {
			for j := 0; j < k / 2; j++ {
				coreSwitch := coreSwitches[coreSwitchCounter]
				addLink(aggSwitch, coreSwitch)
				linkCount++
				coreSwitchCounter++
			}
		}
	}
	return &model.Topology{Devices: devices}, programs()
}

