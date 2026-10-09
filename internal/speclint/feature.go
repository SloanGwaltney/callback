package speclint

import (
	"fmt"
	"regexp"
	"strings"
)

// Statuses a scenario can carry, in the order a scenario moves through them.
const (
	StatusNotImplemented = "not-implemented"
	StatusImplemented    = "implemented"
	StatusDeprecated     = "deprecated"
	StatusWithdrawn      = "withdrawn"
)

var validStatuses = map[string]bool{
	StatusNotImplemented: true,
	StatusImplemented:    true,
	StatusDeprecated:     true,
	StatusWithdrawn:      true,
}

var idPattern = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)

var stepKeywords = []string{"Given", "When", "Then", "And", "But"}

// Feature is one parsed features/*.feature file.
type Feature struct {
	Path      string // repo-relative, forward slashes
	Title     string
	Scenarios []*Scenario
}

// Scenario is a Scenario or Scenario Outline with its status, ID, and steps.
type Scenario struct {
	ID     string
	Title  string
	Status string
	Line   int
	Steps  []Step
}

// Step is one Given/When/Then/And/But line.
type Step struct {
	Keyword string // as written in the spec, e.g. "And"
	Text    string // the step with its keyword removed
	Line    int
}

func (f *Feature) scenario(id string) *Scenario {
	for _, s := range f.Scenarios {
		if s.ID == id {
			return s
		}
	}
	return nil
}

type tag struct {
	name string
	line int
}

// parseFeature reads a .feature file and reports malformed status and ID tags.
func parseFeature(path, src string) (*Feature, []Problem) {
	f := &Feature{Path: path}
	var problems []Problem
	report := func(line int, format string, args ...any) {
		problems = append(problems, Problem{File: path, Line: line, Msg: fmt.Sprintf(format, args...)})
	}

	var pending []tag
	var current *Scenario
	inDocString := ""
	seenIDs := map[string]int{}

	// rejectSpecTags flags status and ID tags placed on Feature, Rule, or Examples.
	rejectSpecTags := func(keyword string) {
		for _, t := range pending {
			if strings.HasPrefix(t.name, "@status:") || strings.HasPrefix(t.name, "@id:") {
				report(t.line, "%s goes on a Scenario, not on %s", t.name, keyword)
			}
		}
		pending = nil
	}

	for i, raw := range strings.Split(src, "\n") {
		line := i + 1
		text := strings.TrimSpace(strings.TrimSuffix(raw, "\r"))

		if inDocString != "" {
			if strings.HasPrefix(text, inDocString) {
				inDocString = ""
			}
			continue
		}

		switch {
		case text == "" || strings.HasPrefix(text, "#") || strings.HasPrefix(text, "|"):
		case strings.HasPrefix(text, `"""`), strings.HasPrefix(text, "```"):
			inDocString = text[:3]
		case strings.HasPrefix(text, "@"):
			for _, name := range strings.Fields(text) {
				if strings.HasPrefix(name, "#") {
					break
				}
				pending = append(pending, tag{name: name, line: line})
			}
		case strings.HasPrefix(text, "Feature:"):
			rejectSpecTags("Feature")
			f.Title = strings.TrimSpace(strings.TrimPrefix(text, "Feature:"))
			current = nil
		case strings.HasPrefix(text, "Rule:"):
			rejectSpecTags("Rule")
			current = nil
		case strings.HasPrefix(text, "Examples:"), strings.HasPrefix(text, "Scenarios:"):
			rejectSpecTags("Examples")
		case strings.HasPrefix(text, "Background:"):
			pending = nil
			current = nil
			report(line, "Background isn't used in this repo; put each scenario's setup in its own Given steps")
		case strings.HasPrefix(text, "Scenario Outline:"), strings.HasPrefix(text, "Scenario:"):
			_, title, _ := strings.Cut(text, ":")
			current = &Scenario{Title: strings.TrimSpace(title), Line: line}
			f.Scenarios = append(f.Scenarios, current)

			var statuses, ids []string
			for _, t := range pending {
				if v, ok := strings.CutPrefix(t.name, "@status:"); ok {
					statuses = append(statuses, v)
				} else if v, ok := strings.CutPrefix(t.name, "@id:"); ok {
					ids = append(ids, v)
				}
			}
			pending = nil

			switch {
			case len(statuses) == 0:
				report(line, "scenario %q has no @status: tag", current.Title)
			case len(statuses) > 1:
				report(line, "scenario %q has %d @status: tags; it needs exactly one", current.Title, len(statuses))
			case !validStatuses[statuses[0]]:
				report(line, "scenario %q has unknown status %q", current.Title, statuses[0])
			default:
				current.Status = statuses[0]
			}

			switch {
			case len(ids) == 0:
				report(line, "scenario %q has no @id: tag", current.Title)
			case len(ids) > 1:
				report(line, "scenario %q has %d @id: tags; it needs exactly one", current.Title, len(ids))
			case !idPattern.MatchString(ids[0]):
				report(line, "scenario ID %q isn't kebab-case (lowercase letters, digits, hyphens)", ids[0])
			default:
				if first, dup := seenIDs[ids[0]]; dup {
					report(line, "scenario ID %q is already used on line %d", ids[0], first)
				} else {
					seenIDs[ids[0]] = line
					current.ID = ids[0]
				}
			}
		default:
			keyword, rest, ok := cutStepKeyword(text)
			if ok && current != nil {
				current.Steps = append(current.Steps, Step{Keyword: keyword, Text: rest, Line: line})
			}
		}
	}

	return f, problems
}

func cutStepKeyword(text string) (keyword, rest string, ok bool) {
	for _, k := range stepKeywords {
		if r, found := strings.CutPrefix(text, k+" "); found {
			return k, strings.TrimSpace(r), true
		}
	}
	return "", "", false
}
