// Command spectrace shows everything linked to one scenario, layer by layer: its entry points,
// the code and tests for each step, and the ADRs behind it. Run it from the repository root:
//
//	go run ./cmd/spectrace move-card-forward
//	go run ./cmd/spectrace features/job-board.feature#move-card-forward
//
// It exits 1 if the scenario or any step has no link in code or in tests, whatever the
// scenario's status. It reads annotations the way speclint does; see docs/spec-annotations.md.
package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"text/tabwriter"

	"callback/internal/speclint"
)

func main() {
	flag.Usage = func() {
		fmt.Fprintln(os.Stderr, "usage: spectrace [-root dir] <scenario-id | features/x.feature#scenario-id>")
		flag.PrintDefaults()
	}
	root := flag.String("root", ".", "repository root")
	flag.Parse()
	if flag.NArg() != 1 {
		flag.Usage()
		os.Exit(2)
	}

	t, err := speclint.TraceScenario(*root, flag.Arg(0))
	if err != nil {
		fmt.Fprintln(os.Stderr, "spectrace:", err)
		os.Exit(2)
	}
	missing := report(os.Stdout, t)
	if missing > 0 {
		os.Exit(1)
	}
}

// report writes the trace and returns how many links are missing.
func report(out io.Writer, t *speclint.Trace) int {
	s := t.Scenario
	fmt.Fprintf(out, "Scenario: %s\n", s.Title)
	fmt.Fprintf(out, "  %s:%d  @id:%s  @status:%s\n", t.Feature, s.Line, s.ID, s.Status)

	w := tabwriter.NewWriter(out, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "\nEntry points (@scenario)")
	links(w, t.Entry)
	for _, st := range t.Steps {
		fmt.Fprintf(w, "\n%s %s  (%s:%d)\n", st.Step.Keyword, st.Step.Text, t.Feature, st.Step.Line)
		links(w, st.Links)
	}
	fmt.Fprintln(w, "\nDecisions (ADR Related)")
	if len(t.ADRs) == 0 {
		fmt.Fprintln(w, "  (none)")
	}
	for _, r := range t.ADRs {
		scope := "this scenario"
		if r.Whole {
			scope = "the whole feature"
		}
		fmt.Fprintf(w, "  %s:%d\t%s\n", r.File, r.Line, scope)
	}
	w.Flush()

	missing := t.Missing()
	fmt.Fprintln(out)
	if len(missing) == 0 {
		fmt.Fprintf(out, "Fully linked: @scenario and all %d steps in code and tests.\n", len(t.Steps))
		return 0
	}
	for _, m := range missing {
		fmt.Fprintln(out, "MISSING:", m)
	}
	return len(missing)
}

func links(w io.Writer, as []speclint.Annotation) {
	if len(as) == 0 {
		fmt.Fprintln(w, "  (none)")
	}
	for _, a := range as {
		scoped := ""
		if a.ID != "" && a.Keyword != "scenario" {
			scoped = "  (scoped #" + a.ID + ")"
		}
		fmt.Fprintf(w, "  %s\t%s:%d\t@%s%s\n", speclint.Layer(a), a.File, a.Line, a.Keyword, scoped)
	}
}
