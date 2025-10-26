package executor

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"reflect"
	"strconv"
	"strings"
	"sync"
)

type PackageFunc func(args ...interface{}) (interface{}, error)

var (
	globalFuncs      = make(map[string]PackageFunc)
	globalFuncsMutex sync.RWMutex
)

func RegisterGlobalFunc(pkg, name string, fn PackageFunc) {
	globalFuncsMutex.Lock()
	defer globalFuncsMutex.Unlock()
	globalFuncs[pkg+"."+name] = fn
}

type Context struct {
	Variables      map[string]interface{}
	Props          map[string]interface{}
	Slots          map[string]string
	Request        interface{}
	Locals         map[string]any
	RedirectURL    string
	RedirectStatus int
	ShouldRedirect bool
	PackageFuncs   map[string]PackageFunc
}

type GalaxyAPI struct {
	ctx    *Context
	Params map[string]interface{}
	Locals map[string]interface{}
}

func (g *GalaxyAPI) Redirect(url string, status int) {
	g.ctx.RedirectURL = url
	g.ctx.RedirectStatus = status
	g.ctx.ShouldRedirect = true
}

func NewContext() *Context {
	ctx := &Context{
		Variables:    make(map[string]interface{}),
		Props:        make(map[string]interface{}),
		Slots:        make(map[string]string),
		Request:      nil,
		Locals:       make(map[string]any),
		PackageFuncs: make(map[string]PackageFunc),
	}

	globalFuncsMutex.RLock()
	for k, v := range globalFuncs {
		ctx.PackageFuncs[k] = v
	}
	globalFuncsMutex.RUnlock()

	galaxyAPI := &GalaxyAPI{
		ctx:    ctx,
		Params: make(map[string]interface{}),
		Locals: ctx.Locals,
	}
	ctx.Variables["Galaxy"] = galaxyAPI
	return ctx
}

func (c *Context) RegisterPackageFunc(pkg, name string, fn PackageFunc) {
	key := pkg + "." + name
	c.PackageFuncs[key] = fn
}

func ExtractImports(code string) (imports string, rest string) {
	return extractImports(code)
}

func extractImports(code string) (imports string, rest string) {
	lines := strings.Split(code, "\n")
	var importLines []string
	var codeLines []string
	inImportBlock := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		if strings.HasPrefix(trimmed, "import (") {
			inImportBlock = true
			importLines = append(importLines, line)
		} else if inImportBlock {
			importLines = append(importLines, line)
			if strings.Contains(trimmed, ")") {
				inImportBlock = false
			}
		} else if strings.HasPrefix(trimmed, "import ") {
			if !strings.Contains(line, " from ") {
				importLines = append(importLines, line)
			}
		} else {
			codeLines = append(codeLines, line)
		}
	}

	if len(importLines) > 0 {
		imports = strings.Join(importLines, "\n") + "\n"
	}
	rest = strings.Join(codeLines, "\n")
	return
}

func (c *Context) Execute(code string) error {
	fset := token.NewFileSet()

	imports, codeWithoutImports := extractImports(code)
	wrappedCode := "package main\n" + imports + "func init() {\n" + codeWithoutImports + "\n}"

	node, err := parser.ParseFile(fset, "", wrappedCode, parser.AllErrors)
	if err != nil {
		return fmt.Errorf("parse error: %w", err)
	}

	for _, decl := range node.Decls {
		if genDecl, ok := decl.(*ast.GenDecl); ok {
			if genDecl.Tok == token.VAR {
				for _, spec := range genDecl.Specs {
					if valueSpec, ok := spec.(*ast.ValueSpec); ok {
						if err := c.processVarSpec(valueSpec); err != nil {
							return err
						}
					}
				}
			}
		}
		if funcDecl, ok := decl.(*ast.FuncDecl); ok {
			if funcDecl.Name.Name == "init" && funcDecl.Body != nil {
				for _, stmt := range funcDecl.Body.List {
					if err := c.executeStmt(stmt); err != nil {
						return err
					}
					if c.ShouldRedirect {
						return nil
					}
				}
			}
		}
	}

	return nil
}

func (c *Context) executeStmt(stmt ast.Stmt) error {
	switch s := stmt.(type) {
	case *ast.IfStmt:
		return c.executeIfStmt(s)
	case *ast.ExprStmt:
		_, err := c.evalExpr(s.X)
		return err
	case *ast.AssignStmt:
		return c.executeAssignStmt(s)
	case *ast.DeclStmt:
		if genDecl, ok := s.Decl.(*ast.GenDecl); ok {
			if genDecl.Tok == token.VAR {
				for _, spec := range genDecl.Specs {
					if valueSpec, ok := spec.(*ast.ValueSpec); ok {
						if err := c.processVarSpec(valueSpec); err != nil {
							return err
						}
					}
				}
			}
		}
		return nil
	default:
		return nil
	}
}

func (c *Context) executeIfStmt(stmt *ast.IfStmt) error {
	cond, err := c.evalExpr(stmt.Cond)
	if err != nil {
		return err
	}

	condBool, ok := cond.(bool)
	if !ok {
		return fmt.Errorf("if condition must be boolean")
	}

	if condBool {
		for _, s := range stmt.Body.List {
			if err := c.executeStmt(s); err != nil {
				return err
			}
			if c.ShouldRedirect {
				return nil
			}
		}
	} else if stmt.Else != nil {
		switch elseStmt := stmt.Else.(type) {
		case *ast.BlockStmt:
			for _, s := range elseStmt.List {
				if err := c.executeStmt(s); err != nil {
					return err
				}
				if c.ShouldRedirect {
					return nil
				}
			}
		case *ast.IfStmt:
			return c.executeIfStmt(elseStmt)
		}
	}

	return nil
}

func (c *Context) executeAssignStmt(stmt *ast.AssignStmt) error {
	if stmt.Tok != token.DEFINE && stmt.Tok != token.ASSIGN {
		return fmt.Errorf("unsupported assignment operator: %v", stmt.Tok)
	}

	if len(stmt.Lhs) == 2 && len(stmt.Rhs) == 1 {
		if ident1, ok := stmt.Lhs[0].(*ast.Ident); ok {
			if ident2, ok := stmt.Lhs[1].(*ast.Ident); ok {
				result, err := c.evalExpr(stmt.Rhs[0])
				if err != nil {
					return err
				}

				// Check if result is a multi-value tuple
				if resultSlice, ok := result.([]interface{}); ok && len(resultSlice) == 2 {
					c.Variables[ident1.Name] = resultSlice[0]
					c.Variables[ident2.Name] = resultSlice[1]
				} else {
					c.Variables[ident1.Name] = result
					c.Variables[ident2.Name] = nil
				}
				return nil
			}
		}
	}

	for i, lhs := range stmt.Lhs {
		if i >= len(stmt.Rhs) {
			break
		}

		val, err := c.evalExpr(stmt.Rhs[i])
		if err != nil {
			return err
		}

		if ident, ok := lhs.(*ast.Ident); ok {
			c.Variables[ident.Name] = val
		}
	}

	return nil
}

func (c *Context) processVarSpec(spec *ast.ValueSpec) error {
	if len(spec.Names) == 2 && len(spec.Values) == 1 {
		result, err := c.evalExpr(spec.Values[0])
		if err != nil {
			return err
		}

		// Check if result is a multi-value tuple (slice with 2 elements)
		if resultSlice, ok := result.([]interface{}); ok && len(resultSlice) == 2 {
			c.Variables[spec.Names[0].Name] = resultSlice[0]
			c.Variables[spec.Names[1].Name] = resultSlice[1]
		} else {
			c.Variables[spec.Names[0].Name] = result
			c.Variables[spec.Names[1].Name] = nil
		}
		return nil
	}

	for i, name := range spec.Names {
		if i < len(spec.Values) {
			value, err := c.evalExpr(spec.Values[i])
			if err != nil {
				return err
			}
			// If single var gets tuple from PackageFunc/method, unwrap first value
			if len(spec.Names) == 1 {
				if tuple, ok := value.([]interface{}); ok && len(tuple) == 2 {
					// Unwrap tuple for single assignment
					value = tuple[0]
				}
			}
			c.Variables[name.Name] = value
		}
	}
	return nil
}

func (c *Context) evalExpr(expr ast.Expr) (interface{}, error) {
	switch e := expr.(type) {
	case *ast.BasicLit:
		return c.evalBasicLit(e)
	case *ast.Ident:
		if e.Name == "nil" {
			return nil, nil
		}
		if e.Name == "true" {
			return true, nil
		}
		if e.Name == "false" {
			return false, nil
		}
		if val, ok := c.Variables[e.Name]; ok {
			return val, nil
		}
		return nil, fmt.Errorf("undefined variable: %s", e.Name)
	case *ast.SelectorExpr:
		return c.evalSelectorExpr(e)
	case *ast.BinaryExpr:
		return c.evalBinaryExpr(e)
	case *ast.UnaryExpr:
		return c.evalUnaryExpr(e)
	case *ast.CallExpr:
		return c.evalCallExpr(e)
	case *ast.CompositeLit:
		return c.evalCompositeLit(e)
	case *ast.ParenExpr:
		return c.evalExpr(e.X)
	case *ast.IndexExpr:
		return c.evalIndexExpr(e)
	case *ast.TypeAssertExpr:
		return c.evalTypeAssertExpr(e)
	default:
		return nil, fmt.Errorf("unsupported expression type: %T", expr)
	}
}

func (c *Context) evalBasicLit(lit *ast.BasicLit) (interface{}, error) {
	switch lit.Kind {
	case token.INT:
		return strconv.ParseInt(lit.Value, 10, 64)
	case token.FLOAT:
		return strconv.ParseFloat(lit.Value, 64)
	case token.STRING:
		s := lit.Value
		if len(s) >= 2 {
			s = s[1 : len(s)-1]
		}
		return s, nil
	case token.CHAR:
		if len(lit.Value) >= 3 {
			return rune(lit.Value[1]), nil
		}
		return nil, fmt.Errorf("invalid char literal")
	default:
		return nil, fmt.Errorf("unsupported literal kind: %v", lit.Kind)
	}
}

func (c *Context) evalBinaryExpr(expr *ast.BinaryExpr) (interface{}, error) {
	left, err := c.evalExpr(expr.X)
	if err != nil {
		return nil, err
	}

	right, err := c.evalExpr(expr.Y)
	if err != nil {
		return nil, err
	}

	switch expr.Op {
	case token.ADD:
		return c.add(left, right)
	case token.SUB:
		return c.sub(left, right)
	case token.MUL:
		return c.mul(left, right)
	case token.QUO:
		return c.div(left, right)
	case token.EQL:
		return c.equal(left, right), nil
	case token.NEQ:
		return !c.equal(left, right), nil
	case token.LSS:
		return c.less(left, right)
	case token.LEQ:
		return c.lessEqual(left, right)
	case token.GTR:
		return c.greater(left, right)
	case token.GEQ:
		return c.greaterEqual(left, right)
	case token.LAND:
		return c.logicalAnd(left, right), nil
	case token.LOR:
		return c.logicalOr(left, right), nil
	default:
		return nil, fmt.Errorf("unsupported binary operator: %v", expr.Op)
	}
}

func (c *Context) evalUnaryExpr(expr *ast.UnaryExpr) (interface{}, error) {
	x, err := c.evalExpr(expr.X)
	if err != nil {
		return nil, err
	}

	switch expr.Op {
	case token.SUB:
		if v, ok := x.(int64); ok {
			return -v, nil
		}
		if v, ok := x.(float64); ok {
			return -v, nil
		}
	case token.NOT:
		if v, ok := x.(bool); ok {
			return !v, nil
		}
	case token.AND:
		// Address-of operator
		if x == nil {
			return nil, fmt.Errorf("cannot take address of nil")
		}
		ptr := reflect.New(reflect.TypeOf(x))
		ptr.Elem().Set(reflect.ValueOf(x))
		return ptr.Interface(), nil
	}
	return nil, fmt.Errorf("unsupported unary operator: %v", expr.Op)
}

func (c *Context) evalSelectorExpr(expr *ast.SelectorExpr) (interface{}, error) {
	x, err := c.evalExpr(expr.X)
	if err != nil {
		return nil, err
	}

	if x == nil {
		return nil, nil
	}

	if m, ok := x.(map[string]interface{}); ok {
		return m[expr.Sel.Name], nil
	}

	if m, ok := x.(map[string]any); ok {
		return m[expr.Sel.Name], nil
	}

	v := reflect.ValueOf(x)

	// Dereference pointers
	for v.Kind() == reflect.Ptr {
		if v.IsNil() {
			return nil, fmt.Errorf("nil pointer dereference")
		}
		v = v.Elem()
	}

	if v.Kind() == reflect.Struct {
		field := v.FieldByName(expr.Sel.Name)
		if field.IsValid() {
			return field.Interface(), nil
		}
	}

	return nil, fmt.Errorf("cannot select field %s from type %T", expr.Sel.Name, x)
}

func (c *Context) evalIndexExpr(expr *ast.IndexExpr) (interface{}, error) {
	x, err := c.evalExpr(expr.X)
	if err != nil {
		return nil, err
	}

	index, err := c.evalExpr(expr.Index)
	if err != nil {
		return nil, err
	}

	if m, ok := x.(map[string]interface{}); ok {
		if key, ok := index.(string); ok {
			return m[key], nil
		}
	}

	if m, ok := x.(map[string]any); ok {
		if key, ok := index.(string); ok {
			return m[key], nil
		}
	}

	v := reflect.ValueOf(x)
	if v.Kind() == reflect.Slice || v.Kind() == reflect.Array {
		if idx, ok := index.(int64); ok {
			if int(idx) >= 0 && int(idx) < v.Len() {
				return v.Index(int(idx)).Interface(), nil
			}
			return nil, fmt.Errorf("index out of bounds")
		}
	}

	return nil, fmt.Errorf("invalid index operation on type %T", x)
}

func (c *Context) evalTypeAssertExpr(expr *ast.TypeAssertExpr) (interface{}, error) {
	x, err := c.evalExpr(expr.X)
	if err != nil {
		return nil, err
	}
	return x, nil
}

func (c *Context) evalCallExpr(expr *ast.CallExpr) (interface{}, error) {
	if sel, ok := expr.Fun.(*ast.SelectorExpr); ok {
		if ident, ok := sel.X.(*ast.Ident); ok && ident.Name == "Galaxy" {
			if sel.Sel.Name == "redirect" || sel.Sel.Name == "Redirect" {
				return c.handleGalaxyRedirect(expr.Args)
			}
		}

		// Try evaluating selector X as object with method
		obj, err := c.evalExpr(sel.X)
		if err == nil && obj != nil {
			// Check if it's a method call on an object
			return c.invokeMethod(obj, sel.Sel.Name, expr.Args)
		}

		// Try as package function
		if ident, ok := sel.X.(*ast.Ident); ok {
			key := ident.Name + "." + sel.Sel.Name
			if fn, exists := c.PackageFuncs[key]; exists {
				var args []interface{}
				for _, argExpr := range expr.Args {
					arg, err := c.evalExpr(argExpr)
					if err != nil {
						return nil, err
					}
					args = append(args, arg)
				}
				result, fnErr := fn(args...)
				// Return as tuple for multi-value assignment (value, error)
				return []interface{}{result, fnErr}, nil
			}
		}
		return nil, nil
	}

	if ident, ok := expr.Fun.(*ast.Ident); ok {
		switch ident.Name {
		case "len":
			if len(expr.Args) != 1 {
				return nil, fmt.Errorf("len expects 1 argument")
			}
			arg, err := c.evalExpr(expr.Args[0])
			if err != nil {
				return nil, err
			}
			v := reflect.ValueOf(arg)
			if v.Kind() == reflect.Slice || v.Kind() == reflect.Array || v.Kind() == reflect.String {
				return int64(v.Len()), nil
			}
		}
	}
	return nil, nil
}

func (c *Context) invokeMethod(obj interface{}, methodName string, args []ast.Expr) (interface{}, error) {
	v := reflect.ValueOf(obj)

	// Try method on value first
	method := v.MethodByName(methodName)
	if !method.IsValid() {
		// Try pointer method
		if v.Kind() != reflect.Ptr && v.CanAddr() {
			method = v.Addr().MethodByName(methodName)
		}
	}

	if !method.IsValid() {
		return nil, fmt.Errorf("method %s not found on type %T", methodName, obj)
	}

	// Evaluate arguments
	var evaledArgs []interface{}
	for _, arg := range args {
		val, err := c.evalExpr(arg)
		if err != nil {
			return nil, err
		}
		evaledArgs = append(evaledArgs, val)
	}

	// Convert args to method parameter types
	argValues, err := c.convertArgsToTypes(evaledArgs, method)
	if err != nil {
		return nil, err
	}

	// Call method
	results := method.Call(argValues)

	// Handle return values
	return c.handleMethodReturns(results)
}

func (c *Context) convertArgsToTypes(args []interface{}, method reflect.Value) ([]reflect.Value, error) {
	methodType := method.Type()
	values := make([]reflect.Value, len(args))

	for i, arg := range args {
		paramType := methodType.In(i)
		argValue := reflect.ValueOf(arg)

		// Handle nil
		if arg == nil {
			values[i] = reflect.Zero(paramType)
			continue
		}

		// Direct assignment
		if argValue.Type().AssignableTo(paramType) {
			values[i] = argValue
			continue
		}

		// Conversion
		if argValue.Type().ConvertibleTo(paramType) {
			values[i] = argValue.Convert(paramType)
			continue
		}

		// Special case: int64 -> int
		if argValue.Kind() == reflect.Int64 && paramType.Kind() == reflect.Int {
			values[i] = reflect.ValueOf(int(arg.(int64)))
			continue
		}

		return nil, fmt.Errorf("cannot convert arg %d from %v to %v", i, argValue.Type(), paramType)
	}

	return values, nil
}

func (c *Context) handleMethodReturns(results []reflect.Value) (interface{}, error) {
	if len(results) == 0 {
		return nil, nil
	}

	// Check if last return is error
	lastResult := results[len(results)-1]
	errorType := reflect.TypeOf((*error)(nil)).Elem()

	if lastResult.Type().Implements(errorType) {
		var err error
		if !lastResult.IsNil() {
			err = lastResult.Interface().(error)
		}

		if len(results) == 1 {
			// Only error return
			return nil, err
		}

		if len(results) == 2 {
			// (value, error) pattern - return as tuple
			return []interface{}{results[0].Interface(), err}, nil
		}

		// Multiple values + error
		tuple := make([]interface{}, len(results))
		for i, r := range results {
			tuple[i] = r.Interface()
		}
		return tuple, nil
	}

	// Single non-error return
	if len(results) == 1 {
		return results[0].Interface(), nil
	}

	// Multiple non-error returns
	tuple := make([]interface{}, len(results))
	for i, r := range results {
		tuple[i] = r.Interface()
	}
	return tuple, nil
}

func (c *Context) handleGalaxyRedirect(args []ast.Expr) (interface{}, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf("Galaxy.redirect expects 2 arguments (url, status)")
	}

	url, err := c.evalExpr(args[0])
	if err != nil {
		return nil, err
	}
	urlStr, ok := url.(string)
	if !ok {
		return nil, fmt.Errorf("redirect URL must be string")
	}

	status, err := c.evalExpr(args[1])
	if err != nil {
		return nil, err
	}
	statusInt, ok := status.(int64)
	if !ok {
		return nil, fmt.Errorf("redirect status must be int")
	}

	c.RedirectURL = urlStr
	c.RedirectStatus = int(statusInt)
	c.ShouldRedirect = true

	return nil, nil
}

func (c *Context) evalCompositeLit(expr *ast.CompositeLit) (interface{}, error) {
	if _, ok := expr.Type.(*ast.ArrayType); ok {
		var result []interface{}
		for _, elt := range expr.Elts {
			val, err := c.evalExpr(elt)
			if err != nil {
				return nil, err
			}
			result = append(result, val)
		}
		return result, nil
	}

	if _, ok := expr.Type.(*ast.MapType); ok {
		result := make(map[string]interface{})
		for _, elt := range expr.Elts {
			kvExpr, ok := elt.(*ast.KeyValueExpr)
			if !ok {
				return nil, fmt.Errorf("expected key-value pair in map literal")
			}

			key, err := c.evalExpr(kvExpr.Key)
			if err != nil {
				return nil, err
			}

			value, err := c.evalExpr(kvExpr.Value)
			if err != nil {
				return nil, err
			}

			keyStr, ok := key.(string)
			if !ok {
				return nil, fmt.Errorf("map keys must be strings")
			}

			result[keyStr] = value
		}
		return result, nil
	}

	if expr.Type == nil && len(expr.Elts) > 0 {
		if _, ok := expr.Elts[0].(*ast.KeyValueExpr); ok {
			result := make(map[string]interface{})
			for _, elt := range expr.Elts {
				kvExpr, ok := elt.(*ast.KeyValueExpr)
				if !ok {
					return nil, fmt.Errorf("expected key-value pair in map literal")
				}

				key, err := c.evalExpr(kvExpr.Key)
				if err != nil {
					return nil, err
				}

				value, err := c.evalExpr(kvExpr.Value)
				if err != nil {
					return nil, err
				}

				keyStr, ok := key.(string)
				if !ok {
					return nil, fmt.Errorf("map keys must be strings")
				}

				result[keyStr] = value
			}
			return result, nil
		}
	}

	return nil, fmt.Errorf("unsupported composite literal")
}

func (c *Context) add(left, right interface{}) (interface{}, error) {
	if lInt, ok := left.(int64); ok {
		if rInt, ok := right.(int64); ok {
			return lInt + rInt, nil
		}
	}
	if lFloat, ok := left.(float64); ok {
		if rFloat, ok := right.(float64); ok {
			return lFloat + rFloat, nil
		}
	}
	if lStr, ok := left.(string); ok {
		if rStr, ok := right.(string); ok {
			return lStr + rStr, nil
		}
	}
	return nil, fmt.Errorf("invalid operands for +")
}

func (c *Context) sub(left, right interface{}) (interface{}, error) {
	if lInt, ok := left.(int64); ok {
		if rInt, ok := right.(int64); ok {
			return lInt - rInt, nil
		}
	}
	if lFloat, ok := left.(float64); ok {
		if rFloat, ok := right.(float64); ok {
			return lFloat - rFloat, nil
		}
	}
	return nil, fmt.Errorf("invalid operands for -")
}

func (c *Context) mul(left, right interface{}) (interface{}, error) {
	if lInt, ok := left.(int64); ok {
		if rInt, ok := right.(int64); ok {
			return lInt * rInt, nil
		}
	}
	if lFloat, ok := left.(float64); ok {
		if rFloat, ok := right.(float64); ok {
			return lFloat * rFloat, nil
		}
	}
	return nil, fmt.Errorf("invalid operands for *")
}

func (c *Context) div(left, right interface{}) (interface{}, error) {
	if lInt, ok := left.(int64); ok {
		if rInt, ok := right.(int64); ok {
			if rInt == 0 {
				return nil, fmt.Errorf("division by zero")
			}
			return lInt / rInt, nil
		}
	}
	if lFloat, ok := left.(float64); ok {
		if rFloat, ok := right.(float64); ok {
			if rFloat == 0 {
				return nil, fmt.Errorf("division by zero")
			}
			return lFloat / rFloat, nil
		}
	}
	return nil, fmt.Errorf("invalid operands for /")
}

func (c *Context) equal(left, right interface{}) bool {
	if left == nil && right == nil {
		return true
	}
	if left == nil || right == nil {
		return false
	}
	return reflect.DeepEqual(left, right)
}

func (c *Context) less(left, right interface{}) (interface{}, error) {
	if lInt, ok := left.(int64); ok {
		if rInt, ok := right.(int64); ok {
			return lInt < rInt, nil
		}
	}
	if lFloat, ok := left.(float64); ok {
		if rFloat, ok := right.(float64); ok {
			return lFloat < rFloat, nil
		}
	}
	return nil, fmt.Errorf("invalid operands for <")
}

func (c *Context) lessEqual(left, right interface{}) (interface{}, error) {
	if lInt, ok := left.(int64); ok {
		if rInt, ok := right.(int64); ok {
			return lInt <= rInt, nil
		}
	}
	if lFloat, ok := left.(float64); ok {
		if rFloat, ok := right.(float64); ok {
			return lFloat <= rFloat, nil
		}
	}
	return nil, fmt.Errorf("invalid operands for <=")
}

func (c *Context) greater(left, right interface{}) (interface{}, error) {
	if lInt, ok := left.(int64); ok {
		if rInt, ok := right.(int64); ok {
			return lInt > rInt, nil
		}
	}
	if lFloat, ok := left.(float64); ok {
		if rFloat, ok := right.(float64); ok {
			return lFloat > rFloat, nil
		}
	}
	return nil, fmt.Errorf("invalid operands for >")
}

func (c *Context) greaterEqual(left, right interface{}) (interface{}, error) {
	if lInt, ok := left.(int64); ok {
		if rInt, ok := right.(int64); ok {
			return lInt >= rInt, nil
		}
	}
	if lFloat, ok := left.(float64); ok {
		if rFloat, ok := right.(float64); ok {
			return lFloat >= rFloat, nil
		}
	}
	return nil, fmt.Errorf("invalid operands for >=")
}

func (c *Context) logicalAnd(left, right interface{}) bool {
	lBool, lOk := left.(bool)
	rBool, rOk := right.(bool)
	if lOk && rOk {
		return lBool && rBool
	}
	return false
}

func (c *Context) logicalOr(left, right interface{}) bool {
	lBool, lOk := left.(bool)
	rBool, rOk := right.(bool)
	if lOk && rOk {
		return lBool || rBool
	}
	return false
}

func (c *Context) Get(name string) (interface{}, bool) {
	val, ok := c.Variables[name]
	return val, ok
}

func (c *Context) Set(name string, value interface{}) {
	c.Variables[name] = value
}

func (c *Context) SetProp(name string, value interface{}) {
	c.Props[name] = value
}

func (c *Context) SetRequest(req interface{}) {
	c.Request = req
	c.Variables["Request"] = req
}

func (c *Context) GetRequest() (interface{}, bool) {
	return c.Request, c.Request != nil
}

func (c *Context) SetLocals(locals map[string]any) {
	c.Locals = locals
	c.Variables["Locals"] = locals
	// Update Galaxy.Locals reference
	if galaxy, ok := c.Variables["Galaxy"].(*GalaxyAPI); ok {
		galaxy.Locals = locals
	}
}

func (c *Context) GetLocals() map[string]any {
	return c.Locals
}

func (c *Context) GetProp(name string) (interface{}, bool) {
	val, ok := c.Props[name]
	return val, ok
}

func (c *Context) SetParams(params map[string]string) {
	if galaxy, ok := c.Variables["Galaxy"].(*GalaxyAPI); ok {
		galaxy.Params = make(map[string]interface{})
		for k, v := range params {
			galaxy.Params[k] = v
		}
	}
}

func (c *Context) GetParams() map[string]interface{} {
	if galaxy, ok := c.Variables["Galaxy"].(*GalaxyAPI); ok {
		return galaxy.Params
	}
	return make(map[string]interface{})
}

func (c *Context) String() string {
	var sb strings.Builder
	sb.WriteString("Context Variables:\n")
	for k, v := range c.Variables {
		sb.WriteString(fmt.Sprintf("  %s: %v (%T)\n", k, v, v))
	}
	return sb.String()
}
