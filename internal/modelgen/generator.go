package modelgen

import "fmt"

// ModelGenerator is implemented by ORM plugins to generate schema files
// from a ModelDef.
type ModelGenerator interface {
	// Name returns the generator name (e.g. "ent", "gorm").
	Name() string
	// Generate produces ORM-specific schema files for the given model.
	// outDir is the base output directory (e.g. "models"); the generator
	// writes to {outDir}/{generatorName}/{snake_name}.go.
	Generate(def *ModelDef, outDir string) error
}

var generators = map[string]ModelGenerator{}

// RegisterGenerator adds a ModelGenerator to the registry.
func RegisterGenerator(g ModelGenerator) {
	generators[g.Name()] = g
}

// GetGenerator returns the ModelGenerator for the given name.
func GetGenerator(name string) (ModelGenerator, error) {
	g, ok := generators[name]
	if !ok {
		valid := make([]string, 0, len(generators))
		for k := range generators {
			valid = append(valid, k)
		}
		return nil, fmt.Errorf("unknown model generator %q; available: %v", name, valid)
	}
	return g, nil
}
