// Package schema provides shared model definition types used by the model
// generator and its ORM plugins (ent, gorm).
package schema

// FieldType represents the CLI-level type abstraction for model fields.
type FieldType string

const (
	FieldString     FieldType = "string"
	FieldInt        FieldType = "int"
	FieldInt64      FieldType = "int64"
	FieldFloat      FieldType = "float"
	FieldBool       FieldType = "bool"
	FieldTime       FieldType = "time"
	FieldUUID       FieldType = "uuid"
	FieldText       FieldType = "text"
	FieldBytes      FieldType = "bytes"
	FieldJSON       FieldType = "json"
	FieldReferences FieldType = "references"
)

// ValidFieldTypes is the set of recognised CLI type names.
var ValidFieldTypes = map[FieldType]bool{
	FieldString:     true,
	FieldInt:        true,
	FieldInt64:      true,
	FieldFloat:      true,
	FieldBool:       true,
	FieldTime:       true,
	FieldUUID:       true,
	FieldText:       true,
	FieldBytes:      true,
	FieldJSON:       true,
	FieldReferences: true,
}

// Field represents a single model field parsed from CLI arguments.
type Field struct {
	Name        string
	Type        FieldType
	Optional    bool      // trailing "?" on type, e.g. age:int?
	Index       bool      // :index modifier
	Unique      bool      // :uniq modifier
	Polymorphic bool      // {polymorphic} option (only valid on references)
	RefModel    string    // explicit target model for references, e.g. references{StorageBlob} (empty = derive from Name)
	Limit       int       // {N} option for sized types, e.g. string{255} → varchar(255). 0 means default.
	RefIDType   FieldType // resolved ID type of referenced model (set by ValidateReferences, not parsing)
}

// ReferencedModel returns the PascalCase name of the model this references field
// points to. If RefModel is set explicitly, it is returned as-is. Otherwise,
// PascalCase(Name) is used.
func (f *Field) ReferencedModel() string {
	if f.RefModel != "" {
		return f.RefModel
	}
	return PascalCase(f.Name)
}

// ModelDefinition is the parsed model definition from CLI arguments.
type ModelDefinition struct {
	Name   string    // PascalCase model name, e.g. "Article"
	IDType FieldType // empty = default auto-increment PK, FieldUUID = string UUID PK
	Fields []Field
}

// HasUUIDPrimaryKey returns true if the model uses a UUID primary key.
func (m *ModelDefinition) HasUUIDPrimaryKey() bool {
	return m.IDType == FieldUUID
}

// HasReferences returns true if the model has any non-polymorphic references fields.
func (m *ModelDefinition) HasReferences() bool {
	for _, f := range m.Fields {
		if f.Type == FieldReferences && !f.Polymorphic {
			return true
		}
	}
	return false
}

// HasIndexes returns true if any field has an index or unique modifier.
func (m *ModelDefinition) HasIndexes() bool {
	for _, f := range m.Fields {
		if f.Index || f.Unique {
			return true
		}
	}
	return false
}
