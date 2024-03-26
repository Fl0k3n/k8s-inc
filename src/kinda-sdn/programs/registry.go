package programs

import (
	"fmt"
	"sync"

	"github.com/Fl0k3n/k8s-inc/kinda-sdn/model"
)

// Thread safe
type P4ProgramRegistry interface {
	Register(details model.P4ProgramDetails, delegate P4Delegate)
	Lookup(programName string) (details model.P4ProgramDetails, ok bool)
	MustLookupDetails(programName string) model.P4ProgramDetails
	MustFindDelegate(programName string) P4Delegate
}

type registeredProgram struct {
	details model.P4ProgramDetails
	delegate P4Delegate
}

type InMemoryP4ProgramRegistry struct {
	registeredPrograms sync.Map // programName: string -> details: *registeredProgram
}

func NewRegistry() P4ProgramRegistry {
	return &InMemoryP4ProgramRegistry{
		registeredPrograms: sync.Map{},
	}
}

func (r *InMemoryP4ProgramRegistry) Register(details model.P4ProgramDetails, delegate P4Delegate) {
	registeredProgram := &registeredProgram{
		details: details,
		delegate: delegate,
	}
	r.registeredPrograms.Store(details.Name, registeredProgram)
}

func (r *InMemoryP4ProgramRegistry) Lookup(programName string) (model.P4ProgramDetails, bool) {
	res, ok := r.registeredPrograms.Load(programName)
	if !ok {
		return model.P4ProgramDetails{}, false
	}
	return res.(*registeredProgram).details, true
}

func (r *InMemoryP4ProgramRegistry) MustLookupDetails(programName string) model.P4ProgramDetails{
	if details, ok := r.Lookup(programName); ok {
		return details
	}
	panic(fmt.Errorf("couldn't find details for program: %s", programName))
}

func (r *InMemoryP4ProgramRegistry) MustFindDelegate(programName string) P4Delegate {
	if res, ok := r.registeredPrograms.Load(programName); ok {
		return res.(*registeredProgram).delegate
	}
	panic(fmt.Errorf("couldn't find delegate for program: %s", programName))
}
