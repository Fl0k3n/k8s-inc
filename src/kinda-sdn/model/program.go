package model

type P4Artifacts struct {
	Arch IncSwitchArch
	P4InfoPath string
	P4PipelinePath string
}

type P4ProgramDetails struct {
	Name string
	ImplementedInterfaces []string
	Artifacts []P4Artifacts
}

func NewProgramDetails(programName string, implementedInterfaces []string, artifacts []P4Artifacts) *P4ProgramDetails {
	return &P4ProgramDetails{
		Name: programName,
		ImplementedInterfaces: implementedInterfaces,
		Artifacts: artifacts,
	}
}

func (p *P4ProgramDetails) GetArtifactsFor(arch IncSwitchArch) (P4Artifacts, bool) {
	for _, artifact := range p.Artifacts {
		if artifact.Arch == arch {
			return artifact, true
		}
	}
	return P4Artifacts{}, false
}
