package xuesos

import "fmt"

const Version = "0.1.0"

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
  fmt <file.xpp>       Format a source file
  version              Show compiler version
  help                 Show this help message`)
}
