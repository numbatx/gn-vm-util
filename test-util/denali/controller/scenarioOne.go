package denalicontroller

import (
	"io/ioutil"
	"os"
	"path/filepath"

	mj "github.com/numbatx/gn-vm-util/test-util/denali/json/model"
	mjwrite "github.com/numbatx/gn-vm-util/test-util/denali/json/write"
)

// RunSingleJSONScenario parses and prepares test, then calls testCallback.
func (r *ScenarioRunner) RunSingleJSONScenario(contextPath string) error {
	var err error
	contextPath, err = filepath.Abs(contextPath)
	if err != nil {
		return err
	}

	// Open our jsonFile
	var jsonFile *os.File
	jsonFile, err = os.Open(contextPath)
	// if we os.Open returns an error then handle it
	if err != nil {
		return err
	}

	// defer the closing of our jsonFile so that we can parse it later on
	defer jsonFile.Close()

	byteValue, err := ioutil.ReadAll(jsonFile)
	if err != nil {
		return err
	}

	r.Parser.ValueInterpreter.FileResolver.SetContext(contextPath)
	scenario, parseErr := r.Parser.ParseScenarioFile(byteValue)
	if parseErr != nil {
		return parseErr
	}

	return r.Executor.ExecuteScenario(scenario, r.Parser.ValueInterpreter.FileResolver)
}

// tool to modify scenarios
// use with extreme caution
func saveModifiedScenario(toPath string, scenario *mj.Scenario) {
	resultJSON := mjwrite.ScenarioToJSONString(scenario)

	err := os.MkdirAll(filepath.Dir(toPath), os.ModePerm)
	if err != nil {
		panic(err)
	}
	err = ioutil.WriteFile(toPath, []byte(resultJSON), 0644)
	if err != nil {
		panic(err)
	}
}
