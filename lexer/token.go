package lexer

import "fmt"

// Position represents a location in the source code.
type Position struct {
	File   string
	Line   int
	Column int
	Offset int
}

func (p Position) String() string {
	if p.File != "" {
		return fmt.Sprintf("%s:%d:%d", p.File, p.Line, p.Column)
	}
	return fmt.Sprintf("%d:%d", p.Line, p.Column)
}

// TokenType represents the type of a lexer token.
type TokenType int

const (
	// Special tokens
	TOKEN_ILLEGAL TokenType = iota
	TOKEN_EOF
	TOKEN_SEMICOLON // auto-inserted

	// Literals
	TOKEN_IDENT
	TOKEN_INT
	TOKEN_FLOAT
	TOKEN_STRING
	TOKEN_CHAR

	// Keywords
	TOKEN_XUEN     // xuen (fn)
	TOKEN_XUET     // xuet (let/immutable)
	TOKEN_XUIAR    // xuiar (var/mutable)
	TOKEN_XUIF     // xuif (if)
	TOKEN_XUELSE   // xuelse (else)
	TOKEN_XUIOR    // xuior (for)
	TOKEN_XUILE    // xuile (while)
	TOKEN_XUIN     // xuin (in)
	TOKEN_XUETURN  // xueturn (return)
	TOKEN_XUIRUCT  // xuiruct (struct)
	TOKEN_XUIMPL   // xuimpl (impl)
	TOKEN_XUENUM   // xuenum (enum)
	TOKEN_XUIATCH  // xuiatch (match)
	TOKEN_XUITRU   // xuitru (true)
	TOKEN_XUINIA   // xuinia (false)
	TOKEN_XUINULL   // xuinull (null)
	TOKEN_XUIMPORT // xuimport (import)
	TOKEN_XUIUB    // xuiub (pub)
	TOKEN_XUIEAK   // xuieak (break)
	TOKEN_XUITINUE // xuitinue (continue)
	TOKEN_XUTRY    // xutry (try)
	TOKEN_XUCATCH  // xucatch (catch)
	TOKEN_XUTHROW  // xuthrow (throw)
	TOKEN_XUSPAWN  // xuspawn (spawn goroutine)

	// Type keywords
	TOKEN_INT_TYPE
	TOKEN_INT8_TYPE
	TOKEN_INT16_TYPE
	TOKEN_INT32_TYPE
	TOKEN_INT64_TYPE
	TOKEN_UINT_TYPE
	TOKEN_FLOAT_TYPE
	TOKEN_FLOAT32_TYPE
	TOKEN_BOOL_TYPE
	TOKEN_STR_TYPE
	TOKEN_CHAR_TYPE
	TOKEN_BYTE_TYPE

	// Operators
	TOKEN_PLUS      // +
	TOKEN_MINUS     // -
	TOKEN_STAR      // *
	TOKEN_SLASH     // /
	TOKEN_PERCENT   // %
	TOKEN_ASSIGN    // =
	TOKEN_EQ        // ==
	TOKEN_NEQ       // !=
	TOKEN_LT        // <
	TOKEN_GT        // >
	TOKEN_LTE       // <=
	TOKEN_GTE       // >=
	TOKEN_AND       // &&
	TOKEN_OR        // ||
	TOKEN_NOT       // !
	TOKEN_AMPERSAND // & (address-of)
	TOKEN_FAT_ARROW // =>
	TOKEN_DOTDOT    // ..

	// Delimiters
	TOKEN_LPAREN   // (
	TOKEN_RPAREN   // )
	TOKEN_LBRACE   // {
	TOKEN_RBRACE   // }
	TOKEN_LBRACKET // [
	TOKEN_RBRACKET // ]
	TOKEN_COLON    // :
	TOKEN_COMMA    // ,
	TOKEN_DOT      // .
	TOKEN_QUESTION // ?

	// String interpolation
	TOKEN_INTERP_START // start of interpolation segment
	TOKEN_INTERP_END   // end of interpolation segment
)

var tokenNames = map[TokenType]string{
	TOKEN_ILLEGAL:   "ILLEGAL",
	TOKEN_EOF:       "EOF",
	TOKEN_SEMICOLON: "SEMICOLON",

	TOKEN_IDENT:  "IDENT",
	TOKEN_INT:    "INT",
	TOKEN_FLOAT:  "FLOAT",
	TOKEN_STRING: "STRING",
	TOKEN_CHAR:   "CHAR",

	TOKEN_XUEN:     "xuen",
	TOKEN_XUET:     "xuet",
	TOKEN_XUIAR:    "xuiar",
	TOKEN_XUIF:     "xuif",
	TOKEN_XUELSE:   "xuelse",
	TOKEN_XUIOR:    "xuior",
	TOKEN_XUILE:    "xuile",
	TOKEN_XUIN:     "xuin",
	TOKEN_XUETURN:  "xueturn",
	TOKEN_XUIRUCT:  "xuiruct",
	TOKEN_XUIMPL:   "xuimpl",
	TOKEN_XUENUM:   "xuenum",
	TOKEN_XUIATCH:  "xuiatch",
	TOKEN_XUITRU:   "xuitru",
	TOKEN_XUINIA:   "xuinia",
	TOKEN_XUINULL:   "xuinull",
	TOKEN_XUIMPORT: "xuimport",
	TOKEN_XUIUB:    "xuiub",
	TOKEN_XUIEAK:   "xuieak",
	TOKEN_XUITINUE: "xuitinue",
	TOKEN_XUTRY:    "xutry",
	TOKEN_XUCATCH:  "xucatch",
	TOKEN_XUTHROW:  "xuthrow",
	TOKEN_XUSPAWN:  "xuspawn",

	TOKEN_INT_TYPE:     "int",
	TOKEN_INT8_TYPE:    "int8",
	TOKEN_INT16_TYPE:   "int16",
	TOKEN_INT32_TYPE:   "int32",
	TOKEN_INT64_TYPE:   "int64",
	TOKEN_UINT_TYPE:    "uint",
	TOKEN_FLOAT_TYPE:   "float",
	TOKEN_FLOAT32_TYPE: "float32",
	TOKEN_BOOL_TYPE:    "bool",
	TOKEN_STR_TYPE:     "str",
	TOKEN_CHAR_TYPE:    "char",
	TOKEN_BYTE_TYPE:    "byte",

	TOKEN_PLUS:      "+",
	TOKEN_MINUS:     "-",
	TOKEN_STAR:      "*",
	TOKEN_SLASH:     "/",
	TOKEN_PERCENT:   "%",
	TOKEN_ASSIGN:    "=",
	TOKEN_EQ:        "==",
	TOKEN_NEQ:       "!=",
	TOKEN_LT:        "<",
	TOKEN_GT:        ">",
	TOKEN_LTE:       "<=",
	TOKEN_GTE:       ">=",
	TOKEN_AND:       "&&",
	TOKEN_OR:        "||",
	TOKEN_NOT:       "!",
	TOKEN_AMPERSAND: "&",
	TOKEN_FAT_ARROW: "=>",
	TOKEN_DOTDOT:    "..",

	TOKEN_LPAREN:   "(",
	TOKEN_RPAREN:   ")",
	TOKEN_LBRACE:   "{",
	TOKEN_RBRACE:   "}",
	TOKEN_LBRACKET: "[",
	TOKEN_RBRACKET: "]",
	TOKEN_COLON:    ":",
	TOKEN_COMMA:    ",",
	TOKEN_DOT:      ".",
	TOKEN_QUESTION: "?",

	TOKEN_INTERP_START: "INTERP_START",
	TOKEN_INTERP_END:   "INTERP_END",
}

func (t TokenType) String() string {
	if name, ok := tokenNames[t]; ok {
		return name
	}
	return fmt.Sprintf("TokenType(%d)", int(t))
}

// Token represents a single lexed token.
type Token struct {
	Type    TokenType
	Literal string
	Pos     Position
}

func (t Token) String() string {
	return fmt.Sprintf("%s(%q) at %s", t.Type, t.Literal, t.Pos)
}

var keywords = map[string]TokenType{
	"xuen":     TOKEN_XUEN,
	"xuet":     TOKEN_XUET,
	"xuiar":    TOKEN_XUIAR,
	"xuif":     TOKEN_XUIF,
	"xuelse":   TOKEN_XUELSE,
	"xuior":    TOKEN_XUIOR,
	"xuile":    TOKEN_XUILE,
	"xuin":     TOKEN_XUIN,
	"xueturn":  TOKEN_XUETURN,
	"xuiruct":  TOKEN_XUIRUCT,
	"xuimpl":   TOKEN_XUIMPL,
	"xuenum":   TOKEN_XUENUM,
	"xuiatch":  TOKEN_XUIATCH,
	"xuitru":   TOKEN_XUITRU,
	"xuinia":   TOKEN_XUINIA,
	"xuinull":   TOKEN_XUINULL,
	"xuimport": TOKEN_XUIMPORT,
	"xuiub":    TOKEN_XUIUB,
	"xuieak":   TOKEN_XUIEAK,
	"xuitinue": TOKEN_XUITINUE,
	"xutry":    TOKEN_XUTRY,
	"xucatch":  TOKEN_XUCATCH,
	"xuthrow":  TOKEN_XUTHROW,
	"xuspawn":  TOKEN_XUSPAWN,
	"int":      TOKEN_INT_TYPE,
	"int8":     TOKEN_INT8_TYPE,
	"int16":    TOKEN_INT16_TYPE,
	"int32":    TOKEN_INT32_TYPE,
	"int64":    TOKEN_INT64_TYPE,
	"uint":     TOKEN_UINT_TYPE,
	"float":    TOKEN_FLOAT_TYPE,
	"float32":  TOKEN_FLOAT32_TYPE,
	"bool":     TOKEN_BOOL_TYPE,
	"str":      TOKEN_STR_TYPE,
	"char":     TOKEN_CHAR_TYPE,
	"byte":     TOKEN_BYTE_TYPE,
}

// LookupIdent returns the keyword TokenType for ident, or TOKEN_IDENT if not a keyword.
func LookupIdent(ident string) TokenType {
	if tok, ok := keywords[ident]; ok {
		return tok
	}
	return TOKEN_IDENT
}

// semicolonPreceding lists token types after which a semicolon is auto-inserted on newline.
var semicolonPreceding = map[TokenType]bool{
	TOKEN_IDENT:        true,
	TOKEN_INT:          true,
	TOKEN_FLOAT:        true,
	TOKEN_STRING:       true,
	TOKEN_CHAR:         true,
	TOKEN_XUITRU:       true,
	TOKEN_XUINIA:       true,
	TOKEN_XUINULL:       true,
	TOKEN_XUETURN:      true,
	TOKEN_XUIEAK:       true,
	TOKEN_XUITINUE:     true,
	TOKEN_RPAREN:       true,
	TOKEN_RBRACKET:     true,
	TOKEN_RBRACE:       true,
	TOKEN_INT_TYPE:     true,
	TOKEN_INT8_TYPE:    true,
	TOKEN_INT16_TYPE:   true,
	TOKEN_INT32_TYPE:   true,
	TOKEN_INT64_TYPE:   true,
	TOKEN_UINT_TYPE:    true,
	TOKEN_FLOAT_TYPE:   true,
	TOKEN_FLOAT32_TYPE: true,
	TOKEN_BOOL_TYPE:    true,
	TOKEN_STR_TYPE:     true,
	TOKEN_CHAR_TYPE:    true,
	TOKEN_BYTE_TYPE:    true,
}
