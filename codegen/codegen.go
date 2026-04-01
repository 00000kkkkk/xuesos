package codegen

import (
	_ "embed"
	"fmt"
	"strings"

	"github.com/00000kkkkk/xusesosplusplus/parser"
)

//go:embed runtime/runtime.c
var runtimeC string

// CCodegen generates C code from the AST.
type CCodegen struct {
	output      strings.Builder
	indent      int
	structDefs  map[string]*parser.XuiructStatement
	implDefs    map[string]*parser.XuimplStatement
	varTypes    map[string]string // variable name -> C type
	tempCounter int
}

// New creates a new C code generator.
func New() *CCodegen {
	return &CCodegen{
		structDefs: make(map[string]*parser.XuiructStatement),
		implDefs:   make(map[string]*parser.XuimplStatement),
		varTypes:   make(map[string]string),
	}
}

// Generate produces C code from a parsed program.
func (g *CCodegen) Generate(program *parser.Program) string {
	// First pass: collect struct and impl definitions
	for _, stmt := range program.Statements {
		switch s := stmt.(type) {
		case *parser.XuiructStatement:
			g.structDefs[s.Name] = s
		case *parser.XuimplStatement:
			g.implDefs[s.Name] = s
		}
	}

	// Emit runtime library (includes all headers, types, and helpers)
	// The runtime provides: XppString*, XppArray*, XppMap*, xpp_print_*,
	// xpp_string_new/concat/eq, error handling globals, etc.
	g.writeln(runtimeC)
	g.writeln("")

	// Forward declare all functions
	for _, stmt := range program.Statements {
		if fn, ok := stmt.(*parser.XuenStatement); ok {
			g.emitFuncForwardDecl(fn)
		}
	}
	// Forward declare methods
	for name, impl := range g.implDefs {
		for _, method := range impl.Methods {
			g.emitMethodForwardDecl(name, method)
		}
	}
	g.writeln("")

	// Emit structs
	for _, stmt := range program.Statements {
		if s, ok := stmt.(*parser.XuiructStatement); ok {
			g.emitStruct(s)
		}
	}

	// Emit all functions and impl methods
	for _, stmt := range program.Statements {
		switch s := stmt.(type) {
		case *parser.XuenStatement:
			g.emitFunction(s)
		case *parser.XuimplStatement:
			g.emitImpl(s)
		}
	}

	// Emit main wrapper
	g.emitMainWrapper(program)

	return g.output.String()
}

func (g *CCodegen) write(s string) {
	g.output.WriteString(s)
}

func (g *CCodegen) writeln(s string) {
	g.output.WriteString(s)
	g.output.WriteString("\n")
}

func (g *CCodegen) writeIndent() {
	for i := 0; i < g.indent; i++ {
		g.write("    ")
	}
}

func (g *CCodegen) emitLine(format string, args ...interface{}) {
	g.writeIndent()
	g.writeln(fmt.Sprintf(format, args...))
}

func (g *CCodegen) newTemp() string {
	g.tempCounter++
	return fmt.Sprintf("_tmp%d", g.tempCounter)
}

// --- Type mapping ---

func (g *CCodegen) mapType(typeName string) string {
	switch typeName {
	case "int", "int64":
		return "int64_t"
	case "int8":
		return "int8_t"
	case "int16":
		return "int16_t"
	case "int32":
		return "int32_t"
	case "uint":
		return "uint64_t"
	case "float", "float32":
		return "double"
	case "bool":
		return "bool"
	case "str":
		return "XppString*"
	case "[]str":
		return "XppArray*"
	case "char":
		return "char"
	case "byte":
		return "uint8_t"
	case "void", "":
		return "void"
	default:
		// Struct type
		return "struct " + typeName
	}
}

// --- Forward declarations ---

func (g *CCodegen) emitFuncForwardDecl(fn *parser.XuenStatement) {
	if fn.Name == "main" {
		return // main is handled separately
	}
	retType := g.mapType(fn.ReturnType)
	params := g.buildParamList(fn.Params)
	g.writeln(fmt.Sprintf("%s xpp_%s(%s);", retType, fn.Name, params))
}

func (g *CCodegen) emitMethodForwardDecl(structName string, method *parser.XuenStatement) {
	retType := g.mapType(method.ReturnType)
	params := g.buildMethodParamList(structName, method.Params)
	g.writeln(fmt.Sprintf("%s xpp_%s_%s(%s);", retType, structName, method.Name, params))
}

func (g *CCodegen) buildParamList(params []parser.Parameter) string {
	if len(params) == 0 {
		return "void"
	}
	var parts []string
	for _, p := range params {
		parts = append(parts, fmt.Sprintf("%s %s", g.mapType(p.TypeName), p.Name))
	}
	return strings.Join(parts, ", ")
}

func (g *CCodegen) buildMethodParamList(structName string, params []parser.Parameter) string {
	var parts []string
	for _, p := range params {
		if p.Name == "self" {
			parts = append(parts, fmt.Sprintf("struct %s *self", structName))
		} else {
			parts = append(parts, fmt.Sprintf("%s %s", g.mapType(p.TypeName), p.Name))
		}
	}
	if len(parts) == 0 {
		return "void"
	}
	return strings.Join(parts, ", ")
}

// --- Struct emission ---

func (g *CCodegen) emitStruct(s *parser.XuiructStatement) {
	g.writeln(fmt.Sprintf("struct %s {", s.Name))
	g.indent++
	for _, f := range s.Fields {
		g.emitLine("%s %s;", g.mapType(f.TypeName), f.Name)
	}
	g.indent--
	g.writeln("};")
	g.writeln("")
}

// --- Function emission ---

func (g *CCodegen) emitFunction(fn *parser.XuenStatement) {
	if fn.Name == "main" {
		return // handled by emitMainWrapper
	}
	retType := g.mapType(fn.ReturnType)
	params := g.buildParamList(fn.Params)
	g.writeln(fmt.Sprintf("%s xpp_%s(%s) {", retType, fn.Name, params))
	g.indent++
	g.emitBlock(fn.Body)
	g.indent--
	g.writeln("}")
	g.writeln("")
}

func (g *CCodegen) emitImpl(s *parser.XuimplStatement) {
	for _, method := range s.Methods {
		retType := g.mapType(method.ReturnType)
		params := g.buildMethodParamList(s.Name, method.Params)
		g.writeln(fmt.Sprintf("%s xpp_%s_%s(%s) {", retType, s.Name, method.Name, params))
		g.indent++
		g.emitBlock(method.Body)
		g.indent--
		g.writeln("}")
		g.writeln("")
	}
}

func (g *CCodegen) emitMainWrapper(program *parser.Program) {
	// Find the user's main function
	var mainFn *parser.XuenStatement
	var topLevel []parser.Statement

	for _, stmt := range program.Statements {
		if fn, ok := stmt.(*parser.XuenStatement); ok && fn.Name == "main" {
			mainFn = fn
		} else {
			switch stmt.(type) {
			case *parser.XuiructStatement, *parser.XuimplStatement, *parser.XuenumStatement,
				*parser.XuenStatement, *parser.XuimportStatement:
				// Skip declarations — already emitted
			default:
				topLevel = append(topLevel, stmt)
			}
		}
	}

	g.writeln("int main(void) {")
	g.indent++

	// Emit top-level statements
	for _, stmt := range topLevel {
		g.emitStatement(stmt)
	}

	// Call user's main if it exists
	if mainFn != nil {
		g.emitBlock(mainFn.Body)
	}

	g.emitLine("return 0;")
	g.indent--
	g.writeln("}")
}

// --- Statement emission ---

func (g *CCodegen) emitBlock(block *parser.BlockStatement) {
	for _, stmt := range block.Statements {
		g.emitStatement(stmt)
	}
}

func (g *CCodegen) emitStatement(stmt parser.Statement) {
	switch s := stmt.(type) {
	case *parser.XuetStatement:
		g.emitXuet(s)
	case *parser.XuiarStatement:
		g.emitXuiar(s)
	case *parser.AssignStatement:
		g.emitAssign(s)
	case *parser.XueturnStatement:
		g.emitXueturn(s)
	case *parser.XueakStatement:
		g.emitLine("break;")
	case *parser.XuitinueStatement:
		g.emitLine("continue;")
	case *parser.XuifStatement:
		g.emitXuif(s)
	case *parser.XuiorStatement:
		g.emitXuior(s)
	case *parser.XuileStatement:
		g.emitXuile(s)
	case *parser.XuiatchStatement:
		g.emitXuiatch(s)
	case *parser.TryStatement:
		g.emitTry(s)
	case *parser.ExpressionStatement:
		g.writeIndent()
		g.emitExpression(s.Expr)
		g.writeln(";")
	case *parser.BlockStatement:
		g.emitLine("{")
		g.indent++
		g.emitBlock(s)
		g.indent--
		g.emitLine("}")
	}
}

func (g *CCodegen) emitXuet(s *parser.XuetStatement) {
	cType := g.inferCType(s.Value, s.TypeName)
	g.varTypes[s.Name] = cType
	g.writeIndent()
	g.write(fmt.Sprintf("const %s %s = ", cType, s.Name))
	g.emitExpression(s.Value)
	g.writeln(";")
}

func (g *CCodegen) emitXuiar(s *parser.XuiarStatement) {
	cType := g.inferCType(s.Value, s.TypeName)
	g.varTypes[s.Name] = cType
	g.writeIndent()
	g.write(fmt.Sprintf("%s %s = ", cType, s.Name))
	g.emitExpression(s.Value)
	g.writeln(";")
}

func (g *CCodegen) emitAssign(s *parser.AssignStatement) {
	g.writeIndent()
	g.emitExpression(s.Target)
	g.write(" = ")
	g.emitExpression(s.Value)
	g.writeln(";")
}

func (g *CCodegen) emitXueturn(s *parser.XueturnStatement) {
	if s.Value == nil {
		g.emitLine("return;")
		return
	}
	g.writeIndent()
	g.write("return ")
	g.emitExpression(s.Value)
	g.writeln(";")
}

func (g *CCodegen) emitXuif(s *parser.XuifStatement) {
	g.writeIndent()
	g.write("if (")
	g.emitExpression(s.Condition)
	g.writeln(") {")
	g.indent++
	g.emitBlock(s.Consequence)
	g.indent--

	if s.Alternative != nil {
		switch alt := s.Alternative.(type) {
		case *parser.BlockStatement:
			g.emitLine("} else {")
			g.indent++
			g.emitBlock(alt)
			g.indent--
			g.emitLine("}")
		case *parser.XuifStatement:
			g.writeIndent()
			g.write("} else ")
			g.emitXuif(alt)
			return
		}
	} else {
		g.emitLine("}")
	}
}

func (g *CCodegen) emitXuior(s *parser.XuiorStatement) {
	// Check if iterable is a range expression
	if rng, ok := s.Iterable.(*parser.RangeExpression); ok {
		g.writeIndent()
		g.write(fmt.Sprintf("for (int64_t %s = ", s.Variable))
		g.emitExpression(rng.Start)
		g.write(fmt.Sprintf("; %s < ", s.Variable))
		g.emitExpression(rng.End)
		g.write(fmt.Sprintf("; %s++", s.Variable))
		g.writeln(") {")
	} else {
		// For array iteration, generate indexed loop
		tmp := g.newTemp()
		g.writeIndent()
		g.write(fmt.Sprintf("for (int64_t %s = 0; %s < /* len */; %s++", tmp, tmp, tmp))
		g.writeln(") {")
		g.indent++
		g.emitLine("// TODO: array iteration")
		g.indent--
	}
	g.indent++
	g.emitBlock(s.Body)
	g.indent--
	g.emitLine("}")
}

func (g *CCodegen) emitXuile(s *parser.XuileStatement) {
	g.writeIndent()
	g.write("while (")
	g.emitExpression(s.Condition)
	g.writeln(") {")
	g.indent++
	g.emitBlock(s.Body)
	g.indent--
	g.emitLine("}")
}

func (g *CCodegen) emitXuiatch(s *parser.XuiatchStatement) {
	for i, arm := range s.Arms {
		isDefault := false
		if ident, ok := arm.Pattern.(*parser.Identifier); ok && ident.Value == "_" {
			isDefault = true
		}

		if isDefault {
			if i > 0 {
				g.writeIndent()
				g.writeln("} else {")
			} else {
				g.emitLine("{")
			}
		} else {
			g.writeIndent()
			if i > 0 {
				g.write("} else if (")
			} else {
				g.write("if (")
			}
			// Use xpp_string_eq for string patterns, == for integers
			if _, ok := arm.Pattern.(*parser.StringLiteral); ok {
				sl := arm.Pattern.(*parser.StringLiteral)
				g.write("xpp_string_eq(")
				g.emitExpression(s.Value)
				g.write(fmt.Sprintf(", xpp_string_new(%q))", sl.Value))
			} else {
				g.write("(")
				g.emitExpression(s.Value)
				g.write(") == (")
				g.emitExpression(arm.Pattern)
				g.write(")")
			}
			g.writeln(") {")
		}

		g.indent++
		switch body := arm.Body.(type) {
		case *parser.BlockStatement:
			g.emitBlock(body)
		case *parser.ExpressionStatement:
			g.writeIndent()
			g.emitExpression(body.Expr)
			g.writeln(";")
		default:
			g.emitStatement(arm.Body)
		}
		g.indent--
	}
	g.emitLine("}")
}

func (g *CCodegen) emitTry(s *parser.TryStatement) {
	// Emit try body directly, with error checking via the global flag
	g.emitLine("_xpp_has_error = 0;")
	g.emitLine("{")
	g.indent++
	g.emitBlock(s.Body)
	g.indent--
	g.emitLine("}")
	g.writeIndent()
	g.writeln("if (_xpp_has_error) {")
	g.indent++
	g.emitLine("const char* %s = _xpp_error_msg;", s.CatchVar)
	g.emitBlock(s.CatchBody)
	g.indent--
	g.emitLine("}")
}

// --- Expression emission ---

func (g *CCodegen) emitExpression(expr parser.Expression) {
	switch e := expr.(type) {
	case *parser.IntegerLiteral:
		g.write(fmt.Sprintf("%dLL", e.Value))
	case *parser.FloatLiteral:
		g.write(fmt.Sprintf("%s", e.Raw))
	case *parser.StringLiteral:
		g.write(fmt.Sprintf("xpp_string_new(%q)", e.Value))
	case *parser.CharLiteral:
		g.write(fmt.Sprintf("'%c'", e.Value))
	case *parser.BoolLiteral:
		if e.Value {
			g.write("true")
		} else {
			g.write("false")
		}
	case *parser.NullLiteral:
		g.write("NULL")
	case *parser.Identifier:
		g.write(e.Value)
	case *parser.PrefixExpression:
		g.write("(")
		g.write(e.Operator)
		g.emitExpression(e.Right)
		g.write(")")
	case *parser.InfixExpression:
		g.emitInfix(e)
	case *parser.CallExpression:
		g.emitCall(e)
	case *parser.MemberExpression:
		g.emitMember(e)
	case *parser.IndexExpression:
		g.emitExpression(e.Left)
		g.write("[")
		g.emitExpression(e.Index)
		g.write("]")
	case *parser.ArrayLiteral:
		g.write("{")
		for i, elem := range e.Elements {
			if i > 0 {
				g.write(", ")
			}
			g.emitExpression(elem)
		}
		g.write("}")
	case *parser.RangeExpression:
		// Range as expression — just emit start (used in for loops directly)
		g.emitExpression(e.Start)
	case *parser.StructLiteral:
		g.write(fmt.Sprintf("(struct %s){", e.Name))
		for i, f := range e.Fields {
			if i > 0 {
				g.write(", ")
			}
			g.write(fmt.Sprintf(".%s = ", f.Name))
			g.emitExpression(f.Value)
		}
		g.write("}")
	case *parser.MapLiteral:
		g.emitMapLiteral(e)
	case *parser.InterpolatedString:
		g.emitInterpolatedString(e)
	case *parser.LambdaExpression:
		g.emitLambda(e)
	case *parser.ThrowExpression:
		g.emitThrow(e)
	case *parser.AddressOfExpression:
		g.write("&")
		g.emitExpression(e.Value)
	case *parser.DerefExpression:
		g.write("(*")
		g.emitExpression(e.Value)
		g.write(")")
	}
}

func (g *CCodegen) emitInfix(e *parser.InfixExpression) {
	leftIsStr := g.isStringTypedExpr(e.Left)
	rightIsStr := g.isStringTypedExpr(e.Right)

	// String comparison
	if e.Operator == "==" || e.Operator == "!=" {
		if leftIsStr || rightIsStr {
			if e.Operator == "!=" {
				g.write("!")
			}
			g.write("xpp_string_eq(")
			g.emitExpression(e.Left)
			g.write(", ")
			g.emitExpression(e.Right)
			g.write(")")
			return
		}
	}

	// String concatenation using runtime library
	if e.Operator == "+" && (leftIsStr || rightIsStr) {
		g.write("xpp_string_concat(")
		g.emitExpression(e.Left)
		g.write(", ")
		g.emitExpression(e.Right)
		g.write(")")
		return
	}

	g.write("(")
	g.emitExpression(e.Left)
	g.write(" ")

	// Map operators
	switch e.Operator {
	case "&&":
		g.write("&&")
	case "||":
		g.write("||")
	default:
		g.write(e.Operator)
	}

	g.write(" ")
	g.emitExpression(e.Right)
	g.write(")")
}

func (g *CCodegen) emitCall(e *parser.CallExpression) {
	// Method call: obj.method(args)
	if member, ok := e.Function.(*parser.MemberExpression); ok {
		// Get struct type from member object
		if ident, ok := member.Object.(*parser.Identifier); ok {
			if def, exists := g.structDefs[ident.Value]; exists {
				_ = def
			}
		}
		// Emit as: xpp_StructName_method(&obj, args...)
		g.write("xpp_")
		// For simplicity, just emit the method call pattern
		g.emitExpression(member.Object)
		g.write("_")
		g.write(member.Member)
		g.write("(")
		g.write("&")
		g.emitExpression(member.Object)
		for _, arg := range e.Arguments {
			g.write(", ")
			g.emitExpression(arg)
		}
		g.write(")")
		return
	}

	// Regular function call
	if ident, ok := e.Function.(*parser.Identifier); ok {
		// Map print to appropriate C function
		if ident.Value == "print" || ident.Value == "println" {
			g.emitPrintCall(e.Arguments)
			return
		}
		// User functions get xpp_ prefix
		g.write("xpp_" + ident.Value)
	} else {
		g.emitExpression(e.Function)
	}

	g.write("(")
	for i, arg := range e.Arguments {
		if i > 0 {
			g.write(", ")
		}
		g.emitExpression(arg)
	}
	g.write(")")
}

func (g *CCodegen) emitPrintCall(args []parser.Expression) {
	if len(args) == 0 {
		g.write(`printf("\n")`)
		return
	}

	// Single argument: use xpp_print_* runtime functions
	if len(args) == 1 {
		arg := args[0]
		switch e := arg.(type) {
		case *parser.StringLiteral:
			g.write(fmt.Sprintf("xpp_print_string(xpp_string_new(%q))", e.Value))
		case *parser.IntegerLiteral:
			g.write(fmt.Sprintf("xpp_print_int(%dLL)", e.Value))
		case *parser.FloatLiteral:
			g.write(fmt.Sprintf("xpp_print_float(%s)", e.Raw))
		case *parser.BoolLiteral:
			if e.Value {
				g.write("xpp_print_bool(true)")
			} else {
				g.write("xpp_print_bool(false)")
			}
		case *parser.Identifier:
			if t, ok := g.varTypes[e.Value]; ok && t == "XppString*" {
				g.write(fmt.Sprintf("xpp_print_string(%s)", e.Value))
			} else if t == "double" {
				g.write(fmt.Sprintf("xpp_print_float(%s)", e.Value))
			} else if t == "bool" {
				g.write(fmt.Sprintf("xpp_print_bool(%s)", e.Value))
			} else if t == "char" {
				g.write(fmt.Sprintf("xpp_print_char(%s)", e.Value))
			} else {
				g.write(fmt.Sprintf("xpp_print_int(%s)", e.Value))
			}
		case *parser.InterpolatedString:
			g.emitPrintInterpolated(e)
		default:
			// Check if the expression looks like a string type
			if g.isStringTypedExpr(arg) {
				g.write("xpp_print_string(")
				g.emitExpression(arg)
				g.write(")")
			} else {
				g.write("xpp_print_int(")
				g.emitExpression(arg)
				g.write(")")
			}
		}
		return
	}

	// Multiple args: print space-separated, detect types
	g.write(`printf("`)
	for i, arg := range args {
		if i > 0 {
			g.write(" ")
		}
		g.write(g.printfFormatFor(arg))
	}
	g.write(`\n"`)
	for _, arg := range args {
		g.write(", ")
		g.emitPrintArg(arg)
	}
	g.write(")")
}

func (g *CCodegen) printfFormatFor(arg parser.Expression) string {
	switch arg.(type) {
	case *parser.StringLiteral:
		return "%s"
	case *parser.FloatLiteral:
		return "%f"
	case *parser.BoolLiteral:
		return "%s"
	}
	if ident, ok := arg.(*parser.Identifier); ok {
		if t, ok := g.varTypes[ident.Value]; ok {
			switch t {
			case "XppString":
				return "%s"
			case "double":
				return "%f"
			case "bool":
				return "%s"
			}
		}
	}
	return "%lld"
}

func (g *CCodegen) emitPrintArg(arg parser.Expression) {
	switch e := arg.(type) {
	case *parser.StringLiteral:
		g.write(fmt.Sprintf("xpp_string_new(%q)->data", e.Value))
	case *parser.BoolLiteral:
		if e.Value {
			g.write(`"xuitru"`)
		} else {
			g.write(`"xuinia"`)
		}
	case *parser.Identifier:
		if t, ok := g.varTypes[e.Value]; ok {
			if t == "bool" {
				g.write(fmt.Sprintf(`%s ? "xuitru" : "xuinia"`, e.Value))
				return
			}
			if t == "XppString*" {
				g.write(fmt.Sprintf("%s->data", e.Value))
				return
			}
			if t == "double" {
				g.write(e.Value)
				return
			}
		}
		g.write(fmt.Sprintf("(long long)%s", e.Value))
	default:
		g.write("(long long)")
		g.emitExpression(arg)
	}
}

func (g *CCodegen) emitPrintInterpolated(e *parser.InterpolatedString) {
	// Build printf format string and args from interpolated parts
	g.write(`printf("`)
	for _, part := range e.Parts {
		if _, ok := part.(*parser.StringLiteral); ok {
			sl := part.(*parser.StringLiteral)
			// Escape % in format string
			escaped := strings.ReplaceAll(sl.Value, "%", "%%")
			escaped = strings.ReplaceAll(escaped, "\\", "\\\\")
			escaped = strings.ReplaceAll(escaped, "\"", "\\\"")
			g.write(escaped)
		} else {
			g.write(g.printfFormatFor(part))
		}
	}
	g.write(`\n"`)
	for _, part := range e.Parts {
		if _, ok := part.(*parser.StringLiteral); ok {
			continue
		}
		g.write(", ")
		g.emitPrintArg(part)
	}
	g.write(")")
}

func (g *CCodegen) isStringTypedExpr(expr parser.Expression) bool {
	if isStringExpr(expr) {
		return true
	}
	if ident, ok := expr.(*parser.Identifier); ok {
		if t, ok := g.varTypes[ident.Value]; ok {
			return t == "XppString*"
		}
	}
	if infix, ok := expr.(*parser.InfixExpression); ok {
		if infix.Operator == "+" && (isStringExpr(infix.Left) || isStringExpr(infix.Right)) {
			return true
		}
	}
	if _, ok := expr.(*parser.InterpolatedString); ok {
		return true
	}
	return false
}

func (g *CCodegen) emitMember(e *parser.MemberExpression) {
	g.emitExpression(e.Object)
	g.write(".")
	g.write(e.Member)
}

func (g *CCodegen) emitMapLiteral(e *parser.MapLiteral) {
	// Maps are emitted as a comment + NULL placeholder since C has no native map type.
	// A real implementation would use a hash table library.
	g.write("/* map literal */ NULL")
}

func (g *CCodegen) emitInterpolatedString(e *parser.InterpolatedString) {
	// For interpolated strings used as values, concatenate the string parts.
	// Full support would require snprintf; for now emit the first string literal
	// or a placeholder.
	if len(e.Parts) == 1 {
		g.emitExpression(e.Parts[0])
		return
	}
	// Multiple parts: emit concatenation of string parts
	// For simplicity, just emit the literal parts joined
	g.write(`"`)
	for _, part := range e.Parts {
		if sl, ok := part.(*parser.StringLiteral); ok {
			escaped := strings.ReplaceAll(sl.Value, "\"", "\\\"")
			g.write(escaped)
		} else {
			g.write("%s") // placeholder for non-literal parts
		}
	}
	g.write(`"`)
}

func (g *CCodegen) emitLambda(e *parser.LambdaExpression) {
	// C doesn't have lambdas natively. Emit as a function pointer comment + NULL.
	// For GCC/Clang with blocks or nested functions, we could do more.
	g.write("/* lambda */ NULL")
}

func (g *CCodegen) emitThrow(e *parser.ThrowExpression) {
	// Set the global error flag and message
	g.write("(_xpp_has_error = 1, _xpp_error_msg = ")
	g.emitExpression(e.Value)
	g.write(")")
}

// --- Helpers ---

func (g *CCodegen) inferCType(expr parser.Expression, declaredType string) string {
	if declaredType != "" {
		return g.mapType(declaredType)
	}
	switch e := expr.(type) {
	case *parser.IntegerLiteral:
		return "int64_t"
	case *parser.FloatLiteral:
		return "double"
	case *parser.StringLiteral:
		return "XppString*"
	case *parser.BoolLiteral:
		return "bool"
	case *parser.CharLiteral:
		return "char"
	case *parser.ArrayLiteral:
		return "int64_t*" // simplified
	case *parser.StructLiteral:
		return "struct " + e.Name
	case *parser.InfixExpression:
		if e.Operator == "+" && (isStringExpr(e.Left) || isStringExpr(e.Right)) {
			return "XppString*"
		}
		// Check if either side is a known string variable
		if e.Operator == "+" {
			if ident, ok := e.Left.(*parser.Identifier); ok {
				if g.varTypes[ident.Value] == "XppString" {
					return "XppString*"
				}
			}
			if ident, ok := e.Right.(*parser.Identifier); ok {
				if g.varTypes[ident.Value] == "XppString" {
					return "XppString*"
				}
			}
		}
		return "int64_t"
	case *parser.InterpolatedString:
		return "XppString*"
	case *parser.CallExpression:
		return "int64_t"
	case *parser.Identifier:
		if t, ok := g.varTypes[e.Value]; ok {
			return t
		}
		return "int64_t"
	}
	return "int64_t" // default
}

func isStringExpr(expr parser.Expression) bool {
	_, ok := expr.(*parser.StringLiteral)
	return ok
}

