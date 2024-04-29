package denalicontroller

import mjparse "github.com/numbatx/gn-vm-util/test-util/denali/json/parse"

// NewDefaultFileResolver yields a new DefaultFileResolver instance.
// Reexported here to avoid having all external packages importing the parser.
// DefaultFileResolver is in parse for local tests only.
func NewDefaultFileResolver() *mjparse.DefaultFileResolver {
	return mjparse.NewDefaultFileResolver()
}
