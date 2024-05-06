package denalijsonparse

import (
	fr "github.com/numbatx/gn-vm-util/test-util/denali/json/fileresolver"
	vi "github.com/numbatx/gn-vm-util/test-util/denali/json/valueinterpreter"
)

// Parser performs parsing of both json tests (older) and scenarios (new).
type Parser struct {
	ValueInterpreter vi.ValueInterpreter
}

// NewParser provides a new Parser instance.
func NewParser(fileResolver fr.FileResolver) Parser {
	return Parser{
		ValueInterpreter: vi.ValueInterpreter{
			FileResolver: fileResolver,
		},
	}
}
