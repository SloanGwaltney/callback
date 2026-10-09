package speclint

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func lintFixture(t *testing.T, name string) []string {
	t.Helper()
	problems, err := Lint(filepath.Join("testdata", name))
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, p := range problems {
		got = append(got, p.String())
	}
	return got
}

// The good fixture covers shared steps, scoped steps, JSX comments, TypeScript tests, and
// not-implemented and withdrawn scenarios, all linked correctly.
func TestGoodRepoHasNoProblems(t *testing.T) {
	if got := lintFixture(t, "good"); len(got) != 0 {
		t.Errorf("expected no problems, got:\n%s", strings.Join(got, "\n"))
	}
}

// The bad fixture has one of each kind of problem. testdata/bad/want.txt is the expected output.
func TestBadRepoReportsEachProblem(t *testing.T) {
	want, err := os.ReadFile(filepath.Join("testdata", "bad", "want.txt"))
	if err != nil {
		t.Fatal(err)
	}
	wantLines := strings.Split(strings.TrimSpace(strings.ReplaceAll(string(want), "\r\n", "\n")), "\n")
	got := lintFixture(t, "bad")

	if strings.Join(got, "\n") != strings.Join(wantLines, "\n") {
		t.Errorf("got:\n%s\n\nwant:\n%s", strings.Join(got, "\n"), strings.Join(wantLines, "\n"))
	}
}

func TestParseAnnotationsOnlyReadsComments(t *testing.T) {
	src := strings.Join([]string{
		"// @then [features/a.feature] line comment",
		"\t/* @then [features/a.feature] block comment */",
		"      {/* @then [features/a.feature#x] jsx comment */}",
		" * @then [features/a.feature] block continuation",
		`s := "// @then [features/a.feature] in a string"`,
		"x() // @then [features/a.feature] trailing comment",
	}, "\n")

	var got []string
	for _, a := range parseAnnotations("f.go", src, false) {
		got = append(got, a.Text)
	}
	want := []string{"line comment", "block comment", "jsx comment", "block continuation"}
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestParseAnnotationsInSQL(t *testing.T) {
	src := strings.Join([]string{
		"-- @then [features/a.feature] sql comment",
		"    -- @then [features/a.feature] indented sql comment",
		"// @then [features/a.feature] not a sql comment",
		"SELECT 1; -- @then [features/a.feature] trailing comment",
		"SELECT '-- @then [features/a.feature] in a string';",
	}, "\n")

	var got []string
	for _, a := range parseAnnotations("queries/jobs.sql", src, false) {
		got = append(got, a.Text)
	}
	want := []string{"sql comment", "indented sql comment"}
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Errorf("got %q, want %q", got, want)
	}

	// -- isn't a comment in Go, so it doesn't count there.
	if got := parseAnnotations("f.go", "-- @then [features/a.feature] x", false); len(got) != 0 {
		t.Errorf("parsed %d annotations from -- in a .go file, want 0", len(got))
	}
}

func TestIsTestFile(t *testing.T) {
	for path, want := range map[string]bool{
		"app_test.go":                   true,
		"frontend/src/App.test.tsx":     true,
		"frontend/src/App.spec.ts":      true,
		"frontend/src/__tests__/App.ts": true,
		"app.go":                        false,
		"frontend/src/App.tsx":          false,
		"frontend/src/testing.ts":       false,
	} {
		if got := isTestFile(path); got != want {
			t.Errorf("isTestFile(%q) = %v, want %v", path, got, want)
		}
	}
}
