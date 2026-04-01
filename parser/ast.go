package parser

import "github.com/00000kkkkk/xusesosplusplus/lexer"

// Node is the interface all AST nodes implement.
type Node interface {
	TokenPos() lexer.Position
	nodeType() string
}

// Expression nodes produce values.
type Expression interface {
	Node
	exprNode()
}

// Statement nodes perform actions.
type Statement interface {
	Node
	stmtNode()
}

// Program is the root AST node.
type Program struct {
	Statements []Statement
}

func (p *Program) TokenPos() lexer.Position {
	if len(p.Statements) > 0 {
		return p.Statements[0].TokenPos()
	}
	return lexer.Position{}
}
func (p *Program) nodeType() string { return "Program" }

// --- Statements ---

// XuetStatement: xuet name = value  /  xuet name type = value
type XuetStatement struct {
	Pos      lexer.Position
	Name     string
	TypeName string // optional, "" if inferred
	Value    Expression
}

func (s *XuetStatement) stmtNode()                {}
func (s *XuetStatement) TokenPos() lexer.Position  { return s.Pos }
func (s *XuetStatement) nodeType() string          { return "XuetStatement" }

// XuiarStatement: xuiar name = value  /  xuiar name type = value
type XuiarStatement struct {
	Pos      lexer.Position
	Name     string
	TypeName string
	Value    Expression
}

func (s *XuiarStatement) stmtNode()                {}
func (s *XuiarStatement) TokenPos() lexer.Position  { return s.Pos }
func (s *XuiarStatement) nodeType() string          { return "XuiarStatement" }

// AssignStatement: name = value
type AssignStatement struct {
	Pos   lexer.Position
	Target Expression
	Value Expression
}

func (s *AssignStatement) stmtNode()                {}
func (s *AssignStatement) TokenPos() lexer.Position  { return s.Pos }
func (s *AssignStatement) nodeType() string          { return "AssignStatement" }

// XueturnStatement: xueturn [expr]
type XueturnStatement struct {
	Pos   lexer.Position
	Value Expression // can be nil
}

func (s *XueturnStatement) stmtNode()                {}
func (s *XueturnStatement) TokenPos() lexer.Position  { return s.Pos }
func (s *XueturnStatement) nodeType() string          { return "XueturnStatement" }

// XueakStatement: xuieak
type XueakStatement struct {
	Pos lexer.Position
}

func (s *XueakStatement) stmtNode()                {}
func (s *XueakStatement) TokenPos() lexer.Position  { return s.Pos }
func (s *XueakStatement) nodeType() string          { return "XueakStatement" }

// XuitinueStatement: xuitinue
type XuitinueStatement struct {
	Pos lexer.Position
}

func (s *XuitinueStatement) stmtNode()                {}
func (s *XuitinueStatement) TokenPos() lexer.Position  { return s.Pos }
func (s *XuitinueStatement) nodeType() string          { return "XuitinueStatement" }

// ExpressionStatement wraps an expression used as a statement (e.g. function call).
type ExpressionStatement struct {
	Pos  lexer.Position
	Expr Expression
}

func (s *ExpressionStatement) stmtNode()                {}
func (s *ExpressionStatement) TokenPos() lexer.Position  { return s.Pos }
func (s *ExpressionStatement) nodeType() string          { return "ExpressionStatement" }

// BlockStatement: { stmts... }
type BlockStatement struct {
	Pos        lexer.Position
	Statements []Statement
}

func (s *BlockStatement) stmtNode()                {}
func (s *BlockStatement) TokenPos() lexer.Position  { return s.Pos }
func (s *BlockStatement) nodeType() string          { return "BlockStatement" }

// XuenStatement: xuen name(params) [returnType] { body }
type XuenStatement struct {
	Pos        lexer.Position
	Name       string
	Params     []Parameter
	ReturnType string // "" if void
	Body       *BlockStatement
}

type Parameter struct {
	Name     string
	TypeName string
}

func (s *XuenStatement) stmtNode()                {}
func (s *XuenStatement) TokenPos() lexer.Position  { return s.Pos }
func (s *XuenStatement) nodeType() string          { return "XuenStatement" }

// XuifStatement: xuif (cond) { body } [xuelse [xuif ...] { body }]
type XuifStatement struct {
	Pos         lexer.Position
	Condition   Expression
	Consequence *BlockStatement
	Alternative Statement // can be *XuifStatement or *BlockStatement or nil
}

func (s *XuifStatement) stmtNode()                {}
func (s *XuifStatement) TokenPos() lexer.Position  { return s.Pos }
func (s *XuifStatement) nodeType() string          { return "XuifStatement" }

// XuiorStatement: xuior (item xuin expr) { body }
type XuiorStatement struct {
	Pos      lexer.Position
	Variable string
	Iterable Expression
	Body     *BlockStatement
}

func (s *XuiorStatement) stmtNode()                {}
func (s *XuiorStatement) TokenPos() lexer.Position  { return s.Pos }
func (s *XuiorStatement) nodeType() string          { return "XuiorStatement" }

// XuileStatement: xuile (cond) { body }
type XuileStatement struct {
	Pos       lexer.Position
	Condition Expression
	Body      *BlockStatement
}

func (s *XuileStatement) stmtNode()                {}
func (s *XuileStatement) TokenPos() lexer.Position  { return s.Pos }
func (s *XuileStatement) nodeType() string          { return "XuileStatement" }

// XuiructStatement: xuiruct Name { fields }
type XuiructStatement struct {
	Pos    lexer.Position
	Name   string
	Fields []Field
}

type Field struct {
	Name     string
	TypeName string
}

func (s *XuiructStatement) stmtNode()                {}
func (s *XuiructStatement) TokenPos() lexer.Position  { return s.Pos }
func (s *XuiructStatement) nodeType() string          { return "XuiructStatement" }

// XuimplStatement: xuimpl Name { methods }
type XuimplStatement struct {
	Pos     lexer.Position
	Name    string
	Methods []*XuenStatement
}

func (s *XuimplStatement) stmtNode()                {}
func (s *XuimplStatement) TokenPos() lexer.Position  { return s.Pos }
func (s *XuimplStatement) nodeType() string          { return "XuimplStatement" }

// XuenumStatement: xuenum Name { variants }
type XuenumStatement struct {
	Pos      lexer.Position
	Name     string
	Variants []string
}

func (s *XuenumStatement) stmtNode()                {}
func (s *XuenumStatement) TokenPos() lexer.Position  { return s.Pos }
func (s *XuenumStatement) nodeType() string          { return "XuenumStatement" }

// XuimportStatement: xuimport "path"
type XuimportStatement struct {
	Pos  lexer.Position
	Path string
}

func (s *XuimportStatement) stmtNode()                {}
func (s *XuimportStatement) TokenPos() lexer.Position  { return s.Pos }
func (s *XuimportStatement) nodeType() string          { return "XuimportStatement" }

// XuiatchStatement: xuiatch (expr) { arms }
type XuiatchStatement struct {
	Pos   lexer.Position
	Value Expression
	Arms  []MatchArm
}

type MatchArm struct {
	Pattern Expression // can be literal, ident "_", etc.
	Body    Statement  // single expr statement or block
}

func (s *XuiatchStatement) stmtNode()                {}
func (s *XuiatchStatement) TokenPos() lexer.Position  { return s.Pos }
func (s *XuiatchStatement) nodeType() string          { return "XuiatchStatement" }

// --- Expressions ---

// Identifier: foo, bar, self
type Identifier struct {
	Pos   lexer.Position
	Value string
}

func (e *Identifier) exprNode()                {}
func (e *Identifier) TokenPos() lexer.Position { return e.Pos }
func (e *Identifier) nodeType() string         { return "Identifier" }

// IntegerLiteral: 42, 0xFF
type IntegerLiteral struct {
	Pos   lexer.Position
	Raw   string
	Value int64
}

func (e *IntegerLiteral) exprNode()                {}
func (e *IntegerLiteral) TokenPos() lexer.Position { return e.Pos }
func (e *IntegerLiteral) nodeType() string         { return "IntegerLiteral" }

// FloatLiteral: 3.14
type FloatLiteral struct {
	Pos   lexer.Position
	Raw   string
	Value float64
}

func (e *FloatLiteral) exprNode()                {}
func (e *FloatLiteral) TokenPos() lexer.Position { return e.Pos }
func (e *FloatLiteral) nodeType() string         { return "FloatLiteral" }

// StringLiteral: "hello"
type StringLiteral struct {
	Pos   lexer.Position
	Value string
}

func (e *StringLiteral) exprNode()                {}
func (e *StringLiteral) TokenPos() lexer.Position { return e.Pos }
func (e *StringLiteral) nodeType() string         { return "StringLiteral" }

// CharLiteral: 'a'
type CharLiteral struct {
	Pos   lexer.Position
	Value rune
}

func (e *CharLiteral) exprNode()                {}
func (e *CharLiteral) TokenPos() lexer.Position { return e.Pos }
func (e *CharLiteral) nodeType() string         { return "CharLiteral" }

// BoolLiteral: xuitru, xuinia
type BoolLiteral struct {
	Pos   lexer.Position
	Value bool
}

func (e *BoolLiteral) exprNode()                {}
func (e *BoolLiteral) TokenPos() lexer.Position { return e.Pos }
func (e *BoolLiteral) nodeType() string         { return "BoolLiteral" }

// NullLiteral: xuinull
type NullLiteral struct {
	Pos lexer.Position
}

func (e *NullLiteral) exprNode()                {}
func (e *NullLiteral) TokenPos() lexer.Position { return e.Pos }
func (e *NullLiteral) nodeType() string         { return "NullLiteral" }

// PrefixExpression: !expr, -expr
type PrefixExpression struct {
	Pos      lexer.Position
	Operator string
	Right    Expression
}

func (e *PrefixExpression) exprNode()                {}
func (e *PrefixExpression) TokenPos() lexer.Position { return e.Pos }
func (e *PrefixExpression) nodeType() string         { return "PrefixExpression" }

// InfixExpression: left op right
type InfixExpression struct {
	Pos      lexer.Position
	Left     Expression
	Operator string
	Right    Expression
}

func (e *InfixExpression) exprNode()                {}
func (e *InfixExpression) TokenPos() lexer.Position { return e.Pos }
func (e *InfixExpression) nodeType() string         { return "InfixExpression" }

// CallExpression: func(args...)
type CallExpression struct {
	Pos       lexer.Position
	Function  Expression
	Arguments []Expression
}

func (e *CallExpression) exprNode()                {}
func (e *CallExpression) TokenPos() lexer.Position { return e.Pos }
func (e *CallExpression) nodeType() string         { return "CallExpression" }

// MemberExpression: obj.field
type MemberExpression struct {
	Pos    lexer.Position
	Object Expression
	Member string
}

func (e *MemberExpression) exprNode()                {}
func (e *MemberExpression) TokenPos() lexer.Position { return e.Pos }
func (e *MemberExpression) nodeType() string         { return "MemberExpression" }

// IndexExpression: arr[index]
type IndexExpression struct {
	Pos   lexer.Position
	Left  Expression
	Index Expression
}

func (e *IndexExpression) exprNode()                {}
func (e *IndexExpression) TokenPos() lexer.Position { return e.Pos }
func (e *IndexExpression) nodeType() string         { return "IndexExpression" }

// ArrayLiteral: [1, 2, 3]
type ArrayLiteral struct {
	Pos      lexer.Position
	Elements []Expression
}

func (e *ArrayLiteral) exprNode()                {}
func (e *ArrayLiteral) TokenPos() lexer.Position { return e.Pos }
func (e *ArrayLiteral) nodeType() string         { return "ArrayLiteral" }

// RangeExpression: start..end
type RangeExpression struct {
	Pos   lexer.Position
	Start Expression
	End   Expression
}

func (e *RangeExpression) exprNode()                {}
func (e *RangeExpression) TokenPos() lexer.Position { return e.Pos }
func (e *RangeExpression) nodeType() string         { return "RangeExpression" }

// StructLiteral: Name { field = value, ... }
type StructLiteral struct {
	Pos    lexer.Position
	Name   string
	Fields []StructFieldValue
}

type StructFieldValue struct {
	Name  string
	Value Expression
}

func (e *StructLiteral) exprNode()                {}
func (e *StructLiteral) TokenPos() lexer.Position { return e.Pos }
func (e *StructLiteral) nodeType() string         { return "StructLiteral" }

// MapLiteral: {"key": value, ...}
type MapLiteral struct {
	Pos   lexer.Position
	Pairs []MapPair
}

type MapPair struct {
	Key   Expression
	Value Expression
}

func (e *MapLiteral) exprNode()                {}
func (e *MapLiteral) TokenPos() lexer.Position { return e.Pos }
func (e *MapLiteral) nodeType() string         { return "MapLiteral" }

// InterpolatedString: "hello {expr} world"
type InterpolatedString struct {
	Pos   lexer.Position
	Parts []Expression // alternating StringLiteral and expressions
}

func (e *InterpolatedString) exprNode()                {}
func (e *InterpolatedString) TokenPos() lexer.Position { return e.Pos }
func (e *InterpolatedString) nodeType() string         { return "InterpolatedString" }

// LambdaExpression: (a, b) => a + b  or  (a, b) => { body }
type LambdaExpression struct {
	Pos    lexer.Position
	Params []Parameter
	Body   Expression      // single expression
	Block  *BlockStatement // block body (one of Body/Block is set)
}

func (e *LambdaExpression) exprNode()                {}
func (e *LambdaExpression) TokenPos() lexer.Position { return e.Pos }
func (e *LambdaExpression) nodeType() string         { return "LambdaExpression" }

// TryStatement: try { body } catch (e) { handler }
type TryStatement struct {
	Pos       lexer.Position
	Body      *BlockStatement
	CatchVar  string
	CatchBody *BlockStatement
}

func (s *TryStatement) stmtNode()               {}
func (s *TryStatement) TokenPos() lexer.Position { return s.Pos }
func (s *TryStatement) nodeType() string         { return "TryStatement" }

// ThrowExpression: throw "error message"
type ThrowExpression struct {
	Pos   lexer.Position
	Value Expression
}

func (e *ThrowExpression) exprNode()                {}
func (e *ThrowExpression) TokenPos() lexer.Position { return e.Pos }
func (e *ThrowExpression) nodeType() string         { return "ThrowExpression" }

// AddressOfExpression: &variable
type AddressOfExpression struct {
	Pos   lexer.Position
	Value Expression
}

func (e *AddressOfExpression) exprNode()                {}
func (e *AddressOfExpression) TokenPos() lexer.Position { return e.Pos }
func (e *AddressOfExpression) nodeType() string         { return "AddressOfExpression" }

// DerefExpression: *pointer
type DerefExpression struct {
	Pos   lexer.Position
	Value Expression
}

func (e *DerefExpression) exprNode()                {}
func (e *DerefExpression) TokenPos() lexer.Position { return e.Pos }
func (e *DerefExpression) nodeType() string         { return "DerefExpression" }
