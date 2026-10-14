package speclint

import (
	"fmt"
	"regexp"
	"strings"
)

// Annotation is one spec link in code or tests: one scenario named on a @scenario line, e.g.
// "@scenario [#some-id] [#other-id]", or a step annotation, e.g.
// "@then [features/x.feature#some-id] I receive a reminder".
type Annotation struct {
	Keyword string // lowercase: scenario, given, when, then, and, but
	// Feature is the bracketed path on a step annotation. A @scenario annotation names no path, so
	// its Feature is the file whose scenario has the ID, filled in once the features are loaded.
	Feature string
	ID      string // the scenario ID after '#', or "" if there isn't one
	Text    string // the step text; empty for @scenario
	File    string // repo-relative path of the source file
	Line    int
	Test    bool // true when the source file is a test
}

// keywordPattern finds an annotation's keyword. What follows it is read by scenarioRefs or
// stepPattern.
var keywordPattern = regexp.MustCompile(`@(scenario|given|when|then|and|but)\b`)

// The path and ID are matched loosely so that malformed references get reported rather than
// silently skipped.
var stepPattern = regexp.MustCompile(`^\s+\[([^\]#]*?)(?:#([^\]]*))?\](.*)`)

// scenarioRef is one "[#id]" on a @scenario line.
var scenarioRef = regexp.MustCompile(`^\[#([^\]\s]*)\]`)

// commentPrefix matches what may precede an annotation on its line: indentation, then a line
// comment, a block comment, a JSX comment, or a block comment continuation.
var commentPrefix = regexp.MustCompile(`^\s*(//|/\*|\{/\*|\*)\s*$`)

// sqlCommentPrefix is commentPrefix for SQL files, which use -- line comments.
var sqlCommentPrefix = regexp.MustCompile(`^\s*--\s*$`)

// commentLine matches a line that is only a comment, so that comment lines between two @scenario
// lines keep them in the same comment block.
var commentLine = regexp.MustCompile(`^\s*(//|/\*|\{/\*|\*|--)`)

// parseAnnotations finds the spec annotations in a source file. An annotation must be the only
// thing in its comment, on a line of its own, so a string that happens to look like one is
// ignored. It reports a @scenario line that isn't a list of [#id] references, and a second
// @scenario line in the same comment block, since one line names every scenario of the code
// under it (docs/adr/0026-link-scenarios-by-id-on-one-line.md).
func parseAnnotations(file, src string, test bool) ([]Annotation, []Problem) {
	prefix := commentPrefix
	if strings.HasSuffix(file, ".sql") {
		prefix = sqlCommentPrefix
	}

	var out []Annotation
	var problems []Problem
	report := func(line int, msg string) {
		problems = append(problems, Problem{File: file, Line: line, Msg: msg})
	}
	scenarioLine := 0 // the @scenario line in the current comment block, or 0
	for i, raw := range strings.Split(src, "\n") {
		line := i + 1
		if !commentLine.MatchString(raw) {
			scenarioLine = 0
		}
		loc := keywordPattern.FindStringSubmatchIndex(raw)
		if loc == nil || !prefix.MatchString(raw[:loc[0]]) {
			continue
		}
		keyword := raw[loc[2]:loc[3]]
		rest := raw[loc[1]:]

		if keyword == "scenario" {
			// Prose that wraps onto a line starting with "@scenario" isn't an annotation.
			if !strings.HasPrefix(strings.TrimSpace(rest), "[") {
				continue
			}
			ids, ok := scenarioRefs(closeComment(rest))
			if !ok {
				report(line, "@scenario names scenarios by ID only, like @scenario [#some-id] [#other-id]")
				continue
			}
			if scenarioLine != 0 {
				report(line, fmt.Sprintf("@scenario is already on line %d of this comment; name every scenario on that one line", scenarioLine))
			} else {
				scenarioLine = line
			}
			seen := map[string]bool{}
			for _, id := range ids {
				if seen[id] {
					report(line, "@scenario names [#"+id+"] twice")
					continue
				}
				seen[id] = true
				out = append(out, Annotation{Keyword: keyword, ID: id, File: file, Line: line, Test: test})
			}
			continue
		}

		m := stepPattern.FindStringSubmatch(rest)
		if m == nil {
			continue
		}
		out = append(out, Annotation{
			Keyword: keyword,
			Feature: strings.TrimSpace(m[1]),
			ID:      strings.TrimSpace(m[2]),
			Text:    closeComment(m[3]),
			File:    file,
			Line:    line,
			Test:    test,
		})
	}
	return out, problems
}

// closeComment trims the end of a block or JSX comment that closes on the same line, with */ or
// */}.
func closeComment(s string) string {
	s = strings.TrimSpace(s)
	if t, ok := strings.CutSuffix(s, "*/}"); ok {
		return strings.TrimSpace(t)
	}
	return strings.TrimSpace(strings.TrimSuffix(s, "*/"))
}

// scenarioRefs reads the IDs from what follows @scenario: one or more [#id], separated by spaces,
// and nothing else.
func scenarioRefs(s string) ([]string, bool) {
	var ids []string
	for s = strings.TrimSpace(s); s != ""; s = strings.TrimSpace(s) {
		m := scenarioRef.FindStringSubmatch(s)
		if m == nil {
			return nil, false
		}
		ids = append(ids, m[1])
		s = s[len(m[0]):]
		if s != "" && s[0] != ' ' && s[0] != '\t' {
			return nil, false
		}
	}
	return ids, len(ids) > 0
}

// isTestFile reports whether a repo-relative path is a test: Go _test.go files, and *.test.* /
// *.spec.* files or anything under __tests__ for TypeScript and JavaScript.
func isTestFile(path string) bool {
	base := path[strings.LastIndex(path, "/")+1:]
	return strings.HasSuffix(base, "_test.go") ||
		strings.Contains(base, ".test.") ||
		strings.Contains(base, ".spec.") ||
		strings.Contains("/"+path, "/__tests__/")
}
