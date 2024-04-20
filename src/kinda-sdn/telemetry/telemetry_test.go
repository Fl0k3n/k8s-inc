package telemetry

import "testing"



func TestTelemetryEntityDifferenceCalculation(t *testing.T) {
	
	empty := func () *TelemetryEntities {
		return makeEntitiesBuilder().build()
	}

	sampleConfigs := map[string]TelemetrySourceConfig{
		"s1": {
			ips: TernaryIpPair{src: "x", dst: "y"},
			srcPort: 1,
			dstPort: 2,
			tunneled: false,
		},
		"s2": {
			ips: TernaryIpPair{src: "x", dst: "y"},
			srcPort: 1,
			dstPort: 2,
			tunneled: false,
		},
		"s3": {
			ips: TernaryIpPair{src: "x", dst: "y"},
			srcPort: 5,
			dstPort: 2,
			tunneled: false,
		},
		"s4": {
			ips: TernaryIpPair{src: "a", dst: "y"},
			srcPort: 5,
			dstPort: 8,
			tunneled: false,
		},
	}

	makeSource := func (u string, v string, configs ...string) sourceWithConfig {
		cfgs := []TelemetrySourceConfig{}
		for _, c := range configs {
			cfgs = append(cfgs, sampleConfigs[c])
		}
		return source(Edge{u, v}, cfgs...)
	}

	equals := func(e1 *TelemetryEntities, e2 *TelemetryEntities) bool {
		if len(e1.Sinks) != len(e2.Sinks) ||
		   len(e1.Transits) != len(e2.Transits) ||
		   len(e1.Sources) != len(e2.Sources) {
			return false
		}
		for s1 := range e1.Sinks {
			if _, ok := e2.Sinks[s1]; !ok {
				return false
			}
		}
		for t1 := range e1.Transits {
			if _, ok := e2.Transits[t1]; !ok {
				return false
			}
		}
		for s1, s1c := range e1.Sources {
			if s2c, ok := e2.Sources[s1]; !ok {
				return false
			} else {
				if len(s1c) != len(s2c) {
					return false
				}
				for _, c1 := range s1c {
					found := false
					for _, c2 := range s2c {
						if c1 == c2 {
							found = true
							break
						}
					}
					if !found {
						return false
					}
				}
			}
		}
		return true
	}

	sampleEntities1 := makeEntitiesBuilder().
		sources(makeSource("a", "b", "s1", "s2", "s3"), makeSource("x", "y", "s3")).
		transits("b", "y", "v", "g1", "g3").
		sinks(Edge{"g1", "g2"}, Edge{"g4", "g5"}).build()

	sampleEntities2 := makeEntitiesBuilder().
		sources(makeSource("a", "b", "s1", "s2"), makeSource("q1", "q2", "s4")).
		transits("b", "y", "g6").
		sinks(Edge{"g1", "g2"}, Edge{"g3", "g4"}).build()
	
	inS1ButNotInS2 := makeEntitiesBuilder().
		sources(makeSource("a", "b", "s3"), makeSource("x", "y", "s3")).
		transits("v", "g1", "g3").
		sinks(Edge{"g4", "g5"}).build()

	inS2ButNotInS1 := makeEntitiesBuilder().
		sources(makeSource("q1", "q2", "s4")).
		transits("g6").
		sinks(Edge{"g3", "g4"}).build()

	tests := []struct{
		name string
		oldEntities *TelemetryEntities
		newEntities *TelemetryEntities
		expectedToAdd *TelemetryEntities
		expectedToRemove *TelemetryEntities
	}{
		{
			name: "empty new and empty old doesn't yield anything to add or to remove",
			oldEntities: empty(),
			newEntities: empty(),
			expectedToAdd: empty(),
			expectedToRemove: empty(),
		},
		{
			name: "empty old yield all new as added and nothing to remove",
			oldEntities: empty(),
			newEntities: sampleEntities1,
			expectedToAdd: sampleEntities1,
			expectedToRemove: empty(),
		},
		{
			name: "empty new yield all old as removed and nothing to add",
			oldEntities: sampleEntities1,
			newEntities: empty(),
			expectedToAdd: empty(),
			expectedToRemove: sampleEntities1,
		},
		{
			name: "difference is properly computed",
			oldEntities: sampleEntities1,
			newEntities: sampleEntities2,
			expectedToAdd: inS2ButNotInS1,
			expectedToRemove: inS1ButNotInS2,
		},
		{
			name: "difference is properly computed",
			oldEntities: sampleEntities2,
			newEntities: sampleEntities1,
			expectedToAdd: inS1ButNotInS2,
			expectedToRemove: inS2ButNotInS1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			added, removed := computeDifferences(tt.oldEntities, tt.newEntities)
			if !equals(added, tt.expectedToAdd) {
				t.Error("entries to add don't match expected ones")
			}
			if !equals(removed, tt.expectedToRemove) {
				t.Error("entries to remove don't match expected ones")
			}
		})
	}
}

type entitiesBuilder struct {
	entities *TelemetryEntities
}

func makeEntitiesBuilder() *entitiesBuilder {
	return &entitiesBuilder{
		entities: newTelemetryEntities(),
	}
}

type sourceWithConfig struct {
	edge Edge
	configs []TelemetrySourceConfig
}

func source(edge Edge, configs ...TelemetrySourceConfig) sourceWithConfig {
	return sourceWithConfig{edge: edge, configs: configs}
}

func (b *entitiesBuilder) sources(sourcesWithConfigs ...sourceWithConfig) *entitiesBuilder {
	for _, s := range sourcesWithConfigs {
		b.entities.Sources[s.edge] = s.configs
	}
	return b
}

func (b *entitiesBuilder) transits(names ...string) *entitiesBuilder {
	for _, n := range names {
		b.entities.Transits[n] = struct{}{}
	}
	return b
}

func (b *entitiesBuilder) sinks(edges ...Edge) *entitiesBuilder {
	for _, e := range edges {
		b.entities.Sinks[e] = struct{}{}
	}
	return b
}

func (b *entitiesBuilder) build() *TelemetryEntities {
	return b.entities
}
