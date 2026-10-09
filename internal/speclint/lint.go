// Package speclint checks that code, tests, and ADRs link correctly to the specs in
// features/*.feature, as described in docs/spec-annotations.md.
package speclint

import (
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
)

// Problem is one lint finding, reported as "file:line: msg".
type Problem struct {
	File string
	Line int
	Msg  string
}

func (p Problem) String() string {
	return fmt.Sprintf("%s:%d: %s", p.File, p.Line, p.Msg)
}

// sourceExts are the files scanned for annotations.
var sourceExts = map[string]bool{
	".go": true, ".ts": true, ".tsx": true, ".js": true, ".jsx": true, ".mjs": true, ".cjs": true,
	".sql": true,
}

// skipDirs are never scanned: dependencies, build output, generated Wails bindings, and test
// fixtures (which contain deliberately broken annotations).
var skipDirs = map[string]bool{
	"node_modules":     true,
	"vendor":           true,
	"dist":             true,
	"testdata":         true,
	"build":            true,
	"frontend/wailsjs": true,
}

// Lint checks the repository at root and returns its problems sorted by file and line.
func Lint(root string) ([]Problem, error) {
	l := &linter{features: map[string]*Feature{}}

	if err := l.loadFeatures(root); err != nil {
		return nil, err
	}
	if err := l.loadAnnotations(root); err != nil {
		return nil, err
	}
	for _, a := range l.annotations {
		l.checkAnnotation(a)
	}
	l.checkCoverage()
	if err := l.checkADRs(root); err != nil {
		return nil, err
	}

	sort.SliceStable(l.problems, func(i, j int) bool {
		a, b := l.problems[i], l.problems[j]
		if a.File != b.File {
			return a.File < b.File
		}
		return a.Line < b.Line
	})
	return l.problems, nil
}

type linter struct {
	features    map[string]*Feature // keyed by repo-relative path
	annotations []Annotation
	problems    []Problem
}

func (l *linter) report(file string, line int, format string, args ...any) {
	l.problems = append(l.problems, Problem{File: file, Line: line, Msg: fmt.Sprintf(format, args...)})
}

func (l *linter) loadFeatures(root string) error {
	paths, err := filepath.Glob(filepath.Join(root, "features", "*.feature"))
	if err != nil {
		return err
	}
	for _, p := range paths {
		src, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		rel := "features/" + filepath.Base(p)
		f, problems := parseFeature(rel, string(src))
		l.features[rel] = f
		l.problems = append(l.problems, problems...)
	}
	return nil
}

func (l *linter) loadAnnotations(root string) error {
	return filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(root, p)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		if d.IsDir() {
			if rel != "." && (strings.HasPrefix(d.Name(), ".") || skipDirs[d.Name()] || skipDirs[rel]) {
				return filepath.SkipDir
			}
			return nil
		}
		if !sourceExts[path.Ext(rel)] {
			return nil
		}
		src, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		l.annotations = append(l.annotations, parseAnnotations(rel, string(src), isTestFile(rel))...)
		return nil
	})
}

// checkAnnotation reports an annotation that doesn't resolve to a live scenario or step in the
// spec it names.
func (l *linter) checkAnnotation(a Annotation) {
	ref := a.Feature
	if a.ID != "" {
		ref += "#" + a.ID
	}

	if !strings.HasPrefix(a.Feature, "features/") || !strings.HasSuffix(a.Feature, ".feature") || strings.Contains(a.Feature, `\`) {
		l.report(a.File, a.Line, "@%s [%s]: the path must be repo-relative with forward slashes, like [features/x.feature]", a.Keyword, ref)
		return
	}
	f, ok := l.features[a.Feature]
	if !ok {
		l.report(a.File, a.Line, "@%s [%s]: %s doesn't exist", a.Keyword, ref, a.Feature)
		return
	}

	var scoped *Scenario
	if a.ID != "" {
		if scoped = f.scenario(a.ID); scoped == nil {
			l.report(a.File, a.Line, "@%s [%s]: %s has no scenario with @id:%s", a.Keyword, ref, a.Feature, a.ID)
			return
		}
		if scoped.Status == StatusWithdrawn {
			l.report(a.File, a.Line, "@%s [%s]: scenario is withdrawn; remove this annotation", a.Keyword, ref)
			return
		}
	}

	if a.Keyword == "scenario" {
		switch {
		case scoped == nil:
			l.report(a.File, a.Line, "@scenario [%s]: needs a scenario ID, like [%s#<id>]", ref, a.Feature)
		case a.Text != scoped.Title:
			l.report(a.File, a.Line, "@scenario [%s]: title %q doesn't match the spec's %q", ref, a.Text, scoped.Title)
		}
		return
	}

	candidates := f.Scenarios
	if scoped != nil {
		candidates = []*Scenario{scoped}
	}
	var live, withdrawn bool
	var otherKeyword string
	for _, s := range candidates {
		for _, st := range s.Steps {
			if st.Text != a.Text {
				continue
			}
			if !strings.EqualFold(st.Keyword, a.Keyword) {
				otherKeyword = st.Keyword
				continue
			}
			if s.Status == StatusWithdrawn {
				withdrawn = true
			} else {
				live = true
			}
		}
	}
	where := a.Feature
	if scoped != nil {
		where = "scenario " + scoped.ID
	}
	switch {
	case live:
	case withdrawn:
		l.report(a.File, a.Line, "@%s [%s] %s: step is only used by withdrawn scenarios; remove this annotation", a.Keyword, ref, a.Text)
	case otherKeyword != "":
		l.report(a.File, a.Line, "@%s [%s] %s: the spec writes this step with %q; use @%s", a.Keyword, ref, a.Text, otherKeyword, strings.ToLower(otherKeyword))
	default:
		l.report(a.File, a.Line, "@%s [%s] %s: no such step in %s", a.Keyword, ref, a.Text, where)
	}
}

// checkCoverage reports implemented scenarios, and their steps, that have no annotation in code
// or in tests.
func (l *linter) checkCoverage() {
	type key struct{ feature, keyword, id, text string }
	index := map[key]*found{}
	for _, a := range l.annotations {
		k := key{a.Feature, a.Keyword, a.ID, a.Text}
		if a.Keyword == "scenario" {
			k.text = "" // titles are checked in checkAnnotation
		}
		if index[k] == nil {
			index[k] = &found{}
		}
		if a.Test {
			index[k].test = true
		} else {
			index[k].code = true
		}
	}
	has := func(k key) found {
		if v := index[k]; v != nil {
			return *v
		}
		return found{}
	}

	paths := make([]string, 0, len(l.features))
	for p := range l.features {
		paths = append(paths, p)
	}
	sort.Strings(paths)

	for _, p := range paths {
		for _, s := range l.features[p].Scenarios {
			if s.Status != StatusImplemented || s.ID == "" {
				continue
			}
			got := has(key{p, "scenario", s.ID, ""})
			l.reportMissing(p, s.Line, got, fmt.Sprintf("implemented scenario %s has no @scenario [%s#%s] annotation", s.ID, p, s.ID))

			for _, st := range s.Steps {
				kw := strings.ToLower(st.Keyword)
				unscoped := has(key{p, kw, "", st.Text})
				scoped := has(key{p, kw, s.ID, st.Text})
				got := found{code: unscoped.code || scoped.code, test: unscoped.test || scoped.test}
				l.reportMissing(p, st.Line, got, fmt.Sprintf("step %q of implemented scenario %s has no @%s [%s] annotation", st.Keyword+" "+st.Text, s.ID, kw, p))
			}
		}
	}
}

func (l *linter) reportMissing(file string, line int, got found, msg string) {
	switch {
	case !got.code && !got.test:
		l.report(file, line, "%s in code or tests", msg)
	case !got.code:
		l.report(file, line, "%s in code", msg)
	case !got.test:
		l.report(file, line, "%s in tests", msg)
	}
}

// found records whether something is annotated in code, in tests, or both.
type found struct{ code, test bool }
