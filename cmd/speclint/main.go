// Command speclint checks that code, tests, and ADRs link correctly to the specs in
// features/*.feature. Run it from the repository root:
//
//	go run ./cmd/speclint
//
// It prints one "file:line: problem" per finding and exits 1 if there are any. See
// docs/spec-annotations.md for the rules it enforces.
package main

import (
	"flag"
	"fmt"
	"os"

	"callback/internal/speclint"
)

func main() {
	flag.Usage = func() {
		fmt.Fprintln(os.Stderr, "usage: speclint [repo-root]")
		flag.PrintDefaults()
	}
	flag.Parse()

	root := "."
	if flag.NArg() > 0 {
		root = flag.Arg(0)
	}

	problems, err := speclint.Lint(root)
	if err != nil {
		fmt.Fprintln(os.Stderr, "speclint:", err)
		os.Exit(2)
	}
	for _, p := range problems {
		fmt.Println(p)
	}
	if len(problems) > 0 {
		fmt.Fprintf(os.Stderr, "speclint: %d problem(s)\n", len(problems))
		os.Exit(1)
	}
}
