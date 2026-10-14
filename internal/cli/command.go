package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"strconv"
	"strings"
)

// command is one thing the command line does, like `job move`.
type command struct {
	// path is the words that name it, like {"job", "move"}.
	path []string
	// takesID names what the ID it takes is of, like "job", or is "" for a command that takes no
	// arguments.
	takesID string
	// summary is the one line the help lists it with.
	summary string
	// about is what its own help says about it, after the usage line.
	about   string
	options []option
	run     func(c *call) error
}

func (cmd *command) name() string { return strings.Join(cmd.path, " ") }

// usage is the command's name with the ID it takes, like "job show <job-id>".
func (cmd *command) usage() string {
	if cmd.takesID == "" {
		return cmd.name()
	}
	return cmd.name() + " <" + cmd.takesID + "-id>"
}

// option is one of a command's options. Every option takes a value, except a flag (value "").
type option struct {
	name  string
	value string
	usage string
	// choices are the values it takes, when it takes one of a fixed set.
	choices []string
}

// jsonOption is on every command.
var jsonOption = option{name: "json", usage: "print the output, and any error, as JSON"}

// findCommand finds the command words name, and returns it with the arguments after its name.
func findCommand(words []string) (*command, []string, error) {
	for _, cmd := range commands {
		if len(words) >= len(cmd.path) && strings.Join(words[:len(cmd.path)], " ") == cmd.name() {
			return cmd, words[len(cmd.path):], nil
		}
	}
	var under []string
	for _, cmd := range commands {
		if cmd.path[0] == words[0] {
			under = append(under, cmd.path[1])
		}
	}
	if len(under) > 0 && (len(words) == 1 || strings.HasPrefix(words[1], "-")) {
		return nil, nil, usageErr("%q needs one of its commands: %s; run `%s --help` to see what each does", words[0], strings.Join(under, ", "), Name)
	}
	return nil, nil, usageErr("there's no command %q; run `%s --help` to see the commands", strings.Join(words[:min(len(words), 2)], " "), Name)
}

// call is one run of a command: its arguments and options, and what it runs over.
type call struct {
	cmd *command
	// id is the ID the command was given, for a command that takes one.
	id     int64
	values map[string]*string
	flags  map[string]*bool
	set    map[string]bool
	asJSON bool
	ctx    context.Context
	env    Env
	jobs   Jobs
}

// parseCall reads a command's arguments and options, which can come in any order, and checks
// them, so a usage error is found before the data is opened.
func parseCall(cmd *command, args []string) (*call, error) {
	var positional []string
	c := &call{cmd: cmd, values: map[string]*string{}, flags: map[string]*bool{}, set: map[string]bool{}}
	fs := flag.NewFlagSet(cmd.name(), flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	for _, o := range append(cmd.options, jsonOption) {
		if o.value == "" {
			c.flags[o.name] = fs.Bool(o.name, false, o.usage)
		} else {
			c.values[o.name] = fs.String(o.name, "", o.usage)
		}
	}
	for {
		if err := fs.Parse(args); err != nil {
			if errors.Is(err, flag.ErrHelp) {
				return nil, err
			}
			return nil, usageErr("%s; run `%s %s --help` to see its options", strings.TrimPrefix(err.Error(), "flag "), Name, cmd.name())
		}
		args = fs.Args()
		if len(args) == 0 {
			break
		}
		positional = append(positional, args[0])
		args = args[1:]
	}
	fs.Visit(func(f *flag.Flag) { c.set[f.Name] = true })
	c.asJSON = *c.flags["json"]

	switch {
	case cmd.takesID == "" && len(positional) > 0:
		return nil, usageErr("`%s %s` takes no arguments, but was given %q", Name, cmd.name(), strings.Join(positional, " "))
	case cmd.takesID == "":
		return c, nil
	case len(positional) != 1:
		return nil, usageErr("`%s %s` takes one %s ID, like `%s %s 12`", Name, cmd.name(), cmd.takesID, Name, cmd.name())
	}
	id, err := strconv.ParseInt(positional[0], 10, 64)
	if err != nil || id < 1 {
		return nil, usageErr("%q isn't a %s ID; IDs are whole numbers, like 12", positional[0], cmd.takesID)
	}
	c.id = id
	return c, nil
}

// value returns an option's value and whether it was given.
func (c *call) value(name string) (string, bool) {
	return *c.values[name], c.set[name]
}

func (c *call) flag(name string) bool { return *c.flags[name] }

// longText reads the text for name from --<name> or, for long text, --<name>-file: a file's path,
// or - for what's piped in. It's taken exactly as written, quotes, line breaks, and blank lines
// included. ok is false when neither was given.
func (c *call) longText(name string) (text string, ok bool, err error) {
	inline, inlineSet := c.value(name)
	path, fileSet := c.value(name + "-file")
	switch {
	case inlineSet && fileSet:
		return "", false, usageErr("give --%s or --%s-file, not both", name, name)
	case inlineSet:
		return inline, true, nil
	case !fileSet:
		return "", false, nil
	}
	var data []byte
	if path == "-" {
		data, err = io.ReadAll(c.env.Stdin)
	} else {
		data, err = c.env.ReadFile(path)
	}
	if errors.Is(err, fs.ErrNotExist) {
		return "", false, validationErr("there's no file at %s", path)
	}
	if err != nil {
		return "", false, fmt.Errorf("read --%s-file: %w", name, err)
	}
	return string(data), true, nil
}

// output writes v as JSON with --json, and calls text otherwise.
func (c *call) output(v any, text func(w io.Writer)) error {
	if c.asJSON {
		return writeJSON(c.env.Stdout, v)
	}
	text(c.env.Stdout)
	return nil
}

// pick finds given among choices, by value or by label, ignoring case and treating spaces and
// hyphens as underscores, so "No response", "no-response", and "no_response" all match. what
// names the choice, like "status", for the error.
func pick[T ~string](what, given string, choices []T, label func(T) string) (T, error) {
	normal := func(s string) string {
		return strings.NewReplacer(" ", "_", "-", "_").Replace(strings.ToLower(strings.TrimSpace(s)))
	}
	for _, choice := range choices {
		if normal(given) == string(choice) || normal(given) == normal(label(choice)) {
			return choice, nil
		}
	}
	return "", &cliError{code: ExitValidation, msg: fmt.Sprintf("there's no %s %q", what, given), choices: values(choices)}
}

func values[T ~string](choices []T) []string {
	out := make([]string, len(choices))
	for i, c := range choices {
		out[i] = string(c)
	}
	return out
}

// described lists choices as "value (Label)", for the help.
func described[T ~string](choices []T, label func(T) string) []string {
	out := make([]string, len(choices))
	for i, c := range choices {
		out[i] = fmt.Sprintf("%s (%s)", c, label(c))
	}
	return out
}
