package lexer

import (
	"testing"
)

// helper: lex tokens, fail on errors
func lexTokens(t *testing.T, src string) []Token {
	t.Helper()
	l := New("test.xpp", src)
	tokens, errs := l.ScanAll()
	if len(errs) > 0 {
		t.Fatalf("unexpected lexer errors: %v", errs)
	}
	return tokens
}

// helper: lex tokens, expect errors
func lexTokensWithErrors(t *testing.T, src string) ([]Token, []LexError) {
	t.Helper()
	l := New("test.xpp", src)
	return l.ScanAll()
}

// helper: check token types (excluding EOF)
func expectTypes(t *testing.T, tokens []Token, expected ...TokenType) {
	t.Helper()
	var got []Token
	for _, tok := range tokens {
		if tok.Type != TOKEN_EOF {
			got = append(got, tok)
		}
	}
	if len(got) != len(expected) {
		t.Fatalf("expected %d tokens, got %d:\n%v", len(expected), len(got), got)
	}
	for i, exp := range expected {
		if got[i].Type != exp {
			t.Errorf("token[%d]: expected %s, got %s (%q)", i, exp, got[i].Type, got[i].Literal)
		}
	}
}

func TestSingleOperators(t *testing.T) {
	tests := []struct {
		input    string
		expected TokenType
		literal  string
	}{
		{"+", TOKEN_PLUS, "+"},
		{"-", TOKEN_MINUS, "-"},
		{"*", TOKEN_STAR, "*"},
		{"/", TOKEN_SLASH, "/"},
		{"%", TOKEN_PERCENT, "%"},
		{"=", TOKEN_ASSIGN, "="},
		{"==", TOKEN_EQ, "=="},
		{"!=", TOKEN_NEQ, "!="},
		{"<", TOKEN_LT, "<"},
		{">", TOKEN_GT, ">"},
		{"<=", TOKEN_LTE, "<="},
		{">=", TOKEN_GTE, ">="},
		{"&&", TOKEN_AND, "&&"},
		{"||", TOKEN_OR, "||"},
		{"!", TOKEN_NOT, "!"},
		{"=>", TOKEN_FAT_ARROW, "=>"},
		{"..", TOKEN_DOTDOT, ".."},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			tokens := lexTokens(t, tt.input)
			if tokens[0].Type != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, tokens[0].Type)
			}
			if tokens[0].Literal != tt.literal {
				t.Errorf("expected literal %q, got %q", tt.literal, tokens[0].Literal)
			}
		})
	}
}

func TestDelimiters(t *testing.T) {
	tokens := lexTokens(t, "( ) { } [ ] , . ?")
	expectTypes(t, tokens,
		TOKEN_LPAREN, TOKEN_RPAREN,
		TOKEN_LBRACE, TOKEN_RBRACE,
		TOKEN_LBRACKET, TOKEN_RBRACKET,
		TOKEN_COMMA, TOKEN_DOT, TOKEN_QUESTION,
	)
}

func TestKeywords(t *testing.T) {
	tests := []struct {
		input    string
		expected TokenType
	}{
		{"xuen", TOKEN_XUEN},
		{"xuet", TOKEN_XUET},
		{"xuiar", TOKEN_XUIAR},
		{"xuif", TOKEN_XUIF},
		{"xuelse", TOKEN_XUELSE},
		{"xuior", TOKEN_XUIOR},
		{"xuile", TOKEN_XUILE},
		{"xuin", TOKEN_XUIN},
		{"xueturn", TOKEN_XUETURN},
		{"xuiruct", TOKEN_XUIRUCT},
		{"xuimpl", TOKEN_XUIMPL},
		{"xuenum", TOKEN_XUENUM},
		{"xuiatch", TOKEN_XUIATCH},
		{"xuitru", TOKEN_XUITRU},
		{"xuinia", TOKEN_XUINIA},
		{"xuinull", TOKEN_XUINULL},
		{"xuimport", TOKEN_XUIMPORT},
		{"xuiub", TOKEN_XUIUB},
		{"xuieak", TOKEN_XUIEAK},
		{"xuitinue", TOKEN_XUITINUE},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			tokens := lexTokens(t, tt.input)
			if tokens[0].Type != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, tokens[0].Type)
			}
		})
	}
}

func TestTypeKeywords(t *testing.T) {
	tests := []struct {
		input    string
		expected TokenType
	}{
		{"int", TOKEN_INT_TYPE},
		{"int8", TOKEN_INT8_TYPE},
		{"int16", TOKEN_INT16_TYPE},
		{"int32", TOKEN_INT32_TYPE},
		{"int64", TOKEN_INT64_TYPE},
		{"uint", TOKEN_UINT_TYPE},
		{"float", TOKEN_FLOAT_TYPE},
		{"float32", TOKEN_FLOAT32_TYPE},
		{"bool", TOKEN_BOOL_TYPE},
		{"str", TOKEN_STR_TYPE},
		{"char", TOKEN_CHAR_TYPE},
		{"byte", TOKEN_BYTE_TYPE},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			tokens := lexTokens(t, tt.input)
			if tokens[0].Type != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, tokens[0].Type)
			}
		})
	}
}

func TestIdentifiers(t *testing.T) {
	tests := []struct {
		input   string
		literal string
	}{
		{"foo", "foo"},
		{"_bar", "_bar"},
		{"camelCase", "camelCase"},
		{"snake_case", "snake_case"},
		{"x1", "x1"},
		{"_", "_"},
		{"nombre", "nombre"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			tokens := lexTokens(t, tt.input)
			if tokens[0].Type != TOKEN_IDENT {
				t.Errorf("expected IDENT, got %s", tokens[0].Type)
			}
			if tokens[0].Literal != tt.literal {
				t.Errorf("expected literal %q, got %q", tt.literal, tokens[0].Literal)
			}
		})
	}
}

func TestUnicodeIdentifiers(t *testing.T) {
	tokens := lexTokens(t, "змінна")
	if tokens[0].Type != TOKEN_IDENT {
		t.Errorf("expected IDENT for Ukrainian identifier, got %s", tokens[0].Type)
	}
	if tokens[0].Literal != "змінна" {
		t.Errorf("expected literal \"змінна\", got %q", tokens[0].Literal)
	}
}

func TestIntegerLiterals(t *testing.T) {
	tests := []struct {
		input   string
		literal string
	}{
		{"0", "0"},
		{"42", "42"},
		{"1000", "1000"},
		{"1_000_000", "1000000"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			tokens := lexTokens(t, tt.input)
			if tokens[0].Type != TOKEN_INT {
				t.Errorf("expected INT, got %s", tokens[0].Type)
			}
			if tokens[0].Literal != tt.literal {
				t.Errorf("expected literal %q, got %q", tt.literal, tokens[0].Literal)
			}
		})
	}
}

func TestHexLiterals(t *testing.T) {
	tests := []struct {
		input   string
		literal string
	}{
		{"0xFF", "0xFF"},
		{"0x1A2B", "0x1A2B"},
		{"0XAB", "0XAB"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			tokens := lexTokens(t, tt.input)
			if tokens[0].Type != TOKEN_INT {
				t.Errorf("expected INT, got %s", tokens[0].Type)
			}
			if tokens[0].Literal != tt.literal {
				t.Errorf("expected literal %q, got %q", tt.literal, tokens[0].Literal)
			}
		})
	}
}

func TestBinaryLiterals(t *testing.T) {
	tokens := lexTokens(t, "0b1010")
	if tokens[0].Type != TOKEN_INT {
		t.Errorf("expected INT, got %s", tokens[0].Type)
	}
	if tokens[0].Literal != "0b1010" {
		t.Errorf("expected literal \"0b1010\", got %q", tokens[0].Literal)
	}
}

func TestOctalLiterals(t *testing.T) {
	tokens := lexTokens(t, "0o17")
	if tokens[0].Type != TOKEN_INT {
		t.Errorf("expected INT, got %s", tokens[0].Type)
	}
	if tokens[0].Literal != "0o17" {
		t.Errorf("expected literal \"0o17\", got %q", tokens[0].Literal)
	}
}

func TestFloatLiterals(t *testing.T) {
	tests := []struct {
		input   string
		literal string
	}{
		{"3.14", "3.14"},
		{"0.5", "0.5"},
		{"1e10", "1e10"},
		{"2.5e-3", "2.5e-3"},
		{"1E5", "1E5"},
		{"3.0", "3.0"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			tokens := lexTokens(t, tt.input)
			if tokens[0].Type != TOKEN_FLOAT {
				t.Errorf("expected FLOAT, got %s (%q)", tokens[0].Type, tokens[0].Literal)
			}
			if tokens[0].Literal != tt.literal {
				t.Errorf("expected literal %q, got %q", tt.literal, tokens[0].Literal)
			}
		})
	}
}

func TestStringLiterals(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		literal string
	}{
		{"simple", `"hello"`, "hello"},
		{"empty", `""`, ""},
		{"with spaces", `"hello world"`, "hello world"},
		{"escape newline", `"line1\nline2"`, "line1\nline2"},
		{"escape tab", `"col1\tcol2"`, "col1\tcol2"},
		{"escape backslash", `"path\\file"`, "path\\file"},
		{"escape quote", `"say \"hi\""`, "say \"hi\""},
		{"unicode content", `"привіт"`, "привіт"},
		{"unicode escape", `"\u0041"`, "A"},
		{"null escape", `"a\0b"`, "a\x00b"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokens := lexTokens(t, tt.input)
			if tokens[0].Type != TOKEN_STRING {
				t.Errorf("expected STRING, got %s", tokens[0].Type)
			}
			if tokens[0].Literal != tt.literal {
				t.Errorf("expected literal %q, got %q", tt.literal, tokens[0].Literal)
			}
		})
	}
}

func TestCharLiterals(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		literal string
	}{
		{"simple", "'a'", "a"},
		{"escape newline", `'\n'`, "\n"},
		{"escape tab", `'\t'`, "\t"},
		{"unicode", "'я'", "я"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokens := lexTokens(t, tt.input)
			if tokens[0].Type != TOKEN_CHAR {
				t.Errorf("expected CHAR, got %s", tokens[0].Type)
			}
			if tokens[0].Literal != tt.literal {
				t.Errorf("expected literal %q, got %q", tt.literal, tokens[0].Literal)
			}
		})
	}
}

func TestLineComments(t *testing.T) {
	tokens := lexTokens(t, "foo // this is a comment\nbar")
	expectTypes(t, tokens,
		TOKEN_IDENT, TOKEN_SEMICOLON, TOKEN_IDENT, TOKEN_SEMICOLON,
	)
	if tokens[0].Literal != "foo" {
		t.Errorf("expected \"foo\", got %q", tokens[0].Literal)
	}
	if tokens[2].Literal != "bar" {
		t.Errorf("expected \"bar\", got %q", tokens[2].Literal)
	}
}

func TestBlockComments(t *testing.T) {
	tokens := lexTokens(t, "foo /* comment */ bar")
	expectTypes(t, tokens,
		TOKEN_IDENT, TOKEN_IDENT, TOKEN_SEMICOLON,
	)
}

func TestBlockCommentMultiline(t *testing.T) {
	tokens := lexTokens(t, "foo /* multi\nline\ncomment */ bar")
	expectTypes(t, tokens,
		TOKEN_IDENT, TOKEN_SEMICOLON, TOKEN_IDENT, TOKEN_SEMICOLON,
	)
}

func TestUnterminatedBlockComment(t *testing.T) {
	_, errs := lexTokensWithErrors(t, "foo /* unterminated")
	if len(errs) == 0 {
		t.Fatal("expected error for unterminated block comment")
	}
	found := false
	for _, e := range errs {
		if contains(e.Message, "unterminated block comment") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected 'unterminated block comment' error, got: %v", errs)
	}
}

func TestPositionTracking(t *testing.T) {
	src := "xuet x = 5\nxuet y = 10"
	tokens := lexTokens(t, src)

	// First token: "xuet" at line 1, col 1
	if tokens[0].Pos.Line != 1 || tokens[0].Pos.Column != 1 {
		t.Errorf("expected xuet at 1:1, got %d:%d", tokens[0].Pos.Line, tokens[0].Pos.Column)
	}

	// "x" at line 1, col 6
	if tokens[1].Pos.Line != 1 || tokens[1].Pos.Column != 6 {
		t.Errorf("expected x at 1:6, got %d:%d", tokens[1].Pos.Line, tokens[1].Pos.Column)
	}

	// Find "xuet" on line 2
	found := false
	for _, tok := range tokens {
		if tok.Type == TOKEN_XUET && tok.Pos.Line == 2 {
			if tok.Pos.Column != 1 {
				t.Errorf("expected second xuet at 2:1, got 2:%d", tok.Pos.Column)
			}
			found = true
			break
		}
	}
	if !found {
		t.Error("expected to find 'xuet' on line 2")
	}
}

func TestSemicolonInsertion(t *testing.T) {
	src := `x
y
xueturn
xuieak
xuitinue
42
"hello"
xuitru
xuinia
xuinull
)`
	tokens := lexTokens(t, src)

	semiCount := 0
	for _, tok := range tokens {
		if tok.Type == TOKEN_SEMICOLON {
			semiCount++
		}
	}
	// x, y, xueturn, xuieak, xuitinue, 42, "hello", xuitru, xuinia, xuinull, ) = 11
	if semiCount != 11 {
		t.Errorf("expected 11 semicolons, got %d", semiCount)
		for i, tok := range tokens {
			t.Logf("  [%d] %s", i, tok)
		}
	}
}

func TestNoSemicolonAfterOperators(t *testing.T) {
	src := "x +\ny"
	tokens := lexTokens(t, src)
	expectTypes(t, tokens,
		TOKEN_IDENT, TOKEN_PLUS, TOKEN_IDENT, TOKEN_SEMICOLON,
	)
}

func TestNoSemicolonAfterBrace(t *testing.T) {
	src := "xuen main() {\nx\n}"
	tokens := lexTokens(t, src)
	expectTypes(t, tokens,
		TOKEN_XUEN, TOKEN_IDENT, TOKEN_LPAREN, TOKEN_RPAREN, TOKEN_LBRACE,
		TOKEN_IDENT, TOKEN_SEMICOLON,
		TOKEN_RBRACE, TOKEN_SEMICOLON,
	)
}

func TestErrorRecoveryIllegalChar(t *testing.T) {
	tokens, errs := lexTokensWithErrors(t, "x @ y")
	if len(errs) == 0 {
		t.Fatal("expected error for illegal character '@'")
	}
	identCount := 0
	for _, tok := range tokens {
		if tok.Type == TOKEN_IDENT {
			identCount++
		}
	}
	if identCount != 2 {
		t.Errorf("expected 2 identifiers despite error, got %d", identCount)
	}
}

func TestUnterminatedString(t *testing.T) {
	_, errs := lexTokensWithErrors(t, `"unterminated`)
	if len(errs) == 0 {
		t.Fatal("expected error for unterminated string")
	}
	found := false
	for _, e := range errs {
		if contains(e.Message, "unterminated string") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected 'unterminated string' error, got: %v", errs)
	}
}

func TestEmptyCharLiteral(t *testing.T) {
	_, errs := lexTokensWithErrors(t, "''")
	if len(errs) == 0 {
		t.Fatal("expected error for empty character literal")
	}
}

func TestFullProgram(t *testing.T) {
	src := `xuen main() {
    xuet name str = "Xuesos"
    xuet x int = 42
    xuif (x > 10) {
        print(name)
    }
}`
	tokens := lexTokens(t, src)

	if len(tokens) < 20 {
		t.Errorf("expected at least 20 tokens for full program, got %d", len(tokens))
	}

	expected := []TokenType{
		TOKEN_XUEN, TOKEN_IDENT, TOKEN_LPAREN, TOKEN_RPAREN, TOKEN_LBRACE,
	}
	for i, exp := range expected {
		if tokens[i].Type != exp {
			t.Errorf("token[%d]: expected %s, got %s", i, exp, tokens[i].Type)
		}
	}

	last := tokens[len(tokens)-1]
	if last.Type != TOKEN_EOF {
		t.Errorf("expected EOF as last token, got %s", last.Type)
	}
}

func TestFibonacciProgram(t *testing.T) {
	src := `xuen fib(n int) int {
    xuif (n <= 1) {
        xueturn n
    }
    xueturn fib(n - 1) + fib(n - 2)
}

xuen main() {
    xuior (i xuin 0..20) {
        xuet result = fib(i)
        print(result)
    }
}`
	tokens := lexTokens(t, src)

	if len(tokens) < 40 {
		t.Errorf("expected at least 40 tokens, got %d", len(tokens))
	}

	hasDotDot := false
	hasXuior := false
	hasXuin := false
	for _, tok := range tokens {
		if tok.Type == TOKEN_DOTDOT {
			hasDotDot = true
		}
		if tok.Type == TOKEN_XUIOR {
			hasXuior = true
		}
		if tok.Type == TOKEN_XUIN {
			hasXuin = true
		}
	}
	if !hasDotDot {
		t.Error("expected DOTDOT token in fibonacci program")
	}
	if !hasXuior {
		t.Error("expected XUIOR (for) token in fibonacci program")
	}
	if !hasXuin {
		t.Error("expected XUIN (in) token in fibonacci program")
	}
}

func TestStructProgram(t *testing.T) {
	src := `xuiruct Player {
    name str
    health int
}

xuimpl Player {
    xuen take_damage(xuiar self, dmg int) {
        self.health = self.health - dmg
    }
}`
	tokens := lexTokens(t, src)

	hasXuiruct := false
	hasXuimpl := false
	for _, tok := range tokens {
		if tok.Type == TOKEN_XUIRUCT {
			hasXuiruct = true
		}
		if tok.Type == TOKEN_XUIMPL {
			hasXuimpl = true
		}
	}
	if !hasXuiruct {
		t.Error("expected XUIRUCT token")
	}
	if !hasXuimpl {
		t.Error("expected XUIMPL token")
	}
}

func TestMatchExpression(t *testing.T) {
	src := `xuiatch (status) {
    "ok" => print("good")
    _ => print("unknown")
}`
	tokens := lexTokens(t, src)

	hasFatArrow := false
	hasXuiatch := false
	for _, tok := range tokens {
		if tok.Type == TOKEN_FAT_ARROW {
			hasFatArrow = true
		}
		if tok.Type == TOKEN_XUIATCH {
			hasXuiatch = true
		}
	}
	if !hasXuiatch {
		t.Error("expected XUIATCH token")
	}
	if !hasFatArrow {
		t.Error("expected FAT_ARROW token")
	}
}

func TestNullableType(t *testing.T) {
	tokens := lexTokens(t, "xuet x ?int = xuinull")
	expectTypes(t, tokens,
		TOKEN_XUET, TOKEN_IDENT, TOKEN_QUESTION, TOKEN_INT_TYPE,
		TOKEN_ASSIGN, TOKEN_XUINULL, TOKEN_SEMICOLON,
	)
}

func TestRangeExpression(t *testing.T) {
	tokens := lexTokens(t, "0..10")
	expectTypes(t, tokens,
		TOKEN_INT, TOKEN_DOTDOT, TOKEN_INT, TOKEN_SEMICOLON,
	)
}

func TestMultipleOperators(t *testing.T) {
	tokens := lexTokens(t, "a + b * c - d / e % f")
	expectTypes(t, tokens,
		TOKEN_IDENT, TOKEN_PLUS, TOKEN_IDENT, TOKEN_STAR, TOKEN_IDENT,
		TOKEN_MINUS, TOKEN_IDENT, TOKEN_SLASH, TOKEN_IDENT, TOKEN_PERCENT,
		TOKEN_IDENT, TOKEN_SEMICOLON,
	)
}

func TestComparisonOperators(t *testing.T) {
	tokens := lexTokens(t, "a == b != c < d > e <= f >= g")
	expectTypes(t, tokens,
		TOKEN_IDENT, TOKEN_EQ, TOKEN_IDENT, TOKEN_NEQ, TOKEN_IDENT,
		TOKEN_LT, TOKEN_IDENT, TOKEN_GT, TOKEN_IDENT, TOKEN_LTE,
		TOKEN_IDENT, TOKEN_GTE, TOKEN_IDENT, TOKEN_SEMICOLON,
	)
}

func TestLogicalOperators(t *testing.T) {
	tokens := lexTokens(t, "a && b || !c")
	expectTypes(t, tokens,
		TOKEN_IDENT, TOKEN_AND, TOKEN_IDENT, TOKEN_OR, TOKEN_NOT,
		TOKEN_IDENT, TOKEN_SEMICOLON,
	)
}

func TestArrayType(t *testing.T) {
	tokens := lexTokens(t, "xuet nums = [1, 2, 3]")
	expectTypes(t, tokens,
		TOKEN_XUET, TOKEN_IDENT, TOKEN_ASSIGN, TOKEN_LBRACKET, TOKEN_INT, TOKEN_COMMA,
		TOKEN_INT, TOKEN_COMMA, TOKEN_INT, TOKEN_RBRACKET, TOKEN_SEMICOLON,
	)
}

func TestEmptySource(t *testing.T) {
	tokens := lexTokens(t, "")
	if len(tokens) != 1 || tokens[0].Type != TOKEN_EOF {
		t.Errorf("expected single EOF token, got %v", tokens)
	}
}

func TestWhitespaceOnly(t *testing.T) {
	tokens := lexTokens(t, "   \t  \n  \t  ")
	if len(tokens) != 1 || tokens[0].Type != TOKEN_EOF {
		t.Errorf("expected single EOF token, got %v", tokens)
	}
}

func TestTokenPositionString(t *testing.T) {
	pos := Position{File: "main.xpp", Line: 5, Column: 12}
	s := pos.String()
	if s != "main.xpp:5:12" {
		t.Errorf("expected \"main.xpp:5:12\", got %q", s)
	}
}

func TestTokenString(t *testing.T) {
	tok := Token{
		Type:    TOKEN_XUEN,
		Literal: "xuen",
		Pos:     Position{File: "test.xpp", Line: 1, Column: 1},
	}
	s := tok.String()
	if s == "" {
		t.Error("expected non-empty token string")
	}
}

func TestSingleAmpersand(t *testing.T) {
	tokens := lexTokens(t, "&x")
	if tokens[0].Type != TOKEN_AMPERSAND {
		t.Errorf("expected AMPERSAND, got %s", tokens[0].Type)
	}
}

func TestSinglePipeError(t *testing.T) {
	_, errs := lexTokensWithErrors(t, "|")
	if len(errs) == 0 {
		t.Fatal("expected error for single '|'")
	}
}

func TestImportStatement(t *testing.T) {
	tokens := lexTokens(t, `xuimport "io"`)
	expectTypes(t, tokens,
		TOKEN_XUIMPORT, TOKEN_STRING, TOKEN_SEMICOLON,
	)
}

func TestVariableDeclarations(t *testing.T) {
	// immutable
	tokens := lexTokens(t, `xuet name = "Xuesos"`)
	expectTypes(t, tokens,
		TOKEN_XUET, TOKEN_IDENT, TOKEN_ASSIGN, TOKEN_STRING, TOKEN_SEMICOLON,
	)

	// mutable
	tokens = lexTokens(t, `xuiar counter = 0`)
	expectTypes(t, tokens,
		TOKEN_XUIAR, TOKEN_IDENT, TOKEN_ASSIGN, TOKEN_INT, TOKEN_SEMICOLON,
	)
}

func TestWhileLoop(t *testing.T) {
	tokens := lexTokens(t, `xuile (x < 10) { x = x + 1 }`)
	expectTypes(t, tokens,
		TOKEN_XUILE, TOKEN_LPAREN, TOKEN_IDENT, TOKEN_LT, TOKEN_INT, TOKEN_RPAREN,
		TOKEN_LBRACE, TOKEN_IDENT, TOKEN_ASSIGN, TOKEN_IDENT, TOKEN_PLUS, TOKEN_INT,
		TOKEN_RBRACE, TOKEN_SEMICOLON,
	)
}

func TestEnumDeclaration(t *testing.T) {
	src := `xuenum Direction {
    Up
    Down
    Left
    Right
}`
	tokens := lexTokens(t, src)
	if tokens[0].Type != TOKEN_XUENUM {
		t.Errorf("expected XUENUM, got %s", tokens[0].Type)
	}
}

func TestPubKeyword(t *testing.T) {
	tokens := lexTokens(t, "xuiub xuen hello() {}")
	if tokens[0].Type != TOKEN_XUIUB {
		t.Errorf("expected XUIUB, got %s", tokens[0].Type)
	}
	if tokens[1].Type != TOKEN_XUEN {
		t.Errorf("expected XUEN, got %s", tokens[1].Type)
	}
}

// helper
func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchString(s, substr)
}

func searchString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// --- Map syntax tests ---

func TestColonToken(t *testing.T) {
	tokens := lexTokens(t, "a : b")
	found := false
	for _, tok := range tokens {
		if tok.Type == TOKEN_COLON {
			found = true
		}
	}
	if !found {
		t.Error("expected COLON token")
	}
}

func TestMapLiteral(t *testing.T) {
	// [str:int] type syntax needs colon
	tokens := lexTokens(t, `xuet m = {"a": 1, "b": 2}`)
	hasColon := false
	for _, tok := range tokens {
		if tok.Type == TOKEN_COLON {
			hasColon = true
		}
	}
	if !hasColon {
		t.Error("expected COLON in map literal")
	}
}

// --- String interpolation tests ---

func TestStringInterpolation(t *testing.T) {
	// "hello {name}" should produce INTERP_STRING tokens
	tokens := lexTokens(t, `"hello {name}"`)
	// Should have: STRING_PART("hello ") IDENT(name) STRING_PART("")
	hasInterpStart := false
	for _, tok := range tokens {
		if tok.Type == TOKEN_INTERP_START {
			hasInterpStart = true
		}
	}
	if !hasInterpStart {
		t.Error("expected INTERP_START token in interpolated string")
	}
}

func TestStringInterpolationMultiple(t *testing.T) {
	tokens := lexTokens(t, `"{a} + {b} = {c}"`)
	// Should have INTERP_START, then STRING/IDENT pairs, then INTERP_END
	hasStart := false
	hasEnd := false
	identCount := 0
	for _, tok := range tokens {
		if tok.Type == TOKEN_INTERP_START {
			hasStart = true
		}
		if tok.Type == TOKEN_INTERP_END {
			hasEnd = true
		}
		if tok.Type == TOKEN_IDENT {
			identCount++
		}
	}
	if !hasStart {
		t.Error("expected INTERP_START")
	}
	if !hasEnd {
		t.Error("expected INTERP_END")
	}
	if identCount != 3 {
		t.Errorf("expected 3 identifiers (a, b, c), got %d", identCount)
	}
}

func TestStringNoInterpolation(t *testing.T) {
	// Regular strings without { should still work
	tokens := lexTokens(t, `"hello world"`)
	if tokens[0].Type != TOKEN_STRING {
		t.Errorf("expected STRING, got %s", tokens[0].Type)
	}
	if tokens[0].Literal != "hello world" {
		t.Errorf("expected 'hello world', got %q", tokens[0].Literal)
	}
}
