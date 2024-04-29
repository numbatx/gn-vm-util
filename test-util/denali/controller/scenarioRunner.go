package denalicontroller

import (
	mj "github.com/numbatx/gn-vm-util/test-util/denali/json/model"
	mjparse "github.com/numbatx/gn-vm-util/test-util/denali/json/parse"
)

// ScenarioExecutor describes a component that can run a VM scenario.
type ScenarioExecutor interface {
	// Reset clears state/world.
	Reset()

	// ExecuteScenario executes the scenario and checks if it passed. Failure is signaled by returning an error.
	// The FileResolver helps with resolving external steps.
	// TODO: group into a "execution context" param.
	ExecuteScenario(*mj.Scenario, mjparse.FileResolver) error
}

// ScenarioRunner is a component that can run json scenarios, using a provided executor.
type ScenarioRunner struct {
	Executor ScenarioExecutor
	Parser   mjparse.Parser
}

// NewScenarioRunner creates new ScenarioRunner instance.
func NewScenarioRunner(executor ScenarioExecutor, fileResolver mjparse.FileResolver) *ScenarioRunner {
	return &ScenarioRunner{
		Executor: executor,
		Parser: mjparse.Parser{
			FileResolver: fileResolver,
		},
	}
}
