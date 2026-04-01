package modelgen

// GoType returns the Go type string for the given FieldType.
func GoType(ft FieldType) string {
	switch ft {
	case FieldString, FieldText:
		return "string"
	case FieldInt:
		return "int"
	case FieldInt64:
		return "int64"
	case FieldFloat:
		return "float64"
	case FieldBool:
		return "bool"
	case FieldTime:
		return "time.Time"
	case FieldUUID:
		return "uuid.UUID"
	case FieldBytes:
		return "[]byte"
	case FieldJSON:
		return "json.RawMessage"
	case FieldReferences:
		return "uint64"
	default:
		return "any"
	}
}
