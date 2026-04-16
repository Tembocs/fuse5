// Command checkartifacts verifies the artifact policy: .gitignore exists
// and excludes the build-output and editor patterns the project does not
// track (Rule: "a tracked source file must never depend on .gitignore to
// disappear from review"; inverse: generated artifacts must be ignored).
package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// requiredPatterns are patterns the root .gitignore must contain. They are
// checked as substrings so comment-prefixed or combined entries still count.
var requiredPatterns = []string{
	"/bin/",
	"stage2_out",
	".fuse-cache/",
	"*.test",
	"*.exe",
	"*.o",
}

func main() {
	f, err := os.Open(".gitignore")
	if err != nil {
		fmt.Fprintf(os.Stderr, "checkartifacts: cannot read .gitignore: %v\n", err)
		os.Exit(1)
	}
	defer f.Close()

	var content strings.Builder
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		content.WriteString(sc.Text())
		content.WriteByte('\n')
	}
	if err := sc.Err(); err != nil {
		fmt.Fprintf(os.Stderr, "checkartifacts: read error: %v\n", err)
		os.Exit(1)
	}
	body := content.String()

	var missing []string
	for _, p := range requiredPatterns {
		if !strings.Contains(body, p) {
			missing = append(missing, p)
		}
	}
	if len(missing) > 0 {
		fmt.Fprintln(os.Stderr, "checkartifacts: .gitignore missing required patterns:")
		for _, m := range missing {
			fmt.Fprintln(os.Stderr, "  -", m)
		}
		os.Exit(1)
	}
	fmt.Println("checkartifacts: ok")
}
