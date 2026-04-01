package parser

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/00000kkkkk/xusesosplusplus/lexer"
)

// Operator precedence levels
const (
	_ int = iota
	LOWEST
	OR          // ||
	AND         // &&
	EQUALS      // == !=
	LESSGREATER // < > <= >=
	SUM         // + -
	PRODUCT     // * / %
	PREFIX      // -x !x
	CALL        // func(x)
	INDEX       // arr[i]  obj.field
)

var precedences = map[lexer.TokenType]int{
	lexer.TOKEN_OR:        OR,
	lexer.TOKEN_AND:       AND,
	lexer.TOKEN_EQ:        EQUALS,
	lexer.TOKEN_NEQ:       EQUALS,
	lexer.TOKEN_LT:        LESSGREATER,
	lexer.TOKEN_GT:        LESSGREATER,
	lexer.TOKEN_LTE:       LESSGREATER,
	lexer.TOKEN_GTE:       LESSGREATER,
	lexer.TOKEN_PLUS:      SUM,
	lexer.TOKEN_MINUS:     SUM,
	lexer.TOKEN_STAR:      PRODUCT,
	lexer.TOKEN_SLASH:     PRODUCT,
	lexer.TOKEN_PERCENT:   PRODUCT,
	lexer.TOKEN_DOTDOT:    SUM, // range has same precedence as +/-
	lexer.TOKEN_LPAREN:    CALL,
	lexer.TOKEN_LBRACKET:  INDEX,
	lexer.TOKEN_DOT:       INDEX,
}

// ParseError represents a parser error.
type ParseError struct {
	Pos     lexer.Position
	Message string
}

func (e ParseError) Error() string {
	return fmt.Sprintf("%s: %s", e.Pos, e.Message)
}

// Parser parses tokens into an AST.
type Parser struct {
	tokens  []lexer.Token
	pos     int
	errors  []ParseError
}

// New creates a new parser from a token list.
func New(tokens []lexer.Token) *Parser {
	return &Parser{tokens: tokens, pos: 0}
}

// Parse parses the full program.
func (p *Parser) Parse() (*Program, []ParseError) {
	program := &Program{}

	for !p.isAtEnd() {
		stmt := p.parseStatement()
		if stmt != nil {
			program.Statements = append(program.Statements, stmt)
		}
	}

	return program, p.errors
}

// Errors returns accumulated parse errors.
func (p *Parser) Errors() []ParseError {
	return p.errors
}

// --- Token helpers ---

func (p *Parser) current() lexer.Token {
	if p.pos >= len(p.tokens) {
		return lexer.Token{Type: lexer.TOKEN_EOF}
	}
	return p.tokens[p.pos]
}

func (p *Parser) peek() lexer.Token {
	next := p.pos + 1
	if next >= len(p.tokens) {
		return lexer.Token{Type: lexer.TOKEN_EOF}
	}
	return p.tokens[next]
}

func (p *Parser) advance() lexer.Token {
	tok := p.current()
	if p.pos < len(p.tokens) {
		p.pos++
	}
	return tok
}

func (p *Parser) isAtEnd() bool {
	return p.current().Type == lexer.TOKEN_EOF
}

func (p *Parser) expect(t lexer.TokenType) lexer.Token {
	if p.current().Type == t {
		return p.advance()
	}
	p.errorf(p.current().Pos, "expected %s, got %s (%q)", t, p.current().Type, p.current().Literal)
	return p.current()
}

func (p *Parser) match(t lexer.TokenType) bool {
	if p.current().Type == t {
		p.advance()
		return true
	}
	return false
}

func (p *Parser) skipSemicolons() {
	for p.current().Type == lexer.TOKEN_SEMICOLON {
		p.advance()
	}
}

func (p *Parser) errorf(pos lexer.Position, format string, args ...interface{}) {
	p.errors = append(p.errors, ParseError{
		Pos:     pos,
		Message: fmt.Sprintf(format, args...),
	})
}

func (p *Parser) peekPrecedence() int {
	if prec, ok := precedences[p.current().Type]; ok {
		return prec
	}
	return LOWEST
}

// --- Statement parsing ---

func (p *Parser) parseStatement() Statement {
	p.skipSemicolons()
	if p.isAtEnd() {
		return nil
	}

	var stmt Statement
	switch p.current().Type {
	case lexer.TOKEN_XUET:
		stmt = p.parseXuetStatement()
	case lexer.TOKEN_XUIAR:
		stmt = p.parseXuiarStatement()
	case lexer.TOKEN_XUEN:
		stmt = p.parseXuenStatement()
	case lexer.TOKEN_XUETURN:
		stmt = p.parseXueturnStatement()
	case lexer.TOKEN_XUIEAK:
		stmt = p.parseXueakStatement()
	case lexer.TOKEN_XUITINUE:
		stmt = p.parseXuitinueStatement()
	case lexer.TOKEN_XUIF:
		stmt = p.parseXuifStatement()
	case lexer.TOKEN_XUIOR:
		stmt = p.parseXuiorStatement()
	case lexer.TOKEN_XUILE:
		stmt = p.parseXuileStatement()
	case lexer.TOKEN_XUIRUCT:
		stmt = p.parseXuiructStatement()
	case lexer.TOKEN_XUIMPL:
		stmt = p.parseXuimplStatement()
	case lexer.TOKEN_XUENUM:
		stmt = p.parseXuenumStatement()
	case lexer.TOKEN_XUIMPORT:
		stmt = p.parseXuimportStatement()
	case lexer.TOKEN_XUIATCH:
		stmt = p.parseXuiatchStatement()
	case lexer.TOKEN_XUTRY:
		stmt = p.parseTryStatement()
	default:
		stmt = p.parseExpressionOrAssignStatement()
	}

	p.skipSemicolons()
	return stmt
}

func (p *Parser) parseXuetStatement() Statement {
	pos := p.advance().Pos // consume xuet

	name := p.expect(lexer.TOKEN_IDENT).Literal

	// Optional type name
	typeName := ""
	if p.current().Type != lexer.TOKEN_ASSIGN {
		typeName = p.parseTypeName()
	}

	p.expect(lexer.TOKEN_ASSIGN)
	value := p.parseExpression(LOWEST)

	return &XuetStatement{
		Pos:      pos,
		Name:     name,
		TypeName: typeName,
		Value:    value,
	}
}

func (p *Parser) parseXuiarStatement() Statement {
	pos := p.advance().Pos // consume xuiar

	name := p.expect(lexer.TOKEN_IDENT).Literal

	typeName := ""
	if p.current().Type != lexer.TOKEN_ASSIGN {
		typeName = p.parseTypeName()
	}

	p.expect(lexer.TOKEN_ASSIGN)
	value := p.parseExpression(LOWEST)

	return &XuiarStatement{
		Pos:      pos,
		Name:     name,
		TypeName: typeName,
		Value:    value,
	}
}

func (p *Parser) parseXuenStatement() Statement {
	pos := p.advance().Pos // consume xuen

	name := p.expect(lexer.TOKEN_IDENT).Literal

	p.expect(lexer.TOKEN_LPAREN)
	params := p.parseParameterList()
	p.expect(lexer.TOKEN_RPAREN)

	// Optional return type (if next token is not {)
	returnType := ""
	if p.current().Type != lexer.TOKEN_LBRACE {
		returnType = p.parseTypeName()
	}

	body := p.parseBlockStatement()

	return &XuenStatement{
		Pos:        pos,
		Name:       name,
		Params:     params,
		ReturnType: returnType,
		Body:       body,
	}
}

func (p *Parser) parseParameterList() []Parameter {
	var params []Parameter

	if p.current().Type == lexer.TOKEN_RPAREN {
		return params
	}

	for {
		// Check for "xuiar" modifier
		if p.current().Type == lexer.TOKEN_XUIAR {
			p.advance() // skip xuiar modifier for now
		}

		name := p.expect(lexer.TOKEN_IDENT).Literal

		// "self" doesn't require a type
		typeName := ""
		if name != "self" && p.current().Type != lexer.TOKEN_COMMA && p.current().Type != lexer.TOKEN_RPAREN {
			typeName = p.parseTypeName()
		}

		params = append(params, Parameter{Name: name, TypeName: typeName})

		if !p.match(lexer.TOKEN_COMMA) {
			break
		}
	}

	return params
}

func (p *Parser) parseTypeName() string {
	// Handle nullable: ?type
	if p.current().Type == lexer.TOKEN_QUESTION {
		p.advance()
		return "?" + p.parseBaseTypeName()
	}
	// Handle array: []type
	if p.current().Type == lexer.TOKEN_LBRACKET && p.peek().Type == lexer.TOKEN_RBRACKET {
		p.advance() // [
		p.advance() // ]
		return "[]" + p.parseBaseTypeName()
	}
	return p.parseBaseTypeName()
}

func (p *Parser) parseBaseTypeName() string {
	tok := p.current()
	switch tok.Type {
	case lexer.TOKEN_INT_TYPE, lexer.TOKEN_INT8_TYPE, lexer.TOKEN_INT16_TYPE,
		lexer.TOKEN_INT32_TYPE, lexer.TOKEN_INT64_TYPE, lexer.TOKEN_UINT_TYPE,
		lexer.TOKEN_FLOAT_TYPE, lexer.TOKEN_FLOAT32_TYPE,
		lexer.TOKEN_BOOL_TYPE, lexer.TOKEN_STR_TYPE,
		lexer.TOKEN_CHAR_TYPE, lexer.TOKEN_BYTE_TYPE:
		p.advance()
		return tok.Literal
	case lexer.TOKEN_IDENT:
		p.advance()
		return tok.Literal
	default:
		p.errorf(tok.Pos, "expected type name, got %s (%q)", tok.Type, tok.Literal)
		return ""
	}
}

func (p *Parser) parseXueturnStatement() Statement {
	pos := p.advance().Pos // consume xueturn

	// Check if there's a value to return
	var value Expression
	if p.current().Type != lexer.TOKEN_SEMICOLON && p.current().Type != lexer.TOKEN_RBRACE && !p.isAtEnd() {
		value = p.parseExpression(LOWEST)
	}

	return &XueturnStatement{Pos: pos, Value: value}
}

func (p *Parser) parseXueakStatement() Statement {
	pos := p.advance().Pos
	return &XueakStatement{Pos: pos}
}

func (p *Parser) parseXuitinueStatement() Statement {
	pos := p.advance().Pos
	return &XuitinueStatement{Pos: pos}
}

func (p *Parser) parseXuifStatement() Statement {
	pos := p.advance().Pos // consume xuif

	p.expect(lexer.TOKEN_LPAREN)
	condition := p.parseExpression(LOWEST)
	p.expect(lexer.TOKEN_RPAREN)

	consequence := p.parseBlockStatement()

	var alternative Statement
	if p.match(lexer.TOKEN_XUELSE) {
		if p.current().Type == lexer.TOKEN_XUIF {
			alternative = p.parseXuifStatement()
		} else {
			alternative = p.parseBlockStatement()
		}
	}

	return &XuifStatement{
		Pos:         pos,
		Condition:   condition,
		Consequence: consequence,
		Alternative: alternative,
	}
}

func (p *Parser) parseXuiorStatement() Statement {
	pos := p.advance().Pos // consume xuior

	p.expect(lexer.TOKEN_LPAREN)
	varName := p.expect(lexer.TOKEN_IDENT).Literal
	p.expect(lexer.TOKEN_XUIN)
	iterable := p.parseExpression(LOWEST)
	p.expect(lexer.TOKEN_RPAREN)

	body := p.parseBlockStatement()

	return &XuiorStatement{
		Pos:      pos,
		Variable: varName,
		Iterable: iterable,
		Body:     body,
	}
}

func (p *Parser) parseXuileStatement() Statement {
	pos := p.advance().Pos // consume xuile

	p.expect(lexer.TOKEN_LPAREN)
	condition := p.parseExpression(LOWEST)
	p.expect(lexer.TOKEN_RPAREN)

	body := p.parseBlockStatement()

	return &XuileStatement{
		Pos:       pos,
		Condition: condition,
		Body:      body,
	}
}

func (p *Parser) parseXuiructStatement() Statement {
	pos := p.advance().Pos // consume xuiruct
	name := p.expect(lexer.TOKEN_IDENT).Literal

	p.expect(lexer.TOKEN_LBRACE)
	p.skipSemicolons()

	var fields []Field
	for p.current().Type != lexer.TOKEN_RBRACE && !p.isAtEnd() {
		fieldName := p.expect(lexer.TOKEN_IDENT).Literal
		fieldType := p.parseTypeName()
		fields = append(fields, Field{Name: fieldName, TypeName: fieldType})
		p.skipSemicolons()
	}

	p.expect(lexer.TOKEN_RBRACE)

	return &XuiructStatement{
		Pos:    pos,
		Name:   name,
		Fields: fields,
	}
}

func (p *Parser) parseXuimplStatement() Statement {
	pos := p.advance().Pos // consume xuimpl
	name := p.expect(lexer.TOKEN_IDENT).Literal

	p.expect(lexer.TOKEN_LBRACE)
	p.skipSemicolons()

	var methods []*XuenStatement
	for p.current().Type != lexer.TOKEN_RBRACE && !p.isAtEnd() {
		if p.current().Type == lexer.TOKEN_XUEN {
			method := p.parseXuenStatement()
			if m, ok := method.(*XuenStatement); ok {
				methods = append(methods, m)
			}
		} else {
			p.errorf(p.current().Pos, "expected xuen inside xuimpl, got %s", p.current().Type)
			p.advance()
		}
		p.skipSemicolons()
	}

	p.expect(lexer.TOKEN_RBRACE)

	return &XuimplStatement{
		Pos:     pos,
		Name:    name,
		Methods: methods,
	}
}

func (p *Parser) parseXuenumStatement() Statement {
	pos := p.advance().Pos // consume xuenum
	name := p.expect(lexer.TOKEN_IDENT).Literal

	p.expect(lexer.TOKEN_LBRACE)
	p.skipSemicolons()

	var variants []string
	for p.current().Type != lexer.TOKEN_RBRACE && !p.isAtEnd() {
		v := p.expect(lexer.TOKEN_IDENT).Literal
		variants = append(variants, v)
		p.match(lexer.TOKEN_COMMA)
		p.skipSemicolons()
	}

	p.expect(lexer.TOKEN_RBRACE)

	return &XuenumStatement{
		Pos:      pos,
		Name:     name,
		Variants: variants,
	}
}

func (p *Parser) parseXuimportStatement() Statement {
	pos := p.advance().Pos // consume xuimport
	path := p.expect(lexer.TOKEN_STRING).Literal

	return &XuimportStatement{Pos: pos, Path: path}
}

func (p *Parser) parseXuiatchStatement() Statement {
	pos := p.advance().Pos // consume xuiatch

	p.expect(lexer.TOKEN_LPAREN)
	value := p.parseExpression(LOWEST)
	p.expect(lexer.TOKEN_RPAREN)

	p.expect(lexer.TOKEN_LBRACE)
	p.skipSemicolons()

	var arms []MatchArm
	for p.current().Type != lexer.TOKEN_RBRACE && !p.isAtEnd() {
		pattern := p.parseExpression(LOWEST)
		p.expect(lexer.TOKEN_FAT_ARROW)

		var body Statement
		if p.current().Type == lexer.TOKEN_LBRACE {
			body = p.parseBlockStatement()
		} else {
			expr := p.parseExpression(LOWEST)
			body = &ExpressionStatement{Pos: expr.TokenPos(), Expr: expr}
		}

		arms = append(arms, MatchArm{Pattern: pattern, Body: body})
		p.skipSemicolons()
	}

	p.expect(lexer.TOKEN_RBRACE)

	return &XuiatchStatement{
		Pos:   pos,
		Value: value,
		Arms:  arms,
	}
}

func (p *Parser) parseBlockStatement() *BlockStatement {
	pos := p.expect(lexer.TOKEN_LBRACE).Pos
	p.skipSemicolons()

	var stmts []Statement
	for p.current().Type != lexer.TOKEN_RBRACE && !p.isAtEnd() {
		stmt := p.parseStatement()
		if stmt != nil {
			stmts = append(stmts, stmt)
		}
	}

	p.expect(lexer.TOKEN_RBRACE)

	return &BlockStatement{Pos: pos, Statements: stmts}
}

func (p *Parser) parseExpressionOrAssignStatement() Statement {
	pos := p.current().Pos
	expr := p.parseExpression(LOWEST)

	// Check for assignment: expr = value
	if p.current().Type == lexer.TOKEN_ASSIGN {
		p.advance()
		value := p.parseExpression(LOWEST)
		return &AssignStatement{Pos: pos, Target: expr, Value: value}
	}

	return &ExpressionStatement{Pos: pos, Expr: expr}
}

// --- Expression parsing (Pratt parser) ---

func (p *Parser) parseExpression(precedence int) Expression {
	left := p.parsePrefixExpression()
	if left == nil {
		return nil
	}

	for !p.isAtEnd() && precedence < p.peekPrecedence() {
		switch p.current().Type {
		case lexer.TOKEN_PLUS, lexer.TOKEN_MINUS, lexer.TOKEN_STAR,
			lexer.TOKEN_SLASH, lexer.TOKEN_PERCENT,
			lexer.TOKEN_EQ, lexer.TOKEN_NEQ,
			lexer.TOKEN_LT, lexer.TOKEN_GT, lexer.TOKEN_LTE, lexer.TOKEN_GTE,
			lexer.TOKEN_AND, lexer.TOKEN_OR:
			left = p.parseInfixExpression(left)
		case lexer.TOKEN_DOTDOT:
			left = p.parseRangeExpression(left)
		case lexer.TOKEN_LPAREN:
			left = p.parseCallExpression(left)
		case lexer.TOKEN_LBRACKET:
			left = p.parseIndexExpression(left)
		case lexer.TOKEN_DOT:
			left = p.parseMemberExpression(left)
		default:
			return left
		}
	}

	return left
}

func (p *Parser) parsePrefixExpression() Expression {
	tok := p.current()

	switch tok.Type {
	case lexer.TOKEN_IDENT:
		p.advance()
		// Check for struct literal: Name { ... }
		if p.current().Type == lexer.TOKEN_LBRACE {
			return p.parseStructLiteral(tok)
		}
		return &Identifier{Pos: tok.Pos, Value: tok.Literal}

	case lexer.TOKEN_INT:
		p.advance()
		val := parseInt(tok.Literal)
		return &IntegerLiteral{Pos: tok.Pos, Raw: tok.Literal, Value: val}

	case lexer.TOKEN_FLOAT:
		p.advance()
		val, _ := strconv.ParseFloat(tok.Literal, 64)
		return &FloatLiteral{Pos: tok.Pos, Raw: tok.Literal, Value: val}

	case lexer.TOKEN_INTERP_START:
		return p.parseInterpolatedString()

	case lexer.TOKEN_STRING:
		p.advance()
		return &StringLiteral{Pos: tok.Pos, Value: tok.Literal}

	case lexer.TOKEN_CHAR:
		p.advance()
		r := []rune(tok.Literal)
		var ch rune
		if len(r) > 0 {
			ch = r[0]
		}
		return &CharLiteral{Pos: tok.Pos, Value: ch}

	case lexer.TOKEN_XUITRU:
		p.advance()
		return &BoolLiteral{Pos: tok.Pos, Value: true}

	case lexer.TOKEN_XUINIA:
		p.advance()
		return &BoolLiteral{Pos: tok.Pos, Value: false}

	case lexer.TOKEN_XUINULL:
		p.advance()
		return &NullLiteral{Pos: tok.Pos}

	case lexer.TOKEN_MINUS, lexer.TOKEN_NOT:
		p.advance()
		right := p.parseExpression(PREFIX)
		return &PrefixExpression{Pos: tok.Pos, Operator: tok.Literal, Right: right}

	case lexer.TOKEN_AMPERSAND:
		p.advance()
		right := p.parseExpression(PREFIX)
		return &AddressOfExpression{Pos: tok.Pos, Value: right}

	case lexer.TOKEN_STAR:
		// * in prefix position is dereference
		p.advance()
		right := p.parseExpression(PREFIX)
		return &DerefExpression{Pos: tok.Pos, Value: right}

	case lexer.TOKEN_LPAREN:
		if p.isLambda() {
			return p.parseLambda()
		}
		p.advance()
		expr := p.parseExpression(LOWEST)
		p.expect(lexer.TOKEN_RPAREN)
		return expr

	case lexer.TOKEN_XUTHROW:
		p.advance()
		val := p.parseExpression(LOWEST)
		return &ThrowExpression{Pos: tok.Pos, Value: val}

	case lexer.TOKEN_LBRACKET:
		return p.parseArrayLiteral()

	case lexer.TOKEN_LBRACE:
		return p.parseMapLiteral()

	default:
		p.errorf(tok.Pos, "unexpected token %s (%q)", tok.Type, tok.Literal)
		p.advance()
		return nil
	}
}

func (p *Parser) parseInfixExpression(left Expression) Expression {
	tok := p.advance()
	prec := precedences[tok.Type]
	right := p.parseExpression(prec)

	return &InfixExpression{
		Pos:      tok.Pos,
		Left:     left,
		Operator: tok.Literal,
		Right:    right,
	}
}

func (p *Parser) parseRangeExpression(left Expression) Expression {
	tok := p.advance() // consume ..
	right := p.parseExpression(SUM)

	return &RangeExpression{
		Pos:   tok.Pos,
		Start: left,
		End:   right,
	}
}

func (p *Parser) parseCallExpression(function Expression) Expression {
	pos := p.advance().Pos // consume (

	var args []Expression
	if p.current().Type != lexer.TOKEN_RPAREN {
		for {
			arg := p.parseExpression(LOWEST)
			if arg != nil {
				args = append(args, arg)
			}
			if !p.match(lexer.TOKEN_COMMA) {
				break
			}
		}
	}

	p.expect(lexer.TOKEN_RPAREN)

	return &CallExpression{
		Pos:       pos,
		Function:  function,
		Arguments: args,
	}
}

func (p *Parser) parseIndexExpression(left Expression) Expression {
	pos := p.advance().Pos // consume [
	index := p.parseExpression(LOWEST)
	p.expect(lexer.TOKEN_RBRACKET)

	return &IndexExpression{
		Pos:   pos,
		Left:  left,
		Index: index,
	}
}

func (p *Parser) parseMemberExpression(left Expression) Expression {
	p.advance() // consume .
	member := p.expect(lexer.TOKEN_IDENT)

	return &MemberExpression{
		Pos:    member.Pos,
		Object: left,
		Member: member.Literal,
	}
}

func (p *Parser) parseArrayLiteral() Expression {
	pos := p.advance().Pos // consume [

	var elements []Expression
	if p.current().Type != lexer.TOKEN_RBRACKET {
		for {
			elem := p.parseExpression(LOWEST)
			if elem != nil {
				elements = append(elements, elem)
			}
			if !p.match(lexer.TOKEN_COMMA) {
				break
			}
		}
	}

	p.expect(lexer.TOKEN_RBRACKET)

	return &ArrayLiteral{Pos: pos, Elements: elements}
}

func (p *Parser) parseStructLiteral(nameTok lexer.Token) Expression {
	pos := nameTok.Pos
	p.advance() // consume {
	p.skipSemicolons()

	var fields []StructFieldValue
	for p.current().Type != lexer.TOKEN_RBRACE && !p.isAtEnd() {
		fieldName := p.expect(lexer.TOKEN_IDENT).Literal
		p.expect(lexer.TOKEN_ASSIGN)
		value := p.parseExpression(LOWEST)
		fields = append(fields, StructFieldValue{Name: fieldName, Value: value})
		p.match(lexer.TOKEN_COMMA)
		p.skipSemicolons()
	}

	p.expect(lexer.TOKEN_RBRACE)

	return &StructLiteral{
		Pos:    pos,
		Name:   nameTok.Literal,
		Fields: fields,
	}
}

// isLambda checks if the current ( starts a lambda expression.
// Heuristic: () => ... or (ident, ...) => ... or (ident type, ...) => ...
func (p *Parser) isLambda() bool {
	// Save position
	saved := p.pos
	defer func() { p.pos = saved }()

	p.advance() // skip (

	// () => ... — empty params lambda
	if p.current().Type == lexer.TOKEN_RPAREN {
		p.advance()
		return p.current().Type == lexer.TOKEN_FAT_ARROW
	}

	// Look for pattern: ident [type] [, ident [type]]* ) =>
	for {
		if p.current().Type != lexer.TOKEN_IDENT {
			return false
		}
		p.advance() // skip ident

		// Optional type
		if p.current().Type != lexer.TOKEN_COMMA && p.current().Type != lexer.TOKEN_RPAREN {
			p.advance() // skip type
		}

		if p.current().Type == lexer.TOKEN_RPAREN {
			p.advance()
			return p.current().Type == lexer.TOKEN_FAT_ARROW
		}
		if p.current().Type == lexer.TOKEN_COMMA {
			p.advance()
			continue
		}
		return false
	}
}

func (p *Parser) parseLambda() Expression {
	pos := p.current().Pos
	p.advance() // skip (

	var params []Parameter
	if p.current().Type != lexer.TOKEN_RPAREN {
		for {
			name := p.expect(lexer.TOKEN_IDENT).Literal
			typeName := ""
			if p.current().Type != lexer.TOKEN_COMMA && p.current().Type != lexer.TOKEN_RPAREN {
				typeName = p.parseTypeName()
			}
			params = append(params, Parameter{Name: name, TypeName: typeName})
			if !p.match(lexer.TOKEN_COMMA) {
				break
			}
		}
	}
	p.expect(lexer.TOKEN_RPAREN)
	p.expect(lexer.TOKEN_FAT_ARROW)

	// Block body or expression body
	if p.current().Type == lexer.TOKEN_LBRACE {
		block := p.parseBlockStatement()
		return &LambdaExpression{Pos: pos, Params: params, Block: block}
	}

	body := p.parseExpression(LOWEST)
	return &LambdaExpression{Pos: pos, Params: params, Body: body}
}

func (p *Parser) parseTryStatement() Statement {
	pos := p.advance().Pos // consume xutry
	body := p.parseBlockStatement()

	p.expect(lexer.TOKEN_XUCATCH)
	p.expect(lexer.TOKEN_LPAREN)
	catchVar := p.expect(lexer.TOKEN_IDENT).Literal
	p.expect(lexer.TOKEN_RPAREN)
	catchBody := p.parseBlockStatement()

	return &TryStatement{
		Pos:       pos,
		Body:      body,
		CatchVar:  catchVar,
		CatchBody: catchBody,
	}
}

func (p *Parser) parseMapLiteral() Expression {
	pos := p.advance().Pos // consume {
	p.skipSemicolons()

	var pairs []MapPair
	if p.current().Type != lexer.TOKEN_RBRACE {
		for {
			key := p.parseExpression(LOWEST)
			p.expect(lexer.TOKEN_COLON)
			value := p.parseExpression(LOWEST)
			pairs = append(pairs, MapPair{Key: key, Value: value})
			p.skipSemicolons()
			if !p.match(lexer.TOKEN_COMMA) {
				p.skipSemicolons()
				break
			}
			p.skipSemicolons()
		}
	}

	p.expect(lexer.TOKEN_RBRACE)

	return &MapLiteral{Pos: pos, Pairs: pairs}
}

func (p *Parser) parseInterpolatedString() Expression {
	pos := p.advance().Pos // consume INTERP_START

	var parts []Expression
	for p.current().Type != lexer.TOKEN_INTERP_END && !p.isAtEnd() {
		if p.current().Type == lexer.TOKEN_STRING {
			tok := p.advance()
			if tok.Literal != "" {
				parts = append(parts, &StringLiteral{Pos: tok.Pos, Value: tok.Literal})
			}
		} else {
			expr := p.parseExpression(LOWEST)
			if expr != nil {
				parts = append(parts, expr)
			}
		}
	}

	p.match(lexer.TOKEN_INTERP_END) // consume INTERP_END

	if len(parts) == 1 {
		return parts[0]
	}
	return &InterpolatedString{Pos: pos, Parts: parts}
}

// parseInt parses integer literals including hex, binary, octal.
func parseInt(s string) int64 {
	s = strings.ReplaceAll(s, "_", "")
	if strings.HasPrefix(s, "0x") || strings.HasPrefix(s, "0X") {
		v, _ := strconv.ParseInt(s[2:], 16, 64)
		return v
	}
	if strings.HasPrefix(s, "0b") || strings.HasPrefix(s, "0B") {
		v, _ := strconv.ParseInt(s[2:], 2, 64)
		return v
	}
	if strings.HasPrefix(s, "0o") || strings.HasPrefix(s, "0O") {
		v, _ := strconv.ParseInt(s[2:], 8, 64)
		return v
	}
	v, _ := strconv.ParseInt(s, 10, 64)
	return v
}
