package model

import (
	"fmt"
	"os"

	"github.com/polagonow/pola/internal/generators/model/schema"
)

// ValidateReferences checks that every non-polymorphic references field points
// at a model that already exists in the canonical models directory, and resolves
// each foreign key's ID type from the referenced model's primary key. It reads
// the single source of truth (the neutral db/models structs) rather than any
// per-ORM file, and mutates def.Fields[i].RefIDType.
func ValidateReferences(def *schema.ModelDefinition, modelsDir string) error {
	siblings, err := schema.ParseModelsDir(modelsDir)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("read models dir %s: %w", modelsDir, err)
	}

	for i := range def.Fields {
		f := &def.Fields[i]
		if f.Type != schema.FieldReferences {
			continue
		}

		// Polymorphic targets resolve at runtime; StorageBlob is a framework-owned
		// model outside db/models. Both use the default auto-increment PK type
		// (RefIDType empty → uint foreign key).
		if f.Polymorphic {
			f.RefIDType = ""
			continue
		}
		refName := f.ReferencedModel()
		if refName == "StorageBlob" {
			f.RefIDType = ""
			continue
		}

		ref, ok := siblings[refName]
		if !ok {
			return fmt.Errorf(
				"model %q not found in %s; generate it first with:\n  pola generate model %s ...",
				refName, modelsDir, refName,
			)
		}
		f.RefIDType = ref.IDType // FieldUUID for string keys, empty for auto-increment
	}
	return nil
}
