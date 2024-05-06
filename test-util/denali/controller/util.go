package denalicontroller

import (
	fr "github.com/numbatx/gn-vm-util/test-util/denali/json/fileresolver"
)

// NewDefaultFileResolver yields a new DefaultFileResolver instance.
// Reexported here to avoid having all external packages importing the parser.
// DefaultFileResolver is in parse for local tests only.
func NewDefaultFileResolver() *fr.DefaultFileResolver {
	return fr.NewDefaultFileResolver()
}
