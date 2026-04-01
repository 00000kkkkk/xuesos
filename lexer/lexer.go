package lexer

import (
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"
)

// LexError represents an error encountered during lexing.
type LexError struct {
	Pos     Position
	Message string
}

func (e LexError) Error() string {
	return fmt.Sprintf("%s: %s", e.Pos, e.Message)
}

// Lexer tokenizes Xuesos++ source code.
type Lexer struct {
	filename    string
	src         string
	pos         int // current byte offset
	line        int // current line (1-based)
	col         int // current column in runes (1-based)
	errors      []LexError
	newlineSeen bool // whether a newline was seen since last token
	pending     []Token // buffered tokens (for string interpolation)
}

// New creates a new Lexer for the given source.
func New(filename, src string) *Lexer {
	return &Lexer{
		filename: filename,
		src:      src,
		pos:      0,
		line:     1,
		col:      1,
	}
}

func (l *Lexer) currentPos() Position {
	return Position{
		File:   l.filename,
		Line:   l.line,
		Column: l.col,
		Offset: l.pos,
	}
}

func (l *Lexer) isAtEnd() bool {
	return l.pos >= len(l.src)
}

func (l *Lexer) peek() rune {
	if l.isAtEnd() {
		return 0
	}
	r, _ := utf8.DecodeRuneInString(l.src[l.pos:])
	return r
}

func (l *Lexer) peekNext() rune {
	if l.isAtEnd() {
		return 0
	}
	_, size := utf8.DecodeRuneInString(l.src[l.pos:])
	next := l.pos + size
	if next >= len(l.src) {
		return 0
	}
	r, _ := utf8.DecodeRuneInString(l.src[next:])
	return r
}

func (l *Lexer) advance() rune {
	if l.isAtEnd() {
		return 0
	}
	r, size := utf8.DecodeRuneInString(l.src[l.pos:])
	l.pos += size
	if r == '\n' {
		l.line++
		l.col = 1
	} else {
		l.col++
	}
	return r
}

func (l *Lexer) match(expected rune) bool {
	if l.peek() == expected {
		l.advance()
		return true
	}
	return false
}

func (l *Lexer) errorf(pos Position, format string, args ...interface{}) {
	l.errors = append(l.errors, LexError{
		Pos:     pos,
		Message: fmt.Sprintf(format, args...),
	})
}

func (l *Lexer) makeToken(typ TokenType, literal string, pos Position) Token {
	return Token{Type: typ, Literal: literal, Pos: pos}
}

func (l *Lexer) skipWhitespaceAndComments() {
	for !l.isAtEnd() {
		ch := l.peek()
		switch {
		case ch == ' ' || ch == '\t' || ch == '\r':
			l.advance()
		case ch == '\n':
			l.newlineSeen = true
			l.advance()
		case ch == '/' && l.peekNext() == '/':
			l.skipLineComment()
		case ch == '/' && l.peekNext() == '*':
			l.skipBlockComment()
		default:
			return
		}
	}
}

func (l *Lexer) skipLineComment() {
	l.advance() // /
	l.advance() // /
	for !l.isAtEnd() && l.peek() != '\n' {
		l.advance()
	}
	// Don't consume the newline here — the main loop will handle it and set newlineSeen
}

func (l *Lexer) skipBlockComment() {
	startPos := l.currentPos()
	l.advance() // /
	l.advance() // *
	for !l.isAtEnd() {
		if l.peek() == '*' && l.peekNext() == '/' {
			l.advance() // *
			l.advance() // /
			return
		}
		if l.peek() == '\n' {
			l.newlineSeen = true
		}
		l.advance()
	}
	l.errorf(startPos, "unterminated block comment")
}

// NextToken returns the next raw token (no semicolon insertion).
func (l *Lexer) NextToken() Token {
	// Drain pending buffer first (from string interpolation)
	if len(l.pending) > 0 {
		tok := l.pending[0]
		l.pending = l.pending[1:]
		return tok
	}
	return l.nextRawToken()
}

// nextRawToken scans the next token from source (used internally and for interpolation).
func (l *Lexer) nextRawToken() Token {
	l.skipWhitespaceAndComments()

	if l.isAtEnd() {
		return l.makeToken(TOKEN_EOF, "", l.currentPos())
	}

	startPos := l.currentPos()
	ch := l.advance()

	switch ch {
	case '(':
		return l.makeToken(TOKEN_LPAREN, "(", startPos)
	case ')':
		return l.makeToken(TOKEN_RPAREN, ")", startPos)
	case '{':
		return l.makeToken(TOKEN_LBRACE, "{", startPos)
	case '}':
		return l.makeToken(TOKEN_RBRACE, "}", startPos)
	case '[':
		return l.makeToken(TOKEN_LBRACKET, "[", startPos)
	case ']':
		return l.makeToken(TOKEN_RBRACKET, "]", startPos)
	case ':':
		return l.makeToken(TOKEN_COLON, ":", startPos)
	case ',':
		return l.makeToken(TOKEN_COMMA, ",", startPos)
	case '?':
		return l.makeToken(TOKEN_QUESTION, "?", startPos)
	case '+':
		return l.makeToken(TOKEN_PLUS, "+", startPos)
	case '*':
		return l.makeToken(TOKEN_STAR, "*", startPos)
	case '/':
		return l.makeToken(TOKEN_SLASH, "/", startPos)
	case '%':
		return l.makeToken(TOKEN_PERCENT, "%", startPos)

	case '-':
		return l.makeToken(TOKEN_MINUS, "-", startPos)

	case '.':
		if l.match('.') {
			return l.makeToken(TOKEN_DOTDOT, "..", startPos)
		}
		return l.makeToken(TOKEN_DOT, ".", startPos)

	case '=':
		if l.match('=') {
			return l.makeToken(TOKEN_EQ, "==", startPos)
		}
		if l.match('>') {
			return l.makeToken(TOKEN_FAT_ARROW, "=>", startPos)
		}
		return l.makeToken(TOKEN_ASSIGN, "=", startPos)

	case '!':
		if l.match('=') {
			return l.makeToken(TOKEN_NEQ, "!=", startPos)
		}
		return l.makeToken(TOKEN_NOT, "!", startPos)

	case '<':
		if l.match('=') {
			return l.makeToken(TOKEN_LTE, "<=", startPos)
		}
		return l.makeToken(TOKEN_LT, "<", startPos)

	case '>':
		if l.match('=') {
			return l.makeToken(TOKEN_GTE, ">=", startPos)
		}
		return l.makeToken(TOKEN_GT, ">", startPos)

	case '&':
		if l.match('&') {
			return l.makeToken(TOKEN_AND, "&&", startPos)
		}
		l.errorf(startPos, "unexpected character '&', did you mean '&&'?")
		return l.makeToken(TOKEN_ILLEGAL, "&", startPos)

	case '|':
		if l.match('|') {
			return l.makeToken(TOKEN_OR, "||", startPos)
		}
		l.errorf(startPos, "unexpected character '|', did you mean '||'?")
		return l.makeToken(TOKEN_ILLEGAL, "|", startPos)

	case '"':
		return l.scanString(startPos)

	case '\'':
		return l.scanChar(startPos)

	default:
		if isDigit(ch) {
			return l.scanNumber(startPos, ch)
		}
		if isIdentStart(ch) {
			return l.scanIdentifier(startPos, ch)
		}
		l.errorf(startPos, "unexpected character %q", ch)
		return l.makeToken(TOKEN_ILLEGAL, string(ch), startPos)
	}
}

func (l *Lexer) scanString(startPos Position) Token {
	var buf strings.Builder
	hasInterp := false
	var tokens []Token

	for !l.isAtEnd() {
		ch := l.peek()
		if ch == '"' {
			l.advance()
			if !hasInterp {
				return l.makeToken(TOKEN_STRING, buf.String(), startPos)
			}
			// Emit final string part + INTERP_END
			tokens = append(tokens, l.makeToken(TOKEN_STRING, buf.String(), l.currentPos()))
			tokens = append(tokens, l.makeToken(TOKEN_INTERP_END, "", l.currentPos()))
			// Return first token, buffer the rest
			first := tokens[0]
			l.pending = append(l.pending, tokens[1:]...)
			return first
		}
		if ch == '\n' {
			l.errorf(startPos, "unterminated string literal")
			return l.makeToken(TOKEN_STRING, buf.String(), startPos)
		}
		if ch == '\\' {
			l.advance()
			escaped := l.scanEscape(startPos)
			buf.WriteRune(escaped)
			continue
		}
		if ch == '{' {
			l.advance()
			// Start interpolation
			if !hasInterp {
				hasInterp = true
				tokens = append(tokens, l.makeToken(TOKEN_INTERP_START, "", startPos))
			}
			// Emit string part so far
			tokens = append(tokens, l.makeToken(TOKEN_STRING, buf.String(), l.currentPos()))
			buf.Reset()
			// Lex expression tokens until matching }
			depth := 1
			for !l.isAtEnd() && depth > 0 {
				l.skipWhitespaceAndComments()
				if l.isAtEnd() {
					break
				}
				if l.peek() == '}' {
					depth--
					if depth == 0 {
						l.advance()
						break
					}
				}
				if l.peek() == '{' {
					depth++
				}
				tok := l.nextRawToken()
				tokens = append(tokens, tok)
			}
			continue
		}
		buf.WriteRune(ch)
		l.advance()
	}
	l.errorf(startPos, "unterminated string literal")
	return l.makeToken(TOKEN_STRING, buf.String(), startPos)
}

func (l *Lexer) scanEscape(stringStart Position) rune {
	if l.isAtEnd() {
		l.errorf(stringStart, "unterminated escape sequence")
		return utf8.RuneError
	}
	ch := l.advance()
	switch ch {
	case 'n':
		return '\n'
	case 't':
		return '\t'
	case 'r':
		return '\r'
	case '\\':
		return '\\'
	case '"':
		return '"'
	case '\'':
		return '\''
	case '0':
		return '\x00'
	case 'u':
		return l.scanUnicodeEscape(4, stringStart)
	case 'U':
		return l.scanUnicodeEscape(8, stringStart)
	default:
		l.errorf(l.currentPos(), "unknown escape sequence '\\%c'", ch)
		return ch
	}
}

func (l *Lexer) scanUnicodeEscape(digits int, stringStart Position) rune {
	var value rune
	for i := 0; i < digits; i++ {
		if l.isAtEnd() {
			l.errorf(stringStart, "unterminated unicode escape sequence")
			return utf8.RuneError
		}
		ch := l.peek()
		if !isHexDigit(ch) {
			l.errorf(l.currentPos(), "invalid hex digit %q in unicode escape", ch)
			return utf8.RuneError
		}
		l.advance()
		value = value*16 + hexVal(ch)
	}
	if !utf8.ValidRune(value) {
		l.errorf(stringStart, "invalid unicode codepoint U+%04X", value)
		return utf8.RuneError
	}
	return value
}

func (l *Lexer) scanChar(startPos Position) Token {
	if l.isAtEnd() {
		l.errorf(startPos, "unterminated character literal")
		return l.makeToken(TOKEN_CHAR, "", startPos)
	}

	var ch rune
	if l.peek() == '\\' {
		l.advance()
		ch = l.scanEscape(startPos)
	} else if l.peek() == '\'' {
		l.errorf(startPos, "empty character literal")
		l.advance()
		return l.makeToken(TOKEN_CHAR, "", startPos)
	} else {
		ch = l.advance()
	}

	if !l.match('\'') {
		l.errorf(startPos, "unterminated character literal (expected closing quote)")
		// consume until closing quote or end of line
		for !l.isAtEnd() && l.peek() != '\'' && l.peek() != '\n' {
			l.advance()
		}
		if l.peek() == '\'' {
			l.advance()
		}
	}

	return l.makeToken(TOKEN_CHAR, string(ch), startPos)
}

func (l *Lexer) scanNumber(startPos Position, first rune) Token {
	var buf strings.Builder
	buf.WriteRune(first)

	// Check for 0x, 0b, 0o prefixes
	if first == '0' && !l.isAtEnd() {
		next := l.peek()
		switch next {
		case 'x', 'X':
			buf.WriteRune(l.advance())
			return l.scanHexNumber(startPos, &buf)
		case 'b', 'B':
			buf.WriteRune(l.advance())
			return l.scanBinaryNumber(startPos, &buf)
		case 'o', 'O':
			buf.WriteRune(l.advance())
			return l.scanOctalNumber(startPos, &buf)
		}
	}

	// Decimal digits
	for !l.isAtEnd() && (isDigit(l.peek()) || l.peek() == '_') {
		ch := l.advance()
		if ch != '_' {
			buf.WriteRune(ch)
		}
	}

	// Check for float: dot followed by digit
	if !l.isAtEnd() && l.peek() == '.' && isDigit(l.peekNext()) {
		buf.WriteRune(l.advance()) // .
		for !l.isAtEnd() && (isDigit(l.peek()) || l.peek() == '_') {
			ch := l.advance()
			if ch != '_' {
				buf.WriteRune(ch)
			}
		}
		// Exponent
		l.scanExponent(&buf)
		return l.makeToken(TOKEN_FLOAT, buf.String(), startPos)
	}

	// Check for exponent without dot (e.g. 1e10)
	if !l.isAtEnd() && (l.peek() == 'e' || l.peek() == 'E') {
		l.scanExponent(&buf)
		return l.makeToken(TOKEN_FLOAT, buf.String(), startPos)
	}

	return l.makeToken(TOKEN_INT, buf.String(), startPos)
}

func (l *Lexer) scanExponent(buf *strings.Builder) {
	if l.isAtEnd() || (l.peek() != 'e' && l.peek() != 'E') {
		return
	}
	buf.WriteRune(l.advance()) // e or E
	if !l.isAtEnd() && (l.peek() == '+' || l.peek() == '-') {
		buf.WriteRune(l.advance())
	}
	for !l.isAtEnd() && isDigit(l.peek()) {
		buf.WriteRune(l.advance())
	}
}

func (l *Lexer) scanHexNumber(startPos Position, buf *strings.Builder) Token {
	if l.isAtEnd() || !isHexDigit(l.peek()) {
		l.errorf(startPos, "expected hex digit after '0x'")
		return l.makeToken(TOKEN_INT, buf.String(), startPos)
	}
	for !l.isAtEnd() && (isHexDigit(l.peek()) || l.peek() == '_') {
		ch := l.advance()
		if ch != '_' {
			buf.WriteRune(ch)
		}
	}
	return l.makeToken(TOKEN_INT, buf.String(), startPos)
}

func (l *Lexer) scanBinaryNumber(startPos Position, buf *strings.Builder) Token {
	if l.isAtEnd() || (l.peek() != '0' && l.peek() != '1') {
		l.errorf(startPos, "expected binary digit after '0b'")
		return l.makeToken(TOKEN_INT, buf.String(), startPos)
	}
	for !l.isAtEnd() && (l.peek() == '0' || l.peek() == '1' || l.peek() == '_') {
		ch := l.advance()
		if ch != '_' {
			buf.WriteRune(ch)
		}
	}
	return l.makeToken(TOKEN_INT, buf.String(), startPos)
}

func (l *Lexer) scanOctalNumber(startPos Position, buf *strings.Builder) Token {
	if l.isAtEnd() || l.peek() < '0' || l.peek() > '7' {
		l.errorf(startPos, "expected octal digit after '0o'")
		return l.makeToken(TOKEN_INT, buf.String(), startPos)
	}
	for !l.isAtEnd() && ((l.peek() >= '0' && l.peek() <= '7') || l.peek() == '_') {
		ch := l.advance()
		if ch != '_' {
			buf.WriteRune(ch)
		}
	}
	return l.makeToken(TOKEN_INT, buf.String(), startPos)
}

func (l *Lexer) scanIdentifier(startPos Position, first rune) Token {
	var buf strings.Builder
	buf.WriteRune(first)
	for !l.isAtEnd() && isIdentPart(l.peek()) {
		buf.WriteRune(l.advance())
	}
	literal := buf.String()
	typ := LookupIdent(literal)
	return l.makeToken(typ, literal, startPos)
}

// ScanAll lexes the entire source, returning all tokens and any errors.
// Performs automatic semicolon insertion (Go-style).
func (l *Lexer) ScanAll() ([]Token, []LexError) {
	var tokens []Token
	var prevType TokenType

	for {
		l.newlineSeen = false
		tok := l.NextToken()

		// Auto-insert semicolon if newline was seen and previous token triggers it
		if l.newlineSeen && semicolonPreceding[prevType] {
			semiPos := tokens[len(tokens)-1].Pos
			semiPos.Column++ // place it just after the previous token
			tokens = append(tokens, Token{
				Type:    TOKEN_SEMICOLON,
				Literal: "\n",
				Pos:     semiPos,
			})
		}

		tokens = append(tokens, tok)

		if tok.Type == TOKEN_EOF {
			// Insert final semicolon if needed
			if semicolonPreceding[prevType] && len(tokens) >= 2 {
				semi := Token{
					Type:    TOKEN_SEMICOLON,
					Literal: "\n",
					Pos:     tok.Pos,
				}
				// Insert before EOF
				tokens = append(tokens[:len(tokens)-1], semi, tok)
			}
			break
		}

		prevType = tok.Type
	}

	return tokens, l.errors
}

// Errors returns any errors accumulated during lexing.
func (l *Lexer) Errors() []LexError {
	return l.errors
}

// Helper functions

func isDigit(r rune) bool {
	return r >= '0' && r <= '9'
}

func isHexDigit(r rune) bool {
	return isDigit(r) || (r >= 'a' && r <= 'f') || (r >= 'A' && r <= 'F')
}

func isIdentStart(r rune) bool {
	return r == '_' || unicode.IsLetter(r)
}

func isIdentPart(r rune) bool {
	return r == '_' || unicode.IsLetter(r) || unicode.IsDigit(r)
}

func hexVal(r rune) rune {
	switch {
	case r >= '0' && r <= '9':
		return r - '0'
	case r >= 'a' && r <= 'f':
		return r - 'a' + 10
	case r >= 'A' && r <= 'F':
		return r - 'A' + 10
	default:
		return 0
	}
}
