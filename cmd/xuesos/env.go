package xuesos

import (
	"fmt"
	"os"
	"runtime"
)

func runEnv(args []string) error {
	env := map[string]string{
		"XUESOS_VERSION":   Version,
		"XUESOS_OS":        runtime.GOOS,
		"XUESOS_ARCH":      runtime.GOARCH,
		"XUESOS_GOVERSION": runtime.Version(),
		"XUESOS_ROOT":      getXuesosRoot(),
		"XUESOS_COMPILER":  findCompiler(),
	}

	if len(args) > 0 {
		// Print specific variable
		for _, key := range args {
			if val, ok := env[key]; ok {
				fmt.Println(val)
			} else {
				fmt.Printf("%s: not set\n", key)
			}
		}
		return nil
	}

	// Print all in deterministic order
	keys := []string{
		"XUESOS_VERSION",
		"XUESOS_OS",
		"XUESOS_ARCH",
		"XUESOS_GOVERSION",
		"XUESOS_ROOT",
		"XUESOS_COMPILER",
	}
	for _, key := range keys {
		fmt.Printf("%s=%q\n", key, env[key])
	}
	return nil
}

func getXuesosRoot() string {
	if root := os.Getenv("XUESOS_ROOT"); root != "" {
		return root
	}
	dir, _ := os.Getwd()
	return dir
}
