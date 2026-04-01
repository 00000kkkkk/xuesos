package xuesos

import (
	"fmt"
	"os"
	"strings"

	"github.com/00000kkkkk/xusesosplusplus/lexer"
)

func runFmt(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("fmt requires a .xpp file argument\nUsage: xuesos fmt <file.xpp>")
	}

	filename := args[0]
	src, err := os.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("cannot read %s: %w", filename, err)
	}

	formatted := formatSource(string(src))

	if err := os.WriteFile(filename, []byte(formatted), 0644); err != nil {
		return fmt.Errorf("cannot write %s: %w", filename, err)
	}

	fmt.Printf("Formatted %s\n", filename)
	return nil
}

func formatSource(src string) string {
	l := lexer.New("", src)
	tokens, _ := l.ScanAll()

	var out strings.Builder
	indent := 0
	prevType := lexer.TOKEN_EOF
	needNewline := false

	for _, tok := range tokens {
		if tok.Type == lexer.TOKEN_EOF {
			break
		}

		// Handle semicolons (auto-inserted) as newlines
		if tok.Type == lexer.TOKEN_SEMICOLON {
			needNewline = true
			prevType = tok.Type
			continue
		}

		// Decrease indent before }
		if tok.Type == lexer.TOKEN_RBRACE {
			indent--
			if indent < 0 {
				indent = 0
			}
			if needNewline || prevType == lexer.TOKEN_LBRACE {
				out.WriteString("\n")
			} else {
				out.WriteString("\n")
			}
			needNewline = false
			writeIndent(&out, indent)
			out.WriteString("}")
			prevType = tok.Type
			continue
		}

		// "xuelse" after "}" stays on same line
		if prevType == lexer.TOKEN_RBRACE && (tok.Type == lexer.TOKEN_XUELSE) {
			out.WriteString(" ")
			needNewline = false
		} else if needNewline {
			out.WriteString("\n")
			writeIndent(&out, indent)
			needNewline = false
		} else if prevType == lexer.TOKEN_LBRACE {
			out.WriteString("\n")
			writeIndent(&out, indent)
		} else if prevType == lexer.TOKEN_RBRACE {
			out.WriteString("\n")
			writeIndent(&out, indent)
		} else if prevType != lexer.TOKEN_EOF && prevType != lexer.TOKEN_LPAREN &&
			tok.Type != lexer.TOKEN_LPAREN && tok.Type != lexer.TOKEN_RPAREN &&
			tok.Type != lexer.TOKEN_COMMA && tok.Type != lexer.TOKEN_DOT &&
			prevType != lexer.TOKEN_DOT && prevType != lexer.TOKEN_NOT {
			out.WriteString(" ")
		}

		// Increase indent after {
		if tok.Type == lexer.TOKEN_LBRACE {
			out.WriteString("{")
			indent++
			prevType = tok.Type
			continue
		}

		// Write token
		switch tok.Type {
		case lexer.TOKEN_STRING:
			out.WriteString(`"` + escapeString(tok.Literal) + `"`)
		case lexer.TOKEN_CHAR:
			out.WriteString("'" + tok.Literal + "'")
		default:
			out.WriteString(tok.Literal)
		}

		prevType = tok.Type
	}

	result := out.String()
	if !strings.HasSuffix(result, "\n") {
		result += "\n"
	}
	return result
}

func writeIndent(out *strings.Builder, level int) {
	for i := 0; i < level; i++ {
		out.WriteString("    ")
	}
}

func escapeString(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "\"", "\\\"")
	s = strings.ReplaceAll(s, "\n", "\\n")
	s = strings.ReplaceAll(s, "\t", "\\t")
	return s
}
