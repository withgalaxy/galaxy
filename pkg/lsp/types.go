package lsp

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
)

// TypeInfo represents type information for a variable
type TypeInfo struct {
	Name     string
	TypeName string
	IsStruct bool
	IsMap    bool
	IsSlice  bool
	Fields   map[string]TypeInfo // For structs/maps
}

// TypeInferencer infers types from Go AST without execution
type TypeInferencer struct {
	types   map[string]TypeInfo
	project *ProjectContext
}

func NewTypeInferencer() *TypeInferencer {
	return &TypeInferencer{
		types: make(map[string]TypeInfo),
	}
}

func NewTypeInferencerWithProject(project *ProjectContext) *TypeInferencer {
	return &TypeInferencer{
		types:   make(map[string]TypeInfo),
		project: project,
	}
}

// InferTypes parses frontmatter and extracts type information
func (ti *TypeInferencer) InferTypes(frontmatter string) error {
	fset := token.NewFileSet()

	// Wrap in package/func for parsing
	wrapped := "package main\nfunc init() {\n" + frontmatter + "\n}"

	node, err := parser.ParseFile(fset, "", wrapped, parser.AllErrors)
	if err != nil {
		return fmt.Errorf("parse error: %w", err)
	}

	// Add Galaxy API types
	ti.addGalaxyTypes()

	// Walk AST and extract type info
	for _, decl := range node.Decls {
		if funcDecl, ok := decl.(*ast.FuncDecl); ok {
			if funcDecl.Body != nil {
				ti.processBlockStmt(funcDecl.Body)
			}
		}
	}

	return nil
}

func (ti *TypeInferencer) addGalaxyTypes() {
	// Galaxy.Locals (map[string]any)
	ti.types["Galaxy"] = TypeInfo{
		Name:     "Galaxy",
		TypeName: "GalaxyAPI",
		IsStruct: true,
		Fields: map[string]TypeInfo{
			"Redirect": {
				Name:     "Redirect",
				TypeName: "func(string, int)",
			},
			"Locals": {
				Name:     "Locals",
				TypeName: "map[string]any",
				IsMap:    true,
				Fields:   make(map[string]TypeInfo),
			},
			"Params": {
				Name:     "Params",
				TypeName: "map[string]interface{}",
				IsMap:    true,
				Fields:   make(map[string]TypeInfo),
			},
		},
	}
}

func (ti *TypeInferencer) processBlockStmt(block *ast.BlockStmt) {
	for _, stmt := range block.List {
		ti.processStmt(stmt)
	}
}

func (ti *TypeInferencer) processStmt(stmt ast.Stmt) {
	switch s := stmt.(type) {
	case *ast.DeclStmt:
		if genDecl, ok := s.Decl.(*ast.GenDecl); ok {
			if genDecl.Tok == token.VAR {
				for _, spec := range genDecl.Specs {
					if valueSpec, ok := spec.(*ast.ValueSpec); ok {
						ti.processVarSpec(valueSpec)
					}
				}
			}
		}
	case *ast.AssignStmt:
		ti.processAssignStmt(s)
	case *ast.IfStmt:
		if s.Body != nil {
			ti.processBlockStmt(s.Body)
		}
		if s.Else != nil {
			if elseBlock, ok := s.Else.(*ast.BlockStmt); ok {
				ti.processBlockStmt(elseBlock)
			} else if elseIf, ok := s.Else.(*ast.IfStmt); ok {
				ti.processStmt(elseIf)
			}
		}
	}
}

func (ti *TypeInferencer) processVarSpec(spec *ast.ValueSpec) {
	for i, name := range spec.Names {
		typeInfo := TypeInfo{Name: name.Name}

		// Explicit type
		if spec.Type != nil {
			typeInfo.TypeName = ti.exprToTypeName(spec.Type)
			typeInfo.IsStruct = ti.isStructType(spec.Type)
			typeInfo.IsMap = ti.isMapType(spec.Type)
			typeInfo.IsSlice = ti.isSliceType(spec.Type)
		}

		// Infer from value
		if i < len(spec.Values) {
			inferredType := ti.inferExprType(spec.Values[i])
			if typeInfo.TypeName == "" {
				typeInfo.TypeName = inferredType.TypeName
				typeInfo.IsStruct = inferredType.IsStruct
				typeInfo.IsMap = inferredType.IsMap
				typeInfo.IsSlice = inferredType.IsSlice
				typeInfo.Fields = inferredType.Fields
			}
		}

		ti.types[name.Name] = typeInfo
	}
}

func (ti *TypeInferencer) processAssignStmt(stmt *ast.AssignStmt) {
	for i, lhs := range stmt.Lhs {
		if ident, ok := lhs.(*ast.Ident); ok {
			if i < len(stmt.Rhs) {
				inferredType := ti.inferExprType(stmt.Rhs[i])
				inferredType.Name = ident.Name
				ti.types[ident.Name] = inferredType
			}
		}
	}
}

func (ti *TypeInferencer) inferExprType(expr ast.Expr) TypeInfo {
	switch e := expr.(type) {
	case *ast.BasicLit:
		return TypeInfo{
			TypeName: ti.basicLitType(e),
		}
	case *ast.CompositeLit:
		return ti.compositeLitType(e)
	case *ast.CallExpr:
		return ti.callExprType(e)
	case *ast.SelectorExpr:
		return ti.selectorExprType(e)
	case *ast.IndexExpr:
		return ti.indexExprType(e)
	case *ast.TypeAssertExpr:
		return ti.typeAssertExprType(e)
	case *ast.Ident:
		if typeInfo, ok := ti.types[e.Name]; ok {
			return typeInfo
		}
	}
	return TypeInfo{TypeName: "interface{}"}
}

func (ti *TypeInferencer) basicLitType(lit *ast.BasicLit) string {
	switch lit.Kind {
	case token.INT:
		return "int64"
	case token.FLOAT:
		return "float64"
	case token.STRING:
		return "string"
	case token.CHAR:
		return "rune"
	}
	return "interface{}"
}

func (ti *TypeInferencer) compositeLitType(lit *ast.CompositeLit) TypeInfo {
	if lit.Type != nil {
		return TypeInfo{
			TypeName: ti.exprToTypeName(lit.Type),
			IsStruct: ti.isStructType(lit.Type),
			IsMap:    ti.isMapType(lit.Type),
			IsSlice:  ti.isSliceType(lit.Type),
		}
	}

	// Infer from elements
	if len(lit.Elts) > 0 {
		if _, ok := lit.Elts[0].(*ast.KeyValueExpr); ok {
			return TypeInfo{
				TypeName: "map[string]interface{}",
				IsMap:    true,
			}
		}
		return TypeInfo{
			TypeName: "[]interface{}",
			IsSlice:  true,
		}
	}

	return TypeInfo{TypeName: "interface{}"}
}

func (ti *TypeInferencer) callExprType(call *ast.CallExpr) TypeInfo {
	// Repository methods return (Entity, error)
	if sel, ok := call.Fun.(*ast.SelectorExpr); ok {
		methodName := sel.Sel.Name

		// Detect repository patterns
		if strings.HasSuffix(methodName, "ByID") || strings.HasSuffix(methodName, "All") {
			// Extract entity type from method name or repo var
			return TypeInfo{
				TypeName: "struct",
				IsStruct: true,
				Fields:   make(map[string]TypeInfo),
			}
		}
	}

	return TypeInfo{TypeName: "interface{}"}
}

func (ti *TypeInferencer) selectorExprType(sel *ast.SelectorExpr) TypeInfo {
	// Get base type
	if ident, ok := sel.X.(*ast.Ident); ok {
		if baseType, ok := ti.types[ident.Name]; ok {
			// Access field/method
			if baseType.IsStruct && baseType.Fields != nil {
				if fieldType, ok := baseType.Fields[sel.Sel.Name]; ok {
					return fieldType
				}
			}
			if baseType.IsMap {
				return TypeInfo{TypeName: "any"}
			}
		}
	}

	return TypeInfo{TypeName: "interface{}"}
}

func (ti *TypeInferencer) indexExprType(idx *ast.IndexExpr) TypeInfo {
	baseType := ti.inferExprType(idx.X)

	if baseType.IsMap {
		return TypeInfo{TypeName: "any"}
	}

	if baseType.IsSlice {
		// Extract element type
		elemType := strings.TrimPrefix(baseType.TypeName, "[]")
		return TypeInfo{TypeName: elemType}
	}

	return TypeInfo{TypeName: "interface{}"}
}

func (ti *TypeInferencer) typeAssertExprType(assert *ast.TypeAssertExpr) TypeInfo {
	if assert.Type != nil {
		return TypeInfo{
			TypeName: ti.exprToTypeName(assert.Type),
			IsStruct: ti.isStructType(assert.Type),
			IsMap:    ti.isMapType(assert.Type),
		}
	}
	return TypeInfo{TypeName: "interface{}"}
}

func (ti *TypeInferencer) exprToTypeName(expr ast.Expr) string {
	switch e := expr.(type) {
	case *ast.Ident:
		return e.Name
	case *ast.SelectorExpr:
		return ti.exprToTypeName(e.X) + "." + e.Sel.Name
	case *ast.StarExpr:
		return "*" + ti.exprToTypeName(e.X)
	case *ast.ArrayType:
		return "[]" + ti.exprToTypeName(e.Elt)
	case *ast.MapType:
		return "map[" + ti.exprToTypeName(e.Key) + "]" + ti.exprToTypeName(e.Value)
	}
	return "interface{}"
}

func (ti *TypeInferencer) isStructType(expr ast.Expr) bool {
	typeName := ti.exprToTypeName(expr)
	return !strings.HasPrefix(typeName, "map[") &&
		!strings.HasPrefix(typeName, "[]") &&
		!ti.isBuiltinType(typeName)
}

func (ti *TypeInferencer) isMapType(expr ast.Expr) bool {
	_, ok := expr.(*ast.MapType)
	return ok
}

func (ti *TypeInferencer) isSliceType(expr ast.Expr) bool {
	_, ok := expr.(*ast.ArrayType)
	return ok
}

func (ti *TypeInferencer) isBuiltinType(typeName string) bool {
	builtins := []string{"string", "int", "int64", "int32", "float64", "float32", "bool", "byte", "rune", "any", "interface{}"}
	for _, b := range builtins {
		if typeName == b {
			return true
		}
	}
	return false
}

func (ti *TypeInferencer) GetType(name string) (TypeInfo, bool) {
	typeInfo, ok := ti.types[name]
	return typeInfo, ok
}

func (ti *TypeInferencer) GetAllTypes() map[string]TypeInfo {
	return ti.types
}
