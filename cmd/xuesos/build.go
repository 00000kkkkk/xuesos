package xuesos

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
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

	// Find the filename (first non-flag argument)
	filename := ""
	for _, a := range args {
		if !strings.HasPrefix(a, "-") {
			filename = a
			break
		}
	}
	if filename == "" {
		return fmt.Errorf("build requires a .xpp file argument\nUsage: xuesos build <file.xpp> [-o name] [--os target] [--arch target]")
	}
	if !strings.HasSuffix(filename, ".xpp") {
		return fmt.Errorf("expected .xpp file, got %q", filename)
	}

	// Parse flags
	targetOS := runtime.GOOS
	targetArch := runtime.GOARCH
	outputName := strings.TrimSuffix(filepath.Base(filename), ".xpp")

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "-o":
			if i+1 < len(args) {
				outputName = args[i+1]
				i++
			}
		case "--os":
			if i+1 < len(args) {
				targetOS = args[i+1]
				i++
			}
		case "--arch":
			if i+1 < len(args) {
				targetArch = args[i+1]
				i++
			}
		}
	}

	fmt.Printf("Building %s for %s/%s\n", filename, targetOS, targetArch)

	src, err := os.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("cannot read %s: %w", filename, err)
	}

	// Lex
	l := lexer.New(filename, string(src))
	tokens, lexErrs := l.ScanAll()
	if len(lexErrs) > 0 {
		for _, e := range lexErrs {
			fmt.Fprint(os.Stderr, FormatErrorWithContext(string(src), e.Pos.Line, e.Pos.Column, e.Message))
		}
		return fmt.Errorf("lexing failed with %d error(s)", len(lexErrs))
	}

	// Parse
	p := parser.New(tokens)
	program, parseErrs := p.Parse()
	if len(parseErrs) > 0 {
		for _, e := range parseErrs {
			fmt.Fprint(os.Stderr, FormatErrorWithContext(string(src), e.Pos.Line, e.Pos.Column, e.Message))
		}
		return fmt.Errorf("parsing failed with %d error(s)", len(parseErrs))
	}

	// Type check
	tc := typechecker.New()
	typeErrs := tc.Check(program)
	if len(typeErrs) > 0 {
		for _, e := range typeErrs {
			fmt.Fprint(os.Stderr, FormatErrorWithContext(string(src), e.Pos.Line, e.Pos.Column, e.Message))
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

	// For cross-compilation, try platform-specific compilers
	compiler := findCompiler()
	if targetOS != runtime.GOOS || targetArch != runtime.GOARCH {
		crossCompiler := fmt.Sprintf("%s-%s-gcc", targetOS, targetArch)
		if path, err := exec.LookPath(crossCompiler); err == nil {
			compiler = path
		}
	}

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
