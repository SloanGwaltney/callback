package speclint

import (
	"regexp"
	"strings"
)

// Annotation is one spec comment in code or tests, e.g.
// "@then [features/x.feature#some-id] I receive a reminder".
type Annotation struct {
	Keyword string // lowercase: scenario, given, when, then, and, but
	Feature string // the bracketed path, e.g. features/x.feature
	ID      string // the scenario ID after '#', or "" if there isn't one
	Text    string
	File    string // repo-relative path of the source file
	Line    int
	Test    bool // true when the source file is a test
}

// The path and ID are matched loosely so that malformed references get reported rather than
// silently skipped.
var annotationPattern = regexp.MustCompile(`@(scenario|given|when|then|and|but)\s+\[([^\]#]*?)(?:#([^\]]*))?\](.*)`)

// commentPrefix matches what may precede an annotation on its line: indentation, then a line
// comment, a block comment, a JSX comment, or a block comment continuation.
var commentPrefix = regexp.MustCompile(`^\s*(//|/\*|\{/\*|\*)\s*$`)

// sqlCommentPrefix is commentPrefix for SQL files, which use -- line comments.
var sqlCommentPrefix = regexp.MustCompile(`^\s*--\s*$`)

// parseAnnotations finds the spec annotations in a source file. An annotation must be the only
// thing in its comment, on a line of its own, so a string that happens to look like one is
// ignored.
func parseAnnotations(file, src string, test bool) []Annotation {
	prefix := commentPrefix
	if strings.HasSuffix(file, ".sql") {
		prefix = sqlCommentPrefix
	}

	var out []Annotation
	for i, raw := range strings.Split(src, "\n") {
		loc := annotationPattern.FindStringSubmatchIndex(raw)
		if loc == nil {
			continue
		}
		if !prefix.MatchString(raw[:loc[0]]) {
			continue
		}
		m := annotationPattern.FindStringSubmatch(raw)
		text := strings.TrimSpace(m[4])
		// Block and JSX comments close on the same line with */ or */}.
		if t, ok := strings.CutSuffix(text, "*/}"); ok {
			text = t
		} else {
			text = strings.TrimSuffix(text, "*/")
		}
		text = strings.TrimSpace(text)
		out = append(out, Annotation{
			Keyword: m[1],
			Feature: strings.TrimSpace(m[2]),
			ID:      strings.TrimSpace(m[3]),
			Text:    text,
			File:    file,
			Line:    i + 1,
			Test:    test,
		})
	}
	return out
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
