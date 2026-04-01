package interpreter

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/00000kkkkk/xusesosplusplus/lexer"
	"github.com/00000kkkkk/xusesosplusplus/parser"
)

// ImportResolver resolves and executes imported .xpp files.
type ImportResolver struct {
	basePath string
	loaded   map[string]bool
}

// NewImportResolver creates a resolver rooted at the given directory.
func NewImportResolver(basePath string) *ImportResolver {
	return &ImportResolver{
		basePath: basePath,
		loaded:   make(map[string]bool),
	}
}

// Resolve loads and executes an imported file in the given interpreter.
func (r *ImportResolver) Resolve(path string, interp *Interpreter) error {
	// Skip stdlib modules (handled as built-ins)
	switch path {
	case "math", "os", "io", "fmt":
		return nil
	}

	// Resolve to .xpp file
	filePath := filepath.Join(r.basePath, path)
	if filepath.Ext(filePath) == "" {
		filePath += ".xpp"
	}

	// Prevent circular imports
	absPath, _ := filepath.Abs(filePath)
	if r.loaded[absPath] {
		return nil
	}
	r.loaded[absPath] = true

	src, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("cannot import %q: %w", path, err)
	}

	// Lex and parse
	l := lexer.New(filePath, string(src))
	tokens, lexErrs := l.ScanAll()
	if len(lexErrs) > 0 {
		return fmt.Errorf("import %q: lex error: %s", path, lexErrs[0])
	}

	p := parser.New(tokens)
	program, parseErrs := p.Parse()
	if len(parseErrs) > 0 {
		return fmt.Errorf("import %q: parse error: %s", path, parseErrs[0])
	}

	// Execute in the interpreter's global scope
	for _, stmt := range program.Statements {
		// Skip main() in imported files
		if fn, ok := stmt.(*parser.XuenStatement); ok && fn.Name == "main" {
			continue
		}
		if _, err := interp.execStatement(stmt, interp.globals); err != nil {
			return fmt.Errorf("import %q: %w", path, err)
		}
	}

	return nil
}
