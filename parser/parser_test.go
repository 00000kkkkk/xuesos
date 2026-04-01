package parser

import (
	"testing"

	"github.com/00000kkkkk/xusesosplusplus/lexer"
)

func parse(t *testing.T, src string) *Program {
	t.Helper()
	l := lexer.New("test.xpp", src)
	tokens, lexErrs := l.ScanAll()
	if len(lexErrs) > 0 {
		t.Fatalf("lexer errors: %v", lexErrs)
	}
	p := New(tokens)
	prog, parseErrs := p.Parse()
	if len(parseErrs) > 0 {
		t.Fatalf("parser errors: %v", parseErrs)
	}
	return prog
}

func parseWithErrors(t *testing.T, src string) (*Program, []ParseError) {
	t.Helper()
	l := lexer.New("test.xpp", src)
	tokens, _ := l.ScanAll()
	p := New(tokens)
	return p.Parse()
}

func TestXuetStatement(t *testing.T) {
	prog := parse(t, `xuet x = 42`)
	if len(prog.Statements) != 1 {
		t.Fatalf("expected 1 statement, got %d", len(prog.Statements))
	}
	stmt, ok := prog.Statements[0].(*XuetStatement)
	if !ok {
		t.Fatalf("expected XuetStatement, got %T", prog.Statements[0])
	}
	if stmt.Name != "x" {
		t.Errorf("expected name 'x', got %q", stmt.Name)
	}
	lit, ok := stmt.Value.(*IntegerLiteral)
	if !ok {
		t.Fatalf("expected IntegerLiteral, got %T", stmt.Value)
	}
	if lit.Value != 42 {
		t.Errorf("expected 42, got %d", lit.Value)
	}
}

func TestXuetWithType(t *testing.T) {
	prog := parse(t, `xuet name str = "hello"`)
	stmt := prog.Statements[0].(*XuetStatement)
	if stmt.TypeName != "str" {
		t.Errorf("expected type 'str', got %q", stmt.TypeName)
	}
	lit := stmt.Value.(*StringLiteral)
	if lit.Value != "hello" {
		t.Errorf("expected 'hello', got %q", lit.Value)
	}
}

func TestXuiarStatement(t *testing.T) {
	prog := parse(t, `xuiar counter = 0`)
	stmt, ok := prog.Statements[0].(*XuiarStatement)
	if !ok {
		t.Fatalf("expected XuiarStatement, got %T", prog.Statements[0])
	}
	if stmt.Name != "counter" {
		t.Errorf("expected name 'counter', got %q", stmt.Name)
	}
}

func TestXuenStatement(t *testing.T) {
	prog := parse(t, `xuen add(a int, b int) int {
		xueturn a + b
	}`)
	stmt, ok := prog.Statements[0].(*XuenStatement)
	if !ok {
		t.Fatalf("expected XuenStatement, got %T", prog.Statements[0])
	}
	if stmt.Name != "add" {
		t.Errorf("expected name 'add', got %q", stmt.Name)
	}
	if len(stmt.Params) != 2 {
		t.Fatalf("expected 2 params, got %d", len(stmt.Params))
	}
	if stmt.Params[0].Name != "a" || stmt.Params[0].TypeName != "int" {
		t.Errorf("param 0: expected 'a int', got '%s %s'", stmt.Params[0].Name, stmt.Params[0].TypeName)
	}
	if stmt.ReturnType != "int" {
		t.Errorf("expected return type 'int', got %q", stmt.ReturnType)
	}
	if len(stmt.Body.Statements) != 1 {
		t.Fatalf("expected 1 body statement, got %d", len(stmt.Body.Statements))
	}
}

func TestXuenNoReturn(t *testing.T) {
	prog := parse(t, `xuen greet(name str) {
		print(name)
	}`)
	stmt := prog.Statements[0].(*XuenStatement)
	if stmt.ReturnType != "" {
		t.Errorf("expected empty return type, got %q", stmt.ReturnType)
	}
}

func TestXuifStatement(t *testing.T) {
	prog := parse(t, `xuif (x > 10) {
		print("big")
	}`)
	stmt, ok := prog.Statements[0].(*XuifStatement)
	if !ok {
		t.Fatalf("expected XuifStatement, got %T", prog.Statements[0])
	}
	if stmt.Alternative != nil {
		t.Error("expected no alternative")
	}
}

func TestXuifXuelse(t *testing.T) {
	prog := parse(t, `xuif (x > 10) {
		print("big")
	} xuelse {
		print("small")
	}`)
	stmt := prog.Statements[0].(*XuifStatement)
	if stmt.Alternative == nil {
		t.Fatal("expected alternative")
	}
	_, ok := stmt.Alternative.(*BlockStatement)
	if !ok {
		t.Fatalf("expected BlockStatement alternative, got %T", stmt.Alternative)
	}
}

func TestXuifXuelseXuif(t *testing.T) {
	prog := parse(t, `xuif (x > 10) {
		print("big")
	} xuelse xuif (x > 5) {
		print("medium")
	} xuelse {
		print("small")
	}`)
	stmt := prog.Statements[0].(*XuifStatement)
	elseIf, ok := stmt.Alternative.(*XuifStatement)
	if !ok {
		t.Fatalf("expected XuifStatement in else, got %T", stmt.Alternative)
	}
	if elseIf.Alternative == nil {
		t.Fatal("expected final else block")
	}
}

func TestXuiorStatement(t *testing.T) {
	prog := parse(t, `xuior (i xuin 0..10) {
		print(i)
	}`)
	stmt, ok := prog.Statements[0].(*XuiorStatement)
	if !ok {
		t.Fatalf("expected XuiorStatement, got %T", prog.Statements[0])
	}
	if stmt.Variable != "i" {
		t.Errorf("expected variable 'i', got %q", stmt.Variable)
	}
	_, ok = stmt.Iterable.(*RangeExpression)
	if !ok {
		t.Fatalf("expected RangeExpression, got %T", stmt.Iterable)
	}
}

func TestXuiorCollection(t *testing.T) {
	prog := parse(t, `xuior (item xuin items) {
		print(item)
	}`)
	stmt := prog.Statements[0].(*XuiorStatement)
	ident, ok := stmt.Iterable.(*Identifier)
	if !ok {
		t.Fatalf("expected Identifier iterable, got %T", stmt.Iterable)
	}
	if ident.Value != "items" {
		t.Errorf("expected 'items', got %q", ident.Value)
	}
}

func TestXuileStatement(t *testing.T) {
	prog := parse(t, `xuile (x < 100) {
		x = x + 1
	}`)
	stmt, ok := prog.Statements[0].(*XuileStatement)
	if !ok {
		t.Fatalf("expected XuileStatement, got %T", prog.Statements[0])
	}
	if len(stmt.Body.Statements) != 1 {
		t.Fatalf("expected 1 body statement, got %d", len(stmt.Body.Statements))
	}
}

func TestXuiructStatement(t *testing.T) {
	prog := parse(t, `xuiruct Player {
		name str
		health int
		alive bool
	}`)
	stmt, ok := prog.Statements[0].(*XuiructStatement)
	if !ok {
		t.Fatalf("expected XuiructStatement, got %T", prog.Statements[0])
	}
	if stmt.Name != "Player" {
		t.Errorf("expected 'Player', got %q", stmt.Name)
	}
	if len(stmt.Fields) != 3 {
		t.Fatalf("expected 3 fields, got %d", len(stmt.Fields))
	}
	if stmt.Fields[0].Name != "name" || stmt.Fields[0].TypeName != "str" {
		t.Errorf("field 0: expected 'name str', got '%s %s'", stmt.Fields[0].Name, stmt.Fields[0].TypeName)
	}
}

func TestXuimplStatement(t *testing.T) {
	prog := parse(t, `xuimpl Player {
		xuen greet(self) {
			print("hello")
		}
	}`)
	stmt, ok := prog.Statements[0].(*XuimplStatement)
	if !ok {
		t.Fatalf("expected XuimplStatement, got %T", prog.Statements[0])
	}
	if stmt.Name != "Player" {
		t.Errorf("expected 'Player', got %q", stmt.Name)
	}
	if len(stmt.Methods) != 1 {
		t.Fatalf("expected 1 method, got %d", len(stmt.Methods))
	}
	if stmt.Methods[0].Name != "greet" {
		t.Errorf("expected method 'greet', got %q", stmt.Methods[0].Name)
	}
}

func TestXuenumStatement(t *testing.T) {
	prog := parse(t, `xuenum Direction {
		Up
		Down
		Left
		Right
	}`)
	stmt, ok := prog.Statements[0].(*XuenumStatement)
	if !ok {
		t.Fatalf("expected XuenumStatement, got %T", prog.Statements[0])
	}
	if stmt.Name != "Direction" {
		t.Errorf("expected 'Direction', got %q", stmt.Name)
	}
	if len(stmt.Variants) != 4 {
		t.Fatalf("expected 4 variants, got %d", len(stmt.Variants))
	}
}

func TestXuimportStatement(t *testing.T) {
	prog := parse(t, `xuimport "io"`)
	stmt, ok := prog.Statements[0].(*XuimportStatement)
	if !ok {
		t.Fatalf("expected XuimportStatement, got %T", prog.Statements[0])
	}
	if stmt.Path != "io" {
		t.Errorf("expected path 'io', got %q", stmt.Path)
	}
}

func TestXuiatchStatement(t *testing.T) {
	prog := parse(t, `xuiatch (status) {
		"ok" => print("good")
		"error" => print("bad")
		_ => print("unknown")
	}`)
	stmt, ok := prog.Statements[0].(*XuiatchStatement)
	if !ok {
		t.Fatalf("expected XuiatchStatement, got %T", prog.Statements[0])
	}
	if len(stmt.Arms) != 3 {
		t.Fatalf("expected 3 arms, got %d", len(stmt.Arms))
	}
}

func TestAssignStatement(t *testing.T) {
	prog := parse(t, `x = 10`)
	stmt, ok := prog.Statements[0].(*AssignStatement)
	if !ok {
		t.Fatalf("expected AssignStatement, got %T", prog.Statements[0])
	}
	target, ok := stmt.Target.(*Identifier)
	if !ok {
		t.Fatalf("expected Identifier target, got %T", stmt.Target)
	}
	if target.Value != "x" {
		t.Errorf("expected 'x', got %q", target.Value)
	}
}

func TestMemberAssign(t *testing.T) {
	prog := parse(t, `self.health = 100`)
	stmt, ok := prog.Statements[0].(*AssignStatement)
	if !ok {
		t.Fatalf("expected AssignStatement, got %T", prog.Statements[0])
	}
	member, ok := stmt.Target.(*MemberExpression)
	if !ok {
		t.Fatalf("expected MemberExpression target, got %T", stmt.Target)
	}
	if member.Member != "health" {
		t.Errorf("expected 'health', got %q", member.Member)
	}
}

// --- Expression tests ---

func TestIntegerExpression(t *testing.T) {
	prog := parse(t, `42`)
	stmt := prog.Statements[0].(*ExpressionStatement)
	lit, ok := stmt.Expr.(*IntegerLiteral)
	if !ok {
		t.Fatalf("expected IntegerLiteral, got %T", stmt.Expr)
	}
	if lit.Value != 42 {
		t.Errorf("expected 42, got %d", lit.Value)
	}
}

func TestHexInteger(t *testing.T) {
	prog := parse(t, `0xFF`)
	stmt := prog.Statements[0].(*ExpressionStatement)
	lit := stmt.Expr.(*IntegerLiteral)
	if lit.Value != 255 {
		t.Errorf("expected 255, got %d", lit.Value)
	}
}

func TestFloatExpression(t *testing.T) {
	prog := parse(t, `3.14`)
	stmt := prog.Statements[0].(*ExpressionStatement)
	lit, ok := stmt.Expr.(*FloatLiteral)
	if !ok {
		t.Fatalf("expected FloatLiteral, got %T", stmt.Expr)
	}
	if lit.Value != 3.14 {
		t.Errorf("expected 3.14, got %f", lit.Value)
	}
}

func TestStringExpression(t *testing.T) {
	prog := parse(t, `"hello"`)
	stmt := prog.Statements[0].(*ExpressionStatement)
	lit, ok := stmt.Expr.(*StringLiteral)
	if !ok {
		t.Fatalf("expected StringLiteral, got %T", stmt.Expr)
	}
	if lit.Value != "hello" {
		t.Errorf("expected 'hello', got %q", lit.Value)
	}
}

func TestBoolLiterals(t *testing.T) {
	prog := parse(t, `xuitru`)
	stmt := prog.Statements[0].(*ExpressionStatement)
	b := stmt.Expr.(*BoolLiteral)
	if !b.Value {
		t.Error("expected true")
	}

	prog = parse(t, `xuinia`)
	stmt = prog.Statements[0].(*ExpressionStatement)
	b = stmt.Expr.(*BoolLiteral)
	if b.Value {
		t.Error("expected false")
	}
}

func TestNullLiteral(t *testing.T) {
	prog := parse(t, `xuinull`)
	stmt := prog.Statements[0].(*ExpressionStatement)
	_, ok := stmt.Expr.(*NullLiteral)
	if !ok {
		t.Fatalf("expected NullLiteral, got %T", stmt.Expr)
	}
}

func TestPrefixNot(t *testing.T) {
	prog := parse(t, `!xuitru`)
	stmt := prog.Statements[0].(*ExpressionStatement)
	pre, ok := stmt.Expr.(*PrefixExpression)
	if !ok {
		t.Fatalf("expected PrefixExpression, got %T", stmt.Expr)
	}
	if pre.Operator != "!" {
		t.Errorf("expected '!', got %q", pre.Operator)
	}
}

func TestPrefixNegate(t *testing.T) {
	prog := parse(t, `-42`)
	stmt := prog.Statements[0].(*ExpressionStatement)
	pre := stmt.Expr.(*PrefixExpression)
	if pre.Operator != "-" {
		t.Errorf("expected '-', got %q", pre.Operator)
	}
}

func TestInfixExpressions(t *testing.T) {
	tests := []struct {
		input    string
		left     int64
		operator string
		right    int64
	}{
		{"5 + 5", 5, "+", 5},
		{"5 - 5", 5, "-", 5},
		{"5 * 5", 5, "*", 5},
		{"5 / 5", 5, "/", 5},
		{"5 % 3", 5, "%", 3},
		{"5 == 5", 5, "==", 5},
		{"5 != 5", 5, "!=", 5},
		{"5 < 10", 5, "<", 10},
		{"5 > 3", 5, ">", 3},
		{"5 <= 5", 5, "<=", 5},
		{"5 >= 5", 5, ">=", 5},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			prog := parse(t, tt.input)
			stmt := prog.Statements[0].(*ExpressionStatement)
			infix, ok := stmt.Expr.(*InfixExpression)
			if !ok {
				t.Fatalf("expected InfixExpression, got %T", stmt.Expr)
			}
			if infix.Operator != tt.operator {
				t.Errorf("expected operator %q, got %q", tt.operator, infix.Operator)
			}
			left := infix.Left.(*IntegerLiteral)
			right := infix.Right.(*IntegerLiteral)
			if left.Value != tt.left {
				t.Errorf("expected left %d, got %d", tt.left, left.Value)
			}
			if right.Value != tt.right {
				t.Errorf("expected right %d, got %d", tt.right, right.Value)
			}
		})
	}
}

func TestOperatorPrecedence(t *testing.T) {
	// a + b * c should parse as a + (b * c)
	prog := parse(t, `a + b * c`)
	stmt := prog.Statements[0].(*ExpressionStatement)
	infix := stmt.Expr.(*InfixExpression)
	if infix.Operator != "+" {
		t.Errorf("expected top-level +, got %q", infix.Operator)
	}
	right, ok := infix.Right.(*InfixExpression)
	if !ok {
		t.Fatalf("expected right to be InfixExpression, got %T", infix.Right)
	}
	if right.Operator != "*" {
		t.Errorf("expected right operator *, got %q", right.Operator)
	}
}

func TestLogicalPrecedence(t *testing.T) {
	// a || b && c should parse as a || (b && c)
	prog := parse(t, `a || b && c`)
	stmt := prog.Statements[0].(*ExpressionStatement)
	infix := stmt.Expr.(*InfixExpression)
	if infix.Operator != "||" {
		t.Errorf("expected top-level ||, got %q", infix.Operator)
	}
	right := infix.Right.(*InfixExpression)
	if right.Operator != "&&" {
		t.Errorf("expected right &&, got %q", right.Operator)
	}
}

func TestGroupedExpression(t *testing.T) {
	// (a + b) * c
	prog := parse(t, `(a + b) * c`)
	stmt := prog.Statements[0].(*ExpressionStatement)
	infix := stmt.Expr.(*InfixExpression)
	if infix.Operator != "*" {
		t.Errorf("expected top-level *, got %q", infix.Operator)
	}
	left, ok := infix.Left.(*InfixExpression)
	if !ok {
		t.Fatalf("expected left InfixExpression, got %T", infix.Left)
	}
	if left.Operator != "+" {
		t.Errorf("expected left +, got %q", left.Operator)
	}
}

func TestCallExpression(t *testing.T) {
	prog := parse(t, `print("hello", 42)`)
	stmt := prog.Statements[0].(*ExpressionStatement)
	call, ok := stmt.Expr.(*CallExpression)
	if !ok {
		t.Fatalf("expected CallExpression, got %T", stmt.Expr)
	}
	fn := call.Function.(*Identifier)
	if fn.Value != "print" {
		t.Errorf("expected 'print', got %q", fn.Value)
	}
	if len(call.Arguments) != 2 {
		t.Fatalf("expected 2 args, got %d", len(call.Arguments))
	}
}

func TestMemberExpression(t *testing.T) {
	prog := parse(t, `self.health`)
	stmt := prog.Statements[0].(*ExpressionStatement)
	member, ok := stmt.Expr.(*MemberExpression)
	if !ok {
		t.Fatalf("expected MemberExpression, got %T", stmt.Expr)
	}
	if member.Member != "health" {
		t.Errorf("expected 'health', got %q", member.Member)
	}
}

func TestChainedMember(t *testing.T) {
	prog := parse(t, `a.b.c`)
	stmt := prog.Statements[0].(*ExpressionStatement)
	outer, ok := stmt.Expr.(*MemberExpression)
	if !ok {
		t.Fatalf("expected MemberExpression, got %T", stmt.Expr)
	}
	if outer.Member != "c" {
		t.Errorf("expected 'c', got %q", outer.Member)
	}
	inner, ok := outer.Object.(*MemberExpression)
	if !ok {
		t.Fatalf("expected inner MemberExpression, got %T", outer.Object)
	}
	if inner.Member != "b" {
		t.Errorf("expected 'b', got %q", inner.Member)
	}
}

func TestMethodCall(t *testing.T) {
	prog := parse(t, `player.take_damage(10)`)
	stmt := prog.Statements[0].(*ExpressionStatement)
	call, ok := stmt.Expr.(*CallExpression)
	if !ok {
		t.Fatalf("expected CallExpression, got %T", stmt.Expr)
	}
	member, ok := call.Function.(*MemberExpression)
	if !ok {
		t.Fatalf("expected MemberExpression function, got %T", call.Function)
	}
	if member.Member != "take_damage" {
		t.Errorf("expected 'take_damage', got %q", member.Member)
	}
}

func TestIndexExpression(t *testing.T) {
	prog := parse(t, `arr[0]`)
	stmt := prog.Statements[0].(*ExpressionStatement)
	idx, ok := stmt.Expr.(*IndexExpression)
	if !ok {
		t.Fatalf("expected IndexExpression, got %T", stmt.Expr)
	}
	index := idx.Index.(*IntegerLiteral)
	if index.Value != 0 {
		t.Errorf("expected index 0, got %d", index.Value)
	}
}

func TestArrayLiteral(t *testing.T) {
	prog := parse(t, `[1, 2, 3]`)
	stmt := prog.Statements[0].(*ExpressionStatement)
	arr, ok := stmt.Expr.(*ArrayLiteral)
	if !ok {
		t.Fatalf("expected ArrayLiteral, got %T", stmt.Expr)
	}
	if len(arr.Elements) != 3 {
		t.Fatalf("expected 3 elements, got %d", len(arr.Elements))
	}
}

func TestRangeExpression(t *testing.T) {
	prog := parse(t, `0..10`)
	stmt := prog.Statements[0].(*ExpressionStatement)
	rng, ok := stmt.Expr.(*RangeExpression)
	if !ok {
		t.Fatalf("expected RangeExpression, got %T", stmt.Expr)
	}
	start := rng.Start.(*IntegerLiteral)
	end := rng.End.(*IntegerLiteral)
	if start.Value != 0 || end.Value != 10 {
		t.Errorf("expected 0..10, got %d..%d", start.Value, end.Value)
	}
}

func TestXueturnStatement(t *testing.T) {
	prog := parse(t, `xueturn 42`)
	stmt, ok := prog.Statements[0].(*XueturnStatement)
	if !ok {
		t.Fatalf("expected XueturnStatement, got %T", prog.Statements[0])
	}
	lit := stmt.Value.(*IntegerLiteral)
	if lit.Value != 42 {
		t.Errorf("expected 42, got %d", lit.Value)
	}
}

func TestXueturnEmpty(t *testing.T) {
	prog := parse(t, `xuen foo() { xueturn }`)
	fn := prog.Statements[0].(*XuenStatement)
	ret := fn.Body.Statements[0].(*XueturnStatement)
	if ret.Value != nil {
		t.Error("expected nil return value")
	}
}

func TestXueakStatement(t *testing.T) {
	prog := parse(t, `xuieak`)
	_, ok := prog.Statements[0].(*XueakStatement)
	if !ok {
		t.Fatalf("expected XueakStatement, got %T", prog.Statements[0])
	}
}

func TestXuitinueStatement(t *testing.T) {
	prog := parse(t, `xuitinue`)
	_, ok := prog.Statements[0].(*XuitinueStatement)
	if !ok {
		t.Fatalf("expected XuitinueStatement, got %T", prog.Statements[0])
	}
}

func TestFullProgram(t *testing.T) {
	src := `xuen fibonacci(n int) int {
		xuif (n <= 1) {
			xueturn n
		}
		xueturn fibonacci(n - 1) + fibonacci(n - 2)
	}

	xuen main() {
		xuior (i xuin 0..20) {
			xuet result = fibonacci(i)
			print(result)
		}
	}`

	prog := parse(t, src)
	if len(prog.Statements) != 2 {
		t.Fatalf("expected 2 statements, got %d", len(prog.Statements))
	}

	fib, ok := prog.Statements[0].(*XuenStatement)
	if !ok {
		t.Fatalf("expected XuenStatement, got %T", prog.Statements[0])
	}
	if fib.Name != "fibonacci" {
		t.Errorf("expected 'fibonacci', got %q", fib.Name)
	}

	main, ok := prog.Statements[1].(*XuenStatement)
	if !ok {
		t.Fatalf("expected XuenStatement, got %T", prog.Statements[1])
	}
	if main.Name != "main" {
		t.Errorf("expected 'main', got %q", main.Name)
	}
}

func TestStructProgram(t *testing.T) {
	src := `xuiruct Player {
		name str
		health int
		alive bool
	}

	xuimpl Player {
		xuen take_damage(xuiar self, dmg int) {
			self.health = self.health - dmg
			xuif (self.health <= 0) {
				self.alive = xuinia
			}
		}
	}`

	prog := parse(t, src)
	if len(prog.Statements) != 2 {
		t.Fatalf("expected 2 statements, got %d", len(prog.Statements))
	}

	st, ok := prog.Statements[0].(*XuiructStatement)
	if !ok {
		t.Fatalf("expected XuiructStatement, got %T", prog.Statements[0])
	}
	if len(st.Fields) != 3 {
		t.Errorf("expected 3 fields, got %d", len(st.Fields))
	}

	impl, ok := prog.Statements[1].(*XuimplStatement)
	if !ok {
		t.Fatalf("expected XuimplStatement, got %T", prog.Statements[1])
	}
	if len(impl.Methods) != 1 {
		t.Errorf("expected 1 method, got %d", len(impl.Methods))
	}
}

func TestEmptyProgram(t *testing.T) {
	prog := parse(t, ``)
	if len(prog.Statements) != 0 {
		t.Errorf("expected 0 statements, got %d", len(prog.Statements))
	}
}

// --- Map tests ---

func TestMapLiteral(t *testing.T) {
	prog := parse(t, `xuet m = {"name": "xuesos", "version": "1.0"}`)
	stmt := prog.Statements[0].(*XuetStatement)
	ml, ok := stmt.Value.(*MapLiteral)
	if !ok {
		t.Fatalf("expected MapLiteral, got %T", stmt.Value)
	}
	if len(ml.Pairs) != 2 {
		t.Fatalf("expected 2 pairs, got %d", len(ml.Pairs))
	}
}

func TestEmptyMapLiteral(t *testing.T) {
	prog := parse(t, `xuet m = {}`)
	stmt := prog.Statements[0].(*XuetStatement)
	ml, ok := stmt.Value.(*MapLiteral)
	if !ok {
		t.Fatalf("expected MapLiteral, got %T", stmt.Value)
	}
	if len(ml.Pairs) != 0 {
		t.Fatalf("expected 0 pairs, got %d", len(ml.Pairs))
	}
}

func TestMapAccess(t *testing.T) {
	prog := parse(t, `m["key"]`)
	stmt := prog.Statements[0].(*ExpressionStatement)
	idx, ok := stmt.Expr.(*IndexExpression)
	if !ok {
		t.Fatalf("expected IndexExpression, got %T", stmt.Expr)
	}
	key, ok := idx.Index.(*StringLiteral)
	if !ok {
		t.Fatalf("expected StringLiteral index, got %T", idx.Index)
	}
	if key.Value != "key" {
		t.Errorf("expected 'key', got %q", key.Value)
	}
}

// --- String interpolation tests ---

func TestStringInterpolation(t *testing.T) {
	prog := parse(t, `xuet name = "world"
xuet msg = "hello {name}!"`)
	if len(prog.Statements) != 2 {
		t.Fatalf("expected 2 statements, got %d", len(prog.Statements))
	}
	stmt := prog.Statements[1].(*XuetStatement)
	// Should be an InterpolatedString or InfixExpression (concat)
	_, isInterp := stmt.Value.(*InterpolatedString)
	_, isInfix := stmt.Value.(*InfixExpression)
	if !isInterp && !isInfix {
		t.Fatalf("expected InterpolatedString or InfixExpression, got %T", stmt.Value)
	}
}

func TestStringInterpolationExpression(t *testing.T) {
	prog := parse(t, `"2 + 2 = {2 + 2}"`)
	stmt := prog.Statements[0].(*ExpressionStatement)
	_, isInterp := stmt.Expr.(*InterpolatedString)
	_, isInfix := stmt.Expr.(*InfixExpression)
	if !isInterp && !isInfix {
		t.Fatalf("expected InterpolatedString or InfixExpression, got %T", stmt.Expr)
	}
}
