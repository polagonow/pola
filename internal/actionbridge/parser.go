package actionbridge

import (
	"fmt"
	"go/types"
	"strings"

	"golang.org/x/tools/go/packages"
)

// ActionDef describes a single action struct discovered in the actions/ package.
type ActionDef struct {
	StructName  string
	Methods     []MethodDef
	HasVars     bool
	Fields      []FieldDef      // exported struct fields (for DI resolution)
	Constructor *ConstructorDef // non-nil if a New<Name>() constructor exists
}

// ConstructorDef describes a New<Name>() constructor function for DI wiring.
type ConstructorDef struct {
	FuncName string     // e.g. "NewBlog"
	Params   []FieldDef // constructor parameters with type info
}

// MethodDef describes an exported method on an action struct.
type MethodDef struct {
	GoName      string     // e.g. "GetPosts"
	JSName      string     // e.g. "getPosts"
	Params      []ParamDef // excludes receiver
	ReturnType  string     // Go type string of first return value
	ReturnTS    string     // TypeScript type of first return value
	HasReturn   bool       // true if returns (T, error), false if returns just error
	HasContext   bool       // true if original method accepts context.Context as first param
	Doc         string     // godoc comment
}

// ParamDef describes a method parameter.
type ParamDef struct {
	Name   string
	GoType string
	TSType string
}

// FieldDef describes an exported struct field (for DI wiring).
type FieldDef struct {
	Name     string
	GoType   string // fully qualified Go type string
	TypePath string // package path for DI resolution
}

// ParseResult holds everything the code generators need.
type ParseResult struct {
	PackageName string
	PackagePath string
	Actions     []ActionDef
	NamedTypes  map[string]*types.Struct // Go struct types referenced in signatures
}

// Parse loads the actions package at the given directory and extracts action definitions.
func Parse(actionsDir string) (*ParseResult, error) {
	cfg := &packages.Config{
		Mode: packages.NeedName | packages.NeedTypes | packages.NeedTypesInfo | packages.NeedSyntax,
		Dir:  actionsDir,
	}

	pkgs, err := packages.Load(cfg, ".")
	if err != nil {
		return nil, fmt.Errorf("load actions package: %w", err)
	}
	if len(pkgs) == 0 {
		return nil, fmt.Errorf("no Go package found in %s", actionsDir)
	}
	pkg := pkgs[0]
	if len(pkg.Errors) > 0 {
		// Collect errors but skip "could not import" errors for generated files
		var errs []string
		for _, e := range pkg.Errors {
			errs = append(errs, e.Error())
		}
		// Only fail if there are real errors (not just missing generated file)
		if !allImportErrors(errs) {
			return nil, fmt.Errorf("package errors: %s", strings.Join(errs, "; "))
		}
	}

	result := &ParseResult{
		PackageName: pkg.Name,
		PackagePath: pkg.PkgPath,
		NamedTypes:  make(map[string]*types.Struct),
	}

	scope := pkg.Types.Scope()
	for _, name := range scope.Names() {
		obj := scope.Lookup(name)
		tn, ok := obj.(*types.TypeName)
		if !ok {
			continue
		}
		st, ok := tn.Type().Underlying().(*types.Struct)
		if !ok {
			continue
		}
		if !tn.Exported() {
			continue
		}

		action := extractAction(tn, st, pkg.Types, result.NamedTypes)
		if action == nil {
			continue
		}
		result.Actions = append(result.Actions, *action)
	}

	// Detect New<Name>() constructor functions and attach to their actions.
	for i := range result.Actions {
		ctorName := "New" + result.Actions[i].StructName
		obj := scope.Lookup(ctorName)
		if obj == nil {
			continue
		}
		fn, ok := obj.(*types.Func)
		if !ok {
			continue
		}
		sig := fn.Type().(*types.Signature)
		// Must return *<StructName>.
		if sig.Results().Len() != 1 {
			continue
		}
		retPtr, ok := sig.Results().At(0).Type().(*types.Pointer)
		if !ok {
			continue
		}
		retNamed, ok := retPtr.Elem().(*types.Named)
		if !ok || retNamed.Obj().Name() != result.Actions[i].StructName {
			continue
		}
		ctor := &ConstructorDef{FuncName: ctorName}
		params := sig.Params()
		for j := 0; j < params.Len(); j++ {
			p := params.At(j)
			fd := FieldDef{
				Name:   p.Name(),
				GoType: p.Type().String(),
			}
			if named, ok := p.Type().(*types.Named); ok {
				if named.Obj().Pkg() != nil {
					fd.TypePath = named.Obj().Pkg().Path()
				}
			} else if ptr, ok := p.Type().(*types.Pointer); ok {
				if named, ok := ptr.Elem().(*types.Named); ok {
					if named.Obj().Pkg() != nil {
						fd.TypePath = named.Obj().Pkg().Path()
					}
				}
			}
			// Action constructors must resolve dependencies from the DI
			// registry via core.MustInvoke inside the body — only *core.Registry
			// (or no parameters at all) is accepted as a parameter type.
			isRegistry := fd.TypePath != "" && fd.GoType == "*"+fd.TypePath+".Registry" && strings.HasSuffix(fd.TypePath, "/core")
			if !isRegistry {
				return nil, fmt.Errorf("%s parameter %d has type %s; action constructors must take (r *core.Registry) or no parameters — resolve dependencies from the registry via core.MustInvoke inside the constructor body", ctorName, j+1, fd.GoType)
			}
			ctor.Params = append(ctor.Params, fd)
		}
		result.Actions[i].Constructor = ctor
	}

	return result, nil
}

func extractAction(tn *types.TypeName, st *types.Struct, pkg *types.Package, namedTypes map[string]*types.Struct) *ActionDef {
	named := tn.Type().(*types.Named)
	methods := collectMethods(named, namedTypes)

	// Check for Vars() method separately (it has a different signature)
	hasVars := hasVarsMethod(named)

	// Only include structs that have at least one exported method or Vars
	if len(methods) == 0 && !hasVars {
		return nil
	}

	var filteredMethods []MethodDef
	for _, m := range methods {
		filteredMethods = append(filteredMethods, m)
	}

	action := &ActionDef{
		StructName: tn.Name(),
		Methods:    filteredMethods,
		HasVars:    hasVars,
		Fields:     extractFields(st),
	}
	return action
}

func collectMethods(named *types.Named, namedTypes map[string]*types.Struct) []MethodDef {
	var methods []MethodDef

	// Check both value and pointer receiver methods
	mset := types.NewMethodSet(types.NewPointer(named))
	for i := 0; i < mset.Len(); i++ {
		sel := mset.At(i)
		fn, ok := sel.Obj().(*types.Func)
		if !ok || !fn.Exported() {
			continue
		}
		// Skip methods from embedded types
		if len(sel.Index()) > 1 {
			continue
		}

		sig := fn.Type().(*types.Signature)
		md := parseMethod(fn.Name(), sig, namedTypes)
		if md != nil {
			methods = append(methods, *md)
		}
	}
	return methods
}

func parseMethod(name string, sig *types.Signature, namedTypes map[string]*types.Struct) *MethodDef {
	results := sig.Results()

	// Must return error or (T, error)
	if results.Len() == 0 || results.Len() > 2 {
		return nil
	}

	lastResult := results.At(results.Len() - 1)
	if !isErrorType(lastResult.Type()) {
		return nil
	}

	md := &MethodDef{
		GoName: name,
		JSName: CamelCase(name),
	}

	if results.Len() == 2 {
		md.HasReturn = true
		retType := results.At(0).Type()
		md.ReturnType = retType.String()
		md.ReturnTS = TSType(retType, namedTypes)
	}

	// Parse parameters (skip context.Context if first param)
	params := sig.Params()
	for i := 0; i < params.Len(); i++ {
		p := params.At(i)
		// Skip context.Context but record that the method wants it
		if isContextType(p.Type()) {
			md.HasContext = true
			continue
		}
		paramName := p.Name()
		if paramName == "" {
			paramName = fmt.Sprintf("arg%d", i)
		}
		md.Params = append(md.Params, ParamDef{
			Name:   paramName,
			GoType: p.Type().String(),
			TSType: TSType(p.Type(), namedTypes),
		})
	}

	return md
}

func extractFields(st *types.Struct) []FieldDef {
	var fields []FieldDef
	for i := 0; i < st.NumFields(); i++ {
		f := st.Field(i)
		if !f.Exported() {
			continue
		}
		fd := FieldDef{
			Name:   f.Name(),
			GoType: f.Type().String(),
		}
		if named, ok := f.Type().(*types.Named); ok {
			if named.Obj().Pkg() != nil {
				fd.TypePath = named.Obj().Pkg().Path()
			}
		} else if ptr, ok := f.Type().(*types.Pointer); ok {
			if named, ok := ptr.Elem().(*types.Named); ok {
				if named.Obj().Pkg() != nil {
					fd.TypePath = named.Obj().Pkg().Path()
				}
			}
		}
		fields = append(fields, fd)
	}
	return fields
}

// hasVarsMethod checks if the named type has a Vars() map[string]any method.
func hasVarsMethod(named *types.Named) bool {
	mset := types.NewMethodSet(types.NewPointer(named))
	for i := 0; i < mset.Len(); i++ {
		sel := mset.At(i)
		fn, ok := sel.Obj().(*types.Func)
		if !ok || fn.Name() != "Vars" {
			continue
		}
		sig := fn.Type().(*types.Signature)
		// Vars() should have no params and return map[string]any
		if sig.Params().Len() != 0 {
			continue
		}
		if sig.Results().Len() != 1 {
			continue
		}
		retType := sig.Results().At(0).Type()
		if mp, ok := retType.(*types.Map); ok {
			if basic, ok := mp.Key().(*types.Basic); ok && basic.Kind() == types.String {
				return true
			}
		}
		return false
	}
	return false
}

func isErrorType(t types.Type) bool {
	if named, ok := t.(*types.Named); ok {
		return named.Obj().Name() == "error" && named.Obj().Pkg() == nil
	}
	return false
}

func isContextType(t types.Type) bool {
	if named, ok := t.(*types.Named); ok {
		return named.Obj().Pkg() != nil &&
			named.Obj().Pkg().Path() == "context" &&
			named.Obj().Name() == "Context"
	}
	return false
}

func allImportErrors(errs []string) bool {
	for _, e := range errs {
		if !strings.Contains(e, "could not import") {
			return false
		}
	}
	return true
}
