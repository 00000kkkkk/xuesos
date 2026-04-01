package xuesos

import (
	"fmt"
	"strings"
)

// FormatErrorWithContext formats an error with source code context
func FormatErrorWithContext(src string, line int, col int, msg string) string {
	lines := strings.Split(src, "\n")
	if line < 1 || line > len(lines) {
		return fmt.Sprintf("error: %s\n", msg)
	}

	var out strings.Builder
	out.WriteString(fmt.Sprintf("error: %s\n", msg))

	// Show line before (if exists)
	if line >= 2 {
		out.WriteString(fmt.Sprintf("  %4d | %s\n", line-1, lines[line-2]))
	}

	// Show error line
	out.WriteString(fmt.Sprintf("  %4d | %s\n", line, lines[line-1]))

	// Show pointer
	pointer := col - 1
	if pointer < 0 {
		pointer = 0
	}
	out.WriteString(fmt.Sprintf("       | %s^\n", strings.Repeat(" ", pointer)))

	// Show line after (if exists)
	if line < len(lines) {
		out.WriteString(fmt.Sprintf("  %4d | %s\n", line+1, lines[line]))
	}

	return out.String()
}
