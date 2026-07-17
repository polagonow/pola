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

	FieldEmail        FieldType = "email"
	FieldURL          FieldType = "url"
	FieldIP           FieldType = "ip"
	FieldIPv4         FieldType = "ipv4"
	FieldIPv6         FieldType = "ipv6"
	FieldMAC          FieldType = "mac"
	FieldAlpha        FieldType = "alpha"
	FieldNumeric      FieldType = "numeric"
	FieldAlphanumeric FieldType = "alphanumeric"
	FieldHexColor     FieldType = "hexcolor"
	FieldCreditCard   FieldType = "creditcard"
	FieldBase64       FieldType = "base64"
	FieldLatitude     FieldType = "latitude"
	FieldLongitude    FieldType = "longitude"
	FieldPort         FieldType = "port"
	FieldSemver       FieldType = "semver"
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

	FieldEmail:        true,
	FieldURL:          true,
	FieldIP:           true,
	FieldIPv4:         true,
	FieldIPv6:         true,
	FieldMAC:          true,
	FieldAlpha:        true,
	FieldNumeric:      true,
	FieldAlphanumeric: true,
	FieldHexColor:     true,
	FieldCreditCard:   true,
	FieldBase64:       true,
	FieldLatitude:     true,
	FieldLongitude:    true,
	FieldPort:         true,
	FieldSemver:       true,
}

// ValidatorFieldTypes are field types that are stored as strings but carry
// a govalidator struct-tag validator for format checking.
var ValidatorFieldTypes = map[FieldType]bool{
	FieldEmail:        true,
	FieldURL:          true,
	FieldIP:           true,
	FieldIPv4:         true,
	FieldIPv6:         true,
	FieldMAC:          true,
	FieldAlpha:        true,
	FieldNumeric:      true,
	FieldAlphanumeric: true,
	FieldHexColor:     true,
	FieldCreditCard:   true,
	FieldBase64:       true,
	FieldLatitude:     true,
	FieldLongitude:    true,
	FieldPort:         true,
	FieldSemver:       true,
}

// Field represents a single model field parsed from CLI arguments.
type Field struct {
	Name        string
	Type        FieldType
	Optional    bool      // trailing "?" on type, e.g. age:int?
	Required    bool      // trailing "!" on type or :required modifier → validate:"required"
	Hidden      bool      // :hidden modifier → json:"-" (never serialized to the client)
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

// ModelDefinition is the parsed model definition. It is produced both from CLI
// arguments (ParseArgs) and by reading a canonical db/models struct back into
// this form (ParseModelFile) — the two must round-trip to the same value.
type ModelDefinition struct {
	Name          string    // PascalCase model name, e.g. "Article"
	IDType        FieldType // empty = default auto-increment PK, FieldUUID = string UUID PK
	Fields        []Field
	HasTimestamps bool // emit/according CreatedAt + UpdatedAt columns
	SoftDeletes   bool // emit/according a nullable DeletedAt column + soft-delete behavior
}

// HasUUIDPrimaryKey returns true if the model uses a UUID primary key.
func (m *ModelDefinition) HasUUIDPrimaryKey() bool {
	return m.IDType == FieldUUID
}

// IDGoType returns the Go type of the model's primary key: "string" for UUID
// keys, "uint" for the default auto-increment key.
func (m *ModelDefinition) IDGoType() string {
	if m.IDType == FieldUUID {
		return "string"
	}
	return "uint"
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

// HasBlobFields returns true if any field references StorageBlob (file upload).
func (m *ModelDefinition) HasBlobFields() bool {
	for _, f := range m.Fields {
		if f.Type == FieldReferences && f.RefModel == "StorageBlob" {
			return true
		}
	}
	return false
}

// BlobFields returns all fields that reference StorageBlob.
func (m *ModelDefinition) BlobFields() []Field {
	var result []Field
	for _, f := range m.Fields {
		if f.Type == FieldReferences && f.RefModel == "StorageBlob" {
			result = append(result, f)
		}
	}
	return result
}
