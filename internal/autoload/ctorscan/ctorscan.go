// Package ctorscan parses Go constructor function declarations and returns
// each parameter's type in a form the autoload code generators can consume:
// the type-expression string to emit into overlay code, and — when a package
// qualifier is used — the alias-aware import path.
//
// The *core.Registry parameter is recognized (matched by resolved import
// path, not by the syntactic qualifier, so aliased core imports work) and
// flagged; other params are opaque to ctorscan and left for the generator
// to feed into core.Invoke[T].
package ctorscan

import (
	"fmt"
	"go/ast"
	"sort"
	"strings"
)

// Param is one constructor parameter, resolved for code generation.
type Param struct {
	// IsRegistry is true when the parameter is *<polaPackage>/core.Registry
	// (regardless of the import alias used in the source file).
	IsRegistry bool
	// Type is the type expression as it must appear in the generated file
	// (e.g. "*gorm.DB", "repositories.TodoRepository"). For registry params
	// the field is still populated for diagnostics but generators use
	// IsRegistry instead of emitting an Invoke call.
	Type string
	// Import describes the package qualifier used by Type. Nil for
	// same-package types, builtins, and registry params.
	Import *Import
}

// Import identifies a package that must be imported by the generated file.
type Import struct {
	// Alias is emitted before the import path when non-empty. Callers merging
	// imports across constructors may rewrite this to disambiguate collisions;
	// the associated Param.Type must be re-rendered to use the same alias.
	Alias string
	Path  string
}

// ScanParams extracts fd's parameters in declaration order, resolving package
// qualifiers through file.Imports. polaPackage is the framework module path
// (e.g. "github.com/polagonow/pola").
//
// Constructor signatures must be exactly `NewX()` or `NewX(r *core.Registry)`.
// Any other parameter type (e.g. `db *gorm.DB`, `repo repositories.FooRepository`)
// returns an error — dependencies must be resolved from the DI registry via
// core.MustInvoke inside the constructor body, not passed positionally.
//
// The function returns a descriptive error naming file, constructor, and
// parameter index when a param uses an unsupported syntactic shape or a
// dot-imported package.
func ScanParams(fd *ast.FuncDecl, file *ast.File, filename, polaPackage string) ([]Param, error) {
	if fd == nil || fd.Type == nil || fd.Type.Params == nil {
		return nil, nil
	}

	aliasToPath, hasDot := buildImportTable(file)

	corePath := polaPackage + "/core"

	var params []Param
	index := 0
	for _, field := range fd.Type.Params.List {
		names := len(field.Names)
		if names == 0 {
			names = 1
		}
		for i := 0; i < names; i++ {
			p, err := resolveParam(field.Type, aliasToPath, hasDot, corePath)
			if err != nil {
				return nil, fmt.Errorf("%s: %s parameter %d: %w", filename, fd.Name.Name, index+1, err)
			}
			if !p.IsRegistry {
				return nil, fmt.Errorf("%s: %s parameter %d has type %s; constructors must take (r *core.Registry) or no parameters — resolve dependencies from the registry via core.MustInvoke inside the constructor body", filename, fd.Name.Name, index+1, p.Type)
			}
			params = append(params, p)
			index++
		}
	}
	return params, nil
}

func buildImportTable(file *ast.File) (map[string]string, bool) {
	table := map[string]string{}
	hasDot := false
	if file == nil {
		return table, false
	}
	for _, spec := range file.Imports {
		if spec == nil || spec.Path == nil {
			continue
		}
		path := strings.Trim(spec.Path.Value, `"`)
		if spec.Name != nil {
			switch spec.Name.Name {
			case ".":
				hasDot = true
				continue
			case "_":
				continue
			default:
				table[spec.Name.Name] = path
				continue
			}
		}
		table[lastSegment(path)] = path
	}
	return table, hasDot
}

func lastSegment(path string) string {
	if i := strings.LastIndex(path, "/"); i >= 0 {
		return path[i+1:]
	}
	return path
}

func resolveParam(expr ast.Expr, aliasToPath map[string]string, hasDot bool, corePath string) (Param, error) {
	switch t := expr.(type) {
	case *ast.Ident:
		if hasDot {
			return Param{}, fmt.Errorf("unresolvable identifier %q with dot import in file; add explicit qualifier", t.Name)
		}
		return Param{Type: t.Name}, nil
	case *ast.SelectorExpr:
		return resolveSelector(t, aliasToPath, false, corePath)
	case *ast.StarExpr:
		switch inner := t.X.(type) {
		case *ast.Ident:
			if hasDot {
				return Param{}, fmt.Errorf("unresolvable identifier %q with dot import in file; add explicit qualifier", inner.Name)
			}
			return Param{Type: "*" + inner.Name}, nil
		case *ast.SelectorExpr:
			return resolveSelector(inner, aliasToPath, true, corePath)
		default:
			return Param{}, fmt.Errorf("unsupported constructor parameter type; parameters must be *core.Registry or named types resolvable from the DI registry")
		}
	default:
		return Param{}, fmt.Errorf("unsupported constructor parameter type; parameters must be *core.Registry or named types resolvable from the DI registry")
	}
}

func resolveSelector(sel *ast.SelectorExpr, aliasToPath map[string]string, pointer bool, corePath string) (Param, error) {
	qualifier, ok := sel.X.(*ast.Ident)
	if !ok {
		return Param{}, fmt.Errorf("unsupported constructor parameter type; qualifier must be a simple identifier")
	}
	path, ok := aliasToPath[qualifier.Name]
	if !ok {
		return Param{}, fmt.Errorf("unresolved package qualifier %q; ensure the import is present", qualifier.Name)
	}
	prefix := ""
	if pointer {
		prefix = "*"
	}
	typeStr := prefix + qualifier.Name + "." + sel.Sel.Name
	if path == corePath && sel.Sel.Name == "Registry" {
		if !pointer {
			return Param{}, fmt.Errorf("parameter is %s; did you mean *core.Registry", typeStr)
		}
		return Param{IsRegistry: true, Type: typeStr}, nil
	}
	imp := &Import{Alias: aliasFor(qualifier.Name, path), Path: path}
	return Param{Type: typeStr, Import: imp}, nil
}

// aliasFor returns an emit alias for the given qualifier and import path.
// When the qualifier equals the path's last segment, no alias is needed;
// otherwise the source-file qualifier is re-emitted so the type string
// matches (this is always safe because that qualifier compiled originally).
func aliasFor(qualifier, path string) string {
	if qualifier == lastSegment(path) {
		return ""
	}
	return qualifier
}

// MergeImports collapses a list of per-parameter imports into a
// deduplicated, sorted set safe to render inside a single generated file.
// When two paths would share an alias, later entries are renamed with a
// numeric suffix to keep them distinct; callers rendering param types
// consult the returned rename table to update qualifiers in the type
// strings.
//
// The skip set (typically containing "<polaPackage>/core") is stripped
// from the output — those imports are added by the generator's own
// template header and must not be duplicated.
func MergeImports(params []Param, skip map[string]struct{}) (imports []Import, renames map[string]string) {
	renames = map[string]string{}
	byPath := map[string]Import{}
	aliasToPath := map[string]string{}
	pathOrder := []string{}

	for _, p := range params {
		if p.Import == nil {
			continue
		}
		if _, drop := skip[p.Import.Path]; drop {
			continue
		}
		if existing, ok := byPath[p.Import.Path]; ok {
			if existing.Alias == "" && p.Import.Alias != "" {
				existing.Alias = p.Import.Alias
				byPath[p.Import.Path] = existing
				aliasToPath[existing.Alias] = p.Import.Path
			}
			continue
		}
		key := effectiveAlias(*p.Import)
		if other, clash := aliasToPath[key]; clash && other != p.Import.Path {
			renamed := disambiguate(key, aliasToPath)
			renames[key+"@"+p.Import.Path] = renamed
			byPath[p.Import.Path] = Import{Alias: renamed, Path: p.Import.Path}
			aliasToPath[renamed] = p.Import.Path
		} else {
			byPath[p.Import.Path] = *p.Import
			aliasToPath[key] = p.Import.Path
		}
		pathOrder = append(pathOrder, p.Import.Path)
	}

	sort.Strings(pathOrder)
	seen := map[string]struct{}{}
	for _, path := range pathOrder {
		if _, dup := seen[path]; dup {
			continue
		}
		seen[path] = struct{}{}
		imports = append(imports, byPath[path])
	}
	return imports, renames
}

func effectiveAlias(imp Import) string {
	if imp.Alias != "" {
		return imp.Alias
	}
	return lastSegment(imp.Path)
}

func disambiguate(base string, taken map[string]string) string {
	for i := 2; ; i++ {
		candidate := fmt.Sprintf("%s%d", base, i)
		if _, in := taken[candidate]; !in {
			return candidate
		}
	}
}
