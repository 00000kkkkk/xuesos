package xuesos

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/00000kkkkk/xusesosplusplus/codegen"
	"github.com/00000kkkkk/xusesosplusplus/lexer"
	"github.com/00000kkkkk/xusesosplusplus/parser"
	"github.com/00000kkkkk/xusesosplusplus/typechecker"
)

func runBuild(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("build requires a .xpp file argument\nUsage: xuesos build <file.xpp>")
	}

	filename := args[0]
	if !strings.HasSuffix(filename, ".xpp") {
		return fmt.Errorf("expected .xpp file, got %q", filename)
	}

	// Parse -o flag for output name
	outputName := strings.TrimSuffix(filepath.Base(filename), ".xpp")
	for i := 0; i < len(args)-1; i++ {
		if args[i] == "-o" {
			outputName = args[i+1]
			break
		}
	}

	src, err := os.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("cannot read %s: %w", filename, err)
	}

	// Lex
	l := lexer.New(filename, string(src))
	tokens, lexErrs := l.ScanAll()
	if len(lexErrs) > 0 {
		for _, e := range lexErrs {
			fmt.Fprintf(os.Stderr, "error: %s\n", e)
		}
		return fmt.Errorf("lexing failed with %d error(s)", len(lexErrs))
	}

	// Parse
	p := parser.New(tokens)
	program, parseErrs := p.Parse()
	if len(parseErrs) > 0 {
		for _, e := range parseErrs {
			fmt.Fprintf(os.Stderr, "error: %s\n", e)
		}
		return fmt.Errorf("parsing failed with %d error(s)", len(parseErrs))
	}

	// Type check
	tc := typechecker.New()
	typeErrs := tc.Check(program)
	if len(typeErrs) > 0 {
		for _, e := range typeErrs {
			fmt.Fprintf(os.Stderr, "type error: %s\n", e)
		}
		return fmt.Errorf("type checking failed with %d error(s)", len(typeErrs))
	}

	// Generate C code
	gen := codegen.New()
	cCode := gen.Generate(program)

	// Write C file
	cFile := outputName + ".c"
	if err := os.WriteFile(cFile, []byte(cCode), 0644); err != nil {
		return fmt.Errorf("cannot write %s: %w", cFile, err)
	}

	// Try to compile with gcc or cc
	compiler := findCompiler()
	if compiler == "" {
		fmt.Printf("Generated C code: %s\n", cFile)
		fmt.Println("No C compiler found (gcc/cc). Compile manually:")
		fmt.Printf("  gcc -o %s %s -lm -lpthread\n", outputName, cFile)
		return nil
	}

	cmd := exec.Command(compiler, "-o", outputName, cFile, "-lm", "-lpthread")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("compilation failed: %w", err)
	}

	// Clean up .c file
	os.Remove(cFile)

	fmt.Printf("Built: %s\n", outputName)
	return nil
}

func findCompiler() string {
	for _, name := range []string{"gcc", "cc", "clang"} {
		if path, err := exec.LookPath(name); err == nil {
			return path
		}
	}
	return ""
}
