package xuesos

import "fmt"

const Version = "0.2.0"

func Execute(args []string) error {
	if len(args) == 0 {
		printUsage()
		return nil
	}
	switch args[0] {
	case "build":
		return runBuild(args[1:])
	case "run":
		return runRun(args[1:])
	case "repl":
		return runRepl()
	case "fmt":
		return runFmt(args[1:])
	case "test":
		return runTest(args[1:])
	case "vet":
		return runVet(args[1:])
	case "doc":
		return runDoc(args[1:])
	case "env":
		return runEnv(args[1:])
	case "lsp":
		return runLsp()
	case "version":
		return runVersion()
	case "help":
		printUsage()
		return nil
	default:
		return fmt.Errorf("unknown command %q, run 'xuesos help' for usage", args[0])
	}
}

func printUsage() {
	fmt.Println(`Xuesos++ Compiler v` + Version + `

Usage:
  xuesos <command> [arguments]

Commands:
  build <file.xpp>    Compile a Xuesos++ source file
  run <file.xpp>      Compile and run a Xuesos++ source file
  repl                 Start interactive REPL
  test [dir]           Run test files (*_test.xpp)
  vet [dir]            Run static analysis on .xpp files
  fmt <file.xpp>       Format a source file
  doc [file.xpp]       Show documentation (builtins if no file)
  env [var]            Show environment info
  version              Show compiler version
  help                 Show this help message`)
}
