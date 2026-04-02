package schema

import "fmt"

// ORMGenerator is implemented by ORM plugins to generate schema files
// from a ModelDefinition.
type ORMGenerator interface {
	// Name returns the generator name (e.g. "ent", "gorm").
	Name() string
	// Generate produces ORM-specific schema files for the given model.
	// outDir is the base output directory (e.g. "models"); the generator
	// writes to {outDir}/{generatorName}/{snake_name}.go.
	Generate(def *ModelDefinition, outDir string) error
}

var ormGenerators = map[string]ORMGenerator{}

// RegisterORMGenerator adds an ORMGenerator to the registry.
func RegisterORMGenerator(g ORMGenerator) {
	ormGenerators[g.Name()] = g
}

// GetORMGenerator returns the ORMGenerator for the given name.
func GetORMGenerator(name string) (ORMGenerator, error) {
	g, ok := ormGenerators[name]
	if !ok {
		valid := make([]string, 0, len(ormGenerators))
		for k := range ormGenerators {
			valid = append(valid, k)
		}
		return nil, fmt.Errorf("unknown model generator %q; available: %v", name, valid)
	}
	return g, nil
}
