package cli

import (
	_ "embed"
	"fmt"
	"io"
	"strings"
	"text/tabwriter"
	"text/template"
)

// writeHelp lists every command with a short description, and points to the overview.
// @scenario [features/agent-cli.feature#cli-help] The help lists every command
// @scenario [features/agent-cli.feature#cli-help-points-to-overview] The help points my agent to the overview
func writeHelp(w io.Writer) error {
	fmt.Fprintf(w, "%s works with your Callback job search, the same data the Callback app shows.\n\n", Name)
	fmt.Fprintf(w, "Usage:\n  %s <command> [arguments] [options]\n\n", Name)
	fmt.Fprintln(w, "Commands:")
	tw := tabwriter.NewWriter(w, 0, 0, 3, ' ', 0)
	for _, cmd := range commands {
		fmt.Fprintf(tw, "  %s\t%s\n", cmd.name(), cmd.summary)
	}
	tw.Flush()
	fmt.Fprintln(w, "\nOptions for every command:")
	tw = tabwriter.NewWriter(w, 0, 0, 3, ' ', 0)
	fmt.Fprintf(tw, "  --%s\t%s\n", jsonOption.name, jsonOption.usage)
	fmt.Fprintf(tw, "  --help, -h\t%s\n", "show a command's options and the values they take")
	tw.Flush()
	fmt.Fprintf(w, "\nRead `%s --overview` before you start. It explains what Callback keeps track of, how the\n", Name)
	fmt.Fprintln(w, "commands fit together, how output, IDs, dates, long text, and errors work, and what the exit")
	fmt.Fprintln(w, "codes mean.")
	fmt.Fprintf(w, "Run `%s <command> --help` to see a command's options.\n", Name)
	return nil
}

// writeCommandHelp shows a command's usage, what it does, and its options with the values they
// take.
// @scenario [features/agent-cli.feature#cli-command-help] The help for a command lists its options and the values they take
// @scenario [features/agent-cli.feature#cli-help-stored-files] The help says how to use a document's stored copy
// @scenario [features/agent-cli.feature#cli-help-lists-choices] The help lists the values for every option that takes a fixed set
func writeCommandHelp(w io.Writer, cmd *command) error {
	fmt.Fprintf(w, "Usage:\n  %s %s [options]\n\n%s\n\nOptions:\n", Name, cmd.usage(), wrap(cmd.about, 96))
	tw := tabwriter.NewWriter(w, 0, 0, 3, ' ', 0)
	for _, o := range append(cmd.options, jsonOption) {
		name := "--" + o.name
		if o.value != "" {
			name += " " + o.value
		}
		fmt.Fprintf(tw, "  %s\t%s\n", name, o.usage)
		if len(o.choices) > 0 {
			fmt.Fprintf(tw, "  \tone of: %s\n", strings.Join(o.choices, ", "))
		}
	}
	return tw.Flush()
}

// wrap breaks text into lines of at most width, keeping its paragraphs.
func wrap(text string, width int) string {
	var paragraphs []string
	for p := range strings.SplitSeq(text, "\n\n") {
		var lines []string
		line := ""
		for word := range strings.FieldsSeq(p) {
			if line != "" && len(line)+1+len(word) > width {
				lines = append(lines, line)
				line = ""
			}
			if line != "" {
				line += " "
			}
			line += word
		}
		paragraphs = append(paragraphs, strings.Join(append(lines, line), "\n"))
	}
	return strings.Join(paragraphs, "\n\n")
}

//go:embed overview.md
var overviewSource string

var overviewTemplate = template.Must(template.New("overview").Parse(overviewSource))

// writeOverview prints the prose overview of Callback and the command line for an agent, like an
// AGENTS.md about the command line.
// @scenario [features/agent-cli.feature#cli-overview] My agent can read an overview of Callback and its commands
// @scenario [features/agent-cli.feature#cli-overview-limits] The overview says what my agent can't do
// @scenario [features/agent-cli.feature#cli-overview-matches-commands] Every command the overview names is one Callback has
// @scenario [features/agent-cli.feature#cli-exit-codes-listed] The overview lists every exit code and what it means
// @scenario [features/agent-cli.feature#cli-overview-read-documents] The overview tells my agent it can read my documents
// @scenario [features/agent-cli.feature#cli-overview-working-safely] The overview tells my agent how to work safely
func writeOverview(w io.Writer) error {
	return overviewTemplate.Execute(w, struct {
		Name      string
		ExitCodes []exitCode
	}{Name, exitCodes})
}
