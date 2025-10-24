package lsp

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
)

// ProjectContext holds project-wide type information
type ProjectContext struct {
	ModulePath   string
	Structs      map[string]*StructInfo // fully qualified name -> struct
	ImportPaths  []string
	RootDir      string
}

type StructInfo struct {
	Name       string
	Package    string
	Fields     map[string]FieldInfo
	ImportPath string
}

type FieldInfo struct {
	Name string
	Type string
}

// NewProjectContext scans project and extracts type info
func NewProjectContext(rootDir string) (*ProjectContext, error) {
	ctx := &ProjectContext{
		Structs:     make(map[string]*StructInfo),
		ImportPaths: make([]string, 0),
		RootDir:     rootDir,
	}

	// Read go.mod for module path
	modPath := filepath.Join(rootDir, "go.mod")
	if data, err := os.ReadFile(modPath); err == nil {
		lines := strings.Split(string(data), "\n")
		for _, line := range lines {
			if strings.HasPrefix(strings.TrimSpace(line), "module ") {
				ctx.ModulePath = strings.TrimSpace(strings.TrimPrefix(line, "module "))
				break
			}
		}
	}

	// Scan src/lib directories for Go files
	srcDirs := []string{"src/lib/models", "src/lib/repositories", "src/lib/services"}
	for _, dir := range srcDirs {
		fullPath := filepath.Join(rootDir, dir)
		if err := ctx.scanDirectory(fullPath, dir); err == nil {
			// Add to import paths
			ctx.ImportPaths = append(ctx.ImportPaths, ctx.ModulePath+"/"+dir)
		}
	}

	return ctx, nil
}

func (pc *ProjectContext) scanDirectory(dirPath, relativePath string) error {
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") {
			continue
		}

		filePath := filepath.Join(dirPath, entry.Name())
		if err := pc.parseFile(filePath, relativePath); err != nil {
			continue
		}
	}

	return nil
}

func (pc *ProjectContext) parseFile(filePath, relativePath string) error {
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, filePath, nil, parser.ParseComments)
	if err != nil {
		return err
	}

	packageName := node.Name.Name
	importPath := pc.ModulePath + "/" + relativePath

	for _, decl := range node.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok || genDecl.Tok != token.TYPE {
			continue
		}

		for _, spec := range genDecl.Specs {
			typeSpec, ok := spec.(*ast.TypeSpec)
			if !ok {
				continue
			}

			structType, ok := typeSpec.Type.(*ast.StructType)
			if !ok {
				continue
			}

			structInfo := &StructInfo{
				Name:       typeSpec.Name.Name,
				Package:    packageName,
				Fields:     make(map[string]FieldInfo),
				ImportPath: importPath,
			}

			// Extract fields
			if structType.Fields != nil {
				for _, field := range structType.Fields.List {
					fieldType := pc.exprToString(field.Type)
					for _, name := range field.Names {
						structInfo.Fields[name.Name] = FieldInfo{
							Name: name.Name,
							Type: fieldType,
						}
					}
				}
			}

			// Store with fully qualified name
			fqn := packageName + "." + typeSpec.Name.Name
			pc.Structs[fqn] = structInfo
			
			// Also store with import path
			pc.Structs[importPath+"."+typeSpec.Name.Name] = structInfo
		}
	}

	return nil
}

func (pc *ProjectContext) exprToString(expr ast.Expr) string {
	switch e := expr.(type) {
	case *ast.Ident:
		return e.Name
	case *ast.SelectorExpr:
		return pc.exprToString(e.X) + "." + e.Sel.Name
	case *ast.StarExpr:
		return "*" + pc.exprToString(e.X)
	case *ast.ArrayType:
		return "[]" + pc.exprToString(e.Elt)
	case *ast.MapType:
		return "map[" + pc.exprToString(e.Key) + "]" + pc.exprToString(e.Value)
	}
	return "interface{}"
}

func (pc *ProjectContext) GetStruct(typeName string) (*StructInfo, bool) {
	s, ok := pc.Structs[typeName]
	return s, ok
}

func (pc *ProjectContext) GetImportPaths() []string {
	// Common imports
	common := []string{
		"fmt",
		"time",
		"github.com/google/uuid",
		"gorm.io/gorm",
	}
	
	return append(common, pc.ImportPaths...)
}
