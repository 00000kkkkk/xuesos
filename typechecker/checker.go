package typechecker

import (
	"fmt"

	"github.com/00000kkkkk/xusesosplusplus/lexer"
	"github.com/00000kkkkk/xusesosplusplus/parser"
)

// TypeError represents a type checking error.
type TypeError struct {
	Pos     lexer.Position
	Message string
}

func (e TypeError) Error() string {
	return fmt.Sprintf("%s: %s", e.Pos, e.Message)
}

// Scope tracks variable types in a scope.
type Scope struct {
	vars    map[string]Type
	mutable map[string]bool
	outer   *Scope
}

func newScope(outer *Scope) *Scope {
	return &Scope{
		vars:    make(map[string]Type),
		mutable: make(map[string]bool),
		outer:   outer,
	}
}

func (s *Scope) define(name string, t Type, isMutable bool) {
	s.vars[name] = t
	s.mutable[name] = isMutable
}

func (s *Scope) lookup(name string) (Type, bool) {
	t, ok := s.vars[name]
	if !ok && s.outer != nil {
		return s.outer.lookup(name)
	}
	return t, ok
}

func (s *Scope) isMutable(name string) bool {
	if m, ok := s.mutable[name]; ok {
		return m
	}
	if s.outer != nil {
		return s.outer.isMutable(name)
	}
	return false
}

// Checker performs type checking on the AST.
type Checker struct {
	errors     []TypeError
	structs    map[string]*StructType
	enums      map[string]*EnumType
	funcTypes  map[string]*FuncType
	scope      *Scope
	currentFn  *FuncType // current function being checked (for return type)
}

// New creates a new type checker.
func New() *Checker {
	c := &Checker{
		structs:   make(map[string]*StructType),
		enums:     make(map[string]*EnumType),
		funcTypes: make(map[string]*FuncType),
		scope:     newScope(nil),
	}
	c.registerBuiltins()
	return c
}

func (c *Checker) registerBuiltins() {
	// print(...) accepts anything
	c.scope.define("print", &FuncType{ReturnType: TypeVoid}, false)
	c.scope.define("println", &FuncType{ReturnType: TypeVoid}, false)
	c.scope.define("len", &FuncType{ParamTypes: []Type{TypeStr}, ReturnType: TypeInt}, false)
	c.scope.define("type", &FuncType{ParamTypes: []Type{TypeStr}, ReturnType: TypeStr}, false)
	c.scope.define("sqrt", &FuncType{ParamTypes: []Type{TypeFloat}, ReturnType: TypeFloat}, false)
	c.scope.define("to_int", &FuncType{ParamTypes: []Type{TypeFloat}, ReturnType: TypeInt}, false)
	c.scope.define("to_float", &FuncType{ParamTypes: []Type{TypeInt}, ReturnType: TypeFloat}, false)
	c.scope.define("to_str", &FuncType{ParamTypes: []Type{TypeInt}, ReturnType: TypeStr}, false)
	c.scope.define("append", &FuncType{ReturnType: &ArrayType{ElementType: TypeVoid}}, false)

	// String/array built-ins
	c.scope.define("contains", &FuncType{ReturnType: TypeBool}, false)
	c.scope.define("split", &FuncType{ReturnType: &ArrayType{ElementType: TypeStr}}, false)
	c.scope.define("trim", &FuncType{ReturnType: TypeStr}, false)
	c.scope.define("replace", &FuncType{ReturnType: TypeStr}, false)
	c.scope.define("upper", &FuncType{ReturnType: TypeStr}, false)
	c.scope.define("lower", &FuncType{ReturnType: TypeStr}, false)
	c.scope.define("starts_with", &FuncType{ReturnType: TypeBool}, false)
	c.scope.define("ends_with", &FuncType{ReturnType: TypeBool}, false)
	c.scope.define("join", &FuncType{ReturnType: TypeStr}, false)
	c.scope.define("slice", &FuncType{ReturnType: TypeVoid}, false)
	c.scope.define("push", &FuncType{ReturnType: TypeVoid}, false)
	c.scope.define("range_arr", &FuncType{ReturnType: &ArrayType{ElementType: TypeInt}}, false)
	c.scope.define("index_of", &FuncType{ReturnType: TypeInt}, false)
	c.scope.define("substr", &FuncType{ReturnType: TypeStr}, false)
	c.scope.define("char_at", &FuncType{ReturnType: TypeStr}, false)
	c.scope.define("char_code", &FuncType{ReturnType: TypeInt}, false)
	c.scope.define("from_char_code", &FuncType{ReturnType: TypeStr}, false)
	c.scope.define("string_len", &FuncType{ReturnType: TypeInt}, false)

	// Math built-ins
	c.scope.define("abs", &FuncType{ReturnType: TypeInt}, false)
	c.scope.define("max", &FuncType{ReturnType: TypeInt}, false)
	c.scope.define("min", &FuncType{ReturnType: TypeInt}, false)
	c.scope.define("math_pi", &FuncType{ReturnType: TypeFloat}, false)
	c.scope.define("math_e", &FuncType{ReturnType: TypeFloat}, false)
	c.scope.define("math_floor", &FuncType{ReturnType: TypeInt}, false)
	c.scope.define("math_ceil", &FuncType{ReturnType: TypeInt}, false)
	c.scope.define("math_pow", &FuncType{ReturnType: TypeFloat}, false)
	c.scope.define("math_round", &FuncType{ReturnType: TypeInt}, false)
	c.scope.define("math_sin", &FuncType{ReturnType: TypeFloat}, false)
	c.scope.define("math_cos", &FuncType{ReturnType: TypeFloat}, false)
	c.scope.define("math_log", &FuncType{ReturnType: TypeFloat}, false)

	// IO/OS built-ins
	c.scope.define("input", &FuncType{ReturnType: TypeStr}, false)
	c.scope.define("exit", &FuncType{ReturnType: TypeVoid}, false)
	c.scope.define("panic", &FuncType{ReturnType: TypeVoid}, false)
	c.scope.define("io_read_file", &FuncType{ReturnType: TypeStr}, false)
	c.scope.define("io_write_file", &FuncType{ReturnType: TypeVoid}, false)
	c.scope.define("os_args", &FuncType{ReturnType: &ArrayType{ElementType: TypeStr}}, false)
	c.scope.define("os_getenv", &FuncType{ReturnType: TypeStr}, false)

	// Map built-ins
	c.scope.define("has_key", &FuncType{ReturnType: TypeBool}, false)
	c.scope.define("keys", &FuncType{ReturnType: &ArrayType{ElementType: TypeStr}}, false)
	c.scope.define("values", &FuncType{ReturnType: &ArrayType{ElementType: TypeVoid}}, false)
	c.scope.define("delete", &FuncType{ReturnType: TypeVoid}, false)

	// Concurrency built-ins
	c.scope.define("spawn", &FuncType{ReturnType: TypeVoid}, false)
	c.scope.define("sleep", &FuncType{ReturnType: TypeVoid}, false)
	c.scope.define("wait", &FuncType{ReturnType: TypeVoid}, false)
	c.scope.define("channel", &FuncType{ReturnType: TypeVoid}, false)
	c.scope.define("send", &FuncType{ReturnType: TypeVoid}, false)
	c.scope.define("recv", &FuncType{ReturnType: TypeVoid}, false)

	// HTTP built-ins
	c.scope.define("http_get", &FuncType{ReturnType: TypeStr}, false)
	c.scope.define("http_status", &FuncType{ReturnType: TypeInt}, false)
	c.scope.define("http_serve", &FuncType{ReturnType: TypeVoid}, false)
	c.scope.define("http_post", &FuncType{ReturnType: TypeStr}, false)

	// TCP Socket built-ins
	c.scope.define("tcp_listen", &FuncType{ReturnType: TypeInt}, false)
	c.scope.define("tcp_accept", &FuncType{ReturnType: TypeInt}, false)
	c.scope.define("tcp_connect", &FuncType{ReturnType: TypeInt}, false)
	c.scope.define("tcp_send", &FuncType{ReturnType: TypeVoid}, false)
	c.scope.define("tcp_recv", &FuncType{ReturnType: TypeStr}, false)
	c.scope.define("tcp_close", &FuncType{ReturnType: TypeVoid}, false)

	// JSON built-ins
	c.scope.define("json_parse", &FuncType{ReturnType: TypeVoid}, false)
	c.scope.define("json_stringify", &FuncType{ReturnType: TypeStr}, false)

	// Filesystem built-ins
	c.scope.define("file_exists", &FuncType{ReturnType: TypeBool}, false)
	c.scope.define("list_dir", &FuncType{ReturnType: &ArrayType{ElementType: TypeStr}}, false)
	c.scope.define("mkdir", &FuncType{ReturnType: TypeVoid}, false)
	c.scope.define("remove", &FuncType{ReturnType: TypeVoid}, false)
	c.scope.define("path_join", &FuncType{ReturnType: TypeStr}, false)

	// Interface built-ins
	c.scope.define("implements", &FuncType{ReturnType: TypeBool}, false)

	// Type casting built-ins
	c.scope.define("cast_int", &FuncType{ReturnType: TypeInt}, false)
	c.scope.define("cast_float", &FuncType{ReturnType: TypeFloat}, false)
	c.scope.define("cast_str", &FuncType{ReturnType: TypeStr}, false)
	c.scope.define("cast_bool", &FuncType{ReturnType: TypeBool}, false)

	// Tuple built-ins
	c.scope.define("tuple", &FuncType{ReturnType: TypeVoid}, false)
	c.scope.define("first", &FuncType{ReturnType: TypeVoid}, false)
	c.scope.define("second", &FuncType{ReturnType: TypeVoid}, false)
	c.scope.define("unpack", &FuncType{ReturnType: TypeVoid}, false)
	c.scope.define("is_error", &FuncType{ReturnType: TypeBool}, false)

	// WaitGroup built-ins
	c.scope.define("wg_new", &FuncType{ReturnType: TypeInt}, false)
	c.scope.define("wg_add", &FuncType{ReturnType: TypeVoid}, false)
	c.scope.define("wg_done", &FuncType{ReturnType: TypeVoid}, false)
	c.scope.define("wg_wait", &FuncType{ReturnType: TypeVoid}, false)

	// Mutex built-ins
	c.scope.define("mutex_new", &FuncType{ReturnType: TypeInt}, false)
	c.scope.define("mutex_lock", &FuncType{ReturnType: TypeVoid}, false)
	c.scope.define("mutex_unlock", &FuncType{ReturnType: TypeVoid}, false)

	// Test built-ins
	c.scope.define("assert", &FuncType{ReturnType: TypeVoid}, false)
	c.scope.define("assert_eq", &FuncType{ReturnType: TypeVoid}, false)

	// Benchmark/timing built-ins
	c.scope.define("benchmark", &FuncType{ReturnType: TypeVoid}, false)
	c.scope.define("time_now", &FuncType{ReturnType: TypeInt}, false)
	c.scope.define("time_since", &FuncType{ReturnType: TypeInt}, false)

	// String functions
	c.scope.define("repeat", &FuncType{ReturnType: TypeStr}, false)
	c.scope.define("pad_left", &FuncType{ReturnType: TypeStr}, false)
	c.scope.define("pad_right", &FuncType{ReturnType: TypeStr}, false)
	c.scope.define("count", &FuncType{ReturnType: TypeInt}, false)
	c.scope.define("reverse", &FuncType{ReturnType: TypeVoid}, false)

	// Array functions
	c.scope.define("sort_arr", &FuncType{ReturnType: &ArrayType{ElementType: TypeVoid}}, false)
	c.scope.define("unique", &FuncType{ReturnType: &ArrayType{ElementType: TypeVoid}}, false)
	c.scope.define("flatten", &FuncType{ReturnType: &ArrayType{ElementType: TypeVoid}}, false)
	c.scope.define("zip", &FuncType{ReturnType: &ArrayType{ElementType: TypeVoid}}, false)
	c.scope.define("enumerate", &FuncType{ReturnType: &ArrayType{ElementType: TypeVoid}}, false)

	// Math functions
	c.scope.define("math_random", &FuncType{ReturnType: TypeFloat}, false)
	c.scope.define("math_rand_int", &FuncType{ReturnType: TypeInt}, false)
	c.scope.define("math_max_int", &FuncType{ReturnType: TypeInt}, false)
	c.scope.define("math_min_int", &FuncType{ReturnType: TypeInt}, false)

	// OS functions
	c.scope.define("os_setenv", &FuncType{ReturnType: TypeVoid}, false)
	c.scope.define("os_cwd", &FuncType{ReturnType: TypeStr}, false)
	c.scope.define("os_hostname", &FuncType{ReturnType: TypeStr}, false)
	c.scope.define("os_platform", &FuncType{ReturnType: TypeStr}, false)
	c.scope.define("os_arch", &FuncType{ReturnType: TypeStr}, false)

	// Pointer/memory
	c.scope.define("alloc", &FuncType{ReturnType: &ArrayType{ElementType: TypeInt}}, false)
	c.scope.define("sizeof", &FuncType{ReturnType: TypeInt}, false)

	// Variadic/format built-ins
	c.scope.define("type_of", &FuncType{ReturnType: TypeStr}, false)
	c.scope.define("format", &FuncType{ReturnType: TypeStr}, false)
	c.scope.define("printf", &FuncType{ReturnType: TypeVoid}, false)

	// Regex built-ins
	c.scope.define("regex_match", &FuncType{ReturnType: TypeBool}, false)
	c.scope.define("regex_find", &FuncType{ReturnType: TypeStr}, false)
	c.scope.define("regex_find_all", &FuncType{ReturnType: &ArrayType{ElementType: TypeStr}}, false)
	c.scope.define("regex_replace", &FuncType{ReturnType: TypeStr}, false)

	// Crypto built-ins
	c.scope.define("sha256", &FuncType{ReturnType: TypeStr}, false)
	c.scope.define("md5_hash", &FuncType{ReturnType: TypeStr}, false)

	// Encoding built-ins
	c.scope.define("base64_encode", &FuncType{ReturnType: TypeStr}, false)
	c.scope.define("base64_decode", &FuncType{ReturnType: TypeStr}, false)
	c.scope.define("url_encode", &FuncType{ReturnType: TypeStr}, false)
	c.scope.define("url_decode", &FuncType{ReturnType: TypeStr}, false)

	// Timer / Ticker built-ins
	c.scope.define("set_timeout", &FuncType{ReturnType: TypeVoid}, false)
	c.scope.define("set_interval", &FuncType{ReturnType: TypeVoid}, false)
	c.scope.define("clear_interval", &FuncType{ReturnType: TypeVoid}, false)

	// Sort with custom comparator
	c.scope.define("sort_by", &FuncType{ReturnType: &ArrayType{ElementType: TypeVoid}}, false)

	// Error wrapping built-ins
	c.scope.define("error_new", &FuncType{ReturnType: TypeVoid}, false)
	c.scope.define("error_wrap", &FuncType{ReturnType: TypeVoid}, false)
	c.scope.define("is_err", &FuncType{ReturnType: TypeBool}, false)

	// Template engine built-ins
	c.scope.define("template", &FuncType{ReturnType: TypeStr}, false)
	c.scope.define("template_file", &FuncType{ReturnType: TypeStr}, false)

	// Reflection / type inspection built-ins
	c.scope.define("fields", &FuncType{ReturnType: &ArrayType{ElementType: TypeStr}}, false)
	c.scope.define("get_field", &FuncType{ReturnType: TypeVoid}, false)
	c.scope.define("set_field", &FuncType{ReturnType: TypeVoid}, false)
	c.scope.define("type_name", &FuncType{ReturnType: TypeStr}, false)
	c.scope.define("is_null", &FuncType{ReturnType: TypeBool}, false)
	c.scope.define("is_int", &FuncType{ReturnType: TypeBool}, false)
	c.scope.define("is_float", &FuncType{ReturnType: TypeBool}, false)
	c.scope.define("is_string", &FuncType{ReturnType: TypeBool}, false)
	c.scope.define("is_bool", &FuncType{ReturnType: TypeBool}, false)
	c.scope.define("is_array", &FuncType{ReturnType: TypeBool}, false)
	c.scope.define("is_map", &FuncType{ReturnType: TypeBool}, false)
	c.scope.define("is_func", &FuncType{ReturnType: TypeBool}, false)
	c.scope.define("to_map", &FuncType{ReturnType: TypeVoid}, false)
	c.scope.define("copy_val", &FuncType{ReturnType: TypeVoid}, false)
}

// Check type-checks a program and returns any errors.
func (c *Checker) Check(program *parser.Program) []TypeError {
	// First pass: register all structs, enums, and function signatures
	for _, stmt := range program.Statements {
		c.registerDeclaration(stmt)
	}

	// Second pass: check all statements
	for _, stmt := range program.Statements {
		c.checkStatement(stmt)
	}

	return c.errors
}

func (c *Checker) errorf(pos lexer.Position, format string, args ...interface{}) {
	c.errors = append(c.errors, TypeError{
		Pos:     pos,
		Message: fmt.Sprintf(format, args...),
	})
}

func (c *Checker) resolve(name string) Type {
	return ResolveTypeName(name, c.structs, c.enums)
}

// --- First pass: register declarations ---

func (c *Checker) registerDeclaration(stmt parser.Statement) {
	switch s := stmt.(type) {
	case *parser.XuiructStatement:
		fields := make(map[string]Type)
		for _, f := range s.Fields {
			t := c.resolve(f.TypeName)
			if t == nil {
				c.errorf(s.Pos, "unknown type %q for field %q in struct %s", f.TypeName, f.Name, s.Name)
				t = TypeVoid
			}
			fields[f.Name] = t
		}
		c.structs[s.Name] = &StructType{Name: s.Name, Fields: fields}

	case *parser.XuenumStatement:
		c.enums[s.Name] = &EnumType{Name: s.Name, Variants: s.Variants}
		// Register enum variants in scope
		for _, v := range s.Variants {
			c.scope.define(v, c.enums[s.Name], false)
		}

	case *parser.XuenStatement:
		paramTypes := make([]Type, 0, len(s.Params))
		for _, p := range s.Params {
			if p.Name == "self" {
				continue
			}
			t := c.resolve(p.TypeName)
			if t == nil {
				c.errorf(s.Pos, "unknown type %q for parameter %q", p.TypeName, p.Name)
				t = TypeVoid
			}
			paramTypes = append(paramTypes, t)
		}
		retType := c.resolve(s.ReturnType)
		if retType == nil {
			retType = TypeVoid
		}
		ft := &FuncType{ParamTypes: paramTypes, ReturnType: retType}
		c.funcTypes[s.Name] = ft
		c.scope.define(s.Name, ft, false)

	case *parser.XuinterfaceStatement:
		// Interfaces are registered but no deeper checks needed
	}
}

// --- Second pass: check statements ---

func (c *Checker) checkStatement(stmt parser.Statement) {
	switch s := stmt.(type) {
	case *parser.XuetStatement:
		c.checkXuet(s)
	case *parser.XuiarStatement:
		c.checkXuiar(s)
	case *parser.AssignStatement:
		c.checkAssign(s)
	case *parser.XuenStatement:
		c.checkXuen(s)
	case *parser.XueturnStatement:
		c.checkXueturn(s)
	case *parser.XuifStatement:
		c.checkXuif(s)
	case *parser.XuiorStatement:
		c.checkXuior(s)
	case *parser.XuiorClassicStatement:
		c.checkXuiorClassic(s)
	case *parser.XuileStatement:
		c.checkXuile(s)
	case *parser.XuiructStatement:
		// already registered in first pass
	case *parser.XuimplStatement:
		c.checkXuimpl(s)
	case *parser.XuenumStatement:
		// already registered in first pass
	case *parser.XuimportStatement:
		// no-op
	case *parser.XuiatchStatement:
		c.checkXuiatch(s)
	case *parser.ExpressionStatement:
		c.checkExpression(s.Expr)
	case *parser.BlockStatement:
		c.checkBlock(s)
	case *parser.XueakStatement, *parser.XuitinueStatement:
		// valid in loops, no type to check
	case *parser.XuinterfaceStatement:
		// registered in first pass, no body to check
	case *parser.XudeferStatement:
		c.checkExpression(s.Call)
	case *parser.XuselectStatement:
		for _, sc := range s.Cases {
			if sc.Channel != nil {
				c.checkExpression(sc.Channel)
			}
			if sc.Body != nil {
				c.checkBlock(sc.Body)
			}
		}
	}
}

func (c *Checker) checkBlock(block *parser.BlockStatement) {
	oldScope := c.scope
	c.scope = newScope(oldScope)
	for _, stmt := range block.Statements {
		c.checkStatement(stmt)
	}
	c.scope = oldScope
}

func (c *Checker) checkXuet(s *parser.XuetStatement) {
	valType := c.checkExpression(s.Value)

	if s.TypeName != "" {
		declared := c.resolve(s.TypeName)
		if declared == nil {
			c.errorf(s.Pos, "unknown type %q", s.TypeName)
		} else if valType != nil && !AssignableTo(valType, declared) {
			c.errorf(s.Pos, "cannot assign %s to %s", valType.TypeName(), declared.TypeName())
		}
		c.scope.define(s.Name, declared, false)
	} else {
		if valType == nil {
			valType = TypeVoid
		}
		c.scope.define(s.Name, valType, false)
	}
}

func (c *Checker) checkXuiar(s *parser.XuiarStatement) {
	valType := c.checkExpression(s.Value)

	if s.TypeName != "" {
		declared := c.resolve(s.TypeName)
		if declared == nil {
			c.errorf(s.Pos, "unknown type %q", s.TypeName)
		} else if valType != nil && !AssignableTo(valType, declared) {
			c.errorf(s.Pos, "cannot assign %s to %s", valType.TypeName(), declared.TypeName())
		}
		c.scope.define(s.Name, declared, true)
	} else {
		if valType == nil {
			valType = TypeVoid
		}
		c.scope.define(s.Name, valType, true)
	}
}

func (c *Checker) checkAssign(s *parser.AssignStatement) {
	valType := c.checkExpression(s.Value)

	switch target := s.Target.(type) {
	case *parser.Identifier:
		existingType, ok := c.scope.lookup(target.Value)
		if !ok {
			c.errorf(s.Pos, "undefined variable %q", target.Value)
			return
		}
		if !c.scope.isMutable(target.Value) {
			c.errorf(s.Pos, "cannot assign to immutable variable %q (declared with xuet)", target.Value)
			return
		}
		if valType != nil && existingType != nil && !AssignableTo(valType, existingType) {
			c.errorf(s.Pos, "cannot assign %s to %s", valType.TypeName(), existingType.TypeName())
		}
	case *parser.MemberExpression:
		c.checkExpression(target)
	case *parser.IndexExpression:
		c.checkExpression(target)
	}
}

func (c *Checker) checkXuen(s *parser.XuenStatement) {
	// Build function type if not already registered (nested functions)
	ft := c.funcTypes[s.Name]
	if ft == nil {
		paramTypes := make([]Type, 0)
		for _, p := range s.Params {
			if p.Name == "self" { continue }
			t := c.resolve(p.TypeName)
			if t == nil { t = TypeVoid }
			paramTypes = append(paramTypes, t)
		}
		retType := c.resolve(s.ReturnType)
		if retType == nil { retType = TypeVoid }
		ft = &FuncType{ParamTypes: paramTypes, ReturnType: retType}
		c.funcTypes[s.Name] = ft
	}
	// Register in current scope (important for nested functions)
	c.scope.define(s.Name, ft, false)

	oldScope := c.scope
	oldFn := c.currentFn
	c.scope = newScope(oldScope)
	c.currentFn = ft

	paramIdx := 0
	for _, p := range s.Params {
		if p.Name == "self" {
			// self type will be inferred from context
			c.scope.define("self", TypeVoid, true)
			continue
		}
		t := c.resolve(p.TypeName)
		if t == nil {
			t = TypeVoid
		}
		c.scope.define(p.Name, t, true)
		paramIdx++
	}

	for _, stmt := range s.Body.Statements {
		c.checkStatement(stmt)
	}

	c.scope = oldScope
	c.currentFn = oldFn
}

func (c *Checker) checkXueturn(s *parser.XueturnStatement) {
	if s.Value == nil {
		return
	}

	valType := c.checkExpression(s.Value)

	if c.currentFn != nil && c.currentFn.ReturnType != nil && !c.currentFn.ReturnType.Equals(TypeVoid) {
		if valType != nil && !AssignableTo(valType, c.currentFn.ReturnType) {
			c.errorf(s.Pos, "cannot return %s from function expecting %s",
				valType.TypeName(), c.currentFn.ReturnType.TypeName())
		}
	}
}

func (c *Checker) checkXuif(s *parser.XuifStatement) {
	condType := c.checkExpression(s.Condition)
	if condType != nil && !condType.Equals(TypeBool) && !IsNumeric(condType) && !condType.Equals(TypeVoid) {
		c.errorf(s.Pos, "condition must be bool, got %s", condType.TypeName())
	}

	c.checkBlock(s.Consequence)

	if s.Alternative != nil {
		switch alt := s.Alternative.(type) {
		case *parser.BlockStatement:
			c.checkBlock(alt)
		case *parser.XuifStatement:
			c.checkXuif(alt)
		}
	}
}

func (c *Checker) checkXuior(s *parser.XuiorStatement) {
	iterType := c.checkExpression(s.Iterable)

	oldScope := c.scope
	c.scope = newScope(oldScope)

	// Infer loop variable type from iterable
	var varType Type
	if iterType != nil {
		switch t := iterType.(type) {
		case *ArrayType:
			varType = t.ElementType
		case *RangeType:
			varType = TypeInt
		case *PrimitiveType:
			if t.Name == "str" {
				varType = TypeChar
			}
		}
	}
	if varType == nil {
		varType = TypeVoid
	}
	c.scope.define(s.Variable, varType, false)

	for _, stmt := range s.Body.Statements {
		c.checkStatement(stmt)
	}
	c.scope = oldScope
}

func (c *Checker) checkXuiorClassic(s *parser.XuiorClassicStatement) {
	oldScope := c.scope
	c.scope = newScope(oldScope)
	if s.Init != nil {
		c.checkStatement(s.Init)
	}
	if s.Condition != nil {
		c.checkExpression(s.Condition)
	}
	if s.Post != nil {
		c.checkStatement(s.Post)
	}
	c.checkBlock(s.Body)
	c.scope = oldScope
}

func (c *Checker) checkXuile(s *parser.XuileStatement) {
	condType := c.checkExpression(s.Condition)
	if condType != nil && !condType.Equals(TypeBool) && !IsNumeric(condType) && !condType.Equals(TypeVoid) {
		c.errorf(s.Pos, "condition must be bool, got %s", condType.TypeName())
	}
	c.checkBlock(s.Body)
}

func (c *Checker) checkXuimpl(s *parser.XuimplStatement) {
	if _, ok := c.structs[s.Name]; !ok {
		c.errorf(s.Pos, "cannot implement methods for undefined struct %q", s.Name)
		return
	}

	for _, method := range s.Methods {
		// Register method then check it
		paramTypes := make([]Type, 0)
		for _, p := range method.Params {
			if p.Name == "self" {
				continue
			}
			t := c.resolve(p.TypeName)
			if t == nil {
				t = TypeVoid
			}
			paramTypes = append(paramTypes, t)
		}
		retType := c.resolve(method.ReturnType)
		if retType == nil {
			retType = TypeVoid
		}

		c.funcTypes[s.Name+"."+method.Name] = &FuncType{
			ParamTypes: paramTypes,
			ReturnType: retType,
		}

		oldScope := c.scope
		oldFn := c.currentFn
		c.scope = newScope(oldScope)
		c.currentFn = &FuncType{ParamTypes: paramTypes, ReturnType: retType}

		// Define self and params
		c.scope.define("self", c.structs[s.Name], true)
		paramIdx := 0
		for _, p := range method.Params {
			if p.Name == "self" {
				continue
			}
			t := c.resolve(p.TypeName)
			if t == nil {
				t = TypeVoid
			}
			c.scope.define(p.Name, t, true)
			paramIdx++
		}

		for _, stmt := range method.Body.Statements {
			c.checkStatement(stmt)
		}

		c.scope = oldScope
		c.currentFn = oldFn
	}
}

func (c *Checker) checkXuiatch(s *parser.XuiatchStatement) {
	c.checkExpression(s.Value)
	for _, arm := range s.Arms {
		// Skip wildcard "_"
		if ident, ok := arm.Pattern.(*parser.Identifier); ok && ident.Value == "_" {
			c.checkStatement(arm.Body)
			continue
		}
		c.checkExpression(arm.Pattern)
		c.checkStatement(arm.Body)
	}
}

// --- Expression type checking ---

func (c *Checker) checkExpression(expr parser.Expression) Type {
	switch e := expr.(type) {
	case *parser.IntegerLiteral:
		return TypeInt
	case *parser.FloatLiteral:
		return TypeFloat
	case *parser.StringLiteral:
		return TypeStr
	case *parser.CharLiteral:
		return TypeChar
	case *parser.BoolLiteral:
		return TypeBool
	case *parser.NullLiteral:
		return TypeNull
	case *parser.Identifier:
		return c.checkIdentifier(e)
	case *parser.PrefixExpression:
		return c.checkPrefix(e)
	case *parser.InfixExpression:
		return c.checkInfix(e)
	case *parser.CallExpression:
		return c.checkCall(e)
	case *parser.MemberExpression:
		return c.checkMember(e)
	case *parser.IndexExpression:
		return c.checkIndex(e)
	case *parser.ArrayLiteral:
		return c.checkArray(e)
	case *parser.RangeExpression:
		return c.checkRange(e)
	case *parser.StructLiteral:
		return c.checkStructLiteral(e)
	default:
		return nil
	}
}

func (c *Checker) checkIdentifier(e *parser.Identifier) Type {
	t, ok := c.scope.lookup(e.Value)
	if !ok {
		c.errorf(e.Pos, "undefined variable %q", e.Value)
		return nil
	}
	return t
}

func (c *Checker) checkPrefix(e *parser.PrefixExpression) Type {
	rightType := c.checkExpression(e.Right)
	if rightType == nil {
		return nil
	}

	switch e.Operator {
	case "-":
		if !IsNumeric(rightType) {
			c.errorf(e.Pos, "cannot negate %s", rightType.TypeName())
			return nil
		}
		return rightType
	case "!":
		return TypeBool
	}
	return nil
}

func (c *Checker) checkInfix(e *parser.InfixExpression) Type {
	leftType := c.checkExpression(e.Left)
	rightType := c.checkExpression(e.Right)

	if leftType == nil || rightType == nil {
		return nil
	}

	switch e.Operator {
	case "+":
		// String concatenation
		if leftType.Equals(TypeStr) || rightType.Equals(TypeStr) {
			return TypeStr
		}
		return c.checkNumericOp(e, leftType, rightType)
	case "-", "*", "/", "%":
		return c.checkNumericOp(e, leftType, rightType)
	case "==", "!=":
		return TypeBool
	case "<", ">", "<=", ">=":
		if !IsNumeric(leftType) || !IsNumeric(rightType) {
			c.errorf(e.Pos, "cannot compare %s and %s", leftType.TypeName(), rightType.TypeName())
		}
		return TypeBool
	case "&&", "||":
		return TypeBool
	}

	return nil
}

func (c *Checker) checkNumericOp(e *parser.InfixExpression, left, right Type) Type {
	// Allow void (unknown/dynamic) in arithmetic — common with map values
	if left.Equals(TypeVoid) || right.Equals(TypeVoid) {
		return TypeVoid
	}
	if !IsNumeric(left) || !IsNumeric(right) {
		c.errorf(e.Pos, "cannot apply %s to %s and %s", e.Operator, left.TypeName(), right.TypeName())
		return nil
	}
	// Float wins in mixed operations
	if IsFloat(left) || IsFloat(right) {
		return TypeFloat
	}
	return TypeInt
}

func (c *Checker) checkCall(e *parser.CallExpression) Type {
	// Method call
	if _, ok := e.Function.(*parser.MemberExpression); ok {
		// For method calls, just check args and return void for now
		for _, arg := range e.Arguments {
			c.checkExpression(arg)
		}
		return TypeVoid
	}

	fnType := c.checkExpression(e.Function)
	if fnType == nil {
		return nil
	}

	// Check args
	for _, arg := range e.Arguments {
		c.checkExpression(arg)
	}

	if ft, ok := fnType.(*FuncType); ok {
		return ft.ReturnType
	}

	return TypeVoid
}

func (c *Checker) checkMember(e *parser.MemberExpression) Type {
	objType := c.checkExpression(e.Object)
	if objType == nil {
		return nil
	}

	if st, ok := objType.(*StructType); ok {
		fieldType, ok := st.Fields[e.Member]
		if !ok {
			c.errorf(e.Pos, "struct %q has no field %q", st.Name, e.Member)
			return nil
		}
		return fieldType
	}

	// .length on arrays/strings
	if e.Member == "length" {
		if objType.Equals(TypeStr) || isArrayType(objType) {
			return TypeInt
		}
	}

	c.errorf(e.Pos, "cannot access member %q on %s", e.Member, objType.TypeName())
	return nil
}

func (c *Checker) checkIndex(e *parser.IndexExpression) Type {
	leftType := c.checkExpression(e.Left)
	idxType := c.checkExpression(e.Index)

	if leftType == nil {
		// Unknown type — allow indexing (could be map or dynamic)
		return TypeVoid
	}

	if arr, ok := leftType.(*ArrayType); ok {
		if idxType != nil && !IsInteger(idxType) {
			c.errorf(e.Pos, "array index must be int, got %s", idxType.TypeName())
		}
		return arr.ElementType
	}
	if leftType.Equals(TypeStr) {
		return TypeChar
	}

	// Allow indexing on void/unknown types (maps, dynamic values)
	if leftType.Equals(TypeVoid) {
		return TypeVoid
	}

	// Unknown container type — be permissive
	_ = idxType
	return nil
}

func (c *Checker) checkArray(e *parser.ArrayLiteral) Type {
	if len(e.Elements) == 0 {
		return &ArrayType{ElementType: TypeVoid}
	}

	firstType := c.checkExpression(e.Elements[0])
	for i := 1; i < len(e.Elements); i++ {
		elemType := c.checkExpression(e.Elements[i])
		if firstType != nil && elemType != nil && !AssignableTo(elemType, firstType) {
			c.errorf(e.Pos, "array element type mismatch: expected %s, got %s",
				firstType.TypeName(), elemType.TypeName())
		}
	}

	if firstType == nil {
		firstType = TypeVoid
	}
	return &ArrayType{ElementType: firstType}
}

func (c *Checker) checkRange(e *parser.RangeExpression) Type {
	startType := c.checkExpression(e.Start)
	endType := c.checkExpression(e.End)

	if startType != nil && !IsInteger(startType) {
		c.errorf(e.Pos, "range start must be int, got %s", startType.TypeName())
	}
	if endType != nil && !IsInteger(endType) {
		c.errorf(e.Pos, "range end must be int, got %s", endType.TypeName())
	}

	return &RangeType{}
}

func (c *Checker) checkStructLiteral(e *parser.StructLiteral) Type {
	st, ok := c.structs[e.Name]
	if !ok {
		c.errorf(e.Pos, "undefined struct %q", e.Name)
		return nil
	}

	for _, f := range e.Fields {
		valType := c.checkExpression(f.Value)
		fieldType, ok := st.Fields[f.Name]
		if !ok {
			c.errorf(e.Pos, "struct %q has no field %q", e.Name, f.Name)
			continue
		}
		if valType != nil && !AssignableTo(valType, fieldType) {
			c.errorf(e.Pos, "cannot assign %s to field %q of type %s",
				valType.TypeName(), f.Name, fieldType.TypeName())
		}
	}

	return st
}

func isArrayType(t Type) bool {
	_, ok := t.(*ArrayType)
	return ok
}
